package contract

import (
	"fmt"
	"strconv"
)

// DurationType defines time-based vs tick-based durations
type DurationType int

const (
	// DurationTypeTime represents time-based durations (s, m, h, d)
	DurationTypeTime DurationType = iota
	// DurationTypeTick represents tick-based durations (t)
	DurationTypeTick
)

// Duration represents a parsed duration specification
type Duration struct {
	Value   int
	Unit    string
	Type    DurationType
	Seconds int64 // For time-based: total seconds; for tick-based: 0
}

// ErrInvalidDuration is returned when duration format is invalid
var ErrInvalidDuration = fmt.Errorf("invalid duration format")

// ParseDuration parses a duration string (e.g., "1m", "30s", "5t")
func ParseDuration(input string) (*Duration, error) {
	if len(input) < 2 {
		return nil, ErrInvalidDuration
	}

	unit := input[len(input)-1:]
	valueStr := input[:len(input)-1]

	value, err := strconv.Atoi(valueStr)
	if err != nil || value <= 0 {
		return nil, fmt.Errorf("%w: invalid duration value", ErrInvalidDuration)
	}

	switch unit {
	case "s":
		if value > 365*24*60*60 {
			return nil, fmt.Errorf("%w: duration exceeds maximum", ErrInvalidDuration)
		}
		return &Duration{
			Value:   value,
			Unit:    unit,
			Type:    DurationTypeTime,
			Seconds: int64(value),
		}, nil

	case "m":
		seconds := int64(value * 60)
		if seconds > 365*24*60*60 {
			return nil, fmt.Errorf("%w: duration exceeds maximum", ErrInvalidDuration)
		}
		return &Duration{
			Value:   value,
			Unit:    unit,
			Type:    DurationTypeTime,
			Seconds: seconds,
		}, nil

	case "h":
		seconds := int64(value * 60 * 60)
		if seconds > 365*24*60*60 {
			return nil, fmt.Errorf("%w: duration exceeds maximum", ErrInvalidDuration)
		}
		return &Duration{
			Value:   value,
			Unit:    unit,
			Type:    DurationTypeTime,
			Seconds: seconds,
		}, nil

	case "d":
		if value > 365 {
			return nil, fmt.Errorf("%w: duration exceeds maximum", ErrInvalidDuration)
		}
		seconds := int64(value * 24 * 60 * 60)
		return &Duration{
			Value:   value,
			Unit:    unit,
			Type:    DurationTypeTime,
			Seconds: seconds,
		}, nil

	case "t":
		if value > 10 {
			return nil, fmt.Errorf("%w: tick duration exceeds maximum (10)", ErrInvalidDuration)
		}
		return &Duration{
			Value:   value,
			Unit:    unit,
			Type:    DurationTypeTick,
			Seconds: 0,
		}, nil

	default:
		return nil, fmt.Errorf("%w: unknown duration unit '%s'", ErrInvalidDuration, unit)
	}
}

// CalculateExpiry calculates expiry time for time-based durations
// For tick-based durations, returns 0
func (d *Duration) CalculateExpiry(startTime int64) int64 {
	if d.Type == DurationTypeTick {
		return 0 // Tick-based doesn't have time-based expiry
	}
	return startTime + d.Seconds
}

// ToYears converts duration to years for Black-Scholes calculation
func (d *Duration) ToYears() float64 {
	if d.Type == DurationTypeTick {
		// Approximate: assume 1 tick = 1 second (this is a simplification)
		return float64(d.Value) / (365.25 * 24 * 60 * 60)
	}
	return float64(d.Seconds) / (365.25 * 24 * 60 * 60)
}

// IsTick returns true if this is a tick-based duration
func (d *Duration) IsTick() bool {
	return d.Type == DurationTypeTick
}

// IsTime returns true if this is a time-based duration
func (d *Duration) IsTime() bool {
	return d.Type == DurationTypeTime
}
