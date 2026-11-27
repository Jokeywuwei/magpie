package gui

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestI18nLanguagesAgree keeps the four dictionaries level: every key one
// language has, every language has, no entry is empty, and a key appears
// once in its block. A word added to one language and forgotten in another
// reads as English for that reader, which is how "User portal" and a run of
// Backup fields had gone missing before this test existed.
func TestI18nLanguagesAgree(t *testing.T) {
	src, err := os.ReadFile("assets/i18n.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	pairs := regexp.MustCompile(`(?ms)^\s{4}"((?:[^"\\]|\\.)*)"\s*:\s*"((?:[^"\\]|\\.)*)"`)
	keysOnly := regexp.MustCompile(`(?m)^\s{4}"((?:[^"\\]|\\.)*)"\s*:`)

	langs := []string{"zh", `"zh-TW"`, "ja", "de"}
	keySets := map[string]map[string]bool{}
	for _, lang := range langs {
		block, ok := i18nBlock(text, lang)
		if !ok {
			t.Fatalf("no %s block in assets/i18n.js", lang)
		}
		seen := map[string]bool{}
		set := map[string]bool{}
		for _, m := range keysOnly.FindAllStringSubmatch(block, -1) {
			k := unquoteI18n(m[1])
			if seen[k] {
				t.Errorf("%s says %q twice", lang, k)
			}
			seen[k] = true
			set[k] = true
		}
		for _, m := range pairs.FindAllStringSubmatch(block, -1) {
			k, v := unquoteI18n(m[1]), unquoteI18n(m[2])
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s has an empty translation for %q", lang, k)
			}
			if _, ok := set[k]; !ok {
				// a key whose value grew past one line still counts
				set[k] = true
				seen[k] = true
			}
		}
		if len(set) == 0 {
			t.Fatalf("%s holds no entries", lang)
		}
		keySets[lang] = set
	}

	base := keySets["ja"]
	for _, lang := range langs {
		set := keySets[lang]
		for k := range base {
			if !set[k] {
				t.Errorf("%s lacks %q, which ja has", lang, k)
			}
		}
		for k := range set {
			if !base[k] {
				t.Errorf("%s has %q, which ja lacks", lang, k)
			}
		}
	}
}

// i18nBlock is one language's object in i18n.js.
func i18nBlock(src, lang string) (string, bool) {
	i := strings.Index(src, "\n  "+lang+": {\n")
	if i < 0 {
		return "", false
	}
	rest := src[i+len(lang)+5:]
	j := strings.Index(rest, "\n  },")
	if j < 0 {
		return "", false
	}
	return rest[:j], true
}

// unquoteI18n is a JS double-quoted string's body as its value.
func unquoteI18n(s string) string {
	if strings.Contains(s, `\`) {
		if v, err := strconv.Unquote(`"` + s + `"`); err == nil {
			return v
		}
	}
	return s
}