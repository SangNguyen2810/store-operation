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

	"github.com/SangNguyen2810/store-operation/app/gateway/internal/config"
	"github.com/SangNguyen2810/store-operation/app/gateway/internal/gateway"
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

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: gateway.New(gateway.Options{
			Upstream:      cfg.Upstream,
			RatePerSec:    cfg.RatePerSec,
			Burst:         cfg.Burst,
			DialTimeout:   cfg.DialTimeout,
			HeaderTimeout: cfg.HeaderTimeout,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		// Longer than the upstream's own 15s write timeout, so its answer fits.
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("listening", "addr", cfg.Addr, "upstream", cfg.Upstream.String(), "rateLimit", cfg.RatePerSec)

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
