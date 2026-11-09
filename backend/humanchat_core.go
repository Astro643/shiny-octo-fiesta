package main

// Human support chat, core pieces that need nothing but the standard library (so they are easy to test):
// the random name + avatar pool, password hashing for the operator login, and signed cookie tokens.
//
// Needs Go 1.24 or newer (crypto/pbkdf2 is in the standard library from 1.24).

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ---- random name + avatar -------------------------------------------------------------------------

// Visitors get a name from this fixed list, chosen at random. While a name is in use by an open conversation it is not
// given to anyone else; if every name is in use, a number is added ("Vex 2").
var aliasNames = []string{
	"Vex", "Lux", "Nyx", "Kai", "Rex", "Zay", "Nox", "Phi", "Aro", "Zen",
	"Flux", "Echo", "Luna", "Nova", "Opal", "Zuno", "Miu", "Blix", "Dori", "Soju",
	"Velto", "Mochi", "Vexon", "Zayro", "Qelro",
}

// The avatar is picked separately, at random, as "<animal>-<color>" (for example "fox-amber"). The pages draw the
// picture from that key, so there are no image files to host. 6 animals x 8 colors = 48 avatars.
var avatarAnimals = []string{"fox", "owl", "bear", "cat", "frog", "bunny"}
var avatarColors = []string{"amber", "coral", "mint", "sky", "violet", "rose", "lime", "slate"}

type Alias struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

func randInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

// randomAlias picks a random name that is not in taken (the names used by open conversations) and a random avatar.
func randomAlias(taken map[string]bool) Alias {
	avatar := avatarAnimals[randInt(len(avatarAnimals))] + "-" + avatarColors[randInt(len(avatarColors))]
	var free []string
	for _, n := range aliasNames {
		if !taken[n] {
			free = append(free, n)
		}
	}
	if len(free) > 0 {
		return Alias{Name: free[randInt(len(free))], Avatar: avatar}
	}
	base := aliasNames[randInt(len(aliasNames))]
	for n := 2; ; n++ {
		name := fmt.Sprintf("%s %d", base, n)
		if !taken[name] {
			return Alias{Name: name, Avatar: avatar}
		}
	}
}

// ---- message text ---------------------------------------------------------------------------------

// cleanBody normalises a chat message: no NUL bytes, valid UTF-8, \n line breaks, at most two blank lines in a row,
// trimmed. ok is false when it is empty or longer than max characters.
func cleanBody(s string, max int) (string, bool) {
	s = strings.ToValidUTF8(strings.ReplaceAll(s, "\x00", ""), "�")
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	for strings.Contains(s, "\n\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n\n", "\n\n\n")
	}
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	if n < 1 || n > max {
		return "", false
	}
	return s, true
}

// ---- password hashing (operator login) ------------------------------------------------------------

// Stored form: pbkdf2-sha256$<iterations>$<salt base64>$<key base64>
const (
	pwPrefix     = "pbkdf2-sha256"
	pwIterations = 600_000 // OWASP's current guidance for PBKDF2-HMAC-SHA256
	pwMinIter    = 100_000
	pwMaxIter    = 5_000_000 // stops a doctored value from turning a login into a CPU bomb
	pwKeyLen     = 32
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, pwIterations, pwKeyLen)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s$%d$%s$%s", pwPrefix, pwIterations,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(strings.TrimSpace(encoded), "$")
	if len(parts) != 4 || parts[0] != pwPrefix {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < pwMinIter || iter > pwMaxIter {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) < 8 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(want) < 16 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyPasswordHash is checked when the username is wrong, so a wrong username and a wrong password take the same time.
var dummyPasswordHash = func() string {
	h, err := hashPassword("not-a-real-password")
	if err != nil {
		panic(err)
	}
	return h
}()

// ---- signed tokens (cookies) ----------------------------------------------------------------------

// signToken returns payload + "." + HMAC. The purpose is mixed into the signature, so a token made for one use
// (a visitor cookie) can never be accepted for another (the operator cookie), even with the same key.
func signToken(key []byte, purpose, payload string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(purpose + "|" + payload))
	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

func verifyToken(key []byte, purpose, token string) (string, bool) {
	i := strings.LastIndex(token, ".")
	if i <= 0 {
		return "", false
	}
	payload, sig := token[:i], token[i+1:]
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(purpose + "|" + payload))
	if !hmac.Equal([]byte(sig), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return "", false
	}
	return payload, true
}

// deriveKey makes a separate key for one purpose from the site's session key.
func deriveKey(base []byte, purpose string) []byte {
	mac := hmac.New(sha256.New, base)
	mac.Write([]byte("derive|" + purpose))
	return mac.Sum(nil)
}

func newVisitorID() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("no randomness available")
	}
	return hex.EncodeToString(b), nil
}

// visitorHash is what the database stores instead of the visitor's secret cookie id.
func visitorHashOf(id string) string {
	sum := sha256.Sum256([]byte("human-visitor|" + id))
	return hex.EncodeToString(sum[:])
}
