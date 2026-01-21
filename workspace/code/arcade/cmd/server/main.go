package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deriv/arcade/config"
	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/api"
	"github.com/deriv/arcade/internal/repository"
	"github.com/deriv/arcade/internal/series"
	"github.com/deriv/arcade/internal/trading"
)

func main() {
	// Setup logging
	var handler slog.Handler
	handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set log level from config
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Recreate handler with configured level
	handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Starting arcade service",
		"port", cfg.Port,
		"log_level", cfg.LogLevel)

	// Create repository manager (handles connection, migration, and initialization)
	// 30s timeout for connection + migration
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()

	repoManager, err := repository.New(initCtx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		slog.Error("Failed to initialize repository manager", "error", err)
		os.Exit(1)
	}
	defer repoManager.Close()

	slog.Info("Database connection established and schema migrated")

	// Initialize services with injected repositories
	accountService := accounts.NewService(repoManager.AccountsRepository())
	seriesService := series.NewService(repoManager.SeriesRepository())
	tradingService := trading.NewService(repoManager.TradingRepository(), accountService, seriesService)

	// Create router
	router := api.NewRouter(accountService, tradingService)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server stopped")
}
