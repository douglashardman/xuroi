package search

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/intelligence"
	"github.com/xuroi/xuroi/api/internal/models"
)

type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ProcessBatch drains the queue and upserts search documents.
func (s *Service) ProcessBatch(ctx context.Context, limit int) (int, error) {
	if limit < 1 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT entity_id, doc_type
		FROM search_index_queue
		ORDER BY enqueued_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type item struct {
		id      string
		docType string
	}
	var batch []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.docType); err != nil {
			return 0, err
		}
		batch = append(batch, it)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(batch) == 0 {
		return 0, nil
	}

	processed := 0
	for _, it := range batch {
		if err := s.indexOne(ctx, it.id, it.docType); err != nil {
			return processed, err
		}
		_, err := s.pool.Exec(ctx, `DELETE FROM search_index_queue WHERE entity_id = $1`, it.id)
		if err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Service) indexOne(ctx context.Context, entityID, docType string) error {
	switch docType {
	case "thread":
		return s.indexThread(ctx, entityID)
	case "post":
		return s.indexPost(ctx, entityID)
	default:
		return nil
	}
}

func (s *Service) indexThread(ctx context.Context, threadID string) error {
	var title, slug, accessLevel, authorName, bodyHTML string
	var categoryID string
	var deleted bool
	err := s.pool.QueryRow(ctx, `
		SELECT t.title, t.slug, c.id, c.access_level, a.display_name,
		       COALESCE(p.body_html, ''), (t.deleted_at IS NOT NULL)
		FROM threads t
		JOIN categories c ON c.id = t.category_id
		JOIN actors a ON a.id = t.author_id
		LEFT JOIN posts p ON p.thread_id = t.id AND p.is_op AND p.deleted_at IS NULL
		WHERE t.id = $1
	`, threadID).Scan(&title, &slug, &categoryID, &accessLevel, &authorName, &bodyHTML, &deleted)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM search_documents WHERE entity_id = $1`, threadID)
		return nil
	}
	if deleted {
		_, err = s.pool.Exec(ctx, `DELETE FROM search_documents WHERE entity_id = $1`, threadID)
		return err
	}
	body := intelligence.StripHTML(bodyHTML)
	return s.upsert(ctx, threadID, "thread", threadID, categoryID, title, body, authorName, slug, title, accessLevel)
}

func (s *Service) indexPost(ctx context.Context, postID string) error {
	var threadID, categoryID, threadSlug, threadTitle, titlePrefix, accessLevel, authorName, bodyHTML string
	var moderationStatus string
	var deleted bool
	var isOP bool
	err := s.pool.QueryRow(ctx, `
		SELECT p.thread_id, c.id, t.slug, t.title, COALESCE(t.title_prefix, ''), c.access_level, a.display_name,
		       p.body_html, p.moderation_status, p.is_op, (p.deleted_at IS NOT NULL OR t.deleted_at IS NOT NULL)
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		JOIN categories c ON c.id = t.category_id
		JOIN actors a ON a.id = p.author_id
		WHERE p.id = $1
	`, postID).Scan(
		&threadID, &categoryID, &threadSlug, &threadTitle, &titlePrefix, &accessLevel, &authorName,
		&bodyHTML, &moderationStatus, &isOP, &deleted,
	)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM search_documents WHERE entity_id = $1`, postID)
		return nil
	}
	if deleted || moderationStatus != "approved" || isOP {
		_, err = s.pool.Exec(ctx, `DELETE FROM search_documents WHERE entity_id = $1`, postID)
		return err
	}
	body := intelligence.StripHTML(bodyHTML)
	displayTitle := models.ThreadDisplayTitle(titlePrefix, threadTitle)
	return s.upsert(ctx, postID, "post", threadID, categoryID, displayTitle, body, authorName, threadSlug, displayTitle, accessLevel)
}

func (s *Service) upsert(ctx context.Context, entityID, docType, threadID, categoryID, title, body, authorName, threadSlug, threadTitle, accessLevel string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO search_documents (
			entity_id, doc_type, thread_id, category_id, title, body,
			author_name, thread_slug, thread_title, access_level, search_vector, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			setweight(to_tsvector('english', $5), 'A') ||
			setweight(to_tsvector('english', $6), 'B') ||
			setweight(to_tsvector('english', $7), 'C'),
			now()
		)
		ON CONFLICT (entity_id) DO UPDATE SET
			doc_type = EXCLUDED.doc_type,
			thread_id = EXCLUDED.thread_id,
			category_id = EXCLUDED.category_id,
			title = EXCLUDED.title,
			body = EXCLUDED.body,
			author_name = EXCLUDED.author_name,
			thread_slug = EXCLUDED.thread_slug,
			thread_title = EXCLUDED.thread_title,
			access_level = EXCLUDED.access_level,
			search_vector = EXCLUDED.search_vector,
			updated_at = now()
	`, entityID, docType, threadID, categoryID, title, body, authorName, threadSlug, threadTitle, accessLevel)
	if err != nil {
		return fmt.Errorf("upsert search doc: %w", err)
	}
	return nil
}