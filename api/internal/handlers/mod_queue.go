package handlers

import (
	"net/http"

	"github.com/xuroi/xuroi/api/internal/query"
)

func (a *API) listModQueue(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireStaff(w, r); !ok {
		return
	}
	items, err := a.reader.ListModQueue(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []query.ModQueueItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

func (a *API) approvePost(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	postID := r.PathValue("id")
	evt, err := a.forum.ModeratePost(r.Context(), postID, staff.ID, "approved")
	if err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found or not pending")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, evt)
}

func (a *API) rejectPost(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	postID := r.PathValue("id")
	evt, err := a.forum.ModeratePost(r.Context(), postID, staff.ID, "rejected")
	if err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found or not pending")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, evt)
}