package dm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/ids"
	"github.com/xuroi/xuroi/api/internal/markdown"
	"github.com/xuroi/xuroi/api/internal/slug"
)

type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Participant struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	AvatarURL   string  `json:"avatar_url,omitempty"`
	URL         string  `json:"url"`
}

type ConversationSummary struct {
	ID            string      `json:"id"`
	Other         Participant `json:"other"`
	LastPreview   string      `json:"last_preview"`
	LastMessageAt time.Time   `json:"last_message_at"`
	UnreadCount   int         `json:"unread_count"`
}

type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	BodyHTML  string    `json:"body_html"`
	IsMine    bool      `json:"is_mine"`
	CreatedAt time.Time `json:"created_at"`
}

type ConversationPage struct {
	ID       string      `json:"id"`
	Other    Participant `json:"other"`
	Messages []Message   `json:"messages"`
}

func pairIDs(a, b string) (low, high string) {
	if a < b {
		return a, b
	}
	return b, a
}

func (s *Service) Privacy(ctx context.Context, actorID string) (string, error) {
	var p string
	err := s.pool.QueryRow(ctx, `SELECT dm_privacy FROM actors WHERE id = $1`, actorID).Scan(&p)
	if err != nil {
		return "", err
	}
	return NormalizePrivacy(p), nil
}

func (s *Service) SetPrivacy(ctx context.Context, actorID, privacy string) error {
	if !ValidPrivacy(privacy) {
		return fmt.Errorf("invalid dm_privacy")
	}
	_, err := s.pool.Exec(ctx, `UPDATE actors SET dm_privacy = $2 WHERE id = $1`, actorID, privacy)
	return err
}

func (s *Service) hasConversation(ctx context.Context, a, b string) (bool, error) {
	low, high := pairIDs(a, b)
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM dm_conversations WHERE participant_a = $1 AND participant_b = $2)
	`, low, high).Scan(&exists)
	return exists, err
}

func (s *Service) CanMessage(ctx context.Context, senderID, recipientID string, allowNew bool) error {
	if senderID == recipientID {
		return ErrSelfDM
	}
	var senderPrivacy, recipientPrivacy string
	err := s.pool.QueryRow(ctx, `
		SELECT s.dm_privacy, r.dm_privacy FROM actors s, actors r
		WHERE s.id = $1 AND r.id = $2
	`, senderID, recipientID).Scan(&senderPrivacy, &recipientPrivacy)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("member not found")
	}
	if err != nil {
		return err
	}
	if NormalizePrivacy(senderPrivacy) == PrivacyOff {
		return ErrSenderDMOff
	}
	existing, err := s.hasConversation(ctx, senderID, recipientID)
	if err != nil {
		return err
	}
	if existing {
		if NormalizePrivacy(recipientPrivacy) == PrivacyOff {
			return ErrDMDisabled
		}
		return nil
	}
	if !allowNew {
		return ErrNotParticipant
	}
	switch NormalizePrivacy(recipientPrivacy) {
	case PrivacyOff:
		return ErrDMDisabled
	case PrivacyFriendsOnly:
		return ErrDMFriendsOnly
	default:
		return nil
	}
}

func (s *Service) FindActorBySlug(ctx context.Context, nameSlug string) (string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, display_name FROM actors WHERE type = 'human'
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	want := strings.ToLower(nameSlug)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return "", err
		}
		if slug.FromDisplayName(name) == want {
			return id, nil
		}
	}
	return "", pgx.ErrNoRows
}

func (s *Service) GetOrCreateConversation(ctx context.Context, senderID, recipientID string) (string, error) {
	if err := s.CanMessage(ctx, senderID, recipientID, true); err != nil {
		return "", err
	}
	low, high := pairIDs(senderID, recipientID)
	var convID string
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM dm_conversations WHERE participant_a = $1 AND participant_b = $2
	`, low, high).Scan(&convID)
	if err == nil {
		return convID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	convID = ids.New("dmc_")
	_, err = s.pool.Exec(ctx, `
		INSERT INTO dm_conversations (id, participant_a, participant_b)
		VALUES ($1, $2, $3)
	`, convID, low, high)
	return convID, err
}

func (s *Service) participant(ctx context.Context, actorID string) (Participant, error) {
	var p Participant
	err := s.pool.QueryRow(ctx, `
		SELECT id, display_name, COALESCE(avatar_url, '')
		FROM actors WHERE id = $1
	`, actorID).Scan(&p.ID, &p.DisplayName, &p.AvatarURL)
	if err != nil {
		return p, err
	}
	p.URL = "/u/" + slug.FromDisplayName(p.DisplayName)
	return p, nil
}

func (s *Service) otherParticipant(ctx context.Context, convID, viewerID string) (Participant, error) {
	var a, b string
	err := s.pool.QueryRow(ctx, `
		SELECT participant_a, participant_b FROM dm_conversations WHERE id = $1
	`, convID).Scan(&a, &b)
	if err != nil {
		return Participant{}, err
	}
	other := b
	if viewerID == b {
		other = a
	} else if viewerID != a {
		return Participant{}, ErrNotParticipant
	}
	return s.participant(ctx, other)
}

func (s *Service) ListConversations(ctx context.Context, actorID string, limit int) ([]ConversationSummary, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.participant_a, c.participant_b, c.last_message_at,
		       m.body_html, m.sender_id,
		       (SELECT count(*)::int FROM dm_messages dm
		        LEFT JOIN dm_reads dr ON dr.conversation_id = c.id AND dr.actor_id = $1
		        WHERE dm.conversation_id = c.id AND dm.sender_id <> $1
		          AND dm.created_at > COALESCE(dr.last_read_at, 'epoch'::timestamptz))
		FROM dm_conversations c
		LEFT JOIN LATERAL (
			SELECT body_html, sender_id FROM dm_messages
			WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1
		) m ON TRUE
		WHERE c.participant_a = $1 OR c.participant_b = $1
		ORDER BY c.last_message_at DESC
		LIMIT $2
	`, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ConversationSummary, 0)
	for rows.Next() {
		var sum ConversationSummary
		var pa, pb string
		var bodyHTML *string
		var senderID *string
		if err := rows.Scan(&sum.ID, &pa, &pb, &sum.LastMessageAt, &bodyHTML, &senderID, &sum.UnreadCount); err != nil {
			return nil, err
		}
		otherID := pb
		if actorID == pb {
			otherID = pa
		}
		sum.Other, err = s.participant(ctx, otherID)
		if err != nil {
			return nil, err
		}
		if bodyHTML != nil {
			sum.LastPreview = truncatePlain(stripHTML(*bodyHTML), 120)
		}
		out = append(out, sum)
	}
	return out, rows.Err()
}

func (s *Service) ListMessages(ctx context.Context, actorID, convID string, limit int) (ConversationPage, error) {
	other, err := s.otherParticipant(ctx, convID, actorID)
	if err != nil {
		return ConversationPage{}, err
	}
	if limit < 1 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, sender_id, body_html, created_at
		FROM dm_messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT $2
	`, convID, limit)
	if err != nil {
		return ConversationPage{}, err
	}
	defer rows.Close()

	msgs := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.BodyHTML, &m.CreatedAt); err != nil {
			return ConversationPage{}, err
		}
		m.IsMine = m.SenderID == actorID
		m.BodyHTML = markdown.EnrichMediaImages(m.BodyHTML)
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return ConversationPage{}, err
	}
	_ = s.MarkRead(ctx, actorID, convID)
	return ConversationPage{ID: convID, Other: other, Messages: msgs}, nil
}

func (s *Service) SendMessage(ctx context.Context, senderID, convID, bodyMarkdown string) (Message, error) {
	other, err := s.otherParticipant(ctx, convID, senderID)
	if err != nil {
		return Message{}, err
	}
	if err := s.CanMessage(ctx, senderID, other.ID, false); err != nil {
		return Message{}, err
	}
	bodyMarkdown = strings.TrimSpace(bodyMarkdown)
	if bodyMarkdown == "" {
		return Message{}, fmt.Errorf("message body required")
	}
	bodyHTML := markdown.ToHTML(bodyMarkdown)
	msgID := ids.New("dmm_")
	now := time.Now()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO dm_messages (id, conversation_id, sender_id, body_markdown, body_html, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, msgID, convID, senderID, bodyMarkdown, bodyHTML, now)
	if err != nil {
		return Message{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE dm_conversations SET last_message_at = $2 WHERE id = $1
	`, convID, now)
	if err != nil {
		return Message{}, err
	}
	preview := truncatePlain(bodyMarkdown, 200)
	_, err = tx.Exec(ctx, `
		INSERT INTO notifications (id, actor_id, type, from_actor_id, title, body, url, created_at)
		VALUES ($1, $2, 'dm_message', $3, $4, $5, $6, $7)
	`, ids.New("ntf_"), other.ID, senderID,
		"New message from "+senderDisplayName(ctx, tx, senderID),
		preview, "/messages/"+convID, now)
	if err != nil {
		return Message{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, err
	}
	return Message{
		ID:        msgID,
		SenderID:  senderID,
		BodyHTML:  markdown.EnrichMediaImages(bodyHTML),
		IsMine:    true,
		CreatedAt: now,
	}, nil
}

func senderDisplayName(ctx context.Context, tx pgx.Tx, actorID string) string {
	var name string
	_ = tx.QueryRow(ctx, `SELECT display_name FROM actors WHERE id = $1`, actorID).Scan(&name)
	return name
}

func (s *Service) MarkRead(ctx context.Context, actorID, convID string) error {
	if _, err := s.otherParticipant(ctx, convID, actorID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO dm_reads (actor_id, conversation_id, last_read_at)
		VALUES ($1, $2, now())
		ON CONFLICT (actor_id, conversation_id) DO UPDATE SET last_read_at = now()
	`, actorID, convID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE actor_id = $1 AND type = 'dm_message' AND url = $2 AND read_at IS NULL
	`, actorID, "/messages/"+convID)
	return err
}

func (s *Service) UnreadConversationCount(ctx context.Context, actorID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*)::int FROM dm_conversations c
		WHERE (c.participant_a = $1 OR c.participant_b = $1)
		  AND EXISTS (
		    SELECT 1 FROM dm_messages dm
		    LEFT JOIN dm_reads dr ON dr.conversation_id = c.id AND dr.actor_id = $1
		    WHERE dm.conversation_id = c.id AND dm.sender_id <> $1
		      AND dm.created_at > COALESCE(dr.last_read_at, 'epoch'::timestamptz)
		  )
	`, actorID).Scan(&n)
	return n, err
}

func stripHTML(html string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(html, "<", " <"), ">", "> "))
}

func truncatePlain(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}