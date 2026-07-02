package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/access"
)

type AdminUser struct {
	ID            string     `json:"id"`
	DisplayName   string     `json:"display_name"`
	Email         string     `json:"email"`
	State         string     `json:"state"`
	BannedUntil   *time.Time `json:"banned_until,omitempty"`
	BanReason     string     `json:"ban_reason,omitempty"`
	BannedByName  string     `json:"banned_by_name,omitempty"`
	WarningCount  int        `json:"warning_count"`
	EmailVerified bool       `json:"email_verified"`
	PostCount     int        `json:"post_count"`
	JoinedAt      time.Time  `json:"joined_at"`
	IsAgent       bool       `json:"is_agent"`
	Permissions   []string   `json:"permissions,omitempty"`
	Entitlements  []string   `json:"entitlements,omitempty"`
}

type AdminOverview struct {
	Members      int `json:"members"`
	Threads      int `json:"threads"`
	Posts        int `json:"posts"`
	OpenReports  int `json:"open_reports"`
	BannedUsers  int `json:"banned_users"`
}

type SitemapEntry struct {
	URL        string    `json:"url"`
	LastMod    time.Time `json:"lastmod"`
	ChangeFreq string    `json:"changefreq,omitempty"`
	Priority   string    `json:"priority,omitempty"`
}

func (r *Reader) ListAdminUsers(ctx context.Context, q string, limit, offset int) ([]AdminUser, int, error) {
	q = strings.TrimSpace(strings.ToLower(q))
	var total int
	countSQL := `
		SELECT COUNT(*)
		FROM actors a
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		WHERE a.type = 'human'
	`
	args := []any{}
	if q != "" {
		countSQL += ` AND (LOWER(a.display_name) LIKE $1 OR LOWER(COALESCE(e.email, '')) LIKE $1)`
		args = append(args, "%"+q+"%")
	}
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listSQL := `
		SELECT a.id, a.display_name, COALESCE(e.email, ''), a.state, a.banned_until, a.ban_reason,
		       COALESCE(m.display_name, ''),
		       (SELECT COUNT(*)::int FROM actor_warnings WHERE actor_id = a.id),
		       COALESCE(e.verified, FALSE), a.created_at,
		       (SELECT COUNT(*)::int FROM posts p WHERE p.author_id = a.id AND p.deleted_at IS NULL),
		       COALESCE((SELECT array_agg(ap.permission ORDER BY ap.permission)
		                 FROM actor_permissions ap WHERE ap.actor_id = a.id), '{}'),
		       COALESCE((SELECT array_agg(ae.entitlement ORDER BY ae.entitlement)
		                 FROM actor_entitlements ae
		                 WHERE ae.actor_id = a.id
		                   AND (ae.expires_at IS NULL OR ae.expires_at > now())), '{}')
		FROM actors a
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		LEFT JOIN actors m ON m.id = a.banned_by
		WHERE a.type = 'human'
	`
	listArgs := []any{}
	if q != "" {
		listSQL += ` AND (LOWER(a.display_name) LIKE $1 OR LOWER(COALESCE(e.email, '')) LIKE $1)`
		listArgs = append(listArgs, "%"+q+"%")
	}
	listSQL += ` ORDER BY a.created_at DESC LIMIT $` + fmt.Sprint(len(listArgs)+1) + ` OFFSET $` + fmt.Sprint(len(listArgs)+2)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(
			&u.ID, &u.DisplayName, &u.Email, &u.State, &u.BannedUntil, &u.BanReason,
			&u.BannedByName, &u.WarningCount, &u.EmailVerified, &u.JoinedAt, &u.PostCount, &u.Permissions, &u.Entitlements,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *Reader) AdminUserByID(ctx context.Context, actorID string) (AdminUser, error) {
	var u AdminUser
	err := r.pool.QueryRow(ctx, `
		SELECT a.id, a.display_name, COALESCE(e.email, ''), a.state, a.banned_until, a.ban_reason,
		       COALESCE(m.display_name, ''),
		       (SELECT COUNT(*)::int FROM actor_warnings WHERE actor_id = a.id),
		       COALESCE(e.verified, FALSE), a.created_at,
		       (SELECT COUNT(*)::int FROM posts p WHERE p.author_id = a.id AND p.deleted_at IS NULL)
		FROM actors a
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		LEFT JOIN actors m ON m.id = a.banned_by
		WHERE a.id = $1 AND a.type = 'human'
	`, actorID).Scan(
		&u.ID, &u.DisplayName, &u.Email, &u.State, &u.BannedUntil, &u.BanReason,
		&u.BannedByName, &u.WarningCount, &u.EmailVerified, &u.JoinedAt, &u.PostCount,
	)
	if err == pgx.ErrNoRows {
		return AdminUser{}, ErrNotFound
	}
	return u, err
}

func (r *Reader) AdminOverview(ctx context.Context) (AdminOverview, error) {
	var out AdminOverview
	err := r.pool.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*)::int FROM actors WHERE type = 'human'),
		  (SELECT COUNT(*)::int FROM threads WHERE deleted_at IS NULL),
		  (SELECT COUNT(*)::int FROM posts WHERE deleted_at IS NULL),
		  (SELECT COUNT(*)::int FROM post_reports WHERE resolved_at IS NULL)
		    + (SELECT COUNT(*)::int FROM thread_reports WHERE resolved_at IS NULL),
		  (SELECT COUNT(*)::int FROM actors WHERE state = 'banned')
	`).Scan(&out.Members, &out.Threads, &out.Posts, &out.OpenReports, &out.BannedUsers)
	return out, err
}

func (r *Reader) SitemapEntries(ctx context.Context) ([]SitemapEntry, error) {
	cats, err := r.listCategories(ctx, access.Viewer{IsGuest: true, Entitlements: map[string]bool{}})
	if err != nil {
		return nil, err
	}

	var entries []SitemapEntry
	entries = append(entries, SitemapEntry{URL: "/", ChangeFreq: "daily", Priority: "1.0"})
	entries = append(entries, SitemapEntry{URL: "/community", ChangeFreq: "daily", Priority: "0.9"})
	for _, c := range cats {
		entries = append(entries, SitemapEntry{
			URL:        c.URL,
			ChangeFreq: "daily",
			Priority:   "0.8",
		})
	}

	rows, err := r.pool.Query(ctx, `
		SELECT slug, id, last_activity_at
		FROM threads
		WHERE deleted_at IS NULL
		ORDER BY last_activity_at DESC
		LIMIT 10000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var slug, id string
		var lastMod time.Time
		if err := rows.Scan(&slug, &id, &lastMod); err != nil {
			return nil, err
		}
		entries = append(entries, SitemapEntry{
			URL:        "/t/" + slug + "--" + id,
			LastMod:    lastMod,
			ChangeFreq: "weekly",
			Priority:   "0.6",
		})
	}
	return entries, rows.Err()
}