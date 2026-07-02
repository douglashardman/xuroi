package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/netutil"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
)

const authEmailSentMsg = "If that email is registered, we sent a link."

func (a *API) passwordResetRequest(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if a.rateLimited(w, "auth-email:ip:"+ip, ratelimit.AuthEmailIPLimit, ratelimit.AuthEmailIPWindow) {
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		writeError(w, http.StatusBadRequest, "email required")
		return
	}
	if a.rateLimited(w, "auth-email:addr:"+email, ratelimit.AuthEmailAddrLimit, ratelimit.AuthEmailAddrWindow) {
		return
	}

	token, actor, err := a.auth.IssuePasswordResetToken(r.Context(), email)
	if errors.Is(err, auth.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "message": authEmailSentMsg})
		return
	}
	if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "valid email required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resetURL := strings.TrimRight(a.siteCfg.Site.URL, "/") + "/reset-password?token=" + token
	if err := a.notify.SendPasswordReset(r.Context(), actor.Email, actor.DisplayName, resetURL); err != nil {
		writeError(w, http.StatusInternalServerError, "could not send email")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "message": authEmailSentMsg})
}

func (a *API) passwordResetConfirm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		writeError(w, http.StatusBadRequest, "token required")
		return
	}

	actor, sessionToken, err := a.auth.ResetPasswordWithToken(r.Context(), req.Token, req.Password)
	if errors.Is(err, auth.ErrInvalidToken) {
		writeError(w, http.StatusBadRequest, "link expired or already used — request a new one")
		return
	}
	if errors.Is(err, auth.ErrInvalidPassword) {
		writeError(w, http.StatusBadRequest, "password must be 8–128 characters")
		return
	}
	if a.respondIfBanned(w, r, actor, err) {
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	setSessionCookie(w, sessionToken)
	enriched, ok := a.enrichActor(w, r, actor)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.authResponse(r.Context(), enriched, sessionToken))
}

func (a *API) magicLinkRequest(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if a.rateLimited(w, "auth-email:ip:"+ip, ratelimit.AuthEmailIPLimit, ratelimit.AuthEmailIPWindow) {
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		writeError(w, http.StatusBadRequest, "email required")
		return
	}
	if a.rateLimited(w, "auth-email:addr:"+email, ratelimit.AuthEmailAddrLimit, ratelimit.AuthEmailAddrWindow) {
		return
	}

	token, actor, err := a.auth.IssueMagicLinkToken(r.Context(), email)
	if errors.Is(err, auth.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "message": authEmailSentMsg})
		return
	}
	if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "valid email required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	signInURL := strings.TrimRight(a.siteCfg.Site.URL, "/") + "/auth/magic?token=" + token
	if err := a.notify.SendMagicLink(r.Context(), actor.Email, actor.DisplayName, signInURL); err != nil {
		writeError(w, http.StatusInternalServerError, "could not send email")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "message": authEmailSentMsg})
}

func (a *API) magicLinkConsume(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "token required")
			return
		}
		token = strings.TrimSpace(req.Token)
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token required")
		return
	}

	actor, sessionToken, err := a.auth.LoginWithMagicLink(r.Context(), token)
	if errors.Is(err, auth.ErrInvalidToken) {
		writeError(w, http.StatusBadRequest, "link expired or already used — request a new one")
		return
	}
	if a.respondIfBanned(w, r, actor, err) {
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	setSessionCookie(w, sessionToken)
	enriched, ok := a.enrichActor(w, r, actor)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.authResponse(r.Context(), enriched, sessionToken))
}