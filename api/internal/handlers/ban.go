package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/netutil"
)

func writeBanError(w http.ResponseWriter, info auth.BanInfo) {
	out := map[string]any{
		"error":      "account banned",
		"ban_reason": info.Reason,
	}
	if info.BannedUntil != nil {
		out["banned_until"] = info.BannedUntil.Format(time.RFC3339)
	}
	if info.BannedByName != "" {
		out["banned_by"] = info.BannedByName
	}
	if info.Duration != "" {
		out["ban_duration"] = info.Duration
	}
	if info.IsIPBan {
		out["is_ip_ban"] = true
	}
	writeJSON(w, http.StatusForbidden, out)
}

func (a *API) checkRequestBanned(w http.ResponseWriter, r *http.Request) (auth.BanInfo, bool) {
	ip := netutil.ClientIP(r)
	if info, err := a.auth.CheckIPBanned(r.Context(), ip); errors.Is(err, auth.ErrBanned) {
		writeBanError(w, info)
		return info, true
	}
	return auth.BanInfo{}, false
}

func (a *API) banStatus(w http.ResponseWriter, r *http.Request) {
	ip := netutil.ClientIP(r)
	if info, banned, err := a.auth.IPBanInfo(r.Context(), ip); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if banned {
		writeJSON(w, http.StatusOK, map[string]any{"banned": true, "ban": info})
		return
	}

	if actor, err := a.actorFromRequest(r); err == nil {
		if details, err := a.auth.CheckSignInAllowedWithInfo(r.Context(), actor.ID); errors.Is(err, auth.ErrBanned) {
			writeJSON(w, http.StatusOK, map[string]any{"banned": true, "ban": details})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"banned": false})
}

func (a *API) withIPBanCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/health" || path == "/v1/health" || path == "/v1/auth/ban-status" {
			next.ServeHTTP(w, r)
			return
		}
		if _, blocked := a.checkRequestBanned(w, r); blocked {
			return
		}
		next.ServeHTTP(w, r)
	})
}