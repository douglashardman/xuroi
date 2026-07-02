package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuroi/xuroi/api/internal/email"
	"github.com/xuroi/xuroi/api/internal/ids"
	"github.com/xuroi/xuroi/api/internal/intelligence"
	"github.com/xuroi/xuroi/api/internal/models"
)

// NotifyMentions records in-app notifications and queues mention emails for new recipients.
func (s *Service) NotifyMentions(ctx context.Context, postID, threadID, authorID string, mentionedActorIDs []string) error {
	if len(mentionedActorIDs) == 0 {
		return nil
	}

	var authorName, threadTitle, threadSlug string
	err := s.pool.QueryRow(ctx, `
		SELECT a.display_name, t.title, t.slug
		FROM posts p
		JOIN actors a ON a.id = p.author_id
		JOIN threads t ON t.id = p.thread_id
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`, postID).Scan(&authorName, &threadTitle, &threadSlug)
	if err != nil {
		return fmt.Errorf("load post context: %w", err)
	}

	postURL := models.ThreadURL(threadSlug, threadID) + "#post-" + postID
	title := authorName + " mentioned you"
	body := "in " + threadTitle

	for _, actorID := range mentionedActorIDs {
		if actorID == authorID {
			continue
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO notifications (id, actor_id, type, from_actor_id, post_id, thread_id, title, body, url)
			VALUES ($1, $2, 'mention', $3, $4, $5, $6, $7, $8)
		`, ids.New("ntf_"), actorID, authorID, postID, threadID, title, body, postURL)
		if err != nil {
			return fmt.Errorf("insert notification: %w", err)
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO post_mentions (post_id, actor_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, postID, actorID)
		if err != nil {
			return fmt.Errorf("insert post_mention: %w", err)
		}
		if s.site.Email.Enabled {
			_, err = s.pool.Exec(ctx, `
				INSERT INTO email_mention_queue (id, actor_id, post_id, scheduled_at)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (actor_id, post_id) DO UPDATE SET scheduled_at = EXCLUDED.scheduled_at
			`, ids.New("emq_"), actorID, postID, time.Now())
			if err != nil {
				return fmt.Errorf("enqueue mention email: %w", err)
			}
		}
	}
	return nil
}

// SyncPostMentions updates post_mentions after an edit and notifies only newly added mentions.
func (s *Service) SyncPostMentions(ctx context.Context, postID, threadID, authorID string, mentionedActorIDs []string) error {
	rows, err := s.pool.Query(ctx, `SELECT actor_id FROM post_mentions WHERE post_id = $1`, postID)
	if err != nil {
		return fmt.Errorf("list post mentions: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		existing[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	want := make(map[string]struct{}, len(mentionedActorIDs))
	for _, id := range mentionedActorIDs {
		if id != authorID {
			want[id] = struct{}{}
		}
	}

	for id := range existing {
		if _, ok := want[id]; !ok {
			_, err = s.pool.Exec(ctx, `DELETE FROM post_mentions WHERE post_id = $1 AND actor_id = $2`, postID, id)
			if err != nil {
				return err
			}
		}
	}

	var newOnes []string
	for id := range want {
		if _, ok := existing[id]; !ok {
			newOnes = append(newOnes, id)
		}
	}
	if len(newOnes) == 0 {
		for id := range want {
			_, err = s.pool.Exec(ctx, `
				INSERT INTO post_mentions (post_id, actor_id) VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, postID, id)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return s.NotifyMentions(ctx, postID, threadID, authorID, newOnes)
}

// ProcessMentionQueue sends due @mention emails.
func (s *Service) ProcessMentionQueue(ctx context.Context, limit int) (int, error) {
	if !s.site.Email.Enabled || !s.cfg.Enabled {
		return 0, nil
	}
	if limit < 1 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
		SELECT q.id, q.actor_id, q.post_id, q.attempts
		FROM email_mention_queue q
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
		id, actorID, postID string
		attempts            int
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.actorID, &it.postID, &it.attempts); err != nil {
			return 0, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	sent := 0
	for _, it := range items {
		ok, err := s.sendMentionEmail(ctx, it.id, it.actorID, it.postID, it.attempts)
		if err != nil {
			return sent, err
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

func (s *Service) sendMentionEmail(ctx context.Context, queueID, actorID, postID string, attempts int) (bool, error) {
	var emailAddr, displayName string
	err := s.pool.QueryRow(ctx, `
		SELECT e.email, a.display_name
		FROM actor_emails e
		JOIN actors a ON a.id = e.actor_id
		LEFT JOIN email_preferences ep ON ep.actor_id = e.actor_id
		WHERE e.actor_id = $1
		  AND COALESCE(ep.mentions_enabled, TRUE) = TRUE
	`, actorID).Scan(&emailAddr, &displayName)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM email_mention_queue WHERE id = $1`, queueID)
		return false, nil
	}

	var authorName, threadTitle, threadSlug, threadID, bodyHTML string
	err = s.pool.QueryRow(ctx, `
		SELECT a.display_name, t.title, t.slug, t.id, p.body_html
		FROM posts p
		JOIN actors a ON a.id = p.author_id
		JOIN threads t ON t.id = p.thread_id
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`, postID).Scan(&authorName, &threadTitle, &threadSlug, &threadID, &bodyHTML)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM email_mention_queue WHERE id = $1`, queueID)
		return false, nil
	}

	excerpt := intelligence.TruncatePlain(intelligence.StripHTML(bodyHTML), 280)
	if excerpt == "" {
		return false, nil
	}

	siteURL := strings.TrimRight(s.site.Site.URL, "/")
	threadURL := siteURL + models.ThreadURL(threadSlug, threadID)
	postURL := threadURL + "#post-" + postID
	communityName := s.site.Email.FromName
	if communityName == "" {
		communityName = s.site.Site.Name + " Community"
	}

	subject, htmlBody, textBody, err := email.RenderMention(email.MentionData{
		CommunityName: communityName,
		SiteURL:       siteURL,
		LogoURL:       email.LogoURL(siteURL),
		Recipient:     displayName,
		AuthorName:    authorName,
		ThreadTitle:   threadTitle,
		PostURL:       postURL,
		Excerpt:       excerpt,
		CommunityURL:  siteURL + "/community",
		SettingsURL:   siteURL + "/settings/email",
		Copyright:     "© 2006–2026 PutterTalk LLC.",
	})
	if err != nil {
		return false, err
	}

	if err := s.mailer.Send(ctx, email.Message{
		To:          emailAddr,
		Subject:     subject,
		HTMLBody:    htmlBody,
		TextBody:    textBody,
		ReplyTo:     s.site.Email.ReplyTo,
		MessageType: "mention",
	}); err != nil {
		_ = s.scheduleMentionRetry(ctx, queueID, attempts+1, err)
		return false, nil
	}
	_, _ = s.pool.Exec(ctx, `DELETE FROM email_mention_queue WHERE id = $1`, queueID)
	return true, nil
}