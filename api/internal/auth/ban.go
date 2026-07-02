package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/ids"
)

type BanDuration string

const (
	Ban7Days     BanDuration = "7d"
	Ban30Days    BanDuration = "30d"
	BanPermanent BanDuration = "permanent"
)

type BanInfo struct {
	Reason       string     `json:"ban_reason"`
	BannedUntil  *time.Time `json:"banned_until,omitempty"`
	BannedByName string     `json:"banned_by,omitempty"`
	Duration     string     `json:"ban_duration,omitempty"`
	IsIPBan      bool       `json:"is_ip_ban,omitempty"`
}

func ParseBanDuration(raw string) (BanDuration, error) {
	switch BanDuration(raw) {
	case Ban7Days, Ban30Days, BanPermanent:
		return BanDuration(raw), nil
	default:
		return "", fmt.Errorf("invalid ban duration")
	}
}

func (d BanDuration) Until() *time.Time {
	switch d {
	case Ban7Days:
		t := time.Now().Add(7 * 24 * time.Hour)
		return &t
	case Ban30Days:
		t := time.Now().Add(30 * 24 * time.Hour)
		return &t
	case BanPermanent:
		return nil
	default:
		return nil
	}
}

func (d BanDuration) Label() string {
	switch d {
	case Ban7Days:
		return "7 days"
	case Ban30Days:
		return "30 days"
	case BanPermanent:
		return "permanent"
	default:
		return string(d)
	}
}

func (s *Service) ActorBanInfo(ctx context.Context, actorID string) (BanInfo, error) {
	var info BanInfo
	var state string
	err := s.pool.QueryRow(ctx, `
		SELECT a.ban_reason, a.banned_until, a.state, COALESCE(m.display_name, '')
		FROM actors a
		LEFT JOIN actors m ON m.id = a.banned_by
		WHERE a.id = $1
	`, actorID).Scan(&info.Reason, &info.BannedUntil, &state, &info.BannedByName)
	if errors.Is(err, pgx.ErrNoRows) {
		return BanInfo{}, ErrNotFound
	}
	if err != nil {
		return BanInfo{}, err
	}
	_ = state
	if info.BannedUntil != nil {
		info.Duration = inferDuration(info.BannedUntil)
	} else if info.Reason != "" {
		info.Duration = string(BanPermanent)
	}
	return info, nil
}

func inferDuration(until *time.Time) string {
	if until == nil {
		return string(BanPermanent)
	}
	remaining := time.Until(*until)
	switch {
	case remaining <= 8*24*time.Hour:
		return string(Ban7Days)
	case remaining <= 31*24*time.Hour:
		return string(Ban30Days)
	default:
		return "temporary"
	}
}

func (s *Service) IPBanInfo(ctx context.Context, ip string) (BanInfo, bool, error) {
	if ip == "" {
		return BanInfo{}, false, nil
	}
	var info BanInfo
	err := s.pool.QueryRow(ctx, `
		SELECT b.reason, b.banned_until, COALESCE(m.display_name, '')
		FROM ip_bans b
		LEFT JOIN actors m ON m.id = b.banned_by
		WHERE b.ip_address = $1
		  AND (b.banned_until IS NULL OR b.banned_until > now())
		ORDER BY b.created_at DESC
		LIMIT 1
	`, ip).Scan(&info.Reason, &info.BannedUntil, &info.BannedByName)
	if errors.Is(err, pgx.ErrNoRows) {
		return BanInfo{}, false, nil
	}
	if err != nil {
		return BanInfo{}, false, err
	}
	info.IsIPBan = true
	if info.BannedUntil != nil {
		info.Duration = inferDuration(info.BannedUntil)
	} else {
		info.Duration = string(BanPermanent)
	}
	return info, true, nil
}

func (s *Service) CheckIPBanned(ctx context.Context, ip string) (BanInfo, error) {
	info, banned, err := s.IPBanInfo(ctx, ip)
	if err != nil {
		return BanInfo{}, err
	}
	if banned {
		return info, ErrBanned
	}
	return BanInfo{}, nil
}

func (s *Service) CheckSignInAllowedWithInfo(ctx context.Context, actorID string) (BanInfo, error) {
	info, err := s.LoadAccessInfo(ctx, actorID)
	if err != nil {
		return BanInfo{}, err
	}
	if !info.IsBanned() {
		return BanInfo{}, nil
	}
	details, err := s.ActorBanInfo(ctx, actorID)
	if err != nil {
		return BanInfo{Reason: info.BanReason, BannedUntil: info.BannedUntil}, ErrBanned
	}
	return details, ErrBanned
}

func (s *Service) banUserInTx(ctx context.Context, tx pgx.Tx, actorID, bannedBy string, duration BanDuration, reason string, extraIPs ...string) error {
	until := duration.Until()

	tag, err := tx.Exec(ctx, `
		UPDATE actors
		SET state = 'banned', banned_until = $2, ban_reason = $3, banned_by = $4
		WHERE id = $1 AND type IN ('human', 'agent')
	`, actorID, until, reason, bannedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	_, err = tx.Exec(ctx, `DELETE FROM sessions WHERE actor_id = $1`, actorID)
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT DISTINCT author_ip FROM posts
		WHERE author_id = $1 AND author_ip IS NOT NULL AND author_ip <> ''
	`, actorID)
	if err != nil {
		return err
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return err
		}
		seen[ip] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, ip := range extraIPs {
		if ip != "" {
			seen[ip] = struct{}{}
		}
	}
	for ip := range seen {
		if err := s.insertIPBan(ctx, tx, ip, actorID, bannedBy, until, reason); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) BanUserFull(ctx context.Context, actorID, bannedBy string, duration BanDuration, reason string, extraIPs ...string) error {
	if reason == "" {
		return ErrInvalidInput
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.banUserInTx(ctx, tx, actorID, bannedBy, duration, reason, extraIPs...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) insertIPBan(ctx context.Context, tx pgx.Tx, ip, actorID, bannedBy string, until *time.Time, reason string) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM ip_bans
		WHERE ip_address = $1 AND actor_id = $2
	`, ip, actorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO ip_bans (id, ip_address, actor_id, reason, banned_until, banned_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ids.New("ipb_"), ip, actorID, reason, until, bannedBy)
	return err
}

func (s *Service) ClearBan(ctx context.Context, actorID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE actors
		SET state = 'valid', banned_until = NULL, ban_reason = '', banned_by = NULL
		WHERE id = $1
	`, actorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM ip_bans WHERE actor_id = $1`, actorID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) ExpireIPBans(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM ip_bans WHERE banned_until IS NOT NULL AND banned_until <= now()
	`)
	return err
}