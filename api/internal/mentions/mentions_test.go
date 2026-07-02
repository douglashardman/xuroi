package mentions

import (
	"testing"
)

func testIndex() *Index {
	return &Index{
		bySlug: map[string]Actor{
			"doug": {"act_doug", "Doug"},
			"pt-guy": {"act_pt", "PT Guy"},
		},
		byName: map[string]Actor{
			"doug":     {"act_doug", "Doug"},
			"pt guy":   {"act_pt", "PT Guy"},
			"puttertalk": {"act_pt2", "PutterTalk"},
		},
	}
}

func TestExpandQuotedAndSlug(t *testing.T) {
	idx := testIndex()
	got := Expand(`Hey @doug and @"PT Guy" — ping @[Doug]`, idx)
	want := `Hey [@Doug](/u/doug) and [@PT Guy](/u/pt-guy) — ping [@Doug](/u/doug)`
	if got.Markdown != want {
		t.Fatalf("markdown:\n got: %q\nwant: %q", got.Markdown, want)
	}
	if len(got.ActorIDs) != 2 {
		t.Fatalf("actor ids: %v", got.ActorIDs)
	}
}

func TestExpandSkipsUnknown(t *testing.T) {
	idx := testIndex()
	got := Expand(`@nobody @doug`, idx)
	want := `@nobody [@Doug](/u/doug)`
	if got.Markdown != want {
		t.Fatalf("got %q want %q", got.Markdown, want)
	}
}

func TestExpandSkipsEmailLike(t *testing.T) {
	idx := testIndex()
	got := Expand(`email me at user@doug.com`, idx)
	if got.Markdown != `email me at user@doug.com` {
		t.Fatalf("got %q", got.Markdown)
	}
}

func TestFilterSelf(t *testing.T) {
	got := FilterSelf([]string{"a", "b", "a"}, "a")
	if len(got) != 1 || got[0] != "b" {
		t.Fatalf("got %v", got)
	}
}