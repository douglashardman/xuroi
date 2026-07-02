package models

import "strings"

func ThreadDisplayTitle(prefix, title string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return title
	}
	return "[" + prefix + "] " + title
}