package site

import "strings"

type MaintenancePolicy struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

func DefaultMaintenancePolicy() MaintenancePolicy {
	return MaintenancePolicy{
		Message: "We're doing a quick tune-up. Back shortly — thanks for your patience.",
	}
}

func (p MaintenancePolicy) Normalized() MaintenancePolicy {
	out := p
	if strings.TrimSpace(out.Message) == "" {
		out.Message = DefaultMaintenancePolicy().Message
	}
	return out
}