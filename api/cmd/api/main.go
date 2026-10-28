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

	"github.com/SangNguyen2810/store-operation/api/internal/analytics/adapters/cache"
	httpadapter "github.com/SangNguyen2810/store-operation/api/internal/analytics/adapters/http"
	"github.com/SangNguyen2810/store-operation/api/internal/analytics/adapters/postgres"
	"github.com/SangNguyen2810/store-operation/api/internal/analytics/app"
	platformcache "github.com/SangNguyen2810/store-operation/api/internal/platform/cache"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/config"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/db"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/httpx"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/rdb"
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
	if cfg.RedisURL != "" {
		rc, err := rdb.Open(ctx, cfg.RedisURL)
		if err != nil {
			return err
		}
		defer rc.Close()
		repo = cache.New(platformcache.New(rc, cfg.CacheTTL), repo)
		slog.Info("cache enabled", "ttl", cfg.CacheTTL)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           newHandler(pool, repo, cfg.AsOf),
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
func newHandler(db httpx.Pinger, repo app.Repo, asOf time.Time) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpx.Health(db))
	mux.HandleFunc("GET /api/meta", httpx.Meta(asOf))
	httpadapter.Register(mux, app.NewService(repo), asOf)
	return httpx.LogRequests(mux)
}
