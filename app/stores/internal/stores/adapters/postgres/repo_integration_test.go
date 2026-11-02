//go:build integration

// These tests run the real SQL against Pagila. They need DATABASE_URL and fail
// without it: `make test-integration` locally, the go-integration job in CI.
// Expected figures were taken from Pagila at the commit pinned in
// .github/actions/load-pagila, using plain SQL written independently of repo.go.
package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/platform/db"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/adapters/postgres"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
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

// Pagila's store table has hundreds of rows, but only stores 1 and 2 stock any inventory,
// and only those are shops the dashboard can report on.
func TestStoresListsOnlyStoresWithInventory(t *testing.T) {
	got, err := repo(t).Stores(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Store{
		{
			ID: 1, Label: "Boksburg, South Africa", Manager: "Warner Hudson", Phone: "490211944645",
			Address: "569 Baicheng Lane", District: "Gauteng", PostalCode: "60304", City: "Boksburg", Country: "South Africa",
			InventoryItems: 2270, ActiveCustomers: 326,
		},
		{
			ID: 2, Label: "Hamilton, New Zealand", Manager: "Lavone O'Reilly", Phone: "657282285970",
			Address: "478 Joliet Way", District: "Hamilton", PostalCode: "77948", City: "Hamilton", Country: "New Zealand",
			InventoryItems: 2311, ActiveCustomers: 273,
		},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d stores %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("store[%d] = %+v\nwant     %+v", i, got[i], want[i])
		}
	}
}
