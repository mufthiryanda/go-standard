package main

import (
	"go-standard/internal/config"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-standard/internal/di"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("worker: config load failed: %v", err)
	}

	server, cleanup, err := di.InitializeWorker(cfg)
	if err != nil {
		log.Fatalf("worker: init failed: %v", err)
	}
	defer cleanup()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("worker: signal received: %v — shutting down", sig)
	case err := <-errCh:
		log.Fatalf("worker: server error: %v", err)
	}
	// cleanup() via defer — triggers asynq.Server.Shutdown()
}
