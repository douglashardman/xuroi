package slug

import (
	"regexp"
	"strings"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// FromDisplayName matches web userURL slug generation.
func FromDisplayName(displayName string) string {
	s := strings.ToLower(strings.TrimSpace(displayName))
	s = nonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}