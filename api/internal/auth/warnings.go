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
	WarningTTL              = 8 * time.Hour
	WarningIncidentWindow   = 24 * time.Hour // drunk-spree posts → one strike
	MaxWarningsBeforeBan    = 3
	AutoBanOnWarningsReason = "Automatic 7-day ban — third warning on your account."
)

var ErrPostAlreadyWarned = errors.New("post already warned")

type ActiveWarning struct {
	Message      string `json:"message"`
	WarnedBy     string `json:"warned_by"`
	ExpiresAt    string `json:"expires_at"`
	WarningCount int    `json:"warning_count"`
	StrikeNumber int    `json:"strike_number"`
}

type WarnResult struct {
	WarningCount int    `json:"warning_count"`
	AutoBanned   bool   `json:"auto_banned"`
	Consolidated bool   `json:"consolidated"`
	PostID       string `json:"post_id,omitempty"`
}

func (s *Service) ActiveWarning(ctx context.Context, actorID string) (*ActiveWarning, error) {
	var w ActiveWarning
	var expires time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT w.message, COALESCE(m.display_name, 'Moderator'), w.expires_at,
		       (SELECT COUNT(*)::int FROM actor_warnings WHERE actor_id = $1)
		FROM actor_warnings w
		LEFT JOIN actors m ON m.id = w.warned_by
		WHERE w.actor_id = $1 AND w.expires_at > now()
		ORDER BY w.created_at DESC
		LIMIT 1
	`, actorID).Scan(&w.Message, &w.WarnedBy, &expires, &w.WarningCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.ExpiresAt = expires.Format(time.RFC3339)
	w.StrikeNumber = w.WarningCount
	return &w, nil
}

func (s *Service) WarningCount(ctx context.Context, actorID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM actor_warnings WHERE actor_id = $1
	`, actorID).Scan(&n)
	return n, err
}

func (s *Service) PostIsWarned(ctx context.Context, postID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM warning_posts WHERE post_id = $1)
	`, postID).Scan(&exists)
	return exists, err
}

func (s *Service) IssueWarning(ctx context.Context, actorID, warnedBy, message string, postID *string) (WarnResult, error) {
	if message == "" {
		return WarnResult{}, ErrInvalidInput
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WarnResult{}, err
	}
	defer tx.Rollback(ctx)

	var human bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM actors WHERE id = $1 AND type = 'human')`, actorID).Scan(&human)
	if err != nil {
		return WarnResult{}, err
	}
	if !human {
		return WarnResult{}, ErrNotFound
	}

	if postID != nil && *postID != "" {
		var warned bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warning_posts WHERE post_id = $1)`, *postID).Scan(&warned)
		if err != nil {
			return WarnResult{}, err
		}
		if warned {
			return WarnResult{}, ErrPostAlreadyWarned
		}
		var authorID string
		err = tx.QueryRow(ctx, `
			SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL
		`, *postID).Scan(&authorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return WarnResult{}, ErrNotFound
		}
		if err != nil {
			return WarnResult{}, err
		}
		if authorID != actorID {
			return WarnResult{}, ErrInvalidInput
		}
	}

	cutoff := time.Now().Add(-WarningIncidentWindow)
	var recentID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM actor_warnings
		WHERE actor_id = $1 AND created_at > $2
		ORDER BY created_at DESC
		LIMIT 1
	`, actorID, cutoff).Scan(&recentID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WarnResult{}, err
	}

	expires := time.Now().Add(WarningTTL)
	result := WarnResult{}

	if postID != nil {
		result.PostID = *postID
	}

	if err == nil && recentID != "" {
		// Same incident window — extend overlay, do not add a strike.
		_, err = tx.Exec(ctx, `
			UPDATE actor_warnings
			SET message = $2, expires_at = $3, warned_by = $4
			WHERE id = $1
		`, recentID, message, expires, warnedBy)
		if err != nil {
			return WarnResult{}, err
		}
		if postID != nil && *postID != "" {
			_, err = tx.Exec(ctx, `
				INSERT INTO warning_posts (post_id, warning_id) VALUES ($1, $2)
			`, *postID, recentID)
			if err != nil {
				return WarnResult{}, fmt.Errorf("link post to warning: %w", err)
			}
		}
		var count int
		err = tx.QueryRow(ctx, `SELECT COUNT(*)::int FROM actor_warnings WHERE actor_id = $1`, actorID).Scan(&count)
		if err != nil {
			return WarnResult{}, err
		}
		result.WarningCount = count
		result.Consolidated = true
		if err := tx.Commit(ctx); err != nil {
			return WarnResult{}, err
		}
		return result, nil
	}

	// New strike.
	if err := s.deductWarningKarma(ctx, tx, actorID); err != nil {
		return WarnResult{}, fmt.Errorf("warning karma: %w", err)
	}

	warningID := ids.New("wrn_")
	_, err = tx.Exec(ctx, `
		INSERT INTO actor_warnings (id, actor_id, message, warned_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, warningID, actorID, message, warnedBy, expires)
	if err != nil {
		return WarnResult{}, fmt.Errorf("insert warning: %w", err)
	}
	if postID != nil && *postID != "" {
		_, err = tx.Exec(ctx, `
			INSERT INTO warning_posts (post_id, warning_id) VALUES ($1, $2)
		`, *postID, warningID)
		if err != nil {
			return WarnResult{}, fmt.Errorf("link post to warning: %w", err)
		}
	}

	var count int
	err = tx.QueryRow(ctx, `SELECT COUNT(*)::int FROM actor_warnings WHERE actor_id = $1`, actorID).Scan(&count)
	if err != nil {
		return WarnResult{}, err
	}
	result.WarningCount = count

	if count >= MaxWarningsBeforeBan {
		if err := s.banUserInTx(ctx, tx, actorID, warnedBy, Ban7Days, AutoBanOnWarningsReason); err != nil {
			return WarnResult{}, err
		}
		result.AutoBanned = true
	}

	if err := tx.Commit(ctx); err != nil {
		return WarnResult{}, err
	}
	return result, nil
}