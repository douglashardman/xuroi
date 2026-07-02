package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *Service) DeleteAccount(ctx context.Context, actorID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var shortID string
	err = tx.QueryRow(ctx, `SELECT RIGHT(id, 8) FROM actors WHERE id = $1 AND type = 'human' AND deleted_at IS NULL`, actorID).Scan(&shortID)
	if err != nil {
		return ErrNotFound
	}

	anonName := fmt.Sprintf("Deleted User %s", shortID)
	_, err = tx.Exec(ctx, `
		UPDATE actors
		SET display_name = $2, bio = '', avatar_url = '', deleted_at = now(), state = 'valid',
		    banned_until = NULL, ban_reason = '', banned_by = NULL
		WHERE id = $1
	`, actorID, anonName)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM actor_emails WHERE actor_id = $1`, actorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM actor_passwords WHERE actor_id = $1`, actorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM actor_sessions WHERE actor_id = $1`, actorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM actor_permissions WHERE actor_id = $1`, actorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM actor_entitlements WHERE actor_id = $1`, actorID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) IsDeleted(ctx context.Context, actorID string) (bool, error) {
	var deleted bool
	err := s.pool.QueryRow(ctx, `
		SELECT deleted_at IS NOT NULL FROM actors WHERE id = $1
	`, actorID).Scan(&deleted)
	if err == pgx.ErrNoRows {
		return false, ErrNotFound
	}
	return deleted, err
}