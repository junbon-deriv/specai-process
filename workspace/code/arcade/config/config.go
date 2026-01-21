package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration for the arcade service
type Config struct {
	Port        int    `mapstructure:"PORT"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	DBMaxConns  int    `mapstructure:"DB_MAX_CONNS"`
	DBMinConns  int    `mapstructure:"DB_MIN_CONNS"`
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("DB_MAX_CONNS", 25)
	viper.SetDefault("DB_MIN_CONNS", 5)

	// Try to read .env file (optional, ignore error if file doesn't exist)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig() // Ignore error if .env file doesn't exist

	// Environment variables override .env file values
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &cfg, nil
}
