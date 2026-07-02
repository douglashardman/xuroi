package query

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xuroi/xuroi/api/internal/events"
)

type AdminLogEntry struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	ActorName string          `json:"actor_name"`
	CreatedAt time.Time       `json:"created_at"`
	Summary   string          `json:"summary"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

var adminOnlyEventTypes = []string{
	events.TypeCategoryCreated,
	events.TypeCategoryUpdated,
	events.TypeCategoryDeleted,
	events.TypeAdminSettingsUpdated,
	events.TypeAdminUserBanned,
	events.TypeAdminUserUnbanned,
	events.TypeAdminBackupTriggered,
	events.TypeAdminEmailBanned,
}

func adminEventTypes() []string {
	out := make([]string, 0, len(adminOnlyEventTypes)+len(ModEventTypes))
	out = append(out, adminOnlyEventTypes...)
	out = append(out, ModEventTypes...)
	return out
}

func (r *Reader) AdminLog(ctx context.Context, limit int) ([]AdminLogEntry, error) {
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
	`, adminEventTypes(), limit)
	if err != nil {
		return nil, fmt.Errorf("admin log: %w", err)
	}
	defer rows.Close()

	var out []AdminLogEntry
	for rows.Next() {
		var entry AdminLogEntry
		if err := rows.Scan(&entry.ID, &entry.Type, &entry.CreatedAt, &entry.Payload, &entry.ActorName); err != nil {
			return nil, err
		}
		entry.Summary = summarizeAdminEvent(entry.Type, entry.Payload)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func summarizeAdminEvent(evtType string, payload json.RawMessage) string {
	switch evtType {
	case events.TypeCategoryCreated:
		var p events.CategoryCreated
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Category created — %s", p.Name)
		}
		return "Category created"
	case events.TypeCategoryUpdated:
		var p events.CategoryUpdated
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Category updated — %s", p.Name)
		}
		return "Category updated"
	case events.TypeCategoryDeleted:
		return "Category deleted"
	case events.TypeAdminSettingsUpdated:
		return "Site settings updated"
	case events.TypeAdminUserBanned:
		var p events.AdminUserAction
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("User banned — %s", p.TargetName)
		}
		return "User banned"
	case events.TypeAdminUserUnbanned:
		var p events.AdminUserAction
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("User restored — %s", p.TargetName)
		}
		return "User restored"
	case events.TypeAdminBackupTriggered:
		return "Backup triggered"
	case events.TypeAdminEmailBanned:
		var p events.AdminEmailBan
		if json.Unmarshal(payload, &p) == nil {
			return fmt.Sprintf("Email banned — %s", p.Email)
		}
		return "Email banned"
	default:
		return SummarizeModEvent(evtType, payload)
	}
}