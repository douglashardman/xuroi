package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuroi/xuroi/api/internal/models"
)

type PublicMember struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	URL         string    `json:"url"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	PostCount   int       `json:"post_count"`
	JoinedAt    time.Time `json:"joined_at"`
}

func (r *Reader) ListPublicMembers(ctx context.Context, q string, limit, offset int) ([]PublicMember, int, error) {
	q = strings.TrimSpace(strings.ToLower(q))
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	countSQL := `
		SELECT COUNT(*)
		FROM actors a
		WHERE a.type = 'human'
		  AND a.deleted_at IS NULL
		  AND a.state != 'banned'
	`
	args := []any{}
	if q != "" {
		countSQL += ` AND LOWER(a.display_name) LIKE $1`
		args = append(args, "%"+q+"%")
	}
	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listSQL := `
		SELECT a.id, a.display_name, COALESCE(a.avatar_url, ''),
		       (SELECT COUNT(*)::int FROM posts p WHERE p.author_id = a.id AND p.deleted_at IS NULL),
		       a.created_at
		FROM actors a
		WHERE a.type = 'human'
		  AND a.deleted_at IS NULL
		  AND a.state != 'banned'
	`
	listArgs := []any{}
	if q != "" {
		listSQL += ` AND LOWER(a.display_name) LIKE $1`
		listArgs = append(listArgs, "%"+q+"%")
	}
	listSQL += ` ORDER BY a.created_at DESC LIMIT $` + fmt.Sprint(len(listArgs)+1) + ` OFFSET $` + fmt.Sprint(len(listArgs)+2)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []PublicMember
	for rows.Next() {
		var m PublicMember
		var avatarURL string
		if err := rows.Scan(&m.ID, &m.DisplayName, &avatarURL, &m.PostCount, &m.JoinedAt); err != nil {
			return nil, 0, err
		}
		m.URL = models.UserURL(m.DisplayName)
		if avatarURL != "" {
			m.AvatarURL = avatarURL
		}
		members = append(members, m)
	}
	if members == nil {
		members = []PublicMember{}
	}
	return members, total, rows.Err()
}