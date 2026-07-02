package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/markdown"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
	"github.com/xuroi/xuroi/api/internal/service"
)

func (a *API) editPost(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req struct {
		BodyMarkdown string `json:"body_markdown"`
		BodyHTML     string `json:"body_html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.BodyHTML == "" {
		req.BodyHTML = markdown.ToHTML(req.BodyMarkdown)
	}

	evt, err := a.forum.EditPost(r.Context(), service.EditPostInput{
		PostID:       postID,
		EditorID:     actor.ID,
		BodyMarkdown: req.BodyMarkdown,
		BodyHTML:     req.BodyHTML,
	})
	if errors.Is(err, service.ErrForbiddenEdit) {
		writeError(w, http.StatusForbidden, "cannot edit this post")
		return
	}
	if errors.Is(err, service.ErrEditWindowClosed) {
		writeError(w, http.StatusForbidden, "edit window closed")
		return
	}
	if errors.Is(err, service.ErrEditDisabled) {
		writeError(w, http.StatusForbidden, "post editing disabled")
		return
	}
	if errors.Is(err, service.ErrThreadLocked) {
		writeError(w, http.StatusForbidden, "thread is locked")
		return
	}
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

	post, perr := a.reader.PostByID(r.Context(), postID, &actor.ID, actor.IsAdmin)
	if perr != nil {
		writeJSON(w, http.StatusOK, evt)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      evt.ID,
		"type":    evt.Type,
		"payload": json.RawMessage(evt.Payload),
		"post":    post,
	})
}

func (a *API) reportPost(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a.rateLimited(w, "report:actor:"+actor.ID, 10, ratelimit.PostActorWindow) {
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	evt, err := a.forum.ReportPost(r.Context(), postID, actor.ID, req.Reason)
	if errors.Is(err, service.ErrAlreadyReported) {
		writeError(w, http.StatusConflict, "you already reported this post")
		return
	}
	if err != nil {
		if err.Error() == "post not found" {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var payload events.PostReported
	_ = json.Unmarshal(evt.Payload, &payload)
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":    "reported",
		"report_id": payload.ReportID,
	})
}