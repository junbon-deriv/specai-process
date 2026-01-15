package contract

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
)

var (
	// Supported symbols
	supportedSymbols = map[string]bool{
		"R_10":  true,
		"R_25":  true,
		"R_50":  true,
		"R_75":  true,
		"R_100": true,
	}
)

// Validator validates contract parameters.
type Validator struct {
	config pricer.ConfigProvider
}

// NewValidator creates a new contract validator.
func NewValidator(config pricer.ConfigProvider) *Validator {
	return &Validator{
		config: config,
	}
}

// ValidateAskRequest validates parameters for an Ask request.
func (v *Validator) ValidateAskRequest(ctx context.Context, req *pricer.AskRequest) error {
	// Validate symbol
	if !supportedSymbols[req.Symbol] {
		return fmt.Errorf("symbol %s: %w", req.Symbol, pricer.ErrInvalidSymbol)
	}

	// Get symbol config for additional validation
	cfg, err := v.config.GetSymbolConfig(req.Symbol)
	if err != nil {
		return err
	}

	// Validate stake
	stake, err := strconv.ParseFloat(req.Stake, 64)
	if err != nil {
		return fmt.Errorf("invalid stake format: %w", pricer.ErrInvalidStake)
	}

	if stake < cfg.MinStake {
		return fmt.Errorf("stake %.2f below minimum %.2f: %w", stake, cfg.MinStake, pricer.ErrInvalidStake)
	}

	// Validate durations
	d1, err := ParseDuration(req.FirstDuration)
	if err != nil {
		return fmt.Errorf("first_duration: %w", err)
	}

	d2, err := ParseDuration(req.SecondDuration)
	if err != nil {
		return fmt.Errorf("second_duration: %w", err)
	}

	// Validate duration consistency (both must be same type)
	if d1.IsTickBased != d2.IsTickBased {
		return fmt.Errorf("durations must be same type (both time-based or both tick-based): %w", pricer.ErrInvalidDuration)
	}

	// Validate duration order: second > first
	if d1.IsTickBased {
		if d2.Value <= d1.Value {
			return fmt.Errorf("second_duration (%dt) must be greater than first_duration (%dt): %w",
				d2.Value, d1.Value, pricer.ErrInvalidDuration)
		}
		// Validate gap: at least 2 ticks
		gap := d2.Value - d1.Value
		if gap < 2 {
			return fmt.Errorf("duration gap (%d ticks) must be at least 2 ticks: %w",
				gap, pricer.ErrInvalidDuration)
		}
	} else {
		s1 := d1.ToSeconds()
		s2 := d2.ToSeconds()
		if s2 <= s1 {
			return fmt.Errorf("second_duration (%ds) must be greater than first_duration (%ds): %w",
				s2, s1, pricer.ErrInvalidDuration)
		}
		// Validate gap: at least 10 seconds
		gap := s2 - s1
		if gap < 10 {
			return fmt.Errorf("duration gap (%d seconds) must be at least 10 seconds: %w",
				gap, pricer.ErrInvalidDuration)
		}
	}

	// Validate pricing time (if provided)
	if req.PricingTime > 0 {
		now := time.Now().Unix()
		if req.PricingTime > now {
			return fmt.Errorf("pricing_time cannot be in the future: %w", pricer.ErrInvalidDuration)
		}
	}

	return nil
}

// ValidateBidRequest validates parameters for a Bid request.
func (v *Validator) ValidateBidRequest(ctx context.Context, req *pricer.BidRequest) error {
	// Validate symbol
	if !supportedSymbols[req.Symbol] {
		return fmt.Errorf("symbol %s: %w", req.Symbol, pricer.ErrInvalidSymbol)
	}

	// Validate required fields for Bid
	if req.StartTime == 0 {
		return pricer.ErrMissingStartTime
	}

	if req.Payout == "" {
		return pricer.ErrMissingPayout
	}

	// Validate payout format
	payout, err := strconv.ParseFloat(req.Payout, 64)
	if err != nil {
		return fmt.Errorf("invalid payout format: %w", pricer.ErrInvalidStake)
	}

	if payout <= 0 {
		return fmt.Errorf("payout must be positive: %w", pricer.ErrInvalidStake)
	}

	// Get symbol config for validation
	cfg, err := v.config.GetSymbolConfig(req.Symbol)
	if err != nil {
		return err
	}

	// Check payout against max
	if payout > cfg.MaxPayout {
		return fmt.Errorf("payout %.2f exceeds maximum %.2f: %w",
			payout, cfg.MaxPayout, pricer.ErrPayoutExceeded)
	}

	// Validate durations
	d1, err := ParseDuration(req.FirstDuration)
	if err != nil {
		return fmt.Errorf("first_duration: %w", err)
	}

	d2, err := ParseDuration(req.SecondDuration)
	if err != nil {
		return fmt.Errorf("second_duration: %w", err)
	}

	// Validate duration consistency
	if d1.IsTickBased != d2.IsTickBased {
		return fmt.Errorf("durations must be same type: %w", pricer.ErrInvalidDuration)
	}

	// Validate duration order
	if d1.IsTickBased {
		if d2.Value <= d1.Value {
			return fmt.Errorf("second_duration must be greater than first_duration: %w",
				pricer.ErrInvalidDuration)
		}
	} else {
		if d2.ToSeconds() <= d1.ToSeconds() {
			return fmt.Errorf("second_duration must be greater than first_duration: %w",
				pricer.ErrInvalidDuration)
		}
	}

	// Validate pricing time (if provided)
	if req.PricingTime > 0 {
		now := time.Now().Unix()
		if req.PricingTime > now {
			return fmt.Errorf("pricing_time cannot be in the future: %w", pricer.ErrInvalidDuration)
		}
	}

	return nil
}

// ParseDuration wraps the package-level ParseDuration function.
func (v *Validator) ParseDuration(s string) (pricer.Duration, error) {
	return ParseDuration(s)
}
