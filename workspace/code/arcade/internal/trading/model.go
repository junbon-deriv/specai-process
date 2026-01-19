package trading

import (
	"time"

	"github.com/shopspring/decimal"
)

// OHLC represents a single candlestick
type OHLC struct {
	Timestamp time.Time       `json:"timestamp"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
}

// PriceSeries represents a temporary preview series
type PriceSeries struct {
	SeriesID   int64     `json:"series_id"`
	AccountID  string    `json:"account_id"`
	SeriesType string    `json:"series_type"`
	Candles    []OHLC    `json:"candles"`
	QuoteValue string    `json:"quote_value"`
	CreatedAt  time.Time `json:"created_at"`
}

// Contract represents a trading contract
type Contract struct {
	ContractID int64  `json:"contract_id"`
	AccountID  string `json:"account_id"`
	SeriesType string `json:"series_type"`
	Sentiment  string `json:"sentiment"`
	
	// Buy details (populated at open_trade)
	BuyPrice decimal.Decimal `json:"buy_price"` // Stake amount
	BuyTime  time.Time       `json:"buy_time"`
	BuyOHLCs []OHLC          `json:"buy_ohlcs"` // Candles 1-10
	
	// Sell details (NULL = OPEN, populated at close_trade)
	SellPrice *decimal.Decimal `json:"sell_price,omitempty"` // Payout amount
	SellTime  *time.Time       `json:"sell_time,omitempty"`
	SellOHLCs []OHLC           `json:"sell_ohlcs,omitempty"` // Candles 11-20
}

// SwipeGetRequest represents SwipeGet input
type SwipeGetRequest struct {
	SeriesType string `json:"series_type"`
	AccountID  string `json:"account_id"`
}

// SwipeGetResponse represents SwipeGet output
type SwipeGetResponse struct {
	OHLCs []OHLC `json:"ohlcs"`
}

// SwipeBuyRequest represents SwipeBuy input
type SwipeBuyRequest struct {
	AccountID     string `json:"account_id"`
	Stake         string `json:"stake"`
	SeriesType    string `json:"series_type"`
	PreviousQuote string `json:"previous_quote"`
	Sentiment     string `json:"sentiment"`
}

// SwipeBuyResponse represents SwipeBuy output
type SwipeBuyResponse struct {
	ContractID   int64     `json:"contract_id"`
	PurchaseTime time.Time `json:"purchase_time"`
	OHLCs        []OHLC    `json:"ohlcs"`
	Payout       string    `json:"payout"`
}

// SwipeListRequest represents SwipeList input
type SwipeListRequest struct {
	AccountID  string  `json:"account_id"`
	SeriesType *string `json:"series_type,omitempty"`
}

// SwipeListResponse represents SwipeList output
type SwipeListResponse struct {
	Contracts []Contract `json:"contracts"`
}

// SeriesConfig holds configuration for a series type
type SeriesConfig struct {
	InitialValue decimal.Decimal
	Volatility   float64
	InterestRate float64
	QuantoDrift  float64
	Interval     time.Duration
	Precision    int32
}

// GetSeriesConfig returns configuration for a series type
func GetSeriesConfig(seriesType string) SeriesConfig {
	configs := map[string]SeriesConfig{
		"Vol50": {
			InitialValue: decimal.NewFromInt(10000),
			Volatility:   0.50,
			InterestRate: 0.0,
			QuantoDrift:  0.0,
			Interval:     1 * time.Second,
			Precision:    3,
		},
		"Vol100": {
			InitialValue: decimal.NewFromInt(50000),
			Volatility:   1.00,
			InterestRate: 0.0,
			QuantoDrift:  0.0,
			Interval:     1 * time.Second,
			Precision:    3,
		},
		"Vol200": {
			InitialValue: decimal.NewFromInt(100000),
			Volatility:   2.00,
			InterestRate: 0.0,
			QuantoDrift:  0.0,
			Interval:     1 * time.Second,
			Precision:    3,
		},
		"Vol300": {
			InitialValue: decimal.NewFromInt(200000),
			Volatility:   3.00,
			InterestRate: 0.0,
			QuantoDrift:  0.0,
			Interval:     1 * time.Second,
			Precision:    3,
		},
	}
	return configs[seriesType]
}
