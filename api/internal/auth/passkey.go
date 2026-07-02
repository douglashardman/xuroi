package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/xuroi/xuroi/api/internal/ids"
)

type PasskeyBeginResponse struct {
	SessionID string          `json:"session_id"`
	Options   json.RawMessage `json:"options"`
}

func (s *Service) BeginPasskeySignup(ctx context.Context, email, displayName, origin string) (PasskeyBeginResponse, error) {
	s.applyOrigin(origin)
	email = normalizeEmail(email)
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || email == "" {
		return PasskeyBeginResponse{}, ErrInvalidInput
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return PasskeyBeginResponse{}, ErrInvalidInput
	}

	var taken bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM actor_emails WHERE email = $1)`, email).Scan(&taken); err != nil {
		return PasskeyBeginResponse{}, err
	}
	if taken {
		return PasskeyBeginResponse{}, ErrEmailTaken
	}
	if err := s.assertDisplayNameAvailable(ctx, displayName); err != nil {
		return PasskeyBeginResponse{}, err
	}

	actorID := ids.New("act_")
	user := WebUser{ActorID: actorID, DisplayName: displayName, Email: email}

	opts := []webauthn.RegistrationOption{
		webauthn.WithPublicKeyCredentialHints([]protocol.PublicKeyCredentialHints{
			protocol.PublicKeyCredentialHintClientDevice,
		}),
	}
	creation, session, err := s.webauthn.BeginRegistration(user, opts...)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}

	// actor_id is NULL until finish — the planned ID lives in session.UserID
	sessionID, err := s.saveWebauthnSession(ctx, "signup", nil, email, displayName, session)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}

	raw, err := json.Marshal(creation)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}
	return PasskeyBeginResponse{SessionID: sessionID, Options: raw}, nil
}

func (s *Service) FinishPasskeySignup(ctx context.Context, sessionID string, credentialJSON []byte) (Actor, string, error) {
	row, err := s.loadWebauthnSession(ctx, sessionID)
	if err != nil {
		return Actor{}, "", err
	}
	if row.Kind != "signup" {
		return Actor{}, "", ErrInvalidSession
	}
	s.applySessionRPID(row.Session.RelyingPartyID)
	actorID := string(row.Session.UserID)
	if actorID == "" {
		return Actor{}, "", ErrInvalidSession
	}

	user := WebUser{
		ActorID:     actorID,
		DisplayName: row.DisplayName,
		Email:       row.Email,
	}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(credentialJSON)
	if err != nil {
		return Actor{}, "", err
	}

	cred, err := s.webauthn.CreateCredential(user, row.Session, parsed)
	if err != nil {
		return Actor{}, "", err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Actor{}, "", err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO actors (id, type, display_name, disclosure_required)
		VALUES ($1, 'human', $2, FALSE)
	`, user.ActorID, user.DisplayName)
	if err != nil {
		return Actor{}, "", fmt.Errorf("insert actor: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO actor_emails (actor_id, email, verified)
		VALUES ($1, $2, FALSE)
	`, user.ActorID, user.Email)
	if err != nil {
		return Actor{}, "", fmt.Errorf("insert email: %w", err)
	}

	credRaw, err := json.Marshal(cred)
	if err != nil {
		return Actor{}, "", err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO webauthn_credentials (id, actor_id, credential_id, credential_json)
		VALUES ($1, $2, $3, $4)
	`, ids.New("wac_"), user.ActorID, cred.ID, credRaw)
	if err != nil {
		return Actor{}, "", err
	}

	token, err := s.insertSession(ctx, tx, user.ActorID)
	if err != nil {
		return Actor{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return Actor{}, "", err
	}
	_ = s.deleteWebauthnSession(ctx, sessionID)

	return Actor{ID: user.ActorID, DisplayName: user.DisplayName, Email: user.Email}, token, nil
}

func (s *Service) BeginPasskeyRegister(ctx context.Context, actorID, origin string) (PasskeyBeginResponse, error) {
	s.applyOrigin(origin)
	user, err := s.loadWebUser(ctx, actorID)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}

	opts := []webauthn.RegistrationOption{
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementPreferred),
		webauthn.WithExclusions(webauthn.Credentials(user.Credentials).CredentialDescriptors()),
	}

	creation, session, err := s.webauthn.BeginRegistration(user, opts...)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}

	sessionID, err := s.saveWebauthnSession(ctx, "registration", &actorID, user.Email, user.DisplayName, session)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}

	raw, err := json.Marshal(creation)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}
	return PasskeyBeginResponse{SessionID: sessionID, Options: raw}, nil
}

func (s *Service) FinishPasskeyRegister(ctx context.Context, actorID, sessionID string, credentialJSON []byte) error {
	row, err := s.loadWebauthnSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if row.Kind != "registration" || row.ActorID == nil || *row.ActorID != actorID {
		return ErrInvalidSession
	}
	s.applySessionRPID(row.Session.RelyingPartyID)

	user, err := s.loadWebUser(ctx, actorID)
	if err != nil {
		return err
	}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(credentialJSON)
	if err != nil {
		return err
	}

	cred, err := s.webauthn.CreateCredential(user, row.Session, parsed)
	if err != nil {
		return err
	}

	if err := s.saveCredential(ctx, actorID, cred, ""); err != nil {
		return err
	}
	return s.deleteWebauthnSession(ctx, sessionID)
}

func (s *Service) BeginPasskeyLogin(ctx context.Context, origin string) (PasskeyBeginResponse, error) {
	s.applyOrigin(origin)
	assertion, session, err := s.webauthn.BeginDiscoverableLogin(
		webauthn.WithAssertionPublicKeyCredentialHints([]protocol.PublicKeyCredentialHints{
			protocol.PublicKeyCredentialHintClientDevice,
		}),
	)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}

	sessionID, err := s.saveWebauthnSession(ctx, "login", nil, "", "", session)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}

	raw, err := json.Marshal(assertion)
	if err != nil {
		return PasskeyBeginResponse{}, err
	}
	return PasskeyBeginResponse{SessionID: sessionID, Options: raw}, nil
}

func (s *Service) FinishPasskeyLogin(ctx context.Context, sessionID string, credentialJSON []byte) (Actor, string, error) {
	row, err := s.loadWebauthnSession(ctx, sessionID)
	if err != nil {
		return Actor{}, "", err
	}
	if row.Kind != "login" {
		return Actor{}, "", ErrInvalidSession
	}
	s.applySessionRPID(row.Session.RelyingPartyID)

	parsed, err := protocol.ParseCredentialRequestResponseBytes(credentialJSON)
	if err != nil {
		return Actor{}, "", err
	}

	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		if len(userHandle) > 0 {
			return s.loadWebUser(ctx, string(userHandle))
		}
		wu, err := s.loadWebUserByCredentialID(ctx, rawID)
		if err != nil {
			return nil, err
		}
		return wu, nil
	}

	validatedUser, validatedCred, err := s.webauthn.ValidatePasskeyLogin(handler, row.Session, parsed)
	if err != nil {
		return Actor{}, "", err
	}

	wu, ok := validatedUser.(WebUser)
	if !ok {
		return Actor{}, "", fmt.Errorf("unexpected user type")
	}

	if err := s.updateCredential(ctx, wu.ActorID, validatedCred); err != nil {
		return Actor{}, "", err
	}

	if err := s.CheckSignInAllowed(ctx, wu.ActorID); err != nil {
		return Actor{ID: wu.ActorID, DisplayName: wu.DisplayName, Email: wu.Email}, "", err
	}

	token, err := s.insertSession(ctx, s.pool, wu.ActorID)
	if err != nil {
		return Actor{}, "", err
	}
	_ = s.deleteWebauthnSession(ctx, sessionID)

	return Actor{ID: wu.ActorID, DisplayName: wu.DisplayName, Email: wu.Email}, token, nil
}