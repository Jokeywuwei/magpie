package gui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/identity"
)

func adminWebHandler(t *testing.T) http.Handler {
	t.Helper()
	return webHandler("3430", "page-key-at-least-sixteen", time.Hour, nil, Handler(nil, nil))
}

func webPageGet(t *testing.T, h http.Handler, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Accept", "text/html")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdministratorSessionOpensTheManagementPage(t *testing.T) {
	sandboxHome(t)
	loginAttempts.reset()
	t.Cleanup(loginAttempts.reset)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Register("user@example.com", "Ann", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	h := adminWebHandler(t)

	// nobody signed in: the management page goes to the platform sign-in,
	// and says nothing of the page
	if rec := webPageGet(t, h, "/"); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/portal/login?next=%2F" {
		t.Fatalf("the page without a session: %d %q %s", rec.Code, rec.Header().Get("Location"), rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/state", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("the state without a session or page key: %d %s", rec.Code, rec.Body)
	}

	// a user's session opens neither the page nor any management API
	userCookie := authSignIn(t, h, "user@example.com", "correct horse battery")
	if rec := webPageGet(t, h, "/", userCookie); rec.Code != http.StatusForbidden || strings.Contains(rec.Body.String(), "Providers") {
		t.Fatalf("a user opened the page: %d %s", rec.Code, rec.Body)
	}
	for _, api := range []string{"/api/state", "/api/providers", "/api/settings", "/api/admin/registration"} {
		if rec := authRequest(h, "GET", api, "", userCookie); rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
			t.Errorf("a user opened %s: %d %s", api, rec.Code, rec.Body)
		}
	}
	// a role in a query, a header or a body changes nothing
	for _, path := range []string{"/api/providers?role=admin&user=admin&owner=admin", "/api/state?admin=1"} {
		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(userCookie)
		req.Header.Set("X-Magpie-User-Role", "admin")
		req.Header.Set("X-HC-User-Role", "ADMIN")
		req.Header.Set("X-Forwarded-User", "admin")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s with forged role fields: %d %s", path, rec.Code, rec.Body)
		}
	}

	// the administrator's session opens the whole page and its API with no
	// page key at all
	adminCookie := authSignIn(t, h, "admin@example.com", "correct horse battery")
	rec := webPageGet(t, h, "/", adminCookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Providers") {
		t.Fatalf("an admin's page without a page key: %d %s", rec.Code, rec.Body)
	}
	for _, api := range []string{"/api/state", "/api/providers", "/api/settings", "/api/admin/users", "/api/admin/registration"} {
		if rec := authRequest(h, "GET", api, "", adminCookie); rec.Code != http.StatusOK {
			t.Errorf("an admin's %s without a page key: %d %s", api, rec.Code, rec.Body)
		}
	}
	// the management page's files too
	if rec := webPageGet(t, h, "/app.js", adminCookie); rec.Code != http.StatusOK {
		t.Fatalf("an admin's page file without a page key: %d", rec.Code)
	}
	// a stale key in an old bookmark doesn't turn an admin's session away
	if rec := webPageGet(t, h, "/?k=an-old-key", adminCookie); rec.Code != http.StatusOK {
		t.Fatalf("an admin with a stale page-key query: %d %s", rec.Code, rec.Body)
	}
	// a write another site sends with the admin's cookie is refused by the
	// page's own origin guard, as is a read it sends
	csrf := httptest.NewRequest("POST", "http://magpie.test/api/settings", strings.NewReader(`{"theme":"dark"}`))
	csrf.Header.Set("Content-Type", "application/json")
	csrf.Header.Set("Origin", "https://evil.example")
	csrf.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, csrf)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a cross-origin write with an admin session: %d %s", rec.Code, rec.Body)
	}
	read := httptest.NewRequest("GET", "/api/state", nil)
	read.Header.Set("Sec-Fetch-Site", "cross-site")
	read.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, read)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a cross-site read with an admin session: %d %s", rec.Code, rec.Body)
	}
}

func TestAdministratorSessionEndsWithTheAccount(t *testing.T) {
	sandboxHome(t)
	loginAttempts.reset()
	t.Cleanup(loginAttempts.reset)
	admin, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Add("second@example.com", "Second", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	h := adminWebHandler(t)

	adminCookie := authSignIn(t, h, "admin@example.com", "correct horse battery")
	if rec := webPageGet(t, h, "/", adminCookie); rec.Code != http.StatusOK {
		t.Fatalf("admin page: %d %s", rec.Code, rec.Body)
	}
	// a sign-out ends it
	if rec := authRequest(h, "POST", "/api/auth/logout", "", adminCookie); rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", rec.Code, rec.Body)
	}
	if rec := webPageGet(t, h, "/", adminCookie); rec.Code != http.StatusFound {
		t.Fatalf("page after logout: %d %s", rec.Code, rec.Body)
	}

	// disabling the account ends it
	adminCookie = authSignIn(t, h, "admin@example.com", "correct horse battery")
	if _, err := identity.SetDisabled(admin.ID, true); err != nil {
		t.Fatal(err)
	}
	if rec := webPageGet(t, h, "/", adminCookie); rec.Code != http.StatusFound {
		t.Fatalf("page after disabling: %d %s", rec.Code, rec.Body)
	}
	if _, err := identity.SetDisabled(admin.ID, false); err != nil {
		t.Fatal(err)
	}

	// a password change ends it, and the old password no longer signs in
	adminCookie = authSignIn(t, h, "admin@example.com", "correct horse battery")
	if err := identity.SetPassword(admin.ID, "another horse battery"); err != nil {
		t.Fatal(err)
	}
	if rec := webPageGet(t, h, "/", adminCookie); rec.Code != http.StatusFound {
		t.Fatalf("page after a password change: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "POST", "/api/auth/login", `{"email":"admin@example.com","password":"correct horse battery"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("the old password still signs in: %d %s", rec.Code, rec.Body)
	}
	adminCookie = authSignIn(t, h, "admin@example.com", "another horse battery")

	// a demotion ends it: the browser signs in again as the user it now is
	if _, err := identity.SetRole(admin.ID, identity.RoleUser); err != nil {
		t.Fatal(err)
	}
	if rec := webPageGet(t, h, "/", adminCookie); rec.Code != http.StatusFound {
		t.Fatalf("page after demotion: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/admin/users", "", adminCookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("an old session after demotion still manages: %d %s", rec.Code, rec.Body)
	}
}

func TestPageKeyCompatibilityIsUnchanged(t *testing.T) {
	sandboxHome(t)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	user, err := identity.Register("user@example.com", "Ann", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	h := adminWebHandler(t)
	pageCookie := &http.Cookie{Name: "magpie_web_3430", Value: "page-key-at-least-sixteen"}

	// the old cookie still opens the page and the page's API, with no
	// account session at all
	if rec := webPageGet(t, h, "/", pageCookie); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Providers") {
		t.Fatalf("page key's page: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/state", "", pageCookie); rec.Code != http.StatusOK {
		t.Fatalf("page key's API: %d %s", rec.Code, rec.Body)
	}
	// the old key was never an account, and the account API is unchanged
	if rec := authRequest(h, "GET", "/api/admin/users", "", pageCookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("page key opened the admin account API: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/admin/registration", "", pageCookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("page key opened registration: %d %s", rec.Code, rec.Body)
	}
	// a user's browser that also has the page key keeps exactly what the
	// key gave it; the key is an operator's credential, not the session's
	userToken, _, err := identity.NewSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	userCookie := &http.Cookie{Name: SessionCookie, Value: userToken}
	if rec := webPageGet(t, h, "/", pageCookie, userCookie); rec.Code != http.StatusOK {
		t.Fatalf("page key and a user session: %d %s", rec.Code, rec.Body)
	}
	// the key's link still trades for its cookie
	link := webPageGet(t, h, "/?k=page-key-at-least-sixteen")
	if link.Code != http.StatusSeeOther || link.Header().Get("Location") != "/" {
		t.Fatalf("page key's link: %d %q", link.Code, link.Header().Get("Location"))
	}
	if ck := link.Result().Cookies(); len(ck) != 1 || ck[0].Name != "magpie_web_3430" || ck[0].Value != "page-key-at-least-sixteen" || !ck[0].HttpOnly || ck[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("page key's cookie: %+v", ck)
	}
}

func TestAdminRegistrationSwitchAndCSRF(t *testing.T) {
	sandboxHome(t)
	loginAttempts.reset()
	t.Cleanup(loginAttempts.reset)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.Register("user@example.com", "Ann", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)
	adminCookie := authSignIn(t, h, "admin@example.com", "correct horse battery")
	userCookie := authSignIn(t, h, "user@example.com", "correct horse battery")

	if open, err := identity.RegistrationOpen(); err != nil || open {
		t.Fatalf("registration starts open: %v %v", open, err)
	}
	rec := authRequest(h, "GET", "/api/admin/registration", "", adminCookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"open":false`) {
		t.Fatalf("registration state: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "GET", "/api/admin/registration", "", userCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("a user read registration: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "POST", "/api/admin/registration", `{"open":true}`, userCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("a user opened registration: %d %s", rec.Code, rec.Body)
	}
	// a cross-origin write is refused, with no change
	req := httptest.NewRequest("POST", "http://magpie.test/api/admin/registration", strings.NewReader(`{"open":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(adminCookie)
	cross := httptest.NewRecorder()
	h.ServeHTTP(cross, req)
	if cross.Code != http.StatusForbidden {
		t.Fatalf("cross-origin registration: %d %s", cross.Code, cross.Body)
	}
	if open, _ := identity.RegistrationOpen(); open {
		t.Fatal("the refused write opened registration")
	}
	// an explicit field is needed
	if rec := authRequest(h, "POST", "/api/admin/registration", `{}`, adminCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("no open field: %d %s", rec.Code, rec.Body)
	}
	// the administrator opens it, and a visitor can register; then closes it
	if rec := authRequest(h, "POST", "/api/admin/registration", `{"open":true}`, adminCookie); rec.Code != http.StatusOK {
		t.Fatalf("open registration: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "POST", "/api/auth/register", `{"email":"new@example.com","name":"New","password":"correct horse battery"}`); rec.Code != http.StatusCreated {
		t.Fatalf("register while open: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "POST", "/api/admin/registration", `{"open":false}`, adminCookie); rec.Code != http.StatusOK {
		t.Fatalf("close registration: %d %s", rec.Code, rec.Body)
	}
	if rec := authRequest(h, "POST", "/api/auth/register", `{"email":"later@example.com","password":"correct horse battery"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("register while closed: %d %s", rec.Code, rec.Body)
	}
	// a role in a registration body is not a field, and makes no admin
	if rec := authRequest(h, "POST", "/api/auth/register", `{"email":"sneaky@example.com","password":"correct horse battery","role":"admin"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("register with a role: %d %s", rec.Code, rec.Body)
	}
	if _, err := identity.ByEmail("sneaky@example.com"); err == nil {
		t.Fatal("a role-forged registration made an account")
	}
}

func TestLoginIsRateLimited(t *testing.T) {
	sandboxHome(t)
	loginAttempts.reset()
	t.Cleanup(loginAttempts.reset)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)
	for i := 0; i < loginEmailLimit; i++ {
		if rec := authRequest(h, "POST", "/api/auth/login", `{"email":"admin@example.com","password":"wrong password"}`); rec.Code != http.StatusUnauthorized {
			t.Fatalf("wrong try %d: %d %s", i, rec.Code, rec.Body)
		}
	}
	rec := authRequest(h, "POST", "/api/auth/login", `{"email":"admin@example.com","password":"correct horse battery"}`)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" || !strings.Contains(rec.Body.String(), "too many") {
		t.Fatalf("the limit after %d failures: %d %q %s", loginEmailLimit, rec.Code, rec.Header().Get("Retry-After"), rec.Body)
	}
	loginAttempts.reset()
	if rec := authRequest(h, "POST", "/api/auth/login", `{"email":"admin@example.com","password":"correct horse battery"}`); rec.Code != http.StatusOK {
		t.Fatalf("after the limit was cleared: %d %s", rec.Code, rec.Body)
	}
}

func TestSecureCookiesBehindHTTPS(t *testing.T) {
	sandboxHome(t)
	loginAttempts.reset()
	t.Cleanup(loginAttempts.reset)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)

	// a direct TLS connection is HTTPS, whatever the headers say
	req := httptest.NewRequest("POST", "https://magpie.test/api/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("TLS login: %d %s", rec.Code, rec.Body)
	}
	sc := rec.Header().Get("Set-Cookie")
	if !strings.Contains(sc, "magpie_session=") || !strings.Contains(sc, "Secure") || !strings.Contains(sc, "HttpOnly") || !strings.Contains(sc, "SameSite=Lax") {
		t.Fatalf("session cookie over TLS: %q", sc)
	}

	// a forwarding header alone is not trusted: no public HTTPS address is
	// configured, so a plain HTTP request that writes it gets no Secure
	loginAttempts.reset()
	req = httptest.NewRequest("POST", "http://magpie.test/api/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("untrusted header login: %d %s", rec.Code, rec.Body)
	}
	if sc := rec.Header().Get("Set-Cookie"); strings.Contains(sc, "Secure") {
		t.Fatalf("an untrusted X-Forwarded-Proto set Secure: %q", sc)
	}

	// configured as a proxy's public address, the header on that host is
	// taken; so is the page key's cookie
	t.Setenv("MAGPIE_PUBLIC_URL", "https://magpie.test")
	loginAttempts.reset()
	req = httptest.NewRequest("POST", "http://magpie.test/api/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Host = "magpie.test"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trusted proxy login: %d %s", rec.Code, rec.Body)
	}
	if sc := rec.Header().Get("Set-Cookie"); !strings.Contains(sc, "Secure") {
		t.Fatalf("a configured proxy's HTTPS header set no Secure: %q", sc)
	}
	// the header on another host is the caller's own, not the configured proxy's
	loginAttempts.reset()
	req = httptest.NewRequest("POST", "http://other.test/api/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Host = "other.test"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("other-host login: %d %s", rec.Code, rec.Body)
	}
	if sc := rec.Header().Get("Set-Cookie"); strings.Contains(sc, "Secure") {
		t.Fatalf("a forwarding header on another host set Secure: %q", sc)
	}
	// an HTTP or unusable public address is not a trusted HTTPS proxy
	t.Setenv("MAGPIE_PUBLIC_URL", "http://magpie.test")
	loginAttempts.reset()
	req = httptest.NewRequest("POST", "http://magpie.test/api/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Host = "magpie.test"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP public address login: %d %s", rec.Code, rec.Body)
	}
	if sc := rec.Header().Get("Set-Cookie"); strings.Contains(sc, "Secure") {
		t.Fatalf("an HTTP public address set Secure: %q", sc)
	}
	t.Setenv("MAGPIE_PUBLIC_URL", "https://magpie.test")
	web := adminWebHandler(t)
	link := httptest.NewRequest("GET", "http://magpie.test/?k=page-key-at-least-sixteen", nil)
	link.Host = "magpie.test"
	link.Header.Set("X-Forwarded-Proto", "https")
	lrec := httptest.NewRecorder()
	web.ServeHTTP(lrec, link)
	if sc := lrec.Header().Get("Set-Cookie"); lrec.Code != http.StatusSeeOther || !strings.Contains(sc, "Secure") {
		t.Fatalf("page key cookie at a configured proxy: %d %q", lrec.Code, sc)
	}
}
