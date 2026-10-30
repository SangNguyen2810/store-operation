// Package httpadapter is the driving adapter that exposes the analytics use
// cases over HTTP. It parses and validates requests and shapes responses; it
// holds no SQL and no business rules.
package httpadapter

import (
	"context"
	"net/http"
	"time"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
	"github.com/SangNguyen2810/store-operation/app/api/internal/platform/httpx"
)

// Analytics is what the handlers need; *app.Service satisfies it.
type Analytics interface {
	Stores(ctx context.Context) ([]domain.Store, error)
	KPIs(ctx context.Context, f domain.Filter) (domain.KPIs, error)
	Revenue(ctx context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error)
	TopFilms(ctx context.Context, f domain.Filter, limit int) ([]domain.FilmStat, error)
}

type handlers struct {
	svc  Analytics
	asOf time.Time
}

// Register adds the analytics routes to mux. asOf anchors the default date range.
func Register(mux *http.ServeMux, svc Analytics, asOf time.Time) {
	h := &handlers{svc: svc, asOf: asOf}
	mux.HandleFunc("GET /api/stores", h.stores)
	mux.HandleFunc("GET /api/kpis", h.kpis)
	mux.HandleFunc("GET /api/revenue", h.revenue)
	mux.HandleFunc("GET /api/top/films", h.topFilms)
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
	f, err := parseFilter(r, h.asOf)
	if err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, err.Error())
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
	f, err := parseFilter(r, h.asOf)
	if err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, err.Error())
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
	f, err := parseFilter(r, h.asOf)
	if err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, err.Error())
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
