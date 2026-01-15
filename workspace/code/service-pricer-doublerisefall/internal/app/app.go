package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/regentmarkets/service-pricer-doublerisefall/api/proto/doublerisefall/v1"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/config"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/contract"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/feed"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/grpcsvc"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// App represents the application with all dependencies.
type App struct {
	cfg         *config.Config
	feedClient  *feed.Client
	validator   *contract.Validator
	pricer      *pricer.Pricer
	grpcService *grpcsvc.Service
	grpcServer  *grpc.Server
	logger      *slog.Logger
}

// Config holds application configuration.
type Config struct {
	GRPCPort          int
	FeedServiceAddr   string
	ConfigPath        string
	LogLevel          string
	FeedRetryAttempts int
	FeedRetryDelay    time.Duration
}

// New creates a new application with all dependencies wired.
func New(cfg *Config) (*App, error) {
	// Setup logger
	logLevel := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("initializing application",
		"grpc_port", cfg.GRPCPort,
		"feed_addr", cfg.FeedServiceAddr,
		"config_path", cfg.ConfigPath,
	)

	// Load configuration
	symbolConfig, err := config.Load(cfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger.Info("configuration loaded successfully")

	// Create feed client
	feedClient, err := feed.NewClient(
		cfg.FeedServiceAddr,
		cfg.FeedRetryAttempts,
		cfg.FeedRetryDelay,
	)
	if err != nil {
		return nil, fmt.Errorf("create feed client: %w", err)
	}

	logger.Info("feed client created", "address", cfg.FeedServiceAddr)

	// Create contract validator
	validator := contract.NewValidator(symbolConfig)

	// Create pricer
	pricerService := pricer.New(symbolConfig, feedClient, validator)

	logger.Info("pricer service initialized")

	// Create gRPC service
	grpcService := grpcsvc.New(pricerService, logger)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register services
	pb.RegisterDoubleRiseFallServiceServer(grpcServer, grpcService)

	// Register health check
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Register reflection for development
	reflection.Register(grpcServer)

	logger.Info("gRPC services registered")

	// Enable config hot-reload
	symbolConfig.Watch(func() {
		logger.Info("configuration reloaded")
	})

	return &App{
		cfg:         symbolConfig,
		feedClient:  feedClient,
		validator:   validator,
		pricer:      pricerService,
		grpcService: grpcService,
		grpcServer:  grpcServer,
		logger:      logger,
	}, nil
}

// Run starts the application and blocks until shutdown.
func (a *App) Run(ctx context.Context, port int) error {
	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", port, err)
	}

	a.logger.Info("starting gRPC server", "port", port)

	// Create error channel for server errors
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := a.grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("serve: %w", err)
		}
	}()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		a.logger.Info("received shutdown signal", "signal", sig.String())
		return a.Shutdown(context.Background())
	case <-ctx.Done():
		a.logger.Info("context cancelled")
		return a.Shutdown(context.Background())
	}
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down application")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Graceful stop for gRPC server
	stopped := make(chan struct{})
	go func() {
		a.grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or timeout
	select {
	case <-stopped:
		a.logger.Info("gRPC server stopped gracefully")
	case <-shutdownCtx.Done():
		a.logger.Warn("shutdown timeout, forcing stop")
		a.grpcServer.Stop()
	}

	// Close feed client
	if err := a.feedClient.Close(); err != nil {
		a.logger.Error("error closing feed client", "error", err)
	}

	a.logger.Info("application shutdown complete")
	return nil
}
