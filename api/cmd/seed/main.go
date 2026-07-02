package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/config"
	"github.com/xuroi/xuroi/api/internal/db"
	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/service"
	xsite "github.com/xuroi/xuroi/api/internal/site"
)

type siteConfig struct {
	SiteID     string `json:"site_id"`
	Categories []struct {
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Forums      []struct {
			Slug            string `json:"slug"`
			Name            string `json:"name"`
			Description     string `json:"description"`
			AccessLevel     string `json:"access_level"`
			ListPublic      *bool  `json:"list_public"`
			PostModeration  *bool  `json:"post_moderation"`
		} `json:"forums"`
	} `json:"categories"`
}

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	forum := service.NewForum(pool, "act_system", xsite.Load().Posts)
	if err := forum.EnsureSystemActor(ctx); err != nil {
		log.Fatalf("system actor: %v", err)
	}

	sitePath := siteJSONPath()
	data, err := os.ReadFile(sitePath)
	if err != nil {
		log.Fatalf("read site config: %v", err)
	}

	var site siteConfig
	if err := json.Unmarshal(data, &site); err != nil {
		log.Fatalf("parse site config: %v", err)
	}

	seedAuthorID, err := ensureSeedPersona(ctx, pool)
	if err != nil {
		log.Fatalf("seed persona actor: %v", err)
	}

	for gi, group := range site.Categories {
		groupID, err := ensureCategory(ctx, pool, forum, ensureCategoryInput{
			Slug:        group.Slug,
			Name:        group.Name,
			Description: group.Description,
			SortOrder:   gi + 1,
			ParentID:    nil,
		})
		if err != nil {
			log.Fatalf("create group %s: %v", group.Slug, err)
		}

		for fi, forumCfg := range group.Forums {
			_, err := ensureCategory(ctx, pool, forum, ensureCategoryInput{
				Slug:           forumCfg.Slug,
				Name:           forumCfg.Name,
				Description:    forumCfg.Description,
				SortOrder:      fi + 1,
				ParentID:       &groupID,
				AccessLevel:    forumCfg.AccessLevel,
				ListPublic:     forumCfg.ListPublic,
				PostModeration: forumCfg.PostModeration,
			})
			if err != nil {
				log.Fatalf("create forum %s: %v", forumCfg.Slug, err)
			}
		}
	}

	if err := retireLegacyCategories(ctx, pool, forum); err != nil {
		log.Fatalf("retire legacy categories: %v", err)
	}

	var welcomeExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM threads WHERE slug = 'welcome-back-to-puttertalk' AND deleted_at IS NULL)
	`).Scan(&welcomeExists); err != nil {
		log.Fatalf("check welcome thread: %v", err)
	}
	welcomeSlug := firstExistingSlug(ctx, pool, "general-discussion", "putter-talk")
	if welcomeSlug != "" && !welcomeExists {
			threadEvt, err := forum.CreateThread(ctx, service.CreateThreadInput{
				CategoryID:   categoryIDForSlug(ctx, pool, welcomeSlug),
				Title:        "Welcome back to PutterTalk",
				AuthorID:     seedAuthorID,
				BodyMarkdown: "We're relaunching. Introduce yourself and tell us what's in your bag.",
				BodyHTML:     "<p>We're relaunching. Introduce yourself and tell us what's in your bag.</p>",
			})
			if err != nil {
				log.Fatalf("welcome thread: %v", err)
			}

			var threadPayload events.ThreadCreated
			if err := json.Unmarshal(threadEvt.Payload, &threadPayload); err != nil {
				log.Fatalf("parse thread payload: %v", err)
			}
			log.Printf("created welcome thread %s", threadPayload.ThreadID)
	}

	fmt.Println("seed complete")
}

const seedPersonaName = "PutterTalk"
const seedPersonaActorID = "act_seed_persona"

// ensureSeedPersona returns the seed persona used for welcome threads.
// Re-runs must not mint a new member row each time.
func ensureSeedPersona(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		SELECT a.id
		FROM actors a
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		WHERE a.type = 'human'
		  AND LOWER(TRIM(a.display_name)) IN ('puttertalk', 'doug')
		  AND e.actor_id IS NULL
		ORDER BY
		  (SELECT COUNT(*)::int FROM posts p WHERE p.author_id = a.id AND p.deleted_at IS NULL) DESC,
		  a.created_at ASC
		LIMIT 1
	`).Scan(&id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		id = seedPersonaActorID
		if _, err := pool.Exec(ctx, `
			INSERT INTO actors (id, type, display_name, disclosure_required)
			VALUES ($1, 'human', $2, FALSE)
			ON CONFLICT (id) DO NOTHING
		`, id, seedPersonaName); err != nil {
			return "", err
		}
	} else if _, err := pool.Exec(ctx, `
		UPDATE actors SET display_name = $1 WHERE id = $2 AND display_name <> $1
	`, seedPersonaName, id); err != nil {
		return "", err
	}

	tag, err := pool.Exec(ctx, `
		DELETE FROM actors a
		WHERE a.type = 'human'
		  AND LOWER(TRIM(a.display_name)) IN ('doug', 'puttertalk')
		  AND a.id <> $1
		  AND NOT EXISTS (SELECT 1 FROM actor_emails e WHERE e.actor_id = a.id)
		  AND NOT EXISTS (SELECT 1 FROM posts p WHERE p.author_id = a.id AND p.deleted_at IS NULL)
		  AND NOT EXISTS (SELECT 1 FROM threads t WHERE t.author_id = a.id AND t.deleted_at IS NULL)
	`, id)
	if err != nil {
		return "", err
	}
	if n := tag.RowsAffected(); n > 0 {
		log.Printf("removed %d duplicate seed persona actor(s)", n)
	}
	return id, nil
}

type ensureCategoryInput struct {
	Slug           string
	Name           string
	Description    string
	SortOrder      int
	ParentID       *string
	AccessLevel    string
	ListPublic     *bool
	PostModeration *bool
}

func ensureCategory(ctx context.Context, pool *pgxpool.Pool, forum *service.Forum, in ensureCategoryInput) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1`, in.Slug).Scan(&id)
	if err == nil {
		log.Printf("skip existing category: %s", in.Slug)
		return id, nil
	}

	evt, err := forum.CreateCategory(ctx, service.CreateCategoryInput{
		Slug:           in.Slug,
		Name:           in.Name,
		Description:    in.Description,
		SortOrder:      in.SortOrder,
		ParentID:       in.ParentID,
		AccessLevel:    in.AccessLevel,
		ListPublic:     in.ListPublic,
		PostModeration: in.PostModeration,
		ActorID:        "act_system",
	})
	if err != nil {
		return "", err
	}

	var payload events.CategoryCreated
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return "", err
	}
	log.Printf("created category %s (event %s)", in.Slug, evt.ID)
	return payload.CategoryID, nil
}

func siteJSONPath() string {
	if p := os.Getenv("SITE_JSON"); p != "" {
		return p
	}
	return filepath.Join("..", "sites", "puttertalk", "site.json")
}

var legacyFlatSlugs = map[string]string{
	"putter-talk":     "general-discussion",
	"witb":            "collections",
	"equipment":       "equipment-reviews",
	"course-strategy": "strategy-technique",
	"bst":             "free-classifieds",
	"feedback":        "site-feedback",
}

func retireLegacyCategories(ctx context.Context, pool *pgxpool.Pool, forum *service.Forum) error {
	for fromSlug, toSlug := range legacyFlatSlugs {
		if err := migrateLegacyCategory(ctx, pool, forum, fromSlug, toSlug); err != nil {
			return err
		}
	}
	rows, err := pool.Query(ctx, `
		SELECT id, slug FROM categories
		WHERE parent_id IS NULL AND (slug = 'test-cat' OR slug LIKE 'test-cat-%')
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, slug string
		if err := rows.Scan(&id, &slug); err != nil {
			return err
		}
		if err := migrateLegacyCategory(ctx, pool, forum, slug, "general-discussion"); err != nil {
			log.Printf("skip test category %s: %v", slug, err)
		}
	}
	return rows.Err()
}

func migrateLegacyCategory(ctx context.Context, pool *pgxpool.Pool, forum *service.Forum, fromSlug, toSlug string) error {
	var fromID string
	var threadCount int
	err := pool.QueryRow(ctx, `
		SELECT c.id,
		       (SELECT COUNT(*)::int FROM threads t WHERE t.category_id = c.id AND t.deleted_at IS NULL)
		FROM categories c
		WHERE c.slug = $1 AND c.parent_id IS NULL
	`, fromSlug).Scan(&fromID, &threadCount)
	if err != nil {
		return nil
	}

	var toID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1`, toSlug).Scan(&toID); err != nil {
		log.Printf("keep legacy category %s (target %s missing)", fromSlug, toSlug)
		return nil
	}

	if threadCount > 0 {
		if _, err := pool.Exec(ctx, `UPDATE threads SET category_id = $1 WHERE category_id = $2`, toID, fromID); err != nil {
			return err
		}
		log.Printf("moved %d thread(s) from %s to %s", threadCount, fromSlug, toSlug)
	}

	if _, err := forum.DeleteCategory(ctx, fromID, "act_system", false); err != nil {
		log.Printf("skip delete %s: %v", fromSlug, err)
		return nil
	}
	log.Printf("removed legacy category %s", fromSlug)
	return nil
}

func firstExistingSlug(ctx context.Context, pool *pgxpool.Pool, slugs ...string) string {
	for _, slug := range slugs {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE slug = $1)`, slug).Scan(&exists); err != nil {
			log.Fatalf("check slug %s: %v", slug, err)
		}
		if exists {
			return slug
		}
	}
	return ""
}

func categoryIDForSlug(ctx context.Context, pool *pgxpool.Pool, slug string) string {
	var id string
	err := pool.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1`, slug).Scan(&id)
	if err != nil {
		log.Fatalf("category slug %s: %v", slug, err)
	}
	return id
}