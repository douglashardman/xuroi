package site

import "strings"

var ClassifiedsForumSlugs = map[string]struct{}{
	"free-classifieds": {},
	"wanted-trade":     {},
	"ebay-items":       {},
}

var ClassifiedsPrefixes = []string{"FS", "WTT", "WTB", "SOLD"}

func IsClassifiedsForum(slug string) bool {
	_, ok := ClassifiedsForumSlugs[strings.TrimSpace(slug)]
	return ok
}

func ValidClassifiedsPrefix(prefix string) bool {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	if prefix == "" {
		return true
	}
	for _, p := range ClassifiedsPrefixes {
		if prefix == p {
			return true
		}
	}
	return false
}