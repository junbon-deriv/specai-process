package trading

import (
	"context"
	"fmt"
	"time"

	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/common"
	"github.com/deriv/arcade/internal/series"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OpenTradeResult represents result from open_trade stored procedure
type OpenTradeResult struct {
	ContractID       int64
	BuyTransactionID int64
	NewBalance       decimal.Decimal
	BuyOHLCs         []series.OHLC
	BuyTime          time.Time
}

// CloseTradeResult represents result from close_trade stored procedure
type CloseTradeResult struct {
	SellTransactionID int64
	NewBalance        decimal.Decimal
	SellTime          time.Time
}

// Repository interface for trading data operations
type Repository interface {
	// CreatePriceSeries stores a price preview series
	CreatePriceSeries(ctx context.Context, accountID, seriesType string, candles []series.OHLC, quoteValue string) (*PriceSeries, error)

	// GetPriceSeries finds price series by ID
	GetPriceSeries(ctx context.Context, seriesID uuid.UUID) (*PriceSeries, error)

	// GetContract retrieves a single contract by ID
	GetContract(ctx context.Context, contractID int64) (*Contract, error)

	// ListContracts retrieves contracts with optional series filter
	ListContracts(ctx context.Context, accountID string, seriesType *string) ([]Contract, error)

	// OpenTrade calls open_trade stored procedure
	OpenTrade(ctx context.Context, accountID string, seriesID uuid.UUID, sentiment string, buyPrice decimal.Decimal) (*OpenTradeResult, error)

	// CloseTrade calls close_trade stored procedure
	CloseTrade(ctx context.Context, accountID string, contractID int64, sellPrice decimal.Decimal, sellOHLCs []series.OHLC) (*CloseTradeResult, error)
}

// Service implements trading business logic
type Service struct {
	repo           Repository
	accountService *accounts.Service
	seriesService  *series.Service
}

// NewService creates a new trading service
func NewService(repo Repository, accountService *accounts.Service, seriesService *series.Service) *Service {
	return &Service{
		repo:           repo,
		accountService: accountService,
		seriesService:  seriesService,
	}
}

// GeneratePreview creates preview candles for trading decision (SwipeGet)
func (s *Service) GeneratePreview(ctx context.Context, accountID, seriesType string) (*SwipeGetResponse, error) {
	// Validate series type
	if err := common.ValidateSeriesType(seriesType); err != nil {
		return nil, err
	}

	// Validate account exists
	if _, err := s.accountService.GetAccount(ctx, accountID); err != nil {
		return nil, err
	}

	// Get series configuration from database
	config, err := s.seriesService.GetConfig(ctx, seriesType)
	if err != nil {
		return nil, err
	}

	// Generate preview candles from initial value using configured count
	startTime := time.Now().UTC()
	candles, err := s.seriesService.GenerateCandles(ctx, seriesType, config.InitialValue, config.PreviewCandles, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to generate candles: %w", err)
	}

	// Store price series with last candle close as quote value
	lastIdx := len(candles) - 1
	quoteValue := common.FormatPrice(candles[lastIdx].Close)
	priceSeries, err := s.repo.CreatePriceSeries(ctx, accountID, seriesType, candles, quoteValue)
	if err != nil {
		return nil, fmt.Errorf("failed to store price series: %w", err)
	}

	return &SwipeGetResponse{
		SeriesID: priceSeries.SeriesID,
		OHLCs:    candles,
	}, nil
}

// ExecuteTrade places a rise/fall binary option with immediate settlement (SwipeBuy)
// Uses stored procedures for atomic operations
func (s *Service) ExecuteTrade(ctx context.Context, req SwipeBuyRequest) (*SwipeBuyResponse, error) {
	// Validate sentiment
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

	// Parse series_id
	seriesID, err := uuid.Parse(req.SeriesID)
	if err != nil {
		return nil, common.NewAPIError(common.ErrCodeInvalidQuote, "Invalid series_id format")
	}

	// Get price series by ID
	priceSeries, err := s.repo.GetPriceSeries(ctx, seriesID)
	if err != nil {
		return nil, common.NewAPIError(common.ErrCodeInvalidQuote, "Price series not found")
	}

	// Verify account owns this series
	if priceSeries.AccountID != req.AccountID {
		return nil, common.NewAPIError(common.ErrCodeInvalidQuote, "Price series does not belong to account")
	}

	// Phase 1: Call open_trade stored procedure to buy contract
	openResult, err := s.repo.OpenTrade(ctx, req.AccountID, seriesID, req.Sentiment, buyPrice)
	if err != nil {
		return nil, err
	}

	// Phase 2: Generate execution candles and settle immediately
	config, err := s.seriesService.GetConfig(ctx, priceSeries.SeriesType)
	if err != nil {
		return nil, err
	}

	// Get last candle from buy OHLCs (last preview candle)
	lastIdx := len(openResult.BuyOHLCs) - 1
	lastCandle := openResult.BuyOHLCs[lastIdx]

	// Calculate start time from config interval
	startTime := lastCandle.Timestamp.Add(time.Duration(config.IntervalSeconds) * time.Second)
	sellCandles, err := s.seriesService.GenerateCandles(ctx, priceSeries.SeriesType, lastCandle.Close, config.ExecutionCandles, startTime)
	if err != nil {
		return nil, err
	}

	// Evaluate outcome
	buyPriceValue := openResult.BuyOHLCs[lastIdx].Close // Entry price (last preview candle close)
	sellLastIdx := len(sellCandles) - 1
	sellPriceValue := sellCandles[sellLastIdx].Close // Exit price (last execution candle close)
	isWin := s.evaluateOutcome(req.Sentiment, buyPriceValue, sellPriceValue)

	// Calculate payout (sell_price) using configured commission
	// Formula: payout = stake / (probability + commission)
	// where probability = 0.5 (50% chance of rise or fall)
	var sellPrice decimal.Decimal
	if isWin {
		probability := decimal.NewFromFloat(0.5)
		divisor := probability.Add(config.Commission) // 0.5 + commission (e.g., 0.5 + 0.03 = 0.53)
		sellPrice = buyPrice.Div(divisor).Round(2)
	} else {
		sellPrice = decimal.Zero
	}

	// Call close_trade stored procedure to settle contract
	_, err = s.repo.CloseTrade(ctx, req.AccountID, openResult.ContractID, sellPrice, sellCandles)
	if err != nil {
		return nil, err
	}

	// Return execution candles (11-20) and payout
	return &SwipeBuyResponse{
		ContractID:   openResult.ContractID,
		PurchaseTime: openResult.BuyTime,
		OHLCs:        sellCandles,
		Payout:       common.FormatAmount(sellPrice),
	}, nil
}

// ListContracts retrieves trading history or a single contract (SwipeContracts)
func (s *Service) ListContracts(ctx context.Context, accountID string, contractID *int64, seriesType *string) (*SwipeListResponse, error) {
	// If contract_id is provided, return single contract
	if contractID != nil {
		contract, err := s.repo.GetContract(ctx, *contractID)
		if err != nil {
			return nil, err
		}

		// Verify contract belongs to account
		if contract.AccountID != accountID {
			return nil, common.ErrAccountNotFound
		}

		return &SwipeListResponse{
			Contracts: []Contract{*contract},
		}, nil
	}

	// Otherwise, list contracts for account
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

// ListInstruments retrieves all active trading instruments (SwipeInstruments)
func (s *Service) ListInstruments(ctx context.Context) (*SwipeInstrumentsResponse, error) {
	// Get all active series types from series service
	seriesTypes, err := s.seriesService.ListActiveSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list series types: %w", err)
	}

	// Convert to instruments
	instruments := make([]Instrument, 0, len(seriesTypes))
	for _, st := range seriesTypes {
		// Parse config to get display name
		config, err := series.ParseConfig(st.Config)
		if err != nil {
			// Skip if config is invalid
			continue
		}

		instruments = append(instruments, Instrument{
			SeriesType:  st.SeriesType,
			DisplayName: config.DisplayName,
		})
	}

	return &SwipeInstrumentsResponse{
		Instruments: instruments,
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
