// cacheproxy adds a bounded, optional Redis response cache in front of the local API.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aurora/backend/internal/responsecache"
	"github.com/redis/go-redis/v9"
)

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	upstream, err := url.Parse(env("AURORA_CACHE_UPSTREAM", "http://127.0.0.1:18083"))
	if err != nil || upstream.Scheme != "http" || upstream.Hostname() != "127.0.0.1" {
		slog.Error("cache upstream must be loopback HTTP")
		os.Exit(1)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     env("AURORA_REDIS_ADDR", "127.0.0.1:6380"),
		Username: env("AURORA_REDIS_USERNAME", "aurora-app"), Password: os.Getenv("AURORA_REDIS_PASSWORD"),
		PoolSize: 4, MaxActiveConns: 8, MinIdleConns: 0, PoolTimeout: 30 * time.Millisecond,
		DialTimeout: 50 * time.Millisecond, ReadTimeout: 50 * time.Millisecond, WriteTimeout: 50 * time.Millisecond,
		ContextTimeoutEnabled: true, MaxRetries: -1, DialerRetries: 1,
	})
	defer client.Close()
	cache := responsecache.New(ctx, upstream, responsecache.RedisStore{Client: client})
	mux := http.NewServeMux()
	mux.Handle("/api/", cache)
	mux.HandleFunc("/internal/cache-stats", cache.Stats)
	server := &http.Server{Addr: "127.0.0.1:18084", Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("cache gateway stopped", "error", err)
			stop()
		}
	}()
	slog.Info("AURORA cache gateway started on loopback")
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
