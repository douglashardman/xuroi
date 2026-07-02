package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (a *API) mergeThread(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}

	sourceID := r.PathValue("id")
	var req struct {
		TargetThreadID string `json:"target_thread_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	targetID := strings.TrimSpace(req.TargetThreadID)
	if targetID == "" {
		writeError(w, http.StatusBadRequest, "target_thread_id required")
		return
	}

	evt, err := a.forum.MergeThreads(r.Context(), sourceID, targetID, staff.ID)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			writeError(w, http.StatusNotFound, msg)
			return
		}
		if strings.Contains(msg, "cannot merge") || strings.Contains(msg, "no posts") {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		writeError(w, http.StatusInternalServerError, msg)
		return
	}

	writeJSON(w, http.StatusOK, evt)
}