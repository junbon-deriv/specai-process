package contract

import (
	"context"
	"testing"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConfigProvider mocks ConfigProvider interface for testing
type MockConfigProvider struct {
	mock.Mock
}

func (m *MockConfigProvider) GetSymbolConfig(symbol string) (*pricer.SymbolConfig, error) {
	args := m.Called(symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pricer.SymbolConfig), args.Error(1)
}

func defaultConfig() *pricer.SymbolConfig {
	return &pricer.SymbolConfig{
		Symbol:         "R_100",
		CommissionRate: 0.05,
		MaxPayout:      1000.00,
		MinStake:       1.00,
		Enabled:        true,
	}
}

// Test ValidateAskRequest - valid request
func TestValidateAskRequest_Valid(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "10.00",
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.NoError(t, err)
}

// Test ValidateAskRequest - unsupported symbol
func TestValidateAskRequest_UnsupportedSymbol(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "INVALID",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "10.00",
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pricer.ErrInvalidSymbol)
}

// Test ValidateAskRequest - stake below minimum
func TestValidateAskRequest_StakeBelowMinimum(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "0.50", // Below min stake of 1.00
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pricer.ErrInvalidStake)
}

// Test ValidateAskRequest - invalid stake format
func TestValidateAskRequest_InvalidStakeFormat(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "invalid",
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pricer.ErrInvalidStake)
}

// Test ValidateAskRequest - mixed duration types
func TestValidateAskRequest_MixedDurationTypes(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s", // Time-based
		SecondDuration: "5t",  // Tick-based
		Stake:          "10.00",
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pricer.ErrInvalidDuration)
}

// Test ValidateAskRequest - duration order violation
func TestValidateAskRequest_DurationOrderViolation(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "60s",
		SecondDuration: "30s", // Second should be greater than first
		Stake:          "10.00",
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pricer.ErrInvalidDuration)
}

// Test ValidateAskRequest - duration gap too small (time-based)
func TestValidateAskRequest_DurationGapTooSmall_TimeBased(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "35s", // Gap of 5s, minimum is 10s
		Stake:          "10.00",
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pricer.ErrInvalidDuration)
	assert.Contains(t, err.Error(), "at least 10 seconds")
}

// Test ValidateAskRequest - duration gap too small (tick-based)
func TestValidateAskRequest_DurationGapTooSmall_TickBased(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "3t",
		SecondDuration: "4t", // Gap of 1 tick, minimum is 2 ticks
		Stake:          "10.00",
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pricer.ErrInvalidDuration)
	assert.Contains(t, err.Error(), "at least 2 ticks")
}

// Test ValidateAskRequest - pricing time in future
func TestValidateAskRequest_PricingTimeInFuture(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		Stake:          "10.00",
		PricingTime:    9999999999, // Far in the future
	}

	err := v.ValidateAskRequest(ctx, req)

	assert.Error(t, err)
}

// Test ValidateBidRequest - valid request
func TestValidateBidRequest_Valid(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	req := &pricer.BidRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		StartTime:      1704067200,
		Stake:          "10.00",
		Payout:         "25.00",
	}

	err := v.ValidateBidRequest(ctx, req)

	assert.NoError(t, err)
}

// Test ValidateBidRequest - missing start time
func TestValidateBidRequest_MissingStartTime(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}

	v := NewValidator(cfg)

	req := &pricer.BidRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		StartTime:      0, // Missing
		Stake:          "10.00",
		Payout:         "25.00",
	}

	err := v.ValidateBidRequest(ctx, req)

	assert.ErrorIs(t, err, pricer.ErrMissingStartTime)
}

// Test ValidateBidRequest - missing payout
func TestValidateBidRequest_MissingPayout(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}

	v := NewValidator(cfg)

	req := &pricer.BidRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		StartTime:      1704067200,
		Stake:          "10.00",
		Payout:         "", // Missing
	}

	err := v.ValidateBidRequest(ctx, req)

	assert.ErrorIs(t, err, pricer.ErrMissingPayout)
}

// Test ValidateBidRequest - payout exceeds maximum
func TestValidateBidRequest_PayoutExceedsMax(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil) // MaxPayout is 1000

	v := NewValidator(cfg)

	req := &pricer.BidRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "30s",
		SecondDuration: "60s",
		StartTime:      1704067200,
		Stake:          "10.00",
		Payout:         "5000.00", // Exceeds 1000
	}

	err := v.ValidateBidRequest(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pricer.ErrPayoutExceeded)
}

// Test ParseDuration wrapper
func TestValidator_ParseDuration(t *testing.T) {
	cfg := &MockConfigProvider{}
	v := NewValidator(cfg)

	tests := []struct {
		input   string
		wantErr bool
	}{
		{"30s", false},
		{"5m", false},
		{"2h", false},
		{"1d", false},
		{"5t", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := v.ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test all supported symbols
func TestValidateAskRequest_AllSupportedSymbols(t *testing.T) {
	ctx := context.Background()

	symbols := []string{"R_10", "R_25", "R_50", "R_75", "R_100"}

	for _, sym := range symbols {
		t.Run(sym, func(t *testing.T) {
			cfg := &MockConfigProvider{}
			cfg.On("GetSymbolConfig", sym).Return(&pricer.SymbolConfig{
				Symbol:         sym,
				CommissionRate: 0.05,
				MaxPayout:      1000.00,
				MinStake:       1.00,
				Enabled:        true,
			}, nil)

			v := NewValidator(cfg)

			req := &pricer.AskRequest{
				Symbol:         sym,
				ContractType:   pricer.ContractTypeRise,
				Currency:       "USD",
				FirstDuration:  "30s",
				SecondDuration: "60s",
				Stake:          "10.00",
			}

			err := v.ValidateAskRequest(ctx, req)
			require.NoError(t, err)
		})
	}
}

// Test tick-based duration validation
func TestValidateAskRequest_TickBasedDurations(t *testing.T) {
	ctx := context.Background()
	cfg := &MockConfigProvider{}
	cfg.On("GetSymbolConfig", "R_100").Return(defaultConfig(), nil)

	v := NewValidator(cfg)

	// Valid tick-based durations with 2+ tick gap
	req := &pricer.AskRequest{
		Symbol:         "R_100",
		ContractType:   pricer.ContractTypeRise,
		Currency:       "USD",
		FirstDuration:  "3t",
		SecondDuration: "5t", // Gap of 2 ticks
		Stake:          "10.00",
	}

	err := v.ValidateAskRequest(ctx, req)
	assert.NoError(t, err)
}
