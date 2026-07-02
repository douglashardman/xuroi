package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/email"
	"github.com/xuroi/xuroi/api/internal/ids"
	"github.com/xuroi/xuroi/api/internal/intelligence"
	"github.com/xuroi/xuroi/api/internal/models"
	"github.com/xuroi/xuroi/api/internal/site"
)

type Service struct {
	pool   *pgxpool.Pool
	mailer email.Mailer
	cfg    email.Config
	site   site.Config
	auth   *auth.Service
}

func New(pool *pgxpool.Pool, mailer email.Mailer, cfg email.Config, siteCfg site.Config, authSvc *auth.Service) *Service {
	return &Service{pool: pool, mailer: mailer, cfg: cfg, site: siteCfg, auth: authSvc}
}

// MarkThreadRead records that the actor has caught up on a thread (clears pending digest).
func (s *Service) MarkThreadRead(ctx context.Context, actorID, threadID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO thread_reads (actor_id, thread_id, last_read_at)
		VALUES ($1, $2, now())
		ON CONFLICT (actor_id, thread_id) DO UPDATE SET last_read_at = EXCLUDED.last_read_at
	`, actorID, threadID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		DELETE FROM email_notification_queue WHERE actor_id = $1 AND thread_id = $2
	`, actorID, threadID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE actor_id = $1 AND thread_id = $2 AND read_at IS NULL
	`, actorID, threadID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// EnqueueThreadReply schedules a debounced digest for thread participants.
func (s *Service) EnqueueThreadReply(ctx context.Context, threadID, postID, authorID string) error {
	if !s.site.Email.Enabled {
		return nil
	}
	delay := s.site.Email.DigestDelayMinutes
	if delay < 0 {
		delay = 0
	}
	if s.cfg.DigestDelayMinutes > 0 && delay == 0 {
		delay = s.cfg.DigestDelayMinutes
	}

	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT p.author_id
		FROM posts p
		LEFT JOIN email_preferences ep ON ep.actor_id = p.author_id
		WHERE p.thread_id = $1 AND p.deleted_at IS NULL
		  AND p.author_id <> $2
		  AND COALESCE(ep.thread_replies_enabled, TRUE) = TRUE
		  AND NOT EXISTS (
		    SELECT 1 FROM email_thread_mutes m
		    WHERE m.actor_id = p.author_id AND m.thread_id = $1
		  )
	`, threadID, authorID)
	if err != nil {
		return fmt.Errorf("list participants: %w", err)
	}
	defer rows.Close()

	scheduled := time.Now().Add(time.Duration(delay) * time.Minute)
	var recipients []string
	for rows.Next() {
		var actorID string
		if err := rows.Scan(&actorID); err != nil {
			return err
		}
		recipients = append(recipients, actorID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, actorID := range recipients {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO email_notification_queue (id, actor_id, thread_id, last_post_id, scheduled_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (actor_id, thread_id) DO UPDATE SET
				last_post_id = EXCLUDED.last_post_id,
				scheduled_at = EXCLUDED.scheduled_at
		`, ids.New("enq_"), actorID, threadID, postID, scheduled)
		if err != nil {
			return fmt.Errorf("enqueue: %w", err)
		}
	}
	return nil
}

// ProcessQueue sends due thread digest emails (one per thread per recipient).
func (s *Service) ProcessQueue(ctx context.Context, limit int) (int, error) {
	if !s.site.Email.Enabled || !s.cfg.Enabled {
		return 0, nil
	}
	if limit < 1 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
		SELECT q.id, q.actor_id, q.thread_id
		FROM email_notification_queue q
		WHERE q.scheduled_at <= now()
		ORDER BY q.scheduled_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type item struct {
		id, actorID, threadID string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.actorID, &it.threadID); err != nil {
			return 0, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	sent := 0
	for _, it := range items {
		ok, err := s.sendThreadDigest(ctx, it.id, it.actorID, it.threadID)
		if err != nil {
			return sent, err
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

func (s *Service) sendThreadDigest(ctx context.Context, queueID, actorID, threadID string) (bool, error) {
	var emailAddr, displayName string
	err := s.pool.QueryRow(ctx, `
		SELECT e.email, a.display_name
		FROM actor_emails e
		JOIN actors a ON a.id = e.actor_id
		WHERE e.actor_id = $1
	`, actorID).Scan(&emailAddr, &displayName)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM email_notification_queue WHERE id = $1`, queueID)
		return false, nil
	}

	var threadTitle, threadSlug string
	err = s.pool.QueryRow(ctx, `
		SELECT title, slug FROM threads WHERE id = $1 AND deleted_at IS NULL
	`, threadID).Scan(&threadTitle, &threadSlug)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM email_notification_queue WHERE id = $1`, queueID)
		return false, nil
	}

	since := time.Time{}
	var lastRead time.Time
	if err := s.pool.QueryRow(ctx, `
		SELECT last_read_at FROM thread_reads WHERE actor_id = $1 AND thread_id = $2
	`, actorID, threadID).Scan(&lastRead); err == nil {
		since = lastRead
	}

	postRows, err := s.pool.Query(ctx, `
		SELECT a.display_name, p.body_html, p.created_at
		FROM posts p
		JOIN actors a ON a.id = p.author_id
		WHERE p.thread_id = $1 AND p.deleted_at IS NULL
		  AND p.created_at > $2 AND p.author_id <> $3
		ORDER BY p.position ASC
		LIMIT 20
	`, threadID, since, actorID)
	if err != nil {
		return false, err
	}
	defer postRows.Close()

	var posts []email.ThreadReplyPost
	for postRows.Next() {
		var author, bodyHTML string
		var created time.Time
		if err := postRows.Scan(&author, &bodyHTML, &created); err != nil {
			return false, err
		}
		excerpt := intelligence.TruncatePlain(intelligence.StripHTML(bodyHTML), 280)
		if excerpt == "" {
			continue
		}
		posts = append(posts, email.ThreadReplyPost{
			Author:  author,
			Excerpt: excerpt,
			When:    created.Format("Jan 2, 3:04 PM"),
		})
	}
	if err := postRows.Err(); err != nil {
		return false, err
	}

	_, _ = s.pool.Exec(ctx, `DELETE FROM email_notification_queue WHERE id = $1`, queueID)
	if len(posts) == 0 {
		return false, nil
	}

	siteURL := strings.TrimRight(s.site.Site.URL, "/")
	threadURL := siteURL + models.ThreadURL(threadSlug, threadID)
	communityName := s.site.Email.FromName
	if communityName == "" {
		communityName = s.site.Site.Name + " Community"
	}
	latestAuthor := ""
	if len(posts) > 0 {
		latestAuthor = posts[len(posts)-1].Author
	}

	var muted bool
	_ = s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM email_thread_mutes WHERE actor_id = $1 AND thread_id = $2)
	`, actorID, threadID).Scan(&muted)
	if muted {
		return false, nil
	}

	unsubToken, err := s.auth.IssueThreadUnsubscribeToken(ctx, actorID, threadID, emailAddr)
	if err != nil {
		return false, fmt.Errorf("unsubscribe token: %w", err)
	}
	unsubscribeURL := siteURL + "/email/unsubscribe?token=" + unsubToken

	subject, htmlBody, textBody, err := email.RenderThreadReply(email.ThreadReplyData{
		CommunityName:    communityName,
		SiteURL:          siteURL,
		LogoURL:          email.LogoURL(siteURL),
		Recipient:        displayName,
		IntroLine:        email.BuildIntroLine(len(posts), latestAuthor, communityName),
		ThreadTitle:      threadTitle,
		ThreadURL:        threadURL,
		CommunityURL:     siteURL + "/community",
		WatchedURL:       siteURL + "/community",
		DisableThreadURL: unsubscribeURL,
		DisableAllURL:    siteURL + "/settings/email",
		UnsubscribeURL:   unsubscribeURL,
		Copyright:        "© 2006–2026 PutterTalk LLC.",
		ReplyCount:       len(posts),
		Posts:            posts,
	})
	if err != nil {
		return false, err
	}

	if err := s.mailer.Send(ctx, email.Message{
		To:                 emailAddr,
		Subject:            subject,
		HTMLBody:           htmlBody,
		TextBody:           textBody,
		ReplyTo:            s.site.Email.ReplyTo,
		MessageType:        "thread_reply",
		ListUnsubscribeURL: unsubscribeURL,
	}); err != nil {
		return false, err
	}
	return true, nil
}