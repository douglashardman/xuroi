package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuroi/xuroi/api/internal/intelligence"
)

func (r *Reader) ThreadLLMText(ctx context.Context, id string) (string, error) {
	meta, err := r.ThreadMeta(ctx, id)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", meta.Title)
	fmt.Fprintf(&b, "Forum: %s\n", meta.Category)
	fmt.Fprintf(&b, "URL: %s\n", meta.URL)
	fmt.Fprintf(&b, "Replies: %d\n", meta.ReplyCount)
	if len(meta.Participants) > 0 {
		fmt.Fprintf(&b, "Participants: %s\n", strings.Join(meta.Participants, ", "))
	}
	if meta.Summary != nil && strings.TrimSpace(*meta.Summary) != "" {
		fmt.Fprintf(&b, "\n## %s\n\n%s\n", meta.SummaryLabel, strings.TrimSpace(*meta.Summary))
	}

	rows, err := r.pool.Query(ctx, `
		SELECT a.display_name, p.body_markdown, p.created_at, p.is_op
		FROM posts p
		JOIN actors a ON a.id = p.author_id
		WHERE p.thread_id = $1 AND p.deleted_at IS NULL AND p.moderation_status = 'approved'
		ORDER BY p.position ASC
		LIMIT 50
	`, id)
	if err != nil {
		return "", fmt.Errorf("llm posts: %w", err)
	}
	defer rows.Close()

	fmt.Fprintf(&b, "\n## Discussion\n\n")
	for rows.Next() {
		var author string
		var body string
		var createdAt time.Time
		var isOP bool
		if err := rows.Scan(&author, &body, &createdAt, &isOP); err != nil {
			return "", err
		}
		label := "Reply"
		if isOP {
			label = "Original post"
		}
		plain := intelligence.TruncatePlain(body, 1200)
		fmt.Fprintf(&b, "### %s — %s\n\n%s\n\n", author, label, plain)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return b.String(), nil
}