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
	events.TypeThreadRestored,
	events.TypePostDeleted,
	events.TypePostRestored,
	events.TypePostEdited,
	events.TypeThreadAcceptedAnswerSet,
	events.TypeThreadAcceptedAnswerClr,
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
		var p events.PostModerated
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Post %s → %s", shortID(p.PostID), p.Status)
		}
		return "Post moderation action"
	case events.TypePostReported:
		var p events.PostReported
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Post reported — %s", p.Reason)
		}
		return "Post reported"
	case events.TypeThreadReported:
		var p events.ThreadReported
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Thread reported — %s", p.Reason)
		}
		return "Thread reported"
	case events.TypeThreadLocked:
		var p events.ThreadModeration
		if json.Unmarshal(payload, &p) == nil && p.LockReason != "" {
			return fmt.Sprintf("Thread locked — %s", p.LockReason)
		}
		return "Thread locked"
	case events.TypeThreadUnlocked:
		return "Thread unlocked"
	case events.TypeThreadPinned:
		return "Thread pinned"
	case events.TypeThreadUnpinned:
		return "Thread unpinned"
	case events.TypeThreadDeleted:
		var p events.ThreadDeleted
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Thread deleted (%s)", shortID(p.ThreadID))
		}
		return "Thread deleted"
	case events.TypeThreadRestored:
		var p events.ThreadRestored
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Thread restored (%s)", shortID(p.ThreadID))
		}
		return "Thread restored"
	case events.TypeThreadMoved:
		return "Thread moved"
	case events.TypePostDeleted:
		var p events.PostDeleted
		if json.Unmarshal(payload, &p) == nil {
			if p.Hard {
				return fmt.Sprintf("Post permanently deleted (%s)", shortID(p.PostID))
			}
			return fmt.Sprintf("Post removed (%s)", shortID(p.PostID))
		}
		return "Post removed"
	case events.TypePostRestored:
		var p events.PostRestored
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Post restored (%s)", shortID(p.PostID))
		}
		return "Post restored"
	case events.TypePostEdited:
		var p events.PostEdited
		if json.Unmarshal(payload, &p) == nil && p.EditReason != nil && *p.EditReason != "" {
			return fmt.Sprintf("Post edited — %s", *p.EditReason)
		}
		return "Post edited"
	case events.TypeThreadAcceptedAnswerSet:
		var p events.AcceptedAnswerChanged
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Accepted answer set (%s)", shortID(p.PostID))
		}
		return "Accepted answer set"
	case events.TypeThreadAcceptedAnswerClr:
		return "Accepted answer cleared"
	default:
		return evtType
	}
}

func shortID(id string) string {
	if len(id) <= 10 {
		return id
	}
	return id[len(id)-8:]
}