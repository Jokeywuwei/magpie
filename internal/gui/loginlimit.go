// Sign-in attempts are limited per account-and-peer and per peer, in
// memory: five failures for one address from one peer in fifteen minutes,
// twenty from one peer over any addresses. A limit per account alone would
// let anyone lock a known administrator out by failing five times, so the
// pair is what is counted: a wrong password from one computer doesn't stop
// the same account from another. The limit survives no restart, which is
// enough for the brute force it stops: an attacker would have to restart
// magpie to clear it, and a real user waits a few minutes. A success clears
// the pair's count, so one typo doesn't follow a user around.
package gui

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	loginWindow     = 15 * time.Minute
	loginEmailLimit = 5
	loginIPLimit    = 20
	loginMapPruneAt = 512
)

type loginTries struct {
	start time.Time
	fails int
}

type loginLimiter struct {
	mu     sync.Mutex
	now    func() time.Time
	byPair map[string]loginTries
	byIP   map[string]loginTries
}

func newLoginLimiter(now func() time.Time) *loginLimiter {
	return &loginLimiter{now: now, byPair: map[string]loginTries{}, byIP: map[string]loginTries{}}
}

// loginAttempts is the process's limiter. Tests that need a clean one reset
// it (or make their own with a clock they move).
var loginAttempts = newLoginLimiter(time.Now)

// retry is how long the next sign-in for email from ip must wait, 0 when
// it may be tried now. A failure older than the window no longer counts.
func (l *loginLimiter) retry(email, ip string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.pruneLocked(now)
	var wait time.Duration
	if t, ok := l.byPair[loginPair(email, ip)]; ok && t.fails >= loginEmailLimit {
		wait = t.start.Add(loginWindow).Sub(now)
	}
	if t, ok := l.byIP[ip]; ok && t.fails >= loginIPLimit {
		if d := t.start.Add(loginWindow).Sub(now); d > wait {
			wait = d
		}
	}
	if wait < 0 {
		return 0
	}
	return wait
}

// fail counts one wrong password for email from ip. It is called after
// the password was checked, so a correct password never adds a failure.
func (l *loginLimiter) fail(email, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.bumpLocked(l.byPair, loginPair(email, ip), now)
	l.bumpLocked(l.byIP, ip, now)
}

// success clears this peer's failures for the address: a person who
// mistyped twice and then signed in isn't held to the mistakes.
func (l *loginLimiter) success(email, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.byPair, loginPair(email, ip))
}

func (l *loginLimiter) bumpLocked(m map[string]loginTries, key string, now time.Time) {
	t, ok := m[key]
	if !ok || now.Sub(t.start) >= loginWindow {
		t = loginTries{start: now}
	}
	t.fails++
	m[key] = t
}

func (l *loginLimiter) pruneLocked(now time.Time) {
	if len(l.byPair) < loginMapPruneAt && len(l.byIP) < loginMapPruneAt {
		return
	}
	for k, t := range l.byPair {
		if now.Sub(t.start) >= loginWindow {
			delete(l.byPair, k)
		}
	}
	for k, t := range l.byIP {
		if now.Sub(t.start) >= loginWindow {
			delete(l.byIP, k)
		}
	}
}

func (l *loginLimiter) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.byPair = map[string]loginTries{}
	l.byIP = map[string]loginTries{}
}

func loginPair(email, ip string) string { return email + "\x00" + ip }

// loginIP is the peer a sign-in came from, for the per-peer limit. It is
// the connection's own address, never a forwarding header a caller could
// write to escape the limit.
func loginIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
