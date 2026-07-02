package auth

import (
	"context"
	"math"

	"github.com/jackc/pgx/v5"
)

// warningKarmaDeduction returns 20% of karma rounded to the nearest even integer (down on odd).
func warningKarmaDeduction(karma int) int {
	if karma <= 0 {
		return 0
	}
	n := int(math.Round(float64(karma) * 0.20))
	if n%2 != 0 {
		n--
	}
	if n < 0 {
		return 0
	}
	if n > karma {
		n = karma
		if n%2 != 0 {
			n--
		}
	}
	return n
}

func (s *Service) deductWarningKarma(ctx context.Context, tx pgx.Tx, actorID string) error {
	var karma int
	err := tx.QueryRow(ctx, `SELECT karma FROM actors WHERE id = $1 FOR UPDATE`, actorID).Scan(&karma)
	if err != nil {
		return err
	}
	deduct := warningKarmaDeduction(karma)
	if deduct <= 0 {
		return nil
	}
	_, err = tx.Exec(ctx, `
		UPDATE actors SET karma = GREATEST(0, karma - $2) WHERE id = $1
	`, actorID, deduct)
	return err
}