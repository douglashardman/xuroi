package query

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *Reader) CategoryEmailWatching(ctx context.Context, actorID, categoryID string) (bool, error) {
	var watching bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM category_email_watches
			WHERE actor_id = $1 AND category_id = $2
		)
	`, actorID, categoryID).Scan(&watching)
	return watching, err
}

func (r *Reader) SetCategoryEmailWatch(ctx context.Context, actorID, categoryID string, enabled bool) error {
	if enabled {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO category_email_watches (actor_id, category_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, actorID, categoryID)
		return err
	}
	_, err := r.pool.Exec(ctx, `
		DELETE FROM category_email_watches WHERE actor_id = $1 AND category_id = $2
	`, actorID, categoryID)
	return err
}

func (r *Reader) ActorTimezone(ctx context.Context, actorID string) (string, error) {
	var tz string
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(timezone, '') FROM actors WHERE id = $1`, actorID).Scan(&tz)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return tz, err
}

func (r *Reader) SetActorTimezone(ctx context.Context, actorID, timezone string) error {
	_, err := r.pool.Exec(ctx, `UPDATE actors SET timezone = $2 WHERE id = $1`, actorID, timezone)
	return err
}

func (r *Reader) BlockActor(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO actor_blocks (blocker_id, blocked_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID)
	return err
}

func (r *Reader) UnblockActor(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM actor_blocks WHERE blocker_id = $1 AND blocked_id = $2
	`, blockerID, blockedID)
	return err
}

func (r *Reader) IsBlocked(ctx context.Context, blockerID, blockedID string) (bool, error) {
	var blocked bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM actor_blocks WHERE blocker_id = $1 AND blocked_id = $2
		)
	`, blockerID, blockedID).Scan(&blocked)
	return blocked, err
}

func (r *Reader) IsBlockedEitherWay(ctx context.Context, actorA, actorB string) (bool, error) {
	var blocked bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM actor_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)
	`, actorA, actorB).Scan(&blocked)
	return blocked, err
}