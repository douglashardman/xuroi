package handlers

import "net/http"

func (a *API) getSiteStatus(w http.ResponseWriter, r *http.Request) {
	maintenance := a.siteCfg.Maintenance.Normalized()
	notice := a.siteCfg.Notice.Normalized()
	writeJSON(w, http.StatusOK, map[string]any{
		"maintenance": map[string]any{
			"enabled": maintenance.Enabled,
			"message": maintenance.Message,
		},
		"notice": map[string]any{
			"enabled": notice.Enabled,
			"message": notice.Message,
		},
		"classifieds": a.siteCfg.Classifieds.Normalized(),
		"trust": map[string]any{
			"abuse_email":     a.siteCfg.Trust.Normalized().AbuseEmail,
			"abuse_note":      a.siteCfg.Trust.Normalized().AbuseNote,
			"dmca_agent_name": a.siteCfg.Trust.Normalized().DMCAAgentName,
			"dmca_agent_note": a.siteCfg.Trust.Normalized().DMCAAgentNote,
		},
	})
}