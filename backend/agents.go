package main

// The four AI agents (website only). Agent 1-3 serve console sessions; Agent 4 answers support, Agent 3 backs it up.
// Each agent has its own key and models:
//   GEMINI_API_KEY_n, PRIMARY_MODEL_n, FALLBACK_MODEL_n   (n = 1..4)
// PRIMARY_MODEL / FALLBACK_MODEL (no number) are optional defaults for all agents.
// Health is written to Redis as status:agent:{n} for the status page.
//
// Gemini is called over its plain REST API (generativelanguage.googleapis.com), so there is no SDK to keep in sync.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultPrimary  = "gemini-3.6-flash"
	defaultFallback = "gemini-3.5-flash-lite"
	agentStatusTTL  = 60 * 30
)

var (
	sessionAgents = []int{1, 2, 3}
	retryDelays   = []float64{1.5, 4.0} // seconds to wait before retrying a busy model, twice at most
	probeSeconds  = 900
	agentPool     = map[int]*Agent{}
	geminiHTTP    = &http.Client{}
)

type Agent struct {
	N        int
	Key      string
	Primary  string
	Fallback string
	mu       sync.Mutex
	fails    int
}

func (a *Agent) Configured() bool { return a.Key != "" }

func initAgents() {
	for n := 1; n <= 4; n++ {
		primary := envStr(fmt.Sprintf("PRIMARY_MODEL_%d", n))
		if primary == "" {
			primary = envStr("PRIMARY_MODEL")
		}
		if primary == "" {
			primary = defaultPrimary
		}
		fallback := envStr(fmt.Sprintf("FALLBACK_MODEL_%d", n))
		if fallback == "" {
			fallback = envStr("FALLBACK_MODEL")
		}
		if fallback == "" {
			fallback = defaultFallback
		}
		agentPool[n] = &Agent{N: n, Key: envStr(fmt.Sprintf("GEMINI_API_KEY_%d", n)), Primary: primary, Fallback: fallback}
	}
	probeSeconds = 900
	if v := envStr("STATUS_PROBE_SECONDS"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			probeSeconds = n
		}
	}
	if probeSeconds < 300 {
		probeSeconds = 300
	}
}

func agentConfigured(n int) bool {
	a, ok := agentPool[n]
	return ok && a.Configured()
}

// ---- Gemini REST ----------------------------------------------------------------------------------

type GenRequest struct {
	System          string
	Contents        string
	Temperature     float64
	MaxOutputTokens int
	ResponseSchema  map[string]any // when set, the reply is forced to JSON matching this schema
	Search          bool           // Google Search grounding
}

type GenResponse struct {
	Text    string
	Sources []string // "title: uri", at most 5
}

// APIError is a non-2xx answer from the Gemini API.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gemini HTTP %d %s: %s", e.Status, e.Code, e.Message)
}

func callGemini(ctx context.Context, key, model string, req GenRequest) (*GenResponse, error) {
	gen := map[string]any{"temperature": req.Temperature}
	if req.MaxOutputTokens > 0 {
		gen["maxOutputTokens"] = req.MaxOutputTokens
	}
	if req.ResponseSchema != nil {
		gen["responseMimeType"] = "application/json"
		gen["responseSchema"] = req.ResponseSchema
	}
	body := map[string]any{
		"contents":         []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": req.Contents}}}},
		"generationConfig": gen,
	}
	if req.System != "" {
		body["systemInstruction"] = map[string]any{"parts": []any{map[string]any{"text": req.System}}}
	}
	if req.Search {
		body["tools"] = []any{map[string]any{"google_search": map[string]any{}}}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + url.PathEscape(model) + ":generateContent"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", key) // header, not query string, so the key never lands in an error message
	resp, err := geminiHTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &e)
		return nil, &APIError{Status: resp.StatusCode, Code: e.Error.Status, Message: e.Error.Message}
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
			GroundingMetadata struct {
				GroundingChunks []struct {
					Web struct {
						URI   string `json:"uri"`
						Title string `json:"title"`
					} `json:"web"`
				} `json:"groundingChunks"`
			} `json:"groundingMetadata"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("gemini: unreadable response")
	}
	out := &GenResponse{}
	if len(parsed.Candidates) > 0 {
		c := parsed.Candidates[0]
		var sb strings.Builder
		for _, p := range c.Content.Parts {
			if !p.Thought {
				sb.WriteString(p.Text)
			}
		}
		out.Text = sb.String()
		for _, chunk := range c.GroundingMetadata.GroundingChunks {
			if chunk.Web.URI != "" && len(out.Sources) < 5 {
				title := chunk.Web.Title
				if title == "" {
					title = "source"
				}
				out.Sources = append(out.Sources, title+": "+chunk.Web.URI)
			}
		}
	}
	return out, nil
}

// ---- error handling -------------------------------------------------------------------------------

// classifyError gives a short, non-sensitive category for a failure (never the raw message).
func classifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout"
	}
	text := strings.ToUpper(err.Error())
	var api *APIError
	if errors.As(err, &api) {
		if api.Status == 429 {
			return "rate_limited"
		}
		if api.Status == 500 || api.Status == 502 || api.Status == 503 || api.Status == 504 {
			return "unavailable"
		}
	}
	if strings.Contains(text, "429") || strings.Contains(text, "RESOURCE_EXHAUSTED") || strings.Contains(text, "QUOTA") {
		return "rate_limited"
	}
	for _, code := range []string{"500", "502", "503", "504", "UNAVAILABLE", "OVERLOADED"} {
		if strings.Contains(text, code) {
			return "unavailable"
		}
	}
	return "error"
}

// quotaSpent: a per-day or billing quota will not recover in seconds, so retrying the same model is pointless.
func quotaSpent(err error) bool {
	text := strings.ToUpper(err.Error())
	return strings.Contains(text, "PER DAY") || strings.Contains(text, "PERDAY") ||
		strings.Contains(text, "BILLING") || strings.Contains(text, "DAILY")
}

func nowSeconds() float64 { return float64(time.Now().UnixNano()) / 1e9 }

func (a *Agent) record(ok bool, started time.Time, role string, err error) {
	a.mu.Lock()
	if ok {
		a.fails = 0
	} else {
		a.fails++
	}
	fails := a.fails
	a.mu.Unlock()
	payload, _ := json.Marshal(map[string]any{
		"ok": ok, "ts": nowSeconds(), "ms": time.Since(started).Milliseconds(),
		"fails": fails, "err": classifyError(err), "role": role,
	})
	// A fresh context: the request's own context may already be cancelled (that is often the failure being recorded).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, e := rdb.Set(ctx, fmt.Sprintf("status:agent:%d", a.N), string(payload), agentStatusTTL, false); e != nil {
		log.Printf("Could not publish health for agent %d", a.N) // health reporting must never break real work
	}
}

// generate calls agent n with its primary model, then its fallback model. parse may return an error to force the fallback.
func generate[T any](ctx context.Context, n int, req GenRequest, timeout time.Duration, parse func(*GenResponse) (T, error)) (T, error) {
	var zero T
	agent := agentPool[n]
	if agent == nil || !agent.Configured() {
		return zero, fmt.Errorf("agent %d is not configured", n)
	}
	type modelRole struct{ role, model string }
	models := []modelRole{{"primary", agent.Primary}}
	if agent.Fallback != "" && agent.Fallback != agent.Primary {
		models = append(models, modelRole{"fallback", agent.Fallback})
	}
	var last error
	for _, m := range models {
		for attempt := 0; attempt <= len(retryDelays); attempt++ {
			started := time.Now()
			result, err := func() (T, error) {
				cctx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()
				resp, err := callGemini(cctx, agent.Key, m.model, req)
				if err != nil {
					return zero, err
				}
				return parse(resp)
			}()
			if err == nil {
				agent.record(true, started, m.role, nil)
				return result, nil
			}
			last = err
			kind := classifyError(err)
			if kind == "" {
				kind = "error"
			}
			log.Printf("Agent %d %s model failed (%s, attempt %d)", n, m.role, kind, attempt+1)
			agent.record(false, started, m.role, err)
			// "Busy" errors are usually momentary: wait briefly and try the same model again.
			// A spent daily quota is not, so skip straight to the fallback.
			if (kind == "rate_limited" || kind == "unavailable" || kind == "timeout") && attempt < len(retryDelays) && !quotaSpent(err) {
				wait := time.Duration(retryDelays[attempt] * (0.75 + rand.Float64()/2) * float64(time.Second))
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return zero, ctx.Err()
				}
				continue
			}
			break
		}
	}
	return zero, last
}

// ---- health ---------------------------------------------------------------------------------------

type agentHealth struct {
	OK    bool    `json:"ok"`
	TS    float64 `json:"ts"`
	MS    float64 `json:"ms"`
	Fails int     `json:"fails"`
	Err   string  `json:"err"`
	Role  string  `json:"role"`
}

func parseHealth(raw string) *agentHealth {
	if raw == "" {
		return nil
	}
	var h agentHealth
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		return nil
	}
	return &h
}

// agentState returns (status, detail) for one agent from its last recorded result.
func agentState(h *agentHealth, now float64) (string, string) {
	if h == nil || now-h.TS > agentStatusTTL {
		return "unknown", "Checking"
	}
	if h.OK {
		secs := fmt.Sprintf("%.1fs", h.MS/1000)
		if h.Role == "fallback" {
			return "degraded", "Fallback model · " + secs
		}
		return "operational", "Responding in " + secs
	}
	reason, found := map[string]string{
		"rate_limited": "Rate limited", "unavailable": "Service unavailable", "timeout": "Timing out",
	}[h.Err]
	if !found {
		reason = "Not responding"
	}
	if h.Fails >= 2 {
		return "outage", reason
	}
	return "degraded", reason
}

func probeOne(ctx context.Context, n int) {
	raw, err := rdb.Get(ctx, fmt.Sprintf("status:agent:%d", n))
	if err == nil {
		if h := parseHealth(raw); h != nil && h.OK && nowSeconds()-h.TS < float64(probeSeconds) {
			return // real traffic already proved it works recently: save the quota
		}
	}
	// Failures are recorded by generate itself.
	_, _ = generate(ctx, n, GenRequest{Contents: "Reply with the single word: ok", Temperature: 0, MaxOutputTokens: 8},
		9*time.Second, func(*GenResponse) (struct{}, error) { return struct{}{}, nil })
}

// probeLoop keeps the status page honest while nobody is using an agent. Cheap: skipped when traffic already proves health.
func probeLoop(ctx context.Context) {
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		return
	}
	for {
		for n := 1; n <= 4; n++ {
			if agentConfigured(n) {
				probeOne(ctx, n)
			}
		}
		select {
		case <-time.After(time.Duration(probeSeconds) * time.Second):
		case <-ctx.Done():
			return
		}
	}
}
