package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/xuroi/xuroi/api/internal/auth"
)

func (a *API) emailUnsubscribe(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" && r.Method == http.MethodPost {
		_ = r.ParseForm()
		token = strings.TrimSpace(r.FormValue("token"))
		if token == "" {
			token = strings.TrimSpace(r.FormValue("List-Unsubscribe"))
		}
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token required")
		return
	}

	title, err := a.auth.UnsubscribeThreadEmail(r.Context(), token)
	if errors.Is(err, auth.ErrInvalidToken) {
		writeError(w, http.StatusBadRequest, "link expired or invalid")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "unsubscribed",
		"thread_title": title,
		"message":      "You will no longer receive emails about this thread.",
	})
}