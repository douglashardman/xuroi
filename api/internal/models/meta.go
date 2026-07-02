package models

import "time"

type ThreadMeta struct {
	ThreadID     string    `json:"thread_id"`
	Title        string    `json:"title"`
	Slug         string    `json:"slug"`
	URL          string    `json:"url"`
	Category     string    `json:"category"`
	Summary      *string   `json:"summary"`
	ReplyCount   int       `json:"reply_count"`
	Participants []string  `json:"participants"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity_at"`
	ModelVersion string    `json:"model_version"`
	SummaryLabel string    `json:"summary_label"`
}