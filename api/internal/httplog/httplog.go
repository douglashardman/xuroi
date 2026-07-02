package httplog

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Middleware logs one JSON line per request.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		entry := map[string]any{
			"time":       start.UTC().Format(time.RFC3339),
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rw.status,
			"duration_ms": time.Since(start).Milliseconds(),
			"bytes":      rw.bytes,
		}
		if q := r.URL.RawQuery; q != "" {
			entry["query"] = q
		}
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			entry["client_ip"] = fwd
		} else if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			entry["client_ip"] = host
		}
		if ua := r.Header.Get("User-Agent"); ua != "" {
			entry["user_agent"] = ua
		}

		b, _ := json.Marshal(entry)
		log.Println(string(b))
	})
}