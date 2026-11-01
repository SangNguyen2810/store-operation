// Package cache is a client for the redis service with an in-process copy.
// It knows nothing about any context; an adapter decides what to read and
// builds the keys.
//
// When the redis service is unavailable Get keeps answering from a small copy
// of recent entries (served only until their own expiry). The service is not
// asked again for a few seconds after a failure, so an outage costs requests
// nothing extra.
package cache

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"
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
