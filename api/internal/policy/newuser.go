package policy

import (
	"regexp"
	"time"
)

type NewUserPolicy struct {
	RestrictLinksHours int `json:"restrict_links_hours"`
	RestrictDMHours    int `json:"restrict_dm_hours"`
}

func (p NewUserPolicy) Normalized() NewUserPolicy {
	out := p
	if out.RestrictLinksHours <= 0 {
		out.RestrictLinksHours = 24
	}
	if out.RestrictDMHours <= 0 {
		out.RestrictDMHours = 72
	}
	return out
}

var linkPattern = regexp.MustCompile(`(?i)(https?://|www\.|\[[^\]]+\]\([^)]+\))`)

func (p NewUserPolicy) LinksRestricted(accountAge time.Duration) bool {
	return accountAge < time.Duration(p.Normalized().RestrictLinksHours)*time.Hour
}

func (p NewUserPolicy) DMRestricted(accountAge time.Duration) bool {
	return accountAge < time.Duration(p.Normalized().RestrictDMHours)*time.Hour
}

func ContainsLink(markdown string) bool {
	return linkPattern.MatchString(markdown)
}

func LinkRestrictionMessage(hours int) string {
	if hours <= 24 {
		return "New members cannot post links for the first 24 hours."
	}
	return "New members cannot post links yet — try again after your account ages in."
}

func DMRestrictionMessage(hours int) string {
	return "New members cannot send direct messages for the first few days."
}

func IsStaffOrAdmin(isStaff, isAdmin bool) bool {
	return isStaff || isAdmin
}

func ShouldAllowLinks(p NewUserPolicy, accountAge time.Duration, markdown string, isStaff, isAdmin bool) error {
	if IsStaffOrAdmin(isStaff, isAdmin) {
		return nil
	}
	if !p.LinksRestricted(accountAge) {
		return nil
	}
	if ContainsLink(markdown) {
		return ErrLinksRestricted
	}
	return nil
}

type policyError string

func (e policyError) Error() string { return string(e) }

var ErrLinksRestricted = policyError(LinkRestrictionMessage(24))