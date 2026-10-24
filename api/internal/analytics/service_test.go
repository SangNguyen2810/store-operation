package analytics

import (
	"context"
	"testing"
	"time"
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
	byFrom map[time.Time]Totals
	seen   []Filter
}

func (f *fakeRepo) Totals(_ context.Context, flt Filter) (Totals, error) {
	f.seen = append(f.seen, flt)
	return f.byFrom[flt.From], nil
}

func TestPreviousWindow(t *testing.T) {
	cases := []struct {
		name             string
		from, to         string
		wantFrom, wantTo string
	}{
		{"30 days", "2022-06-28", "2022-07-27", "2022-05-29", "2022-06-27"},
		{"single day", "2022-07-27", "2022-07-27", "2022-07-26", "2022-07-26"},
		{"across year", "2022-01-05", "2022-01-11", "2021-12-29", "2022-01-04"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := previousWindow(Filter{From: day(c.from), To: day(c.to)})
			if !got.From.Equal(day(c.wantFrom)) || !got.To.Equal(day(c.wantTo)) {
				t.Fatalf("got %s..%s, want %s..%s", got.From.Format("2006-01-02"), got.To.Format("2006-01-02"), c.wantFrom, c.wantTo)
			}
		})
	}
}

func TestKPIsDeltas(t *testing.T) {
	cur := Filter{From: day("2022-07-01"), To: day("2022-07-10")}
	prev := previousWindow(cur)
	repo := &fakeRepo{byFrom: map[time.Time]Totals{
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
	cur := Filter{From: day("2022-07-01"), To: day("2022-07-10")}
	k, err := NewService(&fakeRepo{byFrom: map[time.Time]Totals{}}).KPIs(context.Background(), cur)
	if err != nil {
		t.Fatal(err)
	}
	if k.AvgPayment.Value != 0 || k.Revenue.DeltaPct != nil {
		t.Errorf("empty data should give zero values and nil deltas, got %+v", k)
	}
}
