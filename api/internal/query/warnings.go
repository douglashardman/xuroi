package query

import (
	"context"

	"github.com/xuroi/xuroi/api/internal/models"
)

func (r *Reader) WarnedPostIDs(ctx context.Context, postIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(postIDs))
	if len(postIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT post_id FROM warning_posts WHERE post_id = ANY($1)
	`, postIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (r *Reader) annotateWarnedPosts(ctx context.Context, posts []models.Post) error {
	if len(posts) == 0 {
		return nil
	}
	ids := make([]string, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	warned, err := r.WarnedPostIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range posts {
		posts[i].IsWarned = warned[posts[i].ID]
	}
	return nil
}