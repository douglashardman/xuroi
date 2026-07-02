package intelligence

import (
	"strings"
	"testing"
)

func TestBuildSummary(t *testing.T) {
	s := buildSummary(threadInput{
		Title:        "Odyssey vs Scotty",
		ReplyCount:   3,
		Participants: 2,
		OPBodyHTML:   "<p>Which putter for <b>fast greens</b>?</p>",
	})
	if s == "" {
		t.Fatal("expected summary")
	}
	if !strings.Contains(s, "Odyssey vs Scotty") {
		t.Fatalf("missing title: %q", s)
	}
	if !strings.Contains(s, "fast greens") {
		t.Fatalf("missing OP excerpt: %q", s)
	}
}