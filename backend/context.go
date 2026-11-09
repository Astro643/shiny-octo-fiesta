package main

// Conversation context: a short memory of the last few requests from one server's Console, so a follow-up such as
// "now do the same for the voice channels" or "undo that" makes sense. Kept as a Redis list per server.
//
// What is stored: the owner's own prompt, the final status and a short result line. The result line is written by the
// model, so when the request had read web or webhook content it is NOT stored (only a note saying so). That way
// untrusted text cannot ride along into later requests. Everything here is best-effort: a Redis problem loses context,
// never a request. The memory is forgotten after contextTTLSeconds of inactivity, or at once when the owner starts a
// new conversation (POST /api/session/new).

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

const (
	contextKeep       = 5           // most recent requests remembered per server
	contextTTLSeconds = 2 * 60 * 60 // forgotten after two idle hours
	contextPromptMax  = 400         // characters kept of each prompt
	contextResultMax  = 500         // characters kept of each result line
)

type contextEntry struct {
	Prompt string `json:"p"`
	Status string `json:"s"`
	Result string `json:"r,omitempty"`
}

func contextKey(guildID string) string { return "guild:" + guildID + ":context" }

// oneLine collapses all whitespace, line breaks included, so stored text cannot imitate the prompt's own layout.
func oneLine(s string, limit int) string {
	return clipRunes(strings.Join(strings.Fields(s), " "), limit)
}

// contextWorthy: only requests that reached a verdict are remembered (not cancelled ones, nor traffic stalls).
func contextWorthy(status string) bool {
	switch status {
	case "completed", "failed", "rejected":
		return true
	}
	return false
}

// outcomeLabel is how a final status reads to the agent.
func outcomeLabel(status string) string {
	switch status {
	case "completed":
		return "done"
	case "rejected":
		return "refused"
	}
	return status
}

// rememberExchange adds one finished request to the server's memory. tainted means the request read web or webhook
// content, so its model-written result is left out.
func rememberExchange(ctx context.Context, guildID, prompt, status, text string, tainted bool) {
	if guildID == "" || strings.TrimSpace(prompt) == "" || !contextWorthy(status) {
		return
	}
	entry := contextEntry{Prompt: oneLine(prompt, contextPromptMax), Status: status}
	if tainted {
		entry.Result = "(details left out: this request used web or webhook content)"
	} else {
		entry.Result = oneLine(text, contextResultMax)
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	key := contextKey(guildID)
	if _, err := rdb.do(ctx, "RPUSH", key, string(raw)); err != nil {
		log.Printf("Could not save conversation context")
		return
	}
	_, _ = rdb.do(ctx, "LTRIM", key, -contextKeep, -1)
	_, _ = rdb.do(ctx, "EXPIRE", key, contextTTLSeconds)
}

// loadContext renders the remembered requests, oldest first, ready to drop into the agent's prompt ("" if none).
func loadContext(ctx context.Context, guildID string) string {
	if guildID == "" {
		return ""
	}
	v, err := rdb.do(ctx, "LRANGE", contextKey(guildID), 0, -1)
	if err != nil {
		log.Printf("Could not read conversation context")
		return ""
	}
	var b strings.Builder
	n := 0
	for _, raw := range asStrings(v) {
		var e contextEntry
		if json.Unmarshal([]byte(raw), &e) != nil || e.Prompt == "" {
			continue
		}
		n++
		result := e.Result
		if result == "" {
			result = "(no details)"
		}
		fmt.Fprintf(&b, "%d. OWNER ASKED: %s\n   OUTCOME (%s): %s\n", n, e.Prompt, outcomeLabel(e.Status), result)
	}
	return strings.TrimRight(b.String(), "\n")
}

// forgetContext clears the server's memory (the "New conversation" button).
func forgetContext(ctx context.Context, guildID string) error {
	return rdb.Del(ctx, contextKey(guildID))
}
