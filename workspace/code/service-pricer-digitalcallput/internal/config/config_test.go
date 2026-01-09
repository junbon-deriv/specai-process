package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "symbols.yaml")

	configContent := `symbols:
  - symbol: "USD/JPY"
    min_stake: "1.00000000"
    max_payout: "50000.00000000"
    commission: "0.02000000"
  - symbol: "EUR/USD"
    min_stake: "1.00000000"
    max_payout: "50000.00000000"
    commission: "0.02000000"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test USD/JPY
	usdJpy, err := cfg.GetSymbolConfig("USD/JPY")
	if err != nil {
		t.Errorf("Failed to get USD/JPY config: %v", err)
	}
	if usdJpy.Symbol != "USD/JPY" {
		t.Errorf("Expected symbol USD/JPY, got %s", usdJpy.Symbol)
	}
	if !usdJpy.MinStake.Equal(decimal.NewFromFloat(1.0)) {
		t.Errorf("Expected min_stake 1.0, got %s", usdJpy.MinStake.String())
	}
	if !usdJpy.MaxPayout.Equal(decimal.NewFromFloat(50000.0)) {
		t.Errorf("Expected max_payout 50000.0, got %s", usdJpy.MaxPayout.String())
	}
	if !usdJpy.Commission.Equal(decimal.NewFromFloat(0.02)) {
		t.Errorf("Expected commission 0.02, got %s", usdJpy.Commission.String())
	}

	// Test unknown symbol
	_, err = cfg.GetSymbolConfig("UNKNOWN")
	if err == nil {
		t.Error("Expected error for unknown symbol, got nil")
	}
}
