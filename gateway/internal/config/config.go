// Package config loads the gateway's settings from the environment.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr string
	// Upstream is the one service requests are forwarded to.
	Upstream *url.URL
	// RatePerSec and Burst size each client IP's token bucket. RatePerSec 0
	// turns the limit off.
	RatePerSec float64
	Burst      int
	// DialTimeout bounds connecting to the upstream, HeaderTimeout waiting for
	// its response headers.
	DialTimeout   time.Duration
	HeaderTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{Addr: env("ADDR", ":8080")}

	up, err := url.Parse(env("UPSTREAM_URL", "http://localhost:8081"))
	if err != nil || (up.Scheme != "http" && up.Scheme != "https") || up.Host == "" {
		return Config{}, fmt.Errorf("UPSTREAM_URL must be an http(s) URL such as http://localhost:8081")
	}
	cfg.Upstream = up

	if cfg.RatePerSec, err = strconv.ParseFloat(env("RATE_LIMIT_RPS", "50"), 64); err != nil || cfg.RatePerSec < 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_RPS must be a number >= 0 (0 turns the limit off)")
	}
	if cfg.Burst, err = strconv.Atoi(env("RATE_LIMIT_BURST", "100")); err != nil || cfg.Burst < 1 {
		return Config{}, fmt.Errorf("RATE_LIMIT_BURST must be a whole number >= 1")
	}
	if cfg.DialTimeout, err = duration("UPSTREAM_DIAL_TIMEOUT", "2s"); err != nil {
		return Config{}, err
	}
	if cfg.HeaderTimeout, err = duration("UPSTREAM_HEADER_TIMEOUT", "10s"); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func duration(key, def string) (time.Duration, error) {
	d, err := time.ParseDuration(env(key, def))
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration such as %s", key, def)
	}
	return d, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
