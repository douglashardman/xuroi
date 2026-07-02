package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/access"
	"github.com/xuroi/xuroi/api/internal/models"
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
	Results []Result `json:"results"`
	Total   int      `json:"total"`
}

func Search(ctx context.Context, pool *pgxpool.Pool, opts Options, viewer access.Viewer) (Response, error) {
	q := strings.TrimSpace(opts.Query)
	author := strings.TrimSpace(opts.Author)
	categorySlug := strings.TrimSpace(opts.CategorySlug)
	if q == "" {
		return Response{Query: q, Results: []Result{}}, nil
	}
	limit := opts.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
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
		  AND ($2 = '' OR lower(sd.author_name) LIKE '%' || lower($2) || '%')
		  AND ($3 = '' OR c.slug = $3)
		ORDER BY rank DESC, sd.updated_at DESC
		LIMIT $4
	`, q, author, categorySlug, limit*3)
	if err != nil {
		return Response{}, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0, limit)
	for rows.Next() {
		var r Result
		var slug, body, accessLevel, catName, headline string
		if err := rows.Scan(
			&r.EntityID, &r.DocType, &r.ThreadID, &r.ThreadTitle, &slug,
			&r.AuthorName, &body, &accessLevel, &catName, &r.Rank, &headline,
		); err != nil {
			return Response{}, err
		}
		if !viewer.CanView(accessLevel) {
			continue
		}
		r.ThreadURL = models.ThreadURL(slug, r.ThreadID)
		r.CategoryName = catName
		if headline != "" {
			r.Excerpt = headline
		} else {
			r.Excerpt = body
			if len(r.Excerpt) > 200 {
				r.Excerpt = r.Excerpt[:200] + "…"
			}
		}
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
	return Response{Query: q, Results: results, Total: len(results)}, nil
}