package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/xuroi/xuroi/api/internal/ids"
)

const (
	CookieName     = "xuroi_session"
	SessionDays    = 30
	tokenBytes     = 32
)

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrEmailTaken     = errors.New("email already registered")
	ErrNotFound       = errors.New("not found")
	ErrInvalidSession = errors.New("invalid session")
)

type Actor struct {
	ID            string   `json:"id"`
	DisplayName   string   `json:"display_name"`
	Email         string   `json:"email,omitempty"`
	IsAdmin       bool     `json:"is_admin,omitempty"`
	IsModerator   bool     `json:"is_moderator,omitempty"`
	CanPermBan    bool     `json:"can_perm_ban,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
	EmailVerified bool     `json:"email_verified,omitempty"`
	State         string   `json:"state,omitempty"`
	BannedUntil   *string  `json:"banned_until,omitempty"`
	BanReason     string   `json:"ban_reason,omitempty"`
}

type Service struct {
	pool          *pgxpool.Pool
	webauthn      *webauthn.WebAuthn
	reservedNames map[string]struct{}
}

func NewService(pool *pgxpool.Pool, wa *webauthn.WebAuthn) *Service {
	return &Service{
		pool:          pool,
		webauthn:      wa,
		reservedNames: BuildReservedSet(nil),
	}
}

type RegisterInput struct {
	DisplayName string
	Email       string
	Password    string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (Actor, string, error) {
	displayName := normalizeDisplayName(in.DisplayName)
	email := normalizeEmail(in.Email)
	if displayName == "" || email == "" {
		return Actor{}, "", ErrInvalidInput
	}
	if err := s.assertDisplayNameAvailable(ctx, displayName); err != nil {
		return Actor{}, "", err
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return Actor{}, "", ErrInvalidInput
	}
	if _, err := s.CheckEmailBanned(ctx, email); errors.Is(err, ErrBanned) {
		return Actor{}, "", ErrBanned
	}
	passwordHash, err := hashPassword(in.Password)
	if err != nil {
		return Actor{}, "", err
	}

	actorID := ids.New("act_")
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Actor{}, "", err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO actors (id, type, display_name, disclosure_required)
		VALUES ($1, 'human', $2, FALSE)
	`, actorID, displayName)
	if err != nil {
		return Actor{}, "", fmt.Errorf("insert actor: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO actor_emails (actor_id, email, verified)
		VALUES ($1, $2, FALSE)
	`, actorID, email)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return Actor{}, "", ErrEmailTaken
		}
		return Actor{}, "", fmt.Errorf("insert email: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO actor_passwords (actor_id, password_hash)
		VALUES ($1, $2)
	`, actorID, passwordHash)
	if err != nil {
		return Actor{}, "", fmt.Errorf("insert password: %w", err)
	}

	token, err := s.insertSession(ctx, tx, actorID)
	if err != nil {
		return Actor{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return Actor{}, "", err
	}

	return Actor{ID: actorID, DisplayName: displayName, Email: email}, token, nil
}

func (s *Service) LoginWithPassword(ctx context.Context, email, password string) (Actor, string, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return Actor{}, "", ErrInvalidInput
	}
	if _, err := s.CheckEmailBanned(ctx, email); errors.Is(err, ErrBanned) {
		return Actor{}, "", ErrBanned
	}

	var actor Actor
	var passwordHash *string
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, a.display_name, e.email, p.password_hash
		FROM actor_emails e
		JOIN actors a ON a.id = e.actor_id
		LEFT JOIN actor_passwords p ON p.actor_id = a.id
		WHERE e.email = $1
	`, email).Scan(&actor.ID, &actor.DisplayName, &actor.Email, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, "", ErrNotFound
	}
	if err != nil {
		return Actor{}, "", err
	}
	if passwordHash == nil || *passwordHash == "" {
		return Actor{}, "", ErrNoPassword
	}
	if err := verifyPassword(*passwordHash, password); err != nil {
		return Actor{}, "", err
	}
	if err := s.CheckSignInAllowed(ctx, actor.ID); err != nil {
		return actor, "", err
	}

	token, err := s.insertSession(ctx, s.pool, actor.ID)
	if err != nil {
		return Actor{}, "", err
	}
	return actor, token, nil
}

// LoginLegacy allows email-only sign-in for accounts with no password and no passkey (dev migration).
func (s *Service) LoginLegacy(ctx context.Context, email string) (Actor, string, error) {
	email = normalizeEmail(email)
	if email == "" {
		return Actor{}, "", ErrInvalidInput
	}

	var actor Actor
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, a.display_name, e.email
		FROM actor_emails e
		JOIN actors a ON a.id = e.actor_id
		WHERE e.email = $1
	`, email).Scan(&actor.ID, &actor.DisplayName, &actor.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, "", ErrNotFound
	}
	if err != nil {
		return Actor{}, "", err
	}

	hasPassword, hasPasskey, err := s.AuthMethods(ctx, actor.ID)
	if err != nil {
		return Actor{}, "", err
	}
	if hasPassword || hasPasskey {
		return Actor{}, "", ErrNoPassword
	}
	if err := s.CheckSignInAllowed(ctx, actor.ID); err != nil {
		return actor, "", err
	}

	token, err := s.insertSession(ctx, s.pool, actor.ID)
	if err != nil {
		return Actor{}, "", err
	}
	return actor, token, nil
}

func (s *Service) SetPassword(ctx context.Context, actorID, password string) error {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO actor_passwords (actor_id, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (actor_id) DO UPDATE SET password_hash = EXCLUDED.password_hash, updated_at = now()
	`, actorID, passwordHash)
	return err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	hash := hashToken(token)
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash)
	return err
}

func (s *Service) ActorFromToken(ctx context.Context, token string) (Actor, error) {
	if token == "" {
		return Actor{}, ErrInvalidSession
	}
	hash := hashToken(token)

	var actor Actor
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, a.display_name, COALESCE(e.email, '')
		FROM sessions s
		JOIN actors a ON a.id = s.actor_id
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		WHERE s.token_hash = $1 AND s.expires_at > now()
	`, hash).Scan(&actor.ID, &actor.DisplayName, &actor.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, ErrInvalidSession
	}
	if err != nil {
		return Actor{}, err
	}
	return actor, nil
}

type sessionDB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func (s *Service) insertSession(ctx context.Context, db sessionDB, actorID string) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}

	sessionID := ids.New("ses_")
	expires := time.Now().Add(SessionDays * 24 * time.Hour)
	_, err = db.Exec(ctx, `
		INSERT INTO sessions (id, actor_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, sessionID, actorID, hashToken(token), expires)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return token, nil
}

func newToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func IsAdminEmail(email string, adminEmails []string) bool {
	email = normalizeEmail(email)
	for _, a := range adminEmails {
		if email == normalizeEmail(a) {
			return true
		}
	}
	return false
}

func (s *Service) WithAdminFlag(actor Actor, adminEmails []string) Actor {
	actor.IsAdmin = IsAdminEmail(actor.Email, adminEmails)
	return actor
}