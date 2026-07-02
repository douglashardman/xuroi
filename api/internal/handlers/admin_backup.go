package handlers

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/xuroi/xuroi/api/internal/events"
)

func (a *API) triggerBackup(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r)
	if !ok {
		return
	}

	script := os.Getenv("BACKUP_SCRIPT")
	if script == "" {
		script = filepath.Join("..", "infra", "backup.sh")
	}
	if abs, err := filepath.Abs(script); err == nil {
		script = abs
	}

	cmd := exec.Command(script)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	result := map[string]any{
		"status":    "ok",
		"ran_at":    time.Now().UTC(),
		"output":    string(out),
	}
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		writeJSON(w, http.StatusInternalServerError, result)
		return
	}

	_ = a.forum.LogAdminEvent(r.Context(), events.TypeAdminBackupTriggered, admin.ID, map[string]string{
		"script": script,
	})

	writeJSON(w, http.StatusOK, result)
}