package accounts

import (
	"time"

	"github.com/shopspring/decimal"
)

// Account represents a trader's account
type Account struct {
	AccountID  string          `json:"account_id"`
	ExternalID *string         `json:"external_id,omitempty"`
	Currency   string          `json:"currency"`
	Balance    decimal.Decimal `json:"balance"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Transaction represents a financial movement
type Transaction struct {
	TransactionID   int64           `json:"transaction_id"`
	AccountID       string          `json:"account_id"`
	Type            TransactionType `json:"type"`
	Amount          decimal.Decimal `json:"amount"` // Positive for DEPOSIT/SELL, Negative for WITHDRAWAL/BUY
	IdempotencyID   *string         `json:"idempotency_id,omitempty"`
	ContractID      *int64          `json:"contract_id,omitempty"` // FK to contracts (for BUY/SELL)
	TransactionTime time.Time       `json:"transaction_time"`
}

// TransactionType enumeration
type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "DEPOSIT"
	TransactionTypeWithdrawal TransactionType = "WITHDRAWAL"
	TransactionTypeBuy        TransactionType = "BUY"
	TransactionTypeSell       TransactionType = "SELL"
)

// CreateAccountRequest represents account creation input
type CreateAccountRequest struct {
	Currency   string  `json:"currency"`
	ExternalID *string `json:"external_id,omitempty"`
}

// DepositRequest represents deposit input
type DepositRequest struct {
	Amount    string `json:"amount"`
	DepositID string `json:"deposit_id"`
}

// WithdrawalRequest represents withdrawal input
type WithdrawalRequest struct {
	Amount       string `json:"amount"`
	WithdrawalID string `json:"withdrawal_id"`
}

// TransactionResponse represents transaction result
type TransactionResponse struct {
	Balance         string    `json:"balance"`
	TransactionID   int64     `json:"transaction_id"`
	TransactionTime time.Time `json:"transaction_time"`
}
