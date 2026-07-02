package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/mentions"
	"github.com/xuroi/xuroi/api/internal/query"
)

func (a *API) listNotifications(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	items, err := a.reader.ListNotifications(r.Context(), actor.ID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []query.Notification{}
	}

	unread, err := a.reader.NotificationUnreadCount(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": items,
		"unread_count":  unread,
	})
}

func (a *API) notificationUnreadCount(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	n, err := a.reader.NotificationUnreadCount(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"unread_count": n})
}

func (a *API) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	id := r.PathValue("id")
	ok, err := a.reader.MarkNotificationRead(r.Context(), actor.ID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	}

	unread, _ := a.reader.NotificationUnreadCount(r.Context(), actor.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "unread_count": unread})
}

func (a *API) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	n, err := a.reader.MarkAllNotificationsRead(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "marked": n, "unread_count": 0})
}

func (a *API) processPostMentions(r *http.Request, bodyMarkdown, authorID string) (markdown string, mentioned []string) {
	markdown = bodyMarkdown
	idx, err := mentions.LoadIndex(r.Context(), a.pool)
	if err != nil {
		return markdown, nil
	}
	result := mentions.Expand(bodyMarkdown, idx)
	return result.Markdown, mentions.FilterSelf(result.ActorIDs, authorID)
}