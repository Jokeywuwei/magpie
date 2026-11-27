// Sessions: a browser signs in with an account once, and carries an opaque
// token after that. Only the SHA-256 of a token is kept, so the file holds
// nothing a browser's cookie can be rebuilt from.
package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/yetone/magpie/internal/edit"
)

const (
	// SessionTTL is how long a sign-in lasts without being used. A page
	// that signs in again gets a new one; the old is replaced only when
	// the account has more than MaxSessions.
	SessionTTL = 7 * 24 * time.Hour
	// MaxSessions is how many browsers one account may keep signed in.
	// Signing in on one more ends the oldest.
	MaxSessions = 10
	// sessionTokenLen is the token's bytes: 43 base64url characters.
	sessionTokenLen = 32
)

// session is one signed-in browser, by the hash of its token.
type session struct {
	UserID    string    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func loadSessions() (map[string]session, error) {
	b, err := os.ReadFile(SessionsPath())
	if errors.Is(err, os.ErrNotExist) {
		return map[string]session{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]session{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]session{}
	}
	return out, nil
}

func saveSessions(sessions map[string]session) error {
	b, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(SessionsPath()), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(SessionsPath(), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(SessionsPath(), 0o600); err != nil {
		return err
	}
	return edit.WriteAtomic(SessionsPath(), append(b, '\n'))
}

// pruneSessions drops every expired session, and every session of a user
// no longer here; it returns whether it changed anything.
func pruneSessions(sessions map[string]session, users []User, now time.Time) bool {
	changed := false
	for h, s := range sessions {
		if !now.Before(s.ExpiresAt) || indexID(users, s.UserID) < 0 {
			delete(sessions, h)
			changed = true
		}
	}
	return changed
}

// NewSession signs an account in, and gives the token to put in its cookie
// and when that token stops working. An account may keep MaxSessions
// browsers; the oldest ends when one more signs in.
func NewSession(userID string) (token string, expires time.Time, err error) {
	mu.Lock()
	defer mu.Unlock()
	users, err := load()
	if err != nil {
		return "", time.Time{}, err
	}
	i := indexID(users, userID)
	if i < 0 {
		return "", time.Time{}, ErrNotFound
	}
	if users[i].Disabled {
		return "", time.Time{}, ErrDisabled
	}
	sessions, err := loadSessions()
	if err != nil {
		return "", time.Time{}, err
	}
	now := clock().UTC()
	pruneSessions(sessions, users, now)
	if mine := sessionsOf(sessions, userID); len(mine) >= MaxSessions {
		oldest := mine[0]
		for _, h := range mine[1:] {
			if sessions[h].CreatedAt.Before(sessions[oldest].CreatedAt) {
				oldest = h
			}
		}
		delete(sessions, oldest)
	}
	raw := make([]byte, sessionTokenLen)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expires = now.Add(SessionTTL)
	sessions[tokenHash(token)] = session{UserID: userID, CreatedAt: now, ExpiresAt: expires}
	if err := saveSessions(sessions); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// sessionsOf is the hashes of userID's sessions, by creation, oldest first.
func sessionsOf(sessions map[string]session, userID string) []string {
	out := make([]string, 0, len(sessions))
	for h, s := range sessions {
		if s.UserID == userID {
			out = append(out, h)
		}
	}
	slices.SortFunc(out, func(a, b string) int { return sessions[a].CreatedAt.Compare(sessions[b].CreatedAt) })
	return out
}

// SessionUser is the account a token signs in, if it still does. An expired
// token, a token of an account since disabled or removed, and one this
// magpie never made are all ErrNoSession.
func SessionUser(token string) (User, error) {
	if token == "" {
		return User{}, ErrNoSession
	}
	mu.Lock()
	defer mu.Unlock()
	sessions, err := loadSessions()
	if err != nil {
		return User{}, err
	}
	h := tokenHash(token)
	s, ok := sessions[h]
	if !ok {
		return User{}, ErrNoSession
	}
	now := clock().UTC()
	if !now.Before(s.ExpiresAt) {
		delete(sessions, h)
		_ = saveSessions(sessions)
		return User{}, ErrNoSession
	}
	users, err := load()
	if err != nil {
		return User{}, err
	}
	i := indexID(users, s.UserID)
	if i < 0 || users[i].Disabled {
		delete(sessions, h)
		_ = saveSessions(sessions)
		return User{}, ErrNoSession
	}
	return users[i], nil
}

// EndSession signs one browser out. A token this magpie didn't make is no
// error: signing out a browser already signed out is not a failure.
func EndSession(token string) error {
	if token == "" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	sessions, err := loadSessions()
	if err != nil {
		return err
	}
	h := tokenHash(token)
	if _, ok := sessions[h]; !ok {
		return nil
	}
	delete(sessions, h)
	return saveSessions(sessions)
}

// endUserSessionsLocked ends every session of userID; mu is held.
func endUserSessionsLocked(userID string) error {
	sessions, err := loadSessions()
	if err != nil {
		return err
	}
	changed := false
	for h, s := range sessions {
		if s.UserID == userID {
			delete(sessions, h)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return saveSessions(sessions)
}

// EndUserSessions ends every session of an account, as a password reset or
// a removal does.
func EndUserSessions(userID string) error {
	mu.Lock()
	defer mu.Unlock()
	return endUserSessionsLocked(userID)
}

// PruneSessions drops expired sessions and returns how many it dropped.
func PruneSessions() (int, error) {
	mu.Lock()
	defer mu.Unlock()
	sessions, err := loadSessions()
	if err != nil {
		return 0, err
	}
	users, err := load()
	if err != nil {
		return 0, err
	}
	before := len(sessions)
	pruneSessions(sessions, users, clock().UTC())
	if len(sessions) == before {
		return 0, nil
	}
	return before - len(sessions), saveSessions(sessions)
}
