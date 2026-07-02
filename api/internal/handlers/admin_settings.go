package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/policy"
	"github.com/xuroi/xuroi/api/internal/site"
	"github.com/xuroi/xuroi/api/internal/spam"
)

func (a *API) siteSettingsPayload() map[string]any {
	return map[string]any{
		"name":                   a.siteCfg.Site.Name,
		"tagline":                a.siteCfg.Site.Tagline,
		"posts":                  a.siteCfg.Posts,
		"guests":                 a.siteCfg.Guests,
		"intelligence":           a.siteCfg.Intelligence,
		"email":                  a.siteCfg.Email,
		"admin":                  a.siteCfg.Admin,
		"moderation":             a.siteCfg.Moderation,
		"new_users":              a.siteCfg.NewUsers,
		"spam":                   a.siteCfg.Spam,
		"seo":                    a.siteCfg.SEO,
		"reserved_display_names": a.siteCfg.ReservedDisplayNames,
		"maintenance":            a.siteCfg.Maintenance,
		"registration":           a.siteCfg.Registration,
	}
}

func (a *API) getAdminSiteSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.siteSettingsPayload())
}

func (a *API) patchAdminSiteSettings(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}
	var req struct {
		Name         *string                 `json:"name"`
		Tagline      *string                 `json:"tagline"`
		Posts        *site.PostPolicy        `json:"posts"`
		Guests       *site.GuestPolicy       `json:"guests"`
		Intelligence *site.IntelligencePolicy `json:"intelligence"`
		Email        *site.EmailPolicy       `json:"email"`
		Admin        *site.AdminPolicy       `json:"admin"`
		Moderation   *site.ModerationPolicy  `json:"moderation"`
		NewUsers              *policy.NewUserPolicy `json:"new_users"`
		Spam                  *spam.Policy          `json:"spam"`
		SEO                   *site.SEOPolicy       `json:"seo"`
		ReservedDisplayNames  *[]string              `json:"reserved_display_names"`
		Maintenance           *site.MaintenancePolicy `json:"maintenance"`
		Registration          *site.RegistrationPolicy `json:"registration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	cfg := a.siteCfg
	if req.Name != nil {
		cfg.Site.Name = strings.TrimSpace(*req.Name)
	}
	if req.Tagline != nil {
		cfg.Site.Tagline = strings.TrimSpace(*req.Tagline)
	}
	if req.Posts != nil {
		cfg.Posts = *req.Posts
		if cfg.Posts.EditEnabled && cfg.Posts.EditWindowMinutes <= 0 {
			cfg.Posts.EditWindowMinutes = 30
		}
	}
	if req.Guests != nil {
		cfg.Guests = *req.Guests
	}
	if req.Intelligence != nil {
		cfg.Intelligence = req.Intelligence.Normalized()
	}
	if req.Email != nil {
		cfg.Email = req.Email.Normalized()
	}
	if req.Moderation != nil {
		cfg.Moderation = req.Moderation.Normalized()
	}
	if req.Admin != nil {
		cfg.Admin.Emails = req.Admin.Emails
		if len(cfg.Admin.Emails) == 0 {
			cfg.Admin.Emails = a.siteCfg.Admin.Emails
		}
		cfg.Admin.ModeratorEmails = req.Admin.ModeratorEmails
		cfg.Admin.PermBanModeratorEmails = req.Admin.PermBanModeratorEmails
		if len(cfg.Admin.PermBanModeratorEmails) == 0 {
			cfg.Admin.PermBanModeratorEmails = a.siteCfg.Admin.PermBanModeratorEmails
		}
	}
	if req.NewUsers != nil {
		cfg.NewUsers = req.NewUsers.Normalized()
	}
	if req.Spam != nil {
		incoming := req.Spam.Normalized()
		if !incoming.Enabled {
			incoming.MaxLinksNewUser = cfg.Spam.MaxLinksNewUser
			incoming.NewAccountHours = cfg.Spam.NewAccountHours
			incoming.ScoreThreshold = cfg.Spam.ScoreThreshold
			incoming.HoldForModeration = cfg.Spam.HoldForModeration
			if incoming.MaxLinksNewUser == 0 {
				incoming = cfg.Spam
				incoming.Enabled = false
			}
		}
		cfg.Spam = incoming
	}
	if req.SEO != nil {
		cfg.SEO = *req.SEO
	}
	if req.Maintenance != nil {
		cfg.Maintenance = req.Maintenance.Normalized()
	}
	if req.Registration != nil {
		cfg.Registration = req.Registration.Normalized()
	}
	if req.ReservedDisplayNames != nil {
		names := make([]string, 0, len(*req.ReservedDisplayNames))
		seen := make(map[string]struct{})
		for _, raw := range *req.ReservedDisplayNames {
			name := strings.ToLower(strings.TrimSpace(raw))
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
		cfg.ReservedDisplayNames = names
	}

	if err := site.Save(cfg, cfg.SiteJSONPath); err != nil {
		writeError(w, http.StatusInternalServerError, "save site.json: "+err.Error())
		return
	}
	a.siteCfg = cfg
	a.forum.SetPostPolicy(cfg.Posts)
	a.reader.SetPostPolicy(cfg.Posts)
	a.reader.SetIntelligence(cfg.Intelligence)
	a.reader.SetSEOPolicy(cfg.SEO)
	a.auth.SetReservedDisplayNames(cfg.ReservedDisplayNames)
	a.auth.SetRegistrationPolicy(cfg.Registration)
	_ = a.forum.LogAdminEvent(r.Context(), events.TypeAdminSettingsUpdated, admin.ID, map[string]string{
		"path": cfg.SiteJSONPath,
	})
	writeJSON(w, http.StatusOK, a.siteSettingsPayload())
}