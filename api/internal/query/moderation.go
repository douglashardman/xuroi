package query

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ModQueueItem struct {
	PostID       string    `json:"post_id"`
	ThreadID     string    `json:"thread_id"`
	ThreadTitle  string    `json:"thread_title"`
	ThreadURL    string    `json:"thread_url"`
	CategoryName string    `json:"category_name"`
	AuthorName   string    `json:"author_name"`
	Excerpt      string    `json:"excerpt"`
	IsOP         bool      `json:"is_op"`
	CreatedAt    time.Time `json:"created_at"`
}

func (r *Reader) ListModQueue(ctx context.Context) ([]ModQueueItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.thread_id, t.title, t.slug, c.name, a.display_name,
		       LEFT(REGEXP_REPLACE(p.body_html, '<[^>]+>', ' ', 'g'), 280),
		       p.is_op, p.created_at
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		JOIN categories c ON c.id = t.category_id
		JOIN actors a ON a.id = p.author_id
		WHERE p.moderation_status = 'pending' AND p.deleted_at IS NULL AND t.deleted_at IS NULL
		ORDER BY p.created_at ASC
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("mod queue: %w", err)
	}
	defer rows.Close()

	out := make([]ModQueueItem, 0)
	for rows.Next() {
		var item ModQueueItem
		var slug string
		if err := rows.Scan(
			&item.PostID, &item.ThreadID, &item.ThreadTitle, &slug, &item.CategoryName,
			&item.AuthorName, &item.Excerpt, &item.IsOP, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.ThreadURL = "/t/" + slug + "--" + item.ThreadID
		out = append(out, item)
	}
	return out, rows.Err()
}

func PostModerationStatus(ctx context.Context, pool *pgxpool.Pool, postID string) (string, error) {
	var status string
	err := pool.QueryRow(ctx, `
		SELECT moderation_status FROM posts WHERE id = $1
	`, postID).Scan(&status)
	return status, err
}

func (r *Reader) threadVisibleToViewer(ctx context.Context, threadID string, viewerActorID *string, isStaff bool) (bool, error) {
	if isStaff {
		return true, nil
	}
	var opStatus string
	var opAuthor string
	err := r.pool.QueryRow(ctx, `
		SELECT p.moderation_status, p.author_id
		FROM posts p
		WHERE p.thread_id = $1 AND p.is_op AND p.deleted_at IS NULL
	`, threadID).Scan(&opStatus, &opAuthor)
	if err != nil {
		return false, err
	}
	if opStatus != "pending" {
		return true, nil
	}
	if viewerActorID != nil && *viewerActorID == opAuthor {
		return true, nil
	}
	return false, nil
}