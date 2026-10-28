// Package testutil opens the real Postgres (Pagila) and Redis that the
// integration tests run against. A test skips when the service is unreachable,
// and fails instead when REQUIRE_DB is set (as in CI), so a missing service
// can never pass silently.
package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/SangNguyen2810/store-operation/api/internal/platform/db"
)

const (
	DefaultDatabaseURL = "postgres://postgres:123456@localhost:5432/postgres"
	DefaultRedisURL    = "redis://localhost:6380/15"
)

// DatabaseURL is TEST_DATABASE_URL, or the local Pagila container.
func DatabaseURL() string { return env("TEST_DATABASE_URL", DefaultDatabaseURL) }

// RedisURL is TEST_REDIS_URL, or the Redis from `REDIS_PORT=6380 make redis-up`.
func RedisURL() string { return env("TEST_REDIS_URL", DefaultRedisURL) }

// Pool returns a read-only pool on Pagila, closed when the test ends.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, DatabaseURL())
	if err != nil {
		unreachable(t, "pagila", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Redis returns a client on a reachable Redis, closed when the test ends.
func Redis(t *testing.T) *redis.Client {
	t.Helper()
	opt, err := redis.ParseURL(RedisURL())
	if err != nil {
		t.Fatal(err)
	}
	c := redis.NewClient(opt)
	t.Cleanup(func() { _ = c.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		unreachable(t, "redis", err)
	}
	return c
}

// Prefix returns a key prefix unique to the test and removes its keys at the
// end, so tests do not see each other's entries or a previous run's.
func Prefix(t *testing.T, c *redis.Client) string {
	t.Helper()
	p := "test:" + t.Name() + ":"
	clean := func() {
		ctx := context.Background()
		keys, _ := c.Keys(ctx, p+"*").Result()
		if len(keys) > 0 {
			c.Del(ctx, keys...)
		}
	}
	clean()
	t.Cleanup(clean)
	return p
}

func unreachable(t *testing.T, what string, err error) {
	t.Helper()
	if os.Getenv("REQUIRE_DB") != "" {
		t.Fatalf("%s not reachable: %v", what, err)
	}
	t.Skipf("%s not reachable: %v", what, err)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
