// Package server is the redis service's HTTP API: a small key-value store with
// expiry, in front of Redis.
//
//	GET /v1/cache/{key}            the stored bytes, 404 when missing; X-Cache-Ttl is the seconds left
//	PUT /v1/cache/{key}?ttl=10m    store the request body, 204
//	GET /v1/stats                  counters since start
//	GET /healthz                   whether Redis answers
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SangNguyen2810/store-operation/app/redis/internal/store"
)

const (
	maxValueBytes = 1 << 20
	maxTTL        = 24 * time.Hour
	// Keys under this prefix are settings, not cache entries: they may be stored without an expiry.
	persistentPrefix = "config:"
	ttlHeader        = "X-Cache-Ttl"
)

var validKey = regexp.MustCompile(`^[A-Za-z0-9:._-]{1,200}$`)

// Backend is what the handlers need from the store.
type Backend interface {
	Get(ctx context.Context, key string) (val []byte, ttl time.Duration, found bool, err error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Ping(ctx context.Context) error
	Stats() store.Stats
}

func New(b Backend) http.Handler {
	h := &handler{b: b}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/cache/{key}", h.get)
	mux.HandleFunc("PUT /v1/cache/{key}", h.put)
	mux.HandleFunc("GET /v1/stats", h.stats)
	mux.HandleFunc("GET /healthz", h.health)
	return withRequestLog(mux)
}

type handler struct{ b Backend }

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !validKey.MatchString(key) {
		writeProblem(w, http.StatusBadRequest, "key: want 1 to 200 letters, digits or : . _ -")
		return
	}
	val, ttl, found, err := h.b.Get(r.Context(), key)
	if err != nil {
		h.unavailable(w, r, err)
		return
	}
	if !found {
		writeProblem(w, http.StatusNotFound, "no such key")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	if ttl > 0 {
		w.Header().Set(ttlHeader, strconv.Itoa(int(math.Ceil(ttl.Seconds()))))
	}
	_, _ = w.Write(val)
}

func (h *handler) put(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !validKey.MatchString(key) {
		writeProblem(w, http.StatusBadRequest, "key: want 1 to 200 letters, digits or : . _ -")
		return
	}
	ttl, problem := parseTTL(key, r.URL.Query().Get("ttl"))
	if problem != "" {
		writeProblem(w, http.StatusBadRequest, problem)
		return
	}
	val, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxValueBytes))
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig):
		writeProblem(w, http.StatusRequestEntityTooLarge, "value must be at most 1 MiB")
		return
	case err != nil:
		writeProblem(w, http.StatusBadRequest, "could not read the body")
		return
	case len(val) == 0:
		writeProblem(w, http.StatusBadRequest, "body: want a value")
		return
	}
	if err := h.b.Set(r.Context(), key, val, ttl); err != nil {
		h.unavailable(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseTTL returns the expiry for key, or the reason the request is wrong.
// 0 means the key never expires, which only settings may do.
func parseTTL(key, raw string) (time.Duration, string) {
	if raw == "" {
		if strings.HasPrefix(key, persistentPrefix) {
			return 0, ""
		}
		return 0, "ttl: required for keys outside " + persistentPrefix
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 || d > maxTTL {
		return 0, "ttl: want a duration above 0 and up to 24h, such as 10m"
	}
	return d, ""
}

func (h *handler) stats(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.b.Stats())
}

func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.b.Ping(r.Context()); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "redis unreachable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`+"\n")
}

// unavailable logs the real error and tells the caller only that the backend is down.
func (h *handler) unavailable(w http.ResponseWriter, r *http.Request, err error) {
	slog.Warn("redis failed", "path", r.URL.Path, "request_id", r.Header.Get("X-Request-Id"), "err", err)
	writeProblem(w, http.StatusServiceUnavailable, "cache backend unavailable")
}
