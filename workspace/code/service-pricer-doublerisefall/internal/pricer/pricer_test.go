package pricer

import (
	"context"
	"errors"
	"testing"
)

// MockConfigProvider implements ConfigProvider for testing
type MockConfigProvider struct {
	configs map[string]*SymbolConfig
}

func (m *MockConfigProvider) GetSymbolConfig(symbol string) (*SymbolConfig, error) {
	cfg, ok := m.configs[symbol]
	if !ok {
		return nil, ErrInvalidSymbol
	}
	return cfg, nil
}

// MockFeedProvider implements FeedProvider for testing
type MockFeedProvider struct {
	ticks          map[string]map[int64]*Tick
	ticksFromLimit map[string][]*Tick
}

func (m *MockFeedProvider) GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error) {
	if symbolTicks, ok := m.ticks[symbol]; ok {
		if tick, ok := symbolTicks[epoch]; ok {
			return tick, true, nil
		}
	}
	return nil, false, errors.New("tick not found")
}

func (m *MockFeedProvider) GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*Tick, bool, error) {
	if ticks, ok := m.ticksFromLimit[symbol]; ok {
		if int64(len(ticks)) >= limit {
			return ticks[:limit], true, nil
		}
	}
	return nil, false, errors.New("insufficient ticks")
}

func (m *MockFeedProvider) Subscribe(ctx context.Context, symbol string, start int64) Subscription {
	return &MockSubscription{}
}

// MockSubscription implements Subscription for testing
type MockSubscription struct {
	ch chan *Tick
}

func (m *MockSubscription) C() <-chan *Tick {
	if m.ch == nil {
		m.ch = make(chan *Tick)
		close(m.ch)
	}
	return m.ch
}

func (m *MockSubscription) Err() error {
	return nil
}

func (m *MockSubscription) Close() {}

// MockValidator implements ContractValidator for testing
type MockValidator struct{}

func (m *MockValidator) ParseDuration(s string) (Duration, error) {
	// Simple mock implementation
	if len(s) < 2 {
		return Duration{}, ErrInvalidDuration
	}
	unit := DurationUnit(s[len(s)-1:])
	value := int64(10) // Default value for testing
	return Duration{Value: value, Unit: unit}, nil
}

func (m *MockValidator) ValidateAskRequest(req *AskRequest, config *SymbolConfig) error {
	if req.ContractType == ContractTypeUnspecified {
		return ErrInvalidContractType
	}
	if req.Stake < config.MinStake {
		return ErrInvalidStake
	}
	return nil
}

func (m *MockValidator) ValidateBidRequest(req *BidRequest, config *SymbolConfig) error {
	if req.ContractType == ContractTypeUnspecified {
		return ErrInvalidContractType
	}
	if req.StartTime == 0 {
		return ErrMissingStartTime
	}
	if req.Payout == 0 {
		return ErrMissingPayout
	}
	return nil
}

// Test calculateFairProbability
func TestCalculateFairProbability(t *testing.T) {
	tests := []struct {
		name       string
		t1Seconds  int64
		t2Seconds  int64
		wantApprox float64
	}{
		{
			name:       "t1=60s, t2=120s",
			t1Seconds:  60,
			t2Seconds:  120,
			wantApprox: 0.4167, // ~41.67%
		},
		{
			name:       "t1=30s, t2=60s",
			t1Seconds:  30,
			t2Seconds:  60,
			wantApprox: 0.4167,
		},
		{
			name:       "t1=120s, t2=240s",
			t1Seconds:  120,
			t2Seconds:  240,
			wantApprox: 0.4167,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateFairProbability(tt.t1Seconds, tt.t2Seconds)
			// Check if within 0.01 tolerance
			if got < tt.wantApprox-0.01 || got > tt.wantApprox+0.01 {
				t.Errorf("calculateFairProbability() = %v, want ~%v", got, tt.wantApprox)
			}
		})
	}
}

// Test applyCommission
func TestApplyCommission(t *testing.T) {
	tests := []struct {
		name       string
		pFair      float64
		commission float64
		want       float64
	}{
		{
			name:       "5% commission",
			pFair:      0.4167,
			commission: 0.05,
			want:       0.4667,
		},
		{
			name:       "10% commission",
			pFair:      0.3,
			commission: 0.1,
			want:       0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyCommission(tt.pFair, tt.commission)
			if got != tt.want {
				t.Errorf("applyCommission() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test calculatePayout
func TestCalculatePayout(t *testing.T) {
	tests := []struct {
		name    string
		stake   float64
		pClient float64
		want    float64
	}{
		{
			name:    "stake=10, pClient=0.4667",
			stake:   10.0,
			pClient: 0.4667,
			want:    21.43, // Approximately
		},
		{
			name:    "stake=100, pClient=0.5",
			stake:   100.0,
			pClient: 0.5,
			want:    200.0,
		},
		{
			name:    "invalid pClient=0",
			stake:   10.0,
			pClient: 0.0,
			want:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePayout(tt.stake, tt.pClient)
			// Check if within 0.01 tolerance
			if got < tt.want-0.01 && got > tt.want+0.01 {
				t.Errorf("calculatePayout() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test evaluateContract
func TestEvaluateContract(t *testing.T) {
	tests := []struct {
		name         string
		contractType ContractType
		barrier      float64
		spotT1       float64
		spotT2       float64
		want         bool
	}{
		{
			name:         "RISE win - both above barrier",
			contractType: ContractTypeRise,
			barrier:      100.0,
			spotT1:       101.0,
			spotT2:       102.0,
			want:         true,
		},
		{
			name:         "RISE lose - t1 below barrier",
			contractType: ContractTypeRise,
			barrier:      100.0,
			spotT1:       99.0,
			spotT2:       102.0,
			want:         false,
		},
		{
			name:         "RISE lose - t2 below barrier",
			contractType: ContractTypeRise,
			barrier:      100.0,
			spotT1:       101.0,
			spotT2:       99.0,
			want:         false,
		},
		{
			name:         "FALL win - both below barrier",
			contractType: ContractTypeFall,
			barrier:      100.0,
			spotT1:       99.0,
			spotT2:       98.0,
			want:         true,
		},
		{
			name:         "FALL lose - t1 above barrier",
			contractType: ContractTypeFall,
			barrier:      100.0,
			spotT1:       101.0,
			spotT2:       98.0,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateContract(tt.contractType, tt.barrier, tt.spotT1, tt.spotT2)
			if got != tt.want {
				t.Errorf("evaluateContract() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test CalculateAsk with time-based duration
func TestPricer_CalculateAsk_TimeBased(t *testing.T) {
	mockConfig := &MockConfigProvider{
		configs: map[string]*SymbolConfig{
			"R_100": {
				Symbol:     "R_100",
				Commission: 0.05,
				MaxPayout:  1000.0,
				MinStake:   1.0,
				Enabled:    true,
			},
		},
	}

	mockFeed := &MockFeedProvider{
		ticks: map[string]map[int64]*Tick{
			"R_100": {
				1736930731: {Symbol: "R_100", Time: 1736930731, Quote: "1234.5678"},
			},
		},
	}

	mockValidator := &MockValidator{}

	pricer := NewPricer(mockConfig, mockFeed, mockValidator)

	req := &AskRequest{
		Symbol:       "R_100",
		ContractType: ContractTypeRise,
		Currency:     "USD",
		FirstDuration: Duration{
			Value: 60,
			Unit:  DurationUnitSeconds,
		},
		SecondDuration: Duration{
			Value: 120,
			Unit:  DurationUnitSeconds,
		},
		Stake:       10.0,
		PricingTime: 1736930731,
	}

	result, err := pricer.CalculateAsk(context.Background(), req)
	if err != nil {
		t.Fatalf("CalculateAsk() error = %v", err)
	}

	if result.AskPrice == "" {
		t.Error("CalculateAsk() returned empty AskPrice")
	}
	if result.Currency != "USD" {
		t.Errorf("CalculateAsk() Currency = %v, want USD", result.Currency)
	}
	if result.CurrentSpot != "1234.5678" {
		t.Errorf("CalculateAsk() CurrentSpot = %v, want 1234.5678", result.CurrentSpot)
	}
}

// Test CalculateAsk with invalid symbol
func TestPricer_CalculateAsk_InvalidSymbol(t *testing.T) {
	mockConfig := &MockConfigProvider{
		configs: map[string]*SymbolConfig{},
	}
	mockFeed := &MockFeedProvider{}
	mockValidator := &MockValidator{}

	pricer := NewPricer(mockConfig, mockFeed, mockValidator)

	req := &AskRequest{
		Symbol:       "INVALID",
		ContractType: ContractTypeRise,
		Currency:     "USD",
		FirstDuration: Duration{
			Value: 60,
			Unit:  DurationUnitSeconds,
		},
		SecondDuration: Duration{
			Value: 120,
			Unit:  DurationUnitSeconds,
		},
		Stake: 10.0,
	}

	_, err := pricer.CalculateAsk(context.Background(), req)
	if err == nil {
		t.Error("CalculateAsk() expected error for invalid symbol, got nil")
	}
	if !errors.Is(err, ErrInvalidSymbol) {
		t.Errorf("CalculateAsk() error = %v, want ErrInvalidSymbol", err)
	}
}

// Test formatPrice
func TestFormatPrice(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		decimals int
		want     string
	}{
		{
			name:     "4 decimals",
			value:    0.4667,
			decimals: 4,
			want:     "0.4667",
		},
		{
			name:     "2 decimals",
			value:    21.4285714,
			decimals: 2,
			want:     "21.43",
		},
		{
			name:     "round up",
			value:    21.435,
			decimals: 2,
			want:     "21.44",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPrice(tt.value, tt.decimals)
			if got != tt.want {
				t.Errorf("formatPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}
