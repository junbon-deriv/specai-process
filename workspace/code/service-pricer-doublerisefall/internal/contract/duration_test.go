package contract

import (
	"testing"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        pricer.Duration
		wantErr     bool
		errContains string
	}{
		{
			name:  "valid seconds",
			input: "30s",
			want: pricer.Duration{
				Value:       30,
				Unit:        "s",
				IsTickBased: false,
			},
			wantErr: false,
		},
		{
			name:  "valid minutes",
			input: "5m",
			want: pricer.Duration{
				Value:       5,
				Unit:        "m",
				IsTickBased: false,
			},
			wantErr: false,
		},
		{
			name:  "valid hours",
			input: "2h",
			want: pricer.Duration{
				Value:       2,
				Unit:        "h",
				IsTickBased: false,
			},
			wantErr: false,
		},
		{
			name:  "valid days",
			input: "1d",
			want: pricer.Duration{
				Value:       1,
				Unit:        "d",
				IsTickBased: false,
			},
			wantErr: false,
		},
		{
			name:  "valid ticks",
			input: "5t",
			want: pricer.Duration{
				Value:       5,
				Unit:        "t",
				IsTickBased: true,
			},
			wantErr: false,
		},
		{
			name:        "invalid format - no unit",
			input:       "30",
			wantErr:     true,
			errContains: "invalid duration",
		},
		{
			name:        "invalid format - wrong unit",
			input:       "30x",
			wantErr:     true,
			errContains: "invalid duration",
		},
		{
			name:        "invalid format - no value",
			input:       "s",
			wantErr:     true,
			errContains: "invalid duration",
		},
		{
			name:        "tick duration too small",
			input:       "1t",
			wantErr:     true,
			errContains: "tick duration must be between 2t and 10t",
		},
		{
			name:        "tick duration too large",
			input:       "11t",
			wantErr:     true,
			errContains: "tick duration must be between 2t and 10t",
		},
		{
			name:        "time duration exceeds max",
			input:       "2d",
			wantErr:     true,
			errContains: "duration exceeds maximum of 1 day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Value, got.Value)
			assert.Equal(t, tt.want.Unit, got.Unit)
			assert.Equal(t, tt.want.IsTickBased, got.IsTickBased)
		})
	}
}

func TestDuration_ToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		duration pricer.Duration
		want     int64
	}{
		{
			name: "seconds",
			duration: pricer.Duration{
				Value:       30,
				Unit:        "s",
				IsTickBased: false,
			},
			want: 30,
		},
		{
			name: "minutes",
			duration: pricer.Duration{
				Value:       5,
				Unit:        "m",
				IsTickBased: false,
			},
			want: 300,
		},
		{
			name: "hours",
			duration: pricer.Duration{
				Value:       2,
				Unit:        "h",
				IsTickBased: false,
			},
			want: 7200,
		},
		{
			name: "days",
			duration: pricer.Duration{
				Value:       1,
				Unit:        "d",
				IsTickBased: false,
			},
			want: 86400,
		},
		{
			name: "tick-based returns 0",
			duration: pricer.Duration{
				Value:       5,
				Unit:        "t",
				IsTickBased: true,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.duration.ToSeconds()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDuration_ToTicks(t *testing.T) {
	tests := []struct {
		name     string
		duration pricer.Duration
		want     int64
	}{
		{
			name: "tick-based",
			duration: pricer.Duration{
				Value:       5,
				Unit:        "t",
				IsTickBased: true,
			},
			want: 5,
		},
		{
			name: "time-based returns 0",
			duration: pricer.Duration{
				Value:       30,
				Unit:        "s",
				IsTickBased: false,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.duration.ToTicks()
			assert.Equal(t, tt.want, got)
		})
	}
}
