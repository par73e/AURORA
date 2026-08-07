package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"aurora/backend/internal/config"
	"aurora/backend/internal/database"
	"aurora/backend/internal/httpapi"
	"aurora/backend/internal/mars"
	"aurora/backend/internal/moon"
	"aurora/backend/internal/orbit"
	"aurora/backend/internal/syncer"
	"aurora/backend/internal/voyage"
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
	marsRepository := mars.NewRepository(pool)
	voyageRepository := voyage.NewRepository(pool)
	dataSyncer := syncer.NewWithMoonVoyageMars(repository, moonRepository, marsRepository, voyageRepository)
	initialSyncContext, cancelInitialSync := context.WithTimeout(ctx, 120*time.Second)
	if err := dataSyncer.SyncCelesTrak(initialSyncContext); err != nil {
		slog.Warn("CelesTrak startup sync failed; cached data remains available", "error", err)
	}
	if err := dataSyncer.SyncLaunches(initialSyncContext); err != nil {
		slog.Warn("Launch Library startup sync failed; cached data remains available", "error", err)
	}
	if err := dataSyncer.SyncMoonSpacecraft(initialSyncContext); err != nil {
		slog.Warn("JPL Horizons startup sync failed; static moon data remains available", "error", err)
	}
	if err := dataSyncer.SyncMarsSpacecraft(initialSyncContext); err != nil {
		slog.Warn("JPL Horizons startup sync failed; static mars data remains available", "error", err)
	}
	if err := dataSyncer.SyncDeepSpaceProbes(initialSyncContext); err != nil {
		slog.Warn("JPL Horizons probe sync failed; no deep-space positions available", "error", err)
	}
	cancelInitialSync()

	go schedule(ctx, 2*time.Hour, 45*time.Second, "celestrak", dataSyncer.SyncCelesTrak)
	go schedule(ctx, 30*time.Minute, 45*time.Second, "launch_library_2", dataSyncer.SyncLaunches)
	go schedule(ctx, 24*time.Hour, 45*time.Second, "moon", dataSyncer.SyncMoonSpacecraft)   // 月球轨道：每日 JPL Horizons 同步
	go schedule(ctx, 24*time.Hour, 45*time.Second, "mars", dataSyncer.SyncMarsSpacecraft)   // 火星轨道：每日 JPL Horizons 同步
	go schedule(ctx, 24*time.Hour, 120*time.Second, "probes", dataSyncer.SyncDeepSpaceProbes) // 深空探测器：每日同步（9 个顺序查询，预算放宽）

	server := &http.Server{Addr: ":" + cfg.Port, Handler: httpapi.Router(repository, moonRepository, marsRepository, voyageRepository), ReadHeaderTimeout: 5 * time.Second}
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

// schedule 周期执行数据同步，带自愈（曾出现：僵尸同步 1h38m + 该源 13.5h 零重试）：
//  - 每次运行放独立 goroutine，panic 只记日志不会杀死调度循环，下轮照常触发；
//  - busy 防重叠：上一轮未结束时本轮跳过（记日志）；
//  - 看门狗：运行超过 timeout+宽限仍未结束 → 放弃该轮并复位 busy，保证该源永不
//    因一次僵尸同步而永久停摆（下一轮 tick 会重新开始）。
func schedule(ctx context.Context, interval, timeout time.Duration, name string, run func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var busy atomic.Bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !busy.CompareAndSwap(false, true) {
				slog.Warn("scheduled data sync skipped (previous run still active)", "source", name)
				continue
			}
			// 看门狗：预算 + 10s 宽限后仍未完成 → 放弃该轮（不复用其 goroutine），下轮继续
			watchdog := time.AfterFunc(timeout+10*time.Second, func() {
				if busy.CompareAndSwap(true, false) {
					slog.Error("scheduled data sync hung past budget; abandoning run", "source", name, "budget", timeout)
				}
			})
			go func() {
				defer watchdog.Stop()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("scheduled data sync panicked; will retry next interval", "source", name, "panic", r)
					}
				}()
				defer busy.Store(false)
				runContext, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()
				if err := run(runContext); err != nil {
					slog.Warn("scheduled data sync failed", "source", name, "error", err)
				}
			}()
		}
	}
}
