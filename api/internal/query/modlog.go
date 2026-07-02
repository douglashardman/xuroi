package query

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xuroi/xuroi/api/internal/events"
)

type ModLogEntry struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	ActorName string          `json:"actor_name"`
	CreatedAt time.Time       `json:"created_at"`
	Summary   string          `json:"summary"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

var modEventTypes = []string{
	events.TypePostModerated,
	events.TypePostReported,
	events.TypeThreadReported,
	events.TypeThreadLocked,
	events.TypeThreadUnlocked,
	events.TypeThreadPinned,
	events.TypeThreadUnpinned,
	events.TypeThreadDeleted,
	events.TypeThreadMoved,
	events.TypePostDeleted,
}

func (r *Reader) ModLog(ctx context.Context, limit int) ([]ModLogEntry, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.type, e.created_at, e.payload, COALESCE(a.display_name, 'system')
		FROM events e
		LEFT JOIN actors a ON a.id = e.actor_id
		WHERE e.type = ANY($1)
		ORDER BY e.created_at DESC
		LIMIT $2
	`, modEventTypes, limit)
	if err != nil {
		return nil, fmt.Errorf("mod log: %w", err)
	}
	defer rows.Close()

	var out []ModLogEntry
	for rows.Next() {
		var entry ModLogEntry
		if err := rows.Scan(&entry.ID, &entry.Type, &entry.CreatedAt, &entry.Payload, &entry.ActorName); err != nil {
			return nil, err
		}
		entry.Summary = summarizeModEvent(entry.Type, entry.Payload)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func summarizeModEvent(evtType string, payload json.RawMessage) string {
	switch evtType {
	case events.TypePostModerated:
		return "Post moderation action"
	case events.TypePostReported:
		return "Post reported"
	case events.TypeThreadReported:
		return "Thread reported"
	case events.TypeThreadLocked:
		return "Thread locked"
	case events.TypeThreadUnlocked:
		return "Thread unlocked"
	case events.TypeThreadPinned:
		return "Thread pinned"
	case events.TypeThreadUnpinned:
		return "Thread unpinned"
	case events.TypeThreadDeleted:
		return "Thread deleted"
	case events.TypeThreadMoved:
		return "Thread moved"
	case events.TypePostDeleted:
		return "Post removed"
	default:
		return evtType
	}
}