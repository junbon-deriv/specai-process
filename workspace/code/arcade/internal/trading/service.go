package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/common"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Service implements trading business logic
type Service struct {
	repo           *Repository
	accountService *accounts.Service
	gbmGenerator   *GBMGenerator
	pool           *pgxpool.Pool
}

// NewService creates a new trading service
func NewService(pool *pgxpool.Pool, accountService *accounts.Service) *Service {
	return &Service{
		repo:           NewRepository(pool),
		accountService: accountService,
		gbmGenerator:   NewGBMGenerator(),
		pool:           pool,
	}
}

// GeneratePreview creates a 10-candle OHLC preview series (SwipeGet)
func (s *Service) GeneratePreview(ctx context.Context, accountID, seriesType string) (*SwipeGetResponse, error) {
	// Validate series type
	if err := common.ValidateSeriesType(seriesType); err != nil {
		return nil, err
	}

	// Validate account exists
	if _, err := s.accountService.GetAccount(ctx, accountID); err != nil {
		return nil, err
	}

	// Get series configuration
	config := GetSeriesConfig(seriesType)

	// Generate 10 candles from initial value
	startTime := time.Now().UTC()
	candles := s.gbmGenerator.GenerateCandles(config.InitialValue, config, 10, startTime)

	// Store price series with 10th candle close as quote value
	quoteValue := common.FormatPrice(candles[9].Close)
	_, err := s.repo.CreatePriceSeries(ctx, accountID, seriesType, candles, quoteValue)
	if err != nil {
		return nil, fmt.Errorf("failed to store price series: %w", err)
	}

	return &SwipeGetResponse{
		OHLCs: candles,
	}, nil
}

// ExecuteTrade places a rise/fall binary option with immediate settlement (SwipeBuy)
// Uses stored procedures for atomic operations
func (s *Service) ExecuteTrade(ctx context.Context, req SwipeBuyRequest) (*SwipeBuyResponse, error) {
	// Validate inputs
	if err := common.ValidateSeriesType(req.SeriesType); err != nil {
		return nil, err
	}
	if err := common.ValidateSentiment(req.Sentiment); err != nil {
		return nil, err
	}

	// Parse stake (buy_price)
	buyPrice, err := common.ParseAmount(req.Stake)
	if err != nil {
		return nil, common.NewAPIError(common.ErrCodeInvalidStake, "Invalid stake amount format")
	}
	if err := common.ValidatePositiveAmount(buyPrice); err != nil {
		return nil, common.NewAPIError(common.ErrCodeInvalidStake, "Stake must be positive")
	}

	// Find price series by quote value to get series_id
	priceSeries, err := s.repo.FindPriceSeries(ctx, req.AccountID, req.SeriesType, req.PreviousQuote)
	if err != nil {
		return nil, common.NewAPIError(common.ErrCodeInvalidQuote, "Quote does not match any active preview")
	}

	// Phase 1: Call open_trade stored procedure to buy contract
	var contractID int64
	var buyTxnID int64
	var newBalance decimal.Decimal
	var buyOHLCsJSON []byte
	var buyTime time.Time

	err = s.pool.QueryRow(ctx, `
		SELECT contract_id, buy_transaction_id, new_balance, buy_ohlcs, buy_time
		FROM open_trade($1, $2, $3, $4)
	`, req.AccountID, priceSeries.SeriesID, req.Sentiment, buyPrice).Scan(
		&contractID,
		&buyTxnID,
		&newBalance,
		&buyOHLCsJSON,
		&buyTime,
	)
	if err != nil {
		return nil, s.mapPgError(err)
	}

	// Unmarshal buy candles
	var buyCandles []OHLC
	if err := json.Unmarshal(buyOHLCsJSON, &buyCandles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal buy candles: %w", err)
	}

	// Phase 2: Generate execution candles and settle immediately
	config := GetSeriesConfig(req.SeriesType)
	lastCandle := buyCandles[9]
	startTime := lastCandle.Timestamp.Add(config.Interval)
	sellCandles := s.gbmGenerator.GenerateCandles(lastCandle.Close, config, 10, startTime)

	// Evaluate outcome
	buyPriceValue := buyCandles[9].Close   // Entry price (10th candle close)
	sellPriceValue := sellCandles[9].Close // Exit price (20th candle close)
	isWin := s.evaluateOutcome(req.Sentiment, buyPriceValue, sellPriceValue)

	// Calculate payout (sell_price)
	var sellPrice decimal.Decimal
	if isWin {
		// Payout = buy_price / 0.53
		sellPrice = buyPrice.Div(decimal.NewFromFloat(0.53)).Round(2)
	} else {
		sellPrice = decimal.Zero
	}

	// Marshal sell candles
	sellCandlesJSON, err := json.Marshal(sellCandles)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sell candles: %w", err)
	}

	// Call close_trade stored procedure to settle contract
	var sellTxnID int64
	var finalBalance decimal.Decimal
	var sellTime time.Time

	err = s.pool.QueryRow(ctx, `
		SELECT sell_transaction_id, new_balance, sell_time
		FROM close_trade($1, $2, $3, $4)
	`, req.AccountID, contractID, sellPrice, sellCandlesJSON).Scan(
		&sellTxnID,
		&finalBalance,
		&sellTime,
	)
	if err != nil {
		return nil, s.mapPgError(err)
	}

	// Return execution candles (11-20) and payout
	return &SwipeBuyResponse{
		ContractID:   contractID,
		PurchaseTime: buyTime,
		OHLCs:        sellCandles,
		Payout:       common.FormatAmount(sellPrice),
	}, nil
}

// ListContracts retrieves trading history (SwipeList)
func (s *Service) ListContracts(ctx context.Context, accountID string, seriesType *string) (*SwipeListResponse, error) {
	// Validate account exists
	if _, err := s.accountService.GetAccount(ctx, accountID); err != nil {
		return nil, err
	}

	// Validate series type if provided
	if seriesType != nil {
		if err := common.ValidateSeriesType(*seriesType); err != nil {
			return nil, err
		}
	}

	// Retrieve contracts
	contracts, err := s.repo.ListContracts(ctx, accountID, seriesType)
	if err != nil {
		return nil, err
	}

	return &SwipeListResponse{
		Contracts: contracts,
	}, nil
}

// evaluateOutcome determines if the trade was a win
func (s *Service) evaluateOutcome(sentiment string, candle10Close, candle20Close decimal.Decimal) bool {
	switch sentiment {
	case "rise":
		return candle20Close.GreaterThan(candle10Close)
	case "fall":
		return candle20Close.LessThan(candle10Close)
	default:
		return false
	}
}

// mapPgError converts PostgreSQL error codes to API errors
func (s *Service) mapPgError(err error) error {
	// TODO: Parse PostgreSQL error codes and map to common errors
	// P0001 - Account not found
	// P0002 - Insufficient balance
	// P0003 - Invalid amount
	// P0004 - Invalid sentiment
	// P0005 - Price series not found
	// P0006 - Price series account mismatch
	// P0007 - Series type not active
	// P0008 - Contract not found
	// P0009 - Contract account mismatch
	// P0010 - Contract already settled
	
	errMsg := err.Error()
	if contains(errMsg, "Account not found") || contains(errMsg, "P0001") {
		return common.ErrAccountNotFound
	}
	if contains(errMsg, "Insufficient balance") || contains(errMsg, "P0002") {
		return common.ErrInsufficientBalance
	}
	if contains(errMsg, "Price series") || contains(errMsg, "P0005") {
		return common.ErrInvalidQuote
	}
	
	return fmt.Errorf("database error: %w", err)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
