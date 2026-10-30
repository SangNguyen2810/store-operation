// Package rdb opens the Redis client used by the cache. Timeouts are short and
// retries are off, so a dead Redis costs a request a fraction of a second, not
// seconds, before it falls back to the database.
package rdb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const opTimeout = 250 * time.Millisecond

// Open parses the URL and returns a client. An unreachable Redis is only
// logged: the cache fails open, so the API must still start.
func Open(ctx context.Context, url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	opt.DialTimeout = opTimeout
	opt.ReadTimeout = opTimeout
	opt.WriteTimeout = opTimeout
	opt.MaxRetries = -1

	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("redis unreachable; requests fall back to the database until it returns", "err", err)
	}
	return client, nil
}
