package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/modelaccess"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// A user's key may name a routing group the platform has opened; a closed
// group is not its, and a group it wasn't granted isn't either. An
// ownerless key keeps its existing reach.
func TestAUserKeyNamesAnOpenRoutingGroup(t *testing.T) {
	fresh(t)
	if err := provider.Save(provider.Provider{ID: "relay", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"m"}}); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveGroup(provider.Group{ID: "mine", Name: "Mine", Members: []string{"relay/m"}}); err != nil {
		t.Fatal(err)
	}
	g := provider.Group{ID: "mine"}
	user := access.Identity{Owner: "a", Models: []string{"group/mine"}}
	if groupNamed(user, g) {
		t.Fatal("a user's key named a routing group the platform hasn't opened")
	}
	if _, err := modelaccess.Set("group/mine", true, "", ""); err != nil {
		t.Fatal(err)
	}
	if !groupNamed(user, g) {
		t.Fatal("a user's key lost an open routing group")
	}
	ungranted := access.Identity{Owner: "a", Models: []string{"relay/m"}}
	if groupNamed(ungranted, g) {
		t.Fatal("a user's key named a group it wasn't granted")
	}
	admin := access.Identity{Models: []string{"group/mine"}}
	if !groupNamed(admin, g) {
		t.Fatal("an ownerless key lost its routing group")
	}
}

func boolPtr(v bool) *bool { return &v }

func modelCall(h http.Handler, secret, model string) (int, string) {
	body := `{"model":"` + model + `","messages":[{"role":"user","content":"hi"}],"stream":false}`
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:50000"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+secret)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func modelList(h http.Handler, secret string) []string {
	r := httptest.NewRequest("GET", "/v1/models", nil)
	r.RemoteAddr = "127.0.0.1:50000"
	r.Header.Set("Authorization", "Bearer "+secret)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		return nil
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &out) != nil {
		return nil
	}
	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		if strings.HasPrefix(m.ID, "relay/") {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

// A user's key is held to two server-side rules at once: the model is in
// the platform's open catalog, and the key was granted it. Closing a model
// refuses every user key at once; an ownerless administrator's key keeps
// its existing reach.
func TestUserKeyModelAccessThroughTheGateway(t *testing.T) {
	fresh(t)
	s := settings.Load()
	s.LAN = true
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
	var upstreamCalls atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`)
	}))
	defer up.Close()
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "test-upstream-key",
		Chat: up.URL + "/v1", Models: []string{"m", "n", "closed"}}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"relay/m", "relay/n"} {
		if _, err := modelaccess.Set(id, true, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	// relay/closed is served but never opened
	if modelaccess.Allows("relay/closed") {
		t.Fatal("closed model is open")
	}
	ka, secretA, err := access.CreateUserKey(a.ID, "A's key", []string{"relay/m"})
	if err != nil {
		t.Fatal(err)
	}
	_, secretB, err := access.CreateUserKey(b.ID, "B's key", []string{"relay/n"})
	if err != nil {
		t.Fatal(err)
	}
	h := New().Handler()

	code, body := modelCall(h, secretA, "relay/m")
	if code != http.StatusOK {
		t.Fatalf("A's open and granted model: %d %s", code, body)
	}
	if upstreamCalls.Load() != 1 {
		t.Fatalf("upstream calls after one allowed request: %d", upstreamCalls.Load())
	}
	// open to the platform, but not this key's
	code, body = modelCall(h, secretA, "relay/n")
	if code != http.StatusForbidden || !strings.Contains(body, "error") {
		t.Fatalf("a model not granted to the key: %d %s", code, body)
	}
	// served, but not open to platform users at all
	code, body = modelCall(h, secretA, "relay/closed")
	if code != http.StatusForbidden || !strings.Contains(body, "error") {
		t.Fatalf("a closed model: %d %s", code, body)
	}
	if upstreamCalls.Load() != 1 {
		t.Fatalf("a refused model reached the provider: %d calls", upstreamCalls.Load())
	}

	// B's grant is its own
	if code, body := modelCall(h, secretB, "relay/n"); code != http.StatusOK {
		t.Fatalf("B's own model: %d %s", code, body)
	}
	if code, _ := modelCall(h, secretB, "relay/m"); code != http.StatusForbidden {
		t.Fatalf("B used A's model: %d", code)
	}

	// the model list is the intersection of open and granted
	if ids := modelList(h, secretA); len(ids) != 1 || ids[0] != "relay/m" {
		t.Fatalf("A's model list: %v", ids)
	}
	if ids := modelList(h, secretB); len(ids) != 1 || ids[0] != "relay/n" {
		t.Fatalf("B's model list: %v", ids)
	}

	// closing the model refuses the key that named it, before the provider
	if _, err := modelaccess.Update("relay/m", boolPtr(false), nil, nil); err != nil {
		t.Fatal(err)
	}
	if code, _ := modelCall(h, secretA, "relay/m"); code != http.StatusForbidden {
		t.Fatalf("a closed model after opening: %d", code)
	}
	if ids := modelList(h, secretA); len(ids) != 0 {
		t.Fatalf("A's list after its model closed: %v", ids)
	}
	if _, err := modelaccess.Update("relay/m", boolPtr(true), nil, nil); err != nil {
		t.Fatal(err)
	}
	if code, body := modelCall(h, secretA, "relay/m"); code != http.StatusOK {
		t.Fatalf("the reopened model: %d %s", code, body)
	}

	// an explicit model update takes effect at once
	if _, err := access.UpdateUserKey(a.ID, ka.ID, nil, &[]string{"relay/n"}); err != nil {
		t.Fatal(err)
	}
	if code, _ := modelCall(h, secretA, "relay/m"); code != http.StatusForbidden {
		t.Fatalf("a removed model grant: %d", code)
	}
	if code, body := modelCall(h, secretA, "relay/n"); code != http.StatusOK {
		t.Fatalf("the new model grant: %d %s", code, body)
	}
	if ids := modelList(h, secretA); len(ids) != 1 || ids[0] != "relay/n" {
		t.Fatalf("A's list after the update: %v", ids)
	}

	// an ownerless key keeps the old reach: no platform-catalog filter
	adminSecret, err := access.Update("add-key", access.Change{Name: "Platform admin"})
	if err != nil {
		t.Fatal(err)
	}
	if code, body := modelCall(h, adminSecret, "relay/closed"); code != http.StatusOK {
		t.Fatalf("the administrator's key on a closed model: %d %s", code, body)
	}
}

// A routing group is a user-facing model of its own: an open group may be
// granted to a key, its members are what the gateway routes over, and a
// member closed as a provider model does not take the group away. With
// groups-only, provider models stop for platform users but groups stay.
func TestAnOpenRoutingGroupIsAUserModel(t *testing.T) {
	fresh(t)
	s := settings.Load()
	s.LAN = true
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
	c, err := identity.Register("c@example.com", "C", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	var upstreamCalls atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`)
	}))
	defer up.Close()
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "test-upstream-key",
		Chat: up.URL + "/v1", Models: []string{"m"}}); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveGroup(provider.Group{ID: "flash", Name: "Flash", Members: []string{"relay/m"}}); err != nil {
		t.Fatal(err)
	}
	// the group is the platform's offer; its member stays a provider model
	if _, err := modelaccess.Set("group/flash", true, "Public Flash", "A group"); err != nil {
		t.Fatal(err)
	}
	_, secretA, err := access.CreateUserKey(a.ID, "A's model", []string{"relay/m"})
	if err != nil {
		t.Fatal(err)
	}
	_, secretC, err := access.CreateUserKey(c.ID, "C's group", []string{"group/flash"})
	if err != nil {
		t.Fatal(err)
	}
	h := New().Handler()
	if code, body := modelCall(h, secretC, "group/flash"); code != http.StatusOK {
		t.Fatalf("the open group: %d %s", code, body)
	}
	if code, _ := modelCall(h, secretC, "relay/m"); code != http.StatusForbidden {
		t.Fatal("a group key reached the member by name")
	}
	if code, body := modelCall(h, secretA, "group/flash"); code != http.StatusForbidden {
		t.Fatalf("a key not granted the group used it: %d %s", code, body)
	}
	// the group's members are implementation: closing the member as a
	// provider model does not close the platform's group, and groups-only
	// stops the member for users while the group remains
	if _, err := modelaccess.Update("relay/m", boolPtr(false), nil, nil); err != nil {
		t.Fatal(err)
	}
	if code, body := modelCall(h, secretC, "group/flash"); code != http.StatusOK {
		t.Fatalf("a group whose member is closed: %d %s", code, body)
	}
	if err := modelaccess.SetGroupsOnly(true); err != nil {
		t.Fatal(err)
	}
	if code, body := modelCall(h, secretC, "group/flash"); code != http.StatusOK {
		t.Fatalf("the group under groups-only: %d %s", code, body)
	}
	if code, _ := modelCall(h, secretC, "relay/m"); code != http.StatusForbidden {
		t.Fatal("groups-only did not close the member")
	}
	if code, _ := modelCall(h, secretC, "relay/closed"); code != http.StatusForbidden {
		t.Fatal("groups-only let a provider model through")
	}
	// a closed group is not a user's
	if _, err := modelaccess.Update("group/flash", boolPtr(false), nil, nil); err != nil {
		t.Fatal(err)
	}
	if code, _ := modelCall(h, secretC, "group/flash"); code != http.StatusForbidden {
		t.Fatal("a closed group was used")
	}
	if err := modelaccess.SetGroupsOnly(false); err != nil {
		t.Fatal(err)
	}
}
