package main

// Durable task engine (Inngest), served by the website at /api/inngest.
//
// One Inngest run = one Console request:
//   load    read the one-time job record the authenticated Console wrote (this is what authorises the run)
//   plan    the session's agent audits and plans (primary model, then its fallback model)
//   for each step:
//       wait   step.Sleep when the plan asks for a pause (minutes or hours, nothing held open)
//       do     the website calls the bot, which only executes it
//   finish  write the final status
//
// Inngest memoizes every finished step and re-runs the function from the top after each one, so everything outside
// a step must be deterministic. A Discord action is never repeated: it is guarded by a one-shot Redis marker.
//
// Env:
//   INNGEST_EVENT_KEY     lets the site send the "start" event
//   INNGEST_SIGNING_KEY   verifies that calls to /api/inngest really come from Inngest
//   INNGEST_DEV           set to 1 only for local development with the Inngest Dev Server
//   BOT_URL (the bot's public URL), BOT_PRIVATE_KEY (Ed25519 private key for signing requests).

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
)

const (
	statusTTL        = 60 * 60 * 24
	eventStart       = "controller/command.requested"
	microTaskTimeout = 9 * time.Second
	functionID       = "run-command"
)

// Event payloads. If your inngestgo version declares GenericEvent with a different number of type parameters,
// this is the one place to adjust.
type startData struct {
	CommandID string `json:"command_id"`
}
type startEvent = inngestgo.GenericEvent[startData]

type failData struct {
	Event struct {
		Data startData `json:"data"`
	} `json:"event"`
}
type failEvent = inngestgo.GenericEvent[failData]

type Workflow struct {
	client       inngestgo.Client
	botURL       string
	botPrivateKey ed25519.PrivateKey // Backend's private key for signing requests to bot

	mu        sync.Mutex
	sendAt    time.Time
	sendError string
}

// Step results. Inngest stores these as JSON, so every field is tagged.
type loadResult struct {
	Found   bool   `json:"found"`
	GuildID string `json:"guild_id"`
	Agent   int    `json:"agent"`
	Prompt  string `json:"prompt"`
}

type thinkResult struct {
	Verdict string     `json:"verdict"`
	Turn    *AgentTurn `json:"turn,omitempty"`
}

type botReply struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

type textResult struct {
	OK   bool   `json:"ok"`
	Text string `json:"text"`
}

type auditItem struct {
	turn   int
	action string
	detail string
	ok     bool
}

func newWorkflow() *Workflow {
	w := &Workflow{botURL: urlEnv("BOT_URL")}
	
	// Load Ed25519 key pair for asymmetric signing (required)
	keyPair := LoadBackendKeyPair()
	if keyPair == nil {
		log.Fatal("BOT_PRIVATE_KEY is required but not configured. Generate an Ed25519 key pair and set BOT_PRIVATE_KEY.")
	}
	w.botPrivateKey = keyPair.PrivateKey
	log.Printf("Ed25519 key pair loaded for bot authentication (public key: %s)", GetPublicKeyString(keyPair.PublicKey))
	
	// The SDK reads these itself, so write the cleaned values back.
	for _, name := range []string{"INNGEST_EVENT_KEY", "INNGEST_SIGNING_KEY", "INNGEST_SIGNING_KEY_FALLBACK"} {
		if os.Getenv(name) != "" {
			_ = os.Setenv(name, secretEnv(name))
		}
	}
	if !w.Available() {
		log.Printf("Task engine is not fully configured (BOT_URL, BOT_PRIVATE_KEY, INNGEST_EVENT_KEY, INNGEST_SIGNING_KEY); Console requests will report it as unavailable")
		return w
	}
	client, err := inngestgo.NewClient(inngestgo.ClientOpts{AppID: "controller"})
	if err != nil {
		log.Printf("Could not create the Inngest client: %v", err)
		return w
	}
	w.client = client
	return w
}

func devMode() bool {
	v := strings.ToLower(envStr("INNGEST_DEV"))
	return v != "" && v != "0" && v != "false"
}

// Available: the engine only runs when fully configured. In production the signing key is mandatory: it is what
// stops anyone forging a "continue this run" request to make the bot execute steps of their choosing.
func (w *Workflow) Available() bool {
	if w.botURL == "" || w.botPrivateKey == nil {
		return false
	}
	if devMode() {
		return true
	}
	return os.Getenv("INNGEST_EVENT_KEY") != "" && os.Getenv("INNGEST_SIGNING_KEY") != ""
}

// Register creates the Inngest functions and returns the HTTP handler to mount at /api/inngest.
func (w *Workflow) Register() (http.Handler, error) {
	if w.client == nil {
		return nil, errors.New("inngest client is not available")
	}
	retries := 2
	if _, err := inngestgo.CreateFunction(
		w.client,
		inngestgo.FunctionOpts{ID: functionID, Retries: &retries},
		inngestgo.EventTrigger(eventStart, nil),
		w.runCommand,
	); err != nil {
		return nil, err
	}
	// Runs when the engine gave up on a request (retries used up): close it so it does not stay "processing".
	expr := fmt.Sprintf(`event.data.function_id == "controller-%s"`, functionID)
	if _, err := inngestgo.CreateFunction(
		w.client,
		inngestgo.FunctionOpts{ID: functionID + "-failure"},
		inngestgo.EventTrigger("inngest/function.failed", &expr),
		w.onFailure,
	); err != nil {
		return nil, err
	}
	return w.client.Serve(), nil
}

// CheckEngine feeds the status card. Inngest keys cannot be tested from outside, so this reports whether the last
// attempt to start a run was refused (it reports "degraded" for 5 minutes after a refusal).
func (w *Workflow) CheckEngine() (string, string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sendError != "" && time.Since(w.sendAt) < 5*time.Minute {
		return "degraded", w.sendError
	}
	return "operational", ""
}

// StartRun sends the start event to Inngest. The event id makes a duplicate send start nothing twice.
func (w *Workflow) StartRun(ctx context.Context, commandID string) error {
	id := commandID
	_, err := w.client.Send(ctx, inngestgo.Event{
		Name: eventStart,
		ID:   &id,
		Data: map[string]any{"command_id": commandID},
	})
	w.mu.Lock()
	defer w.mu.Unlock()
	if err != nil {
		w.sendAt, w.sendError = time.Now(), "Inngest refused the start event. Check INNGEST_EVENT_KEY"
		msg := err.Error()
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return fmt.Errorf("inngest send failed (%T): %s", err, msg)
	}
	w.sendError = ""
	return nil
}

// ---- helpers --------------------------------------------------------------------------------------

func setStatus(ctx context.Context, commandID, status, errText string) error {
	if _, err := rdb.Set(ctx, "command:"+commandID+":status", status, statusTTL, false); err != nil {
		return err
	}
	if errText != "" {
		if _, err := rdb.Set(ctx, "command:"+commandID+":error", clipRunes(errText, 2000), statusTTL, false); err != nil {
			return err
		}
	}
	return nil
}

// finish writes the final status, releases the per-guild in-flight slot, and records task-engine health.
// Runs once per request: a retried finish step, or the failure handler after it, finds the marker and does nothing.
func (w *Workflow) finish(ctx context.Context, commandID, guildID, status, errText string, engineOK bool, engineErr, result, goal string, tainted bool) (map[string]string, error) {
	out := map[string]string{"status": status}
	first, err := rdb.Set(ctx, "command:"+commandID+":closed", "1", statusTTL, true)
	if err != nil {
		return nil, err
	}
	if !first {
		return out, nil
	}
	if result != "" {
		if _, err := rdb.Set(ctx, "command:"+commandID+":result", clipRunes(result, 1500), statusTTL, false); err != nil {
			return nil, err
		}
	}
	if err := setStatus(ctx, commandID, status, errText); err != nil {
		return nil, err
	}
	dbFinishRequest(ctx, commandID, status, result, errText) // durable copy; never fails the step
	remember := result
	if remember == "" {
		remember = errText
	}
	rememberExchange(ctx, guildID, goal, status, remember, tainted) // short memory for follow-up requests; never fails the step
	if status != "cancelled" {                               // the cancel route already released the slot
		key := "guild:" + guildID + ":inflight"
		if n, e := rdb.Decr(ctx, key); e != nil {
			log.Printf("Could not update workflow bookkeeping")
		} else if n <= 0 {
			_ = rdb.Del(ctx, key)
		}
	}
	health, _ := json.Marshal(map[string]any{"ok": engineOK, "ts": nowSeconds(), "err": engineErr})
	if _, e := rdb.Set(ctx, "status:workflow", string(health), statusTTL*4, false); e != nil {
		log.Printf("Could not update workflow bookkeeping")
	}
	return out, nil
}

// botResult interprets the bot's reply to one step: (ok, error text, engine_ok).
func botResult(resp botReply) (bool, string, bool) {
	body := loadsMap(resp.Body)
	if resp.Status == 401 {
		log.Printf("BOT AUTH ERROR: status 401, body: %s", clipRunes(resp.Body, 200))
		// Bot rejected the request - check if there's a specific error message
		msg := asStr(body["error"])
		if msg != "" {
			return false, fmt.Sprintf("Bot authentication FAILED: %s. FIX: Ensure BOT_PRIVATE_KEY (backend) and BOT_PUBLIC_KEY (bot) are set and match.", msg), false
		}
		return false, "Bot authentication FAILED (Ed25519). FIX: 1) Set BOT_PRIVATE_KEY in backend, 2) Set matching BOT_PUBLIC_KEY in bot, 3) Restart both services.", false
	}
	if resp.Status < 200 || resp.Status >= 300 || body == nil {
		log.Printf("BOT ERROR: HTTP %d, body: %s", resp.Status, clipRunes(resp.Body, 200))
		code := "none"
		if resp.Status != 0 {
			code = strconv.Itoa(resp.Status)
		}
		return false, fmt.Sprintf("The bot could not be reached or answered with an error (HTTP %s).", code), false
	}
	if okv, _ := body["ok"].(bool); okv {
		return true, "", true
	}
	msg := asStr(body["error"])
	if msg == "" {
		msg = "The step failed."
	}
	return false, clipRunes(msg, 1500), true
}

// describeAction is a short, readable record of a Discord action for the agent's progress log.
func describeAction(action string, p StepParameters) string {
	var parts []string
	add := func(key string, val any) { parts = append(parts, jsonString(key)+": "+jsonString(val)) }
	if p.CategoryName != nil {
		add("category_name", *p.CategoryName)
	}
	if p.ChannelName != nil {
		add("channel_name", *p.ChannelName)
	}
	if p.ChannelID != nil {
		add("channel_id", *p.ChannelID)
	}
	if p.NewName != nil {
		add("new_name", *p.NewName)
	}
	if p.RoleName != nil {
		add("role_name", *p.RoleName)
	}
	if p.MemberID != nil {
		add("member_id", *p.MemberID)
	}
	if p.MemberName != nil {
		add("member_name", *p.MemberName)
	}
	if p.Title != nil {
		add("title", *p.Title)
	}
	if len(parts) == 0 {
		return action
	}
	return action + " " + clipRunes("{"+strings.Join(parts, ", ")+"}", 200)
}

// callBot sends one Discord action. Never repeated: a one-shot marker is set before the call, so if Inngest re-runs
// this step after a crash it reports "not repeated" instead of acting twice.
func (w *Workflow) callBot(ctx context.Context, commandID string, turn int, guildID, action string, params StepParameters) (botReply, error) {
	first, err := rdb.Set(ctx, fmt.Sprintf("command:%s:turn:%d:started", commandID, turn), "1", statusTTL, true)
	if err != nil {
		return botReply{}, err
	}
	if !first {
		return botReply{Status: 0, Body: jsonString(map[string]any{"ok": false, "error": "This step was interrupted and was not repeated, to avoid doing it twice. " +
			"Check the server, then run the request again if needed."})}, nil
	}
	payload, err := json.Marshal(map[string]any{"guild_id": guildID, "command_id": commandID, "action_name": action, "parameters": params})
	if err != nil {
		return botReply{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.botURL+"/execute", bytes.NewReader(payload))
	if err != nil {
		return botReply{}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Sign request with Ed25519 private key
	log.Printf("DEBUG: Sending request to bot with Ed25519 signature (public key: %s)", GetPublicKeyString(w.botPrivateKey.Public().(ed25519.PublicKey)))
	signature := SignRequest(w.botPrivateKey, payload)
	req.Header.Set("X-Bot-Signature", signature)
	// Also send the public key so the bot can verify (optional, but useful for debugging)
	req.Header.Set("X-Bot-Public-Key", GetPublicKeyString(w.botPrivateKey.Public().(ed25519.PublicKey)))
	
	client := &http.Client{Timeout: microTaskTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return botReply{Status: 0, Body: ""}, nil
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 20000))
	return botReply{Status: resp.StatusCode, Body: string(data)}, nil
}

// onFailure runs when the engine gave up on a request.
func (w *Workflow) onFailure(ctx context.Context, input inngestgo.Input[failData]) (any, error) {
	id, err := uuid.Parse(input.Event.Data.Event.Data.CommandID)
	if err != nil {
		return nil, nil
	}
	commandID := id.String()
	guildID, err := rdb.Get(ctx, "command:"+commandID+":guild")
	if err != nil {
		return nil, err
	}
	if guildID != "" {
		if _, err := w.finish(ctx, commandID, guildID, "failed", "The task engine hit an internal error. Please try again.",
			false, "A run failed inside the engine", "", "", false); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// ---- the run --------------------------------------------------------------------------------------

func (w *Workflow) runCommand(ctx context.Context, input inngestgo.Input[startData]) (any, error) {
	parsed, err := uuid.Parse(input.Event.Data.CommandID)
	if err != nil {
		return nil, nil // not one of ours
	}
	commandID := parsed.String()

	job, err := step.Run(ctx, "load", func(ctx context.Context) (loadResult, error) {
		// One-time and written only by the signed-in Console, so a stray event does nothing.
		raw, err := rdb.GetDel(ctx, "command:"+commandID+":job")
		if err != nil {
			return loadResult{}, err
		}
		var j struct {
			GuildID string `json:"guild_id"`
			Agent   int    `json:"agent"`
			Prompt  string `json:"prompt"`
		}
		if raw == "" || json.Unmarshal([]byte(raw), &j) != nil {
			return loadResult{}, nil
		}
		return loadResult{Found: true, GuildID: j.GuildID, Agent: j.Agent, Prompt: j.Prompt}, nil
	})
	if err != nil {
		return nil, err
	}
	if !job.Found {
		return nil, nil
	}
	guildID, agent, goal := job.GuildID, job.Agent, job.Prompt
	toolCfg := enabledTools()
	allowed := map[string]bool{"web_research": true, "write_text": true}
	for name := range discordActionNames {
		allowed[name] = true
	}
	if _, ok := toolCfg["call_webhook"]; ok {
		allowed["call_webhook"] = true
	}
	if _, ok := toolCfg["send_email"]; ok {
		allowed["send_email"] = true
	}

	var observations []string // the agent's memory, rebuilt identically on every replay
	used := map[string]int{"discord": 0, "web_research": 0, "write_text": 0, "call_webhook": 0, "send_email": 0}
	limits := map[string]int{"discord": maxDiscordSteps, "web_research": maxWebCalls, "write_text": maxWriteCalls,
		"call_webhook": maxWebhookCalls, "send_email": maxEmailCalls}
	tainted, toolErrors, delayTotal := false, 0, 0
	var auditQueue []auditItem // irreversible Discord actions, written to MySQL by the next step

	flushAudit := func(ctx context.Context) {
		for _, a := range auditQueue { // INSERT IGNORE: replays and repeats are harmless
			dbAudit(ctx, commandID, a.turn, guildID, "agent", a.action, a.detail, a.ok)
		}
	}

	stop := func(status, errText, result string, engineOK bool, engineErr string) (any, error) {
		_, err := step.Run(ctx, "finish", func(ctx context.Context) (map[string]string, error) {
			flushAudit(ctx)
			return w.finish(ctx, commandID, guildID, status, errText, engineOK, engineErr, result, goal, tainted)
		})
		return nil, err
	}
	stopSimple := func(status, errText, result string) (any, error) {
		return stop(status, errText, result, true, "")
	}
	progressTail := func() string { return clip(strings.Join(observations, "\n"), 900) }

	for turn := 1; turn <= maxTurns; turn++ {
		decided, err := step.Run(ctx, fmt.Sprintf("think-%d", turn), func(ctx context.Context) (thinkResult, error) {
			flushAudit(ctx)
			if c, err := rdb.Get(ctx, "command:"+commandID+":cancel"); err != nil {
				return thinkResult{}, err
			} else if c != "" {
				return thinkResult{Verdict: "cancelled"}, nil
			}
			if k, err := rdb.Get(ctx, "kill:switch"); err != nil {
				return thinkResult{}, err
			} else if k != "" {
				return thinkResult{Verdict: "paused"}, nil
			}
			if turn == 1 {
				if err := setStatus(ctx, commandID, "processing", ""); err != nil {
					return thinkResult{}, err
				}
			}
			raw, err := rdb.Get(ctx, "guild:"+guildID+":snapshot")
			if err != nil {
				return thinkResult{}, err
			}
			snapshot := loadsMap(raw)
			if snapshot == nil {
				return thinkResult{Verdict: "bot_unready"}, nil
			}
			t, err := nextTurn(ctx, agent, snapshot, goal, loadContext(ctx, guildID), append([]string(nil), observations...), turn, toolCfg)
			if err != nil { // primary and fallback model both failed
				return thinkResult{Verdict: "traffic"}, nil
			}
			return thinkResult{Verdict: "ok", Turn: &t}, nil
		})
		if err != nil {
			return nil, err
		}

		switch decided.Verdict {
		case "cancelled":
			return stopSimple("cancelled", "", "")
		case "paused":
			msg := "The service was paused by its operator, so the request stopped."
			if len(observations) > 0 {
				msg += " Progress so far:\n" + progressTail()
			}
			return stopSimple("failed", msg, "")
		case "bot_unready":
			return stopSimple("failed", "The bot has not reported this server yet. Check that it is online and try again.", "")
		case "traffic":
			if len(observations) > 0 {
				return stopSimple("failed", "The agent became unavailable part-way through. Progress so far:\n"+progressTail(), "")
			}
			return stopSimple("blocked_by_traffic", "", "")
		}

		t := decided.Turn
		if t == nil || !t.IsSafe {
			reason := "no reason given"
			if t != nil && t.RejectionReason != "" {
				reason = t.RejectionReason
			}
			return stopSimple("rejected", "Rejected: "+reason, "")
		}
		if t.Done {
			summary := strings.TrimSpace(t.Summary)
			if summary == "" {
				summary = "Done."
			}
			return stopSimple("completed", "", summary)
		}

		action := ""
		if t.ActionName != nil {
			action = *t.ActionName
		}
		params := t.Parameters
		if !allowed[action] {
			shown := action
			if shown == "" {
				shown = "(none)"
			}
			return stopSimple("rejected", "The agent asked for an unsupported action: "+shown, "")
		}

		// ---- waits (durable): the request pauses here, and can be cancelled meanwhile
		delay := 0
		if t.DelaySeconds != nil && *t.DelaySeconds > 0 {
			delay = *t.DelaySeconds
		}
		delayTotal += delay
		if delay > maxDelayStep || delayTotal > maxDelayTotal {
			return stopSimple("failed_sensibility", "Rejected: the requested waiting time is too long (at most 6 hours per wait, 12 hours in total).", "")
		}
		if delay > 0 {
			if _, err := step.Run(ctx, fmt.Sprintf("waiting-%d", turn), func(ctx context.Context) (bool, error) {
				return true, setStatus(ctx, commandID, "waiting", "")
			}); err != nil {
				return nil, err
			}
			step.Sleep(ctx, fmt.Sprintf("wait-%d", turn), time.Duration(delay)*time.Second)
			cancelled, err := step.Run(ctx, fmt.Sprintf("resume-%d", turn), func(ctx context.Context) (bool, error) {
				// True if the owner cancelled while this request was waiting.
				c, err := rdb.Get(ctx, "command:"+commandID+":cancel")
				if err != nil {
					return false, err
				}
				if c != "" {
					return true, nil
				}
				return false, setStatus(ctx, commandID, "processing", "")
			})
			if err != nil {
				return nil, err
			}
			if cancelled {
				return stopSimple("cancelled", "", "")
			}
		}

		kind := action
		if discordActionNames[action] {
			kind = "discord"
		}
		switch {
		case used[kind] >= limits[kind]:
			observations = append(observations, fmt.Sprintf("[%d] %s: BLOCKED, the limit of %d for this request is reached.", turn, action, limits[kind]))
			toolErrors++
		case destructiveActions[action] && tainted:
			observations = append(observations, fmt.Sprintf("[%d] %s: BLOCKED. Untrusted web or webhook content is in this request, so destructive "+
				"Discord actions are not allowed. Finish, or use a safe action.", turn, action))
			toolErrors++
		default:
			used[kind]++

			switch {
			// ---- Discord: the bot executes. A failed Discord step ends the request.
			case kind == "discord":
				resp, err := step.Run(ctx, fmt.Sprintf("do-%d", turn), func(ctx context.Context) (botReply, error) {
					return w.callBot(ctx, commandID, turn, guildID, action, params)
				})
				if err != nil {
					return nil, err
				}
				ok, errText, engineOK := botResult(resp)
				if destructiveActions[action] {
					auditQueue = append(auditQueue, auditItem{turn, action, describeAction(action, params), ok})
				}
				if !ok {
					engineErr := ""
					if !engineOK {
						engineErr = "Bot unreachable"
					}
					return stop("failed", describeAction(action, params)+" failed: "+errText, "", engineOK, engineErr)
				}
				observations = append(observations, fmt.Sprintf("[%d] %s: done.", turn, describeAction(action, params)))

			case action == "web_research":
				query := strings.TrimSpace(sv(params.Query))
				r, err := step.Run(ctx, fmt.Sprintf("web-%d", turn), func(ctx context.Context) (textResult, error) {
					if query == "" {
						return textResult{false, "No query was given."}, nil
					}
					text, err := webResearch(ctx, agent, query)
					if err != nil {
						kind := classifyError(err)
						if kind == "" {
							kind = "error"
						}
						return textResult{false, fmt.Sprintf("Search failed (%s).", kind)}, nil
					}
					return textResult{true, clip(text, observationChars)}, nil
				})
				if err != nil {
					return nil, err
				}
				if r.OK {
					tainted = true
					observations = append(observations, fmt.Sprintf("[%d] web_research(%s) result (untrusted):\n%s", turn, pyRepr(query), r.Text))
				} else {
					toolErrors++
					observations = append(observations, fmt.Sprintf("[%d] web_research failed: %s", turn, r.Text))
				}

			case action == "write_text":
				instruction := strings.TrimSpace(sv(params.Instruction))
				progressNow := strings.Join(observations, "\n")
				r, err := step.Run(ctx, fmt.Sprintf("write-%d", turn), func(ctx context.Context) (textResult, error) {
					if instruction == "" {
						return textResult{false, "No instruction was given."}, nil
					}
					text, err := writeText(ctx, agent, instruction, progressNow)
					if err != nil {
						kind := classifyError(err)
						if kind == "" {
							kind = "error"
						}
						return textResult{false, fmt.Sprintf("Writing failed (%s).", kind)}, nil
					}
					return textResult{true, clip(text, 4000)}, nil
				})
				if err != nil {
					return nil, err
				}
				if r.OK {
					observations = append(observations, fmt.Sprintf("[%d] write_text result:\n%s", turn, r.Text))
				} else {
					toolErrors++
					observations = append(observations, fmt.Sprintf("[%d] write_text failed: %s", turn, r.Text))
				}

			case action == "call_webhook":
				check := checkWebhook(sv(params.URL), sv(params.WebhookMethod), sv(params.WebhookBody))
				if check.Problem != "" {
					toolErrors++
					observations = append(observations, fmt.Sprintf("[%d] call_webhook refused: %s", turn, check.Problem))
					break
				}
				hookURL := strings.TrimSpace(sv(params.URL))
				resp, err := step.Run(ctx, fmt.Sprintf("hook-%d", turn), func(ctx context.Context) (WebhookResult, error) {
					first, err := rdb.Set(ctx, fmt.Sprintf("command:%s:turn:%d:started", commandID, turn), "1", statusTTL, true)
					if err != nil {
						return WebhookResult{}, err
					}
					if !first {
						return WebhookResult{Status: 0, Body: "(not repeated after an interruption)"}, nil // never send a webhook twice
					}
					return callWebhook(ctx, hookURL, check.Method, check.Body), nil
				})
				if err != nil {
					return nil, err
				}
				text := clip(resp.Body, 2000)
				tainted = true
				code := "no response"
				if resp.Status != 0 {
					code = strconv.Itoa(resp.Status)
				}
				observations = append(observations, fmt.Sprintf("[%d] call_webhook %s %s -> HTTP %s (untrusted):\n%s", turn, check.Method, hookURL, code, text))
				if resp.Status < 200 || resp.Status >= 300 {
		log.Printf("BOT ERROR: HTTP %d, body: %s", resp.Status, clipRunes(resp.Body, 200))
					toolErrors++
				}

			case action == "send_email":
				problem := checkEmail(sv(params.EmailTo), sv(params.EmailSubject), sv(params.EmailBody))
				if problem != "" {
					toolErrors++
					observations = append(observations, fmt.Sprintf("[%d] send_email refused: %s", turn, problem))
					break
				}
				to, subject, body := sv(params.EmailTo), sv(params.EmailSubject), sv(params.EmailBody)
				r, err := step.Run(ctx, fmt.Sprintf("mail-%d", turn), func(ctx context.Context) (textResult, error) {
					text, err := sendEmail(ctx, to, subject, body)
					if err != nil {
						return textResult{false, fmt.Sprintf("Email failed (%T).", err)}, nil
					}
					return textResult{true, text}, nil
				})
				if err != nil {
					return nil, err
				}
				observations = append(observations, fmt.Sprintf("[%d] send_email: %s", turn, r.Text))
				if !r.OK {
					toolErrors++
				}
			}
		}

		if toolErrors > 3 {
			return stopSimple("failed", "Stopped after repeated tool errors. Progress so far:\n"+progressTail(), "")
		}
	}

	return stopSimple("failed", fmt.Sprintf("Reached the limit of %d turns before finishing. Progress so far:\n%s", maxTurns, progressTail()), "")
}
