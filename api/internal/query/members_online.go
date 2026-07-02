package query

import (
	"context"
	"fmt"
	"time"

	"github.com/xuroi/xuroi/api/internal/models"
)

const onlineWindow = 15 * time.Minute

type OnlineMember struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"display_name"`
	URL          string    `json:"url"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	LastActiveAt time.Time `json:"last_active_at"`
}

type OnlineMembersResponse struct {
	Count   int            `json:"count"`
	Members []OnlineMember `json:"members"`
}

func (r *Reader) OnlineMembers(ctx context.Context, staffView bool, viewerID string) (OnlineMembersResponse, error) {
	cutoff := time.Now().Add(-onlineWindow)
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.display_name, COALESCE(a.avatar_url, ''), a.last_active_at, a.hide_online_status
		FROM actors a
		WHERE a.type = 'human'
		  AND a.last_active_at IS NOT NULL
		  AND a.last_active_at > $1
		ORDER BY a.last_active_at DESC
		LIMIT 100
	`, cutoff)
	if err != nil {
		return OnlineMembersResponse{}, fmt.Errorf("online members: %w", err)
	}
	defer rows.Close()

	var members []OnlineMember
	visibleCount := 0
	for rows.Next() {
		var m OnlineMember
		var hidden bool
		if err := rows.Scan(&m.ID, &m.DisplayName, &m.AvatarURL, &m.LastActiveAt, &hidden); err != nil {
			return OnlineMembersResponse{}, err
		}
		m.URL = models.UserURL(m.DisplayName)
		include := !hidden || staffView || (viewerID != "" && m.ID == viewerID)
		if include {
			visibleCount++
			members = append(members, m)
		}
	}
	if err := rows.Err(); err != nil {
		return OnlineMembersResponse{}, err
	}
	if members == nil {
		members = []OnlineMember{}
	}
	return OnlineMembersResponse{Count: visibleCount, Members: members}, nil
}

func (r *Reader) ActorHideOnline(ctx context.Context, actorID string) (bool, error) {
	var hidden bool
	err := r.pool.QueryRow(ctx, `
		SELECT hide_online_status FROM actors WHERE id = $1
	`, actorID).Scan(&hidden)
	return hidden, err
}

func (r *Reader) SetActorHideOnline(ctx context.Context, actorID string, hidden bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE actors SET hide_online_status = $2 WHERE id = $1
	`, actorID, hidden)
	return err
}

