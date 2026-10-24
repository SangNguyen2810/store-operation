package postgres_test

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SangNguyen2810/store-operation/api/internal/analytics/adapters/postgres"
	"github.com/SangNguyen2810/store-operation/api/internal/analytics/domain"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/db"
)

// These tests read the live Pagila database (read-only) and skip when it is unreachable (fail if REQUIRE_DB is set, as in CI).
// Expected values come from independent SQL: the sales_by_store view and raw tables.

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:123456@localhost:5432/postgres"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, url)
	if err != nil {
		if os.Getenv("REQUIRE_DB") != "" {
			t.Fatalf("pagila not reachable: %v", err)
		}
		t.Skipf("pagila not reachable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func d(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.005 }

var fullRange = domain.Filter{From: d("2022-01-01"), To: d("2022-12-31")}

func TestPoolIsReadOnly(t *testing.T) {
	pool := testPool(t)
	_, err := pool.Exec(context.Background(), `UPDATE film SET title = title WHERE film_id = 1`)
	if err == nil {
		t.Fatal("write succeeded; the pool must be read-only")
	}
}

func TestStores(t *testing.T) {
	repo := postgres.New(testPool(t))
	stores, err := repo.Stores(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 2 || stores[0].ID != 1 || stores[1].ID != 2 {
		t.Fatalf("got %+v, want stores 1 and 2", stores)
	}
}

func TestTotalsMatchSalesByStore(t *testing.T) {
	pool := testPool(t)
	repo := postgres.New(pool)
	ctx := context.Background()

	var viewTotal float64
	if err := pool.QueryRow(ctx, `SELECT sum(total_sales)::float8 FROM sales_by_store`).Scan(&viewTotal); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Totals(ctx, fullRange)
	if err != nil {
		t.Fatal(err)
	}
	if !near(got.Revenue, viewTotal) {
		t.Errorf("revenue %.2f, sales_by_store says %.2f", got.Revenue, viewTotal)
	}

	store := 1
	var storeTotal float64
	if err := pool.QueryRow(ctx, `SELECT total_sales::float8 FROM sales_by_store WHERE store LIKE 'Boksburg%'`).Scan(&storeTotal); err != nil {
		t.Fatal(err)
	}
	f := fullRange
	f.StoreID = &store
	got, err = repo.Totals(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if !near(got.Revenue, storeTotal) {
		t.Errorf("store 1 revenue %.2f, want %.2f", got.Revenue, storeTotal)
	}
}

func TestTotalsWindowMatchesRawSQL(t *testing.T) {
	pool := testPool(t)
	f := domain.Filter{From: d("2022-06-28"), To: d("2022-07-27")}
	var rev float64
	var payments, rentals, customers int
	ctx := context.Background()
	if err := pool.QueryRow(ctx,
		`SELECT sum(amount)::float8, count(*) FROM payment WHERE payment_date >= '2022-06-28' AND payment_date < '2022-07-28'`).
		Scan(&rev, &payments); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*), count(DISTINCT customer_id) FROM rental WHERE rental_date >= '2022-06-28' AND rental_date < '2022-07-28'`).
		Scan(&rentals, &customers); err != nil {
		t.Fatal(err)
	}
	got, err := postgres.New(pool).Totals(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if !near(got.Revenue, rev) || got.Payments != payments || got.Rentals != rentals || got.ActiveCustomers != customers {
		t.Errorf("got %+v, want revenue %.2f payments %d rentals %d customers %d", got, rev, payments, rentals, customers)
	}
}

func TestRevenueSeries(t *testing.T) {
	repo := postgres.New(testPool(t))
	ctx := context.Background()
	f := domain.Filter{From: d("2022-03-01"), To: d("2022-03-31")}

	totals, err := repo.Totals(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []domain.Granularity{domain.Day, domain.Week, domain.Month} {
		pts, err := repo.Revenue(ctx, f, g)
		if err != nil {
			t.Fatalf("%s: %v", g, err)
		}
		var sum float64
		for _, p := range pts {
			sum += p.Revenue
		}
		if !near(sum, totals.Revenue) {
			t.Errorf("%s: series sums to %.2f, totals say %.2f", g, sum, totals.Revenue)
		}
	}

	days, _ := repo.Revenue(ctx, f, domain.Day)
	if len(days) != 31*2 {
		t.Errorf("daily series has %d rows, want 62 (31 days x 2 stores, zero-filled)", len(days))
	}

	// A window with no payments is still fully zero-filled.
	empty, err := repo.Revenue(ctx, domain.Filter{From: d("2023-01-01"), To: d("2023-01-03")}, domain.Day)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 6 {
		t.Errorf("empty window has %d rows, want 6", len(empty))
	}
	for _, p := range empty {
		if p.Revenue != 0 {
			t.Errorf("expected zero revenue, got %+v", p)
		}
	}

	store := 2
	one, err := repo.Revenue(ctx, domain.Filter{From: f.From, To: f.To, StoreID: &store}, domain.Day)
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 31 {
		t.Errorf("single store daily series has %d rows, want 31", len(one))
	}
}

func TestRevenueRejectsBadGranularity(t *testing.T) {
	_, err := postgres.New(testPool(t)).Revenue(context.Background(), fullRange, "year")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTopFilms(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	films, err := postgres.New(pool).TopFilms(ctx, fullRange, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(films) != 5 {
		t.Fatalf("got %d films, want 5", len(films))
	}
	for i := 1; i < len(films); i++ {
		if films[i].Revenue > films[i-1].Revenue {
			t.Errorf("not sorted by revenue desc: %+v", films)
		}
	}
	var rev float64
	err = pool.QueryRow(ctx, `
		SELECT sum(p.amount)::float8 FROM payment p, rental r, inventory i
		WHERE p.rental_id = r.rental_id AND r.inventory_id = i.inventory_id AND i.film_id = $1`, films[0].FilmID).Scan(&rev)
	if err != nil {
		t.Fatal(err)
	}
	if !near(films[0].Revenue, rev) {
		t.Errorf("top film revenue %.2f, raw SQL says %.2f", films[0].Revenue, rev)
	}
}
