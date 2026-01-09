package contract

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// BarrierType defines the type of barrier specification
type BarrierType int

const (
	// BarrierTypeAbsolute represents an absolute barrier value (e.g., "100.50")
	BarrierTypeAbsolute BarrierType = iota
	// BarrierTypeRelativePlus represents a positive relative barrier (e.g., "+0.50")
	BarrierTypeRelativePlus
	// BarrierTypeRelativeMinus represents a negative relative barrier (e.g., "-0.50")
	BarrierTypeRelativeMinus
	// BarrierTypeATM represents at-the-money (default when no barrier specified)
	BarrierTypeATM
)

// Barrier represents a resolved barrier value
type Barrier struct {
	InputValue    *string
	ResolvedValue decimal.Decimal
	Type          BarrierType
}

// ErrInvalidBarrier is returned when barrier format is invalid
var ErrInvalidBarrier = fmt.Errorf("invalid barrier format")

// ResolveBarrier resolves barrier based on input and entry price
// Returns proper error on invalid format (no silent fallbacks)
func ResolveBarrier(input *string, entryPrice decimal.Decimal) (*Barrier, error) {
	if input == nil || *input == "" {
		// ATM barrier - equals entry price
		return &Barrier{
			InputValue:    nil,
			ResolvedValue: entryPrice,
			Type:          BarrierTypeATM,
		}, nil
	}

	value := strings.TrimSpace(*input)
	if value == "" {
		// Empty string after trimming - ATM
		return &Barrier{
			InputValue:    nil,
			ResolvedValue: entryPrice,
			Type:          BarrierTypeATM,
		}, nil
	}

	if strings.HasPrefix(value, "+") {
		// Relative positive
		offset, err := decimal.NewFromString(value[1:])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid relative barrier format", ErrInvalidBarrier)
		}
		return &Barrier{
			InputValue:    input,
			ResolvedValue: entryPrice.Add(offset),
			Type:          BarrierTypeRelativePlus,
		}, nil
	}

	if strings.HasPrefix(value, "-") {
		// Relative negative
		offset, err := decimal.NewFromString(value[1:])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid relative barrier format", ErrInvalidBarrier)
		}
		return &Barrier{
			InputValue:    input,
			ResolvedValue: entryPrice.Sub(offset),
			Type:          BarrierTypeRelativeMinus,
		}, nil
	}

	// Absolute barrier
	resolved, err := decimal.NewFromString(value)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid absolute barrier format", ErrInvalidBarrier)
	}
	return &Barrier{
		InputValue:    input,
		ResolvedValue: resolved,
		Type:          BarrierTypeAbsolute,
	}, nil
}
