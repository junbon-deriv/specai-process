// Package contract handles contract validation and duration parsing.
package contract

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
)

var durationRegex = regexp.MustCompile(`^(\d+)(s|m|h|d|t)$`)

// Validator validates contract parameters.
type Validator struct {
	config pricer.ConfigProvider
}

// NewValidator creates a new contract validator.
func NewValidator(config pricer.ConfigProvider) *Validator {
	return &Validator{config: config}
}

// ParseDuration parses a duration string into a Duration value.
func (v *Validator) ParseDuration(s string) (pricer.Duration, error) {
	matches := durationRegex.FindStringSubmatch(s)
	if matches == nil {
		return pricer.Duration{}, pricer.ErrInvalidDuration
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return pricer.Duration{}, pricer.ErrInvalidDuration
	}

	unit := pricer.DurationUnit(matches[2])

	// Validate unit is supported
	switch unit {
	case pricer.DurationUnitSeconds, pricer.DurationUnitMinutes,
		pricer.DurationUnitHours, pricer.DurationUnitDays, pricer.DurationUnitTicks:
		// Valid unit
	default:
		return pricer.Duration{}, pricer.ErrInvalidDuration
	}

	return pricer.Duration{Value: value, Unit: unit}, nil
}

// ValidateAskRequest validates ask request parameters.
func (v *Validator) ValidateAskRequest(req *pricer.AskRequest, config *pricer.SymbolConfig) error {
	// Validate symbol is enabled
	if !config.Enabled {
		return pricer.ErrSymbolDisabled
	}

	// Validate contract type
	if req.ContractType != pricer.ContractTypeRise && req.ContractType != pricer.ContractTypeFall {
		return pricer.ErrInvalidContractType
	}

	// Validate currency
	if req.Currency == "" {
		return pricer.ErrInvalidCurrency
	}

	// Validate stake
	if req.Stake < config.MinStake {
		return pricer.ErrInvalidStake
	}

	// Validate pricing time is not in the future
	if req.PricingTime > 0 && req.PricingTime > time.Now().Unix() {
		return pricer.ErrPricingTimeFuture
	}

	// Validate duration order
	if err := v.validateDurationOrder(req.FirstDuration, req.SecondDuration); err != nil {
		return err
	}

	// Validate duration gap
	if err := v.validateDurationGap(req.FirstDuration, req.SecondDuration); err != nil {
		return err
	}

	// Validate duration ranges
	if err := v.validateDurationRanges(req.FirstDuration); err != nil {
		return err
	}
	if err := v.validateDurationRanges(req.SecondDuration); err != nil {
		return err
	}

	return nil
}

// ValidateBidRequest validates bid request parameters.
func (v *Validator) ValidateBidRequest(req *pricer.BidRequest, config *pricer.SymbolConfig) error {
	// Validate symbol is enabled
	if !config.Enabled {
		return pricer.ErrSymbolDisabled
	}

	// Validate contract type
	if req.ContractType != pricer.ContractTypeRise && req.ContractType != pricer.ContractTypeFall {
		return pricer.ErrInvalidContractType
	}

	// Validate currency
	if req.Currency == "" {
		return pricer.ErrInvalidCurrency
	}

	// Validate required fields for bid
	if req.StartTime == 0 {
		return pricer.ErrMissingStartTime
	}

	if req.Payout == 0 {
		return pricer.ErrMissingPayout
	}

	// Validate pricing time is not in the future
	if req.PricingTime > 0 && req.PricingTime > time.Now().Unix() {
		return pricer.ErrPricingTimeFuture
	}

	// Validate duration order
	if err := v.validateDurationOrder(req.FirstDuration, req.SecondDuration); err != nil {
		return err
	}

	// Validate duration gap
	if err := v.validateDurationGap(req.FirstDuration, req.SecondDuration); err != nil {
		return err
	}

	// Validate duration ranges
	if err := v.validateDurationRanges(req.FirstDuration); err != nil {
		return err
	}
	if err := v.validateDurationRanges(req.SecondDuration); err != nil {
		return err
	}

	return nil
}

// validateDurationOrder ensures t2 > t1
func (v *Validator) validateDurationOrder(d1, d2 pricer.Duration) error {
	// Both must be the same type (time-based or tick-based)
	if isTimeBased(d1) != isTimeBased(d2) {
		return pricer.ErrInvalidDuration
	}

	// Convert to comparable units
	var v1, v2 int64
	var err error

	if isTimeBased(d1) {
		v1, err = d1.ToSeconds()
		if err != nil {
			return err
		}
		v2, err = d2.ToSeconds()
		if err != nil {
			return err
		}
	} else {
		// Tick-based - compare directly
		v1 = d1.Value
		v2 = d2.Value
	}

	if v2 <= v1 {
		return pricer.ErrDurationOrder
	}

	return nil
}

// validateDurationGap ensures minimum gap (10s for time, 2t for ticks)
func (v *Validator) validateDurationGap(d1, d2 pricer.Duration) error {
	if isTimeBased(d1) {
		// Time-based: minimum 10 seconds gap
		v1, err := d1.ToSeconds()
		if err != nil {
			return err
		}
		v2, err := d2.ToSeconds()
		if err != nil {
			return err
		}
		if v2-v1 < 10 {
			return pricer.ErrDurationGap
		}
	} else {
		// Tick-based: minimum 2 ticks gap
		if d2.Value-d1.Value < 2 {
			return pricer.ErrDurationGap
		}
	}

	return nil
}

// validateDurationRanges validates duration is within acceptable ranges
func (v *Validator) validateDurationRanges(d pricer.Duration) error {
	if isTimeBased(d) {
		// Time-based: 10s to 86400s (1 day)
		seconds, err := d.ToSeconds()
		if err != nil {
			return err
		}
		if seconds < 10 || seconds > 86400 {
			return fmt.Errorf("%w: duration must be between 10s and 1 day", pricer.ErrInvalidDuration)
		}
	} else {
		// Tick-based: 2t to 10t
		if d.Value < 2 || d.Value > 10 {
			return fmt.Errorf("%w: tick duration must be between 2 and 10 ticks", pricer.ErrInvalidDuration)
		}
	}

	return nil
}

// isTimeBased checks if duration is time-based (not tick-based)
func isTimeBased(d pricer.Duration) bool {
	return d.Unit != pricer.DurationUnitTicks
}
