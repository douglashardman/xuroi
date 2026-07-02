package handlers

import (
	"net/http"
	"os"
	"strings"
)

func withCORS(next http.Handler) http.Handler {
	allowed := corsOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && originAllowed(origin, allowed) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie, X-Session-Token")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsOrigins() []string {
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{
		"http://localhost:4321",
		"http://127.0.0.1:4321",
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if strings.TrimSpace(a) == origin {
			return true
		}
	}
	return false
}