package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/netutil"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
)

func (a *API) emailVerifyConsume(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			token = strings.TrimSpace(req.Token)
		}
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token required")
		return
	}

	actor, err := a.auth.VerifyEmailWithToken(r.Context(), token)
	if errors.Is(err, auth.ErrInvalidToken) {
		writeError(w, http.StatusBadRequest, "link expired or already used — request a new one")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	enriched, _ := a.auth.EnrichActor(r.Context(), actor, a.siteCfg.Admin.Emails, a.siteCfg.Admin.ModeratorEmails, a.siteCfg.Admin.PermBanModeratorEmails)
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "verified",
		"actor":  a.actorJSON(r.Context(), enriched),
	})
}

func (a *API) emailVerifyResend(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if a.rateLimited(w, "auth-email:ip:"+ip, ratelimit.AuthEmailIPLimit, ratelimit.AuthEmailIPWindow) {
		return
	}

	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	enriched, ok := a.enrichActor(w, r, actor)
	if !ok {
		return
	}
	if enriched.EmailVerified {
		writeJSON(w, http.StatusOK, map[string]string{"status": "already_verified"})
		return
	}
	if enriched.Email == "" {
		writeError(w, http.StatusBadRequest, "no email on account")
		return
	}
	if a.rateLimited(w, "auth-email:addr:"+enriched.Email, ratelimit.AuthEmailAddrLimit, ratelimit.AuthEmailAddrWindow) {
		return
	}

	token, _, err := a.auth.IssueEmailVerifyToken(r.Context(), enriched.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	verifyURL := strings.TrimRight(a.siteCfg.Site.URL, "/") + "/auth/verify?token=" + token
	if err := a.notify.SendEmailVerification(r.Context(), enriched.Email, enriched.DisplayName, verifyURL); err != nil {
		writeError(w, http.StatusInternalServerError, "could not send email")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (a *API) sendVerificationEmail(ctx context.Context, actorID string) error {
	token, actor, err := a.auth.IssueEmailVerifyToken(ctx, actorID)
	if err != nil {
		return err
	}
	verifyURL := strings.TrimRight(a.siteCfg.Site.URL, "/") + "/auth/verify?token=" + token
	return a.notify.SendEmailVerification(ctx, actor.Email, actor.DisplayName, verifyURL)
}