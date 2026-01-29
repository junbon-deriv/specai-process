package trading

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/common"
	"github.com/deriv/arcade/internal/series"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// fakeTradingRepository is a fake in-memory implementation of Repository for testing
type fakeTradingRepository struct {
	mu             sync.RWMutex
	priceSeries    map[int64]*PriceSeries
	contracts      map[int64]*Contract
	nextSeriesID   int64
	nextContractID int64
	err            error // For error injection
}

// newFakeTradingRepository creates a new fake trading repository
func newFakeTradingRepository() *fakeTradingRepository {
	return &fakeTradingRepository{
		priceSeries:    make(map[int64]*PriceSeries),
		contracts:      make(map[int64]*Contract),
		nextSeriesID:   1,
		nextContractID: 1,
	}
}

func (f *fakeTradingRepository) CreatePriceSeries(ctx context.Context, accountID, seriesType string, candles []series.OHLC, quoteValue string) (*PriceSeries, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	seriesID := f.nextSeriesID
	f.nextSeriesID++

	ps := &PriceSeries{
		SeriesID:   seriesID,
		AccountID:  accountID,
		SeriesType: seriesType,
		Candles:    append([]series.OHLC{}, candles...),
		QuoteValue: quoteValue,
		CreatedAt:  time.Now(),
	}

	f.priceSeries[seriesID] = ps
	return ps, nil
}

func (f *fakeTradingRepository) FindPriceSeries(ctx context.Context, accountID, seriesType, quoteValue string) (*PriceSeries, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	for _, ps := range f.priceSeries {
		if ps.AccountID == accountID && ps.SeriesType == seriesType && ps.QuoteValue == quoteValue {
			// Return a copy
			return &PriceSeries{
				SeriesID:   ps.SeriesID,
				AccountID:  ps.AccountID,
				SeriesType: ps.SeriesType,
				Candles:    append([]series.OHLC{}, ps.Candles...),
				QuoteValue: ps.QuoteValue,
				CreatedAt:  ps.CreatedAt,
			}, nil
		}
	}

	return nil, common.ErrInvalidQuote
}

func (f *fakeTradingRepository) ListContracts(ctx context.Context, accountID string, seriesType *string) ([]Contract, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	var contracts []Contract
	for _, contract := range f.contracts {
		if contract.AccountID == accountID {
			if seriesType == nil || contract.SeriesType == *seriesType {
				contracts = append(contracts, *contract)
			}
		}
	}

	return contracts, nil
}

func (f *fakeTradingRepository) OpenTrade(ctx context.Context, accountID string, seriesID int64, sentiment string, buyPrice decimal.Decimal) (*OpenTradeResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	// Find and delete price series (consumed)
	ps, exists := f.priceSeries[seriesID]
	if !exists {
		return nil, fmt.Errorf("price series not found")
	}

	if ps.AccountID != accountID {
		return nil, fmt.Errorf("price series does not belong to account")
	}

	delete(f.priceSeries, seriesID)

	// Create contract
	contractID := f.nextContractID
	f.nextContractID++

	buyTime := time.Now()
	contract := &Contract{
		ContractID: contractID,
		AccountID:  accountID,
		SeriesType: ps.SeriesType,
		Sentiment:  sentiment,
		BuyPrice:   buyPrice,
		BuyTime:    buyTime,
		BuyOHLCs:   append([]series.OHLC{}, ps.Candles...),
	}

	f.contracts[contractID] = contract

	return &OpenTradeResult{
		ContractID:       contractID,
		BuyTransactionID: contractID, // Simplified for fake
		NewBalance:       decimal.Zero,
		BuyOHLCs:         ps.Candles,
		BuyTime:          buyTime,
	}, nil
}

func (f *fakeTradingRepository) CloseTrade(ctx context.Context, accountID string, contractID int64, sellPrice decimal.Decimal, sellOHLCs []series.OHLC) (*CloseTradeResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	contract, exists := f.contracts[contractID]
	if !exists {
		return nil, fmt.Errorf("contract not found")
	}

	if contract.AccountID != accountID {
		return nil, fmt.Errorf("contract does not belong to account")
	}

	if contract.SellTime != nil {
		return nil, fmt.Errorf("contract already settled")
	}

	// Update contract with sell details
	sellTime := time.Now()
	contract.SellPrice = &sellPrice
	contract.SellTime = &sellTime
	contract.SellOHLCs = append([]series.OHLC{}, sellOHLCs...)

	return &CloseTradeResult{
		SellTransactionID: contractID, // Simplified for fake
		NewBalance:        decimal.Zero,
		SellTime:          sellTime,
	}, nil
}

// fakeAccountRepository for testing trading service
type fakeAccountRepository struct {
	accounts map[string]*accounts.Account
	mu       sync.RWMutex
}

func newFakeAccountRepository() *fakeAccountRepository {
	return &fakeAccountRepository{
		accounts: make(map[string]*accounts.Account),
	}
}

func (f *fakeAccountRepository) CreateAccount(ctx context.Context, currency string, externalID *string) (*accounts.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	accountID := fmt.Sprintf("SW%d", len(f.accounts)+1)
	account := &accounts.Account{
		AccountID: accountID,
		Currency:  currency,
		Balance:   decimal.Zero,
	}
	f.accounts[accountID] = account
	return account, nil
}

func (f *fakeAccountRepository) GetAccount(ctx context.Context, accountID string) (*accounts.Account, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	account, exists := f.accounts[accountID]
	if !exists {
		return nil, common.ErrAccountNotFound
	}
	return account, nil
}

func (f *fakeAccountRepository) DepositFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*accounts.DepositResult, error) {
	return nil, nil
}

func (f *fakeAccountRepository) WithdrawFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*accounts.WithdrawalResult, error) {
	return nil, nil
}

// fakeSeriesRepository for testing trading service
type fakeSeriesRepository struct{}

func (f *fakeSeriesRepository) GetSeriesConfig(ctx context.Context, seriesType string) (*series.SeriesConfig, error) {
	return &series.SeriesConfig{
		DisplayName:      "Volatility 50",
		InitialValue:     decimal.NewFromInt(1000),
		Volatility:       0.50,
		Drift:            0.0,
		IntervalSeconds:  1,
		PayoutMultiplier: decimal.NewFromFloat(1.8868),
		GeneratorType:    "gbm",
	}, nil
}

func (f *fakeSeriesRepository) ListActiveSeries(ctx context.Context) ([]series.SeriesType, error) {
	return nil, nil
}

// TestGeneratePreview_Success tests successful preview generation
func TestGeneratePreview_Success(t *testing.T) {
	tradingRepo := newFakeTradingRepository()
	accountRepo := newFakeAccountRepository()
	seriesRepo := &fakeSeriesRepository{}

	accountService := accounts.NewService(accountRepo)
	seriesService := series.NewService(seriesRepo)
	tradingService := NewService(tradingRepo, accountService, seriesService)

	// Create account first
	account, _ := accountService.CreateAccount(context.Background(), accounts.CreateAccountRequest{Currency: "USD"})

	result, err := tradingService.GeneratePreview(context.Background(), account.AccountID, "Vol50")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.OHLCs) != 10 {
		t.Errorf("Expected 10 candles, got %d", len(result.OHLCs))
	}

	// Verify price series was stored
	tradingRepo.mu.RLock()
	defer tradingRepo.mu.RUnlock()

	if len(tradingRepo.priceSeries) != 1 {
		t.Errorf("Expected 1 price series, got %d", len(tradingRepo.priceSeries))
	}
}

// TestGeneratePreview_InvalidSeriesType tests preview with invalid series type
func TestGeneratePreview_InvalidSeriesType(t *testing.T) {
	tradingRepo := newFakeTradingRepository()
	accountRepo := newFakeAccountRepository()
	seriesRepo := &fakeSeriesRepository{}

	accountService := accounts.NewService(accountRepo)
	seriesService := series.NewService(seriesRepo)
	tradingService := NewService(tradingRepo, accountService, seriesService)

	_, err := tradingService.GeneratePreview(context.Background(), "SW123", "INVALID")

	if err == nil {
		t.Fatal("Expected error for invalid series type")
	}

	if err != common.ErrInvalidSeriesType {
		t.Errorf("Expected ErrInvalidSeriesType, got %v", err)
	}
}

// TestGeneratePreview_AccountNotFound tests preview when account doesn't exist
func TestGeneratePreview_AccountNotFound(t *testing.T) {
	tradingRepo := newFakeTradingRepository()
	accountRepo := newFakeAccountRepository()
	seriesRepo := &fakeSeriesRepository{}

	accountService := accounts.NewService(accountRepo)
	seriesService := series.NewService(seriesRepo)
	tradingService := NewService(tradingRepo, accountService, seriesService)

	_, err := tradingService.GeneratePreview(context.Background(), "SW999", "Vol50")

	if err != common.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

// TestListContracts_Success tests successful contract listing
func TestListContracts_Success(t *testing.T) {
	tradingRepo := newFakeTradingRepository()
	accountRepo := newFakeAccountRepository()
	seriesRepo := &fakeSeriesRepository{}

	accountService := accounts.NewService(accountRepo)
	seriesService := series.NewService(seriesRepo)
	tradingService := NewService(tradingRepo, accountService, seriesService)

	// Create account
	account, _ := accountService.CreateAccount(context.Background(), accounts.CreateAccountRequest{Currency: "USD"})

	// Manually add a contract to fake repo
	tradingRepo.mu.Lock()
	tradingRepo.contracts[1] = &Contract{
		ContractID: 1,
		AccountID:  account.AccountID,
		SeriesType: "Vol50",
		Sentiment:  "rise",
		BuyPrice:   decimal.NewFromInt(10),
		BuyTime:    time.Now(),
		BuyOHLCs:   []series.OHLC{},
	}
	tradingRepo.mu.Unlock()

	result, err := tradingService.ListContracts(context.Background(), account.AccountID, nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Contracts) != 1 {
		t.Errorf("Expected 1 contract, got %d", len(result.Contracts))
	}

	if result.Contracts[0].ContractID != 1 {
		t.Errorf("Expected contract ID 1, got %d", result.Contracts[0].ContractID)
	}
}

// TestListContracts_FilteredBySeriesType tests contract listing filtered by series type
func TestListContracts_FilteredBySeriesType(t *testing.T) {
	tradingRepo := newFakeTradingRepository()
	accountRepo := newFakeAccountRepository()
	seriesRepo := &fakeSeriesRepository{}

	accountService := accounts.NewService(accountRepo)
	seriesService := series.NewService(seriesRepo)
	tradingService := NewService(tradingRepo, accountService, seriesService)

	account, _ := accountService.CreateAccount(context.Background(), accounts.CreateAccountRequest{Currency: "USD"})

	// Add contracts with different series types
	tradingRepo.mu.Lock()
	tradingRepo.contracts[1] = &Contract{ContractID: 1, AccountID: account.AccountID, SeriesType: "Vol50", BuyTime: time.Now()}
	tradingRepo.contracts[2] = &Contract{ContractID: 2, AccountID: account.AccountID, SeriesType: "Vol100", BuyTime: time.Now()}
	tradingRepo.mu.Unlock()

	seriesType := "Vol50"
	result, err := tradingService.ListContracts(context.Background(), account.AccountID, &seriesType)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Contracts) != 1 {
		t.Errorf("Expected 1 contract, got %d", len(result.Contracts))
	}

	if result.Contracts[0].SeriesType != "Vol50" {
		t.Errorf("Expected Vol50, got %s", result.Contracts[0].SeriesType)
	}
}

// TestEvaluateOutcome tests outcome evaluation logic
func TestEvaluateOutcome(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name      string
		sentiment string
		buyPrice  decimal.Decimal
		sellPrice decimal.Decimal
		expected  bool
	}{
		{
			name:      "Rise win",
			sentiment: "rise",
			buyPrice:  decimal.NewFromInt(1000),
			sellPrice: decimal.NewFromInt(1010),
			expected:  true,
		},
		{
			name:      "Rise loss",
			sentiment: "rise",
			buyPrice:  decimal.NewFromInt(1000),
			sellPrice: decimal.NewFromInt(990),
			expected:  false,
		},
		{
			name:      "Fall win",
			sentiment: "fall",
			buyPrice:  decimal.NewFromInt(1000),
			sellPrice: decimal.NewFromInt(990),
			expected:  true,
		},
		{
			name:      "Fall loss",
			sentiment: "fall",
			buyPrice:  decimal.NewFromInt(1000),
			sellPrice: decimal.NewFromInt(1010),
			expected:  false,
		},
		{
			name:      "Invalid sentiment",
			sentiment: "invalid",
			buyPrice:  decimal.NewFromInt(1000),
			sellPrice: decimal.NewFromInt(1010),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateOutcome(tt.sentiment, tt.buyPrice, tt.sellPrice)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestOpenTrade_Success tests successful trade opening
func TestOpenTrade_Success(t *testing.T) {
	tradingRepo := newFakeTradingRepository()
	accountRepo := newFakeAccountRepository()
	account, _ := accountRepo.CreateAccount(context.Background(), "USD", nil)

	// Create price series
	candles := make([]series.OHLC, 10)
	for i := 0; i < 10; i++ {
		candles[i] = series.OHLC{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Close:     decimal.NewFromInt(1000 + int64(i)),
		}
	}
	ps, _ := tradingRepo.CreatePriceSeries(context.Background(), account.AccountID, "Vol50", candles, "1009.00")

	// Open trade
	result, err := tradingRepo.OpenTrade(context.Background(), account.AccountID, ps.SeriesID, "rise", decimal.NewFromInt(10))

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ContractID == 0 {
		t.Error("Expected contract ID to be set")
	}

	if len(result.BuyOHLCs) != 10 {
		t.Errorf("Expected 10 buy candles, got %d", len(result.BuyOHLCs))
	}

	// Verify price series was deleted
	tradingRepo.mu.RLock()
	_, exists := tradingRepo.priceSeries[ps.SeriesID]
	tradingRepo.mu.RUnlock()

	if exists {
		t.Error("Expected price series to be deleted after open trade")
	}

	// Verify contract was created
	tradingRepo.mu.RLock()
	contract, exists := tradingRepo.contracts[result.ContractID]
	tradingRepo.mu.RUnlock()

	if !exists {
		t.Error("Expected contract to be created")
	}

	if contract.BuyPrice.Cmp(decimal.NewFromInt(10)) != 0 {
		t.Errorf("Expected buy price 10, got %v", contract.BuyPrice)
	}

	if contract.Sentiment != "rise" {
		t.Errorf("Expected sentiment rise, got %s", contract.Sentiment)
	}
}

// TestCloseTrade_Success tests successful trade closing
func TestCloseTrade_Success(t *testing.T) {
	tradingRepo := newFakeTradingRepository()
	accountRepo := newFakeAccountRepository()

	accountService := accounts.NewService(accountRepo)

	account, _ := accountService.CreateAccount(context.Background(), accounts.CreateAccountRequest{Currency: "USD"})

	// Create open contract manually
	tradingRepo.mu.Lock()
	tradingRepo.contracts[1] = &Contract{
		ContractID: 1,
		AccountID:  account.AccountID,
		SeriesType: "Vol50",
		Sentiment:  "rise",
		BuyPrice:   decimal.NewFromInt(10),
		BuyTime:    time.Now(),
		BuyOHLCs:   []series.OHLC{},
	}
	tradingRepo.mu.Unlock()

	// Close trade
	sellCandles := make([]series.OHLC, 10)
	_, err := tradingRepo.CloseTrade(context.Background(), account.AccountID, 1, decimal.NewFromInt(18), sellCandles)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify contract was updated
	tradingRepo.mu.RLock()
	contract := tradingRepo.contracts[1]
	tradingRepo.mu.RUnlock()

	if contract.SellTime == nil {
		t.Error("Expected sell time to be set")
	}

	if contract.SellPrice == nil || contract.SellPrice.Cmp(decimal.NewFromInt(18)) != 0 {
		t.Errorf("Expected sell price 18, got %v", contract.SellPrice)
	}

	if len(contract.SellOHLCs) != 10 {
		t.Errorf("Expected 10 sell candles, got %d", len(contract.SellOHLCs))
	}
}

// TestCloseTrade_ContractAlreadySettled tests closing an already settled contract
func TestCloseTrade_ContractAlreadySettled(t *testing.T) {
	tradingRepo := newFakeTradingRepository()
	accountRepo := newFakeAccountRepository()
	account, _ := accountRepo.CreateAccount(context.Background(), "USD", nil)

	// Create settled contract
	sellTime := time.Now()
	sellPrice := decimal.NewFromInt(18)
	tradingRepo.mu.Lock()
	tradingRepo.contracts[1] = &Contract{
		ContractID: 1,
		AccountID:  account.AccountID,
		SellTime:   &sellTime,
		SellPrice:  &sellPrice,
	}
	tradingRepo.mu.Unlock()

	// Try to close again
	_, err := tradingRepo.CloseTrade(context.Background(), account.AccountID, 1, decimal.NewFromInt(20), []series.OHLC{})

	if err == nil {
		t.Fatal("Expected error when closing already settled contract")
	}
}
