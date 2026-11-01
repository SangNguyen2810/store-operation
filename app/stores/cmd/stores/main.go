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

	"github.com/SangNguyen2810/store-operation/app/stores/internal/platform/cache"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/platform/config"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/platform/db"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/platform/httpx"
	cacheadapter "github.com/SangNguyen2810/store-operation/app/stores/internal/stores/adapters/cache"
	httpadapter "github.com/SangNguyen2810/store-operation/app/stores/internal/stores/adapters/http"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/adapters/postgres"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/app"
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
	if cfg.CacheURL != "" {
		c, err := cache.New(cfg.CacheURL)
		if err != nil {
			return err
		}
		repo = cacheadapter.New(c, repo)
		slog.Info("cache enabled", "url", cfg.CacheURL, "ttl", cacheadapter.TTL)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           newHandler(pool, app.NewService(repo)),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("listening", "addr", cfg.Addr)

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

// newHandler wires the HTTP surface: health and the stores route, behind request logging.
func newHandler(db httpx.Pinger, svc *app.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpx.Health(db))
	httpadapter.Register(mux, svc)
	return httpx.LogRequests(mux)
}
