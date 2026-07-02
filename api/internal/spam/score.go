package spam

import (
	"regexp"
	"strings"
	"time"
)

type Policy struct {
	Enabled           bool `json:"enabled"`
	MaxLinksNewUser   int  `json:"max_links_new_user"`
	NewAccountHours   int  `json:"new_account_hours"`
	ScoreThreshold    int  `json:"score_threshold"`
	HoldForModeration bool `json:"hold_for_moderation"`
}

func (p Policy) Normalized() Policy {
	out := p
	if !out.Enabled {
		return out
	}
	if out.MaxLinksNewUser <= 0 {
		out.MaxLinksNewUser = 2
	}
	if out.NewAccountHours <= 0 {
		out.NewAccountHours = 48
	}
	if out.ScoreThreshold <= 0 {
		out.ScoreThreshold = 6
	}
	out.HoldForModeration = true
	return out
}

type Input struct {
	BodyMarkdown string
	AccountAge   time.Duration
	RecentPosts  int
}

type Result struct {
	Score   int      `json:"score"`
	Reasons []string `json:"reasons,omitempty"`
	Block   bool     `json:"block"`
	Hold    bool     `json:"hold"`
}

var (
	linkRe    = regexp.MustCompile(`(?i)(https?://|www\.|\[[^\]]+\]\([^)]+\))`)
	spamWords = []string{"casino", "viagra", "crypto pump", "click here now", "buy followers"}
)

func Evaluate(policy Policy, in Input) Result {
	policy = policy.Normalized()
	if !policy.Enabled {
		return Result{}
	}

	var score int
	var reasons []string

	links := len(linkRe.FindAllString(in.BodyMarkdown, -1))
	isNew := in.AccountAge < time.Duration(policy.NewAccountHours)*time.Hour

	if isNew && links > policy.MaxLinksNewUser {
		add := 4 + (links - policy.MaxLinksNewUser)
		score += add
		reasons = append(reasons, "too many links for new account")
	} else if links >= 6 {
		score += 3
		reasons = append(reasons, "many links")
	}

	if isNew && in.RecentPosts >= 5 {
		score += 2
		reasons = append(reasons, "high post rate")
	}

	lower := strings.ToLower(in.BodyMarkdown)
	for _, word := range spamWords {
		if strings.Contains(lower, word) {
			score += 3
			reasons = append(reasons, "spam phrase")
			break
		}
	}

	if isNew && len(strings.TrimSpace(in.BodyMarkdown)) < 12 && links > 0 {
		score += 2
		reasons = append(reasons, "short link post")
	}

	out := Result{Score: score, Reasons: reasons}
	if score >= policy.ScoreThreshold+3 {
		out.Block = true
	} else if score >= policy.ScoreThreshold {
		out.Hold = policy.HoldForModeration
		if !out.Hold {
			out.Block = true
		}
	}
	return out
}