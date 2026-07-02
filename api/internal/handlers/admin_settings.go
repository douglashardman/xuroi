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
		"name":      a.siteCfg.Site.Name,
		"tagline":   a.siteCfg.Site.Tagline,
		"email":     a.siteCfg.Email,
		"admin":     a.siteCfg.Admin,
		"new_users": a.siteCfg.NewUsers,
		"spam":      a.siteCfg.Spam,
	})
}

func (a *API) patchAdminSiteSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	var req struct {
		Name     *string                `json:"name"`
		Tagline  *string                `json:"tagline"`
		Email    *site.EmailPolicy      `json:"email"`
		Admin    *site.AdminPolicy      `json:"admin"`
		NewUsers *policy.NewUserPolicy  `json:"new_users"`
		Spam     *spam.Policy           `json:"spam"`
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
	if req.Email != nil {
		cfg.Email = req.Email.Normalized()
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

	if err := site.Save(cfg, cfg.SiteJSONPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.siteCfg = cfg
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved"})
}