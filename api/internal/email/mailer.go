package email

import "context"

// Message is a single outbound email.
type Message struct {
	To                 string
	Subject            string
	HTMLBody           string
	TextBody           string
	ReplyTo            string
	MessageType        string // thread_reply, magic_link, password_reset
	ListUnsubscribeURL string // SES one-click unsubscribe (RFC 8058)
}

// Mailer sends email via a configured provider (SES, SMTP, log).
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}