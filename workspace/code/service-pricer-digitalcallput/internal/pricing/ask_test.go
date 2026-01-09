package pricing

import (
	"testing"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/contract"
	"github.com/shopspring/decimal"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   int
		wantUnit    string
		wantType    contract.DurationType
		wantSeconds int64
		wantError   bool
	}{
		{
			name:        "Valid seconds",
			input:       "30s",
			wantValue:   30,
			wantUnit:    "s",
			wantType:    contract.DurationTypeTime,
			wantSeconds: 30,
			wantError:   false,
		},
		{
			name:        "Valid minutes",
			input:       "5m",
			wantValue:   5,
			wantUnit:    "m",
			wantType:    contract.DurationTypeTime,
			wantSeconds: 300,
			wantError:   false,
		},
		{
			name:        "Valid hours",
			input:       "2h",
			wantValue:   2,
			wantUnit:    "h",
			wantType:    contract.DurationTypeTime,
			wantSeconds: 7200,
			wantError:   false,
		},
		{
			name:        "Valid days",
			input:       "7d",
			wantValue:   7,
			wantUnit:    "d",
			wantType:    contract.DurationTypeTime,
			wantSeconds: 604800,
			wantError:   false,
		},
		{
			name:        "Valid ticks",
			input:       "5t",
			wantValue:   5,
			wantUnit:    "t",
			wantType:    contract.DurationTypeTick,
			wantSeconds: 0,
			wantError:   false,
		},
		{
			name:      "Invalid format",
			input:     "abc",
			wantError: true,
		},
		{
			name:      "Invalid unit",
			input:     "5x",
			wantError: true,
		},
		{
			name:      "Tick duration too large",
			input:     "15t",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := contract.ParseDuration(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseDuration() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration() unexpected error: %v", err)
				return
			}
			if got.Value != tt.wantValue {
				t.Errorf("ParseDuration() Value = %v, want %v", got.Value, tt.wantValue)
			}
			if got.Unit != tt.wantUnit {
				t.Errorf("ParseDuration() Unit = %v, want %v", got.Unit, tt.wantUnit)
			}
			if got.Type != tt.wantType {
				t.Errorf("ParseDuration() Type = %v, want %v", got.Type, tt.wantType)
			}
			if got.Seconds != tt.wantSeconds {
				t.Errorf("ParseDuration() Seconds = %v, want %v", got.Seconds, tt.wantSeconds)
			}
		})
	}
}

func TestResolveBarrier(t *testing.T) {
	tests := []struct {
		name       string
		input      *string
		entryPrice string
		want       string
		wantType   contract.BarrierType
		wantError  bool
	}{
		{
			name:       "ATM barrier (nil)",
			input:      nil,
			entryPrice: "100.00",
			want:       "100.00",
			wantType:   contract.BarrierTypeATM,
			wantError:  false,
		},
		{
			name:       "ATM barrier (empty)",
			input:      stringPtr(""),
			entryPrice: "100.00",
			want:       "100.00",
			wantType:   contract.BarrierTypeATM,
			wantError:  false,
		},
		{
			name:       "Absolute barrier",
			input:      stringPtr("105.50"),
			entryPrice: "100.00",
			want:       "105.50",
			wantType:   contract.BarrierTypeAbsolute,
			wantError:  false,
		},
		{
			name:       "Relative positive",
			input:      stringPtr("+5.50"),
			entryPrice: "100.00",
			want:       "105.50",
			wantType:   contract.BarrierTypeRelativePlus,
			wantError:  false,
		},
		{
			name:       "Relative negative",
			input:      stringPtr("-5.50"),
			entryPrice: "100.00",
			want:       "94.50",
			wantType:   contract.BarrierTypeRelativeMinus,
			wantError:  false,
		},
		{
			name:       "Invalid absolute format",
			input:      stringPtr("abc"),
			entryPrice: "100.00",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entryPrice := mustDecimal(tt.entryPrice)
			got, err := contract.ResolveBarrier(tt.input, entryPrice)

			if tt.wantError {
				if err == nil {
					t.Errorf("ResolveBarrier() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ResolveBarrier() unexpected error: %v", err)
				return
			}

			wantDecimal := mustDecimal(tt.want)
			if !got.ResolvedValue.Equal(wantDecimal) {
				t.Errorf("ResolveBarrier() ResolvedValue = %v, want %v", got.ResolvedValue.String(), tt.want)
			}

			if got.Type != tt.wantType {
				t.Errorf("ResolveBarrier() Type = %v, want %v", got.Type, tt.wantType)
			}
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func mustDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}
