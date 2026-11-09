package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ---- environment ----------------------------------------------------------------------------------

func envStr(name string) string { return strings.TrimSpace(os.Getenv(name)) }

// envFirst returns the first non-empty env var among names.
func envFirst(names ...string) (string, error) {
	for _, n := range names {
		if v := envStr(n); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("missing environment variable %s (accepted names: %s)", names[0], strings.Join(names, ", "))
}

// urlEnv reads a URL setting, tolerant of a missing https:// and a trailing slash.
func urlEnv(name string) string {
	v := strings.TrimRight(strings.Trim(strings.TrimSpace(os.Getenv(name)), "\"'"), "/")
	if v == "" || strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	return "https://" + v
}

// secretEnv reads a secret, tolerant of stray spaces, line breaks and wrapping quotes from copy and paste.
func secretEnv(name string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(os.Getenv(name)), "\"'"))
}

// ---- text -----------------------------------------------------------------------------------------

func clipRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func tailRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
}

// clip shortens long observations the way the Python version did.
func clip(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	return string([]rune(s)[:limit]) + " …(cut)"
}

// jsonString marshals without HTML escaping (so <, > and & stay readable in prompts).
func jsonString(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return ""
	}
	return strings.TrimRight(buf.String(), "\n")
}

// pyRepr approximates Python's repr() of a string (used in the agent's progress log).
func pyRepr(s string) string {
	quote := "'"
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		quote = `"`
	}
	var b strings.Builder
	b.WriteString(quote)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if string(r) == quote {
				b.WriteString(`\`)
			}
			b.WriteRune(r)
		}
	}
	b.WriteString(quote)
	return b.String()
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ---- loosely typed JSON ---------------------------------------------------------------------------

// asStr renders a decoded JSON value as text. json.Number keeps big Discord IDs exact.
func asStr(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

// pyStr mirrors Python's str(): None becomes "None".
func pyStr(v any) string {
	if v == nil {
		return "None"
	}
	return asStr(v)
}

// loadsMap decodes a JSON object (numbers kept exact). Anything else returns nil.
func loadsMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil
	}
	return m
}

// head returns at most n items of a decoded JSON array (empty if v is not an array).
func head(v any, n int) []any {
	list, ok := v.([]any)
	if !ok {
		return []any{}
	}
	if len(list) > n {
		return list[:n]
	}
	return list
}

// ---- HTTP errors ----------------------------------------------------------------------------------

// httpError is an error that carries the status code and {"detail": ...} message to send.
type httpError struct {
	Status int
	Detail string
}

func (e *httpError) Error() string { return e.Detail }

func newErr(status int, detail string) *httpError { return &httpError{Status: status, Detail: detail} }
