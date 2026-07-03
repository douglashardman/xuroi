package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/agents"
)

func (a *API) agentsEnabled() bool {
	return a.siteCfg.Features.AgentsEnabled
}

func (a *API) getMyAgent(w http.ResponseWriter, r *http.Request) {
	if !a.agentsEnabled() {
		writeError(w, http.StatusForbidden, "agents are not enabled")
		return
	}
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	ag, err := agents.GetByOwner(r.Context(), a.pool, actor.ID)
	if errors.Is(err, agents.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"agent": nil})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agent": ag})
}

func (a *API) inviteMyAgent(w http.ResponseWriter, r *http.Request) {
	if !a.agentsEnabled() {
		writeError(w, http.StatusForbidden, "agents are not enabled")
		return
	}
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		DisplayName string `json:"display_name"`
		Bio         string `json:"bio"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name required")
		return
	}
	ag, err := agents.Create(r.Context(), a.pool, actor.ID, req.DisplayName, req.Bio)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"agent": ag})
}

func (a *API) updateMyAgent(w http.ResponseWriter, r *http.Request) {
	if !a.agentsEnabled() {
		writeError(w, http.StatusForbidden, "agents are not enabled")
		return
	}
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	var req struct {
		DisplayName *string `json:"display_name"`
		Bio         *string `json:"bio"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := ""
	if req.DisplayName != nil {
		name = strings.TrimSpace(*req.DisplayName)
	}
	bio := ""
	if req.Bio != nil {
		bio = *req.Bio
	}
	ag, err := agents.Update(r.Context(), a.pool, actor.ID, name, bio)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agent": ag})
}

func (a *API) removeMyAgent(w http.ResponseWriter, r *http.Request) {
	if !a.agentsEnabled() {
		writeError(w, http.StatusForbidden, "agents are not enabled")
		return
	}
	actor, ok := a.requireWritableActor(w, r)
	if !ok {
		return
	}
	if err := agents.Remove(r.Context(), a.pool, actor.ID); err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeAgentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, agents.ErrAlreadyHas):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, agents.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, agents.ErrNameTaken), errors.Is(err, agents.ErrNameInvalid), errors.Is(err, agents.ErrHumanNameClash):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}