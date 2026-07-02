package models

import (
	"time"

	"github.com/xuroi/xuroi/api/internal/slug"
)

type Site struct {
	Name    string `json:"name"`
	Tagline string `json:"tagline,omitempty"`
	URL     string `json:"url"`
}

type Breadcrumb struct {
	Label string  `json:"label"`
	URL   *string `json:"url"`
}

type Pagination struct {
	Current  int     `json:"current"`
	Total    int     `json:"total"`
	PrevURL  *string `json:"prev_url"`
	NextURL  *string `json:"next_url"`
}

type CategoryLatestThread struct {
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	AuthorName     string    `json:"author_name"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

type CategorySummary struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	URL         string  `json:"url"`
	ParentID    *string `json:"parent_id,omitempty"`
	SortOrder   int     `json:"sort_order"`
	IsGroup     bool    `json:"is_group"`
	AccessLevel  string   `json:"access_level"`
	AccessLevels []string `json:"access_levels,omitempty"`
	ListPublic   bool     `json:"list_public"`
	CanView     bool    `json:"can_view"`
	CanPost     bool    `json:"can_post,omitempty"`
	LockedLabel string  `json:"locked_label,omitempty"`
	ThreadCount int     `json:"thread_count"`
	PostCount   int     `json:"post_count"`
	UnreadCount int     `json:"unread_count,omitempty"`
	Latest      *CategoryLatestThread `json:"latest,omitempty"`
}

type CategoryGroup struct {
	ID          string            `json:"id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	SortOrder   int               `json:"sort_order"`
	Forums      []CategorySummary `json:"forums"`
}

type ThreadSummary struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Slug           string    `json:"slug"`
	URL            string    `json:"url"`
	AuthorName     string    `json:"author_name"`
	ReplyCount     int       `json:"reply_count"`
	ViewCount      int       `json:"view_count,omitempty"`
	LastActivityAt time.Time `json:"last_activity_at"`
	IsPinned       bool      `json:"is_pinned"`
	IsLocked       bool      `json:"is_locked"`
	IsUnread       bool      `json:"is_unread,omitempty"`
	HasSummary     bool      `json:"has_summary"`
}

type Author struct {
	Name           string `json:"name"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	URL            string `json:"url"`
	IsAgent        bool   `json:"is_agent"`
	Karma          int    `json:"karma"`
	ActiveWarning  bool   `json:"active_warning,omitempty"`
}

type PostRevision struct {
	Revision  int       `json:"revision"`
	BodyHTML  string    `json:"body_html"`
	EditedAt  time.Time `json:"edited_at"`
	EditorName string   `json:"editor_name"`
}

type PostReport struct {
	ID           string    `json:"id"`
	PostID       string    `json:"post_id"`
	ThreadID     string    `json:"thread_id"`
	ThreadTitle  string    `json:"thread_title"`
	ThreadURL    string    `json:"thread_url"`
	PostAuthor   string    `json:"post_author"`
	PostExcerpt  string    `json:"post_excerpt"`
	ReporterName string    `json:"reporter_name"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
	PostURL      string    `json:"post_url"`
}

type PostAdminAudit struct {
	PostID         string     `json:"post_id"`
	ThreadID       string     `json:"thread_id"`
	ThreadTitle    string     `json:"thread_title"`
	AuthorID       string     `json:"author_id"`
	AuthorName     string     `json:"author_name"`
	AuthorIP       *string    `json:"author_ip"`
	CreatedAt      time.Time  `json:"created_at"`
	EditedAt       *time.Time `json:"edited_at"`
	RevisionCount  int        `json:"revision_count"`
	ReactionCount  int        `json:"reaction_count"`
	IsOP           bool       `json:"is_op"`
}

type UserProfile struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	URL         string    `json:"url"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Bio         string    `json:"bio,omitempty"`
	Karma       int       `json:"karma"`
	PostCount     int        `json:"post_count"`
	JoinedAt      time.Time  `json:"joined_at"`
	LastActiveAt  *time.Time `json:"last_active_at,omitempty"`
}

type QuotedPost struct {
	ID         string `json:"id"`
	AuthorName string `json:"author_name"`
	Excerpt    string `json:"excerpt"`
	URL        string `json:"url"`
}

type Post struct {
	ID            string      `json:"id"`
	Author        Author      `json:"author"`
	BodyHTML      string      `json:"body_html"`
	BodyMarkdown  *string     `json:"body_markdown,omitempty"`
	Quote         *QuotedPost `json:"quote,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	EditedAt      *time.Time  `json:"edited_at"`
	IsOP          bool        `json:"is_op"`
	ReactionCount int         `json:"reaction_count"`
	ReactedByMe   bool        `json:"reacted_by_me,omitempty"`
	CanEdit       bool        `json:"can_edit,omitempty"`
	CanDelete         bool        `json:"can_delete,omitempty"`
	IsWarned          bool        `json:"is_warned,omitempty"`
	ModerationStatus  string      `json:"moderation_status,omitempty"`
	PendingModeration bool        `json:"pending_moderation,omitempty"`
}

type CategoryRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	URL  string `json:"url"`
}

type ThreadDetail struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Slug           string    `json:"slug"`
	URL            string    `json:"url"`
	Summary        *string   `json:"summary,omitempty"`
	ReplyCount     int       `json:"reply_count"`
	ViewCount      int       `json:"view_count,omitempty"`
	IsLocked       bool      `json:"is_locked"`
	LockReason     string    `json:"lock_reason,omitempty"`
	IsPinned       bool      `json:"is_pinned"`
	EmailWatching  bool      `json:"email_watching,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

type HomeResponse struct {
	Site       Site            `json:"site"`
	Groups     []CategoryGroup `json:"groups"`
	Categories []CategorySummary `json:"categories"`
}

type RecentThread struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Slug           string    `json:"slug"`
	URL            string    `json:"url"`
	CategoryName   string    `json:"category_name"`
	CategorySlug   string    `json:"category_slug"`
	ReplyCount     int       `json:"reply_count"`
	LastActivityAt time.Time `json:"last_activity_at"`
	IsUnread       bool      `json:"is_unread,omitempty"`
}

type RecentThreadsResponse struct {
	Site    Site           `json:"site"`
	Threads []RecentThread `json:"threads"`
}

type CategoryPageResponse struct {
	Site        Site            `json:"site"`
	Category    CategorySummary `json:"category"`
	Threads     []ThreadSummary `json:"threads"`
	Pagination  Pagination      `json:"pagination"`
	Breadcrumbs []Breadcrumb    `json:"breadcrumbs"`
}

type ThreadPageResponse struct {
	Site        Site         `json:"site"`
	Thread      ThreadDetail `json:"thread"`
	Category    CategoryRef  `json:"category"`
	Posts       []Post       `json:"posts"`
	Pagination  Pagination   `json:"pagination"`
	Breadcrumbs []Breadcrumb `json:"breadcrumbs"`
	UI          struct {
		ShowModBar      bool   `json:"show_mod_bar"`
		OpenReportCount int    `json:"open_report_count,omitempty"`
		SummaryLabel    string `json:"summary_label,omitempty"`
	} `json:"ui"`
}

func ThreadURL(slug, id string) string {
	return "/t/" + slug + "--" + id
}

func CategoryURL(slug string) string {
	return "/c/" + slug
}

func UserURL(displayName string) string {
	s := slug.FromDisplayName(displayName)
	if s == "" {
		s = "member"
	}
	return "/u/" + s
}