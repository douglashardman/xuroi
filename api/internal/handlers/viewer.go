package handlers

import (
	"errors"
	"net/http"

	"github.com/xuroi/xuroi/api/internal/access"
	"github.com/xuroi/xuroi/api/internal/auth"
)

func (a *API) viewerFromRequest(r *http.Request) (access.Viewer, error) {
	v := access.Viewer{
		IsGuest:      true,
		Entitlements: map[string]bool{},
	}
	actor, err := a.actorFromRequest(r)
	if errors.Is(err, auth.ErrInvalidSession) {
		return v, nil
	}
	if err != nil {
		return v, err
	}
	enriched, err := a.auth.EnrichActor(
		r.Context(),
		actor,
		a.siteCfg.Admin.Emails,
		a.siteCfg.Admin.ModeratorEmails,
		a.siteCfg.Admin.PermBanModeratorEmails,
	)
	if err != nil {
		return v, err
	}
	a.auth.TouchLastActive(r.Context(), actor.ID)
	ents, err := a.auth.LoadEntitlements(r.Context(), actor.ID)
	if err != nil {
		return v, err
	}

	id := actor.ID
	v.ActorID = &id
	v.IsGuest = false
	v.IsMember = enriched.EmailVerified
	v.IsStaff = enriched.IsModerator
	v.IsAdmin = enriched.IsAdmin
	for _, ent := range ents {
		v.Entitlements[ent] = true
	}
	return v, nil
}

func (a *API) viewerFromActor(r *http.Request, actor auth.Actor) (access.Viewer, error) {
	enriched, err := a.auth.EnrichActor(
		r.Context(),
		actor,
		a.siteCfg.Admin.Emails,
		a.siteCfg.Admin.ModeratorEmails,
		a.siteCfg.Admin.PermBanModeratorEmails,
	)
	if err != nil {
		return access.Viewer{IsGuest: true, Entitlements: map[string]bool{}}, err
	}
	ents, err := a.auth.LoadEntitlements(r.Context(), actor.ID)
	if err != nil {
		return access.Viewer{IsGuest: true, Entitlements: map[string]bool{}}, err
	}

	id := actor.ID
	v := access.Viewer{
		ActorID:      &id,
		IsGuest:      false,
		IsMember:     enriched.EmailVerified,
		IsStaff:      enriched.IsModerator,
		IsAdmin:      enriched.IsAdmin,
		Entitlements: map[string]bool{},
	}
	for _, ent := range ents {
		v.Entitlements[ent] = true
	}
	return v, nil
}