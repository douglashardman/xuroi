package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/xuroi/xuroi/api/internal/policy"
	"github.com/xuroi/xuroi/api/internal/spam"
)

type contentCheckResult struct {
	ForcePending bool
}

func (a *API) checkPostContent(ctx context.Context, authorID, markdown string, isStaff, isAdmin bool) (contentCheckResult, error) {
	if isStaff || isAdmin {
		return contentCheckResult{}, nil
	}

	var createdAt time.Time
	err := a.pool.QueryRow(ctx, `SELECT created_at FROM actors WHERE id = $1`, authorID).Scan(&createdAt)
	if err != nil {
		return contentCheckResult{}, fmt.Errorf("actor age: %w", err)
	}
	accountAge := time.Since(createdAt)

	nu := a.siteCfg.NewUsers.Normalized()
	if err := policy.ShouldAllowLinks(nu, accountAge, markdown, isStaff, isAdmin); err != nil {
		return contentCheckResult{}, err
	}

	var recentPosts int
	_ = a.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM posts
		WHERE author_id = $1 AND created_at > now() - interval '1 hour' AND deleted_at IS NULL
	`, authorID).Scan(&recentPosts)

	sp := a.siteCfg.Spam.Normalized()
	result := spam.Evaluate(sp, spam.Input{
		BodyMarkdown: markdown,
		AccountAge:   accountAge,
		RecentPosts:  recentPosts,
	})
	if result.Block {
		msg := "post blocked by spam filter"
		if len(result.Reasons) > 0 {
			msg = "post blocked: " + result.Reasons[0]
		}
		return contentCheckResult{}, errors.New(msg)
	}
	return contentCheckResult{ForcePending: result.Hold}, nil
}

func writeContentPolicyError(w http.ResponseWriter, err error) {
	if errors.Is(err, policy.ErrLinksRestricted) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}