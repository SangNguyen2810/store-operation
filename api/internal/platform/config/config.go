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
}

func Load() (Config, error) {
	cfg := Config{
		Addr:        env("ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://postgres:123456@localhost:5432/postgres"),
	}
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
