// Package cache is a driven adapter that puts a cache in front of another Repo.
// It satisfies the same port as the Postgres adapter, so the use case and
// handler do not know it is there. The caching mechanics live in
// platform/cache; this package decides what is cached and how keys are built.
package cache

import (
	"context"
	"time"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/platform/cache"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/app"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
)

// key keeps the name the API used when it cached stores, so entries written
// before the split are still valid. Bump the version when the stored JSON
// changes shape.
const key = "analytics:v1:stores"

// TTL is how long the store details stay cached: they change rarely.
const TTL = time.Hour

// Compile-time check that Repo satisfies the port.
var _ app.Repo = (*Repo)(nil)

type Repo struct {
	next  app.Repo
	cache *cache.Cache
}

func New(c *cache.Cache, next app.Repo) *Repo { return &Repo{next: next, cache: c} }

func (r *Repo) Stores(ctx context.Context) ([]domain.Store, error) {
	return cache.Fetch(ctx, r.cache, key, TTL, r.next.Stores)
}
