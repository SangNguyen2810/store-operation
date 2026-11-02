//go:build integration

// These tests run the real SQL against Pagila. They need DATABASE_URL and fail
// without it: `make test-integration` locally, the go-integration job in CI.
// Expected figures were taken from Pagila at the commit pinned in
// .github/actions/load-pagila, using plain SQL written independently of repo.go.
package postgres_test

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/app/analytics/internal/analytics/adapters/postgres"
	"github.com/SangNguyen2810/store-operation/app/analytics/internal/analytics/domain"
	"github.com/SangNguyen2810/store-operation/app/analytics/internal/platform/db"
)

func repo(t *testing.T) *postgres.Repo {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is not set: start Pagila and set it (make test-integration does this for localhost:5432)")
	}
	pool, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgres.New(pool)
}

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func window(from, to string, store ...int) domain.Filter {
	f := domain.Filter{From: day(from), To: day(to)}
	if len(store) > 0 {
		f.StoreID = &store[0]
	}
	return f
}

func near(got, want float64) bool { return math.Abs(got-want) < 0.005 }

func TestTotals(t *testing.T) {
	r := repo(t)
	cases := []struct {
		name string
		f    domain.Filter
		want domain.Totals
	}{
		{"all stores", window("2022-07-01", "2022-07-10"), domain.Totals{Revenue: 3608.24, Payments: 878, Rentals: 2520, ActiveCustomers: 592}},
		{"store 1", window("2022-07-01", "2022-07-10", 1), domain.Totals{Revenue: 1822.60, Payments: 440, Rentals: 1229, ActiveCustomers: 526}},
		{"store 2", window("2022-07-01", "2022-07-10", 2), domain.Totals{Revenue: 1785.64, Payments: 438, Rentals: 1291, ActiveCustomers: 533}},
		{"unknown store is zeros, not an error", window("2022-07-01", "2022-07-10", 999), domain.Totals{}},
		{"a window before any payment is zeros", window("2022-01-01", "2022-01-10"), domain.Totals{}},
		// Pagila's payments end on 2022-07-27: the last day is included, and a window after it is empty.
		{"last day is included", window("2022-07-27", "2022-07-27"), domain.Totals{Revenue: 182.60, Payments: 39}},
		{"day after the data is empty", window("2022-07-28", "2022-07-28"), domain.Totals{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := r.Totals(context.Background(), c.f)
			if err != nil {
				t.Fatal(err)
			}
			if !near(got.Revenue, c.want.Revenue) || got.Payments != c.want.Payments {
				t.Errorf("revenue/payments = %.2f/%d, want %.2f/%d", got.Revenue, got.Payments, c.want.Revenue, c.want.Payments)
			}
			// Rentals and customers are counted by rental date, so the day-edge cases only pin payments.
			if c.want.Rentals != 0 && (got.Rentals != c.want.Rentals || got.ActiveCustomers != c.want.ActiveCustomers) {
				t.Errorf("rentals/customers = %d/%d, want %d/%d", got.Rentals, got.ActiveCustomers, c.want.Rentals, c.want.ActiveCustomers)
			}
		})
	}
}

func TestTotalsStoresAddUp(t *testing.T) {
	r := repo(t)
	all, err := r.Totals(context.Background(), window("2022-07-01", "2022-07-10"))
	if err != nil {
		t.Fatal(err)
	}
	var revenue float64
	var payments, rentals int
	for _, s := range []int{1, 2} {
		part, err := r.Totals(context.Background(), window("2022-07-01", "2022-07-10", s))
		if err != nil {
			t.Fatal(err)
		}
		revenue += part.Revenue
		payments += part.Payments
		rentals += part.Rentals
	}
	if !near(revenue, all.Revenue) || payments != all.Payments || rentals != all.Rentals {
		t.Errorf("stores sum to %.2f/%d/%d, all stores say %.2f/%d/%d", revenue, payments, rentals, all.Revenue, all.Payments, all.Rentals)
	}
}

func TestRevenueByDay(t *testing.T) {
	r := repo(t)
	got, err := r.Revenue(context.Background(), window("2022-07-01", "2022-07-10"), domain.Day)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 20 { // 10 days x 2 stores, ordered by bucket then store
		t.Fatalf("%d points, want 20", len(got))
	}
	first, last := got[0], got[19]
	if first.Bucket != "2022-07-01" || first.StoreID != 1 || !near(first.Revenue, 205.51) {
		t.Errorf("first = %+v", first)
	}
	if got[1].Bucket != "2022-07-01" || got[1].StoreID != 2 || !near(got[1].Revenue, 146.59) {
		t.Errorf("second = %+v", got[1])
	}
	if last.Bucket != "2022-07-10" || last.StoreID != 2 || !near(last.Revenue, 171.58) {
		t.Errorf("last = %+v", last)
	}

	// The buckets add up to the totals for the same window.
	var sum float64
	for _, p := range got {
		sum += p.Revenue
	}
	tot, err := r.Totals(context.Background(), window("2022-07-01", "2022-07-10"))
	if err != nil {
		t.Fatal(err)
	}
	if !near(sum, tot.Revenue) {
		t.Errorf("buckets sum to %.2f, totals say %.2f", sum, tot.Revenue)
	}
}

func TestRevenueByWeekAndMonth(t *testing.T) {
	r := repo(t)
	f := window("2022-07-01", "2022-07-10")

	week, err := r.Revenue(context.Background(), f, domain.Week)
	if err != nil {
		t.Fatal(err)
	}
	// Weeks start on Monday, so a window starting on a Friday begins in the week of 06-27.
	want := []domain.RevenuePoint{
		{Bucket: "2022-06-27", StoreID: 1, Revenue: 660.46},
		{Bucket: "2022-06-27", StoreID: 2, Revenue: 456.86},
		{Bucket: "2022-07-04", StoreID: 1, Revenue: 1162.14},
		{Bucket: "2022-07-04", StoreID: 2, Revenue: 1328.78},
	}
	if len(week) != len(want) {
		t.Fatalf("weekly = %+v", week)
	}
	for i, w := range want {
		if week[i].Bucket != w.Bucket || week[i].StoreID != w.StoreID || !near(week[i].Revenue, w.Revenue) {
			t.Errorf("week[%d] = %+v, want %+v", i, week[i], w)
		}
	}

	month, err := r.Revenue(context.Background(), f, domain.Month)
	if err != nil {
		t.Fatal(err)
	}
	if len(month) != 2 || month[0].Bucket != "2022-07-01" || month[1].Bucket != "2022-07-01" {
		t.Errorf("monthly = %+v", month)
	}
}

func TestRevenueIsZeroFilled(t *testing.T) {
	r := repo(t)
	// Pagila has no payments before 2022-01-23: every bucket must still be there, at zero.
	got, err := r.Revenue(context.Background(), window("2022-01-01", "2022-01-10"), domain.Day)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 20 {
		t.Fatalf("%d points, want 20", len(got))
	}
	for _, p := range got {
		if p.Revenue != 0 {
			t.Errorf("%+v, want zero", p)
		}
	}
}

func TestRevenueForOneStore(t *testing.T) {
	r := repo(t)
	got, err := r.Revenue(context.Background(), window("2022-07-01", "2022-07-10", 1), domain.Day)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("%d points, want 10", len(got))
	}
	for _, p := range got {
		if p.StoreID != 1 {
			t.Errorf("%+v is not store 1", p)
		}
	}
}

func TestRevenueRejectsAnInvalidGranularity(t *testing.T) {
	if _, err := repo(t).Revenue(context.Background(), window("2022-07-01", "2022-07-10"), "year"); err == nil {
		t.Fatal("want an error")
	}
}

func TestTopFilms(t *testing.T) {
	r := repo(t)
	got, err := r.TopFilms(context.Background(), window("2022-06-28", "2022-07-27"), 5)
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.FilmStat{
		{FilmID: 327, Title: "FOOL MOCKINGBIRD", Rentals: 7, Revenue: 52.93},
		{FilmID: 244, Title: "DORADO NOTTING", Rentals: 8, Revenue: 51.92},
		{FilmID: 879, Title: "TELEGRAPH VOYAGE", Rentals: 6, Revenue: 49.94},
		// 69 and 127 earn the same, so the lower film id comes first.
		{FilmID: 69, Title: "BEVERLY OUTLAW", Rentals: 8, Revenue: 48.92},
		{FilmID: 127, Title: "CAT CONEHEADS", Rentals: 8, Revenue: 48.92},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i, w := range want {
		g := got[i]
		if g.FilmID != w.FilmID || g.Title != w.Title || g.Rentals != w.Rentals || !near(g.Revenue, w.Revenue) {
			t.Errorf("film[%d] = %+v, want %+v", i, g, w)
		}
	}
}

func TestTopFilmsLimitAndStore(t *testing.T) {
	r := repo(t)
	got, err := r.TopFilms(context.Background(), window("2022-06-28", "2022-07-27"), 3)
	if err != nil || len(got) != 3 {
		t.Fatalf("limit 3: %v, %v", got, err)
	}

	forStore, err := r.TopFilms(context.Background(), window("2022-06-28", "2022-07-27", 1), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(forStore) != 10 {
		t.Fatalf("%d films for store 1, want 10", len(forStore))
	}
	for i := 1; i < len(forStore); i++ {
		if forStore[i].Revenue > forStore[i-1].Revenue {
			t.Errorf("not ranked by revenue at %d: %+v", i, forStore)
		}
	}

	none, err := r.TopFilms(context.Background(), window("2022-06-28", "2022-07-27", 999), 10)
	if err != nil || len(none) != 0 {
		t.Errorf("unknown store: %v, %v", none, err)
	}
}
