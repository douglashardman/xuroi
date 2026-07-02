package query

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type PostReport struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	PostID       string    `json:"post_id,omitempty"`
	ThreadID     string    `json:"thread_id"`
	ThreadTitle  string    `json:"thread_title"`
	ThreadURL    string    `json:"thread_url"`
	PostAuthor   string    `json:"post_author,omitempty"`
	PostExcerpt  string    `json:"post_excerpt,omitempty"`
	ReporterName string    `json:"reporter_name"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
	PostURL      string    `json:"post_url,omitempty"`
}

func (r *Reader) OpenReportCount(ctx context.Context, threadID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*)::int FROM post_reports WHERE resolved_at IS NULL AND thread_id = $1)
		+ (SELECT COUNT(*)::int FROM thread_reports WHERE resolved_at IS NULL AND thread_id = $1)
	`, threadID).Scan(&n)
	return n, err
}

func (r *Reader) ListOpenReports(ctx context.Context, threadID *string, limit int) ([]PostReport, error) {
	if limit <= 0 {
		limit = 50
	}
	args := []any{limit}
	filter := ""
	if threadID != nil && *threadID != "" {
		filter = " AND x.thread_id = $2"
		args = append(args, *threadID)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT x.id, x.kind, x.post_id, x.thread_id, x.title, x.slug, x.post_author,
		       x.excerpt, x.reporter_name, x.reason, x.created_at
		FROM (
			SELECT pr.id, 'post' AS kind, pr.post_id, pr.thread_id, t.title, t.slug,
			       pa.display_name AS post_author,
			       LEFT(REGEXP_REPLACE(p.body_html, '<[^>]+>', ' ', 'g'), 160) AS excerpt,
			       ra.display_name AS reporter_name, pr.reason, pr.created_at
			FROM post_reports pr
			JOIN threads t ON t.id = pr.thread_id
			JOIN posts p ON p.id = pr.post_id
			JOIN actors pa ON pa.id = p.author_id
			JOIN actors ra ON ra.id = pr.reporter_id
			WHERE pr.resolved_at IS NULL
			UNION ALL
			SELECT tr.id, 'thread', NULL::text, tr.thread_id, t.title, t.slug,
			       '' AS post_author, '' AS excerpt,
			       ra.display_name, tr.reason, tr.created_at
			FROM thread_reports tr
			JOIN threads t ON t.id = tr.thread_id
			JOIN actors ra ON ra.id = tr.reporter_id
			WHERE tr.resolved_at IS NULL
		) x
		WHERE 1=1`+filter+`
		ORDER BY x.created_at DESC
		LIMIT $1
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []PostReport
	for rows.Next() {
		var rep PostReport
		var slug string
		var postID *string
		if err := rows.Scan(
			&rep.ID, &rep.Kind, &postID, &rep.ThreadID, &rep.ThreadTitle, &slug, &rep.PostAuthor,
			&rep.PostExcerpt, &rep.ReporterName, &rep.Reason, &rep.CreatedAt,
		); err != nil {
			return nil, err
		}
		rep.ThreadURL = "/t/" + slug + "--" + rep.ThreadID
		if postID != nil && *postID != "" {
			rep.PostID = *postID
			rep.PostURL = rep.ThreadURL + "#post-" + rep.PostID
		}
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func (r *Reader) DismissReport(ctx context.Context, reportID, actorID string) error {
	table := "post_reports"
	if strings.HasPrefix(reportID, "trr_") {
		table = "thread_reports"
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE `+table+`
		SET resolved_at = now(), resolved_by = $2
		WHERE id = $1 AND resolved_at IS NULL
	`, reportID, actorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Reader) ReportExists(ctx context.Context, postID, reporterID string) (bool, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM post_reports WHERE post_id = $1 AND reporter_id = $2 AND resolved_at IS NULL
	`, postID, reporterID).Scan(&id)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}