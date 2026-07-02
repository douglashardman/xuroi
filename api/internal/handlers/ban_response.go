package handlers

import (
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/auth"
)

func (a *API) respondIfBanned(w http.ResponseWriter, r *http.Request, actor auth.Actor, err error) bool {
	if !errors.Is(err, auth.ErrBanned) {
		return false
	}
	if actor.ID != "" {
		if info, derr := a.auth.ActorBanInfo(r.Context(), actor.ID); derr == nil {
			writeBanError(w, info)
			return true
		}
	}
	writeError(w, http.StatusForbidden, "account banned")
	return true
}