package analytics

import (
	"context"
	"math"
	"time"
)

// Repo is the data access seam. The Postgres implementation lives in repo.go.
type Repo interface {
	Stores(ctx context.Context) ([]Store, error)
	Totals(ctx context.Context, f Filter) (Totals, error)
	Revenue(ctx context.Context, f Filter, g Granularity) ([]RevenuePoint, error)
	TopFilms(ctx context.Context, f Filter, limit int) ([]FilmStat, error)
}

type Service struct{ repo Repo }

func NewService(repo Repo) *Service { return &Service{repo: repo} }

func (s *Service) Stores(ctx context.Context) ([]Store, error) { return s.repo.Stores(ctx) }

func (s *Service) Revenue(ctx context.Context, f Filter, g Granularity) ([]RevenuePoint, error) {
	return s.repo.Revenue(ctx, f, g)
}

func (s *Service) TopFilms(ctx context.Context, f Filter, limit int) ([]FilmStat, error) {
	return s.repo.TopFilms(ctx, f, limit)
}

// KPIs compares the window with the immediately preceding window of equal length.
func (s *Service) KPIs(ctx context.Context, f Filter) (KPIs, error) {
	cur, err := s.repo.Totals(ctx, f)
	if err != nil {
		return KPIs{}, err
	}
	prev, err := s.repo.Totals(ctx, previousWindow(f))
	if err != nil {
		return KPIs{}, err
	}
	return KPIs{
		Revenue:         metric(cur.Revenue, prev.Revenue),
		Rentals:         metric(float64(cur.Rentals), float64(prev.Rentals)),
		ActiveCustomers: metric(float64(cur.ActiveCustomers), float64(prev.ActiveCustomers)),
		AvgPayment:      metric(avg(cur), avg(prev)),
	}, nil
}

func previousWindow(f Filter) Filter {
	days := int(f.To.Sub(f.From).Hours()/24) + 1
	prevTo := f.From.AddDate(0, 0, -1)
	return Filter{From: prevTo.AddDate(0, 0, -(days - 1)), To: prevTo, StoreID: f.StoreID}
}

func avg(t Totals) float64 {
	if t.Payments == 0 {
		return 0
	}
	return round2(t.Revenue / float64(t.Payments))
}

func metric(value, previous float64) Metric {
	m := Metric{Value: value, Previous: previous}
	if previous != 0 {
		d := round2((value - previous) / previous * 100)
		m.DeltaPct = &d
	}
	return m
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// endExclusive is the instant after the last day of the window.
func endExclusive(f Filter) time.Time { return f.To.AddDate(0, 0, 1) }
