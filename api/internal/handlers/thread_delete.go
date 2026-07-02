package handlers

import (
	"net/http"
)

func (a *API) deleteThread(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	threadID := r.PathValue("id")
	evt, err := a.forum.DeleteThread(r.Context(), threadID, staff.ID)
	if err != nil {
		if err.Error() == "thread not found" {
			writeError(w, http.StatusNotFound, "thread not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, evt)
}