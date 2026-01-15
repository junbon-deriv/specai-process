package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTempConfig(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "symbols.yaml")
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)
	return configPath
}

// TC-DF-C1Q: Load valid configuration
func TestLoad_ValidConfig(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_50:
    symbol: R_50
    commission_rate: 0.04
    max_payout: 500.00
    min_stake: 2.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)

	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify R_100
	r100, err := cfg.GetSymbolConfig("R_100")
	require.NoError(t, err)
	assert.Equal(t, "R_100", r100.Symbol)
	assert.Equal(t, 0.05, r100.CommissionRate)
	assert.Equal(t, 1000.00, r100.MaxPayout)
	assert.Equal(t, 1.00, r100.MinStake)
	assert.True(t, r100.Enabled)

	// Verify R_50
	r50, err := cfg.GetSymbolConfig("R_50")
	require.NoError(t, err)
	assert.Equal(t, "R_50", r50.Symbol)
	assert.Equal(t, 0.04, r50.CommissionRate)
}

// TC-DF-C2R: Return error for unknown symbol
func TestGetSymbolConfig_UnknownSymbol(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	result, err := cfg.GetSymbolConfig("UNKNOWN")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, pricer.ErrInvalidSymbol)
}

// TC-DF-C3S: Return error for disabled symbol
func TestGetSymbolConfig_DisabledSymbol(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: false
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	result, err := cfg.GetSymbolConfig("R_100")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, pricer.ErrSymbolDisabled)
}

// Test invalid config file path
func TestLoad_InvalidPath(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")

	assert.Nil(t, cfg)
	assert.Error(t, err)
}

// Test invalid YAML content
func TestLoad_InvalidYAML(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: invalid_value
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)

	// Viper may parse this differently, but validation should catch it
	if cfg != nil {
		_, getErr := cfg.GetSymbolConfig("R_100")
		assert.Error(t, getErr)
	} else {
		assert.Error(t, err)
	}
}

// Test validation: empty symbol
func TestLoad_EmptySymbol(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: ""
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symbol cannot be empty")
}

// Test validation: negative commission rate
func TestLoad_NegativeCommissionRate(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: -0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commission_rate must be between 0 and 1")
}

// Test validation: commission rate > 1
func TestLoad_CommissionRateExceedsOne(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 1.5
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commission_rate must be between 0 and 1")
}

// Test validation: non-positive max payout
func TestLoad_NonPositiveMaxPayout(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 0
    min_stake: 1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_payout must be positive")
}

// Test validation: non-positive min stake
func TestLoad_NonPositiveMinStake(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: -1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "min_stake must be positive")
}

// Test reload functionality
func TestConfig_Reload(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Update the config file
	newContent := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.10
    max_payout: 2000.00
    min_stake: 2.00
    enabled: true
`
	err = os.WriteFile(configPath, []byte(newContent), 0644)
	require.NoError(t, err)

	// Reload
	err = cfg.Reload()
	require.NoError(t, err)

	// Verify updated values
	r100, err := cfg.GetSymbolConfig("R_100")
	require.NoError(t, err)
	assert.Equal(t, 0.10, r100.CommissionRate)
	assert.Equal(t, 2000.00, r100.MaxPayout)
	assert.Equal(t, 2.00, r100.MinStake)
}

// Test multiple symbols
func TestLoad_AllSymbols(t *testing.T) {
	content := `
symbols:
  R_10:
    symbol: R_10
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_25:
    symbol: R_25
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_50:
    symbol: R_50
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_75:
    symbol: R_75
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	symbols := []string{"R_10", "R_25", "R_50", "R_75", "R_100"}
	for _, sym := range symbols {
		symCfg, err := cfg.GetSymbolConfig(sym)
		require.NoError(t, err, "Failed to get config for %s", sym)
		assert.Equal(t, sym, symCfg.Symbol)
		assert.True(t, symCfg.Enabled)
	}
}

// Test concurrent access
func TestConfig_ConcurrentAccess(t *testing.T) {
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
`
	configPath := createTempConfig(t, content)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = cfg.GetSymbolConfig("R_100")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
