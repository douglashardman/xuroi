package notify

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/xuroi/xuroi/api/internal/email"
	"github.com/xuroi/xuroi/api/internal/ids"
	"github.com/xuroi/xuroi/api/internal/models"
)

// EnqueueCategoryNewThread notifies forum watchers about a new thread.
func (s *Service) EnqueueCategoryNewThread(ctx context.Context, categoryID, threadID, authorID string) error {
	if !s.site.Email.Enabled {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT w.actor_id
		FROM category_email_watches w
		LEFT JOIN email_preferences ep ON ep.actor_id = w.actor_id
		WHERE w.category_id = $1
		  AND w.actor_id <> $2
		  AND COALESCE(ep.thread_replies_enabled, TRUE) = TRUE
	`, categoryID, authorID)
	if err != nil {
		return fmt.Errorf("category watchers: %w", err)
	}
	defer rows.Close()

	scheduled := time.Now().Add(30 * time.Second)
	for rows.Next() {
		var actorID string
		if err := rows.Scan(&actorID); err != nil {
			return err
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO email_category_queue (id, actor_id, category_id, thread_id, scheduled_at)
			VALUES ($1, $2, $3, $4, $5)
		`, ids.New("ecq_"), actorID, categoryID, threadID, scheduled)
		if err != nil {
			return fmt.Errorf("enqueue category watch: %w", err)
		}
	}
	return rows.Err()
}

func (s *Service) ProcessCategoryQueue(ctx context.Context, limit int) (int, error) {
	if !s.site.Email.Enabled || !s.cfg.Enabled {
		return 0, nil
	}
	if limit < 1 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT q.id, q.actor_id, q.category_id, q.thread_id, q.attempts
		FROM email_category_queue q
		WHERE q.scheduled_at <= now()
		  AND (q.next_retry_at IS NULL OR q.next_retry_at <= now())
		ORDER BY q.scheduled_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type item struct {
		id, actorID, categoryID, threadID string
		attempts                          int
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.actorID, &it.categoryID, &it.threadID, &it.attempts); err != nil {
			return 0, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	sent := 0
	for _, it := range items {
		ok, err := s.sendCategoryWatchEmail(ctx, it.id, it.actorID, it.categoryID, it.threadID, it.attempts)
		if err != nil {
			return sent, err
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

func (s *Service) sendCategoryWatchEmail(ctx context.Context, queueID, actorID, categoryID, threadID string, attempts int) (bool, error) {
	var emailAddr, displayName string
	err := s.pool.QueryRow(ctx, `
		SELECT e.email, a.display_name
		FROM actor_emails e
		JOIN actors a ON a.id = e.actor_id
		WHERE e.actor_id = $1
	`, actorID).Scan(&emailAddr, &displayName)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM email_category_queue WHERE id = $1`, queueID)
		return false, nil
	}

	var threadTitle, titlePrefix, threadSlug, catName string
	err = s.pool.QueryRow(ctx, `
		SELECT t.title, COALESCE(t.title_prefix, ''), t.slug, c.name
		FROM threads t
		JOIN categories c ON c.id = t.category_id
		WHERE t.id = $1 AND t.deleted_at IS NULL
	`, threadID).Scan(&threadTitle, &titlePrefix, &threadSlug, &catName)
	if titlePrefix != "" {
		threadTitle = models.ThreadDisplayTitle(titlePrefix, threadTitle)
	}
	if err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM email_category_queue WHERE id = $1`, queueID)
		return false, nil
	}

	siteURL := strings.TrimRight(s.site.Site.URL, "/")
	threadURL := siteURL + models.ThreadURL(threadSlug, threadID)
	communityName := s.site.Email.FromName
	if communityName == "" {
		communityName = s.site.Site.Name + " Community"
	}
	subject := fmt.Sprintf("New thread in %s", catName)
	textBody := fmt.Sprintf("Hi %s,\n\nA new thread was posted in %s:\n%s\n\n%s\n", displayName, catName, threadTitle, threadURL)
	htmlBody := fmt.Sprintf(`<p>Hi %s,</p><p>A new thread was posted in <strong>%s</strong>:</p><p><a href="%s">%s</a></p>`,
		html.EscapeString(displayName), html.EscapeString(catName), threadURL, html.EscapeString(threadTitle))

	if err := s.mailer.Send(ctx, email.Message{
		To:          emailAddr,
		Subject:     subject,
		HTMLBody:    htmlBody,
		TextBody:    textBody,
		ReplyTo:     s.site.Email.ReplyTo,
		MessageType: "category_watch",
	}); err != nil {
		_ = s.scheduleCategoryRetry(ctx, queueID, attempts+1, err)
		return false, nil
	}
	_, _ = s.pool.Exec(ctx, `DELETE FROM email_category_queue WHERE id = $1`, queueID)
	return true, nil
}