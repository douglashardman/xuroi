package access

import (
	"slices"
	"strings"
)

const (
	LevelPublic     = "public"
	LevelMembers    = "members"
	LevelStaff      = "staff"
	LevelAdmin      = "admin"
	LevelSupporters = "supporters"
	LevelSponsors   = "sponsors"
)

const (
	EntSupporter = "supporter"
	EntSponsor   = "sponsor"
)

var Levels = []string{
	LevelPublic,
	LevelMembers,
	LevelStaff,
	LevelAdmin,
	LevelSupporters,
	LevelSponsors,
}

var Entitlements = []string{EntSupporter, EntSponsor}

type LevelInfo struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

func LevelCatalog() []LevelInfo {
	return []LevelInfo{
		{ID: LevelPublic, Label: "Public", Description: "Anyone can view and post (guests read-only per site policy)."},
		{ID: LevelMembers, Label: "Members", Description: "Signed-in members with verified email."},
		{ID: LevelStaff, Label: "Staff", Description: "Moderators and admins only."},
		{ID: LevelAdmin, Label: "Admin", Description: "Site administrators only."},
		{ID: LevelSupporters, Label: "Supporters", Description: "Members with supporter entitlement (manual or Patreon later)."},
		{ID: LevelSponsors, Label: "Sponsors", Description: "Members with sponsor entitlement (manual or Stripe later)."},
	}
}

func EntitlementCatalog() []LevelInfo {
	return []LevelInfo{
		{ID: EntSupporter, Label: "Supporter", Description: "Access to supporter-only forums. Grant manually until Patreon sync ships."},
		{ID: EntSponsor, Label: "Sponsor", Description: "Access to sponsor-only forums. Grant manually until Stripe sync ships."},
	}
}

func ValidLevel(level string) bool {
	return slices.Contains(Levels, level)
}

func ValidEntitlement(ent string) bool {
	return slices.Contains(Entitlements, ent)
}

func NormalizeLevel(level string) string {
	if ValidLevel(level) {
		return level
	}
	return LevelPublic
}

// NormalizeLevels deduplicates and validates access groups. Empty → public only.
func NormalizeLevels(levels []string) []string {
	if len(levels) == 0 {
		return []string{LevelPublic}
	}
	seen := make(map[string]struct{}, len(levels))
	out := make([]string, 0, len(levels))
	for _, level := range levels {
		n := NormalizeLevel(level)
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	if len(out) == 1 && out[0] == LevelPublic {
		return out
	}
	// Public is redundant when other restrictions exist.
	filtered := out[:0]
	for _, n := range out {
		if n != LevelPublic {
			filtered = append(filtered, n)
		}
	}
	if len(filtered) == 0 {
		return []string{LevelPublic}
	}
	return filtered
}

// PrimaryLevel is the stored access_level column (most restrictive assigned group).
func PrimaryLevel(levels []string) string {
	normalized := NormalizeLevels(levels)
	if len(normalized) == 0 {
		return LevelPublic
	}
	best := normalized[0]
	bestRank := levelRestrictiveness(best)
	for _, level := range normalized[1:] {
		if r := levelRestrictiveness(level); r > bestRank {
			best = level
			bestRank = r
		}
	}
	return best
}

func levelRestrictiveness(level string) int {
	switch NormalizeLevel(level) {
	case LevelPublic:
		return 0
	case LevelMembers:
		return 1
	case LevelSupporters, LevelSponsors:
		return 2
	case LevelStaff:
		return 3
	case LevelAdmin:
		return 4
	default:
		return 0
	}
}

func (v Viewer) CanViewAny(levels []string) bool {
	for _, level := range NormalizeLevels(levels) {
		if v.CanView(level) {
			return true
		}
	}
	return false
}

func (v Viewer) CanPostAny(levels []string) bool {
	for _, level := range NormalizeLevels(levels) {
		if v.CanPost(level) {
			return true
		}
	}
	return false
}

func LockedLabels(levels []string) string {
	normalized := NormalizeLevels(levels)
	if len(normalized) == 1 && normalized[0] == LevelPublic {
		return ""
	}
	labels := make([]string, 0, len(normalized))
	for _, level := range normalized {
		if label := LockedLabel(level); label != "" {
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return ""
	}
	if len(labels) == 1 {
		return labels[0]
	}
	return strings.Join(labels, " or ")
}

func DefaultListPublicAny(levels []string) bool {
	for _, level := range NormalizeLevels(levels) {
		if !DefaultListPublic(level) {
			return false
		}
	}
	return true
}

// ResolveListPublicAny picks list_public from explicit override or group defaults.
func ResolveListPublicAny(levels []string, explicit *bool) bool {
	if explicit != nil {
		return *explicit
	}
	return DefaultListPublicAny(levels)
}

// ResolveCategoryAccess normalizes access_levels from payload (array preferred, legacy string fallback).
func ResolveCategoryAccess(level string, levels []string) ([]string, string) {
	if len(levels) > 0 {
		normalized := NormalizeLevels(levels)
		return normalized, PrimaryLevel(normalized)
	}
	n := NormalizeLevel(level)
	return []string{n}, n
}

type Viewer struct {
	ActorID      *string
	IsGuest      bool
	IsMember     bool
	IsStaff      bool
	IsAdmin      bool
	Entitlements map[string]bool
}

func (v Viewer) HasEntitlement(ent string) bool {
	return v.Entitlements[ent]
}

func (v Viewer) CanView(level string) bool {
	switch NormalizeLevel(level) {
	case LevelPublic:
		return true
	case LevelMembers:
		return v.IsMember || v.IsStaff || v.IsAdmin
	case LevelStaff:
		return v.IsStaff || v.IsAdmin
	case LevelAdmin:
		return v.IsAdmin
	case LevelSupporters:
		return v.HasEntitlement(EntSupporter) || v.IsStaff || v.IsAdmin
	case LevelSponsors:
		return v.HasEntitlement(EntSponsor) || v.IsAdmin
	default:
		return true
	}
}

func (v Viewer) CanPost(level string) bool {
	if !v.CanView(level) {
		return false
	}
	if v.IsGuest {
		return false
	}
	switch NormalizeLevel(level) {
	case LevelPublic:
		return v.IsMember || v.IsStaff || v.IsAdmin
	case LevelMembers:
		return v.IsMember || v.IsStaff || v.IsAdmin
	case LevelStaff:
		return v.IsStaff || v.IsAdmin
	case LevelAdmin:
		return v.IsAdmin
	case LevelSupporters:
		return v.HasEntitlement(EntSupporter) || v.IsStaff || v.IsAdmin
	case LevelSponsors:
		return v.HasEntitlement(EntSponsor) || v.IsAdmin
	default:
		return v.IsMember
	}
}

func DefaultListPublic(level string) bool {
	switch NormalizeLevel(level) {
	case LevelStaff, LevelAdmin:
		return false
	default:
		return true
	}
}

func ResolveListPublic(level string, explicit *bool) bool {
	if explicit != nil {
		return *explicit
	}
	return DefaultListPublic(level)
}

func LockedLabel(level string) string {
	switch NormalizeLevel(level) {
	case LevelMembers:
		return "Members only"
	case LevelStaff:
		return "Staff only"
	case LevelAdmin:
		return "Admins only"
	case LevelSupporters:
		return "Supporters only"
	case LevelSponsors:
		return "Sponsors only"
	default:
		return ""
	}
}