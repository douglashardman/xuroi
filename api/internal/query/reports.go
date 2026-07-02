package query

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type PostReport struct {
	ID           string    `json:"id"`
	PostID       string    `json:"post_id"`
	ThreadID     string    `json:"thread_id"`
	ThreadTitle  string    `json:"thread_title"`
	ThreadURL    string    `json:"thread_url"`
	PostAuthor   string    `json:"post_author"`
	PostExcerpt  string    `json:"post_excerpt"`
	ReporterName string    `json:"reporter_name"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
	PostURL      string    `json:"post_url"`
}

func (r *Reader) OpenReportCount(ctx context.Context, threadID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM post_reports
		WHERE resolved_at IS NULL AND thread_id = $1
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
		filter = " AND pr.thread_id = $2"
		args = append(args, *threadID)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT pr.id, pr.post_id, pr.thread_id, t.title, t.slug, pa.display_name,
		       LEFT(REGEXP_REPLACE(p.body_html, '<[^>]+>', ' ', 'g'), 160),
		       ra.display_name, pr.reason, pr.created_at
		FROM post_reports pr
		JOIN threads t ON t.id = pr.thread_id
		JOIN posts p ON p.id = pr.post_id
		JOIN actors pa ON pa.id = p.author_id
		JOIN actors ra ON ra.id = pr.reporter_id
		WHERE pr.resolved_at IS NULL`+filter+`
		ORDER BY pr.created_at DESC
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
		if err := rows.Scan(
			&rep.ID, &rep.PostID, &rep.ThreadID, &rep.ThreadTitle, &slug, &rep.PostAuthor,
			&rep.PostExcerpt, &rep.ReporterName, &rep.Reason, &rep.CreatedAt,
		); err != nil {
			return nil, err
		}
		rep.ThreadURL = "/t/" + slug + "--" + rep.ThreadID
		rep.PostURL = rep.ThreadURL + "#post-" + rep.PostID
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func (r *Reader) DismissReport(ctx context.Context, reportID, actorID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE post_reports
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