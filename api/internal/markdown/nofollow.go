package markdown

import (
	"regexp"
	"strings"
)

var externalLinkRe = regexp.MustCompile(`(?i)<a\s+([^>]*href="https?://[^"]*"[^>]*)>`)

// ApplyNofollow adds rel="nofollow ugc" to external links in post HTML.
func ApplyNofollow(html string) string {
	if html == "" || !strings.Contains(html, "<a ") {
		return html
	}
	return externalLinkRe.ReplaceAllStringFunc(html, func(tag string) string {
		if strings.Contains(strings.ToLower(tag), `rel="`) {
			return tag
		}
		return strings.Replace(tag, "<a ", `<a rel="nofollow ugc" `, 1)
	})
}