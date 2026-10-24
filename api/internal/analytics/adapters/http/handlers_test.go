package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/api/internal/analytics/domain"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/httpx"
)

type fakeSvc struct {
	filter domain.Filter
	gran   domain.Granularity
	limit  int
	err    error
}

func (f *fakeSvc) Stores(context.Context) ([]domain.Store, error) {
	return []domain.Store{{ID: 1, Label: "A"}}, f.err
}
func (f *fakeSvc) KPIs(_ context.Context, fl domain.Filter) (domain.KPIs, error) {
	f.filter = fl
	return domain.KPIs{}, f.err
}
func (f *fakeSvc) Revenue(_ context.Context, fl domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error) {
	f.filter, f.gran = fl, g
	return []domain.RevenuePoint{}, f.err
}
func (f *fakeSvc) TopFilms(_ context.Context, fl domain.Filter, n int) ([]domain.FilmStat, error) {
	f.filter, f.limit = fl, n
	return []domain.FilmStat{}, f.err
}

func handler(svc Analytics) http.Handler {
	mux := http.NewServeMux()
	Register(mux, svc, asOf)
	return mux
}

var asOf = time.Date(2022, 7, 27, 0, 0, 0, 0, time.UTC)

func do(t *testing.T, svc *fakeSvc, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestDefaults(t *testing.T) {
	svc := &fakeSvc{}
	rec := do(t, svc, "/api/revenue")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if got := svc.filter.From.Format("2006-01-02"); got != "2022-06-28" {
		t.Errorf("default from = %s, want 2022-06-28", got)
	}
	if !svc.filter.To.Equal(asOf) || svc.gran != domain.Day || svc.filter.StoreID != nil {
		t.Errorf("unexpected defaults: %+v gran=%s", svc.filter, svc.gran)
	}
}

func TestParamsPassedThrough(t *testing.T) {
	svc := &fakeSvc{}
	rec := do(t, svc, "/api/revenue?from=2022-03-01&to=2022-03-31&store=2&granularity=week")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if svc.filter.StoreID == nil || *svc.filter.StoreID != 2 || svc.gran != domain.Week {
		t.Errorf("got %+v gran=%s", svc.filter, svc.gran)
	}
	do(t, svc, "/api/top/films?limit=3")
	if svc.limit != 3 {
		t.Errorf("limit = %d, want 3", svc.limit)
	}
}

func TestOnlyToGivenDefaultsFrom(t *testing.T) {
	svc := &fakeSvc{}
	do(t, svc, "/api/kpis?to=2022-03-31")
	if got := svc.filter.From.Format("2006-01-02"); got != "2022-03-02" {
		t.Errorf("from = %s, want 2022-03-02", got)
	}
}

func TestBadRequests(t *testing.T) {
	cases := map[string]string{
		"bad from":          "/api/kpis?from=nope",
		"bad to":            "/api/kpis?to=2022-13-01",
		"from after to":     "/api/kpis?from=2022-07-01&to=2022-06-01",
		"range too long":    "/api/kpis?from=2020-01-01&to=2022-01-01",
		"bad store":         "/api/kpis?store=abc",
		"negative store":    "/api/kpis?store=-1",
		"bad granularity":   "/api/revenue?granularity=year",
		"limit zero":        "/api/top/films?limit=0",
		"limit too large":   "/api/top/films?limit=51",
		"limit not numeric": "/api/top/films?limit=x",
	}
	for name, target := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(t, &fakeSvc{}, target)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d, want 400", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("content-type %q", ct)
			}
			var p httpx.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.Status != 400 || p.Detail == "" {
				t.Errorf("bad problem body: %s", rec.Body)
			}
		})
	}
}

func TestServiceErrorHidesDetail(t *testing.T) {
	rec := do(t, &fakeSvc{err: errors.New("secret sql failure")}, "/api/kpis")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "secret") {
		t.Errorf("internal error leaked: %s", body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	handler(&fakeSvc{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/kpis", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status %d, want 405", rec.Code)
	}
}
