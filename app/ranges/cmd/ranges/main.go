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

	"github.com/SangNguyen2810/store-operation/app/ranges/internal/platform/cache"
	"github.com/SangNguyen2810/store-operation/app/ranges/internal/platform/config"
	"github.com/SangNguyen2810/store-operation/app/ranges/internal/platform/httpx"
	"github.com/SangNguyen2810/store-operation/app/ranges/internal/ranges/adapters/configstore"
	httpadapter "github.com/SangNguyen2810/store-operation/app/ranges/internal/ranges/adapters/http"
	"github.com/SangNguyen2810/store-operation/app/ranges/internal/ranges/app"
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

	var c *cache.Cache
	if cfg.CacheURL != "" {
		if c, err = cache.New(cfg.CacheURL); err != nil {
			return err
		}
		slog.Info("reading the date ranges from the redis service", "url", cfg.CacheURL)
	}
	// With no cache the source serves the built-in date ranges and never refreshes.
	source := configstore.New(c)
	source.Refresh(ctx)
	go source.Run(ctx, configstore.RefreshEvery)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           newHandler(app.NewService(source, cfg.AsOf)),
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

// newHandler wires the HTTP surface: health and the ranges route, behind request logging.
func newHandler(svc *app.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpx.Health)
	httpadapter.Register(mux, svc)
	return httpx.LogRequests(mux)
}
