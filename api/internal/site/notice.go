package site

import "strings"

type NoticePolicy struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

func DefaultNoticePolicy() NoticePolicy {
	return NoticePolicy{}
}

func (p NoticePolicy) Normalized() NoticePolicy {
	out := p
	out.Message = strings.TrimSpace(out.Message)
	if out.Enabled && out.Message == "" {
		out.Enabled = false
	}
	return out
}