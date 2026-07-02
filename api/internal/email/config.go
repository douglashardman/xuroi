package email

import (
	"os"
	"strconv"
	"strings"
)

// Config holds provider credentials (env) and site-facing from/reply settings.
type Config struct {
	Provider string // ses, smtp, log
	Enabled  bool

	FromAddress string
	FromName    string
	ReplyTo     string

	DigestDelayMinutes int

	// SES
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string

	// SMTP
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPTLS  bool
}

func ConfigFromEnv() Config {
	delay := 5
	if v := os.Getenv("XUROI_EMAIL_DIGEST_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			delay = n
		}
	}
	port := 587
	if v := os.Getenv("XUROI_SMTP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			port = n
		}
	}
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("XUROI_EMAIL_PROVIDER")))
	if provider == "" {
		provider = "log"
	}
	return Config{
		Provider:           provider,
		Enabled:            envBool("XUROI_EMAIL_ENABLED", true),
		FromAddress:        os.Getenv("XUROI_EMAIL_FROM"),
		FromName:           os.Getenv("XUROI_EMAIL_FROM_NAME"),
		ReplyTo:            os.Getenv("XUROI_EMAIL_REPLY_TO"),
		DigestDelayMinutes: delay,
		AWSRegion:          firstNonEmpty(os.Getenv("XUROI_AWS_REGION"), os.Getenv("AWS_REGION"), "us-east-1"),
		AWSAccessKeyID:     firstNonEmpty(os.Getenv("XUROI_AWS_ACCESS_KEY_ID"), os.Getenv("AWS_ACCESS_KEY_ID")),
		AWSSecretAccessKey: firstNonEmpty(os.Getenv("XUROI_AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SECRET_ACCESS_KEY")),
		SMTPHost:           os.Getenv("XUROI_SMTP_HOST"),
		SMTPPort:           port,
		SMTPUser:           os.Getenv("XUROI_SMTP_USER"),
		SMTPPass:           os.Getenv("XUROI_SMTP_PASS"),
		SMTPTLS:            envBool("XUROI_SMTP_TLS", true),
	}
}

func (c Config) FromHeader() string {
	if c.FromName != "" && c.FromAddress != "" {
		return c.FromName + " <" + c.FromAddress + ">"
	}
	return c.FromAddress
}

func envBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}