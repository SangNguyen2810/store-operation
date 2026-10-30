package config

import (
	"strings"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	for _, k := range []string{"ADDR", "UPSTREAM_URL", "RATE_LIMIT_RPS", "RATE_LIMIT_BURST", "UPSTREAM_DIAL_TIMEOUT", "UPSTREAM_HEADER_TIMEOUT"} {
		t.Setenv(k, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8080" || cfg.Upstream.String() != "http://localhost:8081" || cfg.RatePerSec != 50 || cfg.Burst != 100 ||
		cfg.DialTimeout != 2*time.Second || cfg.HeaderTimeout != 10*time.Second {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestOverrides(t *testing.T) {
	t.Setenv("ADDR", ":9000")
	t.Setenv("UPSTREAM_URL", "http://api:8081")
	t.Setenv("RATE_LIMIT_RPS", "0")
	t.Setenv("RATE_LIMIT_BURST", "5")
	t.Setenv("UPSTREAM_DIAL_TIMEOUT", "500ms")
	t.Setenv("UPSTREAM_HEADER_TIMEOUT", "3s")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9000" || cfg.Upstream.Host != "api:8081" || cfg.RatePerSec != 0 || cfg.Burst != 5 ||
		cfg.DialTimeout != 500*time.Millisecond || cfg.HeaderTimeout != 3*time.Second {
		t.Errorf("config = %+v", cfg)
	}
}

func TestRejectsBadValues(t *testing.T) {
	for key, val := range map[string]string{
		"UPSTREAM_URL":            "localhost:8081",
		"RATE_LIMIT_RPS":          "-1",
		"RATE_LIMIT_BURST":        "0",
		"UPSTREAM_DIAL_TIMEOUT":   "fast",
		"UPSTREAM_HEADER_TIMEOUT": "0s",
	} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, val)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), key) {
				t.Errorf("err = %v, want it to name %s", err, key)
			}
		})
	}
}
