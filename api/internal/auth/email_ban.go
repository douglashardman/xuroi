package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/ids"
)

func (s *Service) CheckEmailBanned(ctx context.Context, email string) (BanInfo, error) {
	email = normalizeEmail(email)
	if email == "" {
		return BanInfo{}, nil
	}
	var info BanInfo
	err := s.pool.QueryRow(ctx, `
		SELECT b.reason, b.banned_until, COALESCE(m.display_name, '')
		FROM email_bans b
		LEFT JOIN actors m ON m.id = b.banned_by
		WHERE lower(b.email) = lower($1)
		  AND (b.banned_until IS NULL OR b.banned_until > now())
		ORDER BY b.created_at DESC
		LIMIT 1
	`, email).Scan(&info.Reason, &info.BannedUntil, &info.BannedByName)
	if errors.Is(err, pgx.ErrNoRows) {
		return BanInfo{}, nil
	}
	if err != nil {
		return BanInfo{}, err
	}
	if info.BannedUntil != nil {
		info.Duration = inferDuration(info.BannedUntil)
	} else if info.Reason != "" {
		info.Duration = string(BanPermanent)
	}
	return info, ErrBanned
}

func (s *Service) insertEmailBan(ctx context.Context, tx pgx.Tx, email, actorID, bannedBy string, until *time.Time, reason string) error {
	email = normalizeEmail(email)
	if email == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `DELETE FROM email_bans WHERE lower(email) = lower($1)`, email)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO email_bans (id, email, actor_id, reason, banned_until, banned_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ids.New("emb_"), email, actorID, reason, until, bannedBy)
	return err
}

func (s *Service) clearEmailBansForActor(ctx context.Context, tx pgx.Tx, actorID string) error {
	_, err := tx.Exec(ctx, `DELETE FROM email_bans WHERE actor_id = $1`, actorID)
	return err
}