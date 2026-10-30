package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/adapters/cache"
	httpadapter "github.com/SangNguyen2810/store-operation/app/api/internal/analytics/adapters/http"
	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/adapters/postgres"
	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/adapters/ranges"
	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/app"
	platformcache "github.com/SangNguyen2810/store-operation/app/api/internal/platform/cache"
	"github.com/SangNguyen2810/store-operation/app/api/internal/platform/config"
	"github.com/SangNguyen2810/store-operation/app/api/internal/platform/db"
	"github.com/SangNguyen2810/store-operation/app/api/internal/platform/httpx"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	var repo app.Repo = postgres.New(pool)
	var c *platformcache.Cache
	if cfg.CacheURL != "" {
		if c, err = platformcache.New(cfg.CacheURL); err != nil {
			return err
		}
		repo = cache.New(c, repo, cfg.CacheTTL)
		slog.Info("cache enabled", "url", cfg.CacheURL, "ttl", cfg.CacheTTL)
	}
	// With no cache the source serves the built-in date ranges and never refreshes.
	rangeSource := ranges.New(c)
	rangeSource.Refresh(ctx)
	go rangeSource.Run(ctx, ranges.RefreshEvery)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           newHandler(pool, app.NewService(repo, rangeSource, cfg.AsOf), cfg.AsOf),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("listening", "addr", cfg.Addr, "asOf", cfg.AsOf.Format("2006-01-02"))

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// newHandler wires the HTTP surface: health, meta and the analytics routes,
// behind request logging.
func newHandler(db httpx.Pinger, svc *app.Service, asOf time.Time) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpx.Health(db))
	mux.HandleFunc("GET /api/meta", httpx.Meta(asOf))
	httpadapter.Register(mux, svc)
	return httpx.LogRequests(mux)
}
