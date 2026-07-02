package query

import (
	"context"
	"strings"
	"time"
)

func (r *Reader) HasRecentDuplicateBody(ctx context.Context, authorID, bodyMarkdown string, within time.Duration) (bool, error) {
	body := strings.TrimSpace(bodyMarkdown)
	if body == "" {
		return false, nil
	}
	if within < time.Minute {
		within = 5 * time.Minute
	}
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM posts
			WHERE author_id = $1
			  AND deleted_at IS NULL
			  AND trim(body_markdown) = $2
			  AND created_at > now() - ($3::bigint * interval '1 second')
		)
	`, authorID, body, int(within.Seconds())).Scan(&exists)
	return exists, err
}