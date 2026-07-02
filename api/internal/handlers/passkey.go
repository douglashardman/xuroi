package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/netutil"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
)

func requestOrigin(r *http.Request, bodyOrigin string) string {
	if o := strings.TrimSpace(bodyOrigin); o != "" {
		return o
	}
	if o := r.Header.Get("Origin"); o != "" {
		return o
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Host != "" {
			return u.Scheme + "://" + u.Host
		}
	}
	return ""
}

func (a *API) passkeySignupBegin(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if a.rateLimited(w, "register:ip:"+ip, ratelimit.RegisterIPLimit, ratelimit.RegisterIPWindow) {
		return
	}

	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Origin      string `json:"origin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	resp, err := a.auth.BeginPasskeySignup(r.Context(), req.Email, req.DisplayName, requestOrigin(r, req.Origin))
	if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "display_name and valid email required")
		return
	}
	if errors.Is(err, auth.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if errors.Is(err, auth.ErrDisplayNameReserved) {
		writeError(w, http.StatusConflict, "display name is reserved")
		return
	}
	if errors.Is(err, auth.ErrDisplayNameTaken) {
		writeError(w, http.StatusConflict, "display name already taken")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) passkeySignupFinish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.SessionID == "" || len(req.Credential) == 0 {
		writeError(w, http.StatusBadRequest, "session_id and credential required")
		return
	}

	actor, token, err := a.auth.FinishPasskeySignup(r.Context(), req.SessionID, req.Credential)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusBadRequest, "passkey session expired — try again")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "passkey registration failed")
		return
	}

	setSessionCookie(w, token)
	enriched, ok := a.enrichActor(w, r, actor)
	if !ok {
		return
	}
	_ = a.sendVerificationEmail(r.Context(), enriched.ID)
	writeJSON(w, http.StatusCreated, a.authResponse(r.Context(), enriched, token))
}

func (a *API) passkeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req struct {
		Origin string `json:"origin"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	resp, err := a.auth.BeginPasskeyRegister(r.Context(), actor.ID, requestOrigin(r, req.Origin))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) passkeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req struct {
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.SessionID == "" || len(req.Credential) == 0 {
		writeError(w, http.StatusBadRequest, "session_id and credential required")
		return
	}

	if err := a.auth.FinishPasskeyRegister(r.Context(), actor.ID, req.SessionID, req.Credential); err != nil {
		writeError(w, http.StatusBadRequest, "passkey registration failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "passkey_added"})
}

func (a *API) passkeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if a.rateLimited(w, "login:ip:"+ip, ratelimit.LoginIPLimit, ratelimit.LoginIPWindow) {
		return
	}

	var req struct {
		Origin string `json:"origin"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	resp, err := a.auth.BeginPasskeyLogin(r.Context(), requestOrigin(r, req.Origin))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) passkeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.SessionID == "" || len(req.Credential) == 0 {
		writeError(w, http.StatusBadRequest, "session_id and credential required")
		return
	}

	actor, token, err := a.auth.FinishPasskeyLogin(r.Context(), req.SessionID, req.Credential)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusBadRequest, "passkey session expired — try again")
		return
	}
	if a.respondIfBanned(w, r, actor, err) {
		return
	}
	if errors.Is(err, auth.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "passkey not recognized")
		return
	}
	if err != nil {
		writeError(w, http.StatusUnauthorized, "passkey sign-in failed")
		return
	}

	setSessionCookie(w, token)
	enriched, ok := a.enrichActor(w, r, actor)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.authResponse(r.Context(), enriched, token))
}