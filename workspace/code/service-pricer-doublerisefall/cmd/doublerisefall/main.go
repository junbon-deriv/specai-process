// Package main is the application entry point.
package main

import (
	"context"
	"log"
	"os"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/app"
)

func main() {
	// Load configuration from environment
	cfg := &app.Config{
		GRPCPort:        getEnv("GRPC_PORT", "50051"),
		HealthPort:      getEnv("HEALTH_PORT", "8081"),
		FeedServiceAddr: getEnv("FEED_SERVICE_ADDR", "localhost:50051"),
		ConfigPath:      getEnv("CONFIG_PATH", "config/symbols.yml"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}

	// Create application
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to create application: %v", err)
	}

	// Run application
	if err := application.Run(context.Background()); err != nil {
		log.Fatalf("application error: %v", err)
	}
}

// getEnv retrieves environment variable with fallback to default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
