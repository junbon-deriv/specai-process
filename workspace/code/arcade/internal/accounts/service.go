package accounts

import (
	"context"
	"fmt"
	"time"

	"github.com/deriv/arcade/internal/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Service implements account business logic
type Service struct {
	repo *Repository
	pool *pgxpool.Pool
}

// NewService creates a new accounts service
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		repo: NewRepository(pool),
		pool: pool,
	}
}

// CreateAccount creates a new trading account
func (s *Service) CreateAccount(ctx context.Context, req CreateAccountRequest) (*Account, error) {
	// Validate currency format
	if err := common.ValidateCurrency(req.Currency); err != nil {
		return nil, err
	}

	// Create account
	account, err := s.repo.CreateAccount(ctx, req.Currency, req.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account, nil
}

// GetAccount retrieves account details
func (s *Service) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	account, err := s.repo.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return account, nil
}

// GetAccountBalance retrieves current balance
func (s *Service) GetAccountBalance(ctx context.Context, accountID string) (decimal.Decimal, error) {
	balance, err := s.repo.GetAccountBalance(ctx, accountID)
	if err != nil {
		return decimal.Zero, err
	}

	return balance, nil
}

// Deposit credits funds to account with idempotency (uses deposit_funds stored procedure)
func (s *Service) Deposit(ctx context.Context, accountID string, amount decimal.Decimal, depositID string) (*Transaction, error) {
	// Validate amount
	if err := common.ValidatePositiveAmount(amount); err != nil {
		return nil, err
	}

	// Parse deposit ID as UUID
	idempUUID, err := uuid.Parse(depositID)
	if err != nil {
		return nil, fmt.Errorf("invalid deposit_id format: %w", err)
	}

	// Call deposit_funds stored procedure
	var txnID int64
	var newBalance decimal.Decimal
	var isDuplicate bool
	var txnTime time.Time

	err = s.pool.QueryRow(ctx, `
		SELECT transaction_id, new_balance, is_duplicate, transaction_time
		FROM deposit_funds($1, $2, $3)
	`, accountID, amount, idempUUID).Scan(&txnID, &newBalance, &isDuplicate, &txnTime)
	
	if err != nil {
		return nil, s.mapPgError(err)
	}

	// Build transaction response
	transaction := &Transaction{
		TransactionID:   txnID,
		AccountID:       accountID,
		Type:            TransactionTypeDeposit,
		Amount:          amount,
		IdempotencyID:   &depositID,
		TransactionTime: txnTime,
	}

	return transaction, nil
}

// Withdraw debits funds from account with idempotency (uses withdraw_funds stored procedure)
func (s *Service) Withdraw(ctx context.Context, accountID string, amount decimal.Decimal, withdrawalID string) (*Transaction, error) {
	// Validate amount
	if err := common.ValidatePositiveAmount(amount); err != nil {
		return nil, err
	}

	// Parse withdrawal ID as UUID
	idempUUID, err := uuid.Parse(withdrawalID)
	if err != nil {
		return nil, fmt.Errorf("invalid withdrawal_id format: %w", err)
	}

	// Call withdraw_funds stored procedure
	var txnID int64
	var newBalance decimal.Decimal
	var isDuplicate bool
	var txnTime time.Time

	err = s.pool.QueryRow(ctx, `
		SELECT transaction_id, new_balance, is_duplicate, transaction_time
		FROM withdraw_funds($1, $2, $3)
	`, accountID, amount, idempUUID).Scan(&txnID, &newBalance, &isDuplicate, &txnTime)
	
	if err != nil {
		return nil, s.mapPgError(err)
	}

	// Build transaction response (amount is stored as negative in DB)
	transaction := &Transaction{
		TransactionID:   txnID,
		AccountID:       accountID,
		Type:            TransactionTypeWithdrawal,
		Amount:          amount.Neg(), // Negative for withdrawal
		IdempotencyID:   &withdrawalID,
		TransactionTime: txnTime,
	}

	return transaction, nil
}

// mapPgError converts PostgreSQL error codes to API errors
func (s *Service) mapPgError(err error) error {
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
