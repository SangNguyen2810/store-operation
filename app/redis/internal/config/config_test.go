package config

import "testing"

func TestDefaultsAndOverrides(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("REDIS_URL", "")
	if cfg := Load(); cfg.Addr != ":8082" || cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("defaults = %+v", cfg)
	}
	t.Setenv("ADDR", ":9000")
	t.Setenv("REDIS_URL", "redis://redis:6379/2")
	if cfg := Load(); cfg.Addr != ":9000" || cfg.RedisURL != "redis://redis:6379/2" {
		t.Errorf("overrides = %+v", cfg)
	}
}
