package intelligence

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)
var wsRe = regexp.MustCompile(`\s+`)

func StripHTML(html string) string {
	text := htmlTagRe.ReplaceAllString(html, " ")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	return strings.TrimSpace(wsRe.ReplaceAllString(text, " "))
}

func TruncatePlain(s string, max int) string {
	if max < 1 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return strings.TrimSpace(string(runes[:max-1])) + "…"
}