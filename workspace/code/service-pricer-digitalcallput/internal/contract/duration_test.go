package contract

import (
	"testing"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   int
		wantUnit    string
		wantType    DurationType
		wantSeconds int64
		wantError   bool
	}{
		{
			name:        "Valid seconds",
			input:       "30s",
			wantValue:   30,
			wantUnit:    "s",
			wantType:    DurationTypeTime,
			wantSeconds: 30,
			wantError:   false,
		},
		{
			name:        "Valid minutes",
			input:       "5m",
			wantValue:   5,
			wantUnit:    "m",
			wantType:    DurationTypeTime,
			wantSeconds: 300,
			wantError:   false,
		},
		{
			name:        "Valid hours",
			input:       "2h",
			wantValue:   2,
			wantUnit:    "h",
			wantType:    DurationTypeTime,
			wantSeconds: 7200,
			wantError:   false,
		},
		{
			name:        "Valid days",
			input:       "7d",
			wantValue:   7,
			wantUnit:    "d",
			wantType:    DurationTypeTime,
			wantSeconds: 604800,
			wantError:   false,
		},
		{
			name:        "Valid ticks",
			input:       "5t",
			wantValue:   5,
			wantUnit:    "t",
			wantType:    DurationTypeTick,
			wantSeconds: 0,
			wantError:   false,
		},
		{
			name:      "Invalid format - too short",
			input:     "5",
			wantError: true,
		},
		{
			name:      "Invalid format - no number",
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
		{
			name:      "Zero duration",
			input:     "0s",
			wantError: true,
		},
		{
			name:      "Negative duration",
			input:     "-5s",
			wantError: true,
		},
		{
			name:      "Days exceeding maximum",
			input:     "500d",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)

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

func TestDuration_CalculateExpiry(t *testing.T) {
	tests := []struct {
		name       string
		duration   *Duration
		startTime  int64
		wantExpiry int64
	}{
		{
			name: "Time-based - 5 minutes",
			duration: &Duration{
				Value:   5,
				Unit:    "m",
				Type:    DurationTypeTime,
				Seconds: 300,
			},
			startTime:  1000,
			wantExpiry: 1300,
		},
		{
			name: "Time-based - 1 hour",
			duration: &Duration{
				Value:   1,
				Unit:    "h",
				Type:    DurationTypeTime,
				Seconds: 3600,
			},
			startTime:  1000,
			wantExpiry: 4600,
		},
		{
			name: "Tick-based - returns 0",
			duration: &Duration{
				Value:   5,
				Unit:    "t",
				Type:    DurationTypeTick,
				Seconds: 0,
			},
			startTime:  1000,
			wantExpiry: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.duration.CalculateExpiry(tt.startTime)
			if got != tt.wantExpiry {
				t.Errorf("CalculateExpiry() = %v, want %v", got, tt.wantExpiry)
			}
		})
	}
}

func TestDuration_ToYears(t *testing.T) {
	tests := []struct {
		name     string
		duration *Duration
		wantYear float64
	}{
		{
			name: "1 day time-based",
			duration: &Duration{
				Value:   1,
				Unit:    "d",
				Type:    DurationTypeTime,
				Seconds: 86400,
			},
			wantYear: 0.00273785, // approximately 1/365.25
		},
		{
			name: "1 tick-based",
			duration: &Duration{
				Value:   1,
				Unit:    "t",
				Type:    DurationTypeTick,
				Seconds: 0,
			},
			wantYear: 3.168808781402895e-08, // approximately 1/(365.25*24*60*60)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.duration.ToYears()
			// Use a tolerance for floating point comparison
			tolerance := 0.00001
			if got < tt.wantYear-tolerance || got > tt.wantYear+tolerance {
				t.Errorf("ToYears() = %v, want approximately %v", got, tt.wantYear)
			}
		})
	}
}

func TestDuration_IsTick(t *testing.T) {
	timeBased := &Duration{Type: DurationTypeTime}
	tickBased := &Duration{Type: DurationTypeTick}

	if timeBased.IsTick() {
		t.Error("IsTick() should return false for time-based duration")
	}
	if !tickBased.IsTick() {
		t.Error("IsTick() should return true for tick-based duration")
	}
}

func TestDuration_IsTime(t *testing.T) {
	timeBased := &Duration{Type: DurationTypeTime}
	tickBased := &Duration{Type: DurationTypeTick}

	if !timeBased.IsTime() {
		t.Error("IsTime() should return true for time-based duration")
	}
	if tickBased.IsTime() {
		t.Error("IsTime() should return false for tick-based duration")
	}
}
