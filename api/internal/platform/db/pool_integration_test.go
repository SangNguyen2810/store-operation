package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/api/internal/platform/db"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/testutil"
)

func TestOpenSetsSafeSessionDefaults(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	for setting, want := range map[string]string{
		"default_transaction_read_only": "on",
		"TimeZone":                      "UTC",
		"statement_timeout":             "10s",
	} {
		var got string
		if err := pool.QueryRow(ctx, `SELECT current_setting($1)`, setting).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s = %q, want %q", setting, got, want)
		}
	}
}

func TestOpenRejectsBadURL(t *testing.T) {
	if _, err := db.Open(context.Background(), "://nope"); err == nil {
		t.Error("want an error for an unparsable url")
	}
}

func TestOpenFailsWhenUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := db.Open(ctx, "postgres://u:p@127.0.0.1:1/x?connect_timeout=1"); err == nil {
		t.Error("want an error when nothing listens on the port")
	}
}
