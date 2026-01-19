package common

import "errors"

// Standard error definitions for the arcade service
var (
	// ErrAccountNotFound indicates the requested account does not exist
	ErrAccountNotFound = errors.New("account not found")

	// ErrInsufficientBalance indicates balance is less than requested amount
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrInvalidAmount indicates the amount is not valid
	ErrInvalidAmount = errors.New("invalid amount")

	// ErrInvalidCurrency indicates invalid currency code format
	ErrInvalidCurrency = errors.New("invalid currency")

	// ErrInvalidSeriesType indicates series type not recognized
	ErrInvalidSeriesType = errors.New("invalid series type")

	// ErrInvalidSentiment indicates sentiment not rise/fall
	ErrInvalidSentiment = errors.New("invalid sentiment")

	// ErrInvalidQuote indicates previous_quote mismatch
	ErrInvalidQuote = errors.New("invalid quote")

	// ErrDatabaseError indicates a database operation failure
	ErrDatabaseError = errors.New("database error")
)

// APIError represents a structured error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError creates a new API error
func NewAPIError(code, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// Error code constants
const (
	ErrCodeAccountNotFound     = "ACCOUNT_NOT_FOUND"
	ErrCodeInvalidCurrency     = "INVALID_CURRENCY"
	ErrCodeInvalidAmount       = "INVALID_AMOUNT"
	ErrCodeInvalidStake        = "INVALID_STAKE"
	ErrCodeInsufficientBalance = "INSUFFICIENT_BALANCE"
	ErrCodeInvalidSeriesType   = "INVALID_SERIES_TYPE"
	ErrCodeInvalidSentiment    = "INVALID_SENTIMENT"
	ErrCodeInvalidQuote        = "INVALID_QUOTE"
)
