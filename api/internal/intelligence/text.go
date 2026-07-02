package intelligence

import (
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)
var wsRe = regexp.MustCompile(`\s+`)

func StripHTML(htmlStr string) string {
	text := htmlTagRe.ReplaceAllString(htmlStr, " ")
	text = html.UnescapeString(text)
	return strings.TrimSpace(wsRe.ReplaceAllString(text, " "))
}

// NormalizePlainText decodes HTML entities left in stored plain text (e.g. legacy summaries).
func NormalizePlainText(s string) string {
	return strings.TrimSpace(html.UnescapeString(s))
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