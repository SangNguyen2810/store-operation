package app

import (
	"context"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/api/internal/analytics/domain"
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
