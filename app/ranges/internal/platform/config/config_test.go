package config

import (
	"strings"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ADDR", "CACHE_URL", "ASOF_DATE"} {
		t.Setenv(k, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8083" || cfg.CacheURL != "" || cfg.AsOf.Format(dateLayout) != "2022-07-27" {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestLoadFromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("ADDR", ":9000")
	t.Setenv("CACHE_URL", "http://localhost:8082")
	t.Setenv("ASOF_DATE", "2022-01-02")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9000" || cfg.CacheURL != "http://localhost:8082" || cfg.AsOf.Format(dateLayout) != "2022-01-02" {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestLoadRejectsABadDate(t *testing.T) {
	clearEnv(t)
	t.Setenv("ASOF_DATE", "yesterday")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ASOF_DATE") {
		t.Errorf("err = %v, want it to name ASOF_DATE", err)
	}
}
