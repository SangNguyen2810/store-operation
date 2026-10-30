// Package cache is a read-through cache over Redis (cache-aside): look in
// Redis first, and only on a miss run the real query and store its answer for
// a while. It knows nothing about any context; a context's adapter decides what
// to cache and builds the keys.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Cache holds the Redis client, the TTL, and the in-flight lookups.
type Cache struct {
	rdb redis.Cmdable
	ttl time.Duration
	sf  singleflight.Group
}

func New(rdb redis.Cmdable, ttl time.Duration) *Cache {
	return &Cache{rdb: rdb, ttl: ttl}
}

type result[T any] struct {
	val T
	hit bool
}

// Fetch returns the cached value for key, or runs load and caches its result.
// Callers asking for the same key at the same time share one lookup, so a cold
// key causes one query however many requests arrive together. Errors from load
// are returned and never cached; Redis errors never fail the call.
func Fetch[T any](ctx context.Context, c *Cache, key string, load func(context.Context) (T, error)) (T, error) {
	var zero T
	ch := c.sf.DoChan(key, func() (any, error) {
		// Every waiter shares this work, so one client hanging up must not
		// cancel it. The database statement timeout and the Redis timeouts bound it.
		ctx := context.WithoutCancel(ctx)
		if val, ok := get[T](ctx, c.rdb, key); ok {
			return result[T]{val: val, hit: true}, nil
		}
		val, err := load(ctx)
		if err != nil {
			return nil, err
		}
		put(ctx, c.rdb, key, val, c.ttl)
		return result[T]{val: val}, nil
	})

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return zero, res.Err
		}
		out := res.Val.(result[T])
		slog.InfoContext(ctx, "cache", "outcome", outcome(out.hit, res.Shared), "key", key)
		return out.val, nil
	}
}

func outcome(hit, shared bool) string {
	switch {
	case shared:
		return "coalesced"
	case hit:
		return "hit"
	default:
		return "miss"
	}
}

func get[T any](ctx context.Context, rdb redis.Cmdable, key string) (T, bool) {
	var zero T
	b, err := rdb.Get(ctx, key).Bytes()
	switch {
	case err == nil:
		var val T
		if err := json.Unmarshal(b, &val); err != nil {
			slog.WarnContext(ctx, "cache: discarding unreadable entry", "key", key, "err", err)
			return zero, false
		}
		return val, true
	case errors.Is(err, redis.Nil):
	default:
		slog.WarnContext(ctx, "cache: redis get failed", "key", key, "err", err)
	}
	return zero, false
}

func put[T any](ctx context.Context, rdb redis.Cmdable, key string, val T, ttl time.Duration) {
	b, err := json.Marshal(val)
	if err != nil {
		slog.WarnContext(ctx, "cache: cannot encode value", "key", key, "err", err)
		return
	}
	if err := rdb.Set(ctx, key, b, ttl).Err(); err != nil {
		slog.WarnContext(ctx, "cache: redis set failed", "key", key, "err", err)
	}
}
