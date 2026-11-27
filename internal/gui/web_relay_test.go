package gui

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/pluginrt"
)

// The Web relay's page is index-web.html: no Agents view, and none of the
// Sessions or Library scripts. The app's window keeps them, and its own
// page served to a browser too.
func TestWebPageLeavesAgentPagesOut(t *testing.T) {
	web, err := fs.ReadFile(webStaticFS(), "index.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(web)
	for _, gone := range []string{
		`id="view-agents"`,
		`id="agents"`,
		`data-view="agents"`,
		`data-view="sessions"`,
		`data-view="library"`,
		`data-ptab="agents"`,
		"plugins/sessions/client.js",
		"plugins/library/client.js",
	} {
		if strings.Contains(page, gone) {
			t.Errorf("the Web page still has %s", gone)
		}
	}
	full, err := fs.ReadFile(staticFS(), "index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, kept := range []string{
		`id="view-agents"`,
		`data-view="agents"`,
		`data-view="sessions"`,
		`data-view="library"`,
		"plugins/sessions/client.js",
		"plugins/library/client.js",
	} {
		if !strings.Contains(string(full), kept) {
			t.Errorf("the app's page lost %s", kept)
		}
	}
	// the page told which product it is: the relay's page alone leaves the
	// three out, the app's page keeps them
	boot := func(h http.Handler) string {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/boot.js", nil))
		return rec.Body.String()
	}
	if b := boot(webPageHandler(t, webHost{}, nil)); !strings.Contains(b, `"relay":true`) {
		t.Errorf("magpie web's boot.js doesn't say relay: %s", b)
	}
	if b := boot(Handler(nil, nil)); !strings.Contains(b, `"relay":false`) {
		t.Errorf("the app's boot.js says relay: %s", b)
	}
}

// The Web relay's Plugins page is not told the Sessions or Library
// built-ins, and cannot switch or configure them; the app's own window is,
// and can.
func TestWebBuiltinsLeaveRemovedPluginsOut(t *testing.T) {
	builtinHome(t)
	if _, err := pluginrt.Settle(); err != nil {
		t.Fatal(err)
	}
	list := func(h http.Handler) []builtinJSON {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/plugins/builtin", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/plugins/builtin: %d %s", rec.Code, rec.Body)
		}
		var out []builtinJSON
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	has := func(list []builtinJSON, id string) bool {
		for _, b := range list {
			if b.ID == id {
				return true
			}
		}
		return false
	}
	// /api/settings names the built-ins the page draws too (its feature
	// switches): the relay's answer leaves the two out as well
	settingsBuiltins := func(h http.Handler) []builtinJSON {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/settings", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/settings: %d %s", rec.Code, rec.Body)
		}
		var s struct{ Builtins []builtinJSON }
		if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
			t.Fatal(err)
		}
		return s.Builtins
	}
	desktopH := Handler(nil, nil)
	desktop := list(desktopH)
	for _, id := range []string{"sessions", "library"} {
		if !has(desktop, id) {
			t.Errorf("the app's Plugins page lost %s", id)
		}
		if !has(settingsBuiltins(desktopH), id) {
			t.Errorf("the app's /api/settings lost %s", id)
		}
	}
	webH := webPageHandler(t, webHost{}, nil)
	web := list(webH)
	for _, id := range []string{"sessions", "library"} {
		if has(web, id) {
			t.Errorf("the Web Plugins page still offers %s", id)
		}
		if has(settingsBuiltins(webH), id) {
			t.Errorf("the Web /api/settings still names %s", id)
		}
	}
	for _, id := range []string{"gateway", "routing", "usage", "middleware"} {
		if !has(web, id) {
			t.Errorf("the Web Plugins page lost %s", id)
		}
	}
	// and the switch is not a hidden way in
	rec := httptest.NewRecorder()
	webPageHandler(t, webHost{}, nil).ServeHTTP(rec, httptest.NewRequest("POST", "/api/plugins/builtin", strings.NewReader(`{"id":"sessions","on":false}`)))
	if rec.Code == http.StatusOK {
		t.Errorf("the Web page switched sessions: %s", rec.Body)
	}
}
