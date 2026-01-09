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

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/regentmarkets/service-feed/client"
	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/config"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/grpcsvc"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/market"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// Config holds application configuration
type Config struct {
	GRPCAddress string
	FeedHost    string
	FeedPort    string
	ConfigPath  string
	LogLevel    string
}

// App represents the application
type App struct {
	config     *Config
	grpcServer *grpc.Server
	feedClient *client.Client
	logger     *slog.Logger
}

// New creates a new application instance
func New(cfg *Config) (*App, error) {
	// Setup logger
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	return &App{
		config: cfg,
		logger: logger,
	}, nil
}

// Run starts the application
func (a *App) Run(ctx context.Context) error {
	// Load symbol configuration
	a.logger.Info("Loading symbol configuration", "path", a.config.ConfigPath)
	symbolConfig, err := config.Load(a.config.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Connect to service-feed
	feedAddr := fmt.Sprintf("%s:%s", a.config.FeedHost, a.config.FeedPort)
	a.logger.Info("Connecting to service-feed", "address", feedAddr)
	feedClient, err := client.New(feedAddr, 3, 1*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to service-feed: %w", err)
	}
	a.feedClient = feedClient
	defer feedClient.Close()

	// Create market data provider
	marketFeed := market.NewFeed(feedClient)

	// Create pricer
	pricer := pricing.NewPricer(marketFeed, symbolConfig)

	// Create gRPC service
	grpcService := grpcsvc.New(pricer, marketFeed, a.logger)

	// Setup gRPC server
	a.grpcServer = grpc.NewServer()
	pb.RegisterDigitalCallPutServiceServer(a.grpcServer, grpcService)

	// Register health check service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("digitalcallput.v1.DigitalCallPutService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(a.grpcServer, healthServer)

	reflection.Register(a.grpcServer)

	// Start listening
	listener, err := net.Listen("tcp", a.config.GRPCAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		a.logger.Info("Starting gRPC server", "address", a.config.GRPCAddress)
		serverErrors <- a.grpcServer.Serve(listener)
	}()

	// Setup signal handling
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		a.logger.Info("Received shutdown signal", "signal", sig)

		// Graceful shutdown
		a.logger.Info("Initiating graceful shutdown")
		a.grpcServer.GracefulStop()

		return nil
	case <-ctx.Done():
		a.logger.Info("Context cancelled")
		a.grpcServer.GracefulStop()
		return ctx.Err()
	}
}
