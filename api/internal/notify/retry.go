package notify

import (
	"context"
	"time"
)

const maxEmailAttempts = 5

func retryDelay(attempts int) time.Duration {
	switch {
	case attempts <= 1:
		return 2 * time.Minute
	case attempts == 2:
		return 10 * time.Minute
	case attempts == 3:
		return 30 * time.Minute
	default:
		return 2 * time.Hour
	}
}

func (s *Service) scheduleThreadReplyRetry(ctx context.Context, queueID string, attempts int, err error) error {
	if attempts >= maxEmailAttempts {
		_, dbErr := s.pool.Exec(ctx, `DELETE FROM email_notification_queue WHERE id = $1`, queueID)
		return dbErr
	}
	next := time.Now().Add(retryDelay(attempts))
	msg := truncateErr(err)
	_, dbErr := s.pool.Exec(ctx, `
		UPDATE email_notification_queue
		SET attempts = $2, last_error = $3, next_retry_at = $4, scheduled_at = $4
		WHERE id = $1
	`, queueID, attempts, msg, next)
	return dbErr
}

func (s *Service) scheduleCategoryRetry(ctx context.Context, queueID string, attempts int, err error) error {
	if attempts >= maxEmailAttempts {
		_, dbErr := s.pool.Exec(ctx, `DELETE FROM email_category_queue WHERE id = $1`, queueID)
		return dbErr
	}
	next := time.Now().Add(retryDelay(attempts))
	msg := truncateErr(err)
	_, dbErr := s.pool.Exec(ctx, `
		UPDATE email_category_queue
		SET attempts = $2, last_error = $3, next_retry_at = $4, scheduled_at = $4
		WHERE id = $1
	`, queueID, attempts, msg, next)
	return dbErr
}

func (s *Service) scheduleMentionRetry(ctx context.Context, queueID string, attempts int, err error) error {
	if attempts >= maxEmailAttempts {
		_, dbErr := s.pool.Exec(ctx, `DELETE FROM email_mention_queue WHERE id = $1`, queueID)
		return dbErr
	}
	next := time.Now().Add(retryDelay(attempts))
	msg := truncateErr(err)
	_, dbErr := s.pool.Exec(ctx, `
		UPDATE email_mention_queue
		SET attempts = $2, last_error = $3, next_retry_at = $4, scheduled_at = $4
		WHERE id = $1
	`, queueID, attempts, msg, next)
	return dbErr
}

func truncateErr(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > 500 {
		return msg[:500]
	}
	return msg
}