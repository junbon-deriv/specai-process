package pricer

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConfigProvider mocks ConfigProvider interface
type MockConfigProvider struct {
	mock.Mock
}

func (m *MockConfigProvider) GetSymbolConfig(symbol string) (*SymbolConfig, error) {
	args := m.Called(symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SymbolConfig), args.Error(1)
}

// MockFeedProvider mocks FeedProvider interface
type MockFeedProvider struct {
	mock.Mock
}

func (m *MockFeedProvider) GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error) {
	args := m.Called(ctx, symbol, epoch)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*Tick), args.Bool(1), args.Error(2)
}

func (m *MockFeedProvider) GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*Tick, bool, error) {
	args := m.Called(ctx, symbol, start, limit)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]*Tick), args.Bool(1), args.Error(2)
}

func (m *MockFeedProvider) Subscribe(ctx context.Context, symbol string, start int64) *Subscription {
	args := m.Called(ctx, symbol, start)
	return args.Get(0).(*Subscription)
}

func (m *MockFeedProvider) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockContractValidator mocks ContractValidator interface
type MockContractValidator struct {
	mock.Mock
}

func (m *MockContractValidator) ValidateAskRequest(ctx context.Context, req *AskRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockContractValidator) ValidateBidRequest(ctx context.Context, req *BidRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockContractValidator) ParseDuration(s string) (Duration, error) {
	args := m.Called(s)
	return args.Get(0).(Duration), args.Error(1)
}

// Test cases

func TestNew(t *testing.T) {
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	p := New(cfg, feed, contract)

	assert.NotNil(t, p)
	assert.Equal(t, cfg, p.config)
	assert.Equal(t, feed, p.feed)
	assert.Equal(t, contract, p.contract)
}

// TC-DF-A1B: Calculate ask for RISE contract with valid parameters
func TestCalculateAsk_Rise_Valid(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	req := &AskRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "10.00",
		PricingTime:    1704067200,
	}

	// Setup mocks
	contract.On("ValidateAskRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_100").Return(&SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "30s").Return(Duration{Value: 30, Unit: "s", IsTickBased: false}, nil)
	contract.On("ParseDuration", "60s").Return(Duration{Value: 60, Unit: "s", IsTickBased: false}, nil)
	feed.On("GetTickForEpoch", ctx, "R_100", int64(1704067200)).Return(&Tick{
		Symbol: "R_100",
		Time:   1704067200,
		Quote:  "1234.56",
	}, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateAsk(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "10.00", result.AskPrice)
	assert.Equal(t, "USD", result.Currency)
	assert.Equal(t, "1234.56", result.CurrentSpot)
	assert.Equal(t, int64(1704067200), result.CurrentSpotTime)
	assert.NotEmpty(t, result.Payout)
	assert.Equal(t, "1000.00", result.MaxPayout)
	assert.Equal(t, "1.00", result.MinStake)

	cfg.AssertExpectations(t)
	feed.AssertExpectations(t)
	contract.AssertExpectations(t)
}

// TC-DF-A2C: Calculate ask for FALL contract with valid parameters
func TestCalculateAsk_Fall_Valid(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	req := &AskRequest{
		Symbol:         "R_50",
		ContractType:   ContractTypeFall,
		Currency:       "USD",
		FirstDuration:  "5m",
		SecondDuration: "10m",
		Stake:          "25.00",
		PricingTime:    1704067200,
	}

	contract.On("ValidateAskRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_50").Return(&SymbolConfig{
		Symbol:         "R_50",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "5m").Return(Duration{Value: 5, Unit: "m", IsTickBased: false}, nil)
	contract.On("ParseDuration", "10m").Return(Duration{Value: 10, Unit: "m", IsTickBased: false}, nil)
	feed.On("GetTickForEpoch", ctx, "R_50", int64(1704067200)).Return(&Tick{
		Symbol: "R_50",
		Time:   1704067200,
		Quote:  "5678.90",
	}, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateAsk(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "25.00", result.AskPrice)
}

// TC-DF-A3D: Reject ask with stake below minimum
func TestCalculateAsk_StakeBelowMinimum(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	req := &AskRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "0.50",
		PricingTime:    1704067200,
	}

	// Validation should fail for stake below minimum
	contract.On("ValidateAskRequest", ctx, req).Return(ErrInvalidStake)

	p := New(cfg, feed, contract)
	result, err := p.CalculateAsk(ctx, req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInvalidStake)
}

// TC-DF-A4E: Reject ask with payout exceeding maximum
func TestCalculateAsk_PayoutExceeded(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	req := &AskRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "500.00", // High stake will result in payout > 1000
		PricingTime:    1704067200,
	}

	contract.On("ValidateAskRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_100").Return(&SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      100.00, // Low max payout to trigger error
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "30s").Return(Duration{Value: 30, Unit: "s", IsTickBased: false}, nil)
	contract.On("ParseDuration", "60s").Return(Duration{Value: 60, Unit: "s", IsTickBased: false}, nil)
	feed.On("GetTickForEpoch", ctx, "R_100", int64(1704067200)).Return(&Tick{
		Symbol: "R_100",
		Time:   1704067200,
		Quote:  "1234.56",
	}, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateAsk(ctx, req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrPayoutExceeded)
}

// TC-DF-B1F: Evaluate bid for winning RISE contract
func TestCalculateBid_Rise_Win(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	startTime := int64(1704067200)
	pricingTime := startTime + 120 // After expiry

	req := &BidRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		StartTime:      startTime,
		Stake:          "10.00",
		Payout:         "25.00",
		PricingTime:    pricingTime,
	}

	contract.On("ValidateBidRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_100").Return(&SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "30s").Return(Duration{Value: 30, Unit: "s", IsTickBased: false}, nil)
	contract.On("ParseDuration", "60s").Return(Duration{Value: 60, Unit: "s", IsTickBased: false}, nil)

	// Entry tick (barrier) - first tick AFTER start_time
	feed.On("GetTicksFromLimit", ctx, "R_100", startTime+1, int64(1)).Return([]*Tick{
		{Symbol: "R_100", Time: startTime + 1, Quote: "100.00"},
	}, true, nil)

	// Spot at t1 (30s) - above barrier (win condition 1)
	feed.On("GetTickForEpoch", ctx, "R_100", startTime+30).Return(&Tick{
		Symbol: "R_100",
		Time:   startTime + 30,
		Quote:  "105.00",
	}, true, nil)

	// Spot at t2 (60s) - above barrier (win condition 2)
	feed.On("GetTickForEpoch", ctx, "R_100", startTime+60).Return(&Tick{
		Symbol: "R_100",
		Time:   startTime + 60,
		Quote:  "110.00",
	}, true, nil)

	// Current tick
	feed.On("GetTickForEpoch", ctx, "R_100", pricingTime).Return(&Tick{
		Symbol: "R_100",
		Time:   pricingTime,
		Quote:  "108.00",
	}, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateBid(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "25.00", result.BidPrice) // Full payout for win
	assert.True(t, result.IsExpired)
	assert.Equal(t, "100.00", result.Barrier)
	assert.Equal(t, "110.00", result.ExitSpot)
	assert.Equal(t, startTime+60, result.ExitSpotTime)
}

// TC-DF-B2G: Evaluate bid for losing RISE contract (fail at t1)
func TestCalculateBid_Rise_LoseAtT1(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	startTime := int64(1704067200)
	pricingTime := startTime + 120 // After expiry

	req := &BidRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		StartTime:      startTime,
		Stake:          "10.00",
		Payout:         "25.00",
		PricingTime:    pricingTime,
	}

	contract.On("ValidateBidRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_100").Return(&SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "30s").Return(Duration{Value: 30, Unit: "s", IsTickBased: false}, nil)
	contract.On("ParseDuration", "60s").Return(Duration{Value: 60, Unit: "s", IsTickBased: false}, nil)

	// Entry tick (barrier) - first tick AFTER start_time
	feed.On("GetTicksFromLimit", ctx, "R_100", startTime+1, int64(1)).Return([]*Tick{
		{Symbol: "R_100", Time: startTime + 1, Quote: "100.00"},
	}, true, nil)

	// Spot at t1 (30s) - below barrier (FAIL at t1)
	spot1 := &Tick{
		Symbol: "R_100",
		Time:   startTime + 30,
		Quote:  "99.00", // Below barrier
	}
	feed.On("GetTickForEpoch", ctx, "R_100", startTime+30).Return(spot1, true, nil)

	// Current tick
	feed.On("GetTickForEpoch", ctx, "R_100", pricingTime).Return(&Tick{
		Symbol: "R_100",
		Time:   pricingTime,
		Quote:  "105.00",
	}, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateBid(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "0.00", result.BidPrice)  // Loss - bid is 0
	assert.True(t, result.IsExpired)          // Should be expired when t1 fails
	assert.Equal(t, "99.00", result.ExitSpot) // Exit spot should be spot1
	assert.Equal(t, startTime+30, result.ExitSpotTime)
}

// TC-DF-B3H: Evaluate bid for losing RISE contract (fail at t2)
func TestCalculateBid_Rise_LoseAtT2(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	startTime := int64(1704067200)
	pricingTime := startTime + 120

	req := &BidRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		StartTime:      startTime,
		Stake:          "10.00",
		Payout:         "25.00",
		PricingTime:    pricingTime,
	}

	contract.On("ValidateBidRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_100").Return(&SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "30s").Return(Duration{Value: 30, Unit: "s", IsTickBased: false}, nil)
	contract.On("ParseDuration", "60s").Return(Duration{Value: 60, Unit: "s", IsTickBased: false}, nil)

	// Entry tick (barrier) - first tick AFTER start_time
	feed.On("GetTicksFromLimit", ctx, "R_100", startTime+1, int64(1)).Return([]*Tick{
		{Symbol: "R_100", Time: startTime + 1, Quote: "100.00"},
	}, true, nil)

	// Spot at t1 (30s) - above barrier (PASS)
	feed.On("GetTickForEpoch", ctx, "R_100", startTime+30).Return(&Tick{
		Symbol: "R_100",
		Time:   startTime + 30,
		Quote:  "105.00",
	}, true, nil)

	// Spot at t2 (60s) - below barrier (FAIL at t2)
	spot2 := &Tick{
		Symbol: "R_100",
		Time:   startTime + 60,
		Quote:  "98.00", // Dropped below barrier
	}
	feed.On("GetTickForEpoch", ctx, "R_100", startTime+60).Return(spot2, true, nil)

	// Current tick
	feed.On("GetTickForEpoch", ctx, "R_100", pricingTime).Return(&Tick{
		Symbol: "R_100",
		Time:   pricingTime,
		Quote:  "97.00",
	}, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateBid(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "0.00", result.BidPrice) // Loss - bid is 0
	assert.True(t, result.IsExpired)
	assert.Equal(t, "98.00", result.ExitSpot) // Exit spot should be spot2
	assert.Equal(t, startTime+60, result.ExitSpotTime)
}

// TC-DF-B4J: Evaluate bid for winning FALL contract
func TestCalculateBid_Fall_Win(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	startTime := int64(1704067200)
	pricingTime := startTime + 120

	req := &BidRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeFall,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		StartTime:      startTime,
		Stake:          "10.00",
		Payout:         "25.00",
		PricingTime:    pricingTime,
	}

	contract.On("ValidateBidRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_100").Return(&SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "30s").Return(Duration{Value: 30, Unit: "s", IsTickBased: false}, nil)
	contract.On("ParseDuration", "60s").Return(Duration{Value: 60, Unit: "s", IsTickBased: false}, nil)

	// Entry tick (barrier) - first tick AFTER start_time
	feed.On("GetTicksFromLimit", ctx, "R_100", startTime+1, int64(1)).Return([]*Tick{
		{Symbol: "R_100", Time: startTime + 1, Quote: "100.00"},
	}, true, nil)

	// Spot at t1 (30s) - below barrier (win for FALL)
	feed.On("GetTickForEpoch", ctx, "R_100", startTime+30).Return(&Tick{
		Symbol: "R_100",
		Time:   startTime + 30,
		Quote:  "95.00",
	}, true, nil)

	// Spot at t2 (60s) - below barrier (win for FALL)
	spot2 := &Tick{
		Symbol: "R_100",
		Time:   startTime + 60,
		Quote:  "90.00",
	}
	feed.On("GetTickForEpoch", ctx, "R_100", startTime+60).Return(spot2, true, nil)

	// Current tick
	feed.On("GetTickForEpoch", ctx, "R_100", pricingTime).Return(&Tick{
		Symbol: "R_100",
		Time:   pricingTime,
		Quote:  "92.00",
	}, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateBid(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "25.00", result.BidPrice) // Full payout for win
	assert.True(t, result.IsExpired)
	assert.Equal(t, "90.00", result.ExitSpot)
	assert.Equal(t, startTime+60, result.ExitSpotTime)
}

// Test calculateFairProbability with arcsin formula
func TestCalculateFairProbability(t *testing.T) {
	p := &Pricer{}

	tests := []struct {
		name     string
		d1       Duration
		d2       Duration
		expected float64
		delta    float64
	}{
		{
			name:     "time-based 30s/60s",
			d1:       Duration{Value: 30, Unit: "s", IsTickBased: false},
			d2:       Duration{Value: 60, Unit: "s", IsTickBased: false},
			expected: 0.25 + math.Asin(math.Sqrt(30.0/60.0))/(2*math.Pi),
			delta:    0.0001,
		},
		{
			name:     "time-based 1m/2m",
			d1:       Duration{Value: 60, Unit: "s", IsTickBased: false},
			d2:       Duration{Value: 120, Unit: "s", IsTickBased: false},
			expected: 0.25 + math.Asin(math.Sqrt(60.0/120.0))/(2*math.Pi),
			delta:    0.0001,
		},
		{
			name:     "tick-based 3t/5t",
			d1:       Duration{Value: 3, Unit: "t", IsTickBased: true},
			d2:       Duration{Value: 5, Unit: "t", IsTickBased: true},
			expected: 0.25 + math.Asin(math.Sqrt(3.0/5.0))/(2*math.Pi),
			delta:    0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.calculateFairProbability(tt.d1, tt.d2)
			assert.InDelta(t, tt.expected, result, tt.delta)
		})
	}
}

// Test market data unavailable
func TestCalculateAsk_MarketDataUnavailable(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	req := &AskRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "10.00",
		PricingTime:    1704067200,
	}

	contract.On("ValidateAskRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_100").Return(&SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "30s").Return(Duration{Value: 30, Unit: "s", IsTickBased: false}, nil)
	contract.On("ParseDuration", "60s").Return(Duration{Value: 60, Unit: "s", IsTickBased: false}, nil)
	// Return nil tick - market data unavailable
	feed.On("GetTickForEpoch", ctx, "R_100", int64(1704067200)).Return(nil, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateAsk(ctx, req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrMarketDataUnavailable)
}

// Test invalid symbol
func TestCalculateAsk_InvalidSymbol(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	req := &AskRequest{
		Symbol:         "INVALID",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "10.00",
	}

	contract.On("ValidateAskRequest", ctx, req).Return(ErrInvalidSymbol)

	p := New(cfg, feed, contract)
	result, err := p.CalculateAsk(ctx, req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInvalidSymbol)
}

// Test tick-based bid evaluation
func TestCalculateBid_TickBased(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	feed := &MockFeedProvider{}
	contract := &MockContractValidator{}

	startTime := int64(1704067200)
	pricingTime := startTime + 60

	req := &BidRequest{
		Symbol:         "R_100",
		ContractType:   ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "3t",
		SecondDuration: "5t",
		StartTime:      startTime,
		Stake:          "10.00",
		Payout:         "25.00",
		PricingTime:    pricingTime,
	}

	contract.On("ValidateBidRequest", ctx, req).Return(nil)
	cfg.On("GetSymbolConfig", "R_100").Return(&SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}, nil)
	contract.On("ParseDuration", "3t").Return(Duration{Value: 3, Unit: "t", IsTickBased: true}, nil)
	contract.On("ParseDuration", "5t").Return(Duration{Value: 5, Unit: "t", IsTickBased: true}, nil)

	// Entry tick - first tick AFTER start_time
	entryTick := &Tick{Symbol: "R_100", Time: startTime + 1, Quote: "100.00"}
	feed.On("GetTicksFromLimit", ctx, "R_100", startTime+1, int64(1)).Return([]*Tick{entryTick}, true, nil)

	// Tick sequence for tick-based evaluation starting from start_time+1
	ticks := []*Tick{
		{Symbol: "R_100", Time: startTime + 1, Quote: "100.00"},  // Entry tick (index 0)
		{Symbol: "R_100", Time: startTime + 2, Quote: "101.00"},  // Tick 1
		{Symbol: "R_100", Time: startTime + 4, Quote: "102.00"},  // Tick 2
		{Symbol: "R_100", Time: startTime + 6, Quote: "103.00"},  // Tick 3 (t1 evaluation)
		{Symbol: "R_100", Time: startTime + 8, Quote: "104.00"},  // Tick 4
		{Symbol: "R_100", Time: startTime + 10, Quote: "105.00"}, // Tick 5 (t2 evaluation)
	}
	feed.On("GetTicksFromLimit", ctx, "R_100", startTime+1, int64(6)).Return(ticks, true, nil)

	// Current tick
	feed.On("GetTickForEpoch", ctx, "R_100", pricingTime).Return(&Tick{
		Symbol: "R_100",
		Time:   pricingTime,
		Quote:  "106.00",
	}, true, nil)

	p := New(cfg, feed, contract)
	result, err := p.CalculateBid(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "25.00", result.BidPrice) // Win - both t1 and t2 above barrier
	assert.True(t, result.IsExpired)
	assert.Equal(t, "105.00", result.ExitSpot)
	assert.Equal(t, startTime+10, result.ExitSpotTime)
}
