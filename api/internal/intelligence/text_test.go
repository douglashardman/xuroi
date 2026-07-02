package intelligence

import "testing"

func TestStripHTMLDecodesEntities(t *testing.T) {
	got := StripHTML(`<p>I&#39;m testing it. We&#39;ll see what it looks like.</p>`)
	want := "I'm testing it. We'll see what it looks like."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizePlainText(t *testing.T) {
	got := NormalizePlainText(`I&#39;m just going to put this out there`)
	if got != "I'm just going to put this out there" {
		t.Fatalf("got %q", got)
	}
}