package media

import "testing"

func TestValidMediaNameAvatars(t *testing.T) {
	for _, name := range []string{
		"avt_01h2x3y4z5a6b7c8d9e0f1g2h3.webp",
		"avt_01h2x3y4z5a6b7c8d9e0f1g2h3_sm.webp",
		"med_01h2x3y4z5a6b7c8d9e0f1g2h3_thumb.webp",
	} {
		if !ValidMediaName(name) {
			t.Fatalf("expected valid: %s", name)
		}
	}
}