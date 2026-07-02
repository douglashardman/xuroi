package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/auth"
)

func (a *API) enrichActor(w http.ResponseWriter, r *http.Request, actor auth.Actor) (auth.Actor, bool) {
	return a.enrichActorFull(w, r, actor)
}

func (a *API) enrichActorFull(w http.ResponseWriter, r *http.Request, actor auth.Actor) (auth.Actor, bool) {
	enriched, err := a.auth.EnrichActor(r.Context(), actor, a.siteCfg.Admin.Emails, a.siteCfg.Admin.ModeratorEmails, a.siteCfg.Admin.PermBanModeratorEmails)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return auth.Actor{}, false
	}
	a.auth.TouchLastActive(r.Context(), enriched.ID)
	return enriched, true
}

func (a *API) requireWritableActor(w http.ResponseWriter, r *http.Request) (auth.Actor, bool) {
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return auth.Actor{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return auth.Actor{}, false
	}
	enriched, ok := a.enrichActor(w, r, actor)
	if !ok {
		return auth.Actor{}, false
	}
	if _, err := a.auth.CheckWritable(r.Context(), enriched.ID); errors.Is(err, auth.ErrBanned) {
		if info, derr := a.auth.ActorBanInfo(r.Context(), enriched.ID); derr == nil {
			writeBanError(w, info)
		} else {
			writeError(w, http.StatusForbidden, "account banned")
		}
		return auth.Actor{}, false
	} else if errors.Is(err, auth.ErrEmailNotVerified) {
		writeError(w, http.StatusForbidden, "confirm your email before posting")
		return auth.Actor{}, false
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return auth.Actor{}, false
	}
	return enriched, true
}

func (a *API) actorJSON(ctx context.Context, actor auth.Actor) map[string]any {
	ents, _ := a.auth.LoadEntitlements(ctx, actor.ID)
	out := map[string]any{
		"id":             actor.ID,
		"display_name":   actor.DisplayName,
		"email":          actor.Email,
		"is_admin":       actor.IsAdmin,
		"is_moderator":   actor.IsModerator,
		"can_perm_ban":   actor.CanPermBan,
		"permissions":    actor.Permissions,
		"entitlements":   ents,
		"email_verified": actor.EmailVerified,
		"state":          actor.State,
	}
	if actor.BannedUntil != nil {
		out["banned_until"] = *actor.BannedUntil
	}
	if actor.BanReason != "" {
		out["ban_reason"] = actor.BanReason
	}
	if warning, err := a.auth.ActiveWarning(ctx, actor.ID); err == nil && warning != nil {
		out["active_warning"] = warning
	}
	var avatarURL *string
	if err := a.pool.QueryRow(ctx, `SELECT avatar_url FROM actors WHERE id = $1`, actor.ID).Scan(&avatarURL); err == nil && avatarURL != nil && *avatarURL != "" {
		out["avatar_url"] = *avatarURL
	}
	return out
}

func (a *API) authResponse(ctx context.Context, actor auth.Actor, token string) map[string]any {
	out := a.actorJSON(ctx, actor)
	out["token"] = token
	return out
}