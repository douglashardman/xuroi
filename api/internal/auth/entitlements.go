package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/access"
)

type EntitlementRow struct {
	Entitlement string     `json:"entitlement"`
	Source      string     `json:"source"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	GrantedAt   time.Time  `json:"granted_at"`
}

func (s *Service) LoadEntitlements(ctx context.Context, actorID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT entitlement
		FROM actor_entitlements
		WHERE actor_id = $1
		  AND (expires_at IS NULL OR expires_at > now())
		ORDER BY entitlement
	`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var ent string
		if err := rows.Scan(&ent); err != nil {
			return nil, err
		}
		out = append(out, ent)
	}
	return out, rows.Err()
}

func (s *Service) ListEntitlements(ctx context.Context, actorID string) ([]EntitlementRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT entitlement, source, expires_at, granted_at
		FROM actor_entitlements
		WHERE actor_id = $1
		  AND (expires_at IS NULL OR expires_at > now())
		ORDER BY entitlement
	`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EntitlementRow
	for rows.Next() {
		var row EntitlementRow
		if err := rows.Scan(&row.Entitlement, &row.Source, &row.ExpiresAt, &row.GrantedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) SetEntitlements(ctx context.Context, actorID string, entitlements []string, grantedBy string) error {
	seen := map[string]struct{}{}
	for _, ent := range entitlements {
		if !access.ValidEntitlement(ent) {
			return fmt.Errorf("invalid entitlement: %s", ent)
		}
		seen[ent] = struct{}{}
	}

	var actorType string
	err := s.pool.QueryRow(ctx, `SELECT type FROM actors WHERE id = $1`, actorID).Scan(&actorType)
	if err == pgx.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if actorType != "human" {
		return ErrInvalidInput
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM actor_entitlements WHERE actor_id = $1 AND source = 'manual'`, actorID)
	if err != nil {
		return err
	}

	for ent := range seen {
		_, err = tx.Exec(ctx, `
			INSERT INTO actor_entitlements (actor_id, entitlement, source, granted_by)
			VALUES ($1, $2, 'manual', $3)
			ON CONFLICT (actor_id, entitlement) DO UPDATE
			SET source = 'manual', granted_by = EXCLUDED.granted_by, granted_at = now(), expires_at = NULL, external_ref = NULL
		`, actorID, ent, grantedBy)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Service) UpsertProviderEntitlement(ctx context.Context, actorID, entitlement, source, externalRef string, expiresAt *time.Time) error {
	if !access.ValidEntitlement(entitlement) {
		return fmt.Errorf("invalid entitlement: %s", entitlement)
	}
	if source != "stripe" && source != "patreon" {
		return fmt.Errorf("invalid source: %s", source)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO actor_entitlements (actor_id, entitlement, source, external_ref, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (actor_id, entitlement) DO UPDATE
		SET source = EXCLUDED.source,
		    external_ref = EXCLUDED.external_ref,
		    expires_at = EXCLUDED.expires_at,
		    granted_at = now()
	`, actorID, entitlement, source, externalRef, expiresAt)
	return err
}