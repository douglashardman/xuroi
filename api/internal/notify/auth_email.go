package notify

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/xuroi/xuroi/api/internal/email"
)

func (s *Service) SendPasswordReset(ctx context.Context, to, displayName, resetURL string) error {
	return s.sendAuthEmail(ctx, to, email.AuthEmailData{
		Headline: "Reset your password",
		BodyHTML: fmt.Sprintf(
			`<p>Hey %s,</p><p>Someone asked to reset your password. Tap below to pick a new one — the link expires in one hour.</p><p>Wasn't you? Ignore this and your password stays as-is.</p>`,
			html.EscapeString(displayName),
		),
		CTALabel: "Choose new password",
		CTAURL:   resetURL,
	}, "password_reset")
}

func (s *Service) SendEmailVerification(ctx context.Context, to, displayName, verifyURL string) error {
	return s.sendAuthEmail(ctx, to, email.AuthEmailData{
		Headline: "Confirm your email",
		BodyHTML: fmt.Sprintf(
			`<p>Hey %s,</p><p>Tap below to confirm this address and unlock posting. The link expires in 48 hours.</p><p>Didn't sign up? Ignore this.</p>`,
			html.EscapeString(displayName),
		),
		CTALabel: "Confirm email",
		CTAURL:   verifyURL,
	}, "email_verify")
}

func (s *Service) SendMagicLink(ctx context.Context, to, displayName, signInURL string) error {
	return s.sendAuthEmail(ctx, to, email.AuthEmailData{
		Headline: "Your sign-in link",
		BodyHTML: fmt.Sprintf(
			`<p>Hey %s,</p><p>Here's a one-time link to get you back in — no password needed. It expires in 15 minutes and works once.</p><p>Didn't ask for this? Safe to ignore.</p>`,
			html.EscapeString(displayName),
		),
		CTALabel: "Sign me in",
		CTAURL:   signInURL,
	}, "magic_link")
}

func (s *Service) sendAuthEmail(ctx context.Context, to string, data email.AuthEmailData, messageType string) error {
	if !s.site.Email.Enabled {
		return nil
	}
	siteURL := strings.TrimRight(s.site.Site.URL, "/")
	communityName := s.site.Email.FromName
	if communityName == "" {
		communityName = s.site.Site.Name + " Community"
	}
	data.CommunityName = communityName
	data.SiteURL = siteURL
	data.LogoURL = email.LogoURL(siteURL)
	data.CommunityURL = siteURL + "/community"
	data.Copyright = "© 2006–2026 PutterTalk LLC."

	subject, htmlBody, textBody, err := email.RenderAuthEmail(data)
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, email.Message{
		To:          to,
		Subject:     subject,
		HTMLBody:    htmlBody,
		TextBody:    textBody,
		ReplyTo:     s.site.Email.ReplyTo,
		MessageType: messageType,
	})
}