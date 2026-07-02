package projections

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/access"
	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/search"
)

type Projector struct{}

func New() *Projector {
	return &Projector{}
}

func (p *Projector) Apply(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	switch evt.Type {
	case events.TypeCategoryCreated:
		return p.applyCategoryCreated(ctx, tx, evt)
	case events.TypeCategoryUpdated:
		return p.applyCategoryUpdated(ctx, tx, evt)
	case events.TypeCategoryDeleted:
		return p.applyCategoryDeleted(ctx, tx, evt)
	case events.TypeThreadCreated:
		return p.applyThreadCreated(ctx, tx, evt)
	case events.TypePostCreated:
		return p.applyPostCreated(ctx, tx, evt)
	case events.TypePostEdited:
		return p.applyPostEdited(ctx, tx, evt)
	case events.TypePostReactionAdded:
		return p.applyPostReactionAdded(ctx, tx, evt)
	case events.TypePostReactionRemoved:
		return p.applyPostReactionRemoved(ctx, tx, evt)
	case events.TypePostDeleted:
		return p.applyPostDeleted(ctx, tx, evt)
	case events.TypeThreadLocked:
		return p.applyThreadLocked(ctx, tx, evt, true)
	case events.TypeThreadUnlocked:
		return p.applyThreadLocked(ctx, tx, evt, false)
	case events.TypeThreadPinned:
		return p.applyThreadPinned(ctx, tx, evt, true)
	case events.TypeThreadUnpinned:
		return p.applyThreadPinned(ctx, tx, evt, false)
	case events.TypeThreadDeleted:
		return p.applyThreadDeleted(ctx, tx, evt)
	case events.TypeThreadMoved:
		return p.applyThreadMoved(ctx, tx, evt)
	case events.TypePostReported:
		return p.applyPostReported(ctx, tx, evt)
	case events.TypeThreadReported:
		return p.applyThreadReported(ctx, tx, evt)
	case events.TypePostModerated:
		return p.applyPostModerated(ctx, tx, evt)
	default:
		return nil
	}
}

func (p *Projector) Rebuild(ctx context.Context, tx pgx.Tx, evts []events.Event) error {
	if _, err := tx.Exec(ctx, `
		TRUNCATE reactions, post_revisions, posts, threads, categories RESTART IDENTITY CASCADE
	`); err != nil {
		return fmt.Errorf("truncate projections: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE actors SET karma = 0`); err != nil {
		return fmt.Errorf("reset karma: %w", err)
	}

	for _, evt := range evts {
		if err := p.Apply(ctx, tx, evt); err != nil {
			return fmt.Errorf("apply %s seq %d: %w", evt.Type, evt.Sequence, err)
		}
	}
	return nil
}

func (p *Projector) applyCategoryCreated(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.CategoryCreated
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal category.created: %w", err)
	}

	levels, level := access.ResolveCategoryAccess(payload.AccessLevel, payload.AccessLevels)
	listPublic := access.ResolveListPublicAny(levels, payload.ListPublic)
	postMod := false
	if payload.PostModeration != nil {
		postMod = *payload.PostModeration
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO categories (id, slug, name, description, sort_order, parent_id, access_level, access_levels, list_public, post_moderation, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO NOTHING
	`, payload.CategoryID, payload.Slug, payload.Name, payload.Description, payload.SortOrder, payload.ParentID, level, levels, listPublic, postMod, evt.CreatedAt)
	return err
}

func (p *Projector) applyCategoryUpdated(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.CategoryUpdated
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal category.updated: %w", err)
	}

	levels, level := access.ResolveCategoryAccess(payload.AccessLevel, payload.AccessLevels)
	listPublic := access.ResolveListPublicAny(levels, payload.ListPublic)
	postMod := false
	if payload.PostModeration != nil {
		postMod = *payload.PostModeration
	}
	_, err := tx.Exec(ctx, `
		UPDATE categories
		SET slug = $2, name = $3, description = $4, sort_order = $5, parent_id = $6, access_level = $7, access_levels = $8, list_public = $9, post_moderation = $10
		WHERE id = $1
	`, payload.CategoryID, payload.Slug, payload.Name, payload.Description, payload.SortOrder, payload.ParentID, level, levels, listPublic, postMod)
	return err
}

func (p *Projector) applyCategoryDeleted(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.CategoryDeleted
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal category.deleted: %w", err)
	}

	_, err := tx.Exec(ctx, `DELETE FROM categories WHERE id = $1`, payload.CategoryID)
	return err
}

func (p *Projector) applyThreadCreated(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.ThreadCreated
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal thread.created: %w", err)
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO threads (id, category_id, title, slug, author_id, reply_count, created_at, last_activity_at)
		VALUES ($1, $2, $3, $4, $5, 0, $6, $6)
		ON CONFLICT (id) DO NOTHING
	`, payload.ThreadID, payload.CategoryID, payload.Title, payload.Slug, payload.AuthorID, evt.CreatedAt)
	if err != nil {
		return err
	}

	modStatus := "approved"
	if payload.ForcePending {
		modStatus = "pending"
	} else if moderated, err := p.categoryPostModeration(ctx, tx, payload.CategoryID); err != nil {
		return err
	} else if moderated {
		modStatus = "pending"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO posts (id, thread_id, author_id, position, body_markdown, body_html, author_ip, is_op, moderation_status, created_at)
		VALUES ($1, $2, $3, 1, $4, $5, NULLIF($6, ''), TRUE, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, payload.PostID, payload.ThreadID, payload.AuthorID, payload.BodyMarkdown, payload.BodyHTML, payload.AuthorIP, modStatus, evt.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE categories SET thread_count = thread_count + 1, post_count = post_count + 1
		WHERE id = $1
	`, payload.CategoryID)
	if err != nil {
		return err
	}
	if modStatus == "approved" {
		if err := search.EnqueueTx(ctx, tx, payload.ThreadID, "thread"); err != nil {
			return err
		}
		return search.EnqueueTx(ctx, tx, payload.PostID, "post")
	}
	return nil
}

func (p *Projector) applyPostCreated(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.PostCreated
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal post.created: %w", err)
	}

	var position int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) + 1 FROM posts WHERE thread_id = $1
	`, payload.ThreadID).Scan(&position)
	if err != nil {
		return fmt.Errorf("next post position: %w", err)
	}

	modStatus := "approved"
	var categoryID string
	if err := tx.QueryRow(ctx, `SELECT category_id FROM threads WHERE id = $1`, payload.ThreadID).Scan(&categoryID); err != nil {
		return fmt.Errorf("thread category: %w", err)
	}
	if payload.ForcePending {
		modStatus = "pending"
	} else if moderated, err := p.categoryPostModeration(ctx, tx, categoryID); err != nil {
		return err
	} else if moderated {
		modStatus = "pending"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO posts (id, thread_id, author_id, position, body_markdown, body_html, quoted_post_id, quote_markdown, author_ip, is_op, moderation_status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), FALSE, $10, $11)
		ON CONFLICT (id) DO NOTHING
	`, payload.PostID, payload.ThreadID, payload.AuthorID, position, payload.BodyMarkdown, payload.BodyHTML, payload.QuotedPostID, payload.QuoteMarkdown, payload.AuthorIP, modStatus, evt.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE threads
		SET reply_count = reply_count + 1, last_activity_at = $2
		WHERE id = $1
	`, payload.ThreadID, evt.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE categories SET post_count = post_count + 1 WHERE id = $1
	`, categoryID)
	if err != nil {
		return err
	}
	if modStatus == "approved" {
		if err := search.EnqueueTx(ctx, tx, payload.ThreadID, "thread"); err != nil {
			return err
		}
		return search.EnqueueTx(ctx, tx, payload.PostID, "post")
	}
	return nil
}

func (p *Projector) applyPostEdited(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.PostEdited
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal post.edited: %w", err)
	}

	var editorID string
	if evt.ActorID != nil {
		editorID = *evt.ActorID
	}

	var revision int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(revision), 0) + 1 FROM post_revisions WHERE post_id = $1
	`, payload.PostID).Scan(&revision)
	if err != nil {
		return fmt.Errorf("next revision: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE posts SET body_markdown = $2, body_html = $3, edited_at = $4
		WHERE id = $1
	`, payload.PostID, payload.BodyMarkdown, payload.BodyHTML, evt.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO post_revisions (post_id, revision, body_markdown, body_html, editor_id, edited_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, payload.PostID, revision, payload.BodyMarkdown, payload.BodyHTML, editorID, evt.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE threads SET last_activity_at = $2 WHERE id = $1
	`, payload.ThreadID, evt.CreatedAt)
	if err != nil {
		return err
	}
	if err := search.EnqueueTx(ctx, tx, payload.PostID, "post"); err != nil {
		return err
	}
	return search.EnqueueTx(ctx, tx, payload.ThreadID, "thread")
}

func (p *Projector) applyPostReactionAdded(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.PostReactionChanged
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal post.reaction_added: %w", err)
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO reactions (post_id, reactor_id, reaction_type, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, payload.PostID, payload.ReactorID, payload.ReactionType, evt.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE posts SET reaction_count = (
			SELECT count(*) FROM reactions WHERE post_id = $1
		) WHERE id = $1
	`, payload.PostID)
	if err != nil {
		return err
	}

	return p.adjustKarmaForReaction(ctx, tx, payload.PostID, payload.ReactorID, 1)
}

func (p *Projector) applyPostReactionRemoved(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.PostReactionChanged
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal post.reaction_removed: %w", err)
	}

	_, err := tx.Exec(ctx, `
		DELETE FROM reactions
		WHERE post_id = $1 AND reactor_id = $2 AND reaction_type = $3
	`, payload.PostID, payload.ReactorID, payload.ReactionType)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE posts SET reaction_count = (
			SELECT count(*) FROM reactions WHERE post_id = $1
		) WHERE id = $1
	`, payload.PostID)
	if err != nil {
		return err
	}

	return p.adjustKarmaForReaction(ctx, tx, payload.PostID, payload.ReactorID, -1)
}

func (p *Projector) adjustKarmaForReaction(ctx context.Context, tx pgx.Tx, postID, reactorID string, delta int) error {
	var authorID string
	err := tx.QueryRow(ctx, `
		SELECT author_id FROM posts WHERE id = $1
	`, postID).Scan(&authorID)
	if err != nil {
		return fmt.Errorf("karma post author: %w", err)
	}
	if authorID == reactorID {
		return nil
	}

	if delta > 0 {
		_, err = tx.Exec(ctx, `UPDATE actors SET karma = karma + $2 WHERE id = $1`, authorID, delta)
	} else {
		_, err = tx.Exec(ctx, `UPDATE actors SET karma = GREATEST(0, karma + $2) WHERE id = $1`, authorID, delta)
	}
	return err
}

func (p *Projector) applyPostDeleted(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.PostDeleted
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal post.deleted: %w", err)
	}

	var deleterID string
	if evt.ActorID != nil {
		deleterID = *evt.ActorID
	}

	var isOP bool
	err := tx.QueryRow(ctx, `
		SELECT is_op FROM posts WHERE id = $1 AND deleted_at IS NULL
	`, payload.PostID).Scan(&isOP)
	if err != nil {
		return fmt.Errorf("post deleted lookup: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE posts SET deleted_at = $2, deleted_by = NULLIF($3, '')
		WHERE id = $1
	`, payload.PostID, evt.CreatedAt, deleterID)
	if err != nil {
		return err
	}

	if !isOP {
		_, err = tx.Exec(ctx, `
			UPDATE threads SET reply_count = GREATEST(0, reply_count - 1)
			WHERE id = $1
		`, payload.ThreadID)
		if err != nil {
			return err
		}
	}

	var categoryID string
	err = tx.QueryRow(ctx, `SELECT category_id FROM threads WHERE id = $1`, payload.ThreadID).Scan(&categoryID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE categories SET post_count = GREATEST(0, post_count - 1) WHERE id = $1
	`, categoryID)
	if err != nil {
		return err
	}
	_, _ = tx.Exec(ctx, `DELETE FROM search_documents WHERE entity_id = $1`, payload.PostID)
	return search.EnqueueTx(ctx, tx, payload.ThreadID, "thread")
}

func (p *Projector) applyThreadLocked(ctx context.Context, tx pgx.Tx, evt events.Event, locked bool) error {
	var payload events.ThreadModeration
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal thread lock: %w", err)
	}
	_, err := tx.Exec(ctx, `UPDATE threads SET is_locked = $2 WHERE id = $1`, payload.ThreadID, locked)
	return err
}

func (p *Projector) applyThreadDeleted(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.ThreadDeleted
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal thread.deleted: %w", err)
	}

	var categoryID string
	err := tx.QueryRow(ctx, `SELECT category_id FROM threads WHERE id = $1`, payload.ThreadID).Scan(&categoryID)
	if err != nil {
		return fmt.Errorf("thread deleted lookup: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE threads SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, payload.ThreadID, evt.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE categories SET thread_count = GREATEST(0, thread_count - 1) WHERE id = $1
	`, categoryID)
	if err != nil {
		return err
	}
	return search.RemoveThreadTx(ctx, tx, payload.ThreadID)
}

func (p *Projector) applyThreadMoved(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.ThreadMoved
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal thread.moved: %w", err)
	}
	_, err := tx.Exec(ctx, `UPDATE threads SET category_id = $2 WHERE id = $1 AND deleted_at IS NULL`, payload.ThreadID, payload.ToCategory)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE categories SET thread_count = GREATEST(0, thread_count - 1) WHERE id = $1`, payload.FromCategory)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE categories SET thread_count = thread_count + 1 WHERE id = $1`, payload.ToCategory)
	if err != nil {
		return err
	}
	if err := search.EnqueueTx(ctx, tx, payload.ThreadID, "thread"); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id FROM posts WHERE thread_id = $1 AND deleted_at IS NULL`, payload.ThreadID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var postID string
		if err := rows.Scan(&postID); err != nil {
			return err
		}
		if err := search.EnqueueTx(ctx, tx, postID, "post"); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (p *Projector) applyThreadPinned(ctx context.Context, tx pgx.Tx, evt events.Event, pinned bool) error {
	var payload events.ThreadModeration
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal thread pin: %w", err)
	}
	_, err := tx.Exec(ctx, `UPDATE threads SET is_pinned = $2 WHERE id = $1`, payload.ThreadID, pinned)
	return err
}

func (p *Projector) applyThreadReported(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.ThreadReported
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal thread.reported: %w", err)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO thread_reports (id, thread_id, reporter_id, reason, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (thread_id, reporter_id) DO NOTHING
	`, payload.ReportID, payload.ThreadID, payload.ReporterID, payload.Reason, evt.CreatedAt)
	return err
}

func (p *Projector) applyPostReported(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.PostReported
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal post.reported: %w", err)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO post_reports (id, post_id, thread_id, reporter_id, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (post_id, reporter_id) DO NOTHING
	`, payload.ReportID, payload.PostID, payload.ThreadID, payload.ReporterID, payload.Reason, evt.CreatedAt)
	return err
}

func (p *Projector) applyPostModerated(ctx context.Context, tx pgx.Tx, evt events.Event) error {
	var payload events.PostModerated
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal post.moderated: %w", err)
	}
	if payload.Status != "approved" && payload.Status != "rejected" {
		return fmt.Errorf("invalid moderation status: %s", payload.Status)
	}

	_, err := tx.Exec(ctx, `
		UPDATE posts SET moderation_status = $2
		WHERE id = $1 AND deleted_at IS NULL
	`, payload.PostID, payload.Status)
	if err != nil {
		return err
	}

	if payload.Status == "approved" {
		if err := search.EnqueueTx(ctx, tx, payload.ThreadID, "thread"); err != nil {
			return err
		}
		return search.EnqueueTx(ctx, tx, payload.PostID, "post")
	}

	_, err = tx.Exec(ctx, `DELETE FROM search_documents WHERE entity_id = $1`, payload.PostID)
	return err
}

func (p *Projector) categoryPostModeration(ctx context.Context, tx pgx.Tx, categoryID string) (bool, error) {
	var moderated bool
	err := tx.QueryRow(ctx, `SELECT post_moderation FROM categories WHERE id = $1`, categoryID).Scan(&moderated)
	if err != nil {
		return false, fmt.Errorf("category moderation: %w", err)
	}
	return moderated, nil
}