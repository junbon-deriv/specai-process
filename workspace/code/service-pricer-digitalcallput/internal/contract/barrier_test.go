package contract

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestResolveBarrier(t *testing.T) {
	tests := []struct {
		name       string
		input      *string
		entryPrice string
		wantValue  string
		wantType   BarrierType
		wantError  bool
	}{
		{
			name:       "ATM barrier (nil)",
			input:      nil,
			entryPrice: "100.00",
			wantValue:  "100.00",
			wantType:   BarrierTypeATM,
			wantError:  false,
		},
		{
			name:       "ATM barrier (empty)",
			input:      stringPtr(""),
			entryPrice: "100.00",
			wantValue:  "100.00",
			wantType:   BarrierTypeATM,
			wantError:  false,
		},
		{
			name:       "ATM barrier (spaces)",
			input:      stringPtr("  "),
			entryPrice: "100.00",
			wantValue:  "100.00",
			wantType:   BarrierTypeATM,
			wantError:  false,
		},
		{
			name:       "Absolute barrier",
			input:      stringPtr("105.50"),
			entryPrice: "100.00",
			wantValue:  "105.50",
			wantType:   BarrierTypeAbsolute,
			wantError:  false,
		},
		{
			name:       "Relative positive",
			input:      stringPtr("+5.50"),
			entryPrice: "100.00",
			wantValue:  "105.50",
			wantType:   BarrierTypeRelativePlus,
			wantError:  false,
		},
		{
			name:       "Relative negative",
			input:      stringPtr("-5.50"),
			entryPrice: "100.00",
			wantValue:  "94.50",
			wantType:   BarrierTypeRelativeMinus,
			wantError:  false,
		},
		{
			name:       "Invalid absolute format",
			input:      stringPtr("abc"),
			entryPrice: "100.00",
			wantError:  true,
		},
		{
			name:       "Invalid relative plus format",
			input:      stringPtr("+xyz"),
			entryPrice: "100.00",
			wantError:  true,
		},
		{
			name:       "Invalid relative minus format",
			input:      stringPtr("-xyz"),
			entryPrice: "100.00",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entryPrice := mustDecimal(tt.entryPrice)
			got, err := ResolveBarrier(tt.input, entryPrice)

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

			wantDecimal := mustDecimal(tt.wantValue)
			if !got.ResolvedValue.Equal(wantDecimal) {
				t.Errorf("ResolveBarrier() ResolvedValue = %v, want %v", got.ResolvedValue.String(), tt.wantValue)
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
