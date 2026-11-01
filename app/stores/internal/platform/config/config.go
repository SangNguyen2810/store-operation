// Package config loads service settings from the environment.
package config

import "os"

type Config struct {
	Addr        string
	DatabaseURL string
	// CacheURL is the redis service the cache talks to, such as http://localhost:8082.
	// Empty (the default) means no cache.
	CacheURL string
}

func Load() (Config, error) {
	return Config{
		Addr:        env("ADDR", ":8081"),
		DatabaseURL: env("DATABASE_URL", "postgres://postgres:123456@localhost:5432/postgres"),
		CacheURL:    env("CACHE_URL", ""),
	}, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
