package auth

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/xuroi/xuroi/api/internal/site"
	"github.com/xuroi/xuroi/api/internal/slug"
)

var ErrDisplayNameTaken = errors.New("display name taken")

// UnavailableReason explains why a display name cannot be used.
type UnavailableReason string

const (
	ReasonInvalid  UnavailableReason = "invalid"
	ReasonReserved UnavailableReason = "reserved"
	ReasonDenied   UnavailableReason = "denied"
	ReasonTaken    UnavailableReason = "taken"
)

func normalizeDisplayName(name string) string {
	return strings.TrimSpace(name)
}

func displayNameSlug(name string) string {
	return slug.FromDisplayName(name)
}

func validateDisplayName(name string) error {
	n := utf8.RuneCountInString(name)
	if n < 2 || n > 40 {
		return ErrInvalidInput
	}
	s := displayNameSlug(name)
	if s == "" || s == "member" {
		return ErrInvalidInput
	}
	return nil
}

// SetReservedDisplayNames merges site-configured names with built-in defaults.
func (s *Service) SetReservedDisplayNames(names []string) {
	s.reservedNames = BuildReservedSet(names)
}

func (s *Service) SetRegistrationPolicy(p site.RegistrationPolicy) {
	s.regPolicy = p.Normalized()
}

// DisplayNameAvailable reports whether the name is free (case-insensitive + slug).
func (s *Service) DisplayNameAvailable(ctx context.Context, displayName string) (available bool, nameSlug string, reason UnavailableReason, err error) {
	name := normalizeDisplayName(displayName)
	nameSlug = displayNameSlug(name)
	if err := validateDisplayName(name); err != nil {
		return false, nameSlug, ReasonInvalid, nil
	}
	if reservedDisplayName(name, s.reservedNames) {
		return false, nameSlug, ReasonReserved, nil
	}
	if s.regPolicy.UsernameDenied(name) || s.regPolicy.UsernameDenied(nameSlug) {
		return false, nameSlug, ReasonDenied, nil
	}
	taken, err := s.displayNameTaken(ctx, name, nameSlug)
	if err != nil {
		return false, nameSlug, ReasonTaken, err
	}
	if taken {
		return false, nameSlug, ReasonTaken, nil
	}
	return true, nameSlug, "", nil
}

func (s *Service) assertDisplayNameAvailable(ctx context.Context, displayName string) error {
	available, _, reason, err := s.DisplayNameAvailable(ctx, displayName)
	if err != nil {
		return err
	}
	if available {
		return nil
	}
	switch reason {
	case ReasonReserved:
		return ErrDisplayNameReserved
	case ReasonDenied:
		return ErrUsernameDenied
	default:
		return ErrDisplayNameTaken
	}
}

func (s *Service) displayNameTaken(ctx context.Context, displayName, wantSlug string) (bool, error) {
	var exact bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM actors
			WHERE type = 'human'
			  AND deleted_at IS NULL
			  AND LOWER(TRIM(display_name)) = LOWER(TRIM($1))
		)
	`, displayName).Scan(&exact)
	if err != nil {
		return false, err
	}
	if exact {
		return true, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT display_name FROM actors WHERE type = 'human' AND deleted_at IS NULL
	`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var existing string
		if err := rows.Scan(&existing); err != nil {
			return false, err
		}
		if displayNameSlug(existing) == wantSlug {
			return true, nil
		}
	}
	return false, rows.Err()
}