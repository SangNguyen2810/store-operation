package config

import "testing"

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ADDR", "DATABASE_URL", "CACHE_URL"} {
		t.Setenv(k, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8081" || cfg.CacheURL != "" || cfg.DatabaseURL == "" {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestLoadFromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("ADDR", ":9000")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("CACHE_URL", "http://localhost:8082")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9000" || cfg.DatabaseURL != "postgres://x" || cfg.CacheURL != "http://localhost:8082" {
		t.Errorf("cfg = %+v", cfg)
	}
}
