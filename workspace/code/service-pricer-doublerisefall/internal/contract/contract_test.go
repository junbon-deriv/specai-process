package contract

import (
	"errors"
	"testing"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
)

// MockConfigProvider for testing
type MockConfigProvider struct {
	configs map[string]*pricer.SymbolConfig
}

func (m *MockConfigProvider) GetSymbolConfig(symbol string) (*pricer.SymbolConfig, error) {
	cfg, ok := m.configs[symbol]
	if !ok {
		return nil, pricer.ErrInvalidSymbol
	}
	return cfg, nil
}

// Test ParseDuration
func TestValidator_ParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    pricer.Duration
		wantErr bool
	}{
		{
			name:  "parse seconds",
			input: "30s",
			want: pricer.Duration{
				Value: 30,
				Unit:  pricer.DurationUnitSeconds,
			},
			wantErr: false,
		},
		{
			name:  "parse minutes",
			input: "5m",
			want: pricer.Duration{
				Value: 5,
				Unit:  pricer.DurationUnitMinutes,
			},
			wantErr: false,
		},
		{
			name:  "parse hours",
			input: "2h",
			want: pricer.Duration{
				Value: 2,
				Unit:  pricer.DurationUnitHours,
			},
			wantErr: false,
		},
		{
			name:  "parse days",
			input: "1d",
			want: pricer.Duration{
				Value: 1,
				Unit:  pricer.DurationUnitDays,
			},
			wantErr: false,
		},
		{
			name:  "parse ticks",
			input: "5t",
			want: pricer.Duration{
				Value: 5,
				Unit:  pricer.DurationUnitTicks,
			},
			wantErr: false,
		},
		{
			name:    "invalid format - no unit",
			input:   "30",
			want:    pricer.Duration{},
			wantErr: true,
		},
		{
			name:    "invalid format - invalid unit",
			input:   "5x",
			want:    pricer.Duration{},
			wantErr: true,
		},
		{
			name:    "invalid format - empty",
			input:   "",
			want:    pricer.Duration{},
			wantErr: true,
		},
	}

	mockConfig := &MockConfigProvider{}
	validator := NewValidator(mockConfig)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Value != tt.want.Value || got.Unit != tt.want.Unit {
					t.Errorf("ParseDuration() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Test ValidateAskRequest
func TestValidator_ValidateAskRequest(t *testing.T) {
	mockConfig := &MockConfigProvider{
		configs: map[string]*pricer.SymbolConfig{
			"R_100": {
				Symbol:     "R_100",
				Commission: 0.05,
				MaxPayout:  1000.0,
				MinStake:   1.0,
				Enabled:    true,
			},
			"R_10_DISABLED": {
				Symbol:     "R_10_DISABLED",
				Commission: 0.05,
				MaxPayout:  1000.0,
				MinStake:   1.0,
				Enabled:    false,
			},
		},
	}
	validator := NewValidator(mockConfig)

	tests := []struct {
		name    string
		req     *pricer.AskRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: &pricer.AskRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 120,
					Unit:  pricer.DurationUnitSeconds,
				},
				Stake: 10.0,
			},
			wantErr: nil,
		},
		{
			name: "disabled symbol",
			req: &pricer.AskRequest{
				Symbol:       "R_10_DISABLED",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 120,
					Unit:  pricer.DurationUnitSeconds,
				},
				Stake: 10.0,
			},
			wantErr: pricer.ErrSymbolDisabled,
		},
		{
			name: "invalid contract type",
			req: &pricer.AskRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeUnspecified,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 120,
					Unit:  pricer.DurationUnitSeconds,
				},
				Stake: 10.0,
			},
			wantErr: pricer.ErrInvalidContractType,
		},
		{
			name: "stake below minimum",
			req: &pricer.AskRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 120,
					Unit:  pricer.DurationUnitSeconds,
				},
				Stake: 0.5,
			},
			wantErr: pricer.ErrInvalidStake,
		},
		{
			name: "duration order invalid",
			req: &pricer.AskRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 120,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				Stake: 10.0,
			},
			wantErr: pricer.ErrDurationOrder,
		},
		{
			name: "duration gap too small",
			req: &pricer.AskRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 65,
					Unit:  pricer.DurationUnitSeconds,
				},
				Stake: 10.0,
			},
			wantErr: pricer.ErrDurationGap,
		},
		{
			name: "valid tick-based",
			req: &pricer.AskRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 5,
					Unit:  pricer.DurationUnitTicks,
				},
				SecondDuration: pricer.Duration{
					Value: 10,
					Unit:  pricer.DurationUnitTicks,
				},
				Stake: 10.0,
			},
			wantErr: nil,
		},
		{
			name: "tick-based gap too small",
			req: &pricer.AskRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 5,
					Unit:  pricer.DurationUnitTicks,
				},
				SecondDuration: pricer.Duration{
					Value: 6,
					Unit:  pricer.DurationUnitTicks,
				},
				Stake: 10.0,
			},
			wantErr: pricer.ErrDurationGap,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := mockConfig.GetSymbolConfig(tt.req.Symbol)
			err := validator.ValidateAskRequest(tt.req, cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateAskRequest() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateAskRequest() error = nil, want %v", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateAskRequest() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

// Test ValidateBidRequest
func TestValidator_ValidateBidRequest(t *testing.T) {
	mockConfig := &MockConfigProvider{
		configs: map[string]*pricer.SymbolConfig{
			"R_100": {
				Symbol:     "R_100",
				Commission: 0.05,
				MaxPayout:  1000.0,
				MinStake:   1.0,
				Enabled:    true,
			},
		},
	}
	validator := NewValidator(mockConfig)

	tests := []struct {
		name    string
		req     *pricer.BidRequest
		wantErr error
	}{
		{
			name: "valid bid request",
			req: &pricer.BidRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 120,
					Unit:  pricer.DurationUnitSeconds,
				},
				StartTime: 1736930600,
				Stake:     10.0,
				Payout:    27.85,
			},
			wantErr: nil,
		},
		{
			name: "missing start time",
			req: &pricer.BidRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 120,
					Unit:  pricer.DurationUnitSeconds,
				},
				StartTime: 0,
				Stake:     10.0,
				Payout:    27.85,
			},
			wantErr: pricer.ErrMissingStartTime,
		},
		{
			name: "missing payout",
			req: &pricer.BidRequest{
				Symbol:       "R_100",
				ContractType: pricer.ContractTypeRise,
				Currency:     "USD",
				FirstDuration: pricer.Duration{
					Value: 60,
					Unit:  pricer.DurationUnitSeconds,
				},
				SecondDuration: pricer.Duration{
					Value: 120,
					Unit:  pricer.DurationUnitSeconds,
				},
				StartTime: 1736930600,
				Stake:     10.0,
				Payout:    0,
			},
			wantErr: pricer.ErrMissingPayout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := mockConfig.GetSymbolConfig(tt.req.Symbol)
			err := validator.ValidateBidRequest(tt.req, cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateBidRequest() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateBidRequest() error = nil, want %v", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateBidRequest() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

// Test isTimeBased
func Test_isTimeBased(t *testing.T) {
	tests := []struct {
		name string
		d    pricer.Duration
		want bool
	}{
		{
			name: "seconds is time-based",
			d:    pricer.Duration{Value: 60, Unit: pricer.DurationUnitSeconds},
			want: true,
		},
		{
			name: "minutes is time-based",
			d:    pricer.Duration{Value: 5, Unit: pricer.DurationUnitMinutes},
			want: true,
		},
		{
			name: "ticks is not time-based",
			d:    pricer.Duration{Value: 5, Unit: pricer.DurationUnitTicks},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimeBased(tt.d)
			if got != tt.want {
				t.Errorf("isTimeBased() = %v, want %v", got, tt.want)
			}
		})
	}
}
