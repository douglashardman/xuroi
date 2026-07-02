package auth

import (
	"github.com/go-webauthn/webauthn/webauthn"
)

// WebUser implements webauthn.User for ceremony flows.
type WebUser struct {
	ActorID     string
	DisplayName string
	Email       string
	Credentials []webauthn.Credential
}

func (u WebUser) WebAuthnID() []byte {
	return []byte(u.ActorID)
}

func (u WebUser) WebAuthnName() string {
	if u.Email != "" {
		return u.Email
	}
	return u.DisplayName
}

func (u WebUser) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u WebUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}