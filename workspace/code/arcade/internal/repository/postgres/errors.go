package postgres

import (
	"fmt"

	"github.com/deriv/arcade/internal/common"
)

// mapPgError converts PostgreSQL error codes to domain errors
func mapPgError(err error) error {
	errMsg := err.Error()

	// P0001 - Account not found
	if contains(errMsg, "Account not found") || contains(errMsg, "P0001") {
		return common.ErrAccountNotFound
	}

	// P0002 - Insufficient balance
	if contains(errMsg, "Insufficient balance") || contains(errMsg, "P0002") {
		return common.ErrInsufficientBalance
	}

	// P0003 - Invalid amount
	if contains(errMsg, "Invalid") || contains(errMsg, "P0003") {
		return fmt.Errorf("invalid amount: %w", err)
	}

	// P0005 - Price series not found
	if contains(errMsg, "Price series") || contains(errMsg, "P0005") {
		return common.ErrInvalidQuote
	}

	// P0008 - Contract not found
	if contains(errMsg, "Contract not found") || contains(errMsg, "P0008") {
		return fmt.Errorf("contract not found: %w", err)
	}

	// P0010 - Contract already settled
	if contains(errMsg, "already settled") || contains(errMsg, "P0010") {
		return fmt.Errorf("contract already settled: %w", err)
	}

	return fmt.Errorf("database error: %w", err)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
