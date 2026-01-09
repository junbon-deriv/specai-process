package main

import (
	"context"
	"log"
	"os"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/app"
)

func main() {
	// Load configuration from environment variables
	cfg := &app.Config{
		GRPCAddress: getEnv("GRPC_ADDRESS", ":50051"),
		FeedHost:    getEnv("FEED_HOST", "localhost"),
		FeedPort:    getEnv("FEED_PORT", "50052"),
		ConfigPath:  getEnv("CONFIG_PATH", "./config/symbols.yaml"),
		LogLevel:    getEnv("LOG_LEVEL", "INFO"),
	}

	// Create and run application
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	ctx := context.Background()
	if err := application.Run(ctx); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
