package events

import (
	"encoding/json"
	"time"
)

const (
	TypeCategoryCreated = "category.created"
	TypeCategoryUpdated = "category.updated"
	TypeCategoryDeleted = "category.deleted"
	TypeThreadCreated   = "thread.created"
	TypePostCreated     = "post.created"
	TypePostEdited      = "post.edited"
	TypePostDeleted         = "post.deleted"
	TypePostRestored        = "post.restored"
	TypeThreadRestored      = "thread.restored"
	TypePostReactionAdded   = "post.reaction_added"
	TypePostReactionRemoved = "post.reaction_removed"
	TypeThreadLocked        = "thread.locked"
	TypeThreadUnlocked      = "thread.unlocked"
	TypeThreadPinned        = "thread.pinned"
	TypeThreadUnpinned      = "thread.unpinned"
	TypeThreadDeleted       = "thread.deleted"
	TypeThreadMoved         = "thread.moved"
	TypePostReported        = "post.reported"
	TypeThreadReported      = "thread.reported"
	TypePostModerated           = "post.moderated"
	TypeThreadAcceptedAnswerSet = "thread.accepted_answer_set"
	TypeThreadAcceptedAnswerClr = "thread.accepted_answer_cleared"
	TypeAdminSettingsUpdated    = "admin.settings_updated"
	TypeAdminUserBanned         = "admin.user_banned"
	TypeAdminUserUnbanned       = "admin.user_unbanned"
	TypeAdminBackupTriggered    = "admin.backup_triggered"
	TypeAdminEmailBanned        = "admin.email_banned"
)

type Event struct {
	ID             string          `json:"id"`
	StreamID       string          `json:"stream_id"`
	Sequence       int64           `json:"sequence"`
	Type           string          `json:"type"`
	ActorID        *string         `json:"actor_id,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	SchemaVersion  int             `json:"schema_version"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type AppendInput struct {
	StreamID       string
	Type           string
	ActorID        *string
	Payload        any
	SchemaVersion  int
	IdempotencyKey *string
}

type CategoryCreated struct {
	CategoryID  string  `json:"category_id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SortOrder   int     `json:"sort_order"`
	ParentID    *string `json:"parent_id"`
	AccessLevel     string   `json:"access_level,omitempty"`
	AccessLevels    []string `json:"access_levels,omitempty"`
	ListPublic      *bool    `json:"list_public,omitempty"`
	PostModeration  *bool    `json:"post_moderation,omitempty"`
}

type CategoryUpdated struct {
	CategoryID     string   `json:"category_id"`
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	SortOrder      int      `json:"sort_order"`
	ParentID       *string  `json:"parent_id"`
	AccessLevel    string   `json:"access_level,omitempty"`
	AccessLevels   []string `json:"access_levels,omitempty"`
	ListPublic     *bool    `json:"list_public,omitempty"`
	PostModeration *bool    `json:"post_moderation,omitempty"`
}

type CategoryDeleted struct {
	CategoryID string `json:"category_id"`
}

type CategoryReorderItem struct {
	CategoryID string  `json:"category_id"`
	SortOrder  int     `json:"sort_order"`
	ParentID   *string `json:"parent_id"`
}

type CategoriesReordered struct {
	Items []CategoryReorderItem `json:"items"`
}

type ThreadCreated struct {
	ThreadID     string `json:"thread_id"`
	PostID       string `json:"post_id"`
	CategoryID   string `json:"category_id"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	AuthorID     string `json:"author_id"`
	BodyMarkdown string `json:"body_markdown"`
	BodyHTML     string `json:"body_html"`
	AuthorIP      string `json:"author_ip,omitempty"`
	ForcePending  bool   `json:"force_pending,omitempty"`
}

type PostCreated struct {
	PostID        string  `json:"post_id"`
	ThreadID      string  `json:"thread_id"`
	AuthorID      string  `json:"author_id"`
	BodyMarkdown  string  `json:"body_markdown"`
	BodyHTML      string  `json:"body_html"`
	QuotedPostID  *string `json:"quoted_post_id"`
	QuoteMarkdown *string `json:"quote_markdown,omitempty"`
	AuthorIP      string  `json:"author_ip,omitempty"`
	ForcePending  bool    `json:"force_pending,omitempty"`
}

type ThreadMoved struct {
	ThreadID     string `json:"thread_id"`
	FromCategory string `json:"from_category_id"`
	ToCategory   string `json:"to_category_id"`
}

type PostReactionChanged struct {
	PostID       string `json:"post_id"`
	ThreadID     string `json:"thread_id"`
	ReactorID    string `json:"reactor_id"`
	ReactionType string `json:"reaction_type"`
}

type ThreadModeration struct {
	ThreadID   string `json:"thread_id"`
	LockReason string `json:"lock_reason,omitempty"`
}

type PostRestored struct {
	PostID   string `json:"post_id"`
	ThreadID string `json:"thread_id"`
}

type ThreadRestored struct {
	ThreadID string `json:"thread_id"`
}

type ThreadDeleted struct {
	ThreadID string `json:"thread_id"`
	Reason   string `json:"reason,omitempty"`
}

type PostDeleted struct {
	PostID   string `json:"post_id"`
	ThreadID string `json:"thread_id"`
	Reason   string `json:"reason,omitempty"`
	Hard     bool   `json:"hard"`
}

type PostReported struct {
	ReportID   string `json:"report_id"`
	PostID     string `json:"post_id"`
	ThreadID   string `json:"thread_id"`
	ReporterID string `json:"reporter_id"`
	Reason     string `json:"reason"`
}

type ThreadReported struct {
	ReportID   string `json:"report_id"`
	ThreadID   string `json:"thread_id"`
	ReporterID string `json:"reporter_id"`
	Reason     string `json:"reason"`
}

type PostModerated struct {
	PostID   string `json:"post_id"`
	ThreadID string `json:"thread_id"`
	Status   string `json:"status"`
}

type PostEdited struct {
	PostID       string  `json:"post_id"`
	ThreadID     string  `json:"thread_id"`
	BodyMarkdown string  `json:"body_markdown"`
	BodyHTML     string  `json:"body_html"`
	EditReason   *string `json:"edit_reason"`
}

type AcceptedAnswerChanged struct {
	ThreadID string `json:"thread_id"`
	PostID   string `json:"post_id,omitempty"`
}

type AdminUserAction struct {
	ActorID    string `json:"actor_id"`
	TargetName string `json:"target_name"`
	Duration   string `json:"duration,omitempty"`
}

type AdminEmailBan struct {
	Email    string `json:"email"`
	Duration string `json:"duration,omitempty"`
}

func StreamThread(threadID string) string {
	return "thread:" + threadID
}

func StreamSite() string {
	return "site"
}