// Package cache is a driven adapter that puts a cache in front of another Repo.
// It satisfies the same port as the Postgres adapter, so the use cases and
// handlers do not know it is there. The caching mechanics live in
// platform/cache; this package decides what is cached and how keys are built.
package cache

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/app"
	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
	platformcache "github.com/SangNguyen2810/store-operation/app/api/internal/platform/cache"
)

// version is part of every key. Bump it when the stored JSON changes shape, so
// old entries are ignored instead of misread.
const version = "v1"

// Compile-time check that Repo satisfies the port.
var _ app.Repo = (*Repo)(nil)

// storesTTL is longer than the query TTL: store details change rarely.
const storesTTL = time.Hour

// Repo caches every read. Query results live for ttl, store details for an hour.
type Repo struct {
	next  app.Repo
	cache *platformcache.Cache
	ttl   time.Duration
}

func New(c *platformcache.Cache, next app.Repo, ttl time.Duration) *Repo {
	return &Repo{next: next, cache: c, ttl: ttl}
}

func (r *Repo) Stores(ctx context.Context) ([]domain.Store, error) {
	return platformcache.Fetch(ctx, r.cache, strings.Join([]string{"analytics", version, "stores"}, ":"), storesTTL, r.next.Stores)
}

func (r *Repo) Totals(ctx context.Context, f domain.Filter) (domain.Totals, error) {
	return platformcache.Fetch(ctx, r.cache, key("totals", f), r.ttl, func(ctx context.Context) (domain.Totals, error) {
		return r.next.Totals(ctx, f)
	})
}

func (r *Repo) Revenue(ctx context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error) {
	return platformcache.Fetch(ctx, r.cache, key("revenue", f, string(g)), r.ttl, func(ctx context.Context) ([]domain.RevenuePoint, error) {
		return r.next.Revenue(ctx, f, g)
	})
}

func (r *Repo) TopFilms(ctx context.Context, f domain.Filter, limit int) ([]domain.FilmStat, error) {
	return platformcache.Fetch(ctx, r.cache, key("films", f, strconv.Itoa(limit)), r.ttl, func(ctx context.Context) ([]domain.FilmStat, error) {
		return r.next.TopFilms(ctx, f, limit)
	})
}

// key is deterministic and unique per query: the same window, store and
// options always map to the same key, and any difference maps to another.
func key(method string, f domain.Filter, extra ...string) string {
	store := "all"
	if f.StoreID != nil {
		store = strconv.Itoa(*f.StoreID)
	}
	parts := []string{"analytics", version, method, f.From.UTC().Format(time.RFC3339), f.To.UTC().Format(time.RFC3339), store}
	return strings.Join(append(parts, extra...), ":")
}
