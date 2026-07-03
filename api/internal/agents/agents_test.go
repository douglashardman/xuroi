package agents

import "testing"

func TestOwnerLabel(t *testing.T) {
	if got := OwnerLabel("MrDoug"); got != "MrDoug's Agent" {
		t.Fatalf("got %q", got)
	}
	if got := OwnerLabel(""); got != "Member's Agent" {
		t.Fatalf("empty: got %q", got)
	}
}