package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/service"
)

func (a *API) requireAdmin(w http.ResponseWriter, r *http.Request) (auth.Actor, bool) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return auth.Actor{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return auth.Actor{}, false
	}
	actor = a.auth.WithAccessFlags(actor, a.siteCfg.Admin.Emails, a.siteCfg.Admin.ModeratorEmails, a.siteCfg.Admin.PermBanModeratorEmails)
	if !actor.IsAdmin {
		writeError(w, http.StatusForbidden, "admin required")
		return auth.Actor{}, false
	}
	return actor, true
}

func (a *API) moderateThread(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireStaff(w, r); !ok {
		return
	}
	threadID := r.PathValue("id")
	var req struct {
		IsPinned *bool `json:"is_pinned"`
		IsLocked *bool `json:"is_locked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.IsPinned == nil && req.IsLocked == nil {
		writeError(w, http.StatusBadRequest, "is_pinned or is_locked required")
		return
	}

	evts, err := a.forum.ModerateThread(r.Context(), threadID, req.IsPinned, req.IsLocked)
	if err != nil {
		if err.Error() == "thread not found" {
			writeError(w, http.StatusNotFound, "thread not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": evts})
}

func (a *API) deletePost(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	actor = a.auth.WithAccessFlags(actor, a.siteCfg.Admin.Emails, a.siteCfg.Admin.ModeratorEmails, a.siteCfg.Admin.PermBanModeratorEmails)
	postID := r.PathValue("id")

	evt, err := a.forum.DeletePost(r.Context(), postID, actor.ID, actor.IsAdmin)
	if errors.Is(err, service.ErrForbiddenEdit) {
		writeError(w, http.StatusForbidden, "cannot delete this post")
		return
	}
	if errors.Is(err, service.ErrDeleteDisabled) {
		writeError(w, http.StatusForbidden, "post deletion disabled")
		return
	}
	if err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, evt)
}