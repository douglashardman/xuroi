package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/netutil"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
)

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if a.rateLimited(w, "register:ip:"+ip, ratelimit.RegisterIPLimit, ratelimit.RegisterIPWindow) {
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	actor, token, err := a.auth.Register(r.Context(), auth.RegisterInput{
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Password:    req.Password,
	})
	if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "display_name, valid email, and password (8+ chars) required")
		return
	}
	if errors.Is(err, auth.ErrInvalidPassword) {
		writeError(w, http.StatusBadRequest, "password must be 8–128 characters")
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

	setSessionCookie(w, token)
	enriched, ok := a.enrichActor(w, r, actor)
	if !ok {
		return
	}
	_ = a.sendVerificationEmail(r.Context(), enriched.ID)
	writeJSON(w, http.StatusCreated, a.authResponse(r.Context(), enriched, token))
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if a.rateLimited(w, "login:ip:"+ip, ratelimit.LoginIPLimit, ratelimit.LoginIPWindow) {
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	emailKey := "login:fail:" + strings.ToLower(strings.TrimSpace(req.Email))
	if a.blocked(w, emailKey, ratelimit.LoginFailLimit, ratelimit.LoginFailWindow) {
		return
	}

	var actor auth.Actor
	var token string
	var err error
	if strings.TrimSpace(req.Password) != "" {
		actor, token, err = a.auth.LoginWithPassword(r.Context(), req.Email, req.Password)
	} else {
		actor, token, err = a.auth.LoginLegacy(r.Context(), req.Email)
	}
	if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "email and password required")
		return
	}
	if errors.Is(err, auth.ErrWrongPassword) || errors.Is(err, auth.ErrNoPassword) {
		if a.limiter != nil {
			a.limiter.Hit(emailKey, ratelimit.LoginFailWindow)
		}
		if errors.Is(err, auth.ErrNoPassword) {
			writeError(w, http.StatusUnauthorized, "use your passkey or set a password on your account")
		} else {
			writeError(w, http.StatusUnauthorized, "incorrect email or password")
		}
		return
	}
	if a.respondIfBanned(w, r, actor, err) {
		return
	}
	if errors.Is(err, auth.ErrNotFound) {
		if a.limiter != nil {
			a.limiter.Hit(emailKey, ratelimit.LoginFailWindow)
		}
		writeError(w, http.StatusNotFound, "no account with that email")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	setSessionCookie(w, token)
	enriched, ok := a.enrichActor(w, r, actor)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.authResponse(r.Context(), enriched, token))
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Session-Token")
	if token == "" {
		if c, err := r.Cookie(auth.CookieName); err == nil {
			token = c.Value
		}
	}
	if token != "" {
		_ = a.auth.Logout(r.Context(), token)
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "not signed in")
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
	hasPassword, hasPasskey, _ := a.auth.AuthMethods(r.Context(), enriched.ID)
	out := a.actorJSON(r.Context(), enriched)
	out["has_password"] = hasPassword
	out["has_passkey"] = hasPasskey
	writeJSON(w, http.StatusOK, out)
}

func (a *API) setPassword(w http.ResponseWriter, r *http.Request) {
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
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := a.auth.SetPassword(r.Context(), actor.ID, req.Password); errors.Is(err, auth.ErrInvalidPassword) {
		writeError(w, http.StatusBadRequest, "password must be 8–128 characters")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "password_set"})
}

func (a *API) optionalActorID(r *http.Request) *string {
	actor, err := a.actorFromRequest(r)
	if err != nil {
		return nil
	}
	return &actor.ID
}

func (a *API) actorFromRequest(r *http.Request) (auth.Actor, error) {
	if token := r.Header.Get("X-Session-Token"); token != "" {
		return a.auth.ActorFromToken(r.Context(), token)
	}
	c, err := r.Cookie(auth.CookieName)
	if err != nil || c.Value == "" {
		return auth.Actor{}, auth.ErrInvalidSession
	}
	return a.auth.ActorFromToken(r.Context(), c.Value)
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   auth.SessionDays * 24 * 60 * 60,
		Expires:  time.Now().Add(auth.SessionDays * 24 * time.Hour),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}