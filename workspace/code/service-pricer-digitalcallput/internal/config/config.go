package config

import (
	"fmt"
	"os"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// symbolFileConfig represents the YAML file structure
type symbolFileConfig struct {
	Symbols []struct {
		Symbol     string `yaml:"symbol"`
		MinStake   string `yaml:"min_stake"`
		MaxPayout  string `yaml:"max_payout"`
		Commission string `yaml:"commission"`
	} `yaml:"symbols"`
}

// Config holds all symbol configurations
type Config struct {
	symbols map[string]*pricing.SymbolConfig
}

// Load loads symbol configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var fileConfig symbolFileConfig
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg := &Config{
		symbols: make(map[string]*pricing.SymbolConfig),
	}

	for _, s := range fileConfig.Symbols {
		minStake, err := decimal.NewFromString(s.MinStake)
		if err != nil {
			return nil, fmt.Errorf("invalid min_stake for symbol %s: %w", s.Symbol, err)
		}

		maxPayout, err := decimal.NewFromString(s.MaxPayout)
		if err != nil {
			return nil, fmt.Errorf("invalid max_payout for symbol %s: %w", s.Symbol, err)
		}

		commission, err := decimal.NewFromString(s.Commission)
		if err != nil {
			return nil, fmt.Errorf("invalid commission for symbol %s: %w", s.Symbol, err)
		}

		cfg.symbols[s.Symbol] = &pricing.SymbolConfig{
			Symbol:     s.Symbol,
			MinStake:   minStake,
			MaxPayout:  maxPayout,
			Commission: commission,
		}
	}

	return cfg, nil
}

// GetSymbolConfig returns configuration for a given symbol
func (c *Config) GetSymbolConfig(symbol string) (*pricing.SymbolConfig, error) {
	cfg, ok := c.symbols[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol not found: %s", symbol)
	}
	return cfg, nil
}
