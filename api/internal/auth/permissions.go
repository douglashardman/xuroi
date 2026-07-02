package auth

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
)

const (
	PermModerator    = "moderator"
	PermBanPermanent = "perm_ban"
)

var StaffPermissions = []string{PermModerator, PermBanPermanent}

type PermissionInfo struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

func PermissionCatalog() []PermissionInfo {
	return []PermissionInfo{
		{
			ID:          PermModerator,
			Label:       "Moderator",
			Description: "Warn members, 7-day timeouts, remove posts, reports, pin/lock threads.",
		},
		{
			ID:          PermBanPermanent,
			Label:       "Permanent ban & takedown",
			Description: "Permanent bans and one-click permanent takedown (ban + purge all content).",
		},
	}
}

func ValidStaffPermission(p string) bool {
	return slices.Contains(StaffPermissions, p)
}

func HasPermission(perms []string, perm string) bool {
	return slices.Contains(perms, perm)
}

func (s *Service) LoadPermissions(ctx context.Context, actorID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT permission FROM actor_permissions
		WHERE actor_id = $1
		ORDER BY permission
	`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (s *Service) SetActorPermissions(ctx context.Context, actorID string, permissions []string, grantedBy string) error {
	seen := map[string]struct{}{}
	for _, p := range permissions {
		if !ValidStaffPermission(p) {
			return fmt.Errorf("invalid permission: %s", p)
		}
		seen[p] = struct{}{}
	}
	clean := make([]string, 0, len(seen))
	for p := range seen {
		clean = append(clean, p)
	}
	slices.Sort(clean)

	var actorType string
	err := s.pool.QueryRow(ctx, `SELECT type FROM actors WHERE id = $1`, actorID).Scan(&actorType)
	if err == pgx.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if actorType != "human" && actorType != "agent" {
		return ErrInvalidInput
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM actor_permissions WHERE actor_id = $1`, actorID)
	if err != nil {
		return err
	}
	for _, p := range clean {
		_, err = tx.Exec(ctx, `
			INSERT INTO actor_permissions (actor_id, permission, granted_by)
			VALUES ($1, $2, $3)
		`, actorID, p, grantedBy)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func applyStaffFlags(actor *Actor, modEmails, permBanEmails []string, perms []string) {
	actor.Permissions = perms
	actor.IsModerator = actor.IsAdmin ||
		IsModeratorEmail(actor.Email, modEmails) ||
		HasPermission(perms, PermModerator)
	actor.CanPermBan = actor.IsAdmin ||
		IsPermBanModerator(actor.Email, permBanEmails) ||
		HasPermission(perms, PermBanPermanent)
}