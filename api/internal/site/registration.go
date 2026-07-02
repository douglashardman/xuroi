package site

import (
	"strings"
)

type RegistrationPolicy struct {
	UsernameDenylist     []string `json:"username_denylist"`
	BlockedEmailDomains  []string `json:"blocked_email_domains"`
}

func DefaultRegistrationPolicy() RegistrationPolicy {
	return RegistrationPolicy{
		BlockedEmailDomains: []string{
			"mailinator.com",
			"guerrillamail.com",
			"tempmail.com",
			"throwaway.email",
			"yopmail.com",
		},
	}
}

func (p RegistrationPolicy) Normalized() RegistrationPolicy {
	out := p
	out.UsernameDenylist = normalizeNameList(out.UsernameDenylist)
	out.BlockedEmailDomains = normalizeDomainList(out.BlockedEmailDomains)
	return out
}

func normalizeNameList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{})
	for _, raw := range items {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func normalizeDomainList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{})
	for _, raw := range items {
		key := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(raw), "@"))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func (p RegistrationPolicy) UsernameDenied(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, denied := range p.Normalized().UsernameDenylist {
		if name == denied {
			return true
		}
	}
	return false
}

func (p RegistrationPolicy) EmailDomainBlocked(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	for _, blocked := range p.Normalized().BlockedEmailDomains {
		if domain == blocked || strings.HasSuffix(domain, "."+blocked) {
			return true
		}
	}
	return false
}