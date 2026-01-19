package common

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// ParseAmount parses string to decimal with validation
func ParseAmount(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid decimal format: %w", err)
	}
	return d, nil
}

// FormatAmount formats decimal to string with 2 decimal places (for monetary amounts)
func FormatAmount(d decimal.Decimal) string {
	return d.StringFixed(2)
}

// FormatPrice formats decimal to string with 3 decimal places (for OHLC prices)
func FormatPrice(d decimal.Decimal) string {
	return d.StringFixed(3)
}

// ValidatePositiveAmount validates that amount is positive
func ValidatePositiveAmount(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}
	return nil
}

// ValidateNonNegativeAmount validates that amount is non-negative
func ValidateNonNegativeAmount(amount decimal.Decimal) error {
	if amount.LessThan(decimal.Zero) {
		return ErrInvalidAmount
	}
	return nil
}
