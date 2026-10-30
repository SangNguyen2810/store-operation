// Package cache is a read-through cache (cache-aside) over the redis service:
// look there first, and only on a miss run the real query and store its answer
// for a while. It knows nothing about any context; a context's adapter decides
// what to cache and builds the keys.
//
// When the redis service is unavailable the cache keeps working from a small
// in-process copy of recent entries (served only until their own expiry), and
// past that the caller's loader hits the database. The service is not asked
// again for a few seconds after a failure, so an outage costs requests nothing
// extra.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	requestTimeout = 500 * time.Millisecond
	// retryAfter is how long the redis service is left alone after a failure.
	retryAfter = 5 * time.Second
	// defaultTTL applies to entries that carry no expiry of their own.
	defaultTTL = 10 * time.Minute
)

// Cache holds the redis service client, the fallback copy and the in-flight lookups.
type Cache struct {
	remote *remote
	mem    *memory
	now    func() time.Time
	// downUntil is a Unix nano timestamp; until then the service is not called.
	downUntil atomic.Int64
	sf        singleflight.Group
}

// New returns a cache over the redis service at baseURL.
func New(baseURL string) (*Cache, error) {
	r, err := newRemote(baseURL, requestTimeout)
	if err != nil {
		return nil, err
	}
	return newCache(r, time.Now), nil
}

func newCache(r *remote, now func() time.Time) *Cache {
	return &Cache{remote: r, mem: newMemory(now), now: now}
}

func (c *Cache) available() bool { return c.now().UnixNano() >= c.downUntil.Load() }

func (c *Cache) markDown(ctx context.Context, op, key string, err error) {
	if c.downUntil.Swap(c.now().Add(retryAfter).UnixNano()) < c.now().UnixNano() {
		slog.WarnContext(ctx, "cache: redis service unavailable, using the in-memory copy", "op", op, "key", key, "err", err)
	}
}

// Outcome says where a Get was answered, or why it was not.
type Outcome int

const (
	// Miss: the redis service is healthy and has no such key.
	Miss Outcome = iota
	// Hit: the redis service had it.
	Hit
	// Fallback: the redis service is unavailable and the in-memory copy had it.
	Fallback
	// Unavailable: the redis service is unavailable and there is no copy.
	Unavailable
)

// Get returns the cached bytes for key and where they came from.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, Outcome) {
	if c.available() {
		v, ttl, err := c.remote.get(ctx, key)
		switch {
		case err == nil:
			if ttl == 0 {
				ttl = defaultTTL
			}
			c.mem.set(key, v, ttl)
			return v, Hit
		case errors.Is(err, errMiss):
			// The service is healthy and says the key is gone; a stale copy must not outlive it.
			c.mem.delete(key)
			return nil, Miss
		default:
			c.markDown(ctx, "get", key, err)
		}
	}
	if v, ok := c.mem.get(key); ok {
		return v, Fallback
	}
	return nil, Unavailable
}

// Put stores val for ttl in the redis service and in the in-memory copy.
func (c *Cache) Put(ctx context.Context, key string, val []byte, ttl time.Duration) {
	c.mem.set(key, val, ttl)
	if !c.available() {
		return
	}
	if err := c.remote.put(ctx, key, val, ttl); err != nil {
		c.markDown(ctx, "put", key, err)
	}
}

type result[T any] struct {
	val T
	hit string
}

// Fetch returns the cached value for key, or runs load and caches its result for ttl.
// Callers asking for the same key at the same time share one lookup, so a cold
// key causes one query however many requests arrive together. Errors from load
// are returned and never cached; cache errors never fail the call.
func Fetch[T any](ctx context.Context, c *Cache, key string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	var zero T
	ch := c.sf.DoChan(key, func() (any, error) {
		// Every waiter shares this work, so one client hanging up must not
		// cancel it. The database statement timeout and the cache timeouts bound it.
		ctx := context.WithoutCancel(ctx)
		if b, out := c.Get(ctx, key); out == Hit || out == Fallback {
			var val T
			if err := json.Unmarshal(b, &val); err == nil {
				if out == Fallback {
					return result[T]{val: val, hit: "fallback"}, nil
				}
				return result[T]{val: val, hit: "hit"}, nil
			}
			slog.WarnContext(ctx, "cache: discarding unreadable entry", "key", key)
		}
		val, err := load(ctx)
		if err != nil {
			return nil, err
		}
		if b, err := json.Marshal(val); err != nil {
			slog.WarnContext(ctx, "cache: cannot encode value", "key", key, "err", err)
		} else {
			c.Put(ctx, key, b, ttl)
		}
		return result[T]{val: val, hit: "miss"}, nil
	})

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return zero, res.Err
		}
		out := res.Val.(result[T])
		outcome := out.hit
		if res.Shared {
			outcome = "coalesced"
		}
		slog.InfoContext(ctx, "cache", "outcome", outcome, "key", key)
		return out.val, nil
	}
}
