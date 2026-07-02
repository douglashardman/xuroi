package notify

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type EmailPreferences struct {
	ThreadRepliesEnabled bool `json:"thread_replies_enabled"`
	MentionsEnabled      bool `json:"mentions_enabled"`
}

func (s *Service) GetEmailPreferences(ctx context.Context, actorID string) (EmailPreferences, error) {
	prefs := EmailPreferences{
		ThreadRepliesEnabled: true,
		MentionsEnabled:      true,
	}
	err := s.pool.QueryRow(ctx, `
		SELECT thread_replies_enabled, mentions_enabled
		FROM email_preferences WHERE actor_id = $1
	`, actorID).Scan(&prefs.ThreadRepliesEnabled, &prefs.MentionsEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return prefs, nil
	}
	return prefs, err
}

func (s *Service) SetEmailPreferences(ctx context.Context, actorID string, prefs EmailPreferences) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO email_preferences (actor_id, thread_replies_enabled, mentions_enabled, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (actor_id) DO UPDATE SET
			thread_replies_enabled = EXCLUDED.thread_replies_enabled,
			mentions_enabled = EXCLUDED.mentions_enabled,
			updated_at = now()
	`, actorID, prefs.ThreadRepliesEnabled, prefs.MentionsEnabled)
	if err != nil {
		return err
	}

	if !prefs.ThreadRepliesEnabled {
		_, err = tx.Exec(ctx, `DELETE FROM email_notification_queue WHERE actor_id = $1`, actorID)
		if err != nil {
			return err
		}
	}
	if !prefs.MentionsEnabled {
		_, err = tx.Exec(ctx, `DELETE FROM email_mention_queue WHERE actor_id = $1`, actorID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}