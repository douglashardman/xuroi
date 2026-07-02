package handlers

import (
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/media"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
)

func (a *API) uploadMedia(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	if a.rateLimited(w, "media:actor:"+actor.ID, 12, ratelimit.PostActorWindow) {
		return
	}

	if err := r.ParseMultipartForm(media.MaxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file required")
		return
	}
	defer file.Close()

	if header.Size > media.MaxUploadBytes {
		writeError(w, http.StatusBadRequest, "file too large (max 12MB)")
		return
	}

	result, err := a.media.SaveUpload(file)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "too large") || strings.Contains(msg, "unsupported") {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		writeError(w, http.StatusInternalServerError, msg)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (a *API) serveMedia(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	f, err := a.media.Open(name)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read error")
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, name, stat.ModTime(), f)
}