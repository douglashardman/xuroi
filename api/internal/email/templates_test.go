package email

import (
	"strings"
	"testing"
)

func TestRenderThreadReply(t *testing.T) {
	subject, html, text, err := RenderThreadReply(ThreadReplyData{
		CommunityName: "PutterTalk Community",
		SiteURL:       "http://localhost:4321",
		Recipient:     "Doug",
		IntroLine:      "Simon chimed in on a thread you posted in.",
		ThreadTitle:    "Testing all of this out.",
		ThreadURL:      "http://localhost:4321/t/test",
		UnsubscribeURL: "http://localhost:4321/email/unsubscribe?token=test",
		ReplyCount:    1,
		Posts: []ThreadReplyPost{
			{Author: "Simon", Excerpt: "Clean single-reply notification test.", When: "Jul 2"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(subject, "Fresh replies:") {
		t.Fatalf("subject: %q", subject)
	}
	for _, want := range []string{
		"PutterTalk Community",
		"Jump back in",
		"Your threads",
		"Unsubscribe from this thread",
		"/brand/pt-bug.svg",
		"Testing all of this out.",
		"bundle replies into one digest",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("html missing %q", want)
		}
	}
	for _, avoid := range []string{"watching at", "Watched threads", "disable all emails", "Re: "} {
		if strings.Contains(html, avoid) {
			t.Fatalf("html still contains XenForo-style copy %q", avoid)
		}
	}
	if !strings.Contains(text, "Simon") {
		t.Fatalf("text missing author")
	}
	if !strings.Contains(text, "Jump back in:") {
		t.Fatalf("text missing CTA label")
	}
}

func TestBuildIntroLine(t *testing.T) {
	got := BuildIntroLine(1, "Simon", "PutterTalk Community")
	if !strings.Contains(got, "Simon chimed in") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "watching") {
		t.Fatalf("got XenForo-style copy %q", got)
	}
	got = BuildIntroLine(3, "Simon", "PutterTalk Community")
	if !strings.Contains(got, "3 new replies") {
		t.Fatalf("got %q", got)
	}
}