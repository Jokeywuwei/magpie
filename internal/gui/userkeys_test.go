package gui

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/identity"
)

func TestUserKeysAPI(t *testing.T) {
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

	// no sign-in, no keys
	if rec := authRequest(h, "GET", "/api/user/keys", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("keys without a session: %d %s", rec.Code, rec.Body)
	}
	// A makes one: the full secret comes back once, and nothing else does
	rec := authRequest(h, "POST", "/api/user/keys", `{"name":"Laptop","models":["relay/m"]}`, aCookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var made struct {
		Key    access.UserKey `json:"key"`
		Secret string         `json:"secret"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &made); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(made.Secret, access.UserPrefix) || made.Key.ID == "" || made.Key.Name != "Laptop" || made.Key.Prefix == "" || made.Key.Off {
		t.Fatalf("created %+v %q", made, made.Secret)
	}
	if strings.Contains(rec.Body.String(), "hash") {
		t.Fatal("the create answer carries a hash")
	}
	// A's list has the prefix, never the secret or the hash
	rec = authRequest(h, "GET", "/api/user/keys", "", aCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var list struct {
		Keys []access.UserKey `json:"keys"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Keys) != 1 || list.Keys[0].ID != made.Key.ID {
		t.Fatalf("list %+v %v", list, err)
	}
	if strings.Contains(rec.Body.String(), made.Secret) || strings.Contains(rec.Body.String(), "hash") {
		t.Fatalf("the list carries a secret or hash: %s", rec.Body)
	}
	// B neither sees nor touches it
	if rec := authRequest(h, "GET", "/api/user/keys", "", bCookie); rec.Code != http.StatusOK {
		t.Fatalf("B list: %d %s", rec.Code, rec.Body)
	} else {
		var bList struct {
			Keys []access.UserKey `json:"keys"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &bList)
		if len(bList.Keys) != 0 {
			t.Fatalf("B sees %+v", bList.Keys)
		}
	}
	if rec := authRequest(h, "PATCH", "/api/user/keys/"+made.Key.ID, `{"off":true}`, bCookie); rec.Code != http.StatusNotFound {
		t.Fatalf("B turned off A's key: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "DELETE", "/api/user/keys/"+made.Key.ID, "", bCookie); rec.Code != http.StatusNotFound {
		t.Fatalf("B removed A's key: %d %s", rec.Code, rec.Body)
	}
	if _, err := access.Check(made.Secret); err != nil {
		t.Fatalf("A's key after B's tries: %v", err)
	}
	// A turns it off, and the gateway stops taking it at once
	if rec := authRequest(h, "PATCH", "/api/user/keys/"+made.Key.ID, `{"off":true}`, aCookie); rec.Code != http.StatusOK {
		t.Fatalf("off: %d %s", rec.Code, rec.Body)
	}
	if _, err := access.Check(made.Secret); err == nil {
		t.Fatal("a key turned off through the console still signs in")
	}
	if rec := authRequest(h, "PATCH", "/api/user/keys/"+made.Key.ID, `{"off":false}`, aCookie); rec.Code != http.StatusOK {
		t.Fatalf("on: %d %s", rec.Code, rec.Body)
	}
	if _, err := access.Check(made.Secret); err != nil {
		t.Fatalf("a key turned on: %v", err)
	}
	// a bad name is refused
	if rec := authRequest(h, "POST", "/api/user/keys", `{"name":"  ","models":["relay/m"]}`, aCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty name: %d %s", rec.Code, rec.Body)
	}
	// delete, and it stops at once
	if rec := authRequest(h, "DELETE", "/api/user/keys/"+made.Key.ID, "", aCookie); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if _, err := access.Check(made.Secret); err == nil {
		t.Fatal("a deleted key still signs in")
	}
	if rec := authRequest(h, "GET", "/api/user/keys", "", aCookie); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), made.Key.ID) {
		t.Fatalf("after delete: %d %s", rec.Code, rec.Body)
	}
}

func TestUserKeysAreNotTheAdmins(t *testing.T) {
	sandboxHome(t)
	openGUIModel(t)
	admin, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := identity.Register("a@example.com", "A", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)
	adminCookie := authSignIn(t, h, "admin@example.com", "correct horse battery")
	userCookie := authSignIn(t, h, "a@example.com", "correct horse battery")
	// the admin's own list is the admin's own keys, not A's
	if rec := authRequest(h, "POST", "/api/user/keys", `{"name":"Admin's","models":["relay/m"]}`, adminCookie); rec.Code != http.StatusCreated {
		t.Fatalf("admin create: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/user/keys", "", adminCookie); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Admin's") {
		t.Fatalf("admin list: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/user/keys", "", userCookie); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "Admin's") {
		t.Fatalf("A sees the admin's keys: %d %s", rec.Code, rec.Body)
	}
	_ = admin
	_ = user
}
