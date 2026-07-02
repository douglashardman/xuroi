package handlers

import (
	"net/http"
)

func (a *API) staffRemovePost(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}

	postID := r.PathValue("id")
	result, err := a.forum.StaffRemovePost(r.Context(), postID, staff.ID)
	if err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "removed",
		"post_id":        result.PostID,
		"thread_id":      result.ThreadID,
		"thread_removed": result.ThreadRemoved,
	})
}