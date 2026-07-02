package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/query"
	"github.com/xuroi/xuroi/api/internal/service"
)

func (a *API) listAdminUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}

	limit := 50
	offset := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}
	q := r.URL.Query().Get("q")

	users, total, err := a.reader.ListAdminUsers(r.Context(), q, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users": users,
		"total": total,
	})
}

func (a *API) banUser(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}

	actorID := r.PathValue("id")
	var req struct {
		Reason       string `json:"reason"`
		Duration     string `json:"duration"`
		Discouraged  bool   `json:"discouraged"`
		PurgeContent bool   `json:"purge_content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Discouraged {
		if err := a.auth.SetDiscouraged(r.Context(), actorID, true); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "discouraged"})
		return
	}

	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "message to member required")
		return
	}
	duration, err := auth.ParseBanDuration(req.Duration)
	if err != nil {
		writeError(w, http.StatusBadRequest, "duration must be 7d, 30d, or permanent")
		return
	}
	if !auth.CanBanDuration(staff, duration) {
		writeError(w, http.StatusForbidden, "you cannot issue this ban duration")
		return
	}
	if staff.ID == actorID {
		writeError(w, http.StatusBadRequest, "cannot ban yourself")
		return
	}

	var purgeResult *service.PurgeAuthorResult
	if req.PurgeContent {
		purged, perr := a.forum.PurgeAuthorContent(r.Context(), actorID, staff.ID)
		if perr != nil {
			writeError(w, http.StatusInternalServerError, perr.Error())
			return
		}
		purgeResult = &purged
	}

	if err := a.auth.BanUserFull(r.Context(), actorID, staff.ID, duration, req.Reason); errors.Is(err, auth.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	} else if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "message to member required")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := map[string]any{"status": "banned", "duration": string(duration)}
	if purgeResult != nil {
		out["posts_removed"] = purgeResult.PostsRemoved
		out["threads_removed"] = purgeResult.ThreadsRemoved
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) unbanUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}

	actorID := r.PathValue("id")
	if err := a.auth.ClearBan(r.Context(), actorID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = a.auth.SetDiscouraged(r.Context(), actorID, false)
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func (a *API) getAdminOverview(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}

	stats, err := a.reader.AdminOverview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (a *API) warnUser(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}

	actorID := r.PathValue("id")
	var req struct {
		Message string  `json:"message"`
		PostID  *string `json:"post_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "warning message required")
		return
	}

	result, err := a.auth.IssueWarning(r.Context(), actorID, staff.ID, req.Message, req.PostID)
	if errors.Is(err, auth.ErrPostAlreadyWarned) {
		writeError(w, http.StatusConflict, "this post was already warned")
		return
	}
	if errors.Is(err, auth.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "warning message required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := map[string]any{
		"status":        "warned",
		"warning_count": result.WarningCount,
		"auto_banned":   result.AutoBanned,
		"consolidated":  result.Consolidated,
	}
	if result.Consolidated {
		out["message"] = "Same 24h incident — no extra strike (overlay refreshed)"
	}
	if result.AutoBanned {
		out["message"] = "Third strike — member automatically banned for 7 days"
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getAdminUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	actorID := r.PathValue("id")
	user, err := a.reader.AdminUserByID(r.Context(), actorID)
	if errors.Is(err, query.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	perms, err := a.auth.LoadPermissions(r.Context(), actorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":        user,
		"permissions": perms,
	})
}

func (a *API) permissionCatalog(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"permissions": auth.PermissionCatalog(),
	})
}

func (a *API) setUserPermissions(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}

	actorID := r.PathValue("id")
	var req struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if actorID == admin.ID {
		writeError(w, http.StatusBadRequest, "cannot change your own permissions")
		return
	}

	if err := a.auth.SetActorPermissions(r.Context(), actorID, req.Permissions, admin.ID); errors.Is(err, auth.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	} else if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid permission or actor type")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	perms, _ := a.auth.LoadPermissions(r.Context(), actorID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "updated",
		"permissions": perms,
	})
}

func (a *API) setUserEntitlements(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}

	actorID := r.PathValue("id")
	var req struct {
		Entitlements []string `json:"entitlements"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := a.auth.SetEntitlements(r.Context(), actorID, req.Entitlements, admin.ID); errors.Is(err, auth.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	} else if errors.Is(err, auth.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid entitlement or actor type")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ents, _ := a.auth.LoadEntitlements(r.Context(), actorID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "updated",
		"entitlements": ents,
	})
}