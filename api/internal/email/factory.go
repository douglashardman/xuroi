package email

import (
	"fmt"
	"log"
	"strings"
)

// NewMailer returns a configured mailer. Unknown provider falls back to log.
func NewMailer(cfg Config) (Mailer, error) {
	if !cfg.Enabled {
		return LogMailer{}, nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "ses":
		if cfg.FromAddress == "" {
			return nil, fmt.Errorf("XUROI_EMAIL_FROM required for SES")
		}
		m, err := NewSESMailer(cfg)
		if err != nil {
			return nil, err
		}
		log.Printf("email: Amazon SES (%s)", cfg.AWSRegion)
		return m, nil
	case "smtp":
		if cfg.FromAddress == "" || cfg.SMTPHost == "" {
			return nil, fmt.Errorf("XUROI_EMAIL_FROM and XUROI_SMTP_HOST required for SMTP")
		}
		log.Printf("email: SMTP %s:%d", cfg.SMTPHost, cfg.SMTPPort)
		return SMTPMailer{cfg: cfg}, nil
	case "log", "":
		log.Printf("email: log only (set XUROI_EMAIL_PROVIDER=ses to send)")
		return LogMailer{}, nil
	default:
		return nil, fmt.Errorf("unknown email provider %q", cfg.Provider)
	}
}