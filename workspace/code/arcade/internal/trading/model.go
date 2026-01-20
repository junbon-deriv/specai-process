package trading

import (
	"time"

	"github.com/deriv/arcade/internal/series"
	"github.com/shopspring/decimal"
)

// PriceSeries represents a temporary preview series
type PriceSeries struct {
	SeriesID   int64         `json:"series_id"`
	AccountID  string        `json:"account_id"`
	SeriesType string        `json:"series_type"`
	Candles    []series.OHLC `json:"candles"`
	QuoteValue string        `json:"quote_value"`
	CreatedAt  time.Time     `json:"created_at"`
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
	BuyOHLCs []series.OHLC   `json:"buy_ohlcs"` // Candles 1-10

	// Sell details (NULL = OPEN, populated at close_trade)
	SellPrice *decimal.Decimal `json:"sell_price,omitempty"` // Payout amount
	SellTime  *time.Time       `json:"sell_time,omitempty"`
	SellOHLCs []series.OHLC    `json:"sell_ohlcs,omitempty"` // Candles 11-20
}

// SwipeGetRequest represents SwipeGet input
type SwipeGetRequest struct {
	SeriesType string `json:"series_type"`
	AccountID  string `json:"account_id"`
}

// SwipeGetResponse represents SwipeGet output
type SwipeGetResponse struct {
	OHLCs []series.OHLC `json:"ohlcs"`
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
	ContractID   int64         `json:"contract_id"`
	PurchaseTime time.Time     `json:"purchase_time"`
	OHLCs        []series.OHLC `json:"ohlcs"`
	Payout       string        `json:"payout"`
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
