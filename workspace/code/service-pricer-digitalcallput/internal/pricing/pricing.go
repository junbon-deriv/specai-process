package pricing

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
)

var (
	// ErrStakeBelowMinimum indicates the stake is below the configured minimum
	ErrStakeBelowMinimum = errors.New("stake below minimum")
	// ErrPayoutExceedsMaximum indicates the calculated payout exceeds the configured maximum
	ErrPayoutExceedsMaximum = errors.New("payout exceeds maximum")
	// ErrUnknownSymbol indicates the symbol is not configured
	ErrUnknownSymbol = errors.New("unknown symbol")
	// ErrInvalidDuration indicates the duration format is invalid
	ErrInvalidDuration = errors.New("invalid duration format")
	// ErrInvalidBarrier indicates the barrier format is invalid
	ErrInvalidBarrier = errors.New("invalid barrier format")
	// ErrMarketUnavailable indicates market data is not available
	ErrMarketUnavailable = errors.New("market data unavailable")
)

// ContractType represents the type of contract (Call or Put)
type ContractType int

const (
	// ContractTypeCall represents a call option
	ContractTypeCall ContractType = iota + 1
	// ContractTypePut represents a put option
	ContractTypePut
)

// Tick represents a single price point from the market
type Tick struct {
	Symbol    string
	Price     decimal.Decimal
	Timestamp int64
}

// SymbolConfig holds the configuration for a symbol
type SymbolConfig struct {
	Symbol     string
	MinStake   decimal.Decimal
	MaxPayout  decimal.Decimal
	Commission decimal.Decimal
}

// Limits represents trading limits for a symbol
type Limits struct {
	MaxPayout decimal.Decimal
	MinStake  decimal.Decimal
}

// MarketDataProvider defines the interface for market data access
// This interface is defined in pricing (where it's consumed) per architectural rules
type MarketDataProvider interface {
	GetLatestTick(ctx context.Context, symbol string) (*Tick, error)
	GetTick(ctx context.Context, symbol string, timestamp int64) (*Tick, error)
	GetTicksInRange(ctx context.Context, symbol string, from, to int64) ([]*Tick, error)
	StreamTicks(ctx context.Context, symbol string) (<-chan *Tick, error)
}

// ConfigProvider defines the interface for configuration access
// This interface is defined in pricing (where it's consumed) per architectural rules
type ConfigProvider interface {
	GetSymbolConfig(symbol string) (*SymbolConfig, error)
}

// AskInput contains all parameters for Ask price calculation
type AskInput struct {
	Symbol       string
	ContractType ContractType
	Currency     string
	Duration     string
	Stake        string
	Barrier      *string
	PricingTime  *int64
}

// AskResult contains the Ask calculation output
type AskResult struct {
	AskPrice        string
	Currency        string
	CurrentSpot     string
	CurrentSpotTime int64
	Payout          string
	Limits          Limits
}

// BidInput contains all parameters for Bid price calculation
type BidInput struct {
	Symbol       string
	ContractType ContractType
	Currency     string
	Duration     string
	Barrier      *string
	StartTime    int64
	Payout       string // REQUIRED - never recalculated
}

// BidResult contains the Bid calculation output
type BidResult struct {
	BidPrice        string
	IsExpired       bool
	CurrentSpot     string
	CurrentSpotTime int64
	EntrySpot       string
	EntrySpotTime   int64
	ExitSpot        *string
	ExitSpotTime    *int64
	Barrier         string
	StartTime       int64
	ExpiryTime      int64
	Currency        string
}

// Pricer is the main interface for price calculations
type Pricer interface {
	CalculateAsk(ctx context.Context, input AskInput) (*AskResult, error)
	CalculateBid(ctx context.Context, input BidInput) (*BidResult, error)
}

// pricer implements the Pricer interface
type pricer struct {
	market       MarketDataProvider
	config       ConfigProvider
	blackScholes *BlackScholes
}

// NewPricer creates a new Pricer instance
func NewPricer(market MarketDataProvider, config ConfigProvider) Pricer {
	return &pricer{
		market: market,
		config: config,
		blackScholes: &BlackScholes{
			volatility: 0.10, // 10%
			rate:       0.00, // 0%
		},
	}
}
