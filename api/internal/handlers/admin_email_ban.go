package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/ids"
)

func (a *API) listEmailBans(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	rows, err := a.pool.Query(r.Context(), `
		SELECT b.email, b.reason, b.banned_until, b.created_at, COALESCE(m.display_name, '')
		FROM email_bans b
		LEFT JOIN actors m ON m.id = b.banned_by
		WHERE b.banned_until IS NULL OR b.banned_until > now()
		ORDER BY b.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type row struct {
		Email        string     `json:"email"`
		Reason       string     `json:"reason"`
		BannedUntil  *time.Time `json:"banned_until,omitempty"`
		CreatedAt    time.Time  `json:"created_at"`
		BannedByName string     `json:"banned_by_name"`
	}
	var bans []row
	for rows.Next() {
		var b row
		if err := rows.Scan(&b.Email, &b.Reason, &b.BannedUntil, &b.CreatedAt, &b.BannedByName); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		bans = append(bans, b)
	}
	if bans == nil {
		bans = []row{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"bans": bans})
}

func (a *API) banEmail(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}
	var req struct {
		Email    string `json:"email"`
		Reason   string `json:"reason"`
		Duration string `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Reason == "" {
		writeError(w, http.StatusBadRequest, "email and reason required")
		return
	}
	duration, err := auth.ParseBanDuration(req.Duration)
	if err != nil {
		duration = auth.BanPermanent
	}
	until := duration.Until()

	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(), `DELETE FROM email_bans WHERE lower(email) = lower($1)`, email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, err = tx.Exec(r.Context(), `
		INSERT INTO email_bans (id, email, actor_id, reason, banned_until, banned_by)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, ids.New("emb_"), email, req.Reason, until, admin.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = a.forum.LogAdminEvent(r.Context(), events.TypeAdminEmailBanned, admin.ID, events.AdminEmailBan{
		Email:    email,
		Duration: string(duration),
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "banned", "email": email})
}

func (a *API) unbanEmail(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email")))
	if email == "" {
		writeError(w, http.StatusBadRequest, "email required")
		return
	}
	_, err := a.pool.Exec(r.Context(), `DELETE FROM email_bans WHERE lower(email) = lower($1)`, email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared", "email": email})
}

func (a *API) checkEmailBan(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email")))
	if email == "" {
		writeError(w, http.StatusBadRequest, "email required")
		return
	}
	info, err := a.auth.CheckEmailBanned(r.Context(), email)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]any{"banned": false})
		return
	}
	if !errors.Is(err, auth.ErrBanned) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"banned":       true,
		"reason":       info.Reason,
		"banned_until": info.BannedUntil,
		"banned_by":    info.BannedByName,
	})
}