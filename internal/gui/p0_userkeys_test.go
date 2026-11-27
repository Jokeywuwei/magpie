package gui

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/identity"
)

func p0GUIPassword(t *testing.T) string {
	t.Helper()
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// The portal must refuse an explicit empty grant. The old check looked at
// the raw array's length, so [" "] passed, was trimmed to nothing, and made
// a key the gateway then treated as an unrestricted one.
func TestUserKeyAPIRejectsEmptyGrant(t *testing.T) {
	sandboxHome(t)
	openGUIModel(t)
	pw := p0GUIPassword(t)
	if _, err := identity.Add("admin@example.com", "Admin", pw, identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Register("a@example.com", "A", pw); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)
	cookie := authSignIn(t, h, "a@example.com", pw)

	for _, body := range []string{
		`{"name":"x","models":[""]}`,
		`{"name":"x","models":[" "]}`,
		`{"name":"x","models":[null]}`,
		`{"name":"x","models":[]}`,
		`{"name":"x"}`,
	} {
		if rec := authRequest(h, "POST", "/api/user/keys", body, cookie); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	made := authRequest(h, "POST", "/api/user/keys", `{"name":"ok","models":["relay/m"]}`, cookie)
	if made.Code != http.StatusCreated {
		t.Fatalf("a real grant was refused: %d %s", made.Code, made.Body)
	}
	var out struct {
		Key access.UserKey `json:"key"`
	}
	if err := json.Unmarshal(made.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		`{"models":[""]}`,
		`{"models":[" "]}`,
		`{"models":[null]}`,
		`{"models":[]}`,
	} {
		if rec := authRequest(h, "PATCH", "/api/user/keys/"+out.Key.ID, body, cookie); rec.Code != http.StatusBadRequest {
			t.Errorf("PATCH %s: %d %s", body, rec.Code, rec.Body)
		}
	}
	// the key's grant is unchanged by the refusals
	rec := authRequest(h, "GET", "/api/user/keys", "", cookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"relay/m"`) {
		t.Fatalf("the refused PATCHes changed the grant: %d %s", rec.Code, rec.Body)
	}
}
