package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/access"
	"github.com/xuroi/xuroi/api/internal/models"
	"github.com/xuroi/xuroi/api/internal/slug"
)

type Options struct {
	Query        string
	Author       string
	CategorySlug string
	Limit        int
}

type Result struct {
	EntityID     string  `json:"entity_id"`
	DocType      string  `json:"doc_type"`
	ThreadID     string  `json:"thread_id"`
	ThreadTitle  string  `json:"thread_title"`
	ThreadURL    string  `json:"thread_url"`
	CategoryName string  `json:"category_name"`
	AuthorName   string  `json:"author_name"`
	Excerpt      string  `json:"excerpt"`
	Rank         float64 `json:"rank"`
}

type Response struct {
	Query   string   `json:"query"`
	Author  string   `json:"author,omitempty"`
	Results []Result `json:"results"`
	Total   int      `json:"total"`
}

func authorMatches(authorName, filter string) bool {
	f := strings.ToLower(strings.TrimSpace(filter))
	if f == "" {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(authorName), f) {
		return true
	}
	return slug.FromDisplayName(authorName) == f
}

func Search(ctx context.Context, pool *pgxpool.Pool, opts Options, viewer access.Viewer) (Response, error) {
	q := strings.TrimSpace(opts.Query)
	author := strings.TrimSpace(opts.Author)
	categorySlug := strings.TrimSpace(opts.CategorySlug)
	if q == "" && author == "" {
		return Response{Query: q, Author: author, Results: []Result{}}, nil
	}
	limit := opts.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	if q == "" {
		displayName, err := resolveAuthorDisplayName(ctx, pool, author)
		if err != nil {
			return Response{}, err
		}
		if displayName == "" {
			return Response{Author: author, Results: []Result{}}, nil
		}
		return searchByAuthor(ctx, pool, displayName, categorySlug, limit, viewer)
	}

	rows, err := pool.Query(ctx, `
		SELECT sd.entity_id, sd.doc_type, sd.thread_id, sd.thread_title, sd.thread_slug,
		       sd.author_name, sd.body, sd.access_level, c.name,
		       ts_rank(sd.search_vector, query) AS rank,
		       ts_headline('english', sd.body, query, 'MaxFragments=2,MaxWords=30,MinWords=8,StartSel=<mark>,StopSel=</mark>') AS headline
		FROM search_documents sd
		JOIN categories c ON c.id = sd.category_id,
		     plainto_tsquery('english', $1) query
		WHERE sd.search_vector @@ query
		  AND ($2 = '' OR c.slug = $2)
		ORDER BY rank DESC, sd.updated_at DESC
		LIMIT $3
	`, q, categorySlug, limit*3)
	if err != nil {
		return Response{}, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0, limit)
	for rows.Next() {
		var r Result
		var slugName, body, accessLevel, catName, headline string
		if err := rows.Scan(
			&r.EntityID, &r.DocType, &r.ThreadID, &r.ThreadTitle, &slugName,
			&r.AuthorName, &body, &accessLevel, &catName, &r.Rank, &headline,
		); err != nil {
			return Response{}, err
		}
		if !authorMatches(r.AuthorName, author) {
			continue
		}
		if !viewer.CanView(accessLevel) {
			continue
		}
		r.ThreadURL = models.ThreadURL(slugName, r.ThreadID)
		r.CategoryName = catName
		r.Excerpt = excerptText(headline, body)
		results = append(results, r)
		if len(results) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return Response{}, err
	}
	if results == nil {
		results = []Result{}
	}
	return Response{Query: q, Author: author, Results: results, Total: len(results)}, nil
}

func resolveAuthorDisplayName(ctx context.Context, pool *pgxpool.Pool, filter string) (string, error) {
	f := strings.ToLower(strings.TrimSpace(filter))
	if f == "" {
		return "", nil
	}
	rows, err := pool.Query(ctx, `
		SELECT display_name FROM actors WHERE type = 'human'
	`)
	if err != nil {
		return "", fmt.Errorf("resolve author: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
		if strings.EqualFold(strings.TrimSpace(name), f) || slug.FromDisplayName(name) == f {
			return name, nil
		}
	}
	return "", rows.Err()
}

func searchByAuthor(ctx context.Context, pool *pgxpool.Pool, authorName, categorySlug string, limit int, viewer access.Viewer) (Response, error) {
	rows, err := pool.Query(ctx, `
		SELECT sd.entity_id, sd.doc_type, sd.thread_id, sd.thread_title, sd.thread_slug,
		       sd.author_name, sd.body, sd.access_level, c.name
		FROM search_documents sd
		JOIN categories c ON c.id = sd.category_id
		WHERE sd.author_name = $1
		  AND ($2 = '' OR c.slug = $2)
		ORDER BY sd.updated_at DESC
		LIMIT $3
	`, authorName, categorySlug, limit*3)
	if err != nil {
		return Response{}, fmt.Errorf("search author: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0, limit)
	for rows.Next() {
		var r Result
		var slugName, body, accessLevel, catName string
		if err := rows.Scan(
			&r.EntityID, &r.DocType, &r.ThreadID, &r.ThreadTitle, &slugName,
			&r.AuthorName, &body, &accessLevel, &catName,
		); err != nil {
			return Response{}, err
		}
		if !viewer.CanView(accessLevel) {
			continue
		}
		r.ThreadURL = models.ThreadURL(slugName, r.ThreadID)
		r.CategoryName = catName
		r.Excerpt = excerptText("", body)
		results = append(results, r)
		if len(results) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return Response{}, err
	}
	if results == nil {
		results = []Result{}
	}
	return Response{Author: authorName, Results: results, Total: len(results)}, nil
}

func excerptText(headline, body string) string {
	if headline != "" {
		return headline
	}
	excerpt := body
	if len(excerpt) > 200 {
		excerpt = excerpt[:200] + "…"
	}
	return excerpt
}