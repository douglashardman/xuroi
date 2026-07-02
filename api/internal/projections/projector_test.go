package projections_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/db"
	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/projections"
)

func TestRebuildFromEventLog(t *testing.T) {
	ctx := context.Background()
	databaseURL := "postgres://xuroi:xuroi_dev@localhost:5433/xuroi?sslmode=disable"

	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ensureActor(t, pool, "act_test")
	ensureActor(t, pool, "act_author")

	store := events.NewStore(pool)
	projector := projections.New()

	suffix := time.Now().Format("150405.000000")
	categoryID := "cat_test_" + suffix
	threadID := "thr_test_" + suffix
	postID := "pst_test_" + suffix
	categorySlug := "test-cat-" + suffix

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	catPayload, _ := json.Marshal(events.CategoryCreated{
		CategoryID: categoryID, Slug: categorySlug, Name: "Test", Description: "", SortOrder: 1,
	})
	catEvt, err := store.Append(ctx, tx, events.AppendInput{
		StreamID: events.StreamSite(), Type: events.TypeCategoryCreated,
		ActorID: strPtr("act_test"), Payload: json.RawMessage(catPayload),
	})
	if err != nil {
		t.Fatalf("append category: %v", err)
	}
	if err := projector.Apply(ctx, tx, catEvt); err != nil {
		t.Fatalf("project category: %v", err)
	}

	threadPayload, _ := json.Marshal(events.ThreadCreated{
		ThreadID: threadID, PostID: postID, CategoryID: categoryID,
		Title: "Test Thread", Slug: "test-thread-" + suffix, AuthorID: "act_author",
		BodyMarkdown: "hello", BodyHTML: "<p>hello</p>",
	})
	threadEvt, err := store.Append(ctx, tx, events.AppendInput{
		StreamID: events.StreamThread(threadID), Type: events.TypeThreadCreated,
		ActorID: strPtr("act_author"), Payload: json.RawMessage(threadPayload),
	})
	if err != nil {
		t.Fatalf("append thread: %v", err)
	}
	if err := projector.Apply(ctx, tx, threadEvt); err != nil {
		t.Fatalf("project thread: %v", err)
	}

	replyPayload, _ := json.Marshal(events.PostCreated{
		PostID: "pst_reply_" + suffix, ThreadID: threadID,
		AuthorID: "act_author", BodyMarkdown: "reply", BodyHTML: "<p>reply</p>",
	})
	replyEvt, err := store.Append(ctx, tx, events.AppendInput{
		StreamID: events.StreamThread(threadID), Type: events.TypePostCreated,
		ActorID: strPtr("act_author"), Payload: json.RawMessage(replyPayload),
	})
	if err != nil {
		t.Fatalf("append post: %v", err)
	}
	if err := projector.Apply(ctx, tx, replyEvt); err != nil {
		t.Fatalf("project post: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	all, err := store.ListAll(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	rebuildTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("rebuild begin: %v", err)
	}
	defer rebuildTx.Rollback(ctx)
	if err := projector.Rebuild(ctx, rebuildTx, all); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if err := rebuildTx.Commit(ctx); err != nil {
		t.Fatalf("rebuild commit: %v", err)
	}

	var replyCount int
	err = pool.QueryRow(ctx, `SELECT reply_count FROM threads WHERE id = $1`, threadID).Scan(&replyCount)
	if err != nil {
		t.Fatalf("query thread: %v", err)
	}
	if replyCount != 1 {
		t.Fatalf("expected reply_count 1, got %d", replyCount)
	}
}

func ensureActor(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO actors (id, type, display_name) VALUES ($1, 'human', $2)
		ON CONFLICT (id) DO NOTHING
	`, id, id)
	if err != nil {
		t.Fatalf("ensure actor %s: %v", id, err)
	}
}

func strPtr(s string) *string { return &s }