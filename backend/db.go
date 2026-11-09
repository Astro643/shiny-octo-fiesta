package main

// Durable records in Postgres (Neon). Redis stays in charge of live state; Postgres keeps what must outlive Redis's 24 hours:
//
//   requests    one row per Console request: prompt, final status, result or error
//   audit_log   what agents did to a server that cannot be undone (bans, kicks, deletes, ...) and what the operator did
//
// Everything here is best-effort: if the database is not configured, slow or down, every function quietly does nothing
// (or returns an empty list), so a database problem can never stop a request. Rows older than HISTORY_DAYS are deleted.
//
// Env:
//   DATABASE_URL   Neon connection string, e.g. postgresql://user:password@ep-xxxx.region.aws.neon.tech/neondb?sslmode=verify-full
//   HISTORY_DAYS   how long rows are kept (default 30)
//
// Neon notes: use sslmode=verify-full (Neon certificates are signed by a public CA, so no CA file is needed). Both the
// direct and the pooled (-pooler) connection strings work. The first query after the database has been idle can take
// a second or two while Neon wakes it up.

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	queryTimeout = 8 * time.Second  // a little longer than before: Neon may be waking up
	dbBackoff    = 30 * time.Second // after a failure, skip the database this long instead of slowing every request
	purgeEvery   = 6 * time.Hour
)

var dbSchema = []string{
	`CREATE TABLE IF NOT EXISTS requests (
		command_id VARCHAR(36) NOT NULL PRIMARY KEY,
		guild_id VARCHAR(25) NOT NULL,
		agent SMALLINT NULL,
		prompt VARCHAR(2000) NOT NULL,
		status VARCHAR(24) NOT NULL,
		result TEXT NULL,
		error TEXT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		finished_at TIMESTAMPTZ NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_requests_guild ON requests (guild_id, created_at)`,
	`CREATE INDEX IF NOT EXISTS idx_requests_created ON requests (created_at)`,
	`CREATE TABLE IF NOT EXISTS audit_log (
		id BIGSERIAL PRIMARY KEY,
		command_id VARCHAR(36) NULL,
		turn SMALLINT NULL,
		guild_id VARCHAR(25) NULL,
		actor VARCHAR(16) NOT NULL,
		action VARCHAR(64) NOT NULL,
		detail VARCHAR(500) NOT NULL,
		ok SMALLINT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		CONSTRAINT uq_audit_command_turn UNIQUE (command_id, turn)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_guild ON audit_log (guild_id, created_at)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log (created_at)`,
}

var (
	dbPool       *sql.DB
	dbDownUntil  atomic.Int64 // unix nanoseconds
	dbLastErrLog atomic.Int64
)

func historyDays() int {
	n, err := strconv.Atoi(envStr("HISTORY_DAYS"))
	if err != nil || n < 1 {
		if envStr("HISTORY_DAYS") != "" && err == nil {
			return 1
		}
		return 30
	}
	return n
}

func dbConfigured() bool { return envStr("DATABASE_URL") != "" }

func dbNow() time.Time { return time.Now().UTC() }

// pgText makes a string safe for Postgres, which refuses NUL bytes and invalid UTF-8 (MySQL tolerated both).
func pgText(s string, limit int) string {
	s = strings.ToValidUTF8(strings.ReplaceAll(s, "\x00", ""), "\uFFFD")
	return clipRunes(s, limit)
}

func fmtTime(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05") }

// pgConfig turns the Neon connection string into connection settings.
func pgConfig() (*pgx.ConnConfig, error) {
	cfg, err := pgx.ParseConfig(envStr("DATABASE_URL"))
	if err != nil {
		return nil, errors.New("DATABASE_URL must look like postgresql://user:password@host/database?sslmode=verify-full")
	}
	cfg.ConnectTimeout = 10 * time.Second
	// The simple protocol sends no prepared statements, so it also works behind Neon's pooled (PgBouncer) endpoint.
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	if cfg.TLSConfig == nil {
		log.Printf("DATABASE_URL has no TLS (sslmode=disable): Neon requires TLS, add ?sslmode=verify-full")
	} else if cfg.TLSConfig.InsecureSkipVerify {
		log.Printf("DATABASE_URL uses sslmode=require: the connection is encrypted but the server is not verified. Use sslmode=verify-full")
	}
	return cfg, nil
}

// dbInit opens the pool and creates the tables. Returns false (and logs why) if the database cannot be used.
func dbInit(ctx context.Context) bool {
	if !dbConfigured() {
		return false
	}
	cfg, err := pgConfig()
	if err != nil {
		log.Printf("The database is configured but could not be used (%v). Requests still work; history is off.", err)
		return false
	}
	pool := stdlib.OpenDB(*cfg)
	pool.SetMaxOpenConns(4)
	pool.SetMaxIdleConns(2)
	pool.SetConnMaxLifetime(5 * time.Minute)
	pool.SetConnMaxIdleTime(2 * time.Minute) // Neon closes idle connections; do not hold on to dead ones
	ictx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for _, stmt := range dbSchema {
		if _, err := pool.ExecContext(ictx, stmt); err != nil {
			_ = pool.Close()
			log.Printf("The database is configured but could not be used (%T). Requests still work; history is off.", err)
			return false
		}
	}
	dbPool = pool
	log.Printf("History store (Postgres) ready")
	return true
}

func dbClose() {
	if dbPool != nil {
		_ = dbPool.Close()
		dbPool = nil
	}
}

func dbBackingOff() bool { return time.Now().UnixNano() < dbDownUntil.Load() }

func dbFailed(err error) {
	dbDownUntil.Store(time.Now().Add(dbBackoff).UnixNano())
	now := time.Now().UnixNano()
	if now-dbLastErrLog.Load() > int64(60*time.Second) {
		dbLastErrLog.Store(now)
		log.Printf("Database query failed (%T); skipping the database for %ds", err, int(dbBackoff.Seconds()))
	}
}

// dbExec runs one statement. false means it did not run (not configured, backing off, or failed).
func dbExec(ctx context.Context, query string, args ...any) bool {
	pool := dbPool
	if pool == nil || dbBackingOff() {
		return false
	}
	// Detached from the caller's cancellation: a client disconnecting must not count as a database failure.
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), queryTimeout)
	defer cancel()
	if _, err := pool.ExecContext(cctx, query, args...); err != nil {
		dbFailed(err)
		return false
	}
	return true
}

// dbQuery runs a SELECT and calls scan for each row. false means it did not run or failed.
func dbQuery(ctx context.Context, query string, args []any, scan func(*sql.Rows) error) bool {
	pool := dbPool
	if pool == nil || dbBackingOff() {
		return false
	}
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), queryTimeout)
	defer cancel()
	rows, err := pool.QueryContext(cctx, query, args...)
	if err != nil {
		dbFailed(err)
		return false
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			dbFailed(err)
			return false
		}
	}
	if err := rows.Err(); err != nil {
		dbFailed(err)
		return false
	}
	return true
}

// ---- requests -------------------------------------------------------------------------------------

func dbRecordRequest(ctx context.Context, commandID, guildID string, agent int, prompt string) {
	dbExec(ctx, "INSERT INTO requests (command_id, guild_id, agent, prompt, status, created_at) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING",
		commandID, guildID, agent, pgText(prompt, 2000), "processing", dbNow())
}

func dbFinishRequest(ctx context.Context, commandID, status, result, errText string) {
	var res, er any
	if result != "" {
		res = pgText(result, 1500)
	}
	if errText != "" {
		er = pgText(errText, 2000)
	}
	dbExec(ctx, "UPDATE requests SET status=$1, result=$2, error=$3, finished_at=$4 WHERE command_id=$5",
		status, res, er, dbNow(), commandID)
}

func dbSetStatus(ctx context.Context, commandID, status string) {
	dbExec(ctx, "UPDATE requests SET status=$1, finished_at=$2 WHERE command_id=$3", status, dbNow(), commandID)
}

func dbHistory(ctx context.Context, guildID string, limit int) []map[string]any {
	if limit < 1 {
		limit = 1
	}
	if limit > 50 {
		limit = 50
	}
	out := []map[string]any{}
	dbQuery(ctx, "SELECT command_id, status, prompt, created_at FROM requests WHERE guild_id=$1 ORDER BY created_at DESC LIMIT $2",
		[]any{guildID, limit}, func(r *sql.Rows) error {
			var id, status, prompt string
			var at time.Time
			if err := r.Scan(&id, &status, &prompt, &at); err != nil {
				return err
			}
			out = append(out, map[string]any{"id": id, "status": status, "prompt": prompt, "at": fmtTime(at)})
			return nil
		})
	return out
}

// ---- audit log ------------------------------------------------------------------------------------

// dbAudit is idempotent for (command_id, turn): a replayed workflow step cannot write the same action twice.
// Pass "" for commandID / guildID and a negative turn to store NULL.
func dbAudit(ctx context.Context, commandID string, turn int, guildID, actor, action, detail string, ok bool) {
	var cmd, gid, trn any
	if commandID != "" {
		cmd = commandID
	}
	if guildID != "" {
		gid = guildID
	}
	if turn >= 0 {
		trn = turn
	}
	okInt := 0
	if ok {
		okInt = 1
	}
	dbExec(ctx, "INSERT INTO audit_log (command_id, turn, guild_id, actor, action, detail, ok, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING",
		cmd, trn, gid, pgText(actor, 16), pgText(action, 64), pgText(detail, 500), okInt, dbNow())
}

func dbAuditRecent(ctx context.Context, guildID string, limit int) []map[string]any {
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	out := []map[string]any{}
	dbQuery(ctx, "SELECT created_at, actor, action, detail, ok, command_id FROM audit_log WHERE guild_id=$1 ORDER BY id DESC LIMIT $2",
		[]any{guildID, limit}, func(r *sql.Rows) error {
			var at time.Time
			var actor, action, detail string
			var ok int
			var cmd sql.NullString
			if err := r.Scan(&at, &actor, &action, &detail, &ok, &cmd); err != nil {
				return err
			}
			var cmdVal any
			if cmd.Valid {
				cmdVal = cmd.String
			}
			out = append(out, map[string]any{"at": fmtTime(at), "actor": actor, "action": action, "detail": detail,
				"ok": ok != 0, "command_id": cmdVal})
			return nil
		})
	return out
}

// ---- housekeeping ---------------------------------------------------------------------------------

// dbForgetGuild deletes everything held about one server (for a deletion request).
func dbForgetGuild(ctx context.Context, guildID string) {
	dbExec(ctx, "DELETE FROM requests WHERE guild_id=$1", guildID)
	dbExec(ctx, "DELETE FROM audit_log WHERE guild_id=$1", guildID)
}

func dbPurgeOld(ctx context.Context) {
	cutoff := dbNow().Add(-time.Duration(historyDays()) * 24 * time.Hour)
	dbExec(ctx, "DELETE FROM requests WHERE created_at < $1", cutoff)
	dbExec(ctx, "DELETE FROM audit_log WHERE created_at < $1", cutoff)
	humanPurgeOld(ctx) // support chat: conversations idle for humanRetentionDays
}

func dbPurgeLoop(ctx context.Context) {
	select {
	case <-time.After(30 * time.Second):
	case <-ctx.Done():
		return
	}
	for {
		dbPurgeOld(ctx)
		select {
		case <-time.After(purgeEvery):
		case <-ctx.Done():
			return
		}
	}
}

// dbHealthy reports (configured, reachable now) for the operator's status view.
func dbHealthy(ctx context.Context) (bool, bool) {
	if !dbConfigured() {
		return false, false
	}
	dbDownUntil.Store(0)
	ok := dbQuery(ctx, "SELECT 1", nil, func(r *sql.Rows) error { return nil })
	return true, ok
}

