package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/ids"
	"github.com/xuroi/xuroi/api/internal/models"
	"github.com/xuroi/xuroi/api/internal/slug"
)

var (
	ErrDisabled      = errors.New("agents are not enabled on this site")
	ErrAlreadyHas    = errors.New("you already have an agent")
	ErrNotFound      = errors.New("agent not found")
	ErrNameTaken     = errors.New("agent name taken")
	ErrNameInvalid   = errors.New("agent name invalid")
	ErrHumanNameClash = errors.New("agent name matches an existing member name")
)

type Agent struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
	Bio         string `json:"bio,omitempty"`
	AgentLabel  string `json:"agent_label"`
	OwnerID     string `json:"owner_id"`
	OwnerName   string `json:"owner_name"`
	OwnerURL    string `json:"owner_url"`
}

// OwnerLabel returns e.g. "MrDoug's Agent".
func OwnerLabel(ownerDisplayName string) string {
	name := strings.TrimSpace(ownerDisplayName)
	if name == "" {
		return "Member's Agent"
	}
	return name + "'s Agent"
}

func validateAgentName(name string) error {
	n := utf8.RuneCountInString(strings.TrimSpace(name))
	if n < 2 || n > 40 {
		return ErrNameInvalid
	}
	if slug.FromDisplayName(name) == "" {
		return ErrNameInvalid
	}
	return nil
}

func nameTakenByAgent(ctx context.Context, pool *pgxpool.Pool, displayName string, exceptID string) (bool, error) {
	var taken bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM actors
			WHERE type = 'agent' AND deleted_at IS NULL
			  AND LOWER(TRIM(display_name)) = LOWER(TRIM($1))
			  AND ($2 = '' OR id <> $2)
		)
	`, displayName, exceptID).Scan(&taken)
	return taken, err
}

func nameClashesHuman(ctx context.Context, pool *pgxpool.Pool, displayName string) (bool, error) {
	want := strings.ToLower(slug.FromDisplayName(displayName))
	var exact bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM actors
			WHERE type = 'human' AND deleted_at IS NULL
			  AND LOWER(TRIM(display_name)) = LOWER(TRIM($1))
		)
	`, displayName).Scan(&exact)
	if err != nil || exact {
		return exact, err
	}
	rows, err := pool.Query(ctx, `
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
		if strings.ToLower(slug.FromDisplayName(existing)) == want {
			return true, nil
		}
	}
	return false, rows.Err()
}

func loadOwner(ctx context.Context, pool *pgxpool.Pool, ownerID string) (string, error) {
	var name string
	err := pool.QueryRow(ctx, `
		SELECT display_name FROM actors
		WHERE id = $1 AND type = 'human' AND deleted_at IS NULL
	`, ownerID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("owner not found")
	}
	return name, err
}

func scanAgent(id, displayName, bio, ownerID, ownerName string) Agent {
	return Agent{
		ID:          id,
		DisplayName: displayName,
		URL:         models.UserURL(displayName),
		Bio:         strings.TrimSpace(bio),
		AgentLabel:  OwnerLabel(ownerName),
		OwnerID:     ownerID,
		OwnerName:   ownerName,
		OwnerURL:    models.UserURL(ownerName),
	}
}

func GetByOwner(ctx context.Context, pool *pgxpool.Pool, ownerID string) (Agent, error) {
	var a Agent
	var bio string
	err := pool.QueryRow(ctx, `
		SELECT ag.id, ag.display_name, COALESCE(ag.bio, ''), ag.owner_actor_id, o.display_name
		FROM actors ag
		JOIN actors o ON o.id = ag.owner_actor_id
		WHERE ag.type = 'agent' AND ag.deleted_at IS NULL AND ag.owner_actor_id = $1
	`, ownerID).Scan(&a.ID, &a.DisplayName, &bio, &a.OwnerID, &a.OwnerName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Agent{}, ErrNotFound
	}
	if err != nil {
		return Agent{}, err
	}
	a = scanAgent(a.ID, a.DisplayName, bio, a.OwnerID, a.OwnerName)
	return a, nil
}

func Create(ctx context.Context, pool *pgxpool.Pool, ownerID, displayName, bio string) (Agent, error) {
	displayName = strings.TrimSpace(displayName)
	if err := validateAgentName(displayName); err != nil {
		return Agent{}, err
	}
	var has bool
	_ = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM actors
			WHERE type = 'agent' AND deleted_at IS NULL AND owner_actor_id = $1
		)
	`, ownerID).Scan(&has)
	if has {
		return Agent{}, ErrAlreadyHas
	}
	taken, err := nameTakenByAgent(ctx, pool, displayName, "")
	if err != nil {
		return Agent{}, err
	}
	if taken {
		return Agent{}, ErrNameTaken
	}
	clash, err := nameClashesHuman(ctx, pool, displayName)
	if err != nil {
		return Agent{}, err
	}
	if clash {
		return Agent{}, ErrHumanNameClash
	}
	ownerName, err := loadOwner(ctx, pool, ownerID)
	if err != nil {
		return Agent{}, err
	}

	agentID := ids.New("act_")
	_, err = pool.Exec(ctx, `
		INSERT INTO actors (id, type, display_name, disclosure_required, owner_actor_id, bio)
		VALUES ($1, 'agent', $2, TRUE, $3, NULLIF($4, ''))
	`, agentID, displayName, ownerID, strings.TrimSpace(bio))
	if err != nil {
		return Agent{}, fmt.Errorf("create agent: %w", err)
	}
	return scanAgent(agentID, displayName, bio, ownerID, ownerName), nil
}

func Update(ctx context.Context, pool *pgxpool.Pool, ownerID, displayName, bio string) (Agent, error) {
	existing, err := GetByOwner(ctx, pool, ownerID)
	if err != nil {
		return Agent{}, err
	}
	if displayName != "" {
		displayName = strings.TrimSpace(displayName)
		if err := validateAgentName(displayName); err != nil {
			return Agent{}, err
		}
		if !strings.EqualFold(displayName, existing.DisplayName) {
			taken, err := nameTakenByAgent(ctx, pool, displayName, existing.ID)
			if err != nil {
				return Agent{}, err
			}
			if taken {
				return Agent{}, ErrNameTaken
			}
			clash, err := nameClashesHuman(ctx, pool, displayName)
			if err != nil {
				return Agent{}, err
			}
			if clash {
				return Agent{}, ErrHumanNameClash
			}
		}
	} else {
		displayName = existing.DisplayName
	}
	_, err = pool.Exec(ctx, `
		UPDATE actors
		SET display_name = $2, bio = NULLIF($3, '')
		WHERE id = $1 AND type = 'agent' AND owner_actor_id = $4 AND deleted_at IS NULL
	`, existing.ID, displayName, strings.TrimSpace(bio), ownerID)
	if err != nil {
		return Agent{}, err
	}
	return scanAgent(existing.ID, displayName, bio, existing.OwnerID, existing.OwnerName), nil
}

func Remove(ctx context.Context, pool *pgxpool.Pool, ownerID string) error {
	tag, err := pool.Exec(ctx, `
		UPDATE actors SET deleted_at = now()
		WHERE type = 'agent' AND owner_actor_id = $1 AND deleted_at IS NULL
	`, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}