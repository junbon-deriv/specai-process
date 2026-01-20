package series

import (
	"time"

	"github.com/shopspring/decimal"
)

// SeriesType represents a tradable series configuration
type SeriesType struct {
	SeriesType string                 `json:"series_type"`
	Config     map[string]interface{} `json:"config"`
	IsActive   bool                   `json:"is_active"`
	CreatedAt  time.Time              `json:"created_at"`
}

// SeriesConfig holds parsed configuration for a series type
type SeriesConfig struct {
	DisplayName      string
	InitialValue     decimal.Decimal
	Volatility       float64
	Drift            float64
	IntervalSeconds  int
	PayoutMultiplier decimal.Decimal
	GeneratorType    string // "gbm", "jump_diffusion", etc.
}

// OHLC represents a single candlestick
type OHLC struct {
	Timestamp time.Time       `json:"timestamp"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
}

// ParseConfig parses JSONB config into SeriesConfig
func ParseConfig(configMap map[string]interface{}) (*SeriesConfig, error) {
	getFloat := func(key string, def float64) float64 {
		if v, ok := configMap[key].(float64); ok {
			return v
		}
		return def
	}

	getString := func(key string, def string) string {
		if v, ok := configMap[key].(string); ok {
			return v
		}
		return def
	}

	return &SeriesConfig{
		DisplayName:      getString("display_name", "Unknown"),
		InitialValue:     decimal.NewFromFloat(getFloat("initial_value", 1000.0)),
		Volatility:       getFloat("volatility", 1.0),
		Drift:            getFloat("drift", 0.0),
		IntervalSeconds:  int(getFloat("interval_seconds", 1)),
		PayoutMultiplier: decimal.NewFromFloat(getFloat("payout_multiplier", 1.8868)),
		GeneratorType:    getString("generator_type", "gbm"),
	}, nil
}
