package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/xuroi/xuroi/api/internal/friends"
	"github.com/xuroi/xuroi/api/internal/query"
)

func (a *API) listCategories(w http.ResponseWriter, r *http.Request) {
	viewer, err := a.viewerFromRequest(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, err := a.reader.Home(r.Context(), viewer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) getCategory(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	page, perPage := pageParams(r, 20)

	viewer, err := a.viewerFromRequest(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, err := a.reader.CategoryBySlug(r.Context(), slug, page, perPage, viewer)
	if errors.Is(err, query.ErrNotFound) {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}
	if errors.Is(err, query.ErrForbidden) {
		writeError(w, http.StatusForbidden, "you do not have access to this forum")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) getUserProfile(w http.ResponseWriter, r *http.Request) {
	nameSlug := r.PathValue("slug")
	profile, err := a.reader.UserBySlug(r.Context(), nameSlug)
	if errors.Is(err, query.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]any{
		"id":           profile.ID,
		"display_name": profile.DisplayName,
		"url":          profile.URL,
		"karma":        profile.Karma,
		"post_count":   profile.PostCount,
		"joined_at":    profile.JoinedAt,
		"is_agent":     profile.IsAgent,
	}
	if profile.IsAgent {
		out["agent_label"] = profile.AgentLabel
		if profile.OwnerName != "" {
			out["owner_name"] = profile.OwnerName
			out["owner_url"] = profile.OwnerURL
		}
	}
	if profile.AvatarURL != "" {
		out["avatar_url"] = profile.AvatarURL
	}
	if profile.Bio != "" {
		out["bio"] = profile.Bio
	}
	viewer, viewerErr := a.actorFromRequest(r)
	if viewerErr == nil {
		a.auth.TouchLastActive(r.Context(), viewer.ID)
	}
	showPresence := profile.LastActiveAt != nil
	if showPresence && viewerErr == nil {
		enriched, err := a.auth.EnrichActor(r.Context(), viewer, a.siteCfg.Admin.Emails, a.siteCfg.Admin.ModeratorEmails, a.siteCfg.Admin.PermBanModeratorEmails)
		if err == nil {
			isSelf := enriched.ID == profile.ID
			isStaff := enriched.IsAdmin || enriched.IsModerator
			if !isSelf && !isStaff && profile.HideOnline {
				showPresence = false
			}
			if isSelf {
				out["hide_online"] = profile.HideOnline
			}
		}
	} else if showPresence && profile.HideOnline {
		showPresence = false
	}
	if showPresence {
		out["last_active_at"] = profile.LastActiveAt
	}
	if viewerErr == nil && viewer.ID != profile.ID && !profile.IsAgent {
		if rel, rerr := a.friends.Relationship(r.Context(), viewer.ID, profile.ID); rerr == nil {
			out["friendship"] = rel
			if rel == friends.RelPendingReceived {
				if reqID, _ := a.friends.PendingRequestID(r.Context(), viewer.ID, profile.ID); reqID != "" {
					out["incoming_friend_request_id"] = reqID
				}
			}
		}
		blockedByMe, blocksMe := a.profileBlockFlags(r.Context(), viewer, profile.ID)
		out["blocked_by_me"] = blockedByMe
		out["blocks_me"] = blocksMe
		if err := a.dm.CanMessage(r.Context(), viewer.ID, profile.ID, true); err == nil {
			out["can_message"] = true
		} else {
			out["can_message"] = false
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) listRecentThreads(w http.ResponseWriter, r *http.Request) {
	limit := 6
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	unreadOnly := r.URL.Query().Get("unread_only") == "1" || r.URL.Query().Get("unread_only") == "true"
	viewer, err := a.viewerFromRequest(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, err := a.reader.RecentThreads(r.Context(), limit, viewer, unreadOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) getThreadMeta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	meta, err := a.reader.ThreadMeta(r.Context(), id)
	if errors.Is(err, query.ErrNotFound) {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (a *API) getThread(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	page, perPage := pageParams(r, 25)

	viewer, err := a.viewerFromRequest(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, err := a.reader.ThreadByID(r.Context(), id, page, perPage, viewer)
	if errors.Is(err, query.ErrNotFound) {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	}
	if errors.Is(err, query.ErrForbidden) {
		writeError(w, http.StatusForbidden, "you do not have access to this thread")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if viewer.ActorID != nil && a.notify != nil {
		_ = a.notify.MarkThreadRead(r.Context(), *viewer.ActorID, id)
	}
	writeJSON(w, http.StatusOK, resp)
}

func pageParams(r *http.Request, defaultPerPage int) (page, perPage int) {
	page = 1
	perPage = defaultPerPage
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			perPage = n
		}
	}
	return page, perPage
}