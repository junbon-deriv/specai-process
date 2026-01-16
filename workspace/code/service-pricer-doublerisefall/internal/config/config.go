// Package config manages symbol configuration.
package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"gopkg.in/yaml.v3"
)

// Config represents the full configuration file structure.
type Config struct {
	Symbols  map[string]SymbolConfigYAML `yaml:"symbols"`
	Defaults DefaultsYAML                `yaml:"defaults"`
}

// SymbolConfigYAML represents symbol config in YAML format.
type SymbolConfigYAML struct {
	Commission float64 `yaml:"commission"`
	MaxPayout  float64 `yaml:"max_payout"`
	MinStake   float64 `yaml:"min_stake"`
	Enabled    bool    `yaml:"enabled"`
}

// DefaultsYAML represents default config values.
type DefaultsYAML struct {
	Commission float64 `yaml:"commission"`
	MaxPayout  float64 `yaml:"max_payout"`
	MinStake   float64 `yaml:"min_stake"`
}

// Manager handles configuration loading and access.
type Manager struct {
	mu      sync.RWMutex
	configs map[string]*pricer.SymbolConfig
	path    string
}

// NewManager creates a config manager and loads configuration.
func NewManager(path string) (*Manager, error) {
	m := &Manager{
		configs: make(map[string]*pricer.SymbolConfig),
		path:    path,
	}

	if err := m.load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return m, nil
}

// GetSymbolConfig returns configuration for a specific symbol.
func (m *Manager) GetSymbolConfig(symbol string) (*pricer.SymbolConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg, ok := m.configs[symbol]
	if !ok {
		return nil, pricer.ErrInvalidSymbol
	}

	// Return a copy to prevent modification
	configCopy := *cfg
	return &configCopy, nil
}

// Reload reloads configuration from disk.
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.load()
}

// load reads and parses the configuration file.
func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Build symbol configs
	configs := make(map[string]*pricer.SymbolConfig)
	for symbol, symCfg := range cfg.Symbols {
		configs[symbol] = &pricer.SymbolConfig{
			Symbol:     symbol,
			Commission: symCfg.Commission,
			MaxPayout:  symCfg.MaxPayout,
			MinStake:   symCfg.MinStake,
			Enabled:    symCfg.Enabled,
		}
	}

	m.configs = configs
	return nil
}
