package gui

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestPortalI18nKeys keeps the portal's words and its dictionary together:
// every key the page or its scripts translate is in assets/portal/i18n.js,
// and a key the management page also translates reads the same there. A
// sentence changed in the page without its entry, or an entry drifting from
// the management page's, fails here rather than in a user's language.
func TestPortalI18nKeys(t *testing.T) {
	dict := portalDict(t, "assets/portal/i18n.js")

	keys := map[string]bool{}
	html, err := os.ReadFile("assets/portal/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`<[a-zA-Z][^>]*\bdata-t\b[^>]*>([^<]+)<`).FindAllStringSubmatch(string(html), -1) {
		keys[strings.TrimSpace(m[1])] = true
	}
	for _, m := range regexp.MustCompile(`data-(?:ta|tp)="([^"]+)"`).FindAllStringSubmatch(string(html), -1) {
		keys[m[1]] = true
	}
	// the scripts' t("…") calls: every literal inside the call (a conditional
	// picks one of two), as this page writes them
	call := regexp.MustCompile(`\bt\(([^)]*)\)`)
	literal := regexp.MustCompile(`"([^"]+)"`)
	for _, f := range []string{"portal.js", "usage.js", "models.js", "admin-models.js"} {
		b, err := os.ReadFile("assets/portal/" + f)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range call.FindAllStringSubmatch(string(b), -1) {
			for _, lit := range literal.FindAllStringSubmatch(c[1], -1) {
				keys[lit[1]] = true
			}
		}
	}

	for k := range keys {
		if _, ok := dict[k]; !ok {
			t.Errorf("portal key has no entry in assets/portal/i18n.js: %q", k)
		}
	}

	admin, err := os.ReadFile("assets/i18n.js")
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range dict {
		if zv, ok := adminZhValue(string(admin), k); ok && zv != v {
			t.Errorf("portal translation for %q differs from i18n.js:\n portal: %s\n admin:  %s", k, v, zv)
		}
	}
}

// portalDict reads i18n.js's zh object, key to translation.
func portalDict(t *testing.T, path string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, m := range regexp.MustCompile(`"([^"\n]+)"\s*:\s*"([^"]*)"`).FindAllStringSubmatch(string(b), -1) {
		out[unquoteJS(m[1])] = unquoteJS(m[2])
	}
	if len(out) == 0 {
		t.Fatalf("%s holds no entries", path)
	}
	return out
}

// adminZhValue is i18n.js's zh translation for key, when it has one.
func adminZhValue(src, key string) (string, bool) {
	i := strings.Index(src, `"`+key+`":`)
	if i < 0 {
		return "", false
	}
	rest := src[i+len(key)+3:]
	if j := strings.Index(rest, `"`); j >= 0 {
		rest = rest[j:]
	}
	m := regexp.MustCompile(`^"([^"]*)"`).FindStringSubmatch(rest)
	if m == nil {
		return "", false
	}
	return unquoteJS(m[1]), true
}

// unquoteJS turns a JS double-quoted string's body into its value, for the
// two escapes this dictionary uses.
func unquoteJS(s string) string {
	if strings.Contains(s, `\`) {
		if v, err := strconv.Unquote(`"` + s + `"`); err == nil {
			return v
		}
	}
	return s
}