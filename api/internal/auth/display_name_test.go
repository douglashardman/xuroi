package auth

import "testing"

func TestDisplayNameSlug(t *testing.T) {
	if got := displayNameSlug("Gear Tester"); got != "gear-tester" {
		t.Fatalf("got %q", got)
	}
	if got := displayNameSlug("  John_Doe  "); got != "john-doe" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateDisplayName(t *testing.T) {
	if err := validateDisplayName("a"); err == nil {
		t.Fatal("expected too short")
	}
	if err := validateDisplayName("---"); err == nil {
		t.Fatal("expected invalid slug")
	}
	if err := validateDisplayName("Doug"); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
}