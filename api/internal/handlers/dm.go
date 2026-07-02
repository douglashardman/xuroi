package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/dm"
	"github.com/xuroi/xuroi/api/internal/policy"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
)

func (a *API) listDMConversations(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	items, err := a.dm.ListConversations(r.Context(), actor.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []dm.ConversationSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": items})
}

func (a *API) getDMConversation(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	convID := r.PathValue("id")
	page, err := a.dm.ListMessages(r.Context(), actor.ID, convID, 200)
	if errors.Is(err, dm.ErrNotParticipant) {
		writeError(w, http.StatusForbidden, "not your conversation")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) startDMConversation(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		RecipientID   string `json:"recipient_id"`
		RecipientSlug string `json:"recipient_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	recipientID := req.RecipientID
	if recipientID == "" && req.RecipientSlug != "" {
		id, err := a.dm.FindActorBySlug(r.Context(), req.RecipientSlug)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		recipientID = id
	}
	if recipientID == "" {
		writeError(w, http.StatusBadRequest, "recipient_id or recipient_slug required")
		return
	}
	if err := a.checkDMSenderPolicy(r.Context(), actor.ID, actor.IsModerator, actor.IsAdmin); err != nil {
		writeContentPolicyError(w, err)
		return
	}
	convID, err := a.dm.GetOrCreateConversation(r.Context(), actor.ID, recipientID)
	if mapDMError(w, err) {
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"conversation_id": convID, "url": "/messages/" + convID})
}

func (a *API) sendDMMessage(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	convID := r.PathValue("id")
	var req struct {
		BodyMarkdown string `json:"body_markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if a.rateLimited(w, "dm:actor:"+actor.ID, 30, ratelimit.PostActorWindow) {
		return
	}
	if err := a.checkDMSenderPolicy(r.Context(), actor.ID, actor.IsModerator, actor.IsAdmin); err != nil {
		writeContentPolicyError(w, err)
		return
	}
	msg, err := a.dm.SendMessage(r.Context(), actor.ID, convID, req.BodyMarkdown)
	if mapDMError(w, err) {
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (a *API) markDMRead(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	convID := r.PathValue("id")
	if err := a.dm.MarkRead(r.Context(), actor.ID, convID); err != nil {
		if errors.Is(err, dm.ErrNotParticipant) {
			writeError(w, http.StatusForbidden, "not your conversation")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) searchDMMembers(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	members, err := a.dm.SearchMessageable(r.Context(), actor.ID, q, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if members == nil {
		members = []dm.MemberHit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (a *API) getDMPrivacy(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	p, err := a.dm.Privacy(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"dm_privacy": p})
}

func (a *API) setDMPrivacy(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		DMPrivacy string `json:"dm_privacy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !dm.ValidPrivacy(req.DMPrivacy) {
		writeError(w, http.StatusBadRequest, "dm_privacy must be everyone, friends_only, or off")
		return
	}
	if err := a.dm.SetPrivacy(r.Context(), actor.ID, req.DMPrivacy); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"dm_privacy": req.DMPrivacy})
}

func mapDMError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, dm.ErrSelfDM):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, dm.ErrDMDisabled):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, dm.ErrDMFriendsOnly):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, dm.ErrSenderDMOff):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, dm.ErrNotParticipant):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, policy.ErrDMRestricted):
		writeError(w, http.StatusForbidden, err.Error())
	case err.Error() == "member not found":
		writeError(w, http.StatusNotFound, err.Error())
	case err.Error() == "message body required":
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		return false
	}
	return true
}