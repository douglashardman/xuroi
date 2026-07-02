package auth

import (
	"net/url"
	"strings"
)

func RPIDFromOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "localhost"
	}
	u, err := url.Parse(origin)
	if err != nil {
		return "localhost"
	}
	if h := u.Hostname(); h != "" {
		return h
	}
	return "localhost"
}

func (s *Service) applyOrigin(origin string) {
	if s.webauthn == nil || s.webauthn.Config == nil {
		return
	}
	rpID := RPIDFromOrigin(origin)
	s.webauthn.Config.RPID = rpID
	if origin != "" && !originInList(origin, s.webauthn.Config.RPOrigins) {
		s.webauthn.Config.RPOrigins = append(s.webauthn.Config.RPOrigins, origin)
	}
}

func (s *Service) applySessionRPID(sessionRPID string) {
	if s.webauthn == nil || s.webauthn.Config == nil || sessionRPID == "" {
		return
	}
	s.webauthn.Config.RPID = sessionRPID
}

func originInList(origin string, list []string) bool {
	for _, o := range list {
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}