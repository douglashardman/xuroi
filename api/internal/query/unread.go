package query

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/xuroi/xuroi/api/internal/access"
)

const unreadSQL = `t.last_activity_at > COALESCE(tr.last_read_at, '-infinity'::timestamptz)`

func (r *Reader) threadUnreadMap(ctx context.Context, actorID string, threadIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(threadIDs))
	if len(threadIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, (`+unreadSQL+`) AS is_unread
		FROM threads t
		LEFT JOIN thread_reads tr ON tr.thread_id = t.id AND tr.actor_id = $1
		WHERE t.id = ANY($2)
	`, actorID, threadIDs)
	if err != nil {
		return nil, fmt.Errorf("thread unread: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var isUnread bool
		if err := rows.Scan(&id, &isUnread); err != nil {
			return nil, fmt.Errorf("scan thread unread: %w", err)
		}
		out[id] = isUnread
	}
	return out, rows.Err()
}

func (r *Reader) UnreadThreadCount(ctx context.Context, viewer access.Viewer) (int, error) {
	if viewer.ActorID == nil {
		return 0, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.access_level, c.access_levels
		FROM threads t
		JOIN categories c ON c.id = t.category_id
		JOIN posts op ON op.thread_id = t.id AND op.is_op AND op.deleted_at IS NULL
		LEFT JOIN thread_reads tr ON tr.thread_id = t.id AND tr.actor_id = $1
		WHERE t.deleted_at IS NULL
		  AND op.moderation_status = 'approved'
		  AND `+unreadSQL+`
	`, *viewer.ActorID)
	if err != nil {
		return 0, fmt.Errorf("unread thread count: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var accessLevel string
		var accessLevels []string
		if err := rows.Scan(&accessLevel, &accessLevels); err != nil {
			return 0, err
		}
		levels := access.NormalizeLevels(accessLevels)
		if len(levels) == 0 {
			levels = []string{access.NormalizeLevel(accessLevel)}
		}
		if viewer.CanViewAny(levels) {
			count++
		}
	}
	return count, rows.Err()
}

func (r *Reader) CategoryUnreadCounts(ctx context.Context, actorID string, categoryIDs []string) (map[string]int, error) {
	out := make(map[string]int)
	if len(categoryIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT t.category_id, count(*)::int
		FROM threads t
		JOIN posts op ON op.thread_id = t.id AND op.is_op AND op.deleted_at IS NULL
		LEFT JOIN thread_reads tr ON tr.thread_id = t.id AND tr.actor_id = $1
		WHERE t.category_id = ANY($2)
		  AND t.deleted_at IS NULL
		  AND op.moderation_status = 'approved'
		  AND `+unreadSQL+`
		GROUP BY t.category_id
	`, actorID, categoryIDs)
	if err != nil {
		return nil, fmt.Errorf("category unread counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var catID string
		var n int
		if err := rows.Scan(&catID, &n); err != nil {
			return nil, err
		}
		out[catID] = n
	}
	return out, rows.Err()
}

func (r *Reader) MarkCategoryRead(ctx context.Context, actorID, categoryID string) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO thread_reads (actor_id, thread_id, last_read_at)
		SELECT $1, t.id, now()
		FROM threads t
		WHERE t.category_id = $2 AND t.deleted_at IS NULL
		ON CONFLICT (actor_id, thread_id) DO UPDATE SET last_read_at = EXCLUDED.last_read_at
	`, actorID, categoryID)
	if err != nil {
		return 0, fmt.Errorf("mark category read: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *Reader) threadEmailWatching(ctx context.Context, actorID, threadID string) (bool, error) {
	var muted bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM email_thread_mutes WHERE actor_id = $1 AND thread_id = $2)
	`, actorID, threadID).Scan(&muted)
	if err != nil {
		return false, err
	}
	return !muted, nil
}

func (r *Reader) CategoryIDBySlug(ctx context.Context, slug string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM categories WHERE slug = $1
	`, slug).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrNotFound
		}
		return "", err
	}
	return id, nil
}