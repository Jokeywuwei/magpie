package gui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/identity"
)

func portalRequest(h http.Handler, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Accept", "text/html")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPortalPublicPagesNeedNoAdminKey(t *testing.T) {
	sandboxHome(t)
	const key = "page-key-at-least-sixteen"
	h := webHandler("3430", key, time.Hour, nil, Handler(nil, nil))

	for _, c := range []struct{ path, page string }{
		{"/portal/", "home"},
		{"/portal/login", "login"},
		{"/portal/register", "register"},
	} {
		rec := portalRequest(h, c.path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", c.path, rec.Code, rec.Body)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s: Content-Type %q", c.path, ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `data-page="`+c.page+`"`) {
			t.Errorf("%s: body isn't the %s page: %s", c.path, c.page, body[:200])
		}
		if strings.Contains(body, key) || strings.Contains(body, "?k=") {
			t.Errorf("%s carries the admin page key", c.path)
		}
		if rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: headers %v", c.path, rec.Header())
		}
	}
	// its own files, and only those
	for _, f := range []struct{ path, ct string }{
		{"/portal/portal.css", "text/css"},
		{"/portal/portal.js", "text/javascript"},
		{"/portal/usage.js", "text/javascript"},
	} {
		rec := portalRequest(h, f.path)
		if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), f.ct) {
			t.Errorf("%s: %d %q", f.path, rec.Code, rec.Header().Get("Content-Type"))
		}
	}
	for _, path := range []string{"/portal/nope", "/portal/index.html", "/portal/data.json"} {
		if rec := portalRequest(h, path); rec.Code != http.StatusNotFound {
			t.Errorf("%s: %d %s", path, rec.Code, rec.Body)
		}
	}
	// the admin page is still the page key's, or an admin session's, alone
	if rec := portalRequest(h, "/"); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/portal/login?next=%2F" {
		t.Fatalf("the admin page without a key or a session: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	pageCookie := &http.Cookie{Name: "magpie_web_3430", Value: key}
	if rec := portalRequest(h, "/", pageCookie); rec.Code != http.StatusOK {
		t.Fatalf("the admin page with its key: %d %s", rec.Code, rec.Body)
	}
}

func TestPortalConsoleNeedsASignedInAccount(t *testing.T) {
	sandboxHome(t)
	if _, err := identity.Add("admin@example.com", "Admin", "correct horse battery", identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	user, err := identity.Register("user@example.com", "Ann", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	userToken, _, err := identity.NewSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := identity.ByEmail("admin@example.com")
	if err != nil {
		t.Fatal(err)
	}
	adminToken, _, err := identity.NewSession(admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	userCookie := &http.Cookie{Name: SessionCookie, Value: userToken}
	adminCookie := &http.Cookie{Name: SessionCookie, Value: adminToken}
	h := webHandler("3430", "page-key-at-least-sixteen", time.Hour, nil, Handler(nil, nil))

	for _, path := range []string{"/portal/console", "/portal/keys", "/portal/profile", "/portal/models"} {
		rec := portalRequest(h, path)
		if rec.Code != http.StatusFound {
			t.Fatalf("%s without a session: %d %s", path, rec.Code, rec.Body)
		}
		want := "/portal/login?next=" + url.QueryEscape(path)
		if loc := rec.Header().Get("Location"); loc != want {
			t.Errorf("%s redirects to %q, want %q", path, loc, want)
		}
		if rec := portalRequest(h, path, userCookie); rec.Code != http.StatusOK {
			t.Fatalf("%s with a user: %d %s", path, rec.Code, rec.Body)
		}
		if rec := portalRequest(h, path, adminCookie); rec.Code != http.StatusOK {
			t.Fatalf("%s with an admin: %d %s", path, rec.Code, rec.Body)
		}
	}
	// a signed-in account doesn't see the sign-in forms again
	for _, path := range []string{"/portal/login", "/portal/register"} {
		if rec := portalRequest(h, path, userCookie); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/portal/console" {
			t.Errorf("%s signed in: %d %q", path, rec.Code, rec.Header().Get("Location"))
		}
	}
	// and a user session is not the admin page
	if rec := portalRequest(h, "/", userCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("a user session opened the admin page: %d %s", rec.Code, rec.Body)
	}
}

func TestPortalServesNoPathOutsideItsFiles(t *testing.T) {
	sandboxHome(t)
	h := Handler(nil, nil)
	for _, path := range []string{
		"/portal/%2e%2e/index.html",
		"/portal/..%2findex.html",
		"/portal/%2e%2e%2findex.html",
	} {
		rec := portalRequest(h, path)
		if rec.Code == http.StatusOK {
			t.Errorf("%s was served: %s", path, rec.Body)
		}
	}
}
