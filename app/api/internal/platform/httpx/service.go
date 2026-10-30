package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

// Health reports whether the database answers.
func Health(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			WriteProblem(w, http.StatusServiceUnavailable, "database unreachable")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Meta tells the client what "today" is, so the UI never hard-codes the as-of date.
func Meta(asOf time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"asOf": asOf.Format("2006-01-02")})
	}
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
