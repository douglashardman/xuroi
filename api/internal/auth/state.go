package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type ActorState string

const (
	StateValid       ActorState = "valid"
	StateDiscouraged ActorState = "discouraged"
	StateBanned      ActorState = "banned"
)

type AccessInfo struct {
	State         ActorState
	BannedUntil   *time.Time
	BanReason     string
	EmailVerified bool
	IsAgent       bool
}

var (
	ErrBanned           = errors.New("account banned")
	ErrEmailNotVerified = errors.New("email not verified")
)

func (info AccessInfo) IsBanned() bool {
	if info.State != StateBanned {
		return false
	}
	if info.BannedUntil != nil && info.BannedUntil.Before(time.Now()) {
		return false
	}
	return true
}

func (s *Service) LoadAccessInfo(ctx context.Context, actorID string) (AccessInfo, error) {
	var info AccessInfo
	var state string
	var actorType string
	err := s.pool.QueryRow(ctx, `
		SELECT a.state, a.banned_until, a.ban_reason, COALESCE(e.verified, FALSE), a.type
		FROM actors a
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		WHERE a.id = $1
	`, actorID).Scan(&state, &info.BannedUntil, &info.BanReason, &info.EmailVerified, &actorType)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccessInfo{}, ErrNotFound
	}
	if err != nil {
		return AccessInfo{}, err
	}
	info.State = ActorState(state)
	info.IsAgent = actorType == "agent"

	if info.IsBanned() && info.BannedUntil != nil && info.BannedUntil.Before(time.Now()) {
		_ = s.ClearBan(ctx, actorID)
		info.State = StateValid
		info.BannedUntil = nil
		info.BanReason = ""
	}
	return info, nil
}

func (s *Service) CheckWritable(ctx context.Context, actorID string) (AccessInfo, error) {
	info, err := s.LoadAccessInfo(ctx, actorID)
	if err != nil {
		return AccessInfo{}, err
	}
	if info.IsBanned() {
		return info, ErrBanned
	}
	if !info.EmailVerified && !info.IsAgent {
		return info, ErrEmailNotVerified
	}
	return info, nil
}

func (s *Service) CheckSignInAllowed(ctx context.Context, actorID string) error {
	info, err := s.LoadAccessInfo(ctx, actorID)
	if err != nil {
		return err
	}
	if info.IsBanned() {
		return ErrBanned
	}
	return nil
}

func (s *Service) MarkEmailVerified(ctx context.Context, actorID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE actor_emails SET verified = TRUE WHERE actor_id = $1
	`, actorID)
	return err
}

func (s *Service) SetDiscouraged(ctx context.Context, actorID string, discouraged bool) error {
	state := StateValid
	if discouraged {
		state = StateDiscouraged
	}
	_, err := s.pool.Exec(ctx, `UPDATE actors SET state = $2 WHERE id = $1`, actorID, string(state))
	return err
}

func IsModeratorEmail(email string, modEmails []string) bool {
	email = normalizeEmail(email)
	for _, m := range modEmails {
		if email == normalizeEmail(m) {
			return true
		}
	}
	return false
}

func IsPermBanModerator(email string, permBanEmails []string) bool {
	return IsModeratorEmail(email, permBanEmails)
}

func AllowedBanDurations(actor Actor) []BanDuration {
	if actor.IsAdmin {
		return []BanDuration{Ban7Days, Ban30Days, BanPermanent}
	}
	if actor.CanPermBan {
		return []BanDuration{Ban7Days, BanPermanent}
	}
	if actor.IsModerator {
		return []BanDuration{Ban7Days}
	}
	return nil
}

func CanBanDuration(actor Actor, duration BanDuration) bool {
	for _, d := range AllowedBanDurations(actor) {
		if d == duration {
			return true
		}
	}
	return false
}

func (s *Service) WithAccessFlags(actor Actor, adminEmails, modEmails, permBanEmails []string) Actor {
	actor.IsAdmin = IsAdminEmail(actor.Email, adminEmails)
	perms, _ := s.LoadPermissions(context.Background(), actor.ID)
	applyStaffFlags(&actor, modEmails, permBanEmails, perms)
	return actor
}

func (s *Service) EnrichActor(ctx context.Context, actor Actor, adminEmails, modEmails, permBanEmails []string) (Actor, error) {
	actor.IsAdmin = IsAdminEmail(actor.Email, adminEmails)
	perms, err := s.LoadPermissions(ctx, actor.ID)
	if err != nil {
		return actor, err
	}
	applyStaffFlags(&actor, modEmails, permBanEmails, perms)

	info, err := s.LoadAccessInfo(ctx, actor.ID)
	if err != nil {
		return actor, err
	}
	actor.State = string(info.State)
	actor.EmailVerified = info.EmailVerified || info.IsAgent
	actor.BanReason = info.BanReason
	if info.BannedUntil != nil {
		t := info.BannedUntil.Format(time.RFC3339)
		actor.BannedUntil = &t
	}
	return actor, nil
}