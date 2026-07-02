package access

import "slices"

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