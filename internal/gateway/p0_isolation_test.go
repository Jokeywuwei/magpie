package gateway

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/modelaccess"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/usage"
)

// p0Password is a random password for a throwaway account: no usable
// credential is written in this file.
func p0Password(t *testing.T) string {
	t.Helper()
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// p0Fixture builds two platform users, each with a key granted relay/m,
// one upstream that answers, and one usage row owned by B.
func p0Fixture(t *testing.T) (h http.Handler, a, b identity.User, aKey access.UserKey, aSecret, bSecret string, upstream *atomic.Int32) {
	t.Helper()
	fresh(t)
	t.Setenv("MAGPIE_ADDR", "")
	t.Setenv("MAGPIE_PUBLIC_URL", "")
	s := settings.Load()
	s.LAN = true
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}
	upstream = &atomic.Int32{}
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`)
	}))
	t.Cleanup(up.Close)
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "upstream", Chat: up.URL + "/v1", Models: []string{"m"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := modelaccess.Set("relay/m", true, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Add("admin@p0.test", "Admin", p0Password(t), identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	var err error
	if a, err = identity.Register("a@p0.test", "A", p0Password(t)); err != nil {
		t.Fatal(err)
	}
	if b, err = identity.Register("b@p0.test", "B", p0Password(t)); err != nil {
		t.Fatal(err)
	}
	if aKey, aSecret, err = access.CreateUserKey(a.ID, "A key", []string{"relay/m"}); err != nil {
		t.Fatal(err)
	}
	bKey, bSecret, err := access.CreateUserKey(b.ID, "B key", []string{"relay/m"})
	if err != nil {
		t.Fatal(err)
	}
	// a row that belongs to B, as a successful call of B's would leave it
	usage.Append(usage.Record{Time: time.Now(), Agent: "b-agent", Provider: "relay",
		Model: "relay/m", CallerKeyID: bKey.ID, CallerKeyName: "B key", CallerKeyOwner: b.ID,
		Input: 7, Output: 3})
	return lanGuard(New().Handler()), a, b, aKey, aSecret, bSecret, upstream
}

// clearUserGrant writes the state a legacy key (or an administrator's
// empty models-key) can reach: an explicit, empty grant. It writes the
// store directly so the regression holds whatever the creation path does.
func clearUserGrant(t *testing.T, id string) {
	t.Helper()
	raw, err := os.ReadFile(access.Path())
	if err != nil {
		t.Fatal(err)
	}
	var keys []access.Key
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range keys {
		if keys[i].ID == id {
			keys[i].Models = nil
			keys[i].ModelsSet = true
			found = true
		}
	}
	if !found {
		t.Fatalf("no key %s", id)
	}
	out, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(access.Path(), append(out, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(access.Path(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func p0Do(h http.Handler, method, path, secret, body string) (int, string) {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.RemoteAddr = "192.168.1.9:5000"
	req.Host = "127.0.0.1:3425"
	req.Header.Set("Authorization", "Bearer "+secret)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

// A platform user's key with an empty grant is held like a key with a
// grant: it is told nothing of the platform's other tenants, and it can
// trigger nothing. Before the fix, heldKey forgot Owner, so this key read
// every tenant's usage, quotas and routes.
func TestEmptyGrantUserKeyIsHeld(t *testing.T) {
	h, _, b, aKey, aSecret, _, upstream := p0Fixture(t)
	clearUserGrant(t, aKey.ID)

	if code, body := p0Do(h, "GET", "/v1/magpie/usage", aSecret, ""); code != http.StatusForbidden {
		t.Errorf("platform usage: %d %s", code, body)
	}
	if code, body := p0Do(h, "GET", "/v1/magpie/quotas", aSecret, ""); code != http.StatusForbidden {
		t.Errorf("quotas: %d %s", code, body)
	}
	if code, body := p0Do(h, "GET", "/v1/magpie/quotas/history", aSecret, ""); code != http.StatusForbidden {
		t.Errorf("quota history: %d %s", code, body)
	}
	if code, body := p0Do(h, "GET", "/v1/magpie/trace", aSecret, ""); code != http.StatusForbidden {
		t.Errorf("trace: %d %s", code, body)
	}
	if code, body := p0Do(h, "GET", "/v1/magpie/usage/requests", aSecret, ""); code != http.StatusOK || strings.Contains(body, "B key") || strings.Contains(body, b.ID) {
		t.Errorf("request log: %d %s", code, body)
	}
	for _, x := range []struct{ method, path, body string }{
		{"POST", "/v1/magpie/quotas/codex-reset?user=x@example.com", ""},
		{"POST", "/v1/magpie/update", ""},
		{"POST", "/v1/magpie/usage/requests", `{"calls":[]}`},
		{"GET", "/v1/magpie/library", ""},
		{"GET", "/v1/magpie/agent-models", ""},
	} {
		if code, body := p0Do(h, x.method, x.path, aSecret, x.body); code != http.StatusForbidden {
			t.Errorf("%s %s: %d %s", x.method, x.path, code, body)
		}
	}
	// an empty grant still reaches no model
	if code, body := modelCall(h, aSecret, "relay/m"); code != http.StatusForbidden {
		t.Errorf("empty grant called a model: %d %s", code, body)
	}
	if n := upstream.Load(); n != 0 {
		t.Errorf("an empty grant reached the provider %d times", n)
	}
}

// A platform user's key with a grant is held too: the fix must not only
// rescue the empty case.
func TestGrantedUserKeyCannotReadPlatformData(t *testing.T) {
	h, _, _, _, aSecret, _, _ := p0Fixture(t)
	if code, body := p0Do(h, "GET", "/v1/magpie/usage", aSecret, ""); code != http.StatusForbidden {
		t.Errorf("platform usage: %d %s", code, body)
	}
	if code, body := p0Do(h, "GET", "/v1/magpie/quotas", aSecret, ""); code != http.StatusForbidden {
		t.Errorf("quotas: %d %s", code, body)
	}
}

// An ownerless gateway key keeps the administrator's cross-tenant access.
func TestAdminGatewayKeyKeepsPlatformAccess(t *testing.T) {
	h, _, _, _, _, _, _ := p0Fixture(t)
	adminSecret, err := access.Update("add-key", access.Change{Name: "Admin gateway"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v1/magpie/usage", "/v1/magpie/quotas"} {
		if code, body := p0Do(h, "GET", path, adminSecret, ""); code != http.StatusOK {
			t.Errorf("%s: %d %s", path, code, body)
		}
	}
}

// A legacy user key with no explicit grant, when the platform has no open
// model, is filled with an empty grant. It must still be held: the old
// heldKey read it as an unrestricted key.
func TestLegacyEmptyUserKeyIsHeld(t *testing.T) {
	h, _, b, aKey, aSecret, _, _ := p0Fixture(t)
	if _, err := modelaccess.Set("relay/m", false, "", ""); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(access.Path())
	if err != nil {
		t.Fatal(err)
	}
	var keys []access.Key
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for i := range keys {
		if keys[i].ID == aKey.ID {
			keys[i].Models = nil
			keys[i].ModelsSet = false
		}
	}
	out, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(access.Path(), append(out, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := access.Check(aSecret); err != nil {
		t.Fatal(err)
	}
	if code, body := p0Do(h, "GET", "/v1/magpie/usage", aSecret, ""); code != http.StatusForbidden {
		t.Errorf("a legacy empty grant read platform usage: %d %s", code, body)
	}
	if code, body := p0Do(h, "GET", "/v1/magpie/usage/requests", aSecret, ""); code != http.StatusOK || strings.Contains(body, "B key") || strings.Contains(body, b.ID) {
		t.Errorf("a legacy empty grant read another tenant: %d %s", code, body)
	}
}

// A user's key has no MCP/plugin tools by default: the empty list means
// "none" for a platform account, never "every server and tool".
func TestUserKeyHasNoMCPTools(t *testing.T) {
	h, _, _, _, aSecret, _, _ := p0Fixture(t)
	for _, x := range []struct{ path, name, why string }{
		{ToolsMCPPath, "magpie-tools", "may not use the MCP server"},
		{SearchMCPPath, "magpie-web-search", "may not use the MCP server"},
		{VisionMCPPath, "magpie-vision", "may not use the MCP server"},
		{ImageMCPPath, "magpie-image", "may not use the MCP server"},
		{BrowserMCPPath, "magpie-browser", "may not use the MCP server"},
		// these two servers turn a remote caller away before the key's own
		// tool list is read; the refusal is still a 403, and the point is
		// that a user key reaches neither
		{MemoryMCPPath, "magpie-memory", "answers the agents on this computer"},
		{SessionsMCPPath, "magpie-sessions", "answers the agents on this computer alone"},
	} {
		w := mcpPostTo(h, x.path, "192.168.1.9:5000", toolsList, "Authorization", "Bearer "+aSecret)
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), x.why) {
			t.Errorf("%s: %d %s", x.name, w.Code, w.Body)
		}
	}
	// an ownerless key still sees its tools
	adminSecret, err := access.Update("add-key", access.Change{Name: "Admin gateway"})
	if err != nil {
		t.Fatal(err)
	}
	w := mcpPostTo(h, ToolsMCPPath, "192.168.1.9:5000", toolsList, "Authorization", "Bearer "+adminSecret)
	if w.Code != http.StatusOK {
		t.Errorf("an ownerless key lost its tools: %d %s", w.Code, w.Body)
	}
}
