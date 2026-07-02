package handlers

import (
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/auth"
)

func (a *API) requireStaff(w http.ResponseWriter, r *http.Request) (auth.Actor, bool) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return auth.Actor{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return auth.Actor{}, false
	}
	enriched, ok := a.enrichActorFull(w, r, actor)
	if !ok {
		return auth.Actor{}, false
	}
	if !enriched.IsAdmin && !enriched.IsModerator {
		writeError(w, http.StatusForbidden, "moderator or admin required")
		return auth.Actor{}, false
	}
	return enriched, true
}

func (a *API) requireStaffActor(w http.ResponseWriter, r *http.Request, actor auth.Actor) bool {
	enriched, err := a.auth.EnrichActor(r.Context(), actor, a.siteCfg.Admin.Emails, a.siteCfg.Admin.ModeratorEmails, a.siteCfg.Admin.PermBanModeratorEmails)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if !enriched.IsAdmin && !enriched.IsModerator {
		writeError(w, http.StatusForbidden, "moderator or admin required")
		return false
	}
	return true
}