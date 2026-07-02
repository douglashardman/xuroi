package query

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type UserExport struct {
	ExportedAt  time.Time       `json:"exported_at"`
	Profile     json.RawMessage `json:"profile"`
	Posts       json.RawMessage `json:"posts"`
	Threads     json.RawMessage `json:"threads"`
	Reactions   json.RawMessage `json:"reactions"`
	Notifications json.RawMessage `json:"notifications"`
}

func (r *Reader) ExportUserData(ctx context.Context, actorID string) (UserExport, error) {
	var profile struct {
		ID          string     `json:"id"`
		DisplayName string     `json:"display_name"`
		Email       string     `json:"email,omitempty"`
		Bio         string     `json:"bio,omitempty"`
		Karma       int        `json:"karma"`
		JoinedAt    time.Time  `json:"joined_at"`
		LastActive  *time.Time `json:"last_active_at,omitempty"`
	}
	err := r.pool.QueryRow(ctx, `
		SELECT a.id, a.display_name, COALESCE(e.email, ''), COALESCE(a.bio, ''),
		       COALESCE(a.karma, 0), a.created_at, a.last_active_at
		FROM actors a
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		WHERE a.id = $1 AND a.type = 'human' AND a.deleted_at IS NULL
	`, actorID).Scan(
		&profile.ID, &profile.DisplayName, &profile.Email, &profile.Bio,
		&profile.Karma, &profile.JoinedAt, &profile.LastActive,
	)
	if err != nil {
		return UserExport{}, fmt.Errorf("export profile: %w", err)
	}

	postRows, err := r.pool.Query(ctx, `
		SELECT p.id, p.thread_id, t.title, p.body_markdown, p.created_at, p.edited_at
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		WHERE p.author_id = $1 AND p.deleted_at IS NULL
		ORDER BY p.created_at ASC
	`, actorID)
	if err != nil {
		return UserExport{}, err
	}
	defer postRows.Close()

	type exportPost struct {
		ID        string     `json:"id"`
		ThreadID  string     `json:"thread_id"`
		Thread    string     `json:"thread_title"`
		Body      string     `json:"body_markdown"`
		CreatedAt time.Time  `json:"created_at"`
		EditedAt  *time.Time `json:"edited_at,omitempty"`
	}
	var posts []exportPost
	for postRows.Next() {
		var p exportPost
		if err := postRows.Scan(&p.ID, &p.ThreadID, &p.Thread, &p.Body, &p.CreatedAt, &p.EditedAt); err != nil {
			return UserExport{}, err
		}
		posts = append(posts, p)
	}
	if err := postRows.Err(); err != nil {
		return UserExport{}, err
	}

	threadRows, err := r.pool.Query(ctx, `
		SELECT id, title, slug, created_at, reply_count
		FROM threads
		WHERE author_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, actorID)
	if err != nil {
		return UserExport{}, err
	}
	defer threadRows.Close()

	type exportThread struct {
		ID         string    `json:"id"`
		Title      string    `json:"title"`
		Slug       string    `json:"slug"`
		CreatedAt  time.Time `json:"created_at"`
		ReplyCount int       `json:"reply_count"`
	}
	var threads []exportThread
	for threadRows.Next() {
		var t exportThread
		if err := threadRows.Scan(&t.ID, &t.Title, &t.Slug, &t.CreatedAt, &t.ReplyCount); err != nil {
			return UserExport{}, err
		}
		threads = append(threads, t)
	}
	if err := threadRows.Err(); err != nil {
		return UserExport{}, err
	}

	reactionRows, err := r.pool.Query(ctx, `
		SELECT post_id, reaction_type, created_at
		FROM reactions
		WHERE reactor_id = $1
		ORDER BY created_at ASC
	`, actorID)
	if err != nil {
		return UserExport{}, err
	}
	defer reactionRows.Close()

	type exportReaction struct {
		PostID   string    `json:"post_id"`
		Type     string    `json:"reaction_type"`
		CreatedAt time.Time `json:"created_at"`
	}
	var reactions []exportReaction
	for reactionRows.Next() {
		var rx exportReaction
		if err := reactionRows.Scan(&rx.PostID, &rx.Type, &rx.CreatedAt); err != nil {
			return UserExport{}, err
		}
		reactions = append(reactions, rx)
	}
	if err := reactionRows.Err(); err != nil {
		return UserExport{}, err
	}

	notifRows, err := r.pool.Query(ctx, `
		SELECT id, type, title, body, url, created_at, read_at
		FROM notifications
		WHERE actor_id = $1
		ORDER BY created_at DESC
		LIMIT 500
	`, actorID)
	if err != nil {
		return UserExport{}, err
	}
	defer notifRows.Close()

	type exportNotif struct {
		ID        string     `json:"id"`
		Type      string     `json:"type"`
		Title     string     `json:"title"`
		Body      string     `json:"body"`
		URL       string     `json:"url"`
		CreatedAt time.Time  `json:"created_at"`
		ReadAt    *time.Time `json:"read_at,omitempty"`
	}
	var notifications []exportNotif
	for notifRows.Next() {
		var n exportNotif
		if err := notifRows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.URL, &n.CreatedAt, &n.ReadAt); err != nil {
			return UserExport{}, err
		}
		notifications = append(notifications, n)
	}
	if err := notifRows.Err(); err != nil {
		return UserExport{}, err
	}

	profileJSON, _ := json.Marshal(profile)
	postsJSON, _ := json.Marshal(posts)
	threadsJSON, _ := json.Marshal(threads)
	reactionsJSON, _ := json.Marshal(reactions)
	notifsJSON, _ := json.Marshal(notifications)

	return UserExport{
		ExportedAt:    time.Now().UTC(),
		Profile:       profileJSON,
		Posts:         postsJSON,
		Threads:       threadsJSON,
		Reactions:     reactionsJSON,
		Notifications: notifsJSON,
	}, nil
}