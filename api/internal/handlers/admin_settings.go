package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/policy"
	"github.com/xuroi/xuroi/api/internal/site"
	"github.com/xuroi/xuroi/api/internal/spam"
)

func (a *API) getAdminSiteSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
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
		"reserved_display_names": a.siteCfg.ReservedDisplayNames,
	})
}

func (a *API) patchAdminSiteSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
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
		ReservedDisplayNames  *[]string             `json:"reserved_display_names"`
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
		cfg.Spam = req.Spam.Normalized()
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
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.siteCfg = cfg
	a.forum.SetPostPolicy(cfg.Posts)
	a.reader.SetPostPolicy(cfg.Posts)
	a.reader.SetIntelligence(cfg.Intelligence)
	a.auth.SetReservedDisplayNames(cfg.ReservedDisplayNames)
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved"})
}