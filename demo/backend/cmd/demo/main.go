package main

import (
	"aurora/backend/internal/httpapi"
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18082", "listen address")
	data := flag.String("data", "../data", "local snapshot directory")
	flag.Parse()
	handler, closeData, err := httpapi.DemoRouter(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer closeData()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: *addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Printf("AURORA offline demo listening at %s", *addr)
		if e := server.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			log.Printf("server: %v", e)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
