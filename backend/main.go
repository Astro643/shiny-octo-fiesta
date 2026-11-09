package main

// Entry point: configuration, shared globals, routing, startup and shutdown, plus the small routes that were
// left over from dashboard.py (/healthz, the operator command route, the JSON 404).
//
// Env: REDIS_URL / REDIS_TOKEN (or UPSTASH_REDIS_REST_URL / UPSTASH_REDIS_REST_TOKEN), SESSION_SECRET,
// COOKIE_SECURE, SITE_NAME, SUPPORT_EMAIL, ADMIN_API_KEY, PORT. The rest is read by agents.go, db.go,
// tools.go and workflow.go.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const (
	cookieName         = "session_guild_id"
	cookieMaxAge       = 60 * 60 * 12 // 12-hour session
	maxPendingCommands = 10           // per-guild requests in flight at once
	waitlistKey        = "agent:waitlist"
	leaseSeconds       = 120    // an agent is released this long after the last ping
	leaveGraceSeconds  = 15     // a closed console frees its agent after this
	waitAliveSeconds   = 20     // a waiting console must ping at least this often
	idleSeconds        = 20 * 60 // an idle session yields its agent if people are waiting
	maxWaiting         = 50
	maxPromptLength    = 2000
	killSwitchKey      = "kill:switch" // set by the operator: no new or running requests proceed
)

var adminCommands = []string{"help", "status", "pause", "resume", "block", "unblock", "blocklist", "audit", "purge-guild"}

var (
	rdb          *Redis
	wf           *Workflow
	sessionKey   []byte // HMAC key that signs the session cookie, so a guild id cannot be forged
	cookieSecure bool
	siteName     string
	supportEmail string
	adminAPIKey  string // operator key for POST /api/admin/command (the route is off if empty)
)

// loadConfig reads the settings every part of the site shares. It fails with a readable message.
func loadConfig() error {
	redisURL, err := envFirst("REDIS_URL", "UPSTASH_REDIS_REST_URL")
	if err != nil {
		return err
	}
	redisToken, err := envFirst("REDIS_TOKEN", "UPSTASH_REDIS_REST_TOKEN")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(redisURL, "http://") && !strings.HasPrefix(redisURL, "https://") {
		return errors.New("REDIS_URL must be the Upstash REST URL (https://<name>.upstash.io), " +
			"not the redis:// / rediss:// connection string")
	}
	rdb = NewRedis(redisURL, redisToken)

	if secret := os.Getenv("SESSION_SECRET"); secret != "" {
		sessionKey = []byte(secret)
	} else {
		sum := sha256.Sum256([]byte("dash-session:" + redisToken))
		sessionKey = []byte(hex.EncodeToString(sum[:]))
	}
	cookieSecure = strings.ToLower(envOr("COOKIE_SECURE", "true")) != "false"
	siteName = envStr("SITE_NAME")
	if siteName == "" {
		siteName = "Controller"
	}
	supportEmail = envStr("SUPPORT_EMAIL")
	adminAPIKey = envStr("ADMIN_API_KEY")
	return nil
}

// envOr is os.environ.get(name, def): the default applies only when the variable is not set at all.
func envOr(name, def string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return def
}

// accessLog logs method, path (never the query string: /login?token=... carries a one-time secret), status, time.
func accessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start).Round(time.Millisecond))
	}
}

func buildRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), accessLog())
	r.HandleMethodNotAllowed = true

	// This server is API-only: every unknown path is a plain JSON 404.
	r.NoRoute(func(c *gin.Context) { c.JSON(http.StatusNotFound, gin.H{"detail": "Not found"}) })
	r.NoMethod(func(c *gin.Context) { c.JSON(http.StatusMethodNotAllowed, gin.H{"detail": "Method Not Allowed"}) })

	r.GET("/api/public/status", handlePublicStatus)
	r.POST("/api/support/chat", handleSupportChat)
	r.GET("/login", handleLogin)

	guild := r.Group("/", requireGuild())
	guild.POST("/api/session/ping", handleSessionPing)
	guild.POST("/api/session/leave", handleSessionLeave)
	guild.POST("/api/session/new", handleSessionNew)
	guild.POST("/api/execute", handleExecute)
	guild.POST("/api/cancel/:command_id", handleCancel)
	guild.GET("/api/status/:command_id", handleStatus)
	guild.POST("/api/preview", handlePreview)

	r.POST("/api/admin/command", handleAdminCommand)

	// Human support chat: visitors need no sign-in; the operator signs in at /team (see humanchat.go).
	r.GET("/api/human/messages", handleHumanMessages)
	r.POST("/api/human/send", handleHumanSend)
	r.POST("/api/operator/login", handleOperatorLogin)
	op := r.Group("/api/operator", requireOperator())
	op.POST("/logout", handleOperatorLogout)
	op.GET("/inbox", handleOperatorInbox)
	op.GET("/thread/:id", handleOperatorThread)
	op.POST("/reply/:id", handleOperatorReply)
	op.POST("/status/:id", handleOperatorStatus)
	op.POST("/delete/:id", handleOperatorDelete)

	health := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) } // Render health check
	r.GET("/healthz", health)
	r.HEAD("/healthz", health)

	if wf.Available() && wf.client != nil {
		handler, err := wf.Register()
		if err != nil {
			log.Printf("Could not register the task engine: %v", err)
		} else {
			r.Any("/api/inngest", gin.WrapH(handler))
		}
	}
	return r
}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	initAgents()
	wf = newWorkflow()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go probeLoop(ctx) // keeps the status page honest while nobody is using an agent
	dbInit(ctx)       // optional MySQL history; the site works without it
	if dbConfigured() {
		go dbPurgeLoop(ctx)
	}

	port := envStr("PORT")
	if port == "" {
		port = "8000"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           buildRouter(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	log.Printf("Listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	dbClose()
}

// ---- operator command route -----------------------------------------------------------------------

type adminCommandRequest struct {
	Command *string `json:"command"`
}

// requireAdmin: the route does not exist unless ADMIN_API_KEY is set, and failed guesses are rate limited.
func requireAdmin(ctx context.Context, c *gin.Context) error {
	if adminAPIKey == "" {
		return newErr(http.StatusNotFound, "Not found") // feature off: look like any unknown path
	}
	key := "rl:admin:" + visitorHash(c)
	attempts, err := rdb.Incr(ctx, key)
	if err != nil {
		return err
	}
	if attempts == 1 {
		if err := rdb.Expire(ctx, key, 600); err != nil {
			return err
		}
	}
	if attempts > 30 {
		return newErr(http.StatusTooManyRequests, "Too many attempts. Wait a few minutes.")
	}
	header := c.GetHeader("Authorization")
	given := ""
	if len(header) > 7 && strings.EqualFold(header[:7], "bearer ") {
		given = strings.TrimSpace(header[7:])
	}
	if given == "" || subtle.ConstantTimeCompare([]byte(given), []byte(adminAPIKey)) != 1 {
		return newErr(http.StatusUnauthorized, "Invalid admin key.")
	}
	return nil
}

// splitShell splits a command line like Python's shlex.split (POSIX rules, no comments).
func splitShell(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inToken := false
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch r {
		case ' ', '\t', '\n', '\r':
			if inToken {
				args = append(args, cur.String())
				cur.Reset()
				inToken = false
			}
		case '\\':
			if i+1 >= len(rs) {
				return nil, errors.New("no escaped character")
			}
			i++
			cur.WriteRune(rs[i])
			inToken = true
		case '\'':
			inToken = true
			j := i + 1
			for j < len(rs) && rs[j] != '\'' {
				cur.WriteRune(rs[j])
				j++
			}
			if j >= len(rs) {
				return nil, errors.New("no closing quotation")
			}
			i = j
		case '"':
			inToken = true
			j := i + 1
			closed := false
			for j < len(rs) {
				if rs[j] == '"' {
					closed = true
					break
				}
				if rs[j] == '\\' && j+1 < len(rs) && (rs[j+1] == '"' || rs[j+1] == '\\') {
					j++ // inside double quotes a backslash only escapes " and \
				}
				cur.WriteRune(rs[j])
				j++
			}
			if !closed {
				return nil, errors.New("no closing quotation")
			}
			i = j
		default:
			cur.WriteRune(r)
			inToken = true
		}
	}
	if inToken {
		args = append(args, cur.String())
	}
	return args, nil
}

// prettyJSON is json.dumps(data, indent=2) without HTML escaping.
func prettyJSON(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return ""
	}
	return strings.TrimRight(buf.String(), "\n")
}

// orderedStrings is a JSON object that keeps the order its keys were added in.
type orderedStrings struct{ keys, vals []string }

func (o orderedStrings) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(jsonString(k) + ":" + jsonString(o.vals[i]))
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

type adminStatusOut struct {
	Overall        string         `json:"overall"`
	Components     orderedStrings `json:"components"`
	Paused         bool           `json:"paused"`
	BlockedServers int            `json:"blocked_servers"`
	HistoryDB      string         `json:"history_db"`
}

func handleAdminCommand(c *gin.Context) {
	ctx, cancel := reqContext(c)
	defer cancel()
	if err := requireAdmin(ctx, c); err != nil {
		respondError(c, err)
		return
	}
	var body adminCommandRequest
	if err := c.ShouldBindJSON(&body); err != nil || body.Command == nil ||
		utf8.RuneCountInString(*body.Command) < 1 || utf8.RuneCountInString(*body.Command) > 300 {
		respondError(c, newErr(http.StatusUnprocessableEntity, "Invalid request body."))
		return
	}
	command := strings.TrimSpace(*body.Command)
	args, err := splitShell(command)
	if err != nil {
		respondError(c, newErr(http.StatusBadRequest, "Command could not be parsed."))
		return
	}
	if len(args) == 0 || !inList(adminCommands, strings.ToLower(args[0])) {
		respondError(c, newErr(http.StatusForbidden, "Unknown admin command. Try: "+strings.Join(adminCommands, ", ")))
		return
	}
	name, rest := strings.ToLower(args[0]), args[1:]

	reply := func(data any) {
		text, isString := data.(string)
		if !isString {
			text = prettyJSON(data)
		}
		c.JSON(http.StatusOK, gin.H{"stdout": text, "stderr": "", "exit_code": 0})
	}
	// needGuild returns the server ID the command was given, or answers 422 and returns false.
	needGuild := func() (string, bool) {
		if len(rest) == 0 || !guildIDRE.MatchString(rest[0]) {
			respondError(c, newErr(http.StatusUnprocessableEntity, name+" needs a server (guild) ID."))
			return "", false
		}
		return rest[0], true
	}

	switch name {
	case "pause", "resume", "block", "unblock", "purge-guild":
		gid := ""
		if len(rest) > 0 && guildIDRE.MatchString(rest[0]) {
			gid = rest[0]
		}
		dbAudit(ctx, "", -1, gid, "admin", name, command, true)
	}

	switch name {
	case "help":
		reply("Admin commands: " + strings.Join(adminCommands, ", "))
	case "audit":
		gid, ok := needGuild()
		if !ok {
			return
		}
		count := 20
		if len(rest) > 1 && isDigits(rest[1]) {
			if n, err := strconv.Atoi(rest[1]); err == nil {
				count = n
			}
		}
		if !dbConfigured() {
			reply("History is not turned on (DATABASE_URL is not set).")
			return
		}
		reply(dbAuditRecent(ctx, gid, count))
	case "purge-guild":
		gid, ok := needGuild()
		if !ok {
			return
		}
		dbForgetGuild(ctx, gid)
		reply(fmt.Sprintf("Deleted stored history and audit records for %s.", gid))
	case "status":
		invalidateStatus() // always fresh for the operator
		doc := getStatus(ctx)
		paused, err := rdb.Get(ctx, killSwitchKey)
		if err != nil {
			respondError(c, err)
			return
		}
		blocked, err := rdb.SCard(ctx, "blocked:guilds")
		if err != nil {
			respondError(c, err)
			return
		}
		history := "off (DATABASE_URL not set)"
		if configured, reachable := dbHealthy(ctx); configured {
			history = map[bool]string{true: "ok", false: "unreachable"}[reachable]
		}
		out := adminStatusOut{Overall: doc.Label, Paused: paused != "", BlockedServers: blocked, HistoryDB: history}
		for _, comp := range doc.Components {
			out.Components.keys = append(out.Components.keys, comp.Name)
			out.Components.vals = append(out.Components.vals, comp.Status+" · "+comp.Detail)
		}
		reply(out)
	case "pause":
		if _, err := rdb.Set(ctx, killSwitchKey, "1", 0, false); err != nil {
			respondError(c, err)
			return
		}
		reply("Paused. New requests are refused and running requests stop at their next step.")
	case "resume":
		if err := rdb.Del(ctx, killSwitchKey); err != nil {
			respondError(c, err)
			return
		}
		reply("Resumed.")
	case "block":
		gid, ok := needGuild()
		if !ok {
			return
		}
		if err := rdb.SAdd(ctx, "blocked:guilds", gid); err != nil {
			respondError(c, err)
			return
		}
		reply("Blocked " + gid + ".")
	case "unblock":
		gid, ok := needGuild()
		if !ok {
			return
		}
		if err := rdb.SRem(ctx, "blocked:guilds", gid); err != nil {
			respondError(c, err)
			return
		}
		reply("Unblocked " + gid + ".")
	case "blocklist":
		members, err := rdb.SMembers(ctx, "blocked:guilds")
		if err != nil {
			respondError(c, err)
			return
		}
		list := append([]string{}, members...)
		sort.Strings(list)
		reply(list)
	default:
		respondError(c, newErr(http.StatusForbidden, "Unknown admin command."))
	}
}
