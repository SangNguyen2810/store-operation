// Package store is the service's only view of Redis: bytes in, bytes out, with
// an optional expiry. It knows nothing about what the bytes mean.
package store

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// opTimeout bounds every Redis call. With retries off, a dead Redis costs a
// caller a fraction of a second, not seconds.
const opTimeout = 250 * time.Millisecond

// Stats counts what the service has done since it started.
type Stats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Sets   int64 `json:"sets"`
	Errors int64 `json:"errors"`
}

type Store struct {
	rdb                        *redis.Client
	hits, misses, sets, errors atomic.Int64
}

func New(rdb *redis.Client) *Store { return &Store{rdb: rdb} }

// Open connects with short timeouts and no retries. It does not fail when
// Redis is unreachable: the service starts anyway and reports it on /healthz.
func Open(ctx context.Context, url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("REDIS_URL: %w", err)
	}
	opt.DialTimeout, opt.ReadTimeout, opt.WriteTimeout = opTimeout, opTimeout, opTimeout
	opt.MaxRetries = -1
	return redis.NewClient(opt), nil
}

// Get returns the value and how long it has left. ttl is 0 for a key that never expires.
func (s *Store) Get(ctx context.Context, key string) (val []byte, ttl time.Duration, found bool, err error) {
	pipe := s.rdb.Pipeline()
	get := pipe.Get(ctx, key)
	pttl := pipe.PTTL(ctx, key)
	_, execErr := pipe.Exec(ctx)

	if errors.Is(get.Err(), redis.Nil) {
		s.misses.Add(1)
		return nil, 0, false, nil
	}
	if execErr != nil {
		s.errors.Add(1)
		return nil, 0, false, execErr
	}
	val, _ = get.Bytes()
	ttl, expired := remaining(pttl.Val())
	if expired {
		s.misses.Add(1)
		return nil, 0, false, nil
	}
	s.hits.Add(1)
	return val, ttl, true, nil
}

// remaining reads Redis's answer to PTTL: -1 is a key that never expires (ttl 0)
// and -2 is a key that expired between our GET and our PTTL, which is a miss.
func remaining(pttl time.Duration) (ttl time.Duration, expired bool) {
	switch {
	case pttl == time.Duration(-2):
		return 0, true
	case pttl > 0:
		return pttl, false
	}
	return 0, false
}

// Set stores val. A ttl of 0 means the key never expires.
func (s *Store) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	if err := s.rdb.Set(ctx, key, val, ttl).Err(); err != nil {
		s.errors.Add(1)
		return err
	}
	s.sets.Add(1)
	return nil
}

func (s *Store) Ping(ctx context.Context) error { return s.rdb.Ping(ctx).Err() }

func (s *Store) Stats() Stats {
	return Stats{Hits: s.hits.Load(), Misses: s.misses.Load(), Sets: s.sets.Load(), Errors: s.errors.Load()}
}
