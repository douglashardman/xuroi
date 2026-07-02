package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/xuroi/xuroi/api/internal/friends"
)

func (a *API) listFriendRequests(w http.ResponseWriter, r *http.Request) {
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
	incoming, err := a.friends.ListIncoming(r.Context(), actor.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	outgoing, err := a.friends.ListOutgoing(r.Context(), actor.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if incoming == nil {
		incoming = []friends.Request{}
	}
	if outgoing == nil {
		outgoing = []friends.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"incoming": incoming,
		"outgoing": outgoing,
	})
}

func (a *API) sendFriendRequest(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		ToActorID   string `json:"to_actor_id"`
		ToActorSlug string `json:"to_actor_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	toID := req.ToActorID
	if toID == "" && req.ToActorSlug != "" {
		id, err := a.dm.FindActorBySlug(r.Context(), req.ToActorSlug)
		if err != nil {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		toID = id
	}
	if toID == "" {
		writeError(w, http.StatusBadRequest, "to_actor_id or to_actor_slug required")
		return
	}
	reqID, err := a.friends.SendRequest(r.Context(), actor.ID, toID)
	if errors.Is(err, friends.ErrRequestPending) {
		writeJSON(w, http.StatusOK, map[string]string{"request_id": reqID, "status": "pending"})
		return
	}
	if mapFriendError(w, err) {
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	status := "pending"
	if ok, ferr := a.friends.AreFriends(r.Context(), actor.ID, toID); ferr == nil && ok {
		status = "accepted"
	}
	writeJSON(w, http.StatusCreated, map[string]string{"request_id": reqID, "status": status})
}

func (a *API) acceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	if err := a.friends.AcceptRequest(r.Context(), actor.ID, r.PathValue("id")); err != nil {
		if mapFriendError(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (a *API) declineFriendRequest(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	if err := a.friends.DeclineRequest(r.Context(), actor.ID, r.PathValue("id")); err != nil {
		if mapFriendError(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "declined"})
}

func mapFriendError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, friends.ErrSelfFriend):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, friends.ErrAlreadyFriends):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, friends.ErrRequestPending):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, friends.ErrRequestNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, friends.ErrRequestNotYours):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, friends.ErrMemberNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		return false
	}
	return true
}