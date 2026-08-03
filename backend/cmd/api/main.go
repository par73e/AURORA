package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aurora/backend/internal/config"
	"aurora/backend/internal/database"
	"aurora/backend/internal/httpapi"
	"aurora/backend/internal/moon"
	"aurora/backend/internal/orbit"
	"aurora/backend/internal/syncer"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	repository := orbit.NewRepository(pool)
	moonRepository := moon.NewRepository(pool)
	dataSyncer := syncer.New(repository)
	initialSyncContext, cancelInitialSync := context.WithTimeout(ctx, 45*time.Second)
	if err := dataSyncer.SyncCelesTrak(initialSyncContext); err != nil {
		slog.Warn("CelesTrak startup sync failed; cached data remains available", "error", err)
	}
	if err := dataSyncer.SyncLaunches(initialSyncContext); err != nil {
		slog.Warn("Launch Library startup sync failed; cached data remains available", "error", err)
	}
	cancelInitialSync()

	go schedule(ctx, 2*time.Hour, dataSyncer.SyncCelesTrak)
	go schedule(ctx, 30*time.Minute, dataSyncer.SyncLaunches)

	server := &http.Server{Addr: ":" + cfg.Port, Handler: httpapi.Router(repository, moonRepository), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		slog.Info("AURORA API started", "address", "http://localhost:"+cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server stopped unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = server.Shutdown(shutdownContext)
}

func schedule(ctx context.Context, interval time.Duration, run func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runContext, cancel := context.WithTimeout(ctx, 45*time.Second)
			if err := run(runContext); err != nil {
				slog.Warn("scheduled data sync failed", "error", err)
			}
			cancel()
		}
	}
}
