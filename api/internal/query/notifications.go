package query

import (
	"context"
	"fmt"
	"time"
)

type Notification struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	URL           string     `json:"url"`
	FromActorID   *string    `json:"from_actor_id,omitempty"`
	FromActorName *string    `json:"from_actor_name,omitempty"`
	PostID        *string    `json:"post_id,omitempty"`
	ThreadID      *string    `json:"thread_id,omitempty"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (r *Reader) NotificationUnreadCount(ctx context.Context, actorID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*)::int FROM notifications
		WHERE actor_id = $1 AND read_at IS NULL
	`, actorID).Scan(&n)
	return n, err
}

func (r *Reader) ListNotifications(ctx context.Context, actorID string, limit, offset int) ([]Notification, error) {
	if limit < 1 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.type, n.title, n.body, n.url,
		       n.from_actor_id, fa.display_name,
		       n.post_id, n.thread_id, n.read_at, n.created_at
		FROM notifications n
		LEFT JOIN actors fa ON fa.id = n.from_actor_id
		WHERE n.actor_id = $1
		ORDER BY n.created_at DESC
		LIMIT $2 OFFSET $3
	`, actorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	out := make([]Notification, 0)
	for rows.Next() {
		var n Notification
		if err := rows.Scan(
			&n.ID, &n.Type, &n.Title, &n.Body, &n.URL,
			&n.FromActorID, &n.FromActorName,
			&n.PostID, &n.ThreadID, &n.ReadAt, &n.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Reader) MarkNotificationRead(ctx context.Context, actorID, notificationID string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE id = $1 AND actor_id = $2 AND read_at IS NULL
	`, notificationID, actorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Reader) MarkAllNotificationsRead(ctx context.Context, actorID string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE actor_id = $1 AND read_at IS NULL
	`, actorID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}