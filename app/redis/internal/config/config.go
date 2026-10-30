// Package config loads the redis service's settings from the environment.
package config

import "os"

type Config struct {
	Addr     string
	RedisURL string
}

func Load() Config {
	return Config{
		Addr:     env("ADDR", ":8082"),
		RedisURL: env("REDIS_URL", "redis://localhost:6379"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
