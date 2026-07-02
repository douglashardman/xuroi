package spam

import (
	"testing"
	"time"
)

func TestEvaluateBlocksSpammyNewUser(t *testing.T) {
	p := Policy{Enabled: true}.Normalized()
	r := Evaluate(p, Input{
		BodyMarkdown: "buy viagra at http://spam.com and http://more.com and http://third.com",
		AccountAge:   2 * time.Hour,
		RecentPosts:  6,
	})
	if !r.Block && !r.Hold {
		t.Fatalf("expected block or hold, got %+v", r)
	}
}