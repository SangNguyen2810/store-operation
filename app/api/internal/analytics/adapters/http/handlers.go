// Package httpadapter is the driving adapter that exposes the analytics use
// cases over HTTP. It parses and validates requests and shapes responses; it
// holds no SQL and no business rules.
package httpadapter

import (
	"context"
	"net/http"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
	"github.com/SangNguyen2810/store-operation/app/api/internal/platform/httpx"
)

// Analytics is what the handlers need; *app.Service satisfies it.
type Analytics interface {
	Ranges(ctx context.Context) ([]domain.Range, error)
	Stores(ctx context.Context) ([]domain.Store, error)
	KPIs(ctx context.Context, f domain.Filter) (domain.KPIs, error)
	Revenue(ctx context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error)
	TopFilms(ctx context.Context, f domain.Filter, limit int) ([]domain.FilmStat, error)
}

type handlers struct{ svc Analytics }

// Register adds the analytics routes to mux.
func Register(mux *http.ServeMux, svc Analytics) {
	h := &handlers{svc: svc}
	mux.HandleFunc("GET /api/presets", h.presets)
	mux.HandleFunc("GET /api/stores", h.stores)
	mux.HandleFunc("GET /api/kpis", h.kpis)
	mux.HandleFunc("GET /api/revenue", h.revenue)
	mux.HandleFunc("GET /api/top/films", h.topFilms)
}

func (h *handlers) presets(w http.ResponseWriter, r *http.Request) {
	ranges, err := h.svc.Ranges(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRanges(ranges))
}

// filter parses the query against the offered ranges, answering 400 or 500 itself on failure.
func (h *handlers) filter(w http.ResponseWriter, r *http.Request) (domain.Filter, bool) {
	ranges, err := h.svc.Ranges(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return domain.Filter{}, false
	}
	f, err := parseFilter(r, ranges)
	if err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, err.Error())
		return domain.Filter{}, false
	}
	return f, true
}

func (h *handlers) stores(w http.ResponseWriter, r *http.Request) {
	stores, err := h.svc.Stores(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toStores(stores))
}

func (h *handlers) kpis(w http.ResponseWriter, r *http.Request) {
	f, ok := h.filter(w, r)
	if !ok {
		return
	}
	k, err := h.svc.KPIs(r.Context(), f)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toKPIs(k))
}

func (h *handlers) revenue(w http.ResponseWriter, r *http.Request) {
	f, ok := h.filter(w, r)
	if !ok {
		return
	}
	g, err := parseGranularity(r)
	if err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	pts, err := h.svc.Revenue(r.Context(), f, g)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRevenue(pts))
}

func (h *handlers) topFilms(w http.ResponseWriter, r *http.Request) {
	f, ok := h.filter(w, r)
	if !ok {
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	films, err := h.svc.TopFilms(r.Context(), f, limit)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toFilms(films))
}
