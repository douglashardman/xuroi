package access

import "testing"

func TestCanView(t *testing.T) {
	guest := Viewer{IsGuest: true, Entitlements: map[string]bool{}}
	member := Viewer{IsMember: true, Entitlements: map[string]bool{}}
	staff := Viewer{IsMember: true, IsStaff: true, Entitlements: map[string]bool{}}
	admin := Viewer{IsMember: true, IsAdmin: true, Entitlements: map[string]bool{}}
	supporter := Viewer{IsMember: true, Entitlements: map[string]bool{EntSupporter: true}}

	if !guest.CanView(LevelPublic) || guest.CanView(LevelMembers) {
		t.Fatal("guest public/members")
	}
	if !member.CanView(LevelMembers) || member.CanView(LevelStaff) {
		t.Fatal("member levels")
	}
	if !staff.CanView(LevelStaff) || !staff.CanView(LevelSupporters) {
		t.Fatal("staff levels")
	}
	if !admin.CanView(LevelAdmin) || !admin.CanView(LevelSponsors) {
		t.Fatal("admin levels")
	}
	if !supporter.CanView(LevelSupporters) || supporter.CanView(LevelSponsors) {
		t.Fatal("supporter levels")
	}
}

func TestDefaultListPublic(t *testing.T) {
	if !DefaultListPublic(LevelSupporters) || DefaultListPublic(LevelStaff) {
		t.Fatal("default list public")
	}
}

func TestCanViewAny(t *testing.T) {
	supporter := Viewer{IsMember: true, Entitlements: map[string]bool{EntSupporter: true}}
	sponsor := Viewer{IsMember: true, Entitlements: map[string]bool{EntSponsor: true}}
	levels := []string{LevelSupporters, LevelSponsors}
	if !supporter.CanViewAny(levels) || !sponsor.CanViewAny(levels) {
		t.Fatal("either group should grant access")
	}
	member := Viewer{IsMember: true, Entitlements: map[string]bool{}}
	if member.CanViewAny(levels) {
		t.Fatal("plain member should not pass multi-group gate")
	}
}