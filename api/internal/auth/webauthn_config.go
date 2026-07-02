package auth

import (
	"os"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

func NewWebAuthn(siteName string) (*webauthn.WebAuthn, error) {
	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}

	origins := splitEnvList(os.Getenv("WEBAUTHN_RP_ORIGINS"))
	if len(origins) == 0 {
		origins = []string{"http://localhost:4321", "http://127.0.0.1:4321"}
	}

	displayName := siteName
	if v := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_DISPLAY_NAME")); v != "" {
		displayName = v
	}

	return webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: displayName,
		RPOrigins:     origins,
		// false (default): user.id is base64url in JSON — required by browsers / SimpleWebAuthn
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			ResidentKey:             protocol.ResidentKeyRequirementPreferred,
			UserVerification:        protocol.VerificationPreferred,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Registration: webauthn.TimeoutConfig{Timeout: 60 * time.Second},
			Login:        webauthn.TimeoutConfig{Timeout: 60 * time.Second},
		},
	})
}

func splitEnvList(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}