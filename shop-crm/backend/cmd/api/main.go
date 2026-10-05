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

	"github.com/sap-studies/shop-crm/backend/internal/app"
	"github.com/sap-studies/shop-crm/backend/internal/config"
	"github.com/sap-studies/shop-crm/backend/internal/db"
	"github.com/sap-studies/shop-crm/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Migrations run on every boot and are idempotent, which is what lets
	// `docker compose up` bring the whole app up from an empty volume.
	if err := migrations.Up(cfg.DatabaseURL); err != nil {
		return err
	}
	log.Info("migrations applied")

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Build(pool, cfg, log),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      cfg.HTTPTimeout,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api listening", "port", cfg.Port, "cashback", cfg.BonusCashbackPercent, "spend_limit", cfg.BonusSpendLimitPct)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}