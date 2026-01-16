// Package app handles application initialization and lifecycle.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/regentmarkets/service-pricer-doublerisefall/api/proto/doublerisefall/v1"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/config"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/contract"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/feed"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/grpcsvc"
	healthsvc "github.com/regentmarkets/service-pricer-doublerisefall/internal/health"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Config contains application configuration.
type Config struct {
	GRPCPort        string
	HealthPort      string
	FeedServiceAddr string
	ConfigPath      string
	LogLevel        string
}

// App represents the application.
type App struct {
	config       *Config
	logger       *slog.Logger
	grpcServer   *grpc.Server
	healthServer *healthsvc.Server
	feedClient   *feed.Client
	configMgr    *config.Manager
}

// New creates a new application with all dependencies wired.
func New(cfg *Config) (*App, error) {
	// Setup logger
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("initializing application", "config", cfg)

	// Create config manager
	configMgr, err := config.NewManager(cfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	logger.Info("configuration loaded", "path", cfg.ConfigPath)

	// Create feed client
	feedClient, err := feed.NewClient(cfg.FeedServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("feed client: %w", err)
	}
	logger.Info("feed client connected", "addr", cfg.FeedServiceAddr)

	// Create contract validator
	validator := contract.NewValidator(configMgr)

	// Create pricer (core logic)
	pricerSvc := pricer.NewPricer(configMgr, feedClient, validator)

	// Create gRPC service
	svc := grpcsvc.NewService(pricerSvc, validator, logger)

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryLoggingInterceptor(logger)),
		grpc.StreamInterceptor(streamLoggingInterceptor(logger)),
	)

	// Register services
	pb.RegisterDoubleRiseFallServiceServer(grpcServer, svc)
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())
	reflection.Register(grpcServer)

	logger.Info("gRPC server configured")

	// Create HTTP health server
	healthServer := healthsvc.NewServer(cfg.HealthPort, logger)
	logger.Info("health check server configured", "port", cfg.HealthPort)

	return &App{
		config:       cfg,
		logger:       logger,
		grpcServer:   grpcServer,
		healthServer: healthServer,
		feedClient:   feedClient,
		configMgr:    configMgr,
	}, nil
}

// Run starts the application and blocks until shutdown.
func (a *App) Run(ctx context.Context) error {
	// Listen on gRPC port
	lis, err := net.Listen("tcp", ":"+a.config.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	a.logger.Info("starting gRPC server", "port", a.config.GRPCPort)

	// Channel to receive errors from goroutines
	errCh := make(chan error, 2)

	// Start gRPC server in goroutine
	go func() {
		if err := a.grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("grpc server error: %w", err)
		}
	}()

	// Start HTTP health server in goroutine
	go func() {
		if err := a.healthServer.Start(); err != nil {
			errCh <- fmt.Errorf("health server error: %w", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		a.logger.Info("context cancelled, shutting down")
	case sig := <-sigCh:
		a.logger.Info("received signal, shutting down", "signal", sig)
	case err := <-errCh:
		a.logger.Error("server error", "error", err)
		return err
	}

	return a.Shutdown(context.Background())
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down application")

	// Stop HTTP health server
	if err := a.healthServer.Shutdown(); err != nil {
		a.logger.Error("failed to shutdown health server", "error", err)
	}

	// Stop gRPC server
	a.grpcServer.GracefulStop()
	a.logger.Info("gRPC server stopped")

	// Close feed client
	if err := a.feedClient.Close(); err != nil {
		a.logger.Error("failed to close feed client", "error", err)
	}

	a.logger.Info("application shutdown complete")
	return nil
}

// unaryLoggingInterceptor logs unary RPC calls.
func unaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		logger.Debug("unary call", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error("unary call failed", "method", info.FullMethod, "error", err)
		}
		return resp, err
	}
}

// streamLoggingInterceptor logs stream RPC calls.
func streamLoggingInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		logger.Debug("stream call", "method", info.FullMethod)
		err := handler(srv, ss)
		if err != nil {
			logger.Error("stream call failed", "method", info.FullMethod, "error", err)
		}
		return err
	}
}
