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

func TestBuildSummaryDecodesEntities(t *testing.T) {
	s := buildSummary(threadInput{
		Title:        "This is the first new thread.",
		ReplyCount:   4,
		Participants: 2,
		OPBodyHTML:   `<p>This is a thread. I&#39;m testing it. We&#39;ll see what it looks like.</p>`,
	})
	if strings.Contains(s, "&#39;") {
		t.Fatalf("summary still has HTML entities: %q", s)
	}
	if !strings.Contains(s, "I'm testing it") {
		t.Fatalf("missing decoded apostrophe: %q", s)
	}
}