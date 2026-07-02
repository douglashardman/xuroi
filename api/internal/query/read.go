package query

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/access"
	"github.com/xuroi/xuroi/api/internal/intelligence"
	"github.com/xuroi/xuroi/api/internal/markdown"
	"github.com/xuroi/xuroi/api/internal/models"
	"github.com/xuroi/xuroi/api/internal/site"
	"github.com/xuroi/xuroi/api/internal/slug"
)

var ErrNotFound = errors.New("not found")
var ErrForbidden = errors.New("forbidden")

type Reader struct {
	pool         *pgxpool.Pool
	site         models.Site
	postPolicy   site.PostPolicy
	intelligence site.IntelligencePolicy
	seoPolicy    site.SEOPolicy
}

func (r *Reader) SetPostPolicy(p site.PostPolicy) {
	r.postPolicy = p
}

func (r *Reader) SetIntelligence(p site.IntelligencePolicy) {
	r.intelligence = p.Normalized()
}

func (r *Reader) SetSEOPolicy(p site.SEOPolicy) {
	r.seoPolicy = p
}

func NewReader(pool *pgxpool.Pool, site models.Site, postPolicy site.PostPolicy, intelligence site.IntelligencePolicy, seoPolicy site.SEOPolicy) *Reader {
	return &Reader{
		pool:         pool,
		site:         site,
		postPolicy:   postPolicy,
		intelligence: intelligence.Normalized(),
		seoPolicy:    seoPolicy,
	}
}

func (r *Reader) postHTML(html string) string {
	html = markdown.EnrichMediaImages(html)
	if r.seoPolicy.NofollowUserLinks {
		html = markdown.ApplyNofollow(html)
	}
	return html
}

func (r *Reader) Home(ctx context.Context, viewer access.Viewer) (models.HomeResponse, error) {
	groups, flat, err := r.listCategoryTree(ctx, viewer)
	if err != nil {
		return models.HomeResponse{}, err
	}
	if viewer.ActorID != nil && len(flat) > 0 {
		ids := make([]string, len(flat))
		for i, c := range flat {
			ids[i] = c.ID
		}
		counts, err := r.CategoryUnreadCounts(ctx, *viewer.ActorID, ids)
		if err != nil {
			return models.HomeResponse{}, err
		}
		for i := range flat {
			flat[i].UnreadCount = counts[flat[i].ID]
		}
		for gi := range groups {
			for fi := range groups[gi].Forums {
				groups[gi].Forums[fi].UnreadCount = counts[groups[gi].Forums[fi].ID]
			}
		}
	}
	return models.HomeResponse{Site: r.site, Groups: groups, Categories: flat}, nil
}

func (r *Reader) CategoryBySlug(ctx context.Context, slug string, page, perPage int, viewer access.Viewer) (models.CategoryPageResponse, error) {
	var cat models.CategorySummary
	var accessLevel string
	var accessLevels []string
	var listPublic bool
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, parent_id, sort_order, thread_count, post_count, access_level, access_levels, list_public
		FROM categories WHERE slug = $1
	`, slug).Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.Description, &cat.ParentID, &cat.SortOrder, &cat.ThreadCount, &cat.PostCount, &accessLevel, &accessLevels, &listPublic)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CategoryPageResponse{}, ErrNotFound
	}
	if err != nil {
		return models.CategoryPageResponse{}, fmt.Errorf("category: %w", err)
	}
	cat.URL = models.CategoryURL(cat.Slug)

	var childCount int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM categories WHERE parent_id = $1`, cat.ID).Scan(&childCount); err != nil {
		return models.CategoryPageResponse{}, fmt.Errorf("category children: %w", err)
	}
	if childCount > 0 {
		return models.CategoryPageResponse{}, ErrNotFound
	}

	cat = applyCategoryAccess(cat, accessLevel, accessLevels, listPublic, viewer)
	if !cat.CanView {
		return models.CategoryPageResponse{}, ErrForbidden
	}

	var total int
	err = r.pool.QueryRow(ctx, `
		SELECT count(*) FROM threads
		WHERE category_id = $1 AND deleted_at IS NULL
	`, cat.ID).Scan(&total)
	if err != nil {
		return models.CategoryPageResponse{}, fmt.Errorf("count threads: %w", err)
	}

	threads, err := r.listThreadsInCategory(ctx, cat.ID, page, perPage, viewer)
	if err != nil {
		return models.CategoryPageResponse{}, err
	}
	if viewer.ActorID != nil {
		counts, err := r.CategoryUnreadCounts(ctx, *viewer.ActorID, []string{cat.ID})
		if err != nil {
			return models.CategoryPageResponse{}, err
		}
		cat.UnreadCount = counts[cat.ID]
	}

	pagination := paginate(page, perPage, total, func(p int) string {
		if p == 1 {
			return models.CategoryURL(slug)
		}
		return fmt.Sprintf("%s?page=%d", models.CategoryURL(slug), p)
	})

	homeURL := "/community"
	breadcrumbs := []models.Breadcrumb{
		{Label: "Community", URL: &homeURL},
	}
	if cat.ParentID != nil {
		var parentName string
		if err := r.pool.QueryRow(ctx, `SELECT name FROM categories WHERE id = $1`, *cat.ParentID).Scan(&parentName); err == nil && parentName != "" {
			breadcrumbs = append(breadcrumbs, models.Breadcrumb{Label: parentName, URL: nil})
		}
	}
	breadcrumbs = append(breadcrumbs, models.Breadcrumb{Label: cat.Name, URL: nil})

	return models.CategoryPageResponse{
		Site:        r.site,
		Category:    cat,
		Threads:     threads,
		Pagination:  pagination,
		Breadcrumbs: breadcrumbs,
	}, nil
}

func (r *Reader) ThreadByID(ctx context.Context, id string, page, perPage int, viewer access.Viewer) (models.ThreadPageResponse, error) {
	var thread models.ThreadDetail
	var cat models.CategoryRef
	var accessLevel string
	var accessLevels []string
	err := r.pool.QueryRow(ctx, `
		SELECT t.id, t.title, t.slug, t.reply_count, t.view_count, t.is_locked, COALESCE(t.lock_reason, ''), t.is_pinned,
		       t.created_at, t.last_activity_at,
		       c.id, c.name, c.slug, c.access_level, c.access_levels
		FROM threads t
		JOIN categories c ON c.id = t.category_id
		WHERE t.id = $1 AND t.deleted_at IS NULL
	`, id).Scan(
		&thread.ID, &thread.Title, &thread.Slug, &thread.ReplyCount, &thread.ViewCount,
		&thread.IsLocked, &thread.LockReason, &thread.IsPinned, &thread.CreatedAt, &thread.LastActivityAt,
		&cat.ID, &cat.Name, &cat.Slug, &accessLevel, &accessLevels,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ThreadPageResponse{}, ErrNotFound
	}
	if err != nil {
		return models.ThreadPageResponse{}, fmt.Errorf("thread: %w", err)
	}
	levels := access.NormalizeLevels(accessLevels)
	if len(levels) == 0 {
		levels = []string{access.NormalizeLevel(accessLevel)}
	}
	if !viewer.CanViewAny(levels) {
		return models.ThreadPageResponse{}, ErrForbidden
	}
	visible, err := r.threadVisibleToViewer(ctx, id, viewer.ActorID, viewer.IsStaff)
	if err != nil {
		return models.ThreadPageResponse{}, fmt.Errorf("thread visibility: %w", err)
	}
	if !visible {
		return models.ThreadPageResponse{}, ErrNotFound
	}
	thread.URL = models.ThreadURL(thread.Slug, thread.ID)
	cat.URL = models.CategoryURL(cat.Slug)
	thread.Summary = r.threadSummary(ctx, id)
	if viewer.ActorID != nil {
		if watching, err := r.threadEmailWatching(ctx, *viewer.ActorID, id); err == nil {
			thread.EmailWatching = watching
		}
	}

	var total int
	err = r.pool.QueryRow(ctx, `
		SELECT count(*) FROM posts
		WHERE thread_id = $1 AND deleted_at IS NULL
	`, id).Scan(&total)
	if err != nil {
		return models.ThreadPageResponse{}, fmt.Errorf("count posts: %w", err)
	}

	if perPage < 1 {
		perPage = 25
	}
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	posts, err := r.listPosts(ctx, id, page, perPage, viewer)
	if err != nil {
		return models.ThreadPageResponse{}, err
	}
	if viewer.IsStaff {
		if err := r.annotateWarnedPosts(ctx, posts); err != nil {
			return models.ThreadPageResponse{}, err
		}
	}

	threadURL := thread.URL
	pagination := paginate(page, perPage, total, func(p int) string {
		if p == 1 {
			return threadURL
		}
		return fmt.Sprintf("%s?page=%d", threadURL, p)
	})

	homeURL := "/community"
	catURL := cat.URL
	resp := models.ThreadPageResponse{
		Site:       r.site,
		Thread:     thread,
		Category:   cat,
		Posts:      posts,
		Pagination: pagination,
		Breadcrumbs: []models.Breadcrumb{
			{Label: "Home", URL: &homeURL},
			{Label: cat.Name, URL: &catURL},
			{Label: thread.Title, URL: nil},
		},
	}
	resp.UI.ShowModBar = viewer.IsStaff
	resp.UI.SummaryLabel = r.intelligence.SummaryLabel
	if viewer.IsStaff {
		if n, err := r.OpenReportCount(ctx, id); err == nil {
			resp.UI.OpenReportCount = n
		}
	}
	return resp, nil
}

func (r *Reader) ThreadMeta(ctx context.Context, id string) (models.ThreadMeta, error) {
	var meta models.ThreadMeta
	var catName string
	err := r.pool.QueryRow(ctx, `
		SELECT t.id, t.title, t.slug, t.reply_count, t.created_at, t.last_activity_at, c.name
		FROM threads t
		JOIN categories c ON c.id = t.category_id
		WHERE t.id = $1 AND t.deleted_at IS NULL
	`, id).Scan(
		&meta.ThreadID, &meta.Title, &meta.Slug, &meta.ReplyCount,
		&meta.CreatedAt, &meta.LastActivity, &catName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ThreadMeta{}, ErrNotFound
	}
	if err != nil {
		return models.ThreadMeta{}, fmt.Errorf("thread meta: %w", err)
	}
	meta.URL = models.ThreadURL(meta.Slug, meta.ThreadID)
	meta.Category = catName
	meta.SummaryLabel = r.intelligence.SummaryLabel
	meta.Summary, meta.ModelVersion = r.threadIntelligence(ctx, id)

	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT a.display_name
		FROM posts p
		JOIN actors a ON a.id = p.author_id
		WHERE p.thread_id = $1 AND p.deleted_at IS NULL
		ORDER BY a.display_name
	`, id)
	if err != nil {
		return models.ThreadMeta{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return models.ThreadMeta{}, err
		}
		meta.Participants = append(meta.Participants, name)
	}
	return meta, rows.Err()
}

func (r *Reader) UserBySlug(ctx context.Context, nameSlug string) (models.UserProfile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.display_name, a.karma, a.created_at, COALESCE(a.avatar_url, ''), COALESCE(a.bio, ''),
		       COALESCE(a.last_active_at, (
		         SELECT MAX(p.created_at) FROM posts p
		         WHERE p.author_id = a.id AND p.deleted_at IS NULL
		       )) AS last_active_at,
		       a.hide_online_status,
		       (SELECT count(*) FROM posts p WHERE p.author_id = a.id AND p.deleted_at IS NULL)
		FROM actors a
		WHERE a.type = 'human'
	`)
	if err != nil {
		return models.UserProfile{}, fmt.Errorf("list actors: %w", err)
	}
	defer rows.Close()

	want := strings.ToLower(nameSlug)
	for rows.Next() {
		var profile models.UserProfile
		if err := rows.Scan(&profile.ID, &profile.DisplayName, &profile.Karma, &profile.JoinedAt, &profile.AvatarURL, &profile.Bio, &profile.LastActiveAt, &profile.HideOnline, &profile.PostCount); err != nil {
			return models.UserProfile{}, fmt.Errorf("scan actor: %w", err)
		}
		if slug.FromDisplayName(profile.DisplayName) == want {
			profile.URL = models.UserURL(profile.DisplayName)
			return profile, nil
		}
	}
	if err := rows.Err(); err != nil {
		return models.UserProfile{}, err
	}
	return models.UserProfile{}, ErrNotFound
}

func (r *Reader) RecentThreads(ctx context.Context, limit int, viewer access.Viewer, unreadOnly bool) (models.RecentThreadsResponse, error) {
	if limit < 1 {
		limit = 6
	}
	if limit > 50 {
		limit = 50
	}

	querySQL := `
		SELECT t.id, t.title, t.slug, t.reply_count, t.last_activity_at, c.name, c.slug, c.access_level, c.access_levels
		FROM threads t
		JOIN categories c ON c.id = t.category_id
		JOIN posts op ON op.thread_id = t.id AND op.is_op AND op.deleted_at IS NULL
	`
	args := []any{limit * 3}
	if unreadOnly && viewer.ActorID != nil {
		querySQL += `
		LEFT JOIN thread_reads tr ON tr.thread_id = t.id AND tr.actor_id = $2
		WHERE t.deleted_at IS NULL AND op.moderation_status = 'approved' AND ` + unreadSQL + `
		ORDER BY t.last_activity_at DESC
		LIMIT $1`
		args = append(args, *viewer.ActorID)
	} else {
		querySQL += `
		WHERE t.deleted_at IS NULL AND op.moderation_status = 'approved'
		ORDER BY t.last_activity_at DESC
		LIMIT $1`
	}

	rows, err := r.pool.Query(ctx, querySQL, args...)
	if err != nil {
		return models.RecentThreadsResponse{}, fmt.Errorf("recent threads: %w", err)
	}
	defer rows.Close()

	threads := make([]models.RecentThread, 0)
	for rows.Next() {
		var t models.RecentThread
		var accessLevel string
		var accessLevels []string
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Slug, &t.ReplyCount, &t.LastActivityAt, &t.CategoryName, &t.CategorySlug, &accessLevel, &accessLevels,
		); err != nil {
			return models.RecentThreadsResponse{}, fmt.Errorf("scan recent thread: %w", err)
		}
		levels := access.NormalizeLevels(accessLevels)
		if len(levels) == 0 {
			levels = []string{access.NormalizeLevel(accessLevel)}
		}
		if !viewer.CanViewAny(levels) {
			continue
		}
		t.URL = models.ThreadURL(t.Slug, t.ID)
		threads = append(threads, t)
		if len(threads) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return models.RecentThreadsResponse{}, err
	}
	if viewer.ActorID != nil && len(threads) > 0 {
		ids := make([]string, len(threads))
		for i, t := range threads {
			ids[i] = t.ID
		}
		if unread, err := r.threadUnreadMap(ctx, *viewer.ActorID, ids); err == nil {
			for i := range threads {
				threads[i].IsUnread = unread[threads[i].ID]
			}
		}
	}
	return models.RecentThreadsResponse{Site: r.site, Threads: threads}, nil
}

func (r *Reader) threadSummary(ctx context.Context, threadID string) *string {
	if !r.intelligence.Enabled {
		return nil
	}
	summary, _ := r.threadIntelligence(ctx, threadID)
	return summary
}

func (r *Reader) threadIntelligence(ctx context.Context, threadID string) (*string, string) {
	if !r.intelligence.Enabled {
		return nil, "disabled"
	}
	var summary, modelVersion string
	err := r.pool.QueryRow(ctx, `
		SELECT summary, model_version FROM thread_intelligence WHERE thread_id = $1
	`, threadID).Scan(&summary, &modelVersion)
	if errors.Is(err, pgx.ErrNoRows) || summary == "" {
		return nil, "pending"
	}
	if err != nil {
		return nil, "pending"
	}
	normalized := intelligence.NormalizePlainText(summary)
	return &normalized, modelVersion
}

type categoryRow struct {
	models.CategorySummary
}

func (r *Reader) listCategories(ctx context.Context, viewer access.Viewer) ([]models.CategorySummary, error) {
	_, flat, err := r.listCategoryTree(ctx, viewer)
	return flat, err
}

func (r *Reader) listCategoryTree(ctx context.Context, viewer access.Viewer) ([]models.CategoryGroup, []models.CategorySummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, description, parent_id, sort_order, thread_count, post_count, access_level, access_levels, list_public
		FROM categories
		ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]categoryRow)
	var roots []categoryRow
	for rows.Next() {
		var c categoryRow
		var accessLevel string
		var accessLevels []string
		var listPublic bool
		if err := rows.Scan(
			&c.ID, &c.Slug, &c.Name, &c.Description, &c.ParentID, &c.SortOrder,
			&c.ThreadCount, &c.PostCount, &accessLevel, &accessLevels, &listPublic,
		); err != nil {
			return nil, nil, fmt.Errorf("scan category: %w", err)
		}
		c.CategorySummary = applyCategoryAccess(c.CategorySummary, accessLevel, accessLevels, listPublic, viewer)
		byID[c.ID] = c
		if c.ParentID == nil {
			roots = append(roots, c)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	children := make(map[string][]categoryRow)
	for _, c := range byID {
		if c.ParentID != nil {
			children[*c.ParentID] = append(children[*c.ParentID], c)
		}
	}
	for id := range children {
		sortCategoryRows(children[id])
	}

	latest, err := r.latestThreadByCategory(ctx)
	if err != nil {
		return nil, nil, err
	}

	var groups []models.CategoryGroup
	var flat []models.CategorySummary
	for _, root := range roots {
		kids := children[root.ID]
		group := models.CategoryGroup{
			ID:          root.ID,
			Slug:        root.Slug,
			Name:        root.Name,
			Description: root.Description,
			SortOrder:   root.SortOrder,
		}
		if len(kids) == 0 {
			forum := root.CategorySummary
			forum.IsGroup = false
			if includeForumInTree(forum) {
				if forum.CanView {
					forum.Latest = latest[forum.ID]
				}
				group.Forums = append(group.Forums, forum)
				if forum.CanView {
					flat = append(flat, forum)
				}
			}
		} else {
			for _, kid := range kids {
				forum := kid.CategorySummary
				forum.IsGroup = false
				if !includeForumInTree(forum) {
					continue
				}
				if forum.CanView {
					forum.Latest = latest[forum.ID]
				}
				group.Forums = append(group.Forums, forum)
				if forum.CanView {
					flat = append(flat, forum)
				}
			}
		}
		if len(group.Forums) > 0 {
			groups = append(groups, group)
		}
	}
	return groups, flat, nil
}

func includeForumInTree(forum models.CategorySummary) bool {
	if forum.CanView {
		return true
	}
	return forum.ListPublic
}

func (r *Reader) ThreadCategoryAccessLevel(ctx context.Context, threadID string) (string, error) {
	var level string
	err := r.pool.QueryRow(ctx, `
		SELECT c.access_level
		FROM threads t
		JOIN categories c ON c.id = t.category_id
		WHERE t.id = $1 AND t.deleted_at IS NULL
	`, threadID).Scan(&level)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return level, err
}

func (r *Reader) CategoryAccessLevel(ctx context.Context, categoryID string) (string, error) {
	var level string
	err := r.pool.QueryRow(ctx, `SELECT access_level FROM categories WHERE id = $1`, categoryID).Scan(&level)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return level, err
}

func applyCategoryAccess(cat models.CategorySummary, level string, levels []string, listPublic bool, viewer access.Viewer) models.CategorySummary {
	cat.AccessLevels = access.NormalizeLevels(levels)
	if len(cat.AccessLevels) == 0 {
		cat.AccessLevels = []string{access.NormalizeLevel(level)}
	}
	cat.AccessLevel = access.PrimaryLevel(cat.AccessLevels)
	cat.ListPublic = listPublic
	cat.CanView = viewer.CanViewAny(cat.AccessLevels)
	cat.CanPost = viewer.CanPostAny(cat.AccessLevels)
	if cat.CanView {
		cat.URL = models.CategoryURL(cat.Slug)
	} else {
		cat.LockedLabel = access.LockedLabels(cat.AccessLevels)
		cat.URL = ""
		cat.Latest = nil
	}
	return cat
}

func (r *Reader) latestThreadByCategory(ctx context.Context) (map[string]*models.CategoryLatestThread, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (t.category_id)
		       t.category_id, t.title, t.slug, t.id, t.last_activity_at, a.display_name
		FROM threads t
		JOIN actors a ON a.id = t.author_id
		WHERE t.deleted_at IS NULL
		ORDER BY t.category_id, t.last_activity_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("latest threads: %w", err)
	}
	defer rows.Close()

	out := make(map[string]*models.CategoryLatestThread)
	for rows.Next() {
		var categoryID, title, slug, threadID, author string
		var lastActivity time.Time
		if err := rows.Scan(&categoryID, &title, &slug, &threadID, &lastActivity, &author); err != nil {
			return nil, fmt.Errorf("scan latest thread: %w", err)
		}
		out[categoryID] = &models.CategoryLatestThread{
			Title:          title,
			URL:            models.ThreadURL(slug, threadID),
			AuthorName:     author,
			LastActivityAt: lastActivity,
		}
	}
	return out, rows.Err()
}

func sortCategoryRows(rows []categoryRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SortOrder != rows[j].SortOrder {
			return rows[i].SortOrder < rows[j].SortOrder
		}
		return rows[i].Name < rows[j].Name
	})
}

func (r *Reader) listThreadsInCategory(ctx context.Context, categoryID string, page, perPage int, viewer access.Viewer) ([]models.ThreadSummary, error) {
	offset := (page - 1) * perPage
	opFilter := `op.moderation_status = 'approved'`
	if viewer.IsStaff {
		opFilter = `(op.moderation_status = 'approved' OR op.moderation_status = 'pending')`
	} else if viewer.ActorID != nil {
		opFilter = `(op.moderation_status = 'approved' OR (op.moderation_status = 'pending' AND op.author_id = $4))`
	}
	querySQL := `
		SELECT t.id, t.title, t.slug, t.reply_count, t.view_count, t.last_activity_at,
		       t.is_pinned, t.is_locked, a.display_name
		FROM threads t
		JOIN actors a ON a.id = t.author_id
		JOIN posts op ON op.thread_id = t.id AND op.is_op AND op.deleted_at IS NULL
		WHERE t.category_id = $1 AND t.deleted_at IS NULL AND ` + opFilter + `
		ORDER BY t.is_pinned DESC, t.last_activity_at DESC
		LIMIT $2 OFFSET $3
	`
	var rows pgx.Rows
	var err error
	if viewer.IsStaff || viewer.ActorID == nil {
		rows, err = r.pool.Query(ctx, querySQL, categoryID, perPage, offset)
	} else {
		rows, err = r.pool.Query(ctx, querySQL, categoryID, perPage, offset, *viewer.ActorID)
	}
	if err != nil {
		return nil, fmt.Errorf("list threads: %w", err)
	}
	defer rows.Close()

	out := make([]models.ThreadSummary, 0)
	for rows.Next() {
		var t models.ThreadSummary
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Slug, &t.ReplyCount, &t.ViewCount, &t.LastActivityAt,
			&t.IsPinned, &t.IsLocked, &t.AuthorName,
		); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		t.URL = models.ThreadURL(t.Slug, t.ID)
		t.HasSummary = r.threadSummary(ctx, t.ID) != nil
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if viewer.ActorID != nil && len(out) > 0 {
		ids := make([]string, len(out))
		for i, t := range out {
			ids[i] = t.ID
		}
		if unread, err := r.threadUnreadMap(ctx, *viewer.ActorID, ids); err == nil {
			for i := range out {
				out[i].IsUnread = unread[out[i].ID]
			}
		}
	}
	return out, nil
}

func (r *Reader) listPosts(ctx context.Context, threadID string, page, perPage int, viewer access.Viewer) ([]models.Post, error) {
	offset := (page - 1) * perPage
	viewerActorID := viewer.ActorID
	modFilter := `p.moderation_status = 'approved'`
	args := []any{
		threadID, perPage, offset, viewerActorID,
		r.postPolicy.EditEnabled, r.postPolicy.EditWindowMinutes,
		viewer.IsAdmin, r.postPolicy.DeleteEnabled,
	}
	if viewer.IsStaff {
		modFilter = `p.moderation_status IN ('approved', 'pending')`
	} else if viewerActorID != nil {
		modFilter = `(p.moderation_status = 'approved' OR (p.moderation_status = 'pending' AND p.author_id = $9))`
		args = append(args, *viewerActorID)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.body_html, p.created_at, p.edited_at, p.is_op, p.reaction_count,
		       a.display_name, a.type, a.karma, COALESCE(a.avatar_url, ''),
		       p.quoted_post_id, qa.display_name, p.quote_markdown, qp.body_html,
		       CASE WHEN $4::text IS NULL THEN FALSE ELSE EXISTS(
		         SELECT 1 FROM reactions rx
		         WHERE rx.post_id = p.id AND rx.reactor_id = $4 AND rx.reaction_type = 'like'
		       ) END,
		       CASE WHEN $5::boolean AND $4::text IS NOT NULL AND p.author_id = $4::text AND NOT t.is_locked
		            AND p.created_at + ($6::int * interval '1 minute') >= now()
		            THEN p.body_markdown ELSE NULL END,
		       CASE WHEN $5::boolean AND $4::text IS NOT NULL AND p.author_id = $4::text AND NOT t.is_locked
		            AND $6::int > 0
		            AND p.created_at + ($6::int * interval '1 minute') >= now()
		            THEN TRUE ELSE FALSE END,
		       CASE WHEN $7::boolean AND $8::boolean AND NOT p.is_op AND NOT t.is_locked THEN TRUE ELSE FALSE END,
		       EXISTS(
		         SELECT 1 FROM actor_warnings w
		         WHERE w.actor_id = p.author_id AND w.expires_at > now()
		       ),
		       p.moderation_status
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		JOIN actors a ON a.id = p.author_id
		LEFT JOIN posts qp ON qp.id = p.quoted_post_id
		LEFT JOIN actors qa ON qa.id = qp.author_id
		WHERE p.thread_id = $1 AND p.deleted_at IS NULL AND `+modFilter+`
		ORDER BY p.position ASC
		LIMIT $2 OFFSET $3
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	out := make([]models.Post, 0)
	for rows.Next() {
		var p models.Post
		var actorType string
		var quoteID, quoteAuthor, quoteMarkdown, quoteSourceHTML *string
		var bodyMarkdown *string
		if err := rows.Scan(
			&p.ID, &p.BodyHTML, &p.CreatedAt, &p.EditedAt, &p.IsOP, &p.ReactionCount,
			&p.Author.Name, &actorType, &p.Author.Karma, &p.Author.AvatarURL,
			&quoteID, &quoteAuthor, &quoteMarkdown, &quoteSourceHTML,
			&p.ReactedByMe,
			&bodyMarkdown, &p.CanEdit, &p.CanDelete,
			&p.Author.ActiveWarning,
			&p.ModerationStatus,
		); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		p.Author.IsAgent = actorType == "agent"
		p.Author.URL = models.UserURL(p.Author.Name)
		p.BodyHTML = r.postHTML(p.BodyHTML)
		if bodyMarkdown != nil {
			p.BodyMarkdown = bodyMarkdown
		}
		if quoteID != nil && quoteAuthor != nil {
			excerpt := ""
			if quoteMarkdown != nil && strings.TrimSpace(*quoteMarkdown) != "" {
				excerpt = strings.TrimSpace(*quoteMarkdown)
			} else if quoteSourceHTML != nil {
				excerpt = excerptHTML(*quoteSourceHTML, 200)
			}
			if excerpt != "" {
				p.Quote = &models.QuotedPost{
					ID:         *quoteID,
					AuthorName: *quoteAuthor,
					Excerpt:    excerpt,
					URL:        "#post-" + *quoteID,
				}
			}
		}
		p.PendingModeration = p.ModerationStatus == "pending"
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Reader) PostByID(ctx context.Context, postID string, viewerActorID *string, viewerIsAdmin bool) (models.Post, error) {
	var p models.Post
	var actorType string
	var quoteID, quoteAuthor, quoteMarkdown, quoteSourceHTML *string
	var bodyMarkdown *string

	err := r.pool.QueryRow(ctx, `
		SELECT p.id, p.body_html, p.created_at, p.edited_at, p.is_op, p.reaction_count,
		       a.display_name, a.type, a.karma, COALESCE(a.avatar_url, ''),
		       p.quoted_post_id, qa.display_name, p.quote_markdown, qp.body_html,
		       CASE WHEN $2::text IS NULL THEN FALSE ELSE EXISTS(
		         SELECT 1 FROM reactions rx
		         WHERE rx.post_id = p.id AND rx.reactor_id = $2 AND rx.reaction_type = 'like'
		       ) END,
		       CASE WHEN $3::boolean AND $2::text IS NOT NULL AND p.author_id = $2::text AND NOT t.is_locked
		            AND p.created_at + ($4::int * interval '1 minute') >= now()
		            THEN p.body_markdown ELSE NULL END,
		       CASE WHEN $3::boolean AND $2::text IS NOT NULL AND p.author_id = $2::text AND NOT t.is_locked
		            AND $4::int > 0
		            AND p.created_at + ($4::int * interval '1 minute') >= now()
		            THEN TRUE ELSE FALSE END,
		       CASE WHEN $5::boolean AND $6::boolean AND NOT p.is_op AND NOT t.is_locked THEN TRUE ELSE FALSE END,
		       EXISTS(
		         SELECT 1 FROM actor_warnings w
		         WHERE w.actor_id = p.author_id AND w.expires_at > now()
		       )
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		JOIN actors a ON a.id = p.author_id
		LEFT JOIN posts qp ON qp.id = p.quoted_post_id
		LEFT JOIN actors qa ON qa.id = qp.author_id
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`, postID, viewerActorID, r.postPolicy.EditEnabled, r.postPolicy.EditWindowMinutes, viewerIsAdmin, r.postPolicy.DeleteEnabled).Scan(
		&p.ID, &p.BodyHTML, &p.CreatedAt, &p.EditedAt, &p.IsOP, &p.ReactionCount,
		&p.Author.Name, &actorType, &p.Author.Karma, &p.Author.AvatarURL,
		&quoteID, &quoteAuthor, &quoteMarkdown, &quoteSourceHTML,
		&p.ReactedByMe,
		&bodyMarkdown, &p.CanEdit, &p.CanDelete,
		&p.Author.ActiveWarning,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Post{}, ErrNotFound
	}
	if err != nil {
		return models.Post{}, fmt.Errorf("post by id: %w", err)
	}

	p.Author.IsAgent = actorType == "agent"
	p.Author.URL = models.UserURL(p.Author.Name)
	p.BodyHTML = r.postHTML(p.BodyHTML)
	if bodyMarkdown != nil {
		p.BodyMarkdown = bodyMarkdown
	}
	if quoteID != nil && quoteAuthor != nil {
		excerpt := ""
		if quoteMarkdown != nil && strings.TrimSpace(*quoteMarkdown) != "" {
			excerpt = strings.TrimSpace(*quoteMarkdown)
		} else if quoteSourceHTML != nil {
			excerpt = excerptHTML(*quoteSourceHTML, 200)
		}
		if excerpt != "" {
			p.Quote = &models.QuotedPost{
				ID:         *quoteID,
				AuthorName: *quoteAuthor,
				Excerpt:    excerpt,
				URL:        "#post-" + *quoteID,
			}
		}
	}
	return p, nil
}

func excerptHTML(html string, max int) string {
	return intelligence.TruncatePlain(intelligence.StripHTML(html), max)
}

func paginate(page, perPage, total int, pageURL func(int) string) models.Pagination {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	p := models.Pagination{Current: page, Total: totalPages}
	if page > 1 {
		u := pageURL(page - 1)
		p.PrevURL = &u
	}
	if page < totalPages {
		u := pageURL(page + 1)
		p.NextURL = &u
	}
	return p
}