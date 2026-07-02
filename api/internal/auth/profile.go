package auth

import (
	"context"
	"strings"
	"unicode/utf8"
)

const maxBioRunes = 500

func (s *Service) SetBio(ctx context.Context, actorID, bio string) error {
	bio = strings.TrimSpace(bio)
	if utf8.RuneCountInString(bio) > maxBioRunes {
		return ErrInvalidInput
	}
	_, err := s.pool.Exec(ctx, `UPDATE actors SET bio = $2 WHERE id = $1`, actorID, bio)
	return err
}