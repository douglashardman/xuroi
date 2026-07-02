package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	md = goldmark.New(
		goldmark.WithExtensions(extension.Linkify, extension.Strikethrough),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)
	policy       = bluemonday.UGCPolicy()
	spaceRe      = regexp.MustCompile(`\s+`)
	imgTagRe     = regexp.MustCompile(`<img([^>]*)>`)
	mediaSrcRe   = regexp.MustCompile(`src="(/api/media/med_[0-9a-z]{26})\.webp"`)
	imgOnlyParaRe = regexp.MustCompile(`<p>(<img[^>]*>)</p>`)
)

func init() {
	policy.AllowRelativeURLs(true)
	policy.AllowURLSchemes("http", "https")
	policy.AllowElements("details", "summary")
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^spoiler$`)).OnElements("details")
}

// ToHTML renders markdown to sanitized HTML safe for public display.
func ToHTML(source string) string {
	return RenderUGC(source, nil)
}

// RenderUGC renders member markdown with optional word-filter list.
func RenderUGC(source string, wordFilter []string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	source = expandSpoilers(source)

	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		html := policy.Sanitize("<p>" + escapeFallback(source) + "</p>")
		return ApplyWordFilter(html, wordFilter)
	}
	html := EnrichMediaImages(policy.Sanitize(buf.String()))
	return ApplyWordFilter(html, wordFilter)
}

// EnrichMediaImages rewrites inline post images to thumbnails with a full-res lightbox source.
func EnrichMediaImages(html string) string {
	if html == "" || !strings.Contains(html, "<img") {
		return html
	}
	html = imgTagRe.ReplaceAllStringFunc(html, func(tag string) string {
		if strings.Contains(tag, "data-full-src") {
			return tag
		}
		if !mediaSrcRe.MatchString(tag) {
			return tag
		}
		return mediaSrcRe.ReplaceAllString(tag, `src="${1}_thumb.webp" data-full-src="${1}.webp"`)
	})
	return wrapImageGalleries(html)
}

// wrapImageGalleries groups consecutive image paragraphs into a grid gallery.
func wrapImageGalleries(html string) string {
	if !strings.Contains(html, "<img") {
		return html
	}

	var out strings.Builder
	rest := html
	for {
		loc := imgOnlyParaRe.FindStringIndex(rest)
		if loc == nil {
			out.WriteString(rest)
			break
		}
		out.WriteString(rest[:loc[0]])

		segment := rest[loc[0]:]
		var imgs []string
		for {
			m := imgOnlyParaRe.FindStringSubmatchIndex(segment)
			if m == nil || m[0] != 0 {
				break
			}
			imgs = append(imgs, segment[m[2]:m[3]])
			segment = strings.TrimLeft(segment[m[1]:], " \t\n\r")
		}

		switch len(imgs) {
		case 0:
		case 1:
			out.WriteString("<p>")
			out.WriteString(imgs[0])
			out.WriteString("</p>")
		default:
			out.WriteString(`<div class="post-gallery">`)
			for _, img := range imgs {
				out.WriteString(img)
			}
			out.WriteString(`</div>`)
		}
		rest = segment
	}
	return out.String()
}

// IsExcerptOf reports whether quote text appears in source (whitespace-normalized).
// Used to allow trimming quotes without inventing new content.
func IsExcerptOf(quote, source string) bool {
	q := normalizeText(quote)
	if q == "" {
		return true
	}
	return strings.Contains(normalizeText(source), q)
}

func normalizeText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return spaceRe.ReplaceAllString(s, " ")
}

func escapeFallback(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}