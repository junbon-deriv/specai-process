package pricing

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func TestBlackScholes_CalculateForCall(t *testing.T) {
	bs := &BlackScholes{
		volatility: 0.10,
		rate:       0.00,
	}

	tests := []struct {
		name          string
		spot          string
		barrier       string
		durationYears float64
		wantMin       float64
		wantMax       float64
	}{
		{
			name:          "Call ATM 1 year",
			spot:          "100.0",
			barrier:       "100.0",
			durationYears: 1.0,
			wantMin:       0.45,
			wantMax:       0.55,
		},
		{
			name:          "Call ITM 1 year",
			spot:          "110.0",
			barrier:       "100.0",
			durationYears: 1.0,
			wantMin:       0.80,
			wantMax:       1.0,
		},
		{
			name:          "Call OTM 1 year",
			spot:          "90.0",
			barrier:       "100.0",
			durationYears: 1.0,
			wantMin:       0.0,
			wantMax:       0.20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spot := decimal.RequireFromString(tt.spot)
			barrier := decimal.RequireFromString(tt.barrier)

			prob := bs.CalculateForCall(spot, barrier, tt.durationYears)
			probFloat := prob.InexactFloat64()

			if probFloat < tt.wantMin || probFloat > tt.wantMax {
				t.Errorf("CalculateForCall() = %v, want between %v and %v", probFloat, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestBlackScholes_CalculateForPut(t *testing.T) {
	bs := &BlackScholes{
		volatility: 0.10,
		rate:       0.00,
	}

	tests := []struct {
		name          string
		spot          string
		barrier       string
		durationYears float64
		wantMin       float64
		wantMax       float64
	}{
		{
			name:          "Put ATM 1 year",
			spot:          "100.0",
			barrier:       "100.0",
			durationYears: 1.0,
			wantMin:       0.45,
			wantMax:       0.55,
		},
		{
			name:          "Put ITM 1 year",
			spot:          "90.0",
			barrier:       "100.0",
			durationYears: 1.0,
			wantMin:       0.80,
			wantMax:       1.0,
		},
		{
			name:          "Put OTM 1 year",
			spot:          "110.0",
			barrier:       "100.0",
			durationYears: 1.0,
			wantMin:       0.0,
			wantMax:       0.20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spot := decimal.RequireFromString(tt.spot)
			barrier := decimal.RequireFromString(tt.barrier)

			prob := bs.CalculateForPut(spot, barrier, tt.durationYears)
			probFloat := prob.InexactFloat64()

			if probFloat < tt.wantMin || probFloat > tt.wantMax {
				t.Errorf("CalculateForPut() = %v, want between %v and %v", probFloat, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestNormalCDF(t *testing.T) {
	tests := []struct {
		name string
		x    float64
		want float64
	}{
		{
			name: "Zero",
			x:    0.0,
			want: 0.5,
		},
		{
			name: "Positive 1",
			x:    1.0,
			want: 0.8413,
		},
		{
			name: "Negative 1",
			x:    -1.0,
			want: 0.1587,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalCDF(tt.x)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("normalCDF(%v) = %v, want %v", tt.x, got, tt.want)
			}
		})
	}
}
