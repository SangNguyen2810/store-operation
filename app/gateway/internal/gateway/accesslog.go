package gateway

import (
	"log/slog"
	"net/http"
	"time"
)

// statusWriter remembers the status code for the access log.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the real writer (flushing).
func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// withAccessLog writes one line per request, including the ones the gateway
// rejects itself.
func withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", sw.status,
			"dur", time.Since(start),
			"ip", clientIP(r),
			"request_id", r.Header.Get(requestIDHeader),
		)
	})
}
