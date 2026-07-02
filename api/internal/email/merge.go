package email

import "github.com/xuroi/xuroi/api/internal/site"

// MergeSiteDefaults fills env config gaps from site.json.
func MergeSiteDefaults(cfg Config, siteCfg site.Config) Config {
	email := siteCfg.Email.Normalized()
	if cfg.FromAddress == "" {
		cfg.FromAddress = email.FromAddress
	}
	if cfg.FromName == "" {
		cfg.FromName = email.FromName
	}
	if cfg.ReplyTo == "" {
		cfg.ReplyTo = email.ReplyTo
	}
	if cfg.DigestDelayMinutes <= 0 && email.DigestDelayMinutes > 0 {
		cfg.DigestDelayMinutes = email.DigestDelayMinutes
	}
	return cfg
}