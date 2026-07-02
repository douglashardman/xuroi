package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPMailer sends via any RFC5321 server (Mailgun, SendGrid SMTP, etc.).
type SMTPMailer struct {
	cfg Config
}

func (m SMTPMailer) Send(_ context.Context, msg Message) error {
	from := m.cfg.FromAddress
	headers := []string{
		"From: " + m.cfg.FromHeader(),
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	replyTo := msg.ReplyTo
	if replyTo == "" {
		replyTo = m.cfg.ReplyTo
	}
	if replyTo != "" {
		headers = append(headers, "Reply-To: "+replyTo)
	}
	body := strings.Join(headers, "\r\n") + "\r\n\r\n" + msg.HTMLBody

	host := strings.TrimSpace(m.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", host, m.cfg.SMTPPort)
	var auth smtp.Auth
	if m.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPass, host)
	}
	return smtp.SendMail(addr, auth, from, []string{msg.To}, []byte(body))
}