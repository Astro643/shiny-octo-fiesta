package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Redis talks to Upstash over its REST API, exactly like the Python upstash-redis client did.
// That keeps REDIS_URL / REDIS_TOKEN unchanged (no TCP connection string needed).
type Redis struct {
	url   string
	token string
	hc    *http.Client
}

func NewRedis(url, token string) *Redis {
	return &Redis{url: strings.TrimRight(url, "/"), token: token, hc: &http.Client{Timeout: 15 * time.Second}}
}

// do sends one command, e.g. do(ctx, "SET", "k", "v", "EX", 60), and returns Upstash's "result".
func (r *Redis) do(ctx context.Context, args ...any) (any, error) {
	strs := make([]string, len(args))
	for i, a := range args {
		strs[i] = fmt.Sprint(a)
	}
	body, err := json.Marshal(strs)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("redis request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var out struct {
		Result any    `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("redis: unreadable response (HTTP %d)", resp.StatusCode)
	}
	if out.Error != "" {
		return nil, fmt.Errorf("redis: %s", out.Error)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("redis: HTTP %d", resp.StatusCode)
	}
	return out.Result, nil
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

// Get returns "" when the key does not exist.
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	v, err := r.do(ctx, "GET", key)
	if err != nil {
		return "", err
	}
	return asStr(v), nil
}

// GetDel reads and deletes in one atomic command.
func (r *Redis) GetDel(ctx context.Context, key string) (string, error) {
	v, err := r.do(ctx, "GETDEL", key)
	if err != nil {
		return "", err
	}
	return asStr(v), nil
}

// Set writes a value. ex > 0 adds a TTL in seconds; nx only sets if the key is absent.
// It reports whether the value was written.
func (r *Redis) Set(ctx context.Context, key, val string, ex int, nx bool) (bool, error) {
	args := []any{"SET", key, val}
	if ex > 0 {
		args = append(args, "EX", ex)
	}
	if nx {
		args = append(args, "NX")
	}
	v, err := r.do(ctx, args...)
	if err != nil {
		return false, err
	}
	return v != nil, nil
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	args := []any{"DEL"}
	for _, k := range keys {
		args = append(args, k)
	}
	_, err := r.do(ctx, args...)
	return err
}

func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	v, err := r.do(ctx, "EXISTS", key)
	return toInt(v) > 0, err
}

func (r *Redis) Expire(ctx context.Context, key string, seconds int) error {
	_, err := r.do(ctx, "EXPIRE", key, seconds)
	return err
}

func (r *Redis) Incr(ctx context.Context, key string) (int, error) {
	v, err := r.do(ctx, "INCR", key)
	return toInt(v), err
}

func (r *Redis) Decr(ctx context.Context, key string) (int, error) {
	v, err := r.do(ctx, "DECR", key)
	return toInt(v), err
}

func (r *Redis) Ping(ctx context.Context) error {
	_, err := r.do(ctx, "PING")
	return err
}

func (r *Redis) ZCard(ctx context.Context, key string) (int, error) {
	v, err := r.do(ctx, "ZCARD", key)
	return toInt(v), err
}

func (r *Redis) ZRange(ctx context.Context, key string) ([]string, error) {
	v, err := r.do(ctx, "ZRANGE", key, 0, -1)
	if err != nil {
		return nil, err
	}
	return asStrings(v), nil
}

func (r *Redis) ZAdd(ctx context.Context, key string, score float64, member string) error {
	_, err := r.do(ctx, "ZADD", key, strconv.FormatFloat(score, 'f', -1, 64), member)
	return err
}

func (r *Redis) ZRem(ctx context.Context, key, member string) error {
	_, err := r.do(ctx, "ZREM", key, member)
	return err
}

func (r *Redis) SIsMember(ctx context.Context, key, member string) (bool, error) {
	v, err := r.do(ctx, "SISMEMBER", key, member)
	return toInt(v) > 0, err
}

func (r *Redis) SAdd(ctx context.Context, key, member string) error {
	_, err := r.do(ctx, "SADD", key, member)
	return err
}

func (r *Redis) SRem(ctx context.Context, key, member string) error {
	_, err := r.do(ctx, "SREM", key, member)
	return err
}

func (r *Redis) SCard(ctx context.Context, key string) (int, error) {
	v, err := r.do(ctx, "SCARD", key)
	return toInt(v), err
}

func (r *Redis) SMembers(ctx context.Context, key string) ([]string, error) {
	v, err := r.do(ctx, "SMEMBERS", key)
	if err != nil {
		return nil, err
	}
	return asStrings(v), nil
}

func asStrings(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, asStr(item))
	}
	return out
}
