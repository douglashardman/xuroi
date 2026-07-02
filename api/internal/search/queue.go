package search

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnqueueTx schedules an entity for async indexing (debounced per entity_id).
func EnqueueTx(ctx context.Context, tx pgx.Tx, entityID, docType string) error {
	if entityID == "" || (docType != "thread" && docType != "post") {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO search_index_queue (entity_id, doc_type, enqueued_at)
		VALUES ($1, $2, now())
		ON CONFLICT (entity_id) DO UPDATE SET doc_type = EXCLUDED.doc_type, enqueued_at = now()
	`, entityID, docType)
	if err != nil {
		return fmt.Errorf("enqueue search: %w", err)
	}
	return nil
}

// EnqueueAllPending marks every live thread and post for indexing.
func EnqueueAllPending(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `TRUNCATE search_documents, search_index_queue`); err != nil {
		return 0, err
	}

	var threadIDs []string
	rows, err := tx.Query(ctx, `SELECT t.id FROM threads t WHERE t.deleted_at IS NULL`)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		threadIDs = append(threadIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	var postIDs []string
	rows2, err := tx.Query(ctx, `
		SELECT p.id FROM posts p
		JOIN threads t ON t.id = p.thread_id
		WHERE p.deleted_at IS NULL AND t.deleted_at IS NULL
		  AND p.moderation_status = 'approved'
		  AND NOT p.is_op
	`)
	if err != nil {
		return 0, err
	}
	for rows2.Next() {
		var id string
		if err := rows2.Scan(&id); err != nil {
			rows2.Close()
			return 0, err
		}
		postIDs = append(postIDs, id)
	}
	if err := rows2.Err(); err != nil {
		rows2.Close()
		return 0, err
	}
	rows2.Close()

	n := 0
	for _, id := range threadIDs {
		if err := EnqueueTx(ctx, tx, id, "thread"); err != nil {
			return n, err
		}
		n++
	}
	for _, id := range postIDs {
		if err := EnqueueTx(ctx, tx, id, "post"); err != nil {
			return n, err
		}
		n++
	}

	return n, tx.Commit(ctx)
}

// RemoveThread deletes all search documents for a thread.
func RemoveThreadTx(ctx context.Context, tx pgx.Tx, threadID string) error {
	_, err := tx.Exec(ctx, `DELETE FROM search_documents WHERE thread_id = $1`, threadID)
	if err != nil {
		return fmt.Errorf("remove thread search docs: %w", err)
	}
	_, err = tx.Exec(ctx, `DELETE FROM search_index_queue WHERE entity_id = $1`, threadID)
	return err
}