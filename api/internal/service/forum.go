package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/access"
	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/ids"
	"github.com/xuroi/xuroi/api/internal/intelligence"
	"github.com/xuroi/xuroi/api/internal/markdown"
	"github.com/xuroi/xuroi/api/internal/projections"
	"github.com/xuroi/xuroi/api/internal/site"
)

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

type Forum struct {
	pool        *pgxpool.Pool
	events      *events.Store
	projector   *projections.Projector
	systemActor string
	postPolicy  site.PostPolicy
}

func NewForum(pool *pgxpool.Pool, systemActorID string, postPolicy site.PostPolicy) *Forum {
	return &Forum{
		pool:        pool,
		events:      events.NewStore(pool),
		projector:   projections.New(),
		systemActor: systemActorID,
		postPolicy:  postPolicy,
	}
}

type CreateCategoryInput struct {
	Slug        string
	Name        string
	Description string
	SortOrder   int
	ParentID    *string
	AccessLevel    string
	ListPublic     *bool
	PostModeration *bool
	ActorID        string
}

type CreateThreadInput struct {
	CategoryID   string
	Title        string
	AuthorID     string
	BodyMarkdown string
	BodyHTML     string
	AuthorIP     string
}

type CreatePostInput struct {
	ThreadID     string
	AuthorID     string
	BodyMarkdown string
	BodyHTML     string
	QuotedPostID  *string
	QuoteMarkdown *string
	AuthorIP      string
}

func (f *Forum) EnsureSystemActor(ctx context.Context) error {
	_, err := f.pool.Exec(ctx, `
		INSERT INTO actors (id, type, display_name, disclosure_required)
		VALUES ($1, 'service', 'system', FALSE)
		ON CONFLICT (id) DO NOTHING
	`, f.systemActor)
	return err
}

func (f *Forum) CreateCategory(ctx context.Context, in CreateCategoryInput) (events.Event, error) {
	categoryID := ids.New("cat_")
	payload := events.CategoryCreated{
		CategoryID:  categoryID,
		Slug:        in.Slug,
		Name:        in.Name,
		Description: in.Description,
		SortOrder:   in.SortOrder,
		ParentID:    in.ParentID,
		AccessLevel:    access.NormalizeLevel(in.AccessLevel),
		ListPublic:     in.ListPublic,
		PostModeration: in.PostModeration,
	}

	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamSite(),
		Type:     events.TypeCategoryCreated,
		ActorID:  strPtr(in.ActorID),
		Payload:  payload,
	})
}

type UpdateCategoryInput struct {
	CategoryID  string
	Slug        string
	Name        string
	Description string
	SortOrder   int
	ParentID    *string
	AccessLevel    string
	ListPublic     *bool
	PostModeration *bool
	ActorID        string
}

func (f *Forum) UpdateCategory(ctx context.Context, in UpdateCategoryInput) (events.Event, error) {
	if in.CategoryID == "" || in.Slug == "" || in.Name == "" {
		return events.Event{}, errors.New("category_id, slug, and name required")
	}
	if err := f.validateCategoryParent(ctx, in.CategoryID, in.ParentID); err != nil {
		return events.Event{}, err
	}
	payload := events.CategoryUpdated{
		CategoryID:  in.CategoryID,
		Slug:        in.Slug,
		Name:        in.Name,
		Description: in.Description,
		SortOrder:   in.SortOrder,
		ParentID:    in.ParentID,
		AccessLevel:    access.NormalizeLevel(in.AccessLevel),
		ListPublic:     in.ListPublic,
		PostModeration: in.PostModeration,
	}
	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamSite(),
		Type:     events.TypeCategoryUpdated,
		ActorID:  strPtr(in.ActorID),
		Payload:  payload,
	})
}

func (f *Forum) DeleteCategory(ctx context.Context, categoryID, actorID string) (events.Event, error) {
	if categoryID == "" {
		return events.Event{}, errors.New("category_id required")
	}
	var childCount, threadCount int
	err := f.pool.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*)::int FROM categories WHERE parent_id = $1),
		  (SELECT COUNT(*)::int FROM threads WHERE category_id = $1 AND deleted_at IS NULL)
	`, categoryID).Scan(&childCount, &threadCount)
	if err != nil {
		return events.Event{}, fmt.Errorf("category usage: %w", err)
	}
	if childCount > 0 {
		return events.Event{}, errors.New("category has child forums")
	}
	if threadCount > 0 {
		return events.Event{}, errors.New("category has threads")
	}
	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamSite(),
		Type:     events.TypeCategoryDeleted,
		ActorID:  strPtr(actorID),
		Payload:  events.CategoryDeleted{CategoryID: categoryID},
	})
}

type ReorderCategoriesInput struct {
	Items   []events.CategoryReorderItem
	ActorID string
}

func (f *Forum) ReorderCategories(ctx context.Context, in ReorderCategoriesInput) ([]events.Event, error) {
	if len(in.Items) == 0 {
		return nil, errors.New("items required")
	}
	var out []events.Event
	for _, item := range in.Items {
		var slug, name, description, accessLevel string
		var listPublic, postModeration bool
		err := f.pool.QueryRow(ctx, `
			SELECT slug, name, description, access_level, list_public, post_moderation FROM categories WHERE id = $1
		`, item.CategoryID).Scan(&slug, &name, &description, &accessLevel, &listPublic, &postModeration)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("category %s not found", item.CategoryID)
		}
		if err != nil {
			return nil, err
		}
		if err := f.validateCategoryParent(ctx, item.CategoryID, item.ParentID); err != nil {
			return nil, err
		}
		pm := postModeration
		evt, err := f.UpdateCategory(ctx, UpdateCategoryInput{
			CategoryID:     item.CategoryID,
			Slug:           slug,
			Name:           name,
			Description:    description,
			SortOrder:      item.SortOrder,
			ParentID:       item.ParentID,
			AccessLevel:    accessLevel,
			ListPublic:     &listPublic,
			PostModeration: &pm,
			ActorID:        in.ActorID,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, evt)
	}
	return out, nil
}

func (f *Forum) validateCategoryParent(ctx context.Context, categoryID string, parentID *string) error {
	if parentID == nil {
		return nil
	}
	if *parentID == categoryID {
		return errors.New("category cannot be its own parent")
	}
	var parentParent *string
	err := f.pool.QueryRow(ctx, `SELECT parent_id FROM categories WHERE id = $1`, *parentID).Scan(&parentParent)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("parent category not found")
	}
	if err != nil {
		return err
	}
	if parentParent != nil {
		return errors.New("parent must be a top-level group")
	}
	return nil
}

func (f *Forum) CreateThread(ctx context.Context, in CreateThreadInput) (events.Event, error) {
	threadID := ids.New("thr_")
	postID := ids.New("pst_")
	slug := Slugify(in.Title)

	payload := events.ThreadCreated{
		ThreadID:     threadID,
		PostID:       postID,
		CategoryID:   in.CategoryID,
		Title:        in.Title,
		Slug:         slug,
		AuthorID:     in.AuthorID,
		BodyMarkdown: in.BodyMarkdown,
		BodyHTML:     in.BodyHTML,
		AuthorIP:     in.AuthorIP,
	}

	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(threadID),
		Type:     events.TypeThreadCreated,
		ActorID:  strPtr(in.AuthorID),
		Payload:  payload,
	})
}

var (
	ErrInvalidQuote      = errors.New("quoted post not in thread")
	ErrQuoteNotExcerpt   = errors.New("quote must be excerpt from original post")
	ErrForbiddenEdit     = errors.New("cannot edit this post")
	ErrEditWindowClosed  = errors.New("edit window closed")
	ErrEditDisabled      = errors.New("post editing disabled")
	ErrDeleteDisabled    = errors.New("post deletion disabled")
	ErrThreadLocked      = errors.New("thread is locked")
	ErrModeratorRequired = errors.New("moderator required")
	ErrAlreadyReported   = errors.New("already reported")
)

func (f *Forum) CreatePost(ctx context.Context, in CreatePostInput) (events.Event, error) {
	var quoteMD *string
	if in.QuotedPostID != nil {
		if in.QuoteMarkdown != nil {
			trimmed := strings.TrimSpace(*in.QuoteMarkdown)
			if trimmed != "" {
				quoteMD = &trimmed
			}
		}
		if err := f.validateQuote(ctx, in.ThreadID, *in.QuotedPostID, quoteMD); err != nil {
			return events.Event{}, err
		}
	}

	postID := ids.New("pst_")
	payload := events.PostCreated{
		PostID:        postID,
		ThreadID:      in.ThreadID,
		AuthorID:      in.AuthorID,
		BodyMarkdown:  in.BodyMarkdown,
		BodyHTML:      in.BodyHTML,
		QuotedPostID:  in.QuotedPostID,
		QuoteMarkdown: quoteMD,
		AuthorIP:      in.AuthorIP,
	}

	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(in.ThreadID),
		Type:     events.TypePostCreated,
		ActorID:  strPtr(in.AuthorID),
		Payload:  payload,
	})
}

type ToggleReactionResult struct {
	Liked bool `json:"liked"`
	Count int  `json:"count"`
}

type EditPostInput struct {
	PostID       string
	EditorID     string
	BodyMarkdown string
	BodyHTML     string
}

func (f *Forum) EditPost(ctx context.Context, in EditPostInput) (events.Event, error) {
	if strings.TrimSpace(in.BodyMarkdown) == "" {
		return events.Event{}, fmt.Errorf("body required")
	}

	var authorID, threadID string
	var createdAt time.Time
	var isLocked bool
	err := f.pool.QueryRow(ctx, `
		SELECT p.author_id, p.thread_id, p.created_at, t.is_locked
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		WHERE p.id = $1 AND p.deleted_at IS NULL AND t.deleted_at IS NULL
	`, in.PostID).Scan(&authorID, &threadID, &createdAt, &isLocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return events.Event{}, fmt.Errorf("post not found")
	}
	if err != nil {
		return events.Event{}, err
	}
	if authorID != in.EditorID {
		return events.Event{}, ErrForbiddenEdit
	}
	if isLocked {
		return events.Event{}, ErrThreadLocked
	}
	if !f.postPolicy.EditEnabled {
		return events.Event{}, ErrEditDisabled
	}
	if f.postPolicy.EditWindowMinutes <= 0 {
		return events.Event{}, ErrEditDisabled
	}
	if time.Since(createdAt) > time.Duration(f.postPolicy.EditWindowMinutes)*time.Minute {
		return events.Event{}, ErrEditWindowClosed
	}

	payload := events.PostEdited{
		PostID:       in.PostID,
		ThreadID:     threadID,
		BodyMarkdown: in.BodyMarkdown,
		BodyHTML:     in.BodyHTML,
	}

	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(threadID),
		Type:     events.TypePostEdited,
		ActorID:  strPtr(in.EditorID),
		Payload:  payload,
	})
}

func (f *Forum) ModerateThread(ctx context.Context, threadID string, pin, lock *bool) ([]events.Event, error) {
	var current struct {
		pinned bool
		locked bool
	}
	err := f.pool.QueryRow(ctx, `
		SELECT is_pinned, is_locked FROM threads WHERE id = $1 AND deleted_at IS NULL
	`, threadID).Scan(&current.pinned, &current.locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("thread not found")
	}
	if err != nil {
		return nil, err
	}

	var out []events.Event
	emit := func(evtType string) error {
		evt, err := f.appendAndProject(ctx, events.AppendInput{
			StreamID: events.StreamThread(threadID),
			Type:     evtType,
			Payload:  events.ThreadModeration{ThreadID: threadID},
		})
		if err != nil {
			return err
		}
		out = append(out, evt)
		return nil
	}

	if pin != nil && *pin != current.pinned {
		t := events.TypeThreadPinned
		if !*pin {
			t = events.TypeThreadUnpinned
		}
		if err := emit(t); err != nil {
			return out, err
		}
	}
	if lock != nil && *lock != current.locked {
		t := events.TypeThreadLocked
		if !*lock {
			t = events.TypeThreadUnlocked
		}
		if err := emit(t); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (f *Forum) DeletePost(ctx context.Context, postID, actorID string, isAdmin bool) (events.Event, error) {
	var threadID string
	var isOP bool
	err := f.pool.QueryRow(ctx, `
		SELECT thread_id, is_op FROM posts
		WHERE id = $1 AND deleted_at IS NULL
	`, postID).Scan(&threadID, &isOP)
	if errors.Is(err, pgx.ErrNoRows) {
		return events.Event{}, fmt.Errorf("post not found")
	}
	if err != nil {
		return events.Event{}, err
	}
	if !f.postPolicy.DeleteEnabled {
		return events.Event{}, ErrDeleteDisabled
	}
	if !isAdmin {
		return events.Event{}, ErrForbiddenEdit
	}
	if isOP {
		return events.Event{}, ErrForbiddenEdit
	}

	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(threadID),
		Type:     events.TypePostDeleted,
		ActorID:  strPtr(actorID),
		Payload: events.PostDeleted{
			PostID:   postID,
			ThreadID: threadID,
			Hard:     false,
		},
	})
}

func (f *Forum) ToggleReaction(ctx context.Context, postID, reactorID string) (ToggleReactionResult, error) {
	var threadID string
	err := f.pool.QueryRow(ctx, `
		SELECT thread_id FROM posts WHERE id = $1 AND deleted_at IS NULL
	`, postID).Scan(&threadID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ToggleReactionResult{}, fmt.Errorf("post not found")
	}
	if err != nil {
		return ToggleReactionResult{}, err
	}

	const reactionType = "like"
	var exists bool
	err = f.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM reactions
			WHERE post_id = $1 AND reactor_id = $2 AND reaction_type = $3
		)
	`, postID, reactorID, reactionType).Scan(&exists)
	if err != nil {
		return ToggleReactionResult{}, err
	}

	payload := events.PostReactionChanged{
		PostID:       postID,
		ThreadID:     threadID,
		ReactorID:    reactorID,
		ReactionType: reactionType,
	}

	evtType := events.TypePostReactionAdded
	if exists {
		evtType = events.TypePostReactionRemoved
	}

	_, err = f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(threadID),
		Type:     evtType,
		ActorID:  strPtr(reactorID),
		Payload:  payload,
	})
	if err != nil {
		return ToggleReactionResult{}, err
	}

	var count int
	err = f.pool.QueryRow(ctx, `SELECT reaction_count FROM posts WHERE id = $1`, postID).Scan(&count)
	if err != nil {
		return ToggleReactionResult{}, err
	}

	return ToggleReactionResult{Liked: !exists, Count: count}, nil
}

func (f *Forum) ReportPost(ctx context.Context, postID, reporterID, reason string) (events.Event, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) > 500 {
		reason = reason[:500]
	}

	var threadID string
	err := f.pool.QueryRow(ctx, `
		SELECT thread_id FROM posts WHERE id = $1 AND deleted_at IS NULL
	`, postID).Scan(&threadID)
	if errors.Is(err, pgx.ErrNoRows) {
		return events.Event{}, fmt.Errorf("post not found")
	}
	if err != nil {
		return events.Event{}, err
	}

	var exists bool
	err = f.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM post_reports WHERE post_id = $1 AND reporter_id = $2
		)
	`, postID, reporterID).Scan(&exists)
	if err != nil {
		return events.Event{}, err
	}
	if exists {
		return events.Event{}, ErrAlreadyReported
	}

	reportID := ids.New("rpt_")
	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(threadID),
		Type:     events.TypePostReported,
		ActorID:  strPtr(reporterID),
		Payload: events.PostReported{
			ReportID:   reportID,
			PostID:     postID,
			ThreadID:   threadID,
			ReporterID: reporterID,
			Reason:     reason,
		},
	})
}

func (f *Forum) validateQuote(ctx context.Context, threadID, quotedPostID string, quoteMarkdown *string) error {
	var quoteThread, bodyMD, bodyHTML string
	err := f.pool.QueryRow(ctx, `
		SELECT thread_id, body_markdown, body_html FROM posts WHERE id = $1 AND deleted_at IS NULL
	`, quotedPostID).Scan(&quoteThread, &bodyMD, &bodyHTML)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidQuote
	}
	if err != nil {
		return err
	}
	if quoteThread != threadID {
		return ErrInvalidQuote
	}
	if quoteMarkdown != nil && strings.TrimSpace(*quoteMarkdown) != "" {
		q := *quoteMarkdown
		if !markdown.IsExcerptOf(q, bodyMD) && !markdown.IsExcerptOf(q, intelligence.StripHTML(bodyHTML)) {
			return ErrQuoteNotExcerpt
		}
	}
	return nil
}

func (f *Forum) RebuildProjections(ctx context.Context) error {
	evts, err := f.events.ListAll(ctx)
	if err != nil {
		return err
	}

	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rebuild: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := f.projector.Rebuild(ctx, tx, evts); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (f *Forum) appendAndProject(ctx context.Context, in events.AppendInput) (events.Event, error) {
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return events.Event{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	evt, err := f.events.Append(ctx, tx, in)
	if err != nil {
		return events.Event{}, err
	}

	if err := f.projector.Apply(ctx, tx, evt); err != nil {
		return events.Event{}, fmt.Errorf("project: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return events.Event{}, fmt.Errorf("commit: %w", err)
	}

	return evt, nil
}

type StaffRemoveResult struct {
	PostID        string `json:"post_id"`
	ThreadID      string `json:"thread_id"`
	ThreadRemoved bool   `json:"thread_removed"`
}

type PurgeAuthorResult struct {
	PostsRemoved    int `json:"posts_removed"`
	ThreadsRemoved  int `json:"threads_removed"`
}

// StaffRemovePost soft-deletes a post (including OP). Staff bypasses delete_enabled.
// If the post is the only content in the thread (OP, reply_count 0), the thread is removed too.
func (f *Forum) StaffRemovePost(ctx context.Context, postID, staffID string) (StaffRemoveResult, error) {
	var threadID string
	var isOP bool
	var replyCount int
	err := f.pool.QueryRow(ctx, `
		SELECT p.thread_id, p.is_op, t.reply_count
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		WHERE p.id = $1 AND p.deleted_at IS NULL AND t.deleted_at IS NULL
	`, postID).Scan(&threadID, &isOP, &replyCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return StaffRemoveResult{}, fmt.Errorf("post not found")
	}
	if err != nil {
		return StaffRemoveResult{}, err
	}

	_, err = f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(threadID),
		Type:     events.TypePostDeleted,
		ActorID:  strPtr(staffID),
		Payload: events.PostDeleted{
			PostID:   postID,
			ThreadID: threadID,
			Reason:   "staff removal",
		},
	})
	if err != nil {
		return StaffRemoveResult{}, err
	}

	result := StaffRemoveResult{PostID: postID, ThreadID: threadID}
	if isOP && replyCount == 0 {
		_, err = f.appendAndProject(ctx, events.AppendInput{
			StreamID: events.StreamThread(threadID),
			Type:     events.TypeThreadDeleted,
			ActorID:  strPtr(staffID),
			Payload: events.ThreadDeleted{
				ThreadID: threadID,
				Reason:   "staff removal — spam thread",
			},
		})
		if err != nil {
			return StaffRemoveResult{}, err
		}
		result.ThreadRemoved = true
	}
	return result, nil
}

// PurgeAuthorContent removes all posts by an author and deletes any threads they started that are now empty.
func (f *Forum) PurgeAuthorContent(ctx context.Context, authorID, staffID string) (PurgeAuthorResult, error) {
	rows, err := f.pool.Query(ctx, `
		SELECT p.id
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		WHERE p.author_id = $1 AND p.deleted_at IS NULL AND t.deleted_at IS NULL
		ORDER BY p.created_at ASC
	`, authorID)
	if err != nil {
		return PurgeAuthorResult{}, err
	}
	defer rows.Close()

	var postIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return PurgeAuthorResult{}, err
		}
		postIDs = append(postIDs, id)
	}
	if err := rows.Err(); err != nil {
		return PurgeAuthorResult{}, err
	}

	result := PurgeAuthorResult{}
	for _, postID := range postIDs {
		if _, err := f.StaffRemovePost(ctx, postID, staffID); err != nil {
			if err.Error() == "post not found" {
				continue
			}
			return result, err
		}
		result.PostsRemoved++
	}

	threadRows, err := f.pool.Query(ctx, `
		SELECT t.id
		FROM threads t
		WHERE t.author_id = $1 AND t.deleted_at IS NULL
	`, authorID)
	if err != nil {
		return result, err
	}
	defer threadRows.Close()

	for threadRows.Next() {
		var threadID string
		if err := threadRows.Scan(&threadID); err != nil {
			return result, err
		}
		var remaining int
		err = f.pool.QueryRow(ctx, `
			SELECT COUNT(*)::int FROM posts
			WHERE thread_id = $1 AND deleted_at IS NULL
		`, threadID).Scan(&remaining)
		if err != nil {
			return result, err
		}
		if remaining > 0 {
			continue
		}
		_, err = f.appendAndProject(ctx, events.AppendInput{
			StreamID: events.StreamThread(threadID),
			Type:     events.TypeThreadDeleted,
			ActorID:  strPtr(staffID),
			Payload: events.ThreadDeleted{
				ThreadID: threadID,
				Reason:   "staff purge — author content removed",
			},
		})
		if err != nil {
			return result, err
		}
		result.ThreadsRemoved++
	}
	if err := threadRows.Err(); err != nil {
		return result, err
	}
	return result, nil
}

// ModeratePost approves or rejects a pending post.
func (f *Forum) ModeratePost(ctx context.Context, postID, moderatorID, status string) (events.Event, error) {
	if status != "approved" && status != "rejected" {
		return events.Event{}, errors.New("status must be approved or rejected")
	}
	var threadID string
	err := f.pool.QueryRow(ctx, `
		SELECT thread_id FROM posts WHERE id = $1 AND moderation_status = 'pending' AND deleted_at IS NULL
	`, postID).Scan(&threadID)
	if errors.Is(err, pgx.ErrNoRows) {
		return events.Event{}, fmt.Errorf("post not found")
	}
	if err != nil {
		return events.Event{}, err
	}
	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(threadID),
		Type:     events.TypePostModerated,
		ActorID:  strPtr(moderatorID),
		Payload: events.PostModerated{
			PostID:   postID,
			ThreadID: threadID,
			Status:   status,
		},
	})
}

// DeleteThread soft-deletes a thread (staff action).
func (f *Forum) DeleteThread(ctx context.Context, threadID, staffID string) (events.Event, error) {
	var exists bool
	err := f.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM threads WHERE id = $1 AND deleted_at IS NULL)
	`, threadID).Scan(&exists)
	if err != nil {
		return events.Event{}, err
	}
	if !exists {
		return events.Event{}, fmt.Errorf("thread not found")
	}
	return f.appendAndProject(ctx, events.AppendInput{
		StreamID: events.StreamThread(threadID),
		Type:     events.TypeThreadDeleted,
		ActorID:  strPtr(staffID),
		Payload: events.ThreadDeleted{
			ThreadID: threadID,
			Reason:   "staff removal",
		},
	})
}

func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugSanitizer.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "thread"
	}
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	return s
}

func strPtr(s string) *string {
	return &s
}