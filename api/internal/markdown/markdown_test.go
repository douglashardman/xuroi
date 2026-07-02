package markdown_test

import (
	"strings"
	"testing"

	"github.com/xuroi/xuroi/api/internal/markdown"
)

func TestToHTMLParagraphs(t *testing.T) {
	html := markdown.ToHTML("Hello\n\nWorld")
	if !strings.Contains(html, "Hello") || !strings.Contains(html, "World") {
		t.Fatalf("unexpected html: %s", html)
	}
}

func TestToHTMLStripsScript(t *testing.T) {
	html := markdown.ToHTML("<script>alert(1)</script>\n\nSafe text")
	if strings.Contains(strings.ToLower(html), "<script") {
		t.Fatalf("script should be stripped: %s", html)
	}
	if !strings.Contains(html, "Safe text") {
		t.Fatal("expected safe text")
	}
}

func TestToHTMLBoldAndLink(t *testing.T) {
	html := markdown.ToHTML("**bold** and https://puttertalk.com")
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Fatalf("expected bold: %s", html)
	}
	if !strings.Contains(html, `href="https://puttertalk.com"`) {
		t.Fatalf("expected link: %s", html)
	}
}

const testMediaID = "med_01kwgdk88wmb1a9k94r8gnebjx"

func TestToHTMLImage(t *testing.T) {
	html := markdown.ToHTML("![putter](/api/media/" + testMediaID + ".webp)")
	if !strings.Contains(html, `data-full-src="/api/media/`+testMediaID+`.webp"`) {
		t.Fatalf("expected full-src image: %s", html)
	}
	if !strings.Contains(html, `src="/api/media/`+testMediaID+`_thumb.webp"`) {
		t.Fatalf("expected thumb image: %s", html)
	}
}

func TestWrapImageGalleries(t *testing.T) {
	src := `<p>intro</p><p><img src="/api/media/` + testMediaID + `_thumb.webp" data-full-src="/api/media/` + testMediaID + `.webp" alt="a"></p>
<p><img src="/api/media/` + testMediaID + `_thumb.webp" data-full-src="/api/media/` + testMediaID + `.webp" alt="b"></p><p>tail</p>`
	html := markdown.EnrichMediaImages(src)
	if !strings.Contains(html, `class="post-gallery"`) {
		t.Fatalf("expected gallery wrapper: %s", html)
	}
	if strings.Count(html, "<img") != 2 {
		t.Fatalf("expected two images: %s", html)
	}
}

func TestEnrichMediaImagesIdempotent(t *testing.T) {
	raw := `<p><img src="/api/media/` + testMediaID + `.webp" alt="x"></p>`
	once := markdown.EnrichMediaImages(raw)
	if !strings.Contains(once, `data-full-src="/api/media/`+testMediaID+`.webp"`) {
		t.Fatalf("expected enrichment: %s", once)
	}
	twice := markdown.EnrichMediaImages(once)
	if twice != once {
		t.Fatalf("expected idempotent enrich, got %s", twice)
	}
}

func TestIsExcerptOf(t *testing.T) {
	source := "I switched to a LAB putter last season.\n\nBest decision ever."
	if !markdown.IsExcerptOf("LAB putter", source) {
		t.Fatal("partial excerpt should match")
	}
	if !markdown.IsExcerptOf("  lab   putter  ", source) {
		t.Fatal("whitespace-normalized excerpt should match")
	}
	if markdown.IsExcerptOf("fabricated quote text", source) {
		t.Fatal("invented text should not match")
	}
}