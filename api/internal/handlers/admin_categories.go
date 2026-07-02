package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/access"
	"github.com/xuroi/xuroi/api/internal/events"
	"github.com/xuroi/xuroi/api/internal/service"
)

func (a *API) listAdminCategories(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	viewer, err := a.viewerFromRequest(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	home, err := a.reader.Home(r.Context(), viewer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"groups":     home.Groups,
		"categories": home.Categories,
	})
}

func (a *API) updateAdminCategory(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}

	categoryID := r.PathValue("id")
	var req struct {
		Slug         string   `json:"slug"`
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		SortOrder    int      `json:"sort_order"`
		ParentID     *string  `json:"parent_id"`
		AccessLevel  string   `json:"access_level"`
		AccessLevels []string `json:"access_levels"`
		ListPublic   *bool    `json:"list_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "slug and name required")
		return
	}

	evt, err := a.forum.UpdateCategory(r.Context(), service.UpdateCategoryInput{
		CategoryID:   categoryID,
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		SortOrder:    req.SortOrder,
		ParentID:     req.ParentID,
		AccessLevel:  req.AccessLevel,
		AccessLevels: req.AccessLevels,
		ListPublic:   req.ListPublic,
		ActorID:      admin.ID,
	})
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, evt)
}

func (a *API) deleteAdminCategory(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}

	categoryID := r.PathValue("id")
	cascade := r.URL.Query().Get("cascade") == "true"
	evt, err := a.forum.DeleteCategory(r.Context(), categoryID, admin.ID, cascade)
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, evt)
}

func (a *API) reorderAdminCategories(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}

	var req struct {
		Items []events.CategoryReorderItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	evts, err := a.forum.ReorderCategories(r.Context(), service.ReorderCategoriesInput{
		Items:   req.Items,
		ActorID: admin.ID,
	})
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": evts})
}

func (a *API) listAccessLevels(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"levels":       access.LevelCatalog(),
		"entitlements": access.EntitlementCatalog(),
	})
}

func writeCategoryError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"),
		strings.Contains(msg, "has child"),
		strings.Contains(msg, "has threads"),
		strings.Contains(msg, "parent"),
		strings.Contains(msg, "required"),
		strings.Contains(msg, "own parent"):
		writeError(w, http.StatusBadRequest, msg)
	default:
		writeError(w, http.StatusInternalServerError, msg)
	}
}