package email

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"text/template"
)

const (
	colorPink = "#e00080"
	colorBlue = "#27bcee"
	colorInk  = "#1a1a1a"
	colorFoot = "#161719"
)

// ThreadReplyData is rendered into the thread notification email.
type ThreadReplyData struct {
	CommunityName   string
	SiteURL         string
	LogoURL         string
	Recipient       string
	IntroLine       string
	ThreadTitle     string
	ThreadURL       string
	CommunityURL    string
	WatchedURL      string
	DisableThreadURL string
	DisableAllURL   string
	UnsubscribeURL  string
	Copyright       string
	ReplyCount      int
	Posts           []ThreadReplyPost
}

type ThreadReplyPost struct {
	Author  string
	Excerpt string
	When    string
}

// LogoURL builds the absolute bug logo URL for emails.
func LogoURL(siteURL string) string {
	return strings.TrimRight(siteURL, "/") + "/brand/pt-bug.svg"
}

// BuildIntroLine is the pink opener line for thread digest emails.
func BuildIntroLine(replyCount int, latestAuthor, communityName string) string {
	_ = communityName // reserved for per-site variants later
	if replyCount == 1 && latestAuthor != "" {
		return fmt.Sprintf("%s chimed in on a thread you posted in.", latestAuthor)
	}
	replyWord := "replies"
	if replyCount == 1 {
		replyWord = "reply"
	}
	return fmt.Sprintf("%d new %s in a thread you posted in.", replyCount, replyWord)
}

// RenderThreadReply builds HTML + plain text for a thread digest email.
func RenderThreadReply(data ThreadReplyData) (subject, htmlBody, textBody string, err error) {
	if data.LogoURL == "" {
		data.LogoURL = LogoURL(data.SiteURL)
	}
	if data.CommunityURL == "" {
		data.CommunityURL = strings.TrimRight(data.SiteURL, "/") + "/community"
	}
	if data.WatchedURL == "" {
		data.WatchedURL = data.CommunityURL
	}
	if data.Copyright == "" {
		data.Copyright = "© 2006–2026 PutterTalk LLC."
	}

	subject = fmt.Sprintf("Fresh replies: %s", data.ThreadTitle)

	var htmlBuf bytes.Buffer
	if err := threadReplyHTML.Execute(&htmlBuf, data); err != nil {
		return "", "", "", err
	}

	var text strings.Builder
	fmt.Fprintf(&text, "%s\n\n", data.CommunityName)
	fmt.Fprintf(&text, "%s\n\n", data.IntroLine)
	fmt.Fprintf(&text, "%s\n\n", data.ThreadTitle)
	for _, p := range data.Posts {
		fmt.Fprintf(&text, "%s\n%s\n\n", p.Author, p.Excerpt)
	}
	fmt.Fprintf(&text, "Jump back in: %s\n", data.ThreadURL)
	if data.UnsubscribeURL != "" {
		fmt.Fprintf(&text, "\nUnsubscribe from this thread: %s\n", data.UnsubscribeURL)
	}
	fmt.Fprintf(&text, "\n%s\n", data.Copyright)

	return subject, htmlBuf.String(), strings.TrimSpace(text.String()), nil
}

var threadReplyHTML = template.Must(template.New("thread_reply").Funcs(template.FuncMap{
	"e": html.EscapeString,
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{e .CommunityName}}</title>
</head>
<body style="margin:0;padding:0;background:#ececec;font-family:'Open Sans',Helvetica,Arial,sans-serif;color:` + colorInk + `;">
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#ececec;padding:24px 12px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:640px;background:#ffffff;border:1px solid #d8d8d8;border-radius:8px;overflow:hidden;">
          <tr>
            <td style="padding:24px 28px 16px;">
              <table role="presentation" width="100%" cellspacing="0" cellpadding="0">
                <tr>
                  <td width="56" valign="middle" style="padding-right:14px;">
                    <img src="{{e .LogoURL}}" width="48" height="48" alt="{{e .CommunityName}}" style="display:block;border:0;" />
                  </td>
                  <td valign="middle" style="font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:22px;font-weight:800;color:` + colorInk + `;">{{e .CommunityName}}</td>
                </tr>
              </table>
              <div style="height:1px;background:` + colorInk + `;margin-top:18px;"></div>
            </td>
          </tr>
          <tr>
            <td style="padding:8px 28px 0;font-size:16px;line-height:1.55;">
              <p style="margin:0 0 14px;color:` + colorPink + `;font-weight:600;">{{e .IntroLine}}</p>
              <p style="margin:0 0 18px;font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:20px;font-weight:800;color:` + colorInk + `;">{{e .ThreadTitle}}</p>
            </td>
          </tr>
          {{range .Posts}}
          <tr>
            <td style="padding:0 28px 14px;">
              <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f3f3f3;border-left:4px solid ` + colorBlue + `;">
                <tr>
                  <td style="padding:14px 16px;font-size:15px;line-height:1.55;color:#333333;">{{e .Excerpt}}</td>
                </tr>
              </table>
            </td>
          </tr>
          {{end}}
          <tr>
            <td style="padding:6px 28px 22px;">
              <a href="{{e .ThreadURL}}" style="display:inline-block;background:` + colorPink + `;color:#ffffff;text-decoration:none;font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:15px;font-weight:700;padding:12px 22px;border-radius:999px;margin-right:12px;">Jump back in</a>
              <a href="{{e .WatchedURL}}" style="display:inline-block;background:#ffffff;color:` + colorInk + `;text-decoration:none;font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:15px;font-weight:700;padding:10px 20px;border-radius:999px;border:2px solid ` + colorInk + `;">Your threads</a>
            </td>
          </tr>
          <tr>
            <td style="padding:0 28px 18px;font-size:14px;line-height:1.55;color:#666666;">
              <p style="margin:0 0 14px;">Automated heads-up from {{e .CommunityName}} — reply on the site, not to this email.</p>
              <p style="margin:0 0 14px;">You posted in <strong>{{e .ThreadTitle}}</strong>, so we ping you when others weigh in. We bundle replies into one digest so your inbox stays quiet. Open the thread and we'll wait for the next round before emailing again.</p>
              <p style="margin:0;">
                <a href="{{e .UnsubscribeURL}}" style="color:` + colorPink + `;font-weight:600;text-decoration:none;">Unsubscribe from this thread</a>
                &nbsp;·&nbsp;
                <a href="{{e .DisableAllURL}}" style="color:` + colorPink + `;font-weight:600;text-decoration:none;">Turn off all thread emails</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:18px 28px;background:` + colorFoot + `;text-align:center;">
              <a href="{{e .CommunityURL}}" style="color:#ffffff;font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:15px;font-weight:700;text-decoration:none;">Open {{e .CommunityName}}</a>
              {{if .UnsubscribeURL}}
              <div style="margin-top:14px;font-size:12px;color:#9a9a9a;">
                <a href="{{e .UnsubscribeURL}}" style="color:#9a9a9a;text-decoration:underline;">Unsubscribe from this thread</a>
              </div>
              {{end}}
              <div style="margin-top:12px;font-size:12px;color:#9a9a9a;">{{e .Copyright}}</div>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`))

// AuthEmailData is used for magic links, password reset, and verification.
type AuthEmailData struct {
	CommunityName string
	SiteURL       string
	LogoURL       string
	Headline      string
	BodyHTML      string
	CTALabel      string
	CTAURL        string
	CommunityURL  string
	Copyright     string
	FooterNote    string
}

// RenderAuthEmail builds a styled auth email matching the notification design.
func RenderAuthEmail(data AuthEmailData) (subject, htmlBody, textBody string, err error) {
	if data.LogoURL == "" {
		data.LogoURL = LogoURL(data.SiteURL)
	}
	if data.CommunityURL == "" {
		data.CommunityURL = strings.TrimRight(data.SiteURL, "/") + "/community"
	}
	if data.Copyright == "" {
		data.Copyright = "© 2006–2026 PutterTalk LLC."
	}
	if data.FooterNote == "" {
		data.FooterNote = fmt.Sprintf("Automated message from %s — no need to reply.", data.CommunityName)
	}
	subject = data.Headline

	var htmlBuf bytes.Buffer
	if err := authEmailHTML.Execute(&htmlBuf, data); err != nil {
		return "", "", "", err
	}

	text := strings.TrimSpace(stripTags(data.BodyHTML))
	if text == "" {
		text = data.Headline
	}
	textBody = fmt.Sprintf("%s\n\n%s\n\n%s: %s\n\n%s", data.CommunityName, text, data.CTALabel, data.CTAURL, data.FooterNote)

	return subject, htmlBuf.String(), textBody, nil
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

var authEmailHTML = template.Must(template.New("auth").Funcs(template.FuncMap{
	"e":    html.EscapeString,
	"safe": func(s string) string { return s },
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8" /><meta name="viewport" content="width=device-width, initial-scale=1" /></head>
<body style="margin:0;padding:0;background:#ececec;font-family:'Open Sans',Helvetica,Arial,sans-serif;color:` + colorInk + `;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#ececec;padding:24px 12px;"><tr><td align="center">
<table role="presentation" width="100%" style="max-width:640px;background:#fff;border:1px solid #d8d8d8;border-radius:8px;overflow:hidden;">
<tr><td style="padding:24px 28px 16px;">
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr>
    <td width="56" valign="middle" style="padding-right:14px;"><img src="{{e .LogoURL}}" width="48" height="48" alt="{{e .CommunityName}}" style="display:block;border:0;" /></td>
    <td valign="middle" style="font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:22px;font-weight:800;">{{e .CommunityName}}</td>
  </tr></table>
  <div style="height:1px;background:` + colorInk + `;margin-top:18px;"></div>
</td></tr>
<tr><td style="padding:8px 28px 0;font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:20px;font-weight:800;">{{e .Headline}}</td></tr>
<tr><td style="padding:18px 28px;font-size:15px;line-height:1.55;color:#333;">{{safe .BodyHTML}}</td></tr>
<tr><td style="padding:0 28px 22px;"><a href="{{e .CTAURL}}" style="display:inline-block;background:` + colorPink + `;color:#fff;text-decoration:none;font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:15px;font-weight:700;padding:12px 22px;border-radius:999px;">{{e .CTALabel}}</a></td></tr>
<tr><td style="padding:0 28px 18px;font-size:14px;color:#666;">{{e .FooterNote}}</td></tr>
<tr><td style="padding:18px 28px;background:` + colorFoot + `;text-align:center;">
  <a href="{{e .CommunityURL}}" style="color:#fff;font-family:Poppins,'Helvetica Neue',Arial,sans-serif;font-size:15px;font-weight:700;text-decoration:none;">Open {{e .CommunityName}}</a>
  <div style="margin-top:12px;font-size:12px;color:#9a9a9a;">{{e .Copyright}}</div>
</td></tr>
</table></td></tr></table></body></html>`))