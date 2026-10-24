// Package httpapi turns HTTP requests into analytics calls. It holds no SQL.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/SangNguyen2810/store-operation/api/internal/analytics/domain"
)

// Analytics is what the handlers need; *app.Service satisfies it.
type Analytics interface {
	Stores(ctx context.Context) ([]domain.Store, error)
	KPIs(ctx context.Context, f domain.Filter) (domain.KPIs, error)
	Revenue(ctx context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error)
	TopFilms(ctx context.Context, f domain.Filter, limit int) ([]domain.FilmStat, error)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type Server struct {
	svc  Analytics
	db   Pinger
	asOf time.Time
	mux  *http.ServeMux
}

func New(svc Analytics, db Pinger, asOf time.Time) http.Handler {
	s := &Server{svc: svc, db: db, asOf: asOf, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /api/meta", s.meta)
	s.mux.HandleFunc("GET /api/stores", s.stores)
	s.mux.HandleFunc("GET /api/kpis", s.kpis)
	s.mux.HandleFunc("GET /api/revenue", s.revenue)
	s.mux.HandleFunc("GET /api/top/films", s.topFilms)
	return logRequests(s.mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "database unreachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// meta tells the client what "today" is, so the UI never hard-codes the as-of date.
func (s *Server) meta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"asOf": s.asOf.Format(dateLayout)})
}

func (s *Server) stores(w http.ResponseWriter, r *http.Request) {
	stores, err := s.svc.Stores(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stores)
}

func (s *Server) kpis(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r, s.asOf)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	k, err := s.svc.KPIs(r.Context(), f)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, k)
}

func (s *Server) revenue(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r, s.asOf)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	g, err := parseGranularity(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	pts, err := s.svc.Revenue(r.Context(), f, g)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, pts)
}

func (s *Server) topFilms(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r, s.asOf)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	films, err := s.svc.TopFilms(r.Context(), f, limit)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, films)
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "path", r.URL.Path, "err", err)
	writeProblem(w, http.StatusInternalServerError, "internal error")
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "query", r.URL.RawQuery, "dur", time.Since(start))
	})
}
