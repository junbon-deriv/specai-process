package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"github.com/spf13/viper"
)

// Config manages symbol-specific configuration.
type Config struct {
	mu      sync.RWMutex
	symbols map[string]*pricer.SymbolConfig
	v       *viper.Viper
}

// symbolConfigYAML represents the YAML structure for symbol configuration.
type symbolConfigYAML struct {
	Symbol         string  `mapstructure:"symbol"`
	CommissionRate float64 `mapstructure:"commission_rate"`
	MaxPayout      float64 `mapstructure:"max_payout"`
	MinStake       float64 `mapstructure:"min_stake"`
	Enabled        bool    `mapstructure:"enabled"`
}

// configYAML represents the root YAML structure.
type configYAML struct {
	Symbols map[string]symbolConfigYAML `mapstructure:"symbols"`
}

// Load loads configuration from a YAML file.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		symbols: make(map[string]*pricer.SymbolConfig),
		v:       v,
	}

	if err := cfg.load(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// load parses the configuration and validates it.
func (c *Config) load() error {
	var yamlCfg configYAML
	if err := c.v.Unmarshal(&yamlCfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear existing config
	c.symbols = make(map[string]*pricer.SymbolConfig)

	// Load symbols (Viper lowercases all map keys)
	for key, symCfg := range yamlCfg.Symbols {
		// Validate configuration
		if err := c.validateSymbolConfig(&symCfg); err != nil {
			return fmt.Errorf("invalid config for %s: %w", key, err)
		}

		// Store with lowercase key for consistency
		lowerKey := strings.ToLower(key)
		c.symbols[lowerKey] = &pricer.SymbolConfig{
			Symbol:         symCfg.Symbol,
			CommissionRate: symCfg.CommissionRate,
			MaxPayout:      symCfg.MaxPayout,
			MinStake:       symCfg.MinStake,
			Enabled:        symCfg.Enabled,
		}
	}

	return nil
}

// validateSymbolConfig validates a symbol configuration.
func (c *Config) validateSymbolConfig(cfg *symbolConfigYAML) error {
	if cfg.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}

	if cfg.CommissionRate < 0 || cfg.CommissionRate > 1 {
		return fmt.Errorf("commission_rate must be between 0 and 1")
	}

	if cfg.MaxPayout <= 0 {
		return fmt.Errorf("max_payout must be positive")
	}

	if cfg.MinStake <= 0 {
		return fmt.Errorf("min_stake must be positive")
	}

	return nil
}

// GetSymbolConfig returns configuration for a specific symbol.
// Note: Viper lowercases all map keys, so we look up with lowercase.
func (c *Config) GetSymbolConfig(symbol string) (*pricer.SymbolConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Viper lowercases all map keys, so we look up with lowercase
	lookupKey := strings.ToLower(symbol)
	cfg, ok := c.symbols[lookupKey]
	if !ok {
		return nil, fmt.Errorf("symbol %s: %w", symbol, pricer.ErrInvalidSymbol)
	}

	if !cfg.Enabled {
		return nil, fmt.Errorf("symbol %s: %w", symbol, pricer.ErrSymbolDisabled)
	}

	return cfg, nil
}

// Reload reloads configuration from the file.
func (c *Config) Reload() error {
	if err := c.v.ReadInConfig(); err != nil {
		return fmt.Errorf("reload config: %w", err)
	}

	return c.load()
}

// Watch watches the configuration file for changes and reloads automatically.
func (c *Config) Watch(onChange func()) {
	c.v.WatchConfig()
	c.v.OnConfigChange(func(e fsnotify.Event) {
		if err := c.load(); err != nil {
			// Log error but don't crash - keep using existing config
			fmt.Printf("error reloading config: %v\n", err)
			return
		}
		if onChange != nil {
			onChange()
		}
	})
}
