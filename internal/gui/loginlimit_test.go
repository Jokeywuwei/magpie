package gui

import (
	"strconv"
	"testing"
	"time"
)

func TestLoginLimiterCountsAnAddressAndAPeer(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	l := newLoginLimiter(func() time.Time { return now })
	for i := 0; i < loginEmailLimit; i++ {
		if d := l.retry("a@example.com", "192.0.2.1"); d != 0 {
			t.Fatalf("try %d waited %v", i, d)
		}
		l.fail("a@example.com", "192.0.2.1")
	}
	if d := l.retry("a@example.com", "192.0.2.1"); d == 0 {
		t.Fatal("the pair's limit doesn't hold")
	}
	// another computer may still try the same account: a limit per account
	// alone would let anyone lock a known administrator out
	if d := l.retry("a@example.com", "192.0.2.2"); d != 0 {
		t.Fatalf("another peer was held by this one's failures: %v", d)
	}
	// a success clears this peer's count, but not the peer's other failures
	l.success("a@example.com", "192.0.2.1")
	if d := l.retry("a@example.com", "192.0.2.1"); d != 0 {
		t.Fatalf("the pair is still held after signing in: %v", d)
	}
	// and the window ends it
	for i := 0; i < loginIPLimit; i++ {
		l.fail("u"+strconv.Itoa(i)+"@example.com", "192.0.2.1")
	}
	if d := l.retry("new@example.com", "192.0.2.1"); d == 0 {
		t.Fatal("a peer with too many failures is not held")
	}
	now = now.Add(loginWindow)
	if d := l.retry("new@example.com", "192.0.2.1"); d != 0 {
		t.Fatalf("the window didn't end the peer's limit: %v", d)
	}
}
