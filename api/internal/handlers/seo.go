package handlers

import (
	"net/http"
)

func (a *API) getSitemap(w http.ResponseWriter, r *http.Request) {
	entries, err := a.reader.SitemapEntries(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"site":    a.siteCfg.Site,
		"entries": entries,
	})
}