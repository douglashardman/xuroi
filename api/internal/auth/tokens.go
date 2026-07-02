package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/ids"
)

const (
	MagicLinkTTL         = 15 * time.Minute
	PasswordResetTTL     = time.Hour
	EmailVerifyTTL       = 48 * time.Hour
	ThreadUnsubscribeTTL = 365 * 24 * time.Hour
)

type TokenPurpose string

const (
	PurposePasswordReset     TokenPurpose = "password_reset"
	PurposeMagicLink         TokenPurpose = "magic_link"
	PurposeEmailVerify       TokenPurpose = "email_verify"
	PurposeThreadUnsubscribe TokenPurpose = "thread_unsubscribe"
)

var ErrInvalidToken = errors.New("invalid or expired token")

func (s *Service) actorByEmail(ctx context.Context, email string) (Actor, error) {
	email = normalizeEmail(email)
	if email == "" {
		return Actor{}, ErrInvalidInput
	}

	var actor Actor
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, a.display_name, e.email
		FROM actor_emails e
		JOIN actors a ON a.id = e.actor_id
		WHERE e.email = $1
	`, email).Scan(&actor.ID, &actor.DisplayName, &actor.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, ErrNotFound
	}
	if err != nil {
		return Actor{}, err
	}
	return actor, nil
}

func (s *Service) IssuePasswordResetToken(ctx context.Context, email string) (string, Actor, error) {
	actor, err := s.actorByEmail(ctx, email)
	if err != nil {
		return "", Actor{}, err
	}
	if err := s.invalidatePendingTokens(ctx, actor.ID, PurposePasswordReset); err != nil {
		return "", Actor{}, err
	}
	token, err := s.createAuthToken(ctx, actor.ID, actor.Email, PurposePasswordReset, PasswordResetTTL, nil)
	if err != nil {
		return "", Actor{}, err
	}
	return token, actor, nil
}

func (s *Service) IssueEmailVerifyToken(ctx context.Context, actorID string) (string, Actor, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return "", Actor{}, err
	}
	if err := s.invalidatePendingTokens(ctx, actor.ID, PurposeEmailVerify); err != nil {
		return "", Actor{}, err
	}
	token, err := s.createAuthToken(ctx, actor.ID, actor.Email, PurposeEmailVerify, EmailVerifyTTL, nil)
	if err != nil {
		return "", Actor{}, err
	}
	return token, actor, nil
}

func (s *Service) VerifyEmailWithToken(ctx context.Context, rawToken string) (Actor, error) {
	actorID, err := s.consumeToken(ctx, rawToken, PurposeEmailVerify)
	if err != nil {
		return Actor{}, err
	}
	if err := s.MarkEmailVerified(ctx, actorID); err != nil {
		return Actor{}, err
	}
	return s.loadActor(ctx, actorID)
}

func (s *Service) IssueMagicLinkToken(ctx context.Context, email string) (string, Actor, error) {
	actor, err := s.actorByEmail(ctx, email)
	if err != nil {
		return "", Actor{}, err
	}
	if err := s.invalidatePendingTokens(ctx, actor.ID, PurposeMagicLink); err != nil {
		return "", Actor{}, err
	}
	token, err := s.createAuthToken(ctx, actor.ID, actor.Email, PurposeMagicLink, MagicLinkTTL, nil)
	if err != nil {
		return "", Actor{}, err
	}
	return token, actor, nil
}

func (s *Service) ResetPasswordWithToken(ctx context.Context, rawToken, password string) (Actor, string, error) {
	actorID, err := s.consumeToken(ctx, rawToken, PurposePasswordReset)
	if err != nil {
		return Actor{}, "", err
	}
	if err := s.SetPassword(ctx, actorID, password); err != nil {
		return Actor{}, "", err
	}
	_ = s.MarkEmailVerified(ctx, actorID)
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return Actor{}, "", err
	}
	if err := s.CheckSignInAllowed(ctx, actorID); err != nil {
		return actor, "", err
	}
	sessionToken, err := s.insertSession(ctx, s.pool, actorID, SessionDays)
	if err != nil {
		return Actor{}, "", err
	}
	return actor, sessionToken, nil
}

func (s *Service) LoginWithMagicLink(ctx context.Context, rawToken string) (Actor, string, error) {
	actorID, err := s.consumeToken(ctx, rawToken, PurposeMagicLink)
	if err != nil {
		return Actor{}, "", err
	}
	_ = s.MarkEmailVerified(ctx, actorID)
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return Actor{}, "", err
	}
	if err := s.CheckSignInAllowed(ctx, actorID); err != nil {
		return actor, "", err
	}
	sessionToken, err := s.insertSession(ctx, s.pool, actorID, SessionDays)
	if err != nil {
		return Actor{}, "", err
	}
	return actor, sessionToken, nil
}

func (s *Service) loadActor(ctx context.Context, actorID string) (Actor, error) {
	var actor Actor
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, a.display_name, COALESCE(e.email, '')
		FROM actors a
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		WHERE a.id = $1
	`, actorID).Scan(&actor.ID, &actor.DisplayName, &actor.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, ErrNotFound
	}
	return actor, err
}

func (s *Service) IssueThreadUnsubscribeToken(ctx context.Context, actorID, threadID, email string) (string, error) {
	if actorID == "" || threadID == "" {
		return "", ErrInvalidInput
	}
	_, _ = s.pool.Exec(ctx, `
		UPDATE auth_tokens SET used_at = now()
		WHERE actor_id = $1 AND thread_id = $2 AND purpose = $3 AND used_at IS NULL
	`, actorID, threadID, string(PurposeThreadUnsubscribe))
	return s.createAuthToken(ctx, actorID, email, PurposeThreadUnsubscribe, ThreadUnsubscribeTTL, &threadID)
}

// UnsubscribeThreadEmail mutes thread reply emails for the token bearer (idempotent).
func (s *Service) UnsubscribeThreadEmail(ctx context.Context, rawToken string) (threadTitle string, err error) {
	actorID, threadID, err := s.validateThreadUnsubscribeToken(ctx, rawToken)
	if err != nil {
		return "", err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO email_thread_mutes (actor_id, thread_id)
		VALUES ($1, $2)
		ON CONFLICT (actor_id, thread_id) DO NOTHING
	`, actorID, threadID)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `
		DELETE FROM email_notification_queue WHERE actor_id = $1 AND thread_id = $2
	`, actorID, threadID)
	if err != nil {
		return "", err
	}
	err = tx.QueryRow(ctx, `SELECT title FROM threads WHERE id = $1`, threadID).Scan(&threadTitle)
	if err != nil {
		threadTitle = "this thread"
	}
	return threadTitle, tx.Commit(ctx)
}

func (s *Service) validateThreadUnsubscribeToken(ctx context.Context, rawToken string) (actorID, threadID string, err error) {
	if rawToken == "" {
		return "", "", ErrInvalidToken
	}
	err = s.pool.QueryRow(ctx, `
		SELECT actor_id, COALESCE(thread_id, '')
		FROM auth_tokens
		WHERE token_hash = $1 AND purpose = $2 AND expires_at > now()
	`, hashToken(rawToken), string(PurposeThreadUnsubscribe)).Scan(&actorID, &threadID)
	if errors.Is(err, pgx.ErrNoRows) || threadID == "" {
		return "", "", ErrInvalidToken
	}
	if err != nil {
		return "", "", err
	}
	return actorID, threadID, nil
}

func (s *Service) createAuthToken(ctx context.Context, actorID, email string, purpose TokenPurpose, ttl time.Duration, threadID *string) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO auth_tokens (id, purpose, token_hash, actor_id, email, expires_at, thread_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, ids.New("atk_"), string(purpose), hashToken(token), actorID, email, time.Now().Add(ttl), threadID)
	if err != nil {
		return "", fmt.Errorf("insert auth token: %w", err)
	}
	return token, nil
}

func (s *Service) invalidatePendingTokens(ctx context.Context, actorID string, purpose TokenPurpose) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE auth_tokens SET used_at = now()
		WHERE actor_id = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
	`, actorID, string(purpose))
	return err
}

func (s *Service) consumeToken(ctx context.Context, rawToken string, purpose TokenPurpose) (string, error) {
	if rawToken == "" {
		return "", ErrInvalidToken
	}
	hash := hashToken(rawToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var actorID string
	err = tx.QueryRow(ctx, `
		SELECT actor_id FROM auth_tokens
		WHERE token_hash = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
		FOR UPDATE
	`, hash, string(purpose)).Scan(&actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidToken
	}
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `UPDATE auth_tokens SET used_at = now() WHERE token_hash = $1`, hash)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return actorID, nil
}