// The user-facing portal, served at /portal/ beside the admin page at /.
// It is a page of its own, not index.html or index-web.html: its styles and
// script are its own files, it loads none of the admin page's, and it is
// served without the admin page's key. The account it shows is the one
// /api/auth signs in; console and models are served only to a signed-in
// account, and every admin right is still checked by the admin API.
package gui

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/settings"
)

// portalFiles are the portal's static files and their types. Only these
// names are served: the path is never joined into a file path.
var portalFiles = map[string]string{
	"portal.css":      "text/css; charset=utf-8",
	"portal.js":       "text/javascript; charset=utf-8",
	"usage.js":        "text/javascript; charset=utf-8",
	"models.js":       "text/javascript; charset=utf-8",
	"admin-models.js": "text/javascript; charset=utf-8",
	"i18n.js":         "text/javascript; charset=utf-8",
	"bird.svg":        "image/svg+xml; charset=utf-8",
	"deepseek.svg":    "image/svg+xml; charset=utf-8",
}

// portalPage is one of the portal's own pages.
var portalPages = map[string]bool{
	"home": true, "login": true, "register": true,
	"console": true, "keys": true, "profile": true,
	"models": true, "admin-models": true,
}

// portalCSP keeps the portal to its own files: no inline script or style,
// no frames, no other origin. The account cookie stays same-origin.
const portalCSP = "default-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'"

func portalRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /portal", func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw, r, "/portal/", http.StatusFound)
	})
	mux.HandleFunc("GET /portal/", portalServed)
}

// portalServed serves the portal's files and pages. The page is told to the
// script in the body's data-page, so what it draws is the page asked for;
// console, keys, profile and models are redirected to sign-in when no account
// is signed in, and login and register to the console when one is.
func portalServed(rw http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/portal/")
	// the saved accent and its surface tints, as the management page draws
	// them; the portal's CSP allows its own files only, so this is a file
	// rather than an inline style, and it is never cached (a colour changed
	// in Settings shows on the next load)
	if name == "look.css" {
		rw.Header().Set("Content-Type", "text/css; charset=utf-8")
		rw.Header().Set("Cache-Control", "no-store")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = rw.Write([]byte(portalLookCSS(settings.Load())))
		return
	}
	if ct, ok := portalFiles[name]; ok {
		b, err := fs.ReadFile(staticFS(), "portal/"+name)
		if err != nil {
			http.NotFound(rw, r)
			return
		}
		rw.Header().Set("Content-Type", ct)
		rw.Header().Set("Cache-Control", "no-cache")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(rw, r, name, time.Time{}, bytes.NewReader(b))
		return
	}
	page := strings.TrimSuffix(name, "/")
	if page == "" {
		page = "home"
	}
	if !portalPages[page] {
		http.NotFound(rw, r)
		return
	}
	_, signedIn := sessionUser(r)
	switch page {
	case "login", "register":
		if signedIn {
			http.Redirect(rw, r, "/portal/console", http.StatusFound)
			return
		}
	case "console", "keys", "profile", "models", "admin-models":
		if !signedIn {
			next := r.URL.RequestURI()
			if next == "" || !strings.HasPrefix(next, "/portal/") {
				next = "/portal/" + page
			}
			http.Redirect(rw, r, "/portal/login?next="+url.QueryEscape(next), http.StatusFound)
			return
		}
		if page == "admin-models" {
			if u, _ := sessionUser(r); u.Role != identity.RoleAdmin {
				http.Error(rw, "this page is an administrator's", http.StatusForbidden)
				return
			}
		}
	}
	b, err := fs.ReadFile(staticFS(), "portal/index.html")
	if err != nil {
		http.Error(rw, "the portal's page isn't in this build", http.StatusInternalServerError)
		return
	}
	html := strings.Replace(string(b), `data-page="__PAGE__"`, `data-page="`+page+`"`, 1)
	html = portalPrefs(html, r, settings.Load())
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Content-Security-Policy", portalCSP)
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.Header().Set("X-Frame-Options", "DENY")
	rw.Header().Set("Referrer-Policy", "no-referrer")
	_, _ = rw.Write([]byte(html))
}

// publicPage is served without magpie web's page key: the user portal and
// its files. Console, models and the admin's model page check the account
// themselves; the admin page also checks the role.
func publicPage(path string) bool {
	return path == "/portal" || strings.HasPrefix(path, "/portal/")
}

// portalPrefs writes the management page's look and language on the portal's
// page: the theme, the colour-blind set and Reduce motion as attributes, the
// language on <html>, and the look's stylesheet as a same-origin link (the
// portal's CSP allows no inline style). The portal keeps no preferences of
// its own — every page reads these from settings, so the two surfaces cannot
// disagree, and a colour changed in Settings shows on the next load.
func portalPrefs(html string, r *http.Request, s settings.Settings) string {
	lang := portalLang(r, s)
	tag := "zh-CN"
	if lang == "en" {
		tag = "en"
	}
	attrs := `lang="` + tag + `"`
	if s.Theme == "light" || s.Theme == "dark" {
		attrs += ` data-theme="` + s.Theme + `"`
	}
	if s.Colorblind {
		attrs += ` data-colorblind="1"`
	}
	if s.ReduceMotion {
		attrs += ` data-motion="reduce"`
	}
	html = strings.Replace(html, `<html lang="zh-CN">`, `<html `+attrs+`>`, 1)
	html = strings.Replace(html,
		`<link rel="stylesheet" href="/portal/portal.css">`,
		`<link rel="stylesheet" href="/portal/portal.css">`+"\n"+`<link rel="stylesheet" href="/portal/look.css">`, 1)
	// the phone's chrome follows the theme the page shows
	if s.Theme == "light" || s.Theme == "dark" {
		html = strings.Replace(html, `<meta name="theme-color" content="#0a0a0a">`,
			`<meta name="theme-color" content="`+themeColor(s.Theme)+`">`, 1)
	} else {
		html = strings.Replace(html, `<meta name="theme-color" content="#0a0a0a">`,
			`<meta name="theme-color" content="#f7f7f7" media="(prefers-color-scheme: light)">`+"\n"+
				`<meta name="theme-color" content="#0a0a0a" media="(prefers-color-scheme: dark)">`, 1)
	}
	return html
}

// portalLang is the language the portal speaks: ?lang= is a temporary
// override (never saved), then the cookie the footer's switch sets, then the
// saved setting — where "system" reads as Chinese, the portal's own default.
// It never writes the management page's setting.
func portalLang(r *http.Request, s settings.Settings) string {
	if q := r.URL.Query().Get("lang"); q == "en" || q == "zh" {
		return q
	}
	if c, err := r.Cookie("minirelay_lang"); err == nil && (c.Value == "en" || c.Value == "zh") {
		return c.Value
	}
	if strings.HasPrefix(s.Lang, "en") {
		return "en"
	}
	return "zh"
}

// themeColor is a theme's page ground, for the browser's own chrome.
func themeColor(theme string) string {
	if theme == "dark" {
		return "#0a0a0a"
	}
	return "#f7f7f7"
}
