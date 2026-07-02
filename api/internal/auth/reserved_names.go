package auth

import (
	"errors"
	"strings"
)

// ErrDisplayNameReserved is returned when a name is blocked to prevent impersonation.
var ErrDisplayNameReserved = errors.New("display name reserved")

// DefaultReservedDisplayNames blocks staff/site impersonation at registration.
func DefaultReservedDisplayNames() []string {
	return []string{
		"admin",
		"administrator",
		"moderator",
		"mod",
		"staff",
		"support",
		"helpdesk",
		"help",
		"system",
		"official",
		"webmaster",
		"security",
		"puttertalk",
		"xuroi",
		"community",
		"team",
	}
}

func BuildReservedSet(names []string) map[string]struct{} {
	set := make(map[string]struct{})
	seen := make(map[string]struct{})
	for _, raw := range append(DefaultReservedDisplayNames(), names...) {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		set[key] = struct{}{}
		if slug := displayNameSlug(raw); slug != "" {
			set[slug] = struct{}{}
		}
	}
	return set
}

func reservedDisplayName(name string, reserved map[string]struct{}) bool {
	if len(reserved) == 0 {
		return false
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}
	if _, ok := reserved[strings.ToLower(trimmed)]; ok {
		return true
	}
	if _, ok := reserved[displayNameSlug(trimmed)]; ok {
		return true
	}
	return false
}