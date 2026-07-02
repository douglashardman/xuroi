package handlers

import (
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/auth"
)

func (a *API) togglePostReaction(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result, err := a.forum.ToggleReaction(r.Context(), postID, actor.ID)
	if err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}