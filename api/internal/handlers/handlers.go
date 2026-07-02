package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/access"
	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/dm"
	"github.com/xuroi/xuroi/api/internal/friends"
	"github.com/xuroi/xuroi/api/internal/markdown"
	"github.com/xuroi/xuroi/api/internal/media"
	"github.com/xuroi/xuroi/api/internal/netutil"
	"github.com/xuroi/xuroi/api/internal/notify"
	"github.com/xuroi/xuroi/api/internal/query"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
	"github.com/xuroi/xuroi/api/internal/service"
	"github.com/xuroi/xuroi/api/internal/site"
)

type API struct {
	pool    *pgxpool.Pool
	forum   *service.Forum
	reader  *query.Reader
	auth    *auth.Service
	media   *media.Store
	limiter *ratelimit.Limiter
	notify  *notify.Service
	dm      *dm.Service
	friends *friends.Service
	siteCfg site.Config
}

func New(pool *pgxpool.Pool, forum *service.Forum, reader *query.Reader, authSvc *auth.Service, mediaStore *media.Store, limiter *ratelimit.Limiter, notifySvc *notify.Service, siteCfg site.Config) *API {
	friendsSvc := friends.New(pool)
	return &API{pool: pool, forum: forum, reader: reader, auth: authSvc, media: mediaStore, limiter: limiter, notify: notifySvc, dm: dm.New(pool, friendsSvc), friends: friendsSvc, siteCfg: siteCfg}
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /v1/health", a.health)
	mux.HandleFunc("GET /v1/users/{slug}", a.getUserProfile)
	mux.HandleFunc("GET /v1/categories", a.listCategories)
	mux.HandleFunc("GET /v1/categories/{slug}", a.getCategory)
	mux.HandleFunc("GET /v1/threads/recent", a.listRecentThreads)
	mux.HandleFunc("GET /v1/search", a.searchContent)
	mux.HandleFunc("GET /v1/moderation/report-reasons", a.listReportReasons)
	mux.HandleFunc("GET /v1/threads/{id}", a.getThread)
	mux.HandleFunc("DELETE /v1/threads/{id}", a.deleteThread)
	mux.HandleFunc("POST /v1/threads/{id}/read", a.markThreadRead)
	mux.HandleFunc("POST /v1/threads/{id}/report", a.reportThread)
	mux.HandleFunc("GET /v1/notifications", a.listNotifications)
	mux.HandleFunc("GET /v1/notifications/unread-count", a.notificationUnreadCount)
	mux.HandleFunc("POST /v1/notifications/read-all", a.markAllNotificationsRead)
	mux.HandleFunc("POST /v1/notifications/{id}/read", a.markNotificationRead)
	mux.HandleFunc("GET /v1/threads/{id}/meta.json", a.getThreadMeta)
	mux.HandleFunc("POST /v1/categories", a.createCategory)
	mux.HandleFunc("POST /v1/threads", a.createThread)
	mux.HandleFunc("POST /v1/threads/{id}/posts", a.createPost)
	mux.HandleFunc("PATCH /v1/posts/{id}", a.editPost)
	mux.HandleFunc("DELETE /v1/posts/{id}", a.deletePost)
	mux.HandleFunc("POST /v1/posts/{id}/reactions", a.togglePostReaction)
	mux.HandleFunc("POST /v1/posts/{id}/report", a.reportPost)
	mux.HandleFunc("POST /v1/posts/{id}/warn", a.warnPost)
	mux.HandleFunc("POST /v1/posts/{id}/remove", a.staffRemovePost)
	mux.HandleFunc("GET /v1/admin/posts/{id}", a.getPostAdminAudit)
	mux.HandleFunc("GET /v1/admin/reports", a.listReports)
	mux.HandleFunc("POST /v1/admin/reports/{id}/dismiss", a.dismissReport)
	mux.HandleFunc("GET /v1/mod/queue", a.listModQueue)
	mux.HandleFunc("GET /v1/mod/log", a.listModLog)
	mux.HandleFunc("POST /v1/mod/posts/{id}/approve", a.approvePost)
	mux.HandleFunc("POST /v1/mod/posts/{id}/reject", a.rejectPost)
	mux.HandleFunc("GET /v1/posts/{id}/revisions", a.getPostRevisions)
	mux.HandleFunc("PATCH /v1/threads/{id}", a.moderateThread)
	mux.HandleFunc("POST /v1/admin/rebuild-projections", a.rebuildProjections)
	mux.HandleFunc("GET /v1/auth/check-display-name", a.checkDisplayName)
	mux.HandleFunc("POST /v1/auth/register", a.register)
	mux.HandleFunc("POST /v1/auth/login", a.login)
	mux.HandleFunc("POST /v1/auth/password", a.setPassword)
	mux.HandleFunc("POST /v1/auth/password-reset/request", a.passwordResetRequest)
	mux.HandleFunc("POST /v1/auth/password-reset/confirm", a.passwordResetConfirm)
	mux.HandleFunc("POST /v1/auth/magic-link/request", a.magicLinkRequest)
	mux.HandleFunc("GET /v1/auth/magic-link/consume", a.magicLinkConsume)
	mux.HandleFunc("POST /v1/auth/magic-link/consume", a.magicLinkConsume)
	mux.HandleFunc("POST /v1/auth/logout", a.logout)
	mux.HandleFunc("GET /v1/auth/me", a.me)
	mux.HandleFunc("POST /v1/me/avatar", a.uploadAvatar)
	mux.HandleFunc("DELETE /v1/me/avatar", a.deleteAvatar)
	mux.HandleFunc("GET /v1/me/dm-privacy", a.getDMPrivacy)
	mux.HandleFunc("PATCH /v1/me/dm-privacy", a.setDMPrivacy)
	mux.HandleFunc("GET /v1/me/email-preferences", a.getEmailPreferences)
	mux.HandleFunc("PATCH /v1/me/email-preferences", a.setEmailPreferences)
	mux.HandleFunc("GET /v1/dm/members/search", a.searchDMMembers)
	mux.HandleFunc("GET /v1/friends/requests", a.listFriendRequests)
	mux.HandleFunc("POST /v1/friends/requests", a.sendFriendRequest)
	mux.HandleFunc("POST /v1/friends/requests/{id}/accept", a.acceptFriendRequest)
	mux.HandleFunc("POST /v1/friends/requests/{id}/decline", a.declineFriendRequest)
	mux.HandleFunc("GET /v1/dm/conversations", a.listDMConversations)
	mux.HandleFunc("POST /v1/dm/conversations", a.startDMConversation)
	mux.HandleFunc("GET /v1/dm/conversations/{id}", a.getDMConversation)
	mux.HandleFunc("POST /v1/dm/conversations/{id}/messages", a.sendDMMessage)
	mux.HandleFunc("POST /v1/dm/conversations/{id}/read", a.markDMRead)
	mux.HandleFunc("GET /v1/auth/ban-status", a.banStatus)
	mux.HandleFunc("POST /v1/auth/passkey/signup/begin", a.passkeySignupBegin)
	mux.HandleFunc("POST /v1/auth/passkey/signup/finish", a.passkeySignupFinish)
	mux.HandleFunc("POST /v1/auth/passkey/register/begin", a.passkeyRegisterBegin)
	mux.HandleFunc("POST /v1/auth/passkey/register/finish", a.passkeyRegisterFinish)
	mux.HandleFunc("POST /v1/auth/passkey/login/begin", a.passkeyLoginBegin)
	mux.HandleFunc("POST /v1/auth/passkey/login/finish", a.passkeyLoginFinish)
	mux.HandleFunc("GET /v1/email/unsubscribe", a.emailUnsubscribe)
	mux.HandleFunc("POST /v1/email/unsubscribe", a.emailUnsubscribe)
	mux.HandleFunc("GET /v1/auth/email-verify/consume", a.emailVerifyConsume)
	mux.HandleFunc("POST /v1/auth/email-verify/consume", a.emailVerifyConsume)
	mux.HandleFunc("POST /v1/auth/email-verify/resend", a.emailVerifyResend)
	mux.HandleFunc("GET /v1/seo/sitemap", a.getSitemap)
	mux.HandleFunc("GET /v1/admin/overview", a.getAdminOverview)
	mux.HandleFunc("GET /v1/admin/categories", a.listAdminCategories)
	mux.HandleFunc("PATCH /v1/admin/categories/{id}", a.updateAdminCategory)
	mux.HandleFunc("DELETE /v1/admin/categories/{id}", a.deleteAdminCategory)
	mux.HandleFunc("PUT /v1/admin/categories/reorder", a.reorderAdminCategories)
	mux.HandleFunc("GET /v1/admin/access-levels", a.listAccessLevels)
	mux.HandleFunc("GET /v1/admin/site-settings", a.getAdminSiteSettings)
	mux.HandleFunc("PATCH /v1/admin/site-settings", a.patchAdminSiteSettings)
	mux.HandleFunc("PUT /v1/admin/users/{id}/entitlements", a.setUserEntitlements)
	mux.HandleFunc("GET /v1/admin/users", a.listAdminUsers)
	mux.HandleFunc("GET /v1/admin/users/{id}", a.getAdminUser)
	mux.HandleFunc("POST /v1/admin/users/{id}/ban", a.banUser)
	mux.HandleFunc("POST /v1/admin/users/{id}/unban", a.unbanUser)
	mux.HandleFunc("POST /v1/admin/users/{id}/warn", a.warnUser)
	mux.HandleFunc("GET /v1/admin/permissions", a.permissionCatalog)
	mux.HandleFunc("PUT /v1/admin/users/{id}/permissions", a.setUserPermissions)
	mux.HandleFunc("POST /v1/media/upload", a.uploadMedia)
	mux.HandleFunc("GET /v1/media/{name}", a.serveMedia)
	return withCORS(a.withIPBanCheck(mux))
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := a.pool.Ping(ctx)
	status := "ok"
	code := http.StatusOK
	if err != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, map[string]any{
		"status":  status,
		"service": "xuroi-api",
		"db":      err == nil,
	})
}

func (a *API) createCategory(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}

	var req struct {
		Slug        string  `json:"slug"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		SortOrder   int     `json:"sort_order"`
		ParentID    *string `json:"parent_id"`
		AccessLevel  string   `json:"access_level"`
		AccessLevels []string `json:"access_levels"`
		ListPublic   *bool    `json:"list_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "slug and name required")
		return
	}

	evt, err := a.forum.CreateCategory(r.Context(), service.CreateCategoryInput{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		SortOrder:    req.SortOrder,
		ParentID:     req.ParentID,
		AccessLevel:  req.AccessLevel,
		AccessLevels: req.AccessLevels,
		ListPublic:   req.ListPublic,
		ActorID:      admin.ID,
	})
	if err != nil {
		writeCategoryError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, evt)
}

func (a *API) createThread(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CategoryID   string `json:"category_id"`
		Title        string `json:"title"`
		AuthorID     string `json:"author_id"`
		BodyMarkdown string `json:"body_markdown"`
		BodyHTML     string `json:"body_html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.CategoryID == "" || req.Title == "" || req.BodyMarkdown == "" {
		writeError(w, http.StatusBadRequest, "category_id, title, and body_markdown required")
		return
	}
	var viewer access.Viewer
	if req.AuthorID == "" {
		actor, ok := a.requireWritableActor(w, r)
		if !ok {
			return
		}
		req.AuthorID = actor.ID
		v, err := a.viewerFromActor(r, actor)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		viewer = v
	}
	accessLevel, err := a.reader.CategoryAccessLevel(r.Context(), req.CategoryID)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "category not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.AuthorID != "" && viewer.ActorID == nil {
		viewer, err = a.viewerFromRequest(r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if !viewer.CanPost(accessLevel) {
		writeError(w, http.StatusForbidden, "you do not have access to post in this forum")
		return
	}
	if a.rateLimited(w, "thread:actor:"+req.AuthorID, ratelimit.ThreadActorLimit, ratelimit.ThreadActorWindow) {
		return
	}
	isStaff, isAdmin := false, false
	if actor, aerr := a.actorFromRequest(r); aerr == nil {
		isStaff = actor.IsModerator
		isAdmin = actor.IsAdmin
	}
	check, err := a.checkPostContent(r.Context(), req.AuthorID, req.BodyMarkdown, isStaff, isAdmin)
	if err != nil {
		writeContentPolicyError(w, err)
		return
	}
	var mentioned []string
	req.BodyMarkdown, mentioned = a.processPostMentions(r, req.BodyMarkdown, req.AuthorID)
	if req.BodyHTML == "" {
		req.BodyHTML = markdown.ToHTML(req.BodyMarkdown)
	}

	evt, err := a.forum.CreateThread(r.Context(), service.CreateThreadInput{
		CategoryID:   req.CategoryID,
		Title:        req.Title,
		AuthorID:     req.AuthorID,
		BodyMarkdown: req.BodyMarkdown,
		BodyHTML:     req.BodyHTML,
		AuthorIP:     netutil.ClientIP(r),
		ForcePending: check.ForcePending,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var payload struct {
		ThreadID string `json:"thread_id"`
		PostID   string `json:"post_id"`
		Slug     string `json:"slug"`
	}
	_ = json.Unmarshal(evt.Payload, &payload)
	threadURL := ""
	if payload.ThreadID != "" && payload.Slug != "" {
		threadURL = "/t/" + payload.Slug + "--" + payload.ThreadID
	}

	if payload.PostID != "" && len(mentioned) > 0 && a.notify != nil {
		_ = a.notify.NotifyMentions(r.Context(), payload.PostID, payload.ThreadID, req.AuthorID, mentioned)
	}

	pending := false
	if payload.PostID != "" {
		if status, err := query.PostModerationStatus(r.Context(), a.pool, payload.PostID); err == nil && status == "pending" {
			pending = true
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"event":               evt,
		"thread_id":           payload.ThreadID,
		"thread_url":          threadURL,
		"pending_moderation":  pending,
	})
}

func (a *API) createPost(w http.ResponseWriter, r *http.Request) {
	threadID := r.PathValue("id")
	var req struct {
		AuthorID      string  `json:"author_id"`
		BodyMarkdown  string  `json:"body_markdown"`
		BodyHTML      string  `json:"body_html"`
		QuotedPostID  *string `json:"quoted_post_id"`
		QuoteMarkdown *string `json:"quote_markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.BodyMarkdown == "" {
		writeError(w, http.StatusBadRequest, "body_markdown required")
		return
	}
	var viewer access.Viewer
	if req.AuthorID == "" {
		actor, ok := a.requireWritableActor(w, r)
		if !ok {
			return
		}
		req.AuthorID = actor.ID
		v, err := a.viewerFromActor(r, actor)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		viewer = v
	}
	accessLevel, err := a.reader.ThreadCategoryAccessLevel(r.Context(), threadID)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			writeError(w, http.StatusNotFound, "thread not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if viewer.ActorID == nil {
		viewer, err = a.viewerFromRequest(r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if !viewer.CanPost(accessLevel) {
		writeError(w, http.StatusForbidden, "you do not have access to post in this forum")
		return
	}
	if a.rateLimited(w, "post:actor:"+req.AuthorID, ratelimit.PostActorLimit, ratelimit.PostActorWindow) {
		return
	}
	isStaff, isAdmin := false, false
	if actor, aerr := a.actorFromRequest(r); aerr == nil {
		isStaff = actor.IsModerator
		isAdmin = actor.IsAdmin
	}
	check, err := a.checkPostContent(r.Context(), req.AuthorID, req.BodyMarkdown, isStaff, isAdmin)
	if err != nil {
		writeContentPolicyError(w, err)
		return
	}
	var mentioned []string
	req.BodyMarkdown, mentioned = a.processPostMentions(r, req.BodyMarkdown, req.AuthorID)
	if req.BodyHTML == "" {
		req.BodyHTML = markdown.ToHTML(req.BodyMarkdown)
	}

	evt, err := a.forum.CreatePost(r.Context(), service.CreatePostInput{
		ThreadID:      threadID,
		AuthorID:      req.AuthorID,
		BodyMarkdown:  req.BodyMarkdown,
		BodyHTML:      req.BodyHTML,
		QuotedPostID:  req.QuotedPostID,
		QuoteMarkdown: req.QuoteMarkdown,
		AuthorIP:      netutil.ClientIP(r),
		ForcePending:  check.ForcePending,
	})
	if errors.Is(err, service.ErrInvalidQuote) {
		writeError(w, http.StatusBadRequest, "invalid quoted post")
		return
	}
	if errors.Is(err, service.ErrQuoteNotExcerpt) {
		writeError(w, http.StatusBadRequest, "quote must be from the original post")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var payload struct {
		PostID string `json:"post_id"`
	}
	_ = json.Unmarshal(evt.Payload, &payload)

	viewerID := req.AuthorID
	viewerIsAdmin := false
	if actor, aerr := a.actorFromRequest(r); aerr == nil {
		viewerID = actor.ID
		viewerIsAdmin = actor.IsAdmin
	}

	if payload.PostID != "" && a.notify != nil {
		_ = a.notify.EnqueueThreadReply(r.Context(), threadID, payload.PostID, req.AuthorID)
		if len(mentioned) > 0 {
			_ = a.notify.NotifyMentions(r.Context(), payload.PostID, threadID, req.AuthorID, mentioned)
		}
		_ = a.notify.NotifyThreadReply(r.Context(), threadID, payload.PostID, req.AuthorID, mentioned)
	}

	post, perr := a.reader.PostByID(r.Context(), payload.PostID, &viewerID, viewerIsAdmin)
	pending := false
	if status, serr := query.PostModerationStatus(r.Context(), a.pool, payload.PostID); serr == nil && status == "pending" {
		pending = true
	}
	if perr != nil {
		writeJSON(w, http.StatusCreated, map[string]any{
			"event":              evt,
			"pending_moderation": pending,
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":                 evt.ID,
		"type":               evt.Type,
		"payload":            json.RawMessage(evt.Payload),
		"post":               post,
		"pending_moderation": pending,
	})
}

func (a *API) rebuildProjections(w http.ResponseWriter, r *http.Request) {
	if err := a.forum.RebuildProjections(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rebuilt"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (a *API) rateLimited(w http.ResponseWriter, key string, limit int, window time.Duration) bool {
	if a.limiter == nil || a.limiter.Allow(key, limit, window) {
		return false
	}
	a.writeRateLimited(w, key)
	return true
}

func (a *API) blocked(w http.ResponseWriter, key string, limit int, window time.Duration) bool {
	if a.limiter == nil || !a.limiter.Blocked(key, limit, window) {
		return false
	}
	a.writeRateLimited(w, key)
	return true
}

func (a *API) writeRateLimited(w http.ResponseWriter, key string) {
	if a.limiter != nil {
		w.Header().Set("Retry-After", strconv.Itoa(a.limiter.RetryAfter(key)))
	}
	writeError(w, http.StatusTooManyRequests, "too many requests; slow down")
}