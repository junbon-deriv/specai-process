package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/app"
)

func main() {
	// Load configuration from environment variables
	cfg := &app.Config{
		GRPCPort:          getEnvInt("GRPC_PORT", 50051),
		FeedServiceAddr:   getEnv("FEED_SERVICE_ADDR", "localhost:50051"),
		ConfigPath:        getEnv("CONFIG_PATH", "./config/symbols.yaml"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		FeedRetryAttempts: getEnvInt("FEED_RETRY_ATTEMPTS", 3),
		FeedRetryDelay:    time.Duration(getEnvInt("FEED_RETRY_DELAY_MS", 1000)) * time.Millisecond,
	}

	// Create application
	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create application: %v\n", err)
		os.Exit(1)
	}

	// Run application
	if err := application.Run(context.Background(), cfg.GRPCPort); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an environment variable as an integer or returns a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
