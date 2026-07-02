package handlers

import (
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/media"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
)

func (a *API) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	if a.rateLimited(w, "avatar:actor:"+actor.ID, 6, ratelimit.PostActorWindow) {
		return
	}

	if err := r.ParseMultipartForm(media.MaxAvatarBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file required")
		return
	}
	defer file.Close()

	if header.Size > media.MaxAvatarBytes {
		writeError(w, http.StatusBadRequest, "file too large (max 4MB)")
		return
	}

	result, err := a.media.SaveAvatar(file)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "too large") || strings.Contains(msg, "unsupported") || strings.Contains(msg, "too small") {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		writeError(w, http.StatusInternalServerError, msg)
		return
	}

	var oldURL *string
	_ = a.pool.QueryRow(r.Context(), `SELECT avatar_url FROM actors WHERE id = $1`, actor.ID).Scan(&oldURL)

	_, err = a.pool.Exec(r.Context(), `UPDATE actors SET avatar_url = $1 WHERE id = $2`, result.URL, actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if oldURL != nil && *oldURL != "" && *oldURL != result.URL {
		_ = a.media.DeleteAvatarFiles(*oldURL)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"avatar_url":    result.URL,
		"avatar_sm_url": result.SmURL,
	})
}

func (a *API) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}

	var oldURL *string
	_ = a.pool.QueryRow(r.Context(), `SELECT avatar_url FROM actors WHERE id = $1`, actor.ID).Scan(&oldURL)

	_, err := a.pool.Exec(r.Context(), `UPDATE actors SET avatar_url = NULL WHERE id = $1`, actor.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if oldURL != nil && *oldURL != "" {
		_ = a.media.DeleteAvatarFiles(*oldURL)
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}