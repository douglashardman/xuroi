package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/auth"
)

func (a *API) exportMyData(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	export, err := a.reader.ExportUserData(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="xuroi-export.json"`)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(export)
}

func (a *API) logoutAllSessions(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	keepToken := ""
	if c, err := r.Cookie(auth.CookieName); err == nil {
		keepToken = c.Value
	}
	if keepToken == "" {
		keepToken = r.Header.Get("X-Session-Token")
	}
	n, err := a.auth.LogoutAllSessions(r.Context(), actor.ID, keepToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "logged_out_elsewhere",
		"sessions_revoked": n,
	})
}

func (a *API) deleteMyAccount(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		Confirm string `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Confirm) != "DELETE" {
		writeError(w, http.StatusBadRequest, `type DELETE to confirm`)
		return
	}
	if actor.IsAdmin {
		writeError(w, http.StatusForbidden, "admins cannot self-delete here")
		return
	}
	if err := a.auth.DeleteAccount(r.Context(), actor.ID); errors.Is(err, auth.ErrNotFound) {
		writeError(w, http.StatusNotFound, "account not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}