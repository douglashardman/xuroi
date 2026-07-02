package query

import (
	"context"
	"fmt"
)

type categoryCountPair struct {
	Threads int
	Posts   int
}

func (r *Reader) categoryLiveCounts(ctx context.Context, categoryIDs []string) (map[string]categoryCountPair, error) {
	out := make(map[string]categoryCountPair)
	if len(categoryIDs) == 0 {
		return out, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT category_id, count(*)::int
		FROM threads
		WHERE category_id = ANY($1) AND deleted_at IS NULL
		GROUP BY category_id
	`, categoryIDs)
	if err != nil {
		return nil, fmt.Errorf("category thread counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		p := out[id]
		p.Threads = n
		out[id] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = r.pool.Query(ctx, `
		SELECT t.category_id, count(*)::int
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		WHERE t.category_id = ANY($1)
		  AND t.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		  AND p.moderation_status = 'approved'
		GROUP BY t.category_id
	`, categoryIDs)
	if err != nil {
		return nil, fmt.Errorf("category post counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		p := out[id]
		p.Posts = n
		out[id] = p
	}
	return out, rows.Err()
}