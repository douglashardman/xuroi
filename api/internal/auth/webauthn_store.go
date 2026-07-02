package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/ids"
)

const webauthnSessionTTL = 5 * time.Minute

func (s *Service) loadWebUser(ctx context.Context, actorID string) (WebUser, error) {
	var user WebUser
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, a.display_name, COALESCE(e.email, '')
		FROM actors a
		LEFT JOIN actor_emails e ON e.actor_id = a.id
		WHERE a.id = $1
	`, actorID).Scan(&user.ActorID, &user.DisplayName, &user.Email)
	if err == pgx.ErrNoRows {
		return WebUser{}, ErrNotFound
	}
	if err != nil {
		return WebUser{}, err
	}
	creds, err := s.loadCredentials(ctx, actorID)
	if err != nil {
		return WebUser{}, err
	}
	user.Credentials = creds
	return user, nil
}

func (s *Service) loadWebUserByCredentialID(ctx context.Context, rawID []byte) (WebUser, error) {
	var actorID string
	err := s.pool.QueryRow(ctx, `
		SELECT actor_id FROM webauthn_credentials WHERE credential_id = $1
	`, rawID).Scan(&actorID)
	if err == pgx.ErrNoRows {
		return WebUser{}, ErrNotFound
	}
	if err != nil {
		return WebUser{}, err
	}
	return s.loadWebUser(ctx, actorID)
}

func (s *Service) loadCredentials(ctx context.Context, actorID string) ([]webauthn.Credential, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT credential_json FROM webauthn_credentials WHERE actor_id = $1 ORDER BY created_at
	`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []webauthn.Credential
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var cred webauthn.Credential
		if err := json.Unmarshal(raw, &cred); err != nil {
			return nil, fmt.Errorf("decode credential: %w", err)
		}
		out = append(out, cred)
	}
	return out, rows.Err()
}

func (s *Service) saveCredential(ctx context.Context, actorID string, cred *webauthn.Credential, deviceName string) error {
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO webauthn_credentials (id, actor_id, credential_id, credential_json, device_name)
		VALUES ($1, $2, $3, $4, $5)
	`, ids.New("wac_"), actorID, cred.ID, raw, deviceName)
	return err
}

func (s *Service) updateCredential(ctx context.Context, actorID string, cred *webauthn.Credential) error {
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE webauthn_credentials
		SET credential_json = $3, last_used_at = now()
		WHERE actor_id = $1 AND credential_id = $2
	`, actorID, cred.ID, raw)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type webauthnSessionRow struct {
	Kind        string
	ActorID     *string
	Email       string
	DisplayName string
	Session     webauthn.SessionData
}

func (s *Service) saveWebauthnSession(ctx context.Context, kind string, actorID *string, email, displayName string, session *webauthn.SessionData) (string, error) {
	if session == nil {
		return "", fmt.Errorf("missing session data")
	}
	if session.Expires.IsZero() {
		session.Expires = time.Now().Add(webauthnSessionTTL)
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	id := ids.New("wbs_")
	_, err = s.pool.Exec(ctx, `
		INSERT INTO webauthn_sessions (id, kind, actor_id, email, display_name, session_data, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, kind, actorID, email, displayName, raw, session.Expires)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) loadWebauthnSession(ctx context.Context, sessionID string) (webauthnSessionRow, error) {
	var row webauthnSessionRow
	var actorID *string
	var raw []byte
	var expires time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT kind, actor_id, COALESCE(email, ''), COALESCE(display_name, ''), session_data, expires_at
		FROM webauthn_sessions
		WHERE id = $1
	`, sessionID).Scan(&row.Kind, &actorID, &row.Email, &row.DisplayName, &raw, &expires)
	if err == pgx.ErrNoRows {
		return webauthnSessionRow{}, ErrInvalidSession
	}
	if err != nil {
		return webauthnSessionRow{}, err
	}
	if time.Now().After(expires) {
		_ = s.deleteWebauthnSession(ctx, sessionID)
		return webauthnSessionRow{}, ErrInvalidSession
	}
	if err := json.Unmarshal(raw, &row.Session); err != nil {
		return webauthnSessionRow{}, err
	}
	row.ActorID = actorID
	return row, nil
}

func (s *Service) deleteWebauthnSession(ctx context.Context, sessionID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM webauthn_sessions WHERE id = $1`, sessionID)
	return err
}

func (s *Service) hasPassword(ctx context.Context, actorID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM actor_passwords WHERE actor_id = $1)
	`, actorID).Scan(&exists)
	return exists, err
}

func (s *Service) hasPasskey(ctx context.Context, actorID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM webauthn_credentials WHERE actor_id = $1)
	`, actorID).Scan(&exists)
	return exists, err
}

func (s *Service) AuthMethods(ctx context.Context, actorID string) (hasPassword, hasPasskey bool, err error) {
	hasPassword, err = s.hasPassword(ctx, actorID)
	if err != nil {
		return false, false, err
	}
	hasPasskey, err = s.hasPasskey(ctx, actorID)
	return hasPassword, hasPasskey, err
}