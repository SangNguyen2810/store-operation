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

	"github.com/SangNguyen2810/store-operation/api/internal/analytics/adapters/postgres"
	"github.com/SangNguyen2810/store-operation/api/internal/analytics/app"
	"github.com/SangNguyen2810/store-operation/api/internal/config"
	"github.com/SangNguyen2810/store-operation/api/internal/db"
	"github.com/SangNguyen2810/store-operation/api/internal/httpapi"
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

	svc := app.NewService(postgres.New(pool))
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(svc, pool, cfg.AsOf),
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
