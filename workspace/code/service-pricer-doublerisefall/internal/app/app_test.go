package app

import (
	"testing"
)

// Test Config structure
func TestConfig_Structure(t *testing.T) {
	cfg := &Config{
		GRPCPort:        "50051",
		FeedServiceAddr: "localhost:50051",
		ConfigPath:      "config/symbols.yml",
		LogLevel:        "info",
	}

	if cfg.GRPCPort != "50051" {
		t.Errorf("GRPCPort = %v, want 50051", cfg.GRPCPort)
	}
	if cfg.FeedServiceAddr != "localhost:50051" {
		t.Errorf("FeedServiceAddr = %v, want localhost:50051", cfg.FeedServiceAddr)
	}
	if cfg.ConfigPath != "config/symbols.yml" {
		t.Errorf("ConfigPath = %v, want config/symbols.yml", cfg.ConfigPath)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
}

// Test New with invalid config path
func TestNew_InvalidConfigPath(t *testing.T) {
	cfg := &Config{
		GRPCPort:        "50051",
		FeedServiceAddr: "localhost:50051",
		ConfigPath:      "/non/existent/path.yml",
		LogLevel:        "info",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("New() expected error for invalid config path, got nil")
	}
}

// Test New with invalid feed service address
func TestNew_InvalidFeedAddr(t *testing.T) {
	t.Skip("Skipping - requires valid config file and would attempt connection")
}

// Test App lifecycle methods exist
func TestApp_Methods(t *testing.T) {
	// Verify methods exist on App type
	var app *App
	if app == nil {
		// Just checking type has expected methods
		_ = app.Run
		_ = app.Shutdown
	}
}
