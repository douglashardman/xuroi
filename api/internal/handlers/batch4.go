package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/query"
)

var validTimezones = map[string]struct{}{
	"":                    {},
	"America/New_York":    {},
	"America/Chicago":     {},
	"America/Denver":      {},
	"America/Los_Angeles": {},
	"America/Phoenix":     {},
	"America/Anchorage":   {},
	"Pacific/Honolulu":    {},
	"Europe/London":       {},
	"Europe/Paris":        {},
	"Europe/Berlin":       {},
	"Asia/Tokyo":          {},
	"Asia/Singapore":      {},
	"Australia/Sydney":    {},
	"UTC":                 {},
}

func (a *API) getCategoryEmailWatch(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	viewer, err := a.viewerFromActor(r, actor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	slug := r.PathValue("slug")
	page, err := a.reader.CategoryBySlug(r.Context(), slug, 1, 1, viewer)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			writeError(w, http.StatusNotFound, "forum not found")
			return
		}
		if errors.Is(err, query.ErrForbidden) {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	watching, err := a.reader.CategoryEmailWatching(r.Context(), actor.ID, page.Category.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"watching": watching})
}

func (a *API) setCategoryEmailWatch(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	viewer, err := a.viewerFromActor(r, actor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	slug := r.PathValue("slug")
	page, err := a.reader.CategoryBySlug(r.Context(), slug, 1, 1, viewer)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			writeError(w, http.StatusNotFound, "forum not found")
			return
		}
		if errors.Is(err, query.ErrForbidden) {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.reader.SetCategoryEmailWatch(r.Context(), actor.ID, page.Category.ID, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"watching": req.Enabled})
}

func (a *API) getMyTimezone(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	tz, err := a.reader.ActorTimezone(r.Context(), actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"timezone": tz})
}

func (a *API) setMyTimezone(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	tz := strings.TrimSpace(req.Timezone)
	if _, ok := validTimezones[tz]; !ok {
		writeError(w, http.StatusBadRequest, "unsupported timezone")
		return
	}
	if err := a.reader.SetActorTimezone(r.Context(), actor.ID, tz); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"timezone": tz})
}

func (a *API) blockActor(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	blockedID := strings.TrimSpace(r.PathValue("id"))
	if blockedID == "" || blockedID == actor.ID {
		writeError(w, http.StatusBadRequest, "invalid member")
		return
	}
	if err := a.reader.BlockActor(r.Context(), actor.ID, blockedID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "blocked"})
}

func (a *API) unblockActor(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	blockedID := strings.TrimSpace(r.PathValue("id"))
	if blockedID == "" {
		writeError(w, http.StatusBadRequest, "invalid member")
		return
	}
	if err := a.reader.UnblockActor(r.Context(), actor.ID, blockedID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unblocked"})
}

func (a *API) profileBlockFlags(ctx context.Context, viewer auth.Actor, profileID string) (blockedByMe bool, blocksMe bool) {
	if viewer.ID == profileID {
		return false, false
	}
	blockedByMe, _ = a.reader.IsBlocked(ctx, viewer.ID, profileID)
	blocksMe, _ = a.reader.IsBlocked(ctx, profileID, viewer.ID)
	return blockedByMe, blocksMe
}