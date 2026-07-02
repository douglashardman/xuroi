package handlers

import (
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/service"
)

func (a *API) deleteThread(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	threadID := r.PathValue("id")

	if actor.IsModerator || actor.IsAdmin {
		evt, err := a.forum.DeleteThread(r.Context(), threadID, actor.ID, false)
		if err != nil {
			if err.Error() == "thread not found" {
				writeError(w, http.StatusNotFound, "thread not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, evt)
		return
	}

	evt, err := a.forum.DeleteThread(r.Context(), threadID, actor.ID, true)
	if errors.Is(err, service.ErrForbiddenEdit) {
		writeError(w, http.StatusForbidden, "cannot delete this thread")
		return
	}
	if errors.Is(err, service.ErrThreadLocked) {
		writeError(w, http.StatusForbidden, "thread is locked")
		return
	}
	if errors.Is(err, service.ErrEditWindowClosed) {
		writeError(w, http.StatusForbidden, "delete window has closed")
		return
	}
	if errors.Is(err, service.ErrEditDisabled) {
		writeError(w, http.StatusForbidden, "thread deletion disabled")
		return
	}
	if err != nil {
		msg := err.Error()
		if msg == "thread not found" {
			writeError(w, http.StatusNotFound, "thread not found")
			return
		}
		if msg == "cannot delete thread after replies" {
			writeError(w, http.StatusForbidden, msg)
			return
		}
		writeError(w, http.StatusInternalServerError, msg)
		return
	}
	writeJSON(w, http.StatusOK, evt)
}