package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

type fakeRepo struct {
	Repo
	byFrom map[time.Time]domain.Totals
	seen   []domain.Filter
}

func (f *fakeRepo) Totals(_ context.Context, flt domain.Filter) (domain.Totals, error) {
	f.seen = append(f.seen, flt)
	return f.byFrom[flt.From], nil
}

func TestKPIsDeltas(t *testing.T) {
	cur := domain.Filter{From: day("2022-07-01"), To: day("2022-07-10")}
	prev := cur.PreviousWindow()
	repo := &fakeRepo{byFrom: map[time.Time]domain.Totals{
		cur.From:  {Revenue: 150, Payments: 30, Rentals: 40, ActiveCustomers: 10},
		prev.From: {Revenue: 100, Payments: 20, Rentals: 0, ActiveCustomers: 10},
	}}
	k, err := NewService(repo).KPIs(context.Background(), cur)
	if err != nil {
		t.Fatal(err)
	}
	if k.Revenue.DeltaPct == nil || *k.Revenue.DeltaPct != 50 {
		t.Errorf("revenue delta = %v, want 50", k.Revenue.DeltaPct)
	}
	if k.Rentals.DeltaPct != nil {
		t.Errorf("rentals delta = %v, want nil when previous is 0", *k.Rentals.DeltaPct)
	}
	if k.ActiveCustomers.DeltaPct == nil || *k.ActiveCustomers.DeltaPct != 0 {
		t.Errorf("customers delta = %v, want 0", k.ActiveCustomers.DeltaPct)
	}
	if k.AvgPayment.Value != 5 || k.AvgPayment.Previous != 5 {
		t.Errorf("avg payment = %v/%v, want 5/5", k.AvgPayment.Value, k.AvgPayment.Previous)
	}
}

func TestKPIsNoPayments(t *testing.T) {
	cur := domain.Filter{From: day("2022-07-01"), To: day("2022-07-10")}
	k, err := NewService(&fakeRepo{byFrom: map[time.Time]domain.Totals{}}).KPIs(context.Background(), cur)
	if err != nil {
		t.Fatal(err)
	}
	if k.AvgPayment.Value != 0 || k.Revenue.DeltaPct != nil {
		t.Errorf("empty data should give zero values and nil deltas, got %+v", k)
	}
}

var errBoom = errors.New("boom")

// errRepo fails every call after `okCalls` successful Totals calls.
type errRepo struct {
	Repo
	okCalls int
	calls   int
}

func (e *errRepo) Totals(context.Context, domain.Filter) (domain.Totals, error) {
	e.calls++
	if e.calls > e.okCalls {
		return domain.Totals{}, errBoom
	}
	return domain.Totals{}, nil
}

func TestKPIsPropagatesErrors(t *testing.T) {
	f := domain.Filter{From: day("2022-07-01"), To: day("2022-07-10")}
	for name, ok := range map[string]int{"current window fails": 0, "previous window fails": 1} {
		t.Run(name, func(t *testing.T) {
			repo := &errRepo{okCalls: ok}
			if _, err := NewService(repo).KPIs(context.Background(), f); !errors.Is(err, errBoom) {
				t.Errorf("err = %v, want boom", err)
			}
		})
	}
}

func TestKPIsAsksForPreviousWindow(t *testing.T) {
	f := domain.Filter{From: day("2022-07-01"), To: day("2022-07-10")}
	repo := &fakeRepo{byFrom: map[time.Time]domain.Totals{}}
	if _, err := NewService(repo).KPIs(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if len(repo.seen) != 2 || !repo.seen[0].From.Equal(f.From) || !repo.seen[1].From.Equal(f.PreviousWindow().From) {
		t.Errorf("windows asked: %+v", repo.seen)
	}
}

// passRepo records what the pass-through use cases forward.
type passRepo struct {
	Repo
	gran  domain.Granularity
	limit int
	f     domain.Filter
	err   error
}

func (p *passRepo) Stores(context.Context) ([]domain.Store, error) {
	return []domain.Store{{ID: 1, Label: "A"}}, p.err
}
func (p *passRepo) Revenue(_ context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error) {
	p.f, p.gran = f, g
	return []domain.RevenuePoint{{Bucket: "2022-07-01", StoreID: 1, Revenue: 9}}, p.err
}
func (p *passRepo) TopFilms(_ context.Context, f domain.Filter, n int) ([]domain.FilmStat, error) {
	p.f, p.limit = f, n
	return []domain.FilmStat{{FilmID: 7, Title: "T"}}, p.err
}

func TestPassThroughs(t *testing.T) {
	f := domain.Filter{From: day("2022-07-01"), To: day("2022-07-10")}
	repo := &passRepo{}
	svc := NewService(repo)
	ctx := context.Background()

	if s, err := svc.Stores(ctx); err != nil || len(s) != 1 || s[0].ID != 1 {
		t.Errorf("stores = %v, %v", s, err)
	}
	if r, err := svc.Revenue(ctx, f, domain.Week); err != nil || len(r) != 1 || repo.gran != domain.Week || repo.f != f {
		t.Errorf("revenue = %v, %v, gran %q", r, err, repo.gran)
	}
	if r, err := svc.TopFilms(ctx, f, 3); err != nil || len(r) != 1 || repo.limit != 3 || repo.f != f {
		t.Errorf("films = %v, %v, limit %d", r, err, repo.limit)
	}

	repo.err = errBoom
	if _, err := svc.Stores(ctx); !errors.Is(err, errBoom) {
		t.Errorf("stores err = %v", err)
	}
	if _, err := svc.Revenue(ctx, f, domain.Day); !errors.Is(err, errBoom) {
		t.Errorf("revenue err = %v", err)
	}
	if _, err := svc.TopFilms(ctx, f, 1); !errors.Is(err, errBoom) {
		t.Errorf("films err = %v", err)
	}
}
