package gui

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/modelaccess"
	"github.com/yetone/magpie/internal/provider"
)

func saveGUIRelay(t *testing.T, models ...string) {
	t.Helper()
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "test-upstream-key",
		Chat: "https://relay.example/v1", Models: models}); err != nil {
		t.Fatal(err)
	}
}

func openGUIModel(t *testing.T) {
	t.Helper()
	saveGUIRelay(t, "m", "closed")
	if _, err := modelaccess.Set("relay/m", true, "M", "The open one"); err != nil {
		t.Fatal(err)
	}
}

func TestModelAccessAdminAndUserAPIs(t *testing.T) {
	sandboxHome(t)
	saveGUIRelay(t, "m", "closed")
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Register("a@example.com", "A", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)
	adminCookie := authSignIn(t, h, "admin@example.com", "correct horse battery")
	userCookie := authSignIn(t, h, "a@example.com", "correct horse battery")

	if rec := authRequest(h, "GET", "/api/user/models", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("models without a session: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/admin/model-access", "", userCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("a user reads the admin catalog: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "POST", "/api/admin/model-access", `{"id":"relay/m","open":true}`, userCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("a user opens a model: %d %s", rec.Code, rec.Body)
	}
	// the admin sees every served model, all closed, and users see none
	rec := authRequest(h, "GET", "/api/admin/model-access", "", adminCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin catalog: %d %s", rec.Code, rec.Body)
	}
	var all struct {
		Models []modelaccess.Model `json:"models"`
		Total  int                 `json:"total"`
		Open   int                 `json:"open"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &all); err != nil || all.Total != 2 || all.Open != 0 || len(all.Models) != 2 {
		t.Fatalf("admin catalog %+v %v", all, err)
	}
	if rec := authRequest(h, "GET", "/api/user/models", "", userCookie); rec.Code != http.StatusOK {
		t.Fatalf("user catalog: %d %s", rec.Code, rec.Body)
	} else if strings.Contains(rec.Body.String(), "relay/m") || strings.Contains(rec.Body.String(), "closed") {
		t.Fatalf("a closed model reached a user: %s", rec.Body)
	}

	// the admin opens it with the name and description users see
	rec = authRequest(h, "POST", "/api/admin/model-access", `{"id":"relay/m","open":true,"name":"Public M","description":"For everyone"}`, adminCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("open: %d %s", rec.Code, rec.Body)
	}
	var made struct {
		Model modelaccess.Model `json:"model"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &made); err != nil || !made.Model.Open || made.Model.Name != "Public M" {
		t.Fatalf("opened %+v %v", made, err)
	}
	rec = authRequest(h, "GET", "/api/user/models", "", userCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("user catalog after opening: %d %s", rec.Code, rec.Body)
	}
	var userModels struct {
		Models []modelaccess.Model `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &userModels); err != nil || len(userModels.Models) != 1 ||
		userModels.Models[0].ID != "relay/m" || userModels.Models[0].Name != "Public M" || userModels.Models[0].Description != "For everyone" {
		t.Fatalf("the user's model: %+v %v", userModels, err)
	}

	// closing by only its open field keeps the admin's name
	rec = authRequest(h, "POST", "/api/admin/model-access", `{"id":"relay/m","open":false}`, adminCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("close: %d %s", rec.Code, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &made); err != nil || made.Model.Open || made.Model.Name != "Public M" {
		t.Fatalf("closed %+v %v", made, err)
	}
	if rec := authRequest(h, "GET", "/api/user/models", "", userCookie); strings.Contains(rec.Body.String(), "relay/m") {
		t.Fatalf("a closed model still on the plaza: %s", rec.Body)
	}
	// a name alone may be changed
	rec = authRequest(h, "POST", "/api/admin/model-access", `{"id":"relay/m","name":"Renamed"}`, adminCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename: %d %s", rec.Code, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &made); err != nil || made.Model.Name != "Renamed" || made.Model.Open {
		t.Fatalf("renamed %+v %v", made, err)
	}
	// nothing, an unknown model and an over-long name are refused
	for _, body := range []string{
		`{"id":"relay/m"}`,
		`{"id":"relay/nope","open":true}`,
		`{"id":"relay/m","open":true,"name":"` + strings.Repeat("x", 81) + `"}`,
	} {
		if rec := authRequest(h, "POST", "/api/admin/model-access", body, adminCookie); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
}

func TestUserKeyModelGrantsAPI(t *testing.T) {
	sandboxHome(t)
	openGUIModel(t)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Register("a@example.com", "A", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Register("b@example.com", "B", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)
	aCookie := authSignIn(t, h, "a@example.com", "correct horse battery")
	bCookie := authSignIn(t, h, "b@example.com", "correct horse battery")

	for _, body := range []string{
		`{"name":"No models"}`,
		`{"name":"Nothing here","models":[]}`,
		`{"name":"A closed one","models":["relay/closed"]}`,
	} {
		if rec := authRequest(h, "POST", "/api/user/keys", body, aCookie); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	rec := authRequest(h, "POST", "/api/user/keys", `{"name":"Laptop","models":["relay/m"]}`, aCookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var made struct {
		Key    access.UserKey `json:"key"`
		Secret string         `json:"secret"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &made); err != nil || len(made.Key.Models) != 1 || made.Key.Models[0] != "relay/m" || made.Key.LegacyModels {
		t.Fatalf("created %+v %v", made, err)
	}
	// the list shows the grant and never a hash or secret
	rec = authRequest(h, "GET", "/api/user/keys", "", aCookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "relay/m") || strings.Contains(rec.Body.String(), made.Secret) || strings.Contains(rec.Body.String(), "hash") {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	// the console may change the grant, but only to open models
	if rec := authRequest(h, "PATCH", "/api/user/keys/"+made.Key.ID, `{"models":["relay/closed"]}`, aCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("closed model in an update: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "PATCH", "/api/user/keys/"+made.Key.ID, `{"models":["relay/m"],"off":true}`, aCookie); rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if _, err := access.Check(made.Secret); err == nil {
		t.Fatal("a key turned off with its model update still signs in")
	}
	// B may not change it
	if rec := authRequest(h, "PATCH", "/api/user/keys/"+made.Key.ID, `{"models":["relay/m"]}`, bCookie); rec.Code != http.StatusNotFound {
		t.Fatalf("B changed A's grant: %d %s", rec.Code, rec.Body)
	}
}

func TestPortalAdminModelsPage(t *testing.T) {
	sandboxHome(t)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	user, err := identity.Register("a@example.com", "A", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := identity.ByEmail("admin@example.com")
	if err != nil {
		t.Fatal(err)
	}
	userToken, _, err := identity.NewSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	adminToken, _, err := identity.NewSession(admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	h := webHandler("3430", "page-key-at-least-sixteen", 0, nil, Handler(nil, nil))

	rec := portalRequest(h, "/portal/admin-models")
	if rec.Code != http.StatusFound {
		t.Fatalf("anonymous: %d %s", rec.Code, rec.Body)
	}
	if want := "/portal/login?next=" + url.QueryEscape("/portal/admin-models"); rec.Header().Get("Location") != want {
		t.Fatalf("anonymous redirects to %q, want %q", rec.Header().Get("Location"), want)
	}
	if rec := portalRequest(h, "/portal/admin-models", &http.Cookie{Name: SessionCookie, Value: userToken}); rec.Code != http.StatusForbidden {
		t.Fatalf("a user opened the admin page: %d %s", rec.Code, rec.Body)
	}
	rec = portalRequest(h, "/portal/admin-models", &http.Cookie{Name: SessionCookie, Value: adminToken})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `data-page="admin-models"`) {
		t.Fatalf("an admin opened the page: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "page-key-at-least-sixteen") {
		t.Fatal("the admin portal page carries the admin page key")
	}
	for _, f := range []string{"/portal/models.js", "/portal/admin-models.js"} {
		if rec := portalRequest(h, f); rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/javascript") {
			t.Errorf("%s: %d %q", f, rec.Code, rec.Header().Get("Content-Type"))
		}
	}
}
