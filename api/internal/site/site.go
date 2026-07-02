package site

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuroi/xuroi/api/internal/models"
	"github.com/xuroi/xuroi/api/internal/policy"
	"github.com/xuroi/xuroi/api/internal/spam"
)

type PostPolicy struct {
	EditEnabled       bool `json:"edit_enabled"`
	EditWindowMinutes int  `json:"edit_window_minutes"`
	// DeleteEnabled: site owner toggle. When true, admins may soft-delete replies.
	// When false, delete is hidden and blocked for everyone (members never can).
	DeleteEnabled bool `json:"delete_enabled"`
}

type AdminPolicy struct {
	Emails                 []string `json:"emails"`
	ModeratorEmails        []string `json:"moderator_emails"`
	PermBanModeratorEmails []string `json:"perm_ban_moderator_emails"`
}

type GuestPolicy struct {
	ReadOnly  bool `json:"read_only"`
	CanAttach bool `json:"can_attach"`
}

// IntelligencePolicy controls thread summaries (heuristic or LLM via env API key).
type IntelligencePolicy struct {
	Enabled      bool   `json:"enabled"`
	SummaryLabel string `json:"summary_label"`
}

// EmailPolicy is site-owner email settings (provider credentials live in env).
type EmailPolicy struct {
	Enabled            bool   `json:"enabled"`
	FromAddress        string `json:"from_address"`
	FromName           string `json:"from_name"`
	ReplyTo            string `json:"reply_to"`
	DigestDelayMinutes int    `json:"digest_delay_minutes"`
}

func (p EmailPolicy) Normalized() EmailPolicy {
	out := p
	if out.DigestDelayMinutes <= 0 {
		out.DigestDelayMinutes = 5
	}
	if out.FromName == "" {
		out.FromName = "Community"
	}
	return out
}

func (p IntelligencePolicy) Normalized() IntelligencePolicy {
	out := p
	if out.SummaryLabel == "" {
		out.SummaryLabel = "Summary"
	}
	return out
}

type ReportReason struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	AllowDetail bool   `json:"allow_detail,omitempty"`
}

type ModerationPolicy struct {
	ReportReasons []ReportReason `json:"report_reasons"`
}

func (p ModerationPolicy) Normalized() ModerationPolicy {
	out := p
	if len(out.ReportReasons) == 0 {
		out.ReportReasons = DefaultReportReasons()
	}
	return out
}

func DefaultReportReasons() []ReportReason {
	return []ReportReason{
		{ID: "spam", Label: "Spam or advertising"},
		{ID: "harassment", Label: "Harassment or abuse"},
		{ID: "off_topic", Label: "Off-topic"},
		{ID: "inappropriate", Label: "Inappropriate content"},
		{ID: "other", Label: "Other", AllowDetail: true},
	}
}

func (p ModerationPolicy) FormatReportReason(reasonID, detail string) (string, error) {
	reasonID = strings.TrimSpace(reasonID)
	detail = strings.TrimSpace(detail)
	if reasonID == "" {
		return "", errors.New("report reason required")
	}
	for _, r := range p.Normalized().ReportReasons {
		if r.ID != reasonID {
			continue
		}
		if r.AllowDetail && detail != "" {
			text := r.Label + ": " + detail
			if len(text) > 500 {
				text = text[:500]
			}
			return text, nil
		}
		return r.Label, nil
	}
	return "", errors.New("invalid report reason")
}

type Config struct {
	Site                 models.Site
	Posts                PostPolicy
	Admin                AdminPolicy
	Guests               GuestPolicy
	Intelligence         IntelligencePolicy
	Email                EmailPolicy
	Moderation           ModerationPolicy
	NewUsers             policy.NewUserPolicy
	Spam                 spam.Policy
	SEO                  SEOPolicy
	ReservedDisplayNames []string
	SiteJSONPath         string
}

type fileConfig struct {
	Name         string             `json:"name"`
	Tagline      string             `json:"tagline"`
	Posts        PostPolicy         `json:"posts"`
	Admin        AdminPolicy        `json:"admin"`
	Guests       GuestPolicy        `json:"guests"`
	Intelligence IntelligencePolicy `json:"intelligence"`
	Email                 EmailPolicy      `json:"email"`
	Moderation            ModerationPolicy `json:"moderation"`
	ReservedDisplayNames  []string              `json:"reserved_display_names"`
	NewUsers              policy.NewUserPolicy  `json:"new_users"`
	Spam                  spam.Policy           `json:"spam"`
	SEO                   *SEOPolicy            `json:"seo"`
	Features              struct {
		ThreadIntelligence bool `json:"thread_intelligence"`
	} `json:"features"`
}

func Load() Config {
	cfg := Config{
		Site: models.Site{
			Name: "PutterTalk",
			URL:  "http://localhost:4321",
		},
		Posts: PostPolicy{
			EditEnabled:       true,
			EditWindowMinutes: 30,
			DeleteEnabled:     false,
		},
		Intelligence: IntelligencePolicy{
			Enabled:      true,
			SummaryLabel: "Summary",
		},
		Email: EmailPolicy{
			Enabled:            true,
			DigestDelayMinutes: 5,
		},
		SEO: DefaultSEOPolicy(),
	}

	path := os.Getenv("SITE_JSON")
	if path == "" {
		path = filepath.Join("..", "sites", "puttertalk", "site.json")
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	cfg.SiteJSONPath = path
	if data, err := os.ReadFile(path); err == nil {
		var f fileConfig
		if err := json.Unmarshal(data, &f); err == nil {
			if f.Name != "" {
				cfg.Site.Name = f.Name
			}
			if f.Tagline != "" {
				cfg.Site.Tagline = f.Tagline
			}
			cfg.Posts = f.Posts
			cfg.Admin = f.Admin
			cfg.Guests = f.Guests
			cfg.Intelligence = f.Intelligence.Normalized()
			if !f.Intelligence.Enabled && f.Features.ThreadIntelligence {
				cfg.Intelligence.Enabled = true
			}
			cfg.Intelligence = cfg.Intelligence.Normalized()
			cfg.Email = f.Email.Normalized()
			if cfg.Email.FromName == "Community" && cfg.Site.Name != "" {
				cfg.Email.FromName = cfg.Site.Name
			}
			cfg.Moderation = f.Moderation.Normalized()
			cfg.ReservedDisplayNames = f.ReservedDisplayNames
			cfg.NewUsers = f.NewUsers.Normalized()
			cfg.Spam = f.Spam.Normalized()
			if f.SEO != nil {
				cfg.SEO = *f.SEO
			}
		}
	}

	if v := os.Getenv("SITE_URL"); v != "" {
		cfg.Site.URL = v
	}

	if cfg.Posts.EditEnabled && cfg.Posts.EditWindowMinutes <= 0 {
		cfg.Posts.EditWindowMinutes = 30
	}

	cfg.Intelligence = cfg.Intelligence.Normalized()
	cfg.Email = cfg.Email.Normalized()
	if cfg.Email.FromName == "Community" && cfg.Site.Name != "" {
		cfg.Email.FromName = cfg.Site.Name
	}
	cfg.Moderation = cfg.Moderation.Normalized()
	cfg.NewUsers = cfg.NewUsers.Normalized()
	cfg.Spam = cfg.Spam.Normalized()
	return cfg
}

// Save writes the editable subset back to site.json (admin settings).
func Save(cfg Config, path string) error {
	if path == "" {
		path = cfg.SiteJSONPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	patch := map[string]any{
		"name":                   cfg.Site.Name,
		"tagline":                cfg.Site.Tagline,
		"posts":                  cfg.Posts,
		"guests":                 cfg.Guests,
		"intelligence":           cfg.Intelligence,
		"email":                  cfg.Email,
		"admin":                  cfg.Admin,
		"moderation":             cfg.Moderation,
		"new_users":              cfg.NewUsers,
		"spam":                   cfg.Spam,
		"seo":                    cfg.SEO,
		"reserved_display_names": cfg.ReservedDisplayNames,
	}
	for k, v := range patch {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		raw[k] = b
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}