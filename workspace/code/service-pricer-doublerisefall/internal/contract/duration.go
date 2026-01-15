package contract

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
)

var (
	// durationRegex matches duration strings like "30s", "1m", "2h", "1d", "5t"
	durationRegex = regexp.MustCompile(`^(\d+)([smhdt])$`)
)

// ParseDuration parses a duration string into a Duration struct.
// Supported formats:
//   - Time-based: "30s", "1m", "2h", "1d"
//   - Tick-based: "5t", "10t"
func ParseDuration(s string) (pricer.Duration, error) {
	matches := durationRegex.FindStringSubmatch(s)
	if matches == nil {
		return pricer.Duration{}, pricer.ErrInvalidDuration
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return pricer.Duration{}, pricer.ErrInvalidDuration
	}

	unit := matches[2]
	isTickBased := unit == "t"

	// Validate value ranges
	if isTickBased {
		if value < 2 || value > 10 {
			return pricer.Duration{}, fmt.Errorf("tick duration must be between 2t and 10t: %w", pricer.ErrInvalidDuration)
		}
	} else {
		// Convert to seconds for validation
		var seconds int64
		switch unit {
		case "s":
			seconds = value
		case "m":
			seconds = value * 60
		case "h":
			seconds = value * 3600
		case "d":
			seconds = value * 86400
		}

		// Max 1 day (86400 seconds)
		if seconds > 86400 {
			return pricer.Duration{}, fmt.Errorf("duration exceeds maximum of 1 day: %w", pricer.ErrInvalidDuration)
		}
	}

	return pricer.Duration{
		Value:       value,
		Unit:        unit,
		IsTickBased: isTickBased,
	}, nil
}
