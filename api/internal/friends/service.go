package friends

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/ids"
	"github.com/xuroi/xuroi/api/internal/slug"
)

var (
	ErrSelfFriend       = errors.New("cannot friend yourself")
	ErrAlreadyFriends   = errors.New("already friends")
	ErrRequestPending   = errors.New("friend request already pending")
	ErrRequestNotFound  = errors.New("friend request not found")
	ErrRequestNotYours  = errors.New("not your friend request")
	ErrMemberNotFound   = errors.New("member not found")
)

const (
	RelNone            = "none"
	RelFriends         = "friends"
	RelPendingSent     = "pending_sent"
	RelPendingReceived = "pending_received"
)

type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Member struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	URL         string `json:"url"`
}

type Request struct {
	ID          string    `json:"id"`
	From        Member    `json:"from"`
	To          Member    `json:"to"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
}

func (s *Service) AreFriends(ctx context.Context, a, b string) (bool, error) {
	if a == b {
		return false, nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM friend_requests
			WHERE status = 'accepted'
			  AND ((from_actor_id = $1 AND to_actor_id = $2)
			    OR (from_actor_id = $2 AND to_actor_id = $1))
		)
	`, a, b).Scan(&exists)
	return exists, err
}

func (s *Service) PendingRequestID(ctx context.Context, viewerID, otherID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM friend_requests
		WHERE from_actor_id = $2 AND to_actor_id = $1 AND status = 'pending'
	`, viewerID, otherID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (s *Service) Relationship(ctx context.Context, viewerID, otherID string) (string, error) {
	if viewerID == "" || otherID == "" || viewerID == otherID {
		return RelNone, nil
	}
	friends, err := s.AreFriends(ctx, viewerID, otherID)
	if err != nil {
		return "", err
	}
	if friends {
		return RelFriends, nil
	}
	var status string
	var fromID string
	err = s.pool.QueryRow(ctx, `
		SELECT status, from_actor_id FROM friend_requests
		WHERE status = 'pending'
		  AND ((from_actor_id = $1 AND to_actor_id = $2)
		    OR (from_actor_id = $2 AND to_actor_id = $1))
		ORDER BY created_at DESC
		LIMIT 1
	`, viewerID, otherID).Scan(&status, &fromID)
	if errors.Is(err, pgx.ErrNoRows) {
		return RelNone, nil
	}
	if err != nil {
		return "", err
	}
	if fromID == viewerID {
		return RelPendingSent, nil
	}
	return RelPendingReceived, nil
}

func (s *Service) member(ctx context.Context, actorID string) (Member, error) {
	var m Member
	err := s.pool.QueryRow(ctx, `
		SELECT id, display_name, COALESCE(avatar_url, '')
		FROM actors WHERE id = $1
	`, actorID).Scan(&m.ID, &m.DisplayName, &m.AvatarURL)
	if err != nil {
		return m, err
	}
	m.URL = "/u/" + slug.FromDisplayName(m.DisplayName)
	return m, nil
}

func (s *Service) SendRequest(ctx context.Context, fromID, toID string) (string, error) {
	if fromID == toID {
		return "", ErrSelfFriend
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM actors WHERE id = $1 AND type = 'human')`, toID).Scan(&exists); err != nil {
		return "", err
	}
	if !exists {
		return "", ErrMemberNotFound
	}
	if friends, err := s.AreFriends(ctx, fromID, toID); err != nil {
		return "", err
	} else if friends {
		return "", ErrAlreadyFriends
	}

	// If they already sent us a request, accept it.
	var incomingID string
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM friend_requests
		WHERE from_actor_id = $2 AND to_actor_id = $1 AND status = 'pending'
	`, fromID, toID).Scan(&incomingID)
	if err == nil {
		if err := s.AcceptRequest(ctx, fromID, incomingID); err != nil {
			return "", err
		}
		return incomingID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	var pendingID string
	err = s.pool.QueryRow(ctx, `
		SELECT id FROM friend_requests
		WHERE from_actor_id = $1 AND to_actor_id = $2 AND status = 'pending'
	`, fromID, toID).Scan(&pendingID)
	if err == nil {
		return pendingID, ErrRequestPending
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	reqID := ids.New("frq_")
	now := time.Now()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO friend_requests (id, from_actor_id, to_actor_id, status, created_at)
		VALUES ($1, $2, $3, 'pending', $4)
	`, reqID, fromID, toID, now)
	if err != nil {
		return "", err
	}
	fromName, _ := s.displayName(ctx, tx, fromID)
	_, err = tx.Exec(ctx, `
		INSERT INTO notifications (id, actor_id, type, from_actor_id, title, body, url, created_at)
		VALUES ($1, $2, 'friend_request', $3, $4, $5, $6, $7)
	`, ids.New("ntf_"), toID, fromID,
		"Friend request from "+fromName,
		fromName+" wants to connect with you.",
		"/messages", now)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return reqID, nil
}

func (s *Service) AcceptRequest(ctx context.Context, actorID, requestID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var fromID, toID string
	err = tx.QueryRow(ctx, `
		SELECT from_actor_id, to_actor_id FROM friend_requests
		WHERE id = $1 AND status = 'pending'
	`, requestID).Scan(&fromID, &toID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRequestNotFound
	}
	if err != nil {
		return err
	}
	if actorID != toID {
		return ErrRequestNotYours
	}
	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE friend_requests SET status = 'accepted', responded_at = $2 WHERE id = $1
	`, requestID, now)
	if err != nil {
		return err
	}
	accepterName, _ := s.displayName(ctx, tx, actorID)
	_, err = tx.Exec(ctx, `
		INSERT INTO notifications (id, actor_id, type, from_actor_id, title, body, url, created_at)
		VALUES ($1, $2, 'friend_accepted', $3, $4, $5, $6, $7)
	`, ids.New("ntf_"), fromID, actorID,
		accepterName+" accepted your friend request",
		"You are now connected.",
		"/u/"+slug.FromDisplayName(accepterName), now)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) DeclineRequest(ctx context.Context, actorID, requestID string) error {
	var toID string
	err := s.pool.QueryRow(ctx, `
		SELECT to_actor_id FROM friend_requests WHERE id = $1 AND status = 'pending'
	`, requestID).Scan(&toID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRequestNotFound
	}
	if err != nil {
		return err
	}
	if actorID != toID {
		return ErrRequestNotYours
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE friend_requests SET status = 'declined', responded_at = now() WHERE id = $1
	`, requestID)
	return err
}

func (s *Service) ListIncoming(ctx context.Context, actorID string, limit int) ([]Request, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, from_actor_id, to_actor_id, status, created_at, responded_at
		FROM friend_requests
		WHERE to_actor_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT $2
	`, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanRequests(ctx, rows)
}

func (s *Service) ListOutgoing(ctx context.Context, actorID string, limit int) ([]Request, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, from_actor_id, to_actor_id, status, created_at, responded_at
		FROM friend_requests
		WHERE from_actor_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT $2
	`, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanRequests(ctx, rows)
}

func (s *Service) PendingIncomingCount(ctx context.Context, actorID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*)::int FROM friend_requests
		WHERE to_actor_id = $1 AND status = 'pending'
	`, actorID).Scan(&n)
	return n, err
}

func (s *Service) scanRequests(ctx context.Context, rows pgx.Rows) ([]Request, error) {
	out := make([]Request, 0)
	for rows.Next() {
		var req Request
		var fromID, toID string
		if err := rows.Scan(&req.ID, &fromID, &toID, &req.Status, &req.CreatedAt, &req.RespondedAt); err != nil {
			return nil, err
		}
		var err error
		req.From, err = s.member(ctx, fromID)
		if err != nil {
			return nil, err
		}
		req.To, err = s.member(ctx, toID)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (s *Service) displayName(ctx context.Context, q querier, actorID string) (string, error) {
	var name string
	err := q.QueryRow(ctx, `SELECT display_name FROM actors WHERE id = $1`, actorID).Scan(&name)
	return name, err
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}