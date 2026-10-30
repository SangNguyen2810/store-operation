// Package config loads service settings from the environment.
package config

import (
	"fmt"
	"os"
	"time"
)

const dateLayout = "2006-01-02"

type Config struct {
	Addr        string
	DatabaseURL string
	// AsOf is the dashboard's "today". Pagila's payments end on 2022-07-27,
	// so "last 30 days" needs a fixed anchor to show anything.
	AsOf time.Time
	// CacheURL is the redis service the cache talks to, such as http://localhost:8082.
	// Empty (the default) means no cache.
	CacheURL string
	// CacheTTL is how long a cached query result stays valid.
	CacheTTL time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Addr:        env("ADDR", ":8081"),
		DatabaseURL: env("DATABASE_URL", "postgres://postgres:123456@localhost:5432/postgres"),
		CacheURL:    env("CACHE_URL", ""),
	}
	ttl, err := time.ParseDuration(env("CACHE_TTL", "10m"))
	if err != nil || ttl <= 0 {
		return Config{}, fmt.Errorf("CACHE_TTL must be a positive duration such as 10m")
	}
	cfg.CacheTTL = ttl
	asOf, err := time.Parse(dateLayout, env("ASOF_DATE", "2022-07-27"))
	if err != nil {
		return Config{}, fmt.Errorf("ASOF_DATE: %w", err)
	}
	cfg.AsOf = asOf
	return cfg, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
