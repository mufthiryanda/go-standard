package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-standard/internal/config"
	"go-standard/internal/di"
	"go-standard/internal/esindex"
	"go-standard/internal/handler"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func main() {
	// ── Config ──────────────────────────────────────────────────────────────
	cfg, err := config.LoadConfig()
	if err != nil {
		// zap not yet available; use stderr before structured logging is ready.
		_, _ = fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// ── Wire DI ─────────────────────────────────────────────────────────────
	app, cleanup, err := di.InitializeApp(cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	logger := app.Logger

	// ── Elasticsearch index bootstrap ────────────────────────────────────────
	if err := esindex.Bootstrap(app.ES); err != nil {
		logger.Fatal("esindex bootstrap failed", zap.Error(err))
	}
	logger.Info("esindex bootstrap complete")

	// ── Fiber ────────────────────────────────────────────────────────────────
	fiberApp := fiber.New(fiber.Config{
		AppName:       cfg.App.Name,
		BodyLimit:     4 * 1024 * 1024, // 4 MB
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  30 * time.Second,
		IdleTimeout:   60 * time.Second,
		StrictRouting: false,
		CaseSensitive: false,
		ErrorHandler:  app.ErrorHandler,
	})

	handler.SetupRoutes(
		fiberApp,
		app.UserHandler,
		fiber.Handler(app.AuthMW),
		fiber.Handler(app.DefaultLimiter),
		fiber.Handler(app.AuthLimiter),
		fiber.Handler(app.Recover),
		fiber.Handler(app.RequestID),
		fiber.Handler(app.LoggerMW),
		fiber.Handler(app.CORS),
		app.ErrorHandler,
	)

	// ── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine so the main goroutine can listen for signals.
	serverErr := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.App.Port)
		logger.Info("server starting", zap.String("addr", addr), zap.String("env", cfg.App.Env))
		if sErr := fiberApp.Listen(addr); sErr != nil {
			serverErr <- sErr
		}
	}()

	select {
	case sig := <-quit:
		logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	case sErr := <-serverErr:
		logger.Error("server error", zap.Error(sErr))
	}

	logger.Info("shutting down server...")
	if sErr := fiberApp.ShutdownWithTimeout(30 * time.Second); sErr != nil {
		logger.Error("server shutdown error", zap.Error(sErr))
	}

	logger.Info("server stopped gracefully")
}
