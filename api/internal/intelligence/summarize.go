package intelligence

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxPostsPerSummary = 40
const maxCharsPerPost = 2000

type Service struct {
	pool        *pgxpool.Pool
	summarizer  Summarizer
	enabled     bool
}

func New(pool *pgxpool.Pool, summarizer Summarizer, enabled bool) *Service {
	return &Service{pool: pool, summarizer: summarizer, enabled: enabled}
}

type threadInput struct {
	ID           string
	Title        string
	ReplyCount   int
	PostCount    int
	OPBodyHTML   string
	Participants int
}

// SummarizeStale generates or refreshes summaries for threads that are missing
// intelligence or have new posts since the last run.
func (s *Service) SummarizeStale(ctx context.Context, limit int) (int, error) {
	if !s.enabled {
		return 0, nil
	}
	if limit < 1 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.title, t.reply_count,
		       (SELECT count(*) FROM posts p WHERE p.thread_id = t.id AND p.deleted_at IS NULL),
		       (SELECT p.body_html FROM posts p WHERE p.thread_id = t.id AND p.deleted_at IS NULL AND p.is_op
		        ORDER BY p.position LIMIT 1),
		       (SELECT count(DISTINCT p.author_id) FROM posts p WHERE p.thread_id = t.id AND p.deleted_at IS NULL)
		FROM threads t
		LEFT JOIN thread_intelligence ti ON ti.thread_id = t.id
		WHERE t.deleted_at IS NULL
		  AND (ti.thread_id IS NULL OR ti.post_count < (
		        SELECT count(*) FROM posts p WHERE p.thread_id = t.id AND p.deleted_at IS NULL
		      ))
		ORDER BY t.last_activity_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return 0, fmt.Errorf("list stale threads: %w", err)
	}
	defer rows.Close()

	var inputs []threadInput
	for rows.Next() {
		var in threadInput
		var opHTML *string
		if err := rows.Scan(&in.ID, &in.Title, &in.ReplyCount, &in.PostCount, &opHTML, &in.Participants); err != nil {
			return 0, fmt.Errorf("scan thread: %w", err)
		}
		if opHTML != nil {
			in.OPBodyHTML = *opHTML
		}
		inputs = append(inputs, in)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	n := 0
	for _, in := range inputs {
		summary, version := s.generateSummary(ctx, in)
		if err := s.upsert(ctx, in.ID, summary, version, in.PostCount); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (s *Service) generateSummary(ctx context.Context, in threadInput) (string, string) {
	if s.summarizer != nil {
		posts, err := s.loadThreadPosts(ctx, in.ID)
		if err == nil && len(posts) > 0 {
			text, err := s.summarizer.SummarizeThread(ctx, ThreadSummaryInput{
				Title: in.Title,
				Posts: posts,
			})
			if err == nil && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text), s.summarizer.ModelVersion()
			}
		}
	}
	return buildSummary(in), HeuristicModelVersion
}

func (s *Service) loadThreadPosts(ctx context.Context, threadID string) ([]ThreadPostInput, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.display_name, p.is_op, p.body_html
		FROM posts p
		JOIN actors a ON a.id = p.author_id
		WHERE p.thread_id = $1 AND p.deleted_at IS NULL
		ORDER BY p.position ASC
		LIMIT $2
	`, threadID, maxPostsPerSummary)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ThreadPostInput
	for rows.Next() {
		var author string
		var isOP bool
		var bodyHTML string
		if err := rows.Scan(&author, &isOP, &bodyHTML); err != nil {
			return nil, err
		}
		plain := TruncatePlain(StripHTML(bodyHTML), maxCharsPerPost)
		if plain == "" {
			continue
		}
		out = append(out, ThreadPostInput{
			Author:    author,
			IsOP:      isOP,
			BodyPlain: plain,
		})
	}
	return out, rows.Err()
}

func (s *Service) upsert(ctx context.Context, threadID, summary, modelVersion string, postCount int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO thread_intelligence (thread_id, summary, model_version, post_count)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (thread_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			model_version = EXCLUDED.model_version,
			post_count = EXCLUDED.post_count,
			updated_at = now()
	`, threadID, summary, modelVersion, postCount)
	if err != nil {
		return fmt.Errorf("upsert intelligence: %w", err)
	}
	return nil
}

func buildSummary(in threadInput) string {
	title := strings.TrimSpace(in.Title)
	op := TruncatePlain(StripHTML(in.OPBodyHTML), 220)

	var parts []string
	if in.ReplyCount == 0 {
		parts = append(parts, fmt.Sprintf("%s — new thread awaiting replies.", title))
	} else {
		replyWord := "replies"
		if in.ReplyCount == 1 {
			replyWord = "reply"
		}
		partWord := "participants"
		if in.Participants == 1 {
			partWord = "participant"
		}
		parts = append(parts, fmt.Sprintf(
			"%s — %d %s from %d %s.",
			title, in.ReplyCount, replyWord, in.Participants, partWord,
		))
	}
	if op != "" {
		parts = append(parts, op)
	}
	return strings.Join(parts, " ")
}