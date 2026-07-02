package markdown

import (
	"regexp"
	"strings"
)

func ApplyWordFilter(html string, words []string) string {
	if html == "" || len(words) == 0 {
		return html
	}
	out := html
	for _, word := range words {
		w := strings.TrimSpace(word)
		if w == "" {
			continue
		}
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `\b`)
		stars := strings.Repeat("*", len([]rune(w)))
		out = re.ReplaceAllString(out, stars)
	}
	return out
}