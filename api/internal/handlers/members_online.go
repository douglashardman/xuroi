package handlers

import (
	"encoding/json"
	"net/http"
)

func (a *API) listOnlineMembers(w http.ResponseWriter, r *http.Request) {
	staffView := false
	if actor, err := a.actorFromRequest(r); err == nil {
		enriched, err := a.auth.EnrichActor(r.Context(), actor, a.siteCfg.Admin.Emails, a.siteCfg.Admin.ModeratorEmails, a.siteCfg.Admin.PermBanModeratorEmails)
		if err == nil && (enriched.IsAdmin || enriched.IsModerator) {
			staffView = true
		}
	}
	resp, err := a.reader.OnlineMembers(r.Context(), staffView)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) getOnlinePrivacy(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	hidden, err := a.reader.ActorHideOnline(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"hide_online": hidden})
}

func (a *API) setOnlinePrivacy(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	var req struct {
		HideOnline bool `json:"hide_online"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := a.reader.SetActorHideOnline(r.Context(), actor.ID, req.HideOnline); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"hide_online": req.HideOnline})
}