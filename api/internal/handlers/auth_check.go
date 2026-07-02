package handlers

import (
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/netutil"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
)

func (a *API) checkDisplayName(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if a.rateLimited(w, "check-name:ip:"+ip, 30, ratelimit.RegisterIPWindow) {
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}

	available, nameSlug, reason, err := a.auth.DisplayNameAvailable(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := map[string]any{
		"available": available,
		"slug":      nameSlug,
	}
	if !available && reason != "" {
		out["reason"] = string(reason)
	}
	writeJSON(w, http.StatusOK, out)
}