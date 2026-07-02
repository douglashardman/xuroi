package handlers

import (
	"encoding/json"
	"net/http"
)

func (a *API) getEmailPreferences(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	prefs, err := a.notify.GetEmailPreferences(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (a *API) setEmailPreferences(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		ThreadRepliesEnabled *bool `json:"thread_replies_enabled"`
		MentionsEnabled      *bool `json:"mentions_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ThreadRepliesEnabled == nil && req.MentionsEnabled == nil {
		writeError(w, http.StatusBadRequest, "at least one preference required")
		return
	}

	prefs, err := a.notify.GetEmailPreferences(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.ThreadRepliesEnabled != nil {
		prefs.ThreadRepliesEnabled = *req.ThreadRepliesEnabled
	}
	if req.MentionsEnabled != nil {
		prefs.MentionsEnabled = *req.MentionsEnabled
	}

	if err := a.notify.SetEmailPreferences(r.Context(), actor.ID, prefs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}