package handlers

import (
	"net/http"
)

func (a *API) listReportReasons(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"reasons": a.siteCfg.Moderation.Normalized().ReportReasons,
	})
}