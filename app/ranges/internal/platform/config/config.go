// Package config loads service settings from the environment.
package config

import (
	"fmt"
	"os"
	"time"
)

const dateLayout = "2006-01-02"

type Config struct {
	Addr string
	// AsOf is the dashboard's "today". Pagila's payments end on 2022-07-27,
	// so "last 30 days" needs a fixed anchor to show anything.
	AsOf time.Time
	// CacheURL is the redis service that holds the list, such as http://localhost:8082.
	// Empty (the default) means the built-in list is always served.
	CacheURL string
}

func Load() (Config, error) {
	cfg := Config{Addr: env("ADDR", ":8083"), CacheURL: env("CACHE_URL", "")}
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
