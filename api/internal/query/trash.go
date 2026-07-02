package query

import (
	"context"
	"fmt"
	"time"
)

type TrashPost struct {
	PostID       string    `json:"post_id"`
	ThreadID     string    `json:"thread_id"`
	ThreadTitle  string    `json:"thread_title"`
	ThreadURL    string    `json:"thread_url"`
	CategoryName string    `json:"category_name"`
	AuthorName   string    `json:"author_name"`
	Excerpt      string    `json:"excerpt"`
	IsOP         bool      `json:"is_op"`
	DeletedAt    time.Time `json:"deleted_at"`
}

type TrashThread struct {
	ThreadID     string    `json:"thread_id"`
	Title        string    `json:"title"`
	ThreadURL    string    `json:"thread_url"`
	CategoryName string    `json:"category_name"`
	AuthorName   string    `json:"author_name"`
	ReplyCount   int       `json:"reply_count"`
	DeletedAt    time.Time `json:"deleted_at"`
}

type ModTrashResponse struct {
	Posts   []TrashPost   `json:"posts"`
	Threads []TrashThread `json:"threads"`
}

func (r *Reader) ListModTrash(ctx context.Context, limit int) (ModTrashResponse, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	postRows, err := r.pool.Query(ctx, `
		SELECT p.id, p.thread_id, t.title, t.slug, c.name, a.display_name,
		       LEFT(REGEXP_REPLACE(p.body_html, '<[^>]+>', ' ', 'g'), 200),
		       p.is_op, p.deleted_at
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		JOIN categories c ON c.id = t.category_id
		JOIN actors a ON a.id = p.author_id
		WHERE p.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		ORDER BY p.deleted_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return ModTrashResponse{}, fmt.Errorf("trash posts: %w", err)
	}
	defer postRows.Close()

	var posts []TrashPost
	for postRows.Next() {
		var item TrashPost
		var slug string
		if err := postRows.Scan(
			&item.PostID, &item.ThreadID, &item.ThreadTitle, &slug, &item.CategoryName,
			&item.AuthorName, &item.Excerpt, &item.IsOP, &item.DeletedAt,
		); err != nil {
			return ModTrashResponse{}, err
		}
		item.ThreadURL = "/t/" + slug + "--" + item.ThreadID
		posts = append(posts, item)
	}
	if err := postRows.Err(); err != nil {
		return ModTrashResponse{}, err
	}

	threadRows, err := r.pool.Query(ctx, `
		SELECT t.id, t.title, t.slug, c.name, a.display_name, t.reply_count, t.deleted_at
		FROM threads t
		JOIN categories c ON c.id = t.category_id
		JOIN actors a ON a.id = t.author_id
		WHERE t.deleted_at IS NOT NULL
		ORDER BY t.deleted_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return ModTrashResponse{}, fmt.Errorf("trash threads: %w", err)
	}
	defer threadRows.Close()

	var threads []TrashThread
	for threadRows.Next() {
		var item TrashThread
		var slug string
		if err := threadRows.Scan(
			&item.ThreadID, &item.Title, &slug, &item.CategoryName,
			&item.AuthorName, &item.ReplyCount, &item.DeletedAt,
		); err != nil {
			return ModTrashResponse{}, err
		}
		item.ThreadURL = "/t/" + slug + "--" + item.ThreadID
		threads = append(threads, item)
	}
	if err := threadRows.Err(); err != nil {
		return ModTrashResponse{}, err
	}

	if posts == nil {
		posts = []TrashPost{}
	}
	if threads == nil {
		threads = []TrashThread{}
	}
	return ModTrashResponse{Posts: posts, Threads: threads}, nil
}