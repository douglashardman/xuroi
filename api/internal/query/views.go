package query

import (
	"context"
	"fmt"
	"time"
)

const viewDedupWindow = 6 * time.Hour

func (r *Reader) RecordThreadView(ctx context.Context, actorID, threadID string) error {
	var lastCounted time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT last_counted_at FROM thread_view_dedup
		WHERE actor_id = $1 AND thread_id = $2
	`, actorID, threadID).Scan(&lastCounted)
	if err == nil && time.Since(lastCounted) < viewDedupWindow {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO thread_view_dedup (actor_id, thread_id, last_counted_at)
		VALUES ($1, $2, now())
		ON CONFLICT (actor_id, thread_id) DO UPDATE SET last_counted_at = EXCLUDED.last_counted_at
	`, actorID, threadID); err != nil {
		return fmt.Errorf("view dedup: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE threads SET view_count = view_count + 1 WHERE id = $1 AND deleted_at IS NULL
	`, threadID); err != nil {
		return fmt.Errorf("increment view: %w", err)
	}
	return tx.Commit(ctx)
}