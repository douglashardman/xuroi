package handlers

import (
	"net/http"
	"strconv"

	"github.com/xuroi/xuroi/api/internal/search"
)

func (a *API) searchContent(w http.ResponseWriter, r *http.Request) {
	limit := 20
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
	resp, err := search.Search(r.Context(), a.pool, search.Options{
		Query:        r.URL.Query().Get("q"),
		Author:       r.URL.Query().Get("author"),
		CategorySlug: r.URL.Query().Get("category"),
		Limit:        limit,
	}, viewer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}