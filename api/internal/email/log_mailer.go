package email

import (
	"context"
	"log"
)

// LogMailer prints emails to stdout (dev default).
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	if msg.ListUnsubscribeURL != "" {
		log.Printf("--- email [%s] to=%s subject=%q list-unsubscribe=%s ---\n%s\n--- end email ---",
			msg.MessageType, msg.To, msg.Subject, msg.ListUnsubscribeURL, msg.TextBody)
		return nil
	}
	log.Printf("--- email [%s] to=%s subject=%q ---\n%s\n--- end email ---",
		msg.MessageType, msg.To, msg.Subject, msg.TextBody)
	return nil
}