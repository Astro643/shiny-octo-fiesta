package main

// Auth (GET /login?token=...), console sessions (one person per agent, everyone else waits in line),
// and the execute / cancel / status routes.
//
// Redis contract (shared with the bot and unchanged from the Python version):
//   dash_auth:{token}                       string  -> guild id (or JSON {"guild_id": "..."}); TTL = expiry
//   agent:{1|2|3}:lease / guild:{id}:agent  which server holds an agent right now (session lease)
//   agent:waitlist                          zset    -> servers waiting for an agent
//   command:{id}:job                        string  -> one-time job read by the Inngest run
//   command:{id}:status                     processing | waiting | completed | blocked_by_traffic
//                                           | rejected | failed | cancelled
//   command:{id}:error / :result            text written by the task engine
//   guild:{id}:snapshot                     channels/roles snapshot written by the bot

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	tokenRE   = regexp.MustCompile(`^[A-Za-z0-9_\-]{8,128}$`) // blocks Redis key injection
	guildIDRE = regexp.MustCompile(`^\d{15,25}$`)

	linkPreviewAgents = []string{"discordbot", "slackbot", "twitterbot", "facebookexternalhit", "telegrambot", "whatsapp"}
	failedStates      = map[string]bool{"failed": true, "rejected": true}
)

// ---- small helpers --------------------------------------------------------------------------------

// reqContext is detached from the client's connection: a visitor closing the tab must not leave a half-done update.
func reqContext(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Second)
}

func respondError(c *gin.Context, err error) {
	var he *httpError
	if errors.As(err, &he) {
		c.AbortWithStatusJSON(he.Status, gin.H{"detail": he.Detail})
		return
	}
	log.Printf("Unhandled error on %s: %v", c.Request.URL.Path, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"detail": "Internal server error"})
}

func clientIP(c *gin.Context) string {
	if fwd := c.GetHeader("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return host
	}
	return "unknown"
}

// visitorHash is salted, so no raw IPs are stored.
func visitorHash(c *gin.Context) string {
	sum := sha256.Sum256(append(append([]byte{}, sessionKey...), []byte(clientIP(c))...))
	return hex.EncodeToString(sum[:])[:24]
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	}
	f, err := strconv.ParseFloat(asStr(v), 64)
	return err != nil || f != 0
}

// ---- signed session cookie ------------------------------------------------------------------------

func signGuild(guildID string) string {
	mac := hmac.New(sha256.New, sessionKey)
	mac.Write([]byte(guildID))
	return guildID + "." + hex.EncodeToString(mac.Sum(nil))
}

// verifyCookie returns the guild id if the signed cookie is authentic, else "".
func verifyCookie(v string) string {
	i := strings.LastIndex(v, ".")
	if v == "" || i < 0 {
		return ""
	}
	guildID, sig := v[:i], v[i+1:]
	mac := hmac.New(sha256.New, sessionKey)
	mac.Write([]byte(guildID))
	expected := hex.EncodeToString(mac.Sum(nil))
	if hmac.Equal([]byte(sig), []byte(expected)) && guildIDRE.MatchString(guildID) {
		return guildID
	}
	return ""
}

// requireGuild is the middleware that puts the authenticated guild id on the request, or answers 401.
func requireGuild() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, _ := c.Cookie(cookieName)
		guildID := verifyCookie(cookie)
		if guildID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Session expired. Open a fresh dashboard link."})
			return
		}
		c.Set("guild_id", guildID)
		c.Next()
	}
}

// extractGuildID: dash_auth values may be a bare guild id or a JSON object containing one.
func extractGuildID(raw string) string {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "{") {
		m := loadsMap(value)
		if m == nil {
			return ""
		}
		value = strings.TrimSpace(asStr(m["guild_id"]))
	}
	if guildIDRE.MatchString(value) {
		return value
	}
	return ""
}

// ---- login ----------------------------------------------------------------------------------------

const baseCSS = `
*{box-sizing:border-box}html,body{margin:0;height:100%}
body{background:#EEEDE8;color:#050505;font:15px/1.55 Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;
-webkit-font-smoothing:antialiased;display:grid;place-items:center}
.card{background:#F8F7F4;border:1px solid #D1D0CB;border-radius:14px;padding:40px 44px;max-width:420px;margin:24px}
h1{font:400 30px/1.05 Anton,Impact,'Arial Narrow Bold',sans-serif;text-transform:uppercase;margin:0 0 10px}
p{margin:0;color:#5F5E61}
.mark{width:24px;height:24px;display:block;margin-bottom:20px;color:#000}
`

func messagePage(c *gin.Context, title, body string, status int) {
	page := "<!doctype html><html lang='en'><head><meta charset='utf-8'>" +
		"<meta name='viewport' content='width=device-width,initial-scale=1'><title>" + html.EscapeString(title) + "</title>" +
		"<link rel='stylesheet' href='https://fonts.googleapis.com/css2?family=Anton&display=swap'>" +
		"<style>" + baseCSS + "</style></head><body><div class='card'><svg class='mark' viewBox='0 0 24 24' aria-hidden='true'><path d='M2.5 5h19L12 20.5z' fill='currentColor'/></svg>" +
		"<h1>" + html.EscapeString(title) + "</h1><p>" + html.EscapeString(body) + "</p></div></body></html>"
	c.Header("Cache-Control", "no-store")
	c.Data(status, "text/html; charset=utf-8", []byte(page))
}

// handleLogin is the magic-link sign-in. The token is stored in Redis by the bot (dash_auth:{token}) with a TTL,
// and is deleted the moment it is used, so every link works once and then never again.
func handleLogin(c *gin.Context) {
	// Chat apps fetch links to build previews; such a request must not burn the user's link.
	ua := strings.ToLower(c.GetHeader("User-Agent"))
	for _, name := range linkPreviewAgents {
		if strings.Contains(ua, name) {
			messagePage(c, "Magic link", "Open this link in your browser to sign in.", http.StatusOK)
			return
		}
	}
	token := c.Query("token")
	if !tokenRE.MatchString(token) {
		messagePage(c, "Invalid link", "This magic link is malformed.", http.StatusBadRequest)
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()
	raw, err := rdb.GetDel(ctx, "dash_auth:"+token) // atomic read AND delete: the link can be used exactly once
	if err != nil {
		log.Printf("Redis lookup failed during login: %v", err)
		messagePage(c, "Service unavailable", "Could not verify your link. Please try again shortly.", http.StatusServiceUnavailable)
		return
	}
	guildID := extractGuildID(raw)
	if guildID == "" {
		messagePage(c, "Link already used or expired",
			"A magic link works only once and for a short time. Run /dashboard in Discord to get a new one.", http.StatusForbidden)
		return
	}
	// A 303 redirect strips the token from the address bar / history.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieName, signGuild(guildID), cookieMaxAge, "/", "", cookieSecure, true)
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusSeeOther, "/dashboard")
}

// ---- console sessions: one person per agent, everyone else waits in line --------------------------

// release gives up this guild's agent (now, or after a short grace so a page reload keeps it).
func release(ctx context.Context, guildID string, grace int) error {
	held, err := rdb.Get(ctx, "guild:"+guildID+":agent")
	if err != nil {
		return err
	}
	if isDigits(held) {
		n, _ := strconv.Atoi(held)
		lease, err := rdb.Get(ctx, fmt.Sprintf("agent:%d:lease", n))
		if err != nil {
			return err
		}
		if lease == guildID {
			if grace > 0 {
				if err := rdb.Expire(ctx, fmt.Sprintf("agent:%d:lease", n), grace); err != nil {
					return err
				}
				return rdb.Expire(ctx, "guild:"+guildID+":agent", grace)
			}
			if err := rdb.Del(ctx, fmt.Sprintf("agent:%d:lease", n)); err != nil {
				return err
			}
		}
	}
	return rdb.Del(ctx, "guild:"+guildID+":agent")
}

// heldAgent is the agent this guild currently holds (0 if none). Refreshes the lease.
func heldAgent(ctx context.Context, guildID string) (int, error) {
	held, err := rdb.Get(ctx, "guild:"+guildID+":agent")
	if err != nil {
		return 0, err
	}
	if !isDigits(held) {
		return 0, nil
	}
	n, _ := strconv.Atoi(held)
	lease, err := rdb.Get(ctx, fmt.Sprintf("agent:%d:lease", n))
	if err != nil {
		return 0, err
	}
	if lease != guildID {
		return 0, rdb.Del(ctx, "guild:"+guildID+":agent")
	}
	if err := rdb.Expire(ctx, fmt.Sprintf("agent:%d:lease", n), leaseSeconds); err != nil {
		return 0, err
	}
	if err := rdb.Expire(ctx, "guild:"+guildID+":agent", leaseSeconds); err != nil {
		return 0, err
	}
	return n, nil
}

func sessionPing(ctx context.Context, guildID string) (map[string]any, error) {
	var slots []int
	for _, n := range sessionAgents {
		if agentConfigured(n) {
			slots = append(slots, n)
		}
	}
	if len(slots) == 0 {
		return map[string]any{"state": "unavailable"}, nil
	}

	n, err := heldAgent(ctx, guildID)
	if err != nil {
		return nil, err
	}
	if n != 0 {
		active, err := rdb.Exists(ctx, "guild:"+guildID+":last_active")
		if err != nil {
			return nil, err
		}
		if !active {
			waiting, err := rdb.ZCard(ctx, waitlistKey)
			if err != nil {
				return nil, err
			}
			if waiting > 0 { // idle, and people are waiting: yield
				if err := release(ctx, guildID, 0); err != nil {
					return nil, err
				}
				return map[string]any{"state": "idle_released"}, nil
			}
			if _, err := rdb.Set(ctx, "guild:"+guildID+":last_active", "1", idleSeconds, false); err != nil {
				return nil, err
			}
		}
		return map[string]any{"state": "active", "agent": n}, nil
	}

	// Join / stay in line. Drop people whose consoles have gone quiet.
	now := nowSeconds()
	if _, err := rdb.Set(ctx, "wait:"+guildID, "1", waitAliveSeconds, false); err != nil {
		return nil, err
	}
	members, err := rdb.ZRange(ctx, waitlistKey)
	if err != nil {
		return nil, err
	}
	inLine := false
	for _, m := range members {
		if m == guildID {
			inLine = true
		}
	}
	if !inLine {
		if len(members) >= maxWaiting {
			return map[string]any{"state": "full"}, nil
		}
		if err := rdb.ZAdd(ctx, waitlistKey, now, guildID); err != nil {
			return nil, err
		}
		members = append(members, guildID)
	}
	var alive []string
	for _, m := range members {
		if m == guildID {
			alive = append(alive, m)
			continue
		}
		exists, err := rdb.Exists(ctx, "wait:"+m)
		if err != nil {
			return nil, err
		}
		if exists {
			alive = append(alive, m)
		} else if err := rdb.ZRem(ctx, waitlistKey, m); err != nil {
			return nil, err
		}
	}
	position := 0 // 0 = next in line (order is by join time)
	for i, m := range alive {
		if m == guildID {
			position = i
			break
		}
	}

	var free []int
	for _, k := range slots {
		leased, err := rdb.Exists(ctx, fmt.Sprintf("agent:%d:lease", k))
		if err != nil {
			return nil, err
		}
		if !leased {
			free = append(free, k)
		}
	}
	if position < len(free) {
		for _, k := range free {
			got, err := rdb.Set(ctx, fmt.Sprintf("agent:%d:lease", k), guildID, leaseSeconds, true)
			if err != nil {
				return nil, err
			}
			if got {
				if _, err := rdb.Set(ctx, "guild:"+guildID+":agent", strconv.Itoa(k), leaseSeconds, false); err != nil {
					return nil, err
				}
				if _, err := rdb.Set(ctx, "guild:"+guildID+":last_active", "1", idleSeconds, false); err != nil {
					return nil, err
				}
				if err := rdb.ZRem(ctx, waitlistKey, guildID); err != nil {
					return nil, err
				}
				if err := rdb.Del(ctx, "wait:"+guildID); err != nil {
					return nil, err
				}
				return map[string]any{"state": "active", "agent": k}, nil
			}
		}
	}
	return map[string]any{"state": "waiting", "position": position + 1, "ahead": position, "waiting": len(alive)}, nil
}

func handleSessionPing(c *gin.Context) {
	guildID := c.GetString("guild_id")
	ctx, cancel := reqContext(c)
	defer cancel()
	res, err := sessionPing(ctx, guildID)
	if err != nil {
		log.Printf("Session ping failed for guild %s: %v", guildID, err)
		respondError(c, newErr(http.StatusServiceUnavailable, "Could not reach the queue. Retrying."))
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

// handleSessionLeave is sent when the console closes. Frees the agent shortly after (a reload within the grace keeps it).
func handleSessionLeave(c *gin.Context) {
	guildID := c.GetString("guild_id")
	ctx, cancel := reqContext(c)
	defer cancel()
	if err := release(ctx, guildID, leaveGraceSeconds); err != nil {
		log.Printf("Session leave failed for guild %s", guildID)
	} else if err := rdb.ZRem(ctx, waitlistKey, guildID); err != nil {
		log.Printf("Session leave failed for guild %s", guildID)
	} else if err := rdb.Del(ctx, "wait:"+guildID); err != nil {
		log.Printf("Session leave failed for guild %s", guildID)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleSessionNew starts a fresh conversation: the agent forgets this server's earlier requests.
func handleSessionNew(c *gin.Context) {
	guildID := c.GetString("guild_id")
	ctx, cancel := reqContext(c)
	defer cancel()
	if err := forgetContext(ctx, guildID); err != nil {
		log.Printf("New conversation failed for guild %s", guildID)
		respondError(c, newErr(http.StatusServiceUnavailable, "Could not start a new conversation right now. Please try again."))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- execute / cancel / status --------------------------------------------------------------------

// executeForGuild enqueues a task for one guild, using the agent its Console session holds.
func executeForGuild(ctx context.Context, guildID, prompt string) (map[string]any, error) {
	if !wf.Available() || wf.client == nil {
		return nil, newErr(http.StatusServiceUnavailable, "The task engine is not set up yet.")
	}
	killed, err := rdb.Get(ctx, killSwitchKey)
	if err != nil {
		return nil, err
	}
	if killed != "" {
		return nil, newErr(http.StatusServiceUnavailable, "The service is paused by its operator. Please try again later.")
	}
	blocked, err := rdb.SIsMember(ctx, "blocked:guilds", guildID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, newErr(http.StatusForbidden, "This server has been blocked from using the service.")
	}

	inflightKey := "guild:" + guildID + ":inflight"
	counted := false
	commandID := ""
	// cleanup undoes what a failed start already did, and turns the cause into the 503 the user sees.
	cleanup := func(cause error) error {
		var he *httpError
		if errors.As(cause, &he) {
			return cause
		}
		log.Printf("Failed to start command for guild %s: %v", guildID, cause)
		if commandID != "" {
			_, _ = rdb.Set(ctx, "command:"+commandID+":status", "failed", statusTTL, false)
			_ = rdb.Del(ctx, "command:"+commandID+":job")
			dbSetStatus(ctx, commandID, "failed")
		}
		if counted {
			_, _ = rdb.Decr(ctx, inflightKey)
		}
		return newErr(http.StatusServiceUnavailable, "The task engine is unavailable. Please try again.")
	}

	agent, err := heldAgent(ctx, guildID)
	if err != nil {
		return nil, cleanup(err)
	}
	if agent == 0 {
		return nil, newErr(http.StatusConflict, "You do not have an agent right now. Wait for your turn.")
	}
	if _, err := rdb.Set(ctx, "guild:"+guildID+":last_active", "1", idleSeconds, false); err != nil {
		return nil, cleanup(err)
	}

	n, err := rdb.Incr(ctx, inflightKey)
	if err != nil {
		return nil, cleanup(err)
	}
	if n > maxPendingCommands {
		if _, err := rdb.Decr(ctx, inflightKey); err != nil {
			return nil, cleanup(err)
		}
		return nil, newErr(http.StatusTooManyRequests, "Too many requests in progress. Wait for some to finish.")
	}
	counted = true
	if err := rdb.Expire(ctx, inflightKey, statusTTL); err != nil {
		return nil, cleanup(err)
	}

	commandID = uuid.NewString()
	if _, err := rdb.Set(ctx, "command:"+commandID+":status", "processing", statusTTL, false); err != nil {
		return nil, cleanup(err)
	}
	if _, err := rdb.Set(ctx, "command:"+commandID+":guild", guildID, statusTTL, false); err != nil {
		return nil, cleanup(err)
	}
	job := jsonString(map[string]any{"guild_id": guildID, "agent": agent, "prompt": prompt})
	if _, err := rdb.Set(ctx, "command:"+commandID+":job", job, 600, false); err != nil {
		return nil, cleanup(err)
	}
	dbRecordRequest(ctx, commandID, guildID, agent, prompt) // before the run starts, so its final update finds the row
	if err := wf.StartRun(ctx, commandID); err != nil {
		return nil, cleanup(err)
	}
	return map[string]any{"command_id": commandID, "status": "processing", "agent": agent}, nil
}

func handleExecute(c *gin.Context) {
	var body struct {
		Prompt *string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Prompt == nil {
		respondError(c, newErr(http.StatusUnprocessableEntity, "A prompt is required."))
		return
	}
	if n := utf8.RuneCountInString(*body.Prompt); n < 1 || n > maxPromptLength {
		respondError(c, newErr(http.StatusUnprocessableEntity, fmt.Sprintf("The prompt must be between 1 and %d characters.", maxPromptLength)))
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()
	res, err := executeForGuild(ctx, c.GetString("guild_id"), *body.Prompt)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// handleCancel cancels a request that is waiting for a scheduled step. Steps that already ran are not undone.
func handleCancel(c *gin.Context) {
	guildID := c.GetString("guild_id")
	parsed, err := uuid.Parse(c.Param("command_id"))
	if err != nil {
		respondError(c, newErr(http.StatusBadRequest, "Invalid command id."))
		return
	}
	commandID := parsed.String()
	ctx, cancel := reqContext(c)
	defer cancel()

	fail := func(err error) {
		var he *httpError
		if errors.As(err, &he) {
			respondError(c, err)
			return
		}
		log.Printf("Cancel failed for %s: %v", commandID, err)
		respondError(c, newErr(http.StatusServiceUnavailable, "Could not cancel right now. Please try again."))
	}
	owner, err := rdb.Get(ctx, "command:"+commandID+":guild")
	if err != nil {
		fail(err)
		return
	}
	if owner != guildID {
		fail(newErr(http.StatusNotFound, "Request not found."))
		return
	}
	state, err := rdb.Get(ctx, "command:"+commandID+":status")
	if err != nil {
		fail(err)
		return
	}
	if state != "waiting" {
		fail(newErr(http.StatusConflict, "Only a waiting request can be cancelled."))
		return
	}
	if _, err := rdb.Set(ctx, "command:"+commandID+":cancel", "1", statusTTL, false); err != nil {
		fail(err)
		return
	}
	if _, err := rdb.Set(ctx, "command:"+commandID+":status", "cancelled", statusTTL, false); err != nil {
		fail(err)
		return
	}
	dbSetStatus(ctx, commandID, "cancelled")
	inflightKey := "guild:" + guildID + ":inflight"
	left, err := rdb.Decr(ctx, inflightKey)
	if err != nil {
		fail(err)
		return
	}
	if left <= 0 {
		if err := rdb.Del(ctx, inflightKey); err != nil {
			fail(err)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"command_id": commandID, "status": "cancelled"})
}

// handleStatus is the fast status probe used by the UI for live state tracking.
func handleStatus(c *gin.Context) {
	guildID := c.GetString("guild_id")
	parsed, err := uuid.Parse(c.Param("command_id")) // rejects anything that is not a UUID
	if err != nil {
		respondError(c, newErr(http.StatusBadRequest, "Invalid command id."))
		return
	}
	commandID := parsed.String()
	ctx, cancel := reqContext(c)
	defer cancel()
	c.Header("Cache-Control", "no-store")

	unavailable := func(err error) {
		log.Printf("Status lookup failed for %s: %v", commandID, err)
		respondError(c, newErr(http.StatusServiceUnavailable, "Status unavailable."))
	}
	owner, err := rdb.Get(ctx, "command:"+commandID+":guild")
	if err != nil {
		unavailable(err)
		return
	}
	if owner != guildID {
		c.JSON(http.StatusOK, gin.H{"command_id": commandID, "status": "unknown", "error": nil, "result": nil})
		return
	}
	state, err := rdb.Get(ctx, "command:"+commandID+":status")
	if err != nil {
		unavailable(err)
		return
	}
	errText, result := "", ""
	if failedStates[state] {
		errText, err = rdb.Get(ctx, "command:"+commandID+":error")
	} else if state == "completed" {
		result, err = rdb.Get(ctx, "command:"+commandID+":result")
	}
	if err != nil {
		unavailable(err)
		return
	}
	var errOut, resOut any
	if e := clipRunes(errText, 1200); e != "" {
		errOut = e
	}
	if r := clipRunes(result, 1500); r != "" {
		resOut = r
	}
	if state == "" {
		state = "unknown"
	}
	c.JSON(http.StatusOK, gin.H{"command_id": commandID, "status": state, "error": errOut, "result": resOut})
}

// handlePreview generates a dry run preview of all steps for a given prompt
func handlePreview(c *gin.Context) {
	guildID := c.GetString("guild_id")
	var body PreviewRequest
	if err := c.ShouldBindJSON(&body); err != nil || body.Prompt == "" {
		respondError(c, newErr(http.StatusUnprocessableEntity, "A prompt is required."))
		return
	}
	if n := utf8.RuneCountInString(body.Prompt); n < 1 || n > maxPromptLength {
		respondError(c, newErr(http.StatusUnprocessableEntity, fmt.Sprintf("The prompt must be between 1 and %d characters.", maxPromptLength)))
		return
	}

	ctx, cancel := reqContext(c)
	defer cancel()

	// Get the agent for this guild
	agent, err := heldAgent(ctx, guildID)
	if err != nil {
		respondError(c, newErr(http.StatusServiceUnavailable, "Could not determine agent."))
		return
	}
	if agent == 0 {
		respondError(c, newErr(http.StatusConflict, "You do not have an agent right now."))
		return
	}

	// Get the server snapshot
	raw, err := rdb.Get(ctx, "guild:"+guildID+":snapshot")
	if err != nil {
		respondError(c, newErr(http.StatusServiceUnavailable, "Could not get server snapshot."))
		return
	}
	snapshot := loadsMap(raw)
	if snapshot == nil {
		respondError(c, newErr(http.StatusServiceUnavailable, "Server snapshot not available."))
		return
	}

	// Generate the preview plan
	toolCfg := enabledTools()
	plan, err := GeneratePreviewPlan(ctx, agent, snapshot, body.Prompt, toolCfg)
	if err != nil {
		log.Printf("Preview generation failed: %v", err)
		// Return a minimal plan with error
		c.JSON(http.StatusOK, gin.H{
			"complete":    false,
			"incomplete":  true,
			"total_turns": 0,
			"steps":       []PreviewStep{},
			"summary":     "Could not generate preview.",
			"question":    "Can you clarify your request?",
		})
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, plan)
}
