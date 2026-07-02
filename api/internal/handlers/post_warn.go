package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/auth"
)

func (a *API) warnPost(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}

	postID := r.PathValue("id")
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "warning message required")
		return
	}

	var authorID string
	err := a.pool.QueryRow(r.Context(), `
		SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL
	`, postID).Scan(&authorID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result, err := a.auth.IssueWarning(r.Context(), authorID, staff.ID, req.Message, &postID)
	if errors.Is(err, auth.ErrPostAlreadyWarned) {
		writeError(w, http.StatusConflict, "this post was already warned")
		return
	}
	if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid warning request")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := map[string]any{
		"status":        "warned",
		"warning_count": result.WarningCount,
		"auto_banned":   result.AutoBanned,
		"consolidated":  result.Consolidated,
		"post_id":       postID,
	}
	if result.Consolidated {
		out["message"] = "Added to current warning window — no extra strike"
	}
	if result.AutoBanned {
		out["message"] = "Third strike — member automatically banned for 7 days"
	}
	writeJSON(w, http.StatusOK, out)
}