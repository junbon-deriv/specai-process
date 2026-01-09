package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// MockMarketProvider implements MarketDataProvider for testing
type mockMarketProvider struct {
	latestTick *Tick
	tickMap    map[int64]*Tick
	streamErr  error
}

func (m *mockMarketProvider) GetLatestTick(ctx context.Context, symbol string) (*Tick, error) {
	if m.latestTick != nil {
		return m.latestTick, nil
	}
	return &Tick{
		Symbol:    symbol,
		Price:     decimal.NewFromFloat(100.0),
		Timestamp: time.Now().Unix(),
	}, nil
}

func (m *mockMarketProvider) GetTick(ctx context.Context, symbol string, timestamp int64) (*Tick, error) {
	if tick, ok := m.tickMap[timestamp]; ok {
		return tick, nil
	}
	// Default: return a tick at the requested timestamp
	return &Tick{
		Symbol:    symbol,
		Price:     decimal.NewFromFloat(100.0),
		Timestamp: timestamp,
	}, nil
}

func (m *mockMarketProvider) GetTicksInRange(ctx context.Context, symbol string, from, to int64) ([]*Tick, error) {
	// Return ticks in the range from tickMap
	var ticks []*Tick
	for ts, tick := range m.tickMap {
		if ts >= from && ts <= to {
			ticks = append(ticks, tick)
		}
	}
	return ticks, nil
}

func (m *mockMarketProvider) StreamTicks(ctx context.Context, symbol string) (<-chan *Tick, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan *Tick)
	close(ch) // Close immediately for testing
	return ch, nil
}

// MockConfigProvider implements ConfigProvider for testing
type mockConfigProvider struct {
	config *SymbolConfig
}

func (m *mockConfigProvider) GetSymbolConfig(symbol string) (*SymbolConfig, error) {
	if m.config != nil {
		return m.config, nil
	}
	return &SymbolConfig{
		Symbol:     symbol,
		MinStake:   decimal.NewFromFloat(1.0),
		MaxPayout:  decimal.NewFromFloat(50000.0),
		Commission: decimal.NewFromFloat(0.02),
	}, nil
}

func TestCalculateBid_ActiveContract(t *testing.T) {
	now := time.Now().Unix()
	startTime := now - 300 // Started 5 minutes ago

	market := &mockMarketProvider{
		latestTick: &Tick{
			Symbol:    "USD/JPY",
			Price:     decimal.NewFromFloat(102.0),
			Timestamp: now,
		},
		tickMap: map[int64]*Tick{
			startTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(100.0),
				Timestamp: startTime,
			},
		},
	}

	config := &mockConfigProvider{}
	pricer := NewPricer(market, config)

	input := BidInput{
		Symbol:       "USD/JPY",
		ContractType: ContractTypeCall,
		Currency:     "USD",
		Duration:     "15m", // 15 minutes total, 10 minutes remaining
		Barrier:      nil,   // ATM = 100
		StartTime:    startTime,
		Payout:       "10.00000000",
	}

	result, err := pricer.CalculateBid(context.Background(), input)
	if err != nil {
		t.Fatalf("CalculateBid() error = %v", err)
	}

	if result.IsExpired {
		t.Error("CalculateBid() IsExpired should be false for active contract")
	}

	if result.EntrySpot != "100.00000000" {
		t.Errorf("CalculateBid() EntrySpot = %v, want 100.00000000", result.EntrySpot)
	}

	if result.CurrentSpot != "102.00000000" {
		t.Errorf("CalculateBid() CurrentSpot = %v, want 102.00000000", result.CurrentSpot)
	}

	if result.Barrier != "100.00000000" {
		t.Errorf("CalculateBid() Barrier = %v, want 100.00000000 (ATM)", result.Barrier)
	}

	// Bid price should be positive and less than or equal to payout for active contract
	bidPrice, _ := decimal.NewFromString(result.BidPrice)
	payout, _ := decimal.NewFromString(input.Payout)
	if bidPrice.LessThanOrEqual(decimal.Zero) {
		t.Errorf("CalculateBid() BidPrice = %v should be positive", result.BidPrice)
	}
	if bidPrice.GreaterThan(payout) {
		t.Errorf("CalculateBid() BidPrice = %v should not exceed Payout = %v", result.BidPrice, input.Payout)
	}
}

func TestCalculateBid_ExpiredWinningContract(t *testing.T) {
	now := time.Now().Unix()
	startTime := now - 600 // Started 10 minutes ago
	expiryTime := now - 60 // Expired 1 minute ago

	market := &mockMarketProvider{
		latestTick: &Tick{
			Symbol:    "USD/JPY",
			Price:     decimal.NewFromFloat(105.0),
			Timestamp: now,
		},
		tickMap: map[int64]*Tick{
			startTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(100.0),
				Timestamp: startTime,
			},
			expiryTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(105.0), // Exit price > barrier (100)
				Timestamp: expiryTime,
			},
		},
	}

	config := &mockConfigProvider{}
	pricer := NewPricer(market, config)

	input := BidInput{
		Symbol:       "USD/JPY",
		ContractType: ContractTypeCall, // Call wins if exit > barrier
		Currency:     "USD",
		Duration:     "9m",
		Barrier:      nil, // ATM = 100
		StartTime:    startTime,
		Payout:       "10.00000000",
	}

	result, err := pricer.CalculateBid(context.Background(), input)
	if err != nil {
		t.Fatalf("CalculateBid() error = %v", err)
	}

	if !result.IsExpired {
		t.Error("CalculateBid() IsExpired should be true for expired contract")
	}

	// Winning contract should return full payout
	if result.BidPrice != "10.00000000" {
		t.Errorf("CalculateBid() BidPrice = %v, want 10.00000000 (full payout for winning contract)", result.BidPrice)
	}

	if result.ExitSpot == nil {
		t.Error("CalculateBid() ExitSpot should not be nil for expired contract")
	}
}

func TestCalculateBid_ExpiredLosingContract(t *testing.T) {
	now := time.Now().Unix()
	startTime := now - 600
	expiryTime := now - 60

	market := &mockMarketProvider{
		latestTick: &Tick{
			Symbol:    "USD/JPY",
			Price:     decimal.NewFromFloat(95.0),
			Timestamp: now,
		},
		tickMap: map[int64]*Tick{
			startTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(100.0),
				Timestamp: startTime,
			},
			expiryTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(95.0), // Exit price < barrier (100)
				Timestamp: expiryTime,
			},
		},
	}

	config := &mockConfigProvider{}
	pricer := NewPricer(market, config)

	input := BidInput{
		Symbol:       "USD/JPY",
		ContractType: ContractTypeCall, // Call loses if exit < barrier
		Currency:     "USD",
		Duration:     "9m",
		Barrier:      nil, // ATM = 100
		StartTime:    startTime,
		Payout:       "10.00000000",
	}

	result, err := pricer.CalculateBid(context.Background(), input)
	if err != nil {
		t.Fatalf("CalculateBid() error = %v", err)
	}

	if !result.IsExpired {
		t.Error("CalculateBid() IsExpired should be true for expired contract")
	}

	// Losing contract should return zero
	if result.BidPrice != "0.00000000" {
		t.Errorf("CalculateBid() BidPrice = %v, want 0.00000000 (losing contract)", result.BidPrice)
	}
}

func TestCalculateBid_PutWinning(t *testing.T) {
	now := time.Now().Unix()
	startTime := now - 600
	expiryTime := now - 60

	market := &mockMarketProvider{
		latestTick: &Tick{
			Symbol:    "USD/JPY",
			Price:     decimal.NewFromFloat(95.0),
			Timestamp: now,
		},
		tickMap: map[int64]*Tick{
			startTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(100.0),
				Timestamp: startTime,
			},
			expiryTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(95.0), // Exit price < barrier
				Timestamp: expiryTime,
			},
		},
	}

	config := &mockConfigProvider{}
	pricer := NewPricer(market, config)

	input := BidInput{
		Symbol:       "USD/JPY",
		ContractType: ContractTypePut, // Put wins if exit < barrier
		Currency:     "USD",
		Duration:     "9m",
		Barrier:      nil, // ATM = 100
		StartTime:    startTime,
		Payout:       "10.00000000",
	}

	result, err := pricer.CalculateBid(context.Background(), input)
	if err != nil {
		t.Fatalf("CalculateBid() error = %v", err)
	}

	// Winning Put contract should return full payout
	if result.BidPrice != "10.00000000" {
		t.Errorf("CalculateBid() BidPrice = %v, want 10.00000000 (winning Put contract)", result.BidPrice)
	}
}

func TestCalculateBid_WithAbsoluteBarrier(t *testing.T) {
	now := time.Now().Unix()
	startTime := now - 60

	barrier := "105.00000000"
	market := &mockMarketProvider{
		latestTick: &Tick{
			Symbol:    "USD/JPY",
			Price:     decimal.NewFromFloat(103.0),
			Timestamp: now,
		},
		tickMap: map[int64]*Tick{
			startTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(100.0),
				Timestamp: startTime,
			},
		},
	}

	config := &mockConfigProvider{}
	pricer := NewPricer(market, config)

	input := BidInput{
		Symbol:       "USD/JPY",
		ContractType: ContractTypeCall,
		Currency:     "USD",
		Duration:     "5m",
		Barrier:      &barrier,
		StartTime:    startTime,
		Payout:       "10.00000000",
	}

	result, err := pricer.CalculateBid(context.Background(), input)
	if err != nil {
		t.Fatalf("CalculateBid() error = %v", err)
	}

	if result.Barrier != "105.00000000" {
		t.Errorf("CalculateBid() Barrier = %v, want 105.00000000", result.Barrier)
	}
}

func TestCalculateBid_WithRelativeBarrier(t *testing.T) {
	now := time.Now().Unix()
	startTime := now - 60

	barrier := "+5.00000000"
	market := &mockMarketProvider{
		latestTick: &Tick{
			Symbol:    "USD/JPY",
			Price:     decimal.NewFromFloat(103.0),
			Timestamp: now,
		},
		tickMap: map[int64]*Tick{
			startTime: {
				Symbol:    "USD/JPY",
				Price:     decimal.NewFromFloat(100.0), // Entry price
				Timestamp: startTime,
			},
		},
	}

	config := &mockConfigProvider{}
	pricer := NewPricer(market, config)

	input := BidInput{
		Symbol:       "USD/JPY",
		ContractType: ContractTypeCall,
		Currency:     "USD",
		Duration:     "5m",
		Barrier:      &barrier,
		StartTime:    startTime,
		Payout:       "10.00000000",
	}

	result, err := pricer.CalculateBid(context.Background(), input)
	if err != nil {
		t.Fatalf("CalculateBid() error = %v", err)
	}

	// Barrier should be entry price (100) + offset (5) = 105
	if result.Barrier != "105.00000000" {
		t.Errorf("CalculateBid() Barrier = %v, want 105.00000000 (100 + 5)", result.Barrier)
	}
}

func TestIsWin(t *testing.T) {
	tests := []struct {
		name         string
		contractType ContractType
		exitPrice    string
		barrier      string
		want         bool
	}{
		{
			name:         "Call wins - exit > barrier",
			contractType: ContractTypeCall,
			exitPrice:    "105.0",
			barrier:      "100.0",
			want:         true,
		},
		{
			name:         "Call loses - exit < barrier",
			contractType: ContractTypeCall,
			exitPrice:    "95.0",
			barrier:      "100.0",
			want:         false,
		},
		{
			name:         "Call loses - exit = barrier",
			contractType: ContractTypeCall,
			exitPrice:    "100.0",
			barrier:      "100.0",
			want:         false,
		},
		{
			name:         "Put wins - exit < barrier",
			contractType: ContractTypePut,
			exitPrice:    "95.0",
			barrier:      "100.0",
			want:         true,
		},
		{
			name:         "Put loses - exit > barrier",
			contractType: ContractTypePut,
			exitPrice:    "105.0",
			barrier:      "100.0",
			want:         false,
		},
		{
			name:         "Put loses - exit = barrier",
			contractType: ContractTypePut,
			exitPrice:    "100.0",
			barrier:      "100.0",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitPrice := mustDecimal(tt.exitPrice)
			barrier := mustDecimal(tt.barrier)
			got := isWin(tt.contractType, exitPrice, barrier)
			if got != tt.want {
				t.Errorf("isWin() = %v, want %v", got, tt.want)
			}
		})
	}
}
