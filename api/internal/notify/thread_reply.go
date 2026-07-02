package notify

import (
	"context"
	"fmt"

	"github.com/xuroi/xuroi/api/internal/ids"
	"github.com/xuroi/xuroi/api/internal/intelligence"
	"github.com/xuroi/xuroi/api/internal/models"
)

// NotifyThreadReply creates in-app notifications for thread participants when someone replies.
// skipActorIDs are excluded (e.g. author, users already @mentioned on this post).
func (s *Service) NotifyThreadReply(ctx context.Context, threadID, postID, authorID string, skipActorIDs []string) error {
	skip := make(map[string]struct{}, len(skipActorIDs)+1)
	skip[authorID] = struct{}{}
	for _, id := range skipActorIDs {
		skip[id] = struct{}{}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT p.author_id
		FROM posts p
		WHERE p.thread_id = $1 AND p.deleted_at IS NULL
		  AND p.author_id <> $2
		  AND NOT EXISTS (
		    SELECT 1 FROM email_thread_mutes m
		    WHERE m.actor_id = p.author_id AND m.thread_id = $1
		  )
	`, threadID, authorID)
	if err != nil {
		return fmt.Errorf("list participants: %w", err)
	}
	defer rows.Close()

	var recipients []string
	for rows.Next() {
		var actorID string
		if err := rows.Scan(&actorID); err != nil {
			return err
		}
		if _, ok := skip[actorID]; ok {
			continue
		}
		recipients = append(recipients, actorID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(recipients) == 0 {
		return nil
	}

	var authorName, threadTitle, threadSlug, bodyHTML string
	err = s.pool.QueryRow(ctx, `
		SELECT a.display_name, t.title, t.slug, p.body_html
		FROM posts p
		JOIN actors a ON a.id = p.author_id
		JOIN threads t ON t.id = p.thread_id
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`, postID).Scan(&authorName, &threadTitle, &threadSlug, &bodyHTML)
	if err != nil {
		return fmt.Errorf("load post context: %w", err)
	}

	postURL := models.ThreadURL(threadSlug, threadID) + "#post-" + postID
	title := authorName + " replied"
	body := "in " + threadTitle
	excerpt := intelligence.TruncatePlain(intelligence.StripHTML(bodyHTML), 120)
	if excerpt != "" {
		body = excerpt
	}

	for _, actorID := range recipients {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO notifications (id, actor_id, type, from_actor_id, post_id, thread_id, title, body, url)
			VALUES ($1, $2, 'thread_reply', $3, $4, $5, $6, $7, $8)
		`, ids.New("ntf_"), actorID, authorID, postID, threadID, title, body, postURL)
		if err != nil {
			return fmt.Errorf("insert thread_reply notification: %w", err)
		}
	}
	return nil
}