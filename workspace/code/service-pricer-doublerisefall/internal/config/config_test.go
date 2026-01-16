package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
)

// Test NewManager with valid config
func TestNewManager_ValidConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "symbols.yml")

	configContent := `symbols:
  R_10:
    commission: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_100:
    commission: 0.06
    max_payout: 2000.00
    min_stake: 2.00
    enabled: true

defaults:
  commission: 0.05
  max_payout: 1000.00
  min_stake: 1.00
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test NewManager
	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if mgr == nil {
		t.Fatal("NewManager() returned nil manager")
	}

	// Verify loaded configs
	cfg, err := mgr.GetSymbolConfig("R_10")
	if err != nil {
		t.Errorf("GetSymbolConfig(R_10) error = %v", err)
	}
	if cfg.Commission != 0.05 {
		t.Errorf("R_10 commission = %v, want 0.05", cfg.Commission)
	}
	if cfg.MaxPayout != 1000.00 {
		t.Errorf("R_10 max_payout = %v, want 1000.00", cfg.MaxPayout)
	}
	if cfg.MinStake != 1.00 {
		t.Errorf("R_10 min_stake = %v, want 1.00", cfg.MinStake)
	}
	if !cfg.Enabled {
		t.Error("R_10 should be enabled")
	}

	cfg2, err := mgr.GetSymbolConfig("R_100")
	if err != nil {
		t.Errorf("GetSymbolConfig(R_100) error = %v", err)
	}
	if cfg2.Commission != 0.06 {
		t.Errorf("R_100 commission = %v, want 0.06", cfg2.Commission)
	}
}

// Test NewManager with non-existent file
func TestNewManager_FileNotFound(t *testing.T) {
	_, err := NewManager("/non/existent/path.yml")
	if err == nil {
		t.Error("NewManager() expected error for non-existent file, got nil")
	}
}

// Test NewManager with invalid YAML
func TestNewManager_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yml")

	invalidContent := `this is not: valid: yaml: content`
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err = NewManager(configPath)
	if err == nil {
		t.Error("NewManager() expected error for invalid YAML, got nil")
	}
}

// Test GetSymbolConfig
func TestManager_GetSymbolConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "symbols.yml")

	configContent := `symbols:
  R_10:
    commission: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_25:
    commission: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: false

defaults:
  commission: 0.05
  max_payout: 1000.00
  min_stake: 1.00
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	tests := []struct {
		name    string
		symbol  string
		wantErr bool
	}{
		{
			name:    "existing symbol R_10",
			symbol:  "R_10",
			wantErr: false,
		},
		{
			name:    "existing symbol R_25",
			symbol:  "R_25",
			wantErr: false,
		},
		{
			name:    "non-existing symbol",
			symbol:  "INVALID",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := mgr.GetSymbolConfig(tt.symbol)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSymbolConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg == nil {
				t.Error("GetSymbolConfig() returned nil config")
			}
			if !tt.wantErr && cfg.Symbol != tt.symbol {
				t.Errorf("GetSymbolConfig() symbol = %v, want %v", cfg.Symbol, tt.symbol)
			}
		})
	}
}

// Test GetSymbolConfig returns error for invalid symbol
func TestManager_GetSymbolConfig_InvalidSymbol(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "symbols.yml")

	configContent := `symbols:
  R_10:
    commission: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true

defaults:
  commission: 0.05
  max_payout: 1000.00
  min_stake: 1.00
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	_, err = mgr.GetSymbolConfig("INVALID")
	if err == nil {
		t.Error("GetSymbolConfig() expected error for invalid symbol, got nil")
	}
	if err != pricer.ErrInvalidSymbol {
		t.Errorf("GetSymbolConfig() error = %v, want ErrInvalidSymbol", err)
	}
}

// Test Reload
func TestManager_Reload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "symbols.yml")

	initialContent := `symbols:
  R_10:
    commission: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true

defaults:
  commission: 0.05
  max_payout: 1000.00
  min_stake: 1.00
`

	err := os.WriteFile(configPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Verify initial config
	cfg, err := mgr.GetSymbolConfig("R_10")
	if err != nil {
		t.Fatalf("GetSymbolConfig() error = %v", err)
	}
	if cfg.Commission != 0.05 {
		t.Errorf("Initial commission = %v, want 0.05", cfg.Commission)
	}

	// Update config file
	updatedContent := `symbols:
  R_10:
    commission: 0.10
    max_payout: 2000.00
    min_stake: 2.00
    enabled: true

defaults:
  commission: 0.05
  max_payout: 1000.00
  min_stake: 1.00
`

	err = os.WriteFile(configPath, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Reload config
	err = mgr.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	// Verify reloaded config
	cfg, err = mgr.GetSymbolConfig("R_10")
	if err != nil {
		t.Fatalf("GetSymbolConfig() after reload error = %v", err)
	}
	if cfg.Commission != 0.10 {
		t.Errorf("After reload commission = %v, want 0.10", cfg.Commission)
	}
	if cfg.MaxPayout != 2000.00 {
		t.Errorf("After reload max_payout = %v, want 2000.00", cfg.MaxPayout)
	}
}

// Test GetSymbolConfig returns copy
func TestManager_GetSymbolConfig_ReturnsCopy(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "symbols.yml")

	configContent := `symbols:
  R_10:
    commission: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true

defaults:
  commission: 0.05
  max_payout: 1000.00
  min_stake: 1.00
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Get config twice
	cfg1, err := mgr.GetSymbolConfig("R_10")
	if err != nil {
		t.Fatalf("GetSymbolConfig() error = %v", err)
	}

	cfg2, err := mgr.GetSymbolConfig("R_10")
	if err != nil {
		t.Fatalf("GetSymbolConfig() error = %v", err)
	}

	// Modify one config
	cfg1.Commission = 0.99

	// Verify the other config is unchanged
	if cfg2.Commission != 0.05 {
		t.Error("GetSymbolConfig() should return copies, not references")
	}
}
