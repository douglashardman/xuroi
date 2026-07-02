package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
	"github.com/xuroi/xuroi/api/internal/service"
)

func (a *API) reportThread(w http.ResponseWriter, r *http.Request) {
	threadID := r.PathValue("id")
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a.rateLimited(w, "report:actor:"+actor.ID, 10, ratelimit.PostActorWindow) {
		return
	}

	var req struct {
		ReasonID string `json:"reason_id"`
		Detail   string `json:"detail"`
		Reason   string `json:"reason"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	reason := strings.TrimSpace(req.Reason)
	if req.ReasonID != "" {
		formatted, err := a.siteCfg.Moderation.FormatReportReason(req.ReasonID, req.Detail)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		reason = formatted
	} else if reason == "" && len(a.siteCfg.Moderation.Normalized().ReportReasons) > 0 {
		writeError(w, http.StatusBadRequest, "report reason required")
		return
	}

	evt, err := a.forum.ReportThread(r.Context(), threadID, actor.ID, reason)
	if errors.Is(err, service.ErrAlreadyReported) {
		writeError(w, http.StatusConflict, "you already reported this thread")
		return
	}
	if err != nil {
		if err.Error() == "thread not found" {
			writeError(w, http.StatusNotFound, "thread not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var payload events.ThreadReported
	_ = json.Unmarshal(evt.Payload, &payload)
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":    "reported",
		"report_id": payload.ReportID,
	})
}