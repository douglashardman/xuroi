package handlers

import (
	"errors"
	"net/http"
	"strconv"

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
	writeJSON(w, http.StatusOK, profile)
}

func (a *API) listRecentThreads(w http.ResponseWriter, r *http.Request) {
	limit := 6
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	viewer, err := a.viewerFromRequest(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, err := a.reader.RecentThreads(r.Context(), limit, viewer)
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