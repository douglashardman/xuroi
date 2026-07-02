package handlers

import (
	"encoding/json"
	"net/http"
)

func (a *API) setAcceptedAnswer(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	threadID := r.PathValue("id")
	var req struct {
		PostID string `json:"post_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.PostID == "" {
		writeError(w, http.StatusBadRequest, "post_id required")
		return
	}
	evt, err := a.forum.SetAcceptedAnswer(r.Context(), threadID, req.PostID, staff.ID)
	if err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "set", "event": evt})
}

func (a *API) clearAcceptedAnswer(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	threadID := r.PathValue("id")
	evt, err := a.forum.ClearAcceptedAnswer(r.Context(), threadID, staff.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "cleared", "event": evt})
}