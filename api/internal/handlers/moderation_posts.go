package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/markdown"
	"github.com/xuroi/xuroi/api/internal/service"
)

func (a *API) staffPurgePost(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	postID := r.PathValue("id")
	if err := a.forum.StaffHardDeletePost(r.Context(), postID, staff.ID); err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "purged", "post_id": postID})
}

func (a *API) staffRestorePost(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	postID := r.PathValue("id")
	evt, err := a.forum.RestorePost(r.Context(), postID, staff.ID)
	if err != nil {
		if err.Error() == "deleted post not found" {
			writeError(w, http.StatusNotFound, "deleted post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "restored", "event": evt})
}

func (a *API) staffRestoreThread(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	threadID := r.PathValue("id")
	evt, err := a.forum.RestoreThread(r.Context(), threadID, staff.ID)
	if err != nil {
		if err.Error() == "deleted thread not found" {
			writeError(w, http.StatusNotFound, "deleted thread not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "restored", "event": evt})
}

func (a *API) staffEditPost(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	postID := r.PathValue("id")
	var req struct {
		BodyMarkdown string `json:"body_markdown"`
		BodyHTML     string `json:"body_html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.BodyMarkdown = strings.TrimSpace(req.BodyMarkdown)
	if req.BodyHTML == "" {
		req.BodyHTML = markdown.ToHTML(req.BodyMarkdown)
	}
	evt, err := a.forum.StaffEditPost(r.Context(), service.EditPostInput{
		PostID:       postID,
		EditorID:     staff.ID,
		BodyMarkdown: req.BodyMarkdown,
		BodyHTML:     req.BodyHTML,
	})
	if err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		if err.Error() == "body required" {
			writeError(w, http.StatusBadRequest, "body_markdown required")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	post, perr := a.reader.PostByID(r.Context(), postID, &staff.ID, staff.IsAdmin)
	if perr != nil {
		writeJSON(w, http.StatusOK, map[string]any{"event": evt})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event": evt, "post": post})
}