package main

// Human support chat.
//
//   Visitors   open /human-support, type a message, and get a random name + avatar tied to their browser (a signed cookie).
//              No sign-in. One conversation per browser. The first message also gets a "we usually reply within 24 hours" notice.
//   Operator   opens /reply-support, signs in with a username and password, sees every conversation by its alias only
//              (never a server ID or Discord account), and replies.
//
// Everything lives in Postgres (the same DATABASE_URL as db.go). Unlike the history store, chat must never lose a
// message silently, so these functions return errors to the caller instead of quietly doing nothing.
//
// Env (operator login; the routes answer 404 until both are set):
//   OPERATOR_USERNAME        e.g. "owner"
//   OPERATOR_PASSWORD_HASH   made with the hashpw tool (pbkdf2-sha256$...). Use a long random password.
//
// Routes (add to buildRouter, see main.go):
//   GET  /api/human/messages?after=ID   visitor: my messages since ID
//   POST /api/human/send                visitor: {"body": "..."}
//   POST /api/operator/login            {"username","password"}
//   POST /api/operator/logout
//   GET  /api/operator/inbox            all conversations
//   GET  /api/operator/thread/:id?after=ID
//   POST /api/operator/reply/:id        {"body": "..."}
//   POST /api/operator/status/:id       {"status": "open"|"closed"}
//   POST /api/operator/delete/:id       delete one conversation (for a deletion request)

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	humanMaxBody            = 1000 // characters per visitor message
	operatorMaxBody         = 2000 // characters per operator reply
	humanMaxPerConversation = 400  // messages in one conversation
	humanReplyHours         = 24
	humanRetentionDays      = 90
	humanNotice             = "Thanks for your message. Our usual reply time is within 24 hours."

	visitorCookie          = "hc_session"
	visitorCookieSeconds   = 30 * 24 * 60 * 60
	operatorCookie         = "op_session"
	operatorSessionSeconds = 2 * 60 * 60
	operatorHeader         = "X-Requested-With" // POSTs from the operator page must send this; cross-site forms cannot
	operatorHeaderValue    = "support"
)

var humanSchema = []string{
	`CREATE TABLE IF NOT EXISTS chat_conversations (
		id BIGSERIAL PRIMARY KEY,
		visitor_hash CHAR(64) NOT NULL UNIQUE,
		alias_name VARCHAR(40) NOT NULL,
		alias_avatar VARCHAR(24) NOT NULL,
		status VARCHAR(12) NOT NULL DEFAULT 'open',
		created_at TIMESTAMPTZ NOT NULL,
		last_message_at TIMESTAMPTZ NOT NULL,
		last_sender VARCHAR(10) NOT NULL DEFAULT 'visitor',
		visitor_read_id BIGINT NOT NULL DEFAULT 0,
		operator_read_id BIGINT NOT NULL DEFAULT 0
	)`,
	`CREATE INDEX IF NOT EXISTS idx_chat_conv_last ON chat_conversations (last_message_at)`,
	`CREATE TABLE IF NOT EXISTS chat_messages (
		id BIGSERIAL PRIMARY KEY,
		conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
		sender VARCHAR(10) NOT NULL,
		body VARCHAR(2000) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_chat_msg_conv ON chat_messages (conversation_id, id)`,
}

// Package-level variables are set up before init functions run, so dbInit (db.go) creates these tables too.
func init() { dbSchema = append(dbSchema, humanSchema...) }

// ---- helpers --------------------------------------------------------------------------------------

func humanDB() (*sql.DB, error) {
	if dbPool == nil {
		return nil, newErr(http.StatusServiceUnavailable, "Support chat is unavailable right now. Please try again shortly.")
	}
	return dbPool, nil
}

// humanFail answers with a plain error; a database error is logged without its details.
func humanFail(c *gin.Context, err error) {
	var he *httpError
	if errors.As(err, &he) {
		respondError(c, err)
		return
	}
	log.Printf("Human chat error on %s (%T)", c.Request.URL.Path, err)
	respondError(c, newErr(http.StatusServiceUnavailable, "Support chat is unavailable right now. Please try again shortly."))
}

func humanLimit(ctx context.Context, key string, ttl, limit int, msg string) error {
	n, err := rdb.Incr(ctx, key)
	if err != nil {
		return err // no limiter, no chat: fail closed, like the AI support chat
	}
	if n == 1 {
		if err := rdb.Expire(ctx, key, ttl); err != nil {
			return err
		}
	}
	if n > limit {
		return newErr(http.StatusTooManyRequests, msg)
	}
	return nil
}

type chatMessage struct {
	ID     int64  `json:"id"`
	Sender string `json:"sender"` // visitor | operator | system
	Body   string `json:"body"`
	At     string `json:"at"`
}

func scanMessages(rows *sql.Rows) ([]chatMessage, error) {
	defer rows.Close()
	out := []chatMessage{}
	for rows.Next() {
		var m chatMessage
		var at time.Time
		if err := rows.Scan(&m.ID, &m.Sender, &m.Body, &at); err != nil {
			return nil, err
		}
		m.At = at.UTC().Format(time.RFC3339)
		out = append(out, m)
	}
	return out, rows.Err()
}

func queryAfter(c *gin.Context) int64 {
	n, err := strconv.ParseInt(c.Query("after"), 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// ---- visitor --------------------------------------------------------------------------------------

func visitorKey() []byte { return deriveKey(sessionKey, "visitor") }

func visitorIDFromCookie(c *gin.Context) string {
	v, err := c.Cookie(visitorCookie)
	if err != nil {
		return ""
	}
	id, ok := verifyToken(visitorKey(), "visitor", v)
	if !ok || len(id) != 48 {
		return ""
	}
	return id
}

func setVisitorCookie(c *gin.Context, id string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(visitorCookie, signToken(visitorKey(), "visitor", id), visitorCookieSeconds, "/", "", cookieSecure, true)
}

// handleHumanMessages returns the visitor's alias and their messages after ?after=ID (the page polls this).
func handleHumanMessages(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	empty := gin.H{"alias": nil, "status": "none", "messages": []chatMessage{}, "reply_hours": humanReplyHours}
	id := visitorIDFromCookie(c)
	if id == "" {
		c.JSON(http.StatusOK, empty)
		return
	}
	db, err := humanDB()
	if err != nil {
		humanFail(c, err)
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()

	var convID int64
	var alias Alias
	var status string
	err = db.QueryRowContext(ctx, `SELECT id, alias_name, alias_avatar, status FROM chat_conversations WHERE visitor_hash=$1`,
		visitorHashOf(id)).Scan(&convID, &alias.Name, &alias.Avatar, &status)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusOK, empty)
		return
	}
	if err != nil {
		humanFail(c, err)
		return
	}
	rows, err := db.QueryContext(ctx, `SELECT id, sender, body, created_at FROM chat_messages WHERE conversation_id=$1 AND id>$2 ORDER BY id LIMIT 200`,
		convID, queryAfter(c))
	if err != nil {
		humanFail(c, err)
		return
	}
	msgs, err := scanMessages(rows)
	if err != nil {
		humanFail(c, err)
		return
	}
	if len(msgs) > 0 {
		_, _ = db.ExecContext(ctx, `UPDATE chat_conversations SET visitor_read_id = GREATEST(visitor_read_id, $1) WHERE id=$2`,
			msgs[len(msgs)-1].ID, convID)
	}
	c.JSON(http.StatusOK, gin.H{"alias": alias, "status": status, "messages": msgs, "reply_hours": humanReplyHours})
}

// handleHumanSend saves a visitor message. The first one creates the conversation, picks the alias and adds the notice.
func handleHumanSend(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var in struct {
		Body *string `json:"body"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Body == nil {
		respondError(c, newErr(http.StatusUnprocessableEntity, "A message is required."))
		return
	}
	text, ok := cleanBody(*in.Body, humanMaxBody)
	if !ok {
		respondError(c, newErr(http.StatusUnprocessableEntity, fmt.Sprintf("Messages must be between 1 and %d characters.", humanMaxBody)))
		return
	}
	db, err := humanDB()
	if err != nil {
		humanFail(c, err)
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()

	visitor := visitorHash(c) // salted hash of the IP, used only for rate limits
	tooFast := "You are sending messages too quickly. Please wait a little and try again."
	for _, l := range []struct {
		label      string
		ttl, limit int
	}{{"10m", 600, 10}, {"day", 86400, 80}} {
		if err := humanLimit(ctx, fmt.Sprintf("rl:human:%s:%s", l.label, visitor), l.ttl, l.limit, tooFast); err != nil {
			humanFail(c, err)
			return
		}
	}
	if err := humanLimit(ctx, fmt.Sprintf("rl:human:global:%d", time.Now().Unix()/3600), 3600, 400,
		"Support is very busy right now. Please try again later."); err != nil {
		humanFail(c, err)
		return
	}

	id := visitorIDFromCookie(c)
	newCookie := false
	if id == "" {
		if id, err = newVisitorID(); err != nil {
			humanFail(c, err)
			return
		}
		newCookie = true
	}
	hash := visitorHashOf(id)

	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chat_conversations WHERE visitor_hash=$1)`, hash).Scan(&exists); err != nil {
		humanFail(c, err)
		return
	}
	taken := map[string]bool{}
	if !exists {
		// Starting a conversation is limited on its own, so one person cannot open many.
		if err := humanLimit(ctx, "rl:human:newconv:"+visitor, 86400, 3, "Too many new conversations from your connection today."); err != nil {
			humanFail(c, err)
			return
		}
		if err := humanLimit(ctx, fmt.Sprintf("rl:human:newconv:global:%d", time.Now().Unix()/3600), 3600, 100,
			"Support is very busy right now. Please try again later."); err != nil {
			humanFail(c, err)
			return
		}
		rows, err := db.QueryContext(ctx, `SELECT alias_name FROM chat_conversations WHERE status='open'`)
		if err != nil {
			humanFail(c, err)
			return
		}
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				rows.Close()
				humanFail(c, err)
				return
			}
			taken[n] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			humanFail(c, err)
			return
		}
	}

	alias := randomAlias(taken)
	now := dbNow()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		humanFail(c, err)
		return
	}
	defer tx.Rollback() // no effect after a successful Commit

	var convID int64
	var inserted bool
	var out Alias
	err = tx.QueryRowContext(ctx, `INSERT INTO chat_conversations (visitor_hash, alias_name, alias_avatar, status, created_at, last_message_at, last_sender)
		VALUES ($1,$2,$3,'open',$4,$4,'visitor')
		ON CONFLICT (visitor_hash) DO UPDATE SET status='open', last_message_at=EXCLUDED.last_message_at, last_sender='visitor'
		RETURNING id, alias_name, alias_avatar, (xmax = 0)`, hash, alias.Name, alias.Avatar, now).Scan(&convID, &out.Name, &out.Avatar, &inserted)
	if err != nil {
		humanFail(c, err)
		return
	}
	if !inserted {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_messages WHERE conversation_id=$1`, convID).Scan(&count); err != nil {
			humanFail(c, err)
			return
		}
		if count >= humanMaxPerConversation {
			humanFail(c, newErr(http.StatusTooManyRequests, "This conversation is full. Please wait for our reply."))
			return
		}
	}
	msgs := []chatMessage{}
	add := func(sender, body string, at time.Time) error {
		m := chatMessage{Sender: sender, Body: body, At: at.UTC().Format(time.RFC3339)}
		if err := tx.QueryRowContext(ctx, `INSERT INTO chat_messages (conversation_id, sender, body, created_at) VALUES ($1,$2,$3,$4) RETURNING id`,
			convID, sender, body, at).Scan(&m.ID); err != nil {
			return err
		}
		msgs = append(msgs, m)
		return nil
	}
	if err := add("visitor", text, now); err != nil {
		humanFail(c, err)
		return
	}
	if inserted {
		if err := add("system", humanNotice, now.Add(time.Millisecond)); err != nil {
			humanFail(c, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		humanFail(c, err)
		return
	}
	if newCookie {
		setVisitorCookie(c, id)
	}
	c.JSON(http.StatusOK, gin.H{"alias": out, "messages": msgs, "reply_hours": humanReplyHours})
}

// ---- operator -------------------------------------------------------------------------------------

func operatorUser() string  { return envStr("OPERATOR_USERNAME") }
func operatorHash() string  { return envStr("OPERATOR_PASSWORD_HASH") }
func operatorEnabled() bool { return operatorUser() != "" && operatorHash() != "" }
func operatorKey() []byte   { return deriveKey(sessionKey, "operator") }
func notFound(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"detail": "Not found"})
}

// requireOperator: every operator route (except login) goes through this. A valid, unexpired, signed operator cookie is
// needed, and any request that changes something must also carry the custom header.
func requireOperator() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if !operatorEnabled() {
			notFound(c)
			return
		}
		cookie, _ := c.Cookie(operatorCookie)
		payload, ok := verifyToken(operatorKey(), "operator", cookie)
		exp, err := strconv.ParseInt(payload, 10, 64)
		if !ok || err != nil || time.Now().Unix() > exp {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Please sign in."})
			return
		}
		if c.Request.Method != http.MethodGet && c.GetHeader(operatorHeader) != operatorHeaderValue {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "Forbidden."})
			return
		}
		c.Next()
	}
}

func handleOperatorLogin(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if !operatorEnabled() {
		notFound(c)
		return
	}
	var in struct {
		Username *string `json:"username"`
		Password *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Username == nil || in.Password == nil ||
		len(*in.Username) > 200 || len(*in.Password) > 200 {
		respondError(c, newErr(http.StatusUnprocessableEntity, "Enter your username and password."))
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()
	// Guesses are limited per connection and across the whole site, and counted before any hashing work is done.
	ipKey := "rl:operator:login:" + visitorHash(c)
	if err := humanLimit(ctx, ipKey, 900, 10, "Too many attempts. Wait a few minutes."); err != nil {
		humanFail(c, err)
		return
	}
	if err := humanLimit(ctx, "rl:operator:login:global", 900, 100, "Too many attempts. Wait a few minutes."); err != nil {
		humanFail(c, err)
		return
	}
	userOK := subtle.ConstantTimeCompare([]byte(strings.ToLower(strings.TrimSpace(*in.Username))), []byte(strings.ToLower(operatorUser()))) == 1
	hash := operatorHash()
	if !userOK {
		hash = dummyPasswordHash // same work either way, so timing does not reveal whether the username exists
	}
	passOK := verifyPassword(hash, *in.Password)
	if !userOK || !passOK {
		log.Printf("Operator sign-in failed")
		respondError(c, newErr(http.StatusUnauthorized, "Wrong username or password."))
		return
	}
	_ = rdb.Del(ctx, ipKey)
	exp := time.Now().Add(operatorSessionSeconds * time.Second).Unix()
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(operatorCookie, signToken(operatorKey(), "operator", strconv.FormatInt(exp, 10)), operatorSessionSeconds, "/", "", cookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func handleOperatorLogout(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(operatorCookie, "", -1, "/", "", cookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type convSummary struct {
	ID         int64  `json:"id"`
	Alias      Alias  `json:"alias"`
	Status     string `json:"status"`
	LastAt     string `json:"last_at"`
	LastSender string `json:"last_sender"`
	Unread     int    `json:"unread"`
	Preview    string `json:"preview"`
}

// handleOperatorInbox lists every conversation, newest first. The page builds its tabs (unanswered, answered, closed) from it.
func handleOperatorInbox(c *gin.Context) {
	db, err := humanDB()
	if err != nil {
		humanFail(c, err)
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()
	rows, err := db.QueryContext(ctx, `SELECT c.id, c.alias_name, c.alias_avatar, c.status, c.last_message_at, c.last_sender,
		(SELECT COUNT(*) FROM chat_messages m WHERE m.conversation_id=c.id AND m.sender='visitor' AND m.id > c.operator_read_id),
		COALESCE((SELECT m.body FROM chat_messages m WHERE m.conversation_id=c.id AND m.sender <> 'system' ORDER BY m.id DESC LIMIT 1), '')
		FROM chat_conversations c ORDER BY c.last_message_at DESC LIMIT 300`)
	if err != nil {
		humanFail(c, err)
		return
	}
	defer rows.Close()
	out := []convSummary{}
	for rows.Next() {
		var s convSummary
		var at time.Time
		var preview string
		if err := rows.Scan(&s.ID, &s.Alias.Name, &s.Alias.Avatar, &s.Status, &at, &s.LastSender, &s.Unread, &preview); err != nil {
			humanFail(c, err)
			return
		}
		s.LastAt = at.UTC().Format(time.RFC3339)
		s.Preview = oneLine(preview, 90)
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		humanFail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": out})
}

func convIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		respondError(c, newErr(http.StatusBadRequest, "Invalid conversation."))
		return 0, false
	}
	return id, true
}

// handleOperatorThread returns one conversation's messages after ?after=ID and marks them read.
func handleOperatorThread(c *gin.Context) {
	id, ok := convIDParam(c)
	if !ok {
		return
	}
	db, err := humanDB()
	if err != nil {
		humanFail(c, err)
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()
	var alias Alias
	var status string
	err = db.QueryRowContext(ctx, `SELECT alias_name, alias_avatar, status FROM chat_conversations WHERE id=$1`, id).Scan(&alias.Name, &alias.Avatar, &status)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(c, newErr(http.StatusNotFound, "Conversation not found."))
		return
	}
	if err != nil {
		humanFail(c, err)
		return
	}
	rows, err := db.QueryContext(ctx, `SELECT id, sender, body, created_at FROM chat_messages WHERE conversation_id=$1 AND id>$2 ORDER BY id LIMIT 500`, id, queryAfter(c))
	if err != nil {
		humanFail(c, err)
		return
	}
	msgs, err := scanMessages(rows)
	if err != nil {
		humanFail(c, err)
		return
	}
	if len(msgs) > 0 {
		_, _ = db.ExecContext(ctx, `UPDATE chat_conversations SET operator_read_id = GREATEST(operator_read_id, $1) WHERE id=$2`, msgs[len(msgs)-1].ID, id)
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "alias": alias, "status": status, "messages": msgs})
}

func handleOperatorReply(c *gin.Context) {
	id, ok := convIDParam(c)
	if !ok {
		return
	}
	var in struct {
		Body *string `json:"body"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Body == nil {
		respondError(c, newErr(http.StatusUnprocessableEntity, "A message is required."))
		return
	}
	text, ok := cleanBody(*in.Body, operatorMaxBody)
	if !ok {
		respondError(c, newErr(http.StatusUnprocessableEntity, fmt.Sprintf("Replies must be between 1 and %d characters.", operatorMaxBody)))
		return
	}
	db, err := humanDB()
	if err != nil {
		humanFail(c, err)
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()
	now := dbNow()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		humanFail(c, err)
		return
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE chat_conversations SET status='open', last_message_at=$1, last_sender='operator' WHERE id=$2`, now, id)
	if err != nil {
		humanFail(c, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		respondError(c, newErr(http.StatusNotFound, "Conversation not found."))
		return
	}
	m := chatMessage{Sender: "operator", Body: text, At: now.UTC().Format(time.RFC3339)}
	if err := tx.QueryRowContext(ctx, `INSERT INTO chat_messages (conversation_id, sender, body, created_at) VALUES ($1,'operator',$2,$3) RETURNING id`,
		id, text, now).Scan(&m.ID); err != nil {
		humanFail(c, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `UPDATE chat_conversations SET operator_read_id = GREATEST(operator_read_id, $1) WHERE id=$2`, m.ID, id); err != nil {
		humanFail(c, err)
		return
	}
	if err := tx.Commit(); err != nil {
		humanFail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": m})
}

func handleOperatorStatus(c *gin.Context) {
	id, ok := convIDParam(c)
	if !ok {
		return
	}
	var in struct {
		Status *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Status == nil || (*in.Status != "open" && *in.Status != "closed") {
		respondError(c, newErr(http.StatusUnprocessableEntity, "Status must be open or closed."))
		return
	}
	db, err := humanDB()
	if err != nil {
		humanFail(c, err)
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()
	res, err := db.ExecContext(ctx, `UPDATE chat_conversations SET status=$1 WHERE id=$2`, *in.Status, id)
	if err != nil {
		humanFail(c, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		respondError(c, newErr(http.StatusNotFound, "Conversation not found."))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "status": *in.Status})
}

// handleOperatorDelete removes a conversation and all its messages (for a deletion request).
func handleOperatorDelete(c *gin.Context) {
	id, ok := convIDParam(c)
	if !ok {
		return
	}
	db, err := humanDB()
	if err != nil {
		humanFail(c, err)
		return
	}
	ctx, cancel := reqContext(c)
	defer cancel()
	if _, err := db.ExecContext(ctx, `DELETE FROM chat_conversations WHERE id=$1`, id); err != nil {
		humanFail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// humanPurgeOld deletes conversations with no activity for humanRetentionDays (messages go with them).
func humanPurgeOld(ctx context.Context) {
	cutoff := dbNow().Add(-humanRetentionDays * 24 * time.Hour)
	dbExec(ctx, "DELETE FROM chat_conversations WHERE last_message_at < $1", cutoff)
}
