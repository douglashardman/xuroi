package notify

import (
	"context"
	"fmt"
)

func (s *Service) ThreadEmailWatching(ctx context.Context, actorID, threadID string) (bool, error) {
	var muted bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM email_thread_mutes WHERE actor_id = $1 AND thread_id = $2)
	`, actorID, threadID).Scan(&muted)
	if err != nil {
		return false, fmt.Errorf("thread watch status: %w", err)
	}
	return !muted, nil
}

func (s *Service) SetThreadEmailWatch(ctx context.Context, actorID, threadID string, enabled bool) error {
	if enabled {
		_, err := s.pool.Exec(ctx, `
			DELETE FROM email_thread_mutes WHERE actor_id = $1 AND thread_id = $2
		`, actorID, threadID)
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO email_thread_mutes (actor_id, thread_id, muted_at)
		VALUES ($1, $2, now())
		ON CONFLICT (actor_id, thread_id) DO NOTHING
	`, actorID, threadID)
	return err
}