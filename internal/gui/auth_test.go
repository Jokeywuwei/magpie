package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/identity"
)

// authRequest does a request against h, with the session cookies given.
func authRequest(h http.Handler, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func authCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie && c.Value != "" {
			return c
		}
	}
	t.Fatalf("no %s cookie in %v", SessionCookie, rec.Header().Values("Set-Cookie"))
	return nil
}

func authAccount(t *testing.T, rec *httptest.ResponseRecorder) identity.Account {
	t.Helper()
	var out struct {
		User identity.Account `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q: %v", rec.Body, err)
	}
	return out.User
}

func authSignIn(t *testing.T, h http.Handler, email, password string) *http.Cookie {
	t.Helper()
	rec := authRequest(h, "POST", "/api/auth/login", `{"email":"`+email+`","password":"`+password+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body)
	}
	return authCookie(t, rec)
}

func TestAuthRegisterLoginMeLogout(t *testing.T) {
	sandboxHome(t)
	loginAttempts.reset()
	t.Cleanup(loginAttempts.reset)
	h := Handler(nil, nil)

	// no admin: sign-up is closed, and says why
	rec := authRequest(h, "POST", "/api/auth/register", `{"email":"user@example.com","password":"correct horse battery"}`)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "closed") {
		t.Fatalf("register before an admin: %d %s", rec.Code, rec.Body)
	}

	admin, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	// an admin alone doesn't open sign-up: registration is off until the
	// admin turns it on
	rec = authRequest(h, "POST", "/api/auth/register", `{"email":"user@example.com","password":"correct horse battery"}`)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "closed") {
		t.Fatalf("register with an admin and registration off: %d %s", rec.Code, rec.Body)
	}
	if err := identity.SetRegistrationOpen(true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = identity.SetRegistrationOpen(false) })
	rec = authRequest(h, "POST", "/api/auth/register", `{"email":"user@example.com","name":"Ann","password":"correct horse battery"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	user := authAccount(t, rec)
	if user.Role != identity.RoleUser || user.Name != "Ann" || user.ID == "" {
		t.Fatalf("registered %+v", user)
	}
	cookie := authCookie(t, rec)
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Secure {
		t.Fatalf("cookie %+v", cookie)
	}
	rec = authRequest(h, "GET", "/api/auth/me", "", cookie)
	if rec.Code != http.StatusOK || authAccount(t, rec).ID != user.ID {
		t.Fatalf("me: %d %s", rec.Code, rec.Body)
	}
	rec = authRequest(h, "POST", "/api/auth/logout", "", cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", rec.Code, rec.Body)
	}
	if rec = authRequest(h, "GET", "/api/auth/me", "", cookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout: %d %s", rec.Code, rec.Body)
	}
	if rec = authRequest(h, "POST", "/api/auth/login", `{"email":"user@example.com","password":"wrong password"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d %s", rec.Code, rec.Body)
	}
	if rec = authRequest(h, "POST", "/api/auth/register", `{"email":"USER@example.com","password":"correct horse battery"}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body)
	}
	adminCookie := authSignIn(t, h, "admin@example.com", "correct horse battery")
	if adminCookie.Value == "" {
		t.Fatal("the admin's sign-in has no token")
	}
	// the part of the email before @ is a username: it signs in too, and
	// the handler keeps the email field for it.
	rec = authRequest(h, "POST", "/api/auth/login", `{"email":"Admin","password":"correct horse battery"}`)
	if rec.Code != http.StatusOK || authAccount(t, rec).ID != admin.ID {
		t.Fatalf("sign in by username: %d %s", rec.Code, rec.Body)
	}
	rec = authRequest(h, "POST", "/api/auth/login", `{"email":"user","password":"correct horse battery"}`)
	if rec.Code != http.StatusOK || authAccount(t, rec).ID != user.ID {
		t.Fatalf("sign in by the registered account's username: %d %s", rec.Code, rec.Body)
	}
	if _, err := identity.SetDisabled(admin.ID, true); err == nil {
		t.Fatal("the only admin was disabled")
	}
}

func TestAdminRoutesRequireAnAdmin(t *testing.T) {
	sandboxHome(t)
	h := Handler(nil, nil)
	admin, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := identity.Register("user@example.com", "Ann", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	adminCookie := authSignIn(t, h, "admin@example.com", "correct horse battery")
	userCookie := authSignIn(t, h, "user@example.com", "correct horse battery")

	if rec := authRequest(h, "GET", "/api/admin/users", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/admin/users", "", userCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("user session: %d %s", rec.Code, rec.Body)
	}
	rec := authRequest(h, "GET", "/api/admin/users", "", adminCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), "$"+"argon") {
		t.Fatalf("the admin list carries a hash: %s", rec.Body)
	}
	var list struct {
		Users []identity.Account `json:"users"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Users) != 2 {
		t.Fatalf("users %v, %v", list.Users, err)
	}
	if rec := authRequest(h, "POST", "/api/admin/users", `{"email":"third@example.com","password":"correct horse battery","role":"user"}`, userCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("user creating an account: %d %s", rec.Code, rec.Body)
	}
	rec = authRequest(h, "POST", "/api/admin/users", `{"email":"third@example.com","name":"Cy","password":"correct horse battery","role":"user"}`, adminCookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("admin creating an account: %d %s", rec.Code, rec.Body)
	}
	third := authAccount(t, rec)
	// a role change, and the last admin kept
	if rec := authRequest(h, "PATCH", "/api/admin/users/"+third.ID, `{"role":"admin"}`, adminCookie); rec.Code != http.StatusOK {
		t.Fatalf("promote: %d %s", rec.Code, rec.Body)
	}
	thirdCookie := authSignIn(t, h, "third@example.com", "correct horse battery")
	if rec := authRequest(h, "PATCH", "/api/admin/users/"+admin.ID, `{"role":"user"}`, thirdCookie); rec.Code != http.StatusOK {
		t.Fatalf("demote with another admin: %d %s", rec.Code, rec.Body)
	}
	// a role change ends the sessions that carried the old role
	if rec := authRequest(h, "GET", "/api/admin/users", "", adminCookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("a demoted admin's old session still manages: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "PATCH", "/api/admin/users/"+third.ID, `{"disabled":true}`, thirdCookie); rec.Code != http.StatusConflict {
		t.Fatalf("disable the last admin: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "PATCH", "/api/admin/users/"+third.ID, `{"role":"user"}`, thirdCookie); rec.Code != http.StatusConflict {
		t.Fatalf("demote the last admin: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "DELETE", "/api/admin/users/"+third.ID, "", thirdCookie); rec.Code != http.StatusConflict {
		t.Fatalf("remove the last admin: %d %s", rec.Code, rec.Body)
	}
	// a password reset signs the user's browser out
	if rec := authRequest(h, "POST", "/api/admin/users/"+user.ID+"/password", `{"password":"another password"}`, thirdCookie); rec.Code != http.StatusNoContent {
		t.Fatalf("reset: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/auth/me", "", userCookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("me after a reset: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "POST", "/api/admin/users/"+user.ID+"/password", `{"password":"short"}`, thirdCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("short password: %d %s", rec.Code, rec.Body)
	}
	// and a removal
	if rec := authRequest(h, "DELETE", "/api/admin/users/"+user.ID, "", thirdCookie); rec.Code != http.StatusNoContent {
		t.Fatalf("remove: %d %s", rec.Code, rec.Body)
	}
	if _, err := identity.ByID(user.ID); err == nil {
		t.Fatal("the removed account is still here")
	}
}

func TestWebAuthBypassesThePageKey(t *testing.T) {
	sandboxHome(t)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	h := webHandler("3430", "page-key-at-least-sixteen", time.Hour, nil, Handler(nil, nil))

	// the page's own API is still behind the page key
	rec := authRequest(h, "GET", "/api/state", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("/api/state without the page key: %d %s", rec.Code, rec.Body)
	}
	// the account routes are not: no page cookie, and a real sign-in
	rec = authRequest(h, "GET", "/api/admin/users", "")
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "admin") {
		t.Fatalf("admin route without a page key: %d %s", rec.Code, rec.Body)
	}
	// the admin page's key is not an account: it doesn't open the account
	// API, and a user's session doesn't open the admin page or its API
	pageCookie := &http.Cookie{Name: "magpie_web_3430", Value: "page-key-at-least-sixteen"}
	if rec = authRequest(h, "GET", "/api/admin/users", "", pageCookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("the page key opened the admin account API: %d %s", rec.Code, rec.Body)
	}
	rec = authRequest(h, "POST", "/api/auth/login", `{"email":"admin@example.com","password":"correct horse battery"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login without a page key: %d %s", rec.Code, rec.Body)
	}
	cookie := authCookie(t, rec)
	if rec = authRequest(h, "GET", "/api/auth/me", "", cookie); rec.Code != http.StatusOK {
		t.Fatalf("me: %d %s", rec.Code, rec.Body)
	}
	if rec = authRequest(h, "GET", "/api/admin/users", "", cookie); rec.Code != http.StatusOK {
		t.Fatalf("admin list: %d %s", rec.Code, rec.Body)
	}
	user, err := identity.Register("user@example.com", "Ann", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := identity.NewSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	userCookie := &http.Cookie{Name: SessionCookie, Value: token}
	// a signed-in user is not an admin: the page and its API are refused,
	// with no page key
	if rec = authRequest(h, "GET", "/api/providers", "", userCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("a user session opened the admin API: %d %s", rec.Code, rec.Body)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(userCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a user session opened the admin page: %d %s", rec.Code, rec.Body)
	}
}

func TestCrossOriginLoginRefused(t *testing.T) {
	sandboxHome(t)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "http://magpie.test/api/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	Handler(nil, nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin login: %d %s", rec.Code, rec.Body)
	}
}
