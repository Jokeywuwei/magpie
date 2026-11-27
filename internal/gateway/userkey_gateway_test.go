package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/modelaccess"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/usage"
)

// A user's key signs in for the account it belongs to, carries that owner
// through the gateway, and stops when the key or the account does. The
// management API still needs an admin key.
func TestUserKeyThroughTheGateway(t *testing.T) {
	fresh(t)
	s := settings.Load()
	s.LAN = true // a keyed request from this computer is counted and named
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	a, err := identity.Register("a@example.com", "A", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	b, err := identity.Register("b@example.com", "B", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer upstream-secret" {
			t.Error("the caller's credential reached the provider")
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":1}}`)
	}))
	defer up.Close()
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "upstream-secret", Chat: up.URL + "/v1", Models: []string{"m"}}); err != nil {
		t.Fatal(err)
	}
	// the platform opens relay/m: a user key is only given open models,
	// and the gateway only serves it one that is still open
	if _, err := modelaccess.Set("relay/m", true, "", ""); err != nil {
		t.Fatal(err)
	}
	ka, secretA, err := access.CreateUserKey(a.ID, "A's laptop", []string{"relay/m"})
	if err != nil {
		t.Fatal(err)
	}
	_, secretB, err := access.CreateUserKey(b.ID, "B's laptop", []string{"relay/m"})
	if err != nil {
		t.Fatal(err)
	}
	h := New().Handler()
	call := func(secret string) int {
		t.Helper()
		r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"relay/m","messages":[{"role":"user","content":"hi"}],"stream":false}`))
		r.RemoteAddr = "127.0.0.1:50000"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+secret)
		// a claim of another owner, which nothing may read
		r.Header.Set("X-Magpie-Owner", b.ID)
		r.Header.Set("X-Magpie-User", b.ID)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if code := call(secretA); code != http.StatusOK {
		t.Fatalf("A's key: %d", code)
	}
	recs := usage.Load(time.Time{})
	if len(recs) != 1 || recs[0].CallerKeyOwner != a.ID || recs[0].CallerKeyID != ka.ID {
		t.Fatalf("the call's owner: %+v", recs)
	}
	if code := call(secretB); code != http.StatusOK {
		t.Fatalf("B's key: %d", code)
	}
	recs = usage.Load(time.Time{})
	if len(recs) != 2 || recs[1].CallerKeyOwner != b.ID {
		t.Fatalf("B's call: %+v", recs)
	}
	// the key off, on, then gone
	if _, err := access.SetUserKeyOff(a.ID, ka.ID, true); err != nil {
		t.Fatal(err)
	}
	if code := call(secretA); code != http.StatusUnauthorized {
		t.Fatalf("a turned-off key: %d", code)
	}
	if _, err := access.SetUserKeyOff(a.ID, ka.ID, false); err != nil {
		t.Fatal(err)
	}
	if code := call(secretA); code != http.StatusOK {
		t.Fatalf("a turned-on key: %d", code)
	}
	// the account disabled stops the key even while the key is on
	if _, err := identity.SetDisabled(a.ID, true); err != nil {
		t.Fatal(err)
	}
	if code := call(secretA); code != http.StatusUnauthorized {
		t.Fatalf("a disabled account's key: %d", code)
	}
	if _, err := identity.SetDisabled(a.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := access.DeleteUserKey(a.ID, ka.ID); err != nil {
		t.Fatal(err)
	}
	if code := call(secretA); code != http.StatusUnauthorized {
		t.Fatalf("a deleted key: %d", code)
	}
	if code := call(secretB); code != http.StatusOK {
		t.Fatalf("B's key after A's key went: %d", code)
	}
	// a removed account's key is refused too, even before its keys are swept
	if _, _, err := access.CreateUserKey(a.ID, "A's second", []string{"relay/m"}); err != nil {
		t.Fatal(err)
	}
	if err := identity.Remove(a.ID); err != nil {
		t.Fatal(err)
	}
	if code := call(secretA); code != http.StatusUnauthorized {
		t.Fatalf("a removed account's key: %d", code)
	}

	// the management API is still an admin key's alone
	var seen string
	SetAdminHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = r.URL.Path; w.WriteHeader(http.StatusNoContent) }))
	t.Cleanup(func() { adminHandler.Store(nil) })
	adminSecret, err := access.Update("add-key", access.Change{Name: "Platform admin"})
	if err != nil {
		t.Fatal(err)
	}
	keys, _ := access.List()
	var adminID string
	for _, k := range keys {
		if k.Name == "Platform admin" {
			adminID = k.ID
		}
	}
	if _, err := access.Update("admin-key", access.Change{Key: adminID, Admin: access.AdminAny}); err != nil {
		t.Fatal(err)
	}
	lan := lanGuard(New().Handler())
	adminCall := func(key string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", AdminPath+"providers", nil)
		r.RemoteAddr = "127.0.0.1:50000"
		r.Host = "127.0.0.1:3425"
		r.Header.Set("Authorization", "Bearer "+key)
		w := httptest.NewRecorder()
		lan.ServeHTTP(w, r)
		return w
	}
	if w := adminCall(adminSecret); w.Code != http.StatusNoContent || seen != "/providers" {
		t.Fatalf("admin key: %d %s %q", w.Code, w.Body, seen)
	}
	if w := adminCall(secretB); w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), AdminCode) {
		t.Fatalf("a user's key on the management API: %d %s", w.Code, w.Body)
	}
}
