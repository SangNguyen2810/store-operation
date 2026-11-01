package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// Health reports that the process is up. This service has no hard dependency
// to check: without the redis service it serves the built-in list.
func Health(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// LogRequests writes one line per request. When a gateway in front sent an
// X-Request-Id, the line carries it, so one request can be followed across services.
func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		attrs := []any{"method", r.Method, "path", r.URL.Path, "query", r.URL.RawQuery, "dur", time.Since(start)}
		if id := r.Header.Get("X-Request-Id"); id != "" {
			attrs = append(attrs, "request_id", id)
		}
		slog.Info("request", attrs...)
	})
}
