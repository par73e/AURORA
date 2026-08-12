package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
	_ "time/tzdata" // 内嵌 IANA 时区数据：月相/评分按 timezone 参数解析本地日期，不依赖系统 zoneinfo

	"aurora/backend/internal/astronomyevent"
	"aurora/backend/internal/config"
	"aurora/backend/internal/database"
	"aurora/backend/internal/httpapi"
	observerlocation "aurora/backend/internal/location"
	"aurora/backend/internal/mars"
	"aurora/backend/internal/moon"
	"aurora/backend/internal/observatory"
	"aurora/backend/internal/orbit"
	"aurora/backend/internal/syncer"
	"aurora/backend/internal/voyage"
	"github.com/joho/godotenv"
)

func main() {
	loadLocalEnvironment()
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
	eventRepository := astronomyevent.NewRepository(pool)
	geocoder := observerlocation.NewAMapClient(cfg.AMapWebKey)
	conditions := observatory.NewClient()
	moons := observatory.NewMoonService()
	var lights observatory.LightPollutionProvider = observatory.NewLightPollutionClient(cfg.LightPollutionKey)
	if cfg.LightPollutionURL != "" {
		lights = observatory.NewLightPollutionClientWithURLs(cfg.LightPollutionKey, cfg.LightPollutionURL, &http.Client{Timeout: 8 * time.Second})
	}
	if cfg.LightPollutionDataPath != "" {
		rasterLights, rasterErr := observatory.OpenRasterLightProvider(cfg.LightPollutionDataPath)
		if rasterErr != nil {
			slog.Warn("local light pollution raster unavailable; using configured fallback", "path", cfg.LightPollutionDataPath, "error", rasterErr)
		} else {
			lights = rasterLights
			defer rasterLights.Close()
			slog.Info("local VIIRS light pollution raster loaded", "path", cfg.LightPollutionDataPath)
		}
	}
	dataSyncer := syncer.NewWithMoonVoyageMars(repository, moonRepository, marsRepository, voyageRepository)
	eventSourceSyncer := astronomyevent.NewSourceSyncer(eventRepository)
	ephemerisSyncer := astronomyevent.NewEphemerisSyncer(eventRepository)

	server := &http.Server{Addr: ":" + cfg.Port, Handler: httpapi.Router(repository, moonRepository, marsRepository, voyageRepository, eventRepository, geocoder, conditions, moons, lights), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		slog.Info("AURORA API started", "address", "http://localhost:"+cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server stopped unexpectedly", "error", err)
			stop()
		}
	}()
	// 先提供缓存数据和健康接口；远端源变慢时不再把整个网站卡在启动阶段。
	go runStartupSync(ctx, dataSyncer)
	go runAstronomySourceStartupSync(ctx, eventSourceSyncer)
	go runPlanetaryEphemerisSync(ctx, ephemerisSyncer)
	go schedule(ctx, 2*time.Hour, 45*time.Second, "celestrak", dataSyncer.SyncCelesTrak)
	go schedule(ctx, 30*time.Minute, 45*time.Second, "launch_library_2", dataSyncer.SyncLaunches)
	go schedule(ctx, 24*time.Hour, 45*time.Second, "moon", dataSyncer.SyncMoonSpacecraft)                         // 月球轨道：每日 JPL Horizons 同步
	go schedule(ctx, 24*time.Hour, 45*time.Second, "mars", dataSyncer.SyncMarsSpacecraft)                         // 火星轨道：每日 JPL Horizons 同步
	go schedule(ctx, 24*time.Hour, 120*time.Second, "probes", dataSyncer.SyncDeepSpaceProbes)                     // 深空探测器：每日同步（9 个顺序查询，预算放宽）
	go schedule(ctx, 24*time.Hour, 45*time.Second, "astronomy_sources", eventSourceSyncer.SyncOfficialSources)    // 天象权威资料：每日校验并缓存
	go schedule(ctx, 24*time.Hour, 90*time.Second, "planetary_ephemeris", ephemerisSyncer.SyncPlanetaryPositions) // JPL 行星星历：每日缓存未来 18 个月

	<-ctx.Done()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = server.Shutdown(shutdownContext)
}

// loadLocalEnvironment 只服务本地开发；生产环境仍由部署系统注入环境变量。
func loadLocalEnvironment() {
	for _, filename := range []string{".env", "backend/.env"} {
		if err := godotenv.Load(filename); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("load local environment", "file", filename, "error", err)
			return
		}
	}
}

// runStartupSync 在 API 已可用后刷新缓存；所有同步共用原先的 120 秒预算，避免首启无限占用网络请求。
func runStartupSync(parent context.Context, dataSyncer *syncer.Syncer) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	if err := dataSyncer.SyncCelesTrak(ctx); err != nil {
		slog.Warn("CelesTrak startup sync failed; cached data remains available", "error", err)
	}
	if err := dataSyncer.SyncLaunches(ctx); err != nil {
		slog.Warn("Launch Library startup sync failed; cached data remains available", "error", err)
	}
	if err := dataSyncer.SyncMoonSpacecraft(ctx); err != nil {
		slog.Warn("JPL Horizons moon startup sync failed; static moon data remains available", "error", err)
	}
	if err := dataSyncer.SyncMarsSpacecraft(ctx); err != nil {
		slog.Warn("JPL Horizons mars startup sync failed; static mars data remains available", "error", err)
	}
	if err := dataSyncer.SyncDeepSpaceProbes(ctx); err != nil {
		slog.Warn("JPL Horizons probe startup sync failed; cached data remains available", "error", err)
	}
}

// runAstronomySourceStartupSync 在服务可用后缓存公开权威资料；资料源失败不会影响已存事件查询。
func runAstronomySourceStartupSync(parent context.Context, sourceSyncer *astronomyevent.SourceSyncer) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	if err := sourceSyncer.SyncOfficialSources(ctx); err != nil {
		slog.Warn("astronomy source startup sync failed; cached events remain available", "error", err)
	}
}

func runPlanetaryEphemerisSync(parent context.Context, syncer *astronomyevent.EphemerisSyncer) {
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	if err := syncer.SyncPlanetaryPositions(ctx); err != nil {
		slog.Warn("planetary ephemeris startup sync failed; cached event data remains available", "error", err)
	}
}

// schedule 周期执行数据同步，带自愈（曾出现：僵尸同步 1h38m + 该源 13.5h 零重试）：
//   - 每次运行放独立 goroutine，panic 只记日志不会杀死调度循环，下轮照常触发；
//   - 代际防重叠：activeGen 保存当前运行轮次；非零时本轮跳过（记日志）；
//   - 看门狗：运行超过 timeout+宽限仍未结束 → 放弃该轮（CAS 仅当仍是自己这轮才复位）；
//     僵尸 goroutine 迟归时其 CAS 无法覆盖后继轮次（代际不匹配）→ 源永不因僵尸同步停摆。
func schedule(ctx context.Context, interval, timeout time.Duration, name string, run func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var tickGen atomic.Uint64
	var activeGen atomic.Uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gen := tickGen.Add(1)
			if !activeGen.CompareAndSwap(0, gen) {
				slog.Warn("scheduled data sync skipped (previous run still active)", "source", name)
				continue
			}
			// 看门狗：预算 + 10s 宽限后仍未完成 → 放弃该轮（仅复位自己这一代）
			watchdog := time.AfterFunc(timeout+10*time.Second, func() {
				if activeGen.CompareAndSwap(gen, 0) {
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
				// 仅当自己仍是当前轮次才释放（僵尸迟归不会覆盖后继轮次）
				defer activeGen.CompareAndSwap(gen, 0)
				runContext, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()
				if err := run(runContext); err != nil {
					slog.Warn("scheduled data sync failed", "source", name, "error", err)
				}
			}()
		}
	}
}
