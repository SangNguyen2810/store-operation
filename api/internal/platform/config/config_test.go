package config

import (
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ADDR", "DATABASE_URL", "REDIS_URL", "CACHE_TTL", "ASOF_DATE"} {
		t.Setenv(k, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8081" || cfg.RedisURL != "" || cfg.CacheTTL != 10*time.Minute {
		t.Errorf("defaults = %+v", cfg)
	}
	if got := cfg.AsOf.Format(dateLayout); got != "2022-07-27" {
		t.Errorf("asOf = %s", got)
	}
	if cfg.DatabaseURL == "" {
		t.Error("database url should have a default")
	}
}

func TestLoadFromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("ADDR", ":9000")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("REDIS_URL", "redis://localhost:6380/0")
	t.Setenv("CACHE_TTL", "90s")
	t.Setenv("ASOF_DATE", "2022-01-02")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9000" || cfg.DatabaseURL != "postgres://x" || cfg.RedisURL != "redis://localhost:6380/0" || cfg.CacheTTL != 90*time.Second {
		t.Errorf("cfg = %+v", cfg)
	}
	if got := cfg.AsOf.Format(dateLayout); got != "2022-01-02" {
		t.Errorf("asOf = %s", got)
	}
}

func TestLoadRejectsBadValues(t *testing.T) {
	for name, kv := range map[string][2]string{
		"ttl not a duration": {"CACHE_TTL", "soon"},
		"ttl zero":           {"CACHE_TTL", "0s"},
		"ttl negative":       {"CACHE_TTL", "-5m"},
		"bad as-of date":     {"ASOF_DATE", "27/07/2022"},
	} {
		t.Run(name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(kv[0], kv[1])
			if _, err := Load(); err == nil {
				t.Error("want an error")
			}
		})
	}
}
