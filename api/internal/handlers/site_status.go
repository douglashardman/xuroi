package handlers

import "net/http"

func (a *API) getSiteStatus(w http.ResponseWriter, r *http.Request) {
	maintenance := a.siteCfg.Maintenance.Normalized()
	writeJSON(w, http.StatusOK, map[string]any{
		"maintenance": map[string]any{
			"enabled": maintenance.Enabled,
			"message": maintenance.Message,
		},
	})
}