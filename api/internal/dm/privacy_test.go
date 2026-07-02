package dm

import "testing"

func TestNormalizePrivacy(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"everyone", PrivacyEveryone},
		{"friends_only", PrivacyFriendsOnly},
		{"off", PrivacyOff},
		{"", PrivacyEveryone},
		{"bogus", PrivacyEveryone},
	}
	for _, tc := range tests {
		if got := NormalizePrivacy(tc.in); got != tc.want {
			t.Errorf("NormalizePrivacy(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidPrivacy(t *testing.T) {
	if !ValidPrivacy(PrivacyEveryone) || !ValidPrivacy(PrivacyFriendsOnly) || !ValidPrivacy(PrivacyOff) {
		t.Fatal("expected known privacy values to be valid")
	}
	if ValidPrivacy("strangers") || ValidPrivacy("") {
		t.Fatal("expected unknown privacy values to be invalid")
	}
}