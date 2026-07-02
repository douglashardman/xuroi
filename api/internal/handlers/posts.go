package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/events"
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
	var mentioned []string
	req.BodyMarkdown, mentioned = a.processPostMentions(r, req.BodyMarkdown, actor.ID)
	if req.BodyHTML == "" {
		req.BodyHTML = a.renderPostHTML(req.BodyMarkdown)
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

	var edited struct {
		ThreadID string `json:"thread_id"`
	}
	_ = json.Unmarshal(evt.Payload, &edited)
	if edited.ThreadID != "" && a.notify != nil {
		_ = a.notify.SyncPostMentions(r.Context(), postID, edited.ThreadID, actor.ID, mentioned)
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
		ReasonID string `json:"reason_id"`
		Detail   string `json:"detail"`
		Reason   string `json:"reason"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	reason := strings.TrimSpace(req.Reason)
	if req.ReasonID != "" {
		formatted, err := a.siteCfg.Moderation.FormatReportReason(req.ReasonID, req.Detail)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		reason = formatted
	} else if reason == "" && len(a.siteCfg.Moderation.Normalized().ReportReasons) > 0 {
		writeError(w, http.StatusBadRequest, "report reason required")
		return
	}

	evt, err := a.forum.ReportPost(r.Context(), postID, actor.ID, reason)
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

func (a *API) deletePost(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}

	evt, err := a.forum.DeletePost(r.Context(), postID, staff.ID, staff.IsAdmin)
	if errors.Is(err, service.ErrDeleteDisabled) {
		writeError(w, http.StatusForbidden, "post deletion disabled")
		return
	}
	if errors.Is(err, service.ErrForbiddenEdit) {
		writeError(w, http.StatusForbidden, "cannot delete this post")
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
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      evt.ID,
		"type":    evt.Type,
		"payload": json.RawMessage(evt.Payload),
	})
}

func (a *API) moderateThread(w http.ResponseWriter, r *http.Request) {
	threadID := r.PathValue("id")
	actor, ok := a.requireStaff(w, r)
	if !ok {
		return
	}

	var req struct {
		IsPinned    *bool   `json:"is_pinned"`
		IsLocked    *bool   `json:"is_locked"`
		LockReason  string  `json:"lock_reason"`
		CategoryID  *string `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.CategoryID != nil && strings.TrimSpace(*req.CategoryID) != "" {
		evt, err := a.forum.MoveThread(r.Context(), threadID, strings.TrimSpace(*req.CategoryID), actor.ID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": []any{evt}})
		return
	}
	if req.IsPinned == nil && req.IsLocked == nil {
		writeError(w, http.StatusBadRequest, "is_pinned, is_locked, or category_id required")
		return
	}

	events, err := a.forum.ModerateThread(r.Context(), threadID, req.IsPinned, req.IsLocked, req.LockReason)
	if err != nil {
		if err.Error() == "thread not found" {
			writeError(w, http.StatusNotFound, "thread not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}