package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/query"
)

func (a *API) getPostAdminAudit(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !a.requireStaffActor(w, r, actor) {
		return
	}

	postID := r.PathValue("id")
	audit, err := a.reader.PostAdminAudit(r.Context(), postID)
	if errors.Is(err, query.ErrNotFound) {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, audit)
}

func (a *API) getPostRevisions(w http.ResponseWriter, r *http.Request) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	actor = a.auth.WithAccessFlags(actor, a.siteCfg.Admin.Emails, a.siteCfg.Admin.ModeratorEmails, a.siteCfg.Admin.PermBanModeratorEmails)

	postID := r.PathValue("id")
	var authorID string
	err = a.pool.QueryRow(r.Context(), `SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL`, postID).Scan(&authorID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if authorID != actor.ID && !auth.IsAdminEmail(actor.Email, a.siteCfg.Admin.Emails) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	revisions, err := a.reader.PostRevisions(r.Context(), postID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revisions": revisions})
}

func (a *API) listReports(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireStaff(w, r); !ok {
		return
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}

	var threadID *string
	if raw := r.URL.Query().Get("thread_id"); raw != "" {
		threadID = &raw
	}

	reports, err := a.reader.ListOpenReports(r.Context(), threadID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
}

func (a *API) dismissReport(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.requireStaff(w, r)
	if !ok {
		return
	}

	reportID := r.PathValue("id")
	if err := a.reader.DismissReport(r.Context(), reportID, actor.ID); errors.Is(err, query.ErrNotFound) {
		writeError(w, http.StatusNotFound, "report not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}