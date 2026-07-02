package handlers

import (
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/query"
)

func (a *API) markThreadRead(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	threadID := r.PathValue("id")
	if _, err := a.reader.ThreadMeta(r.Context(), threadID); errors.Is(err, query.ErrNotFound) {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.notify.MarkThreadRead(r.Context(), actor.ID, threadID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}