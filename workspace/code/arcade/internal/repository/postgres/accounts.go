package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// AccountsRepository implements accounts.Repository interface
type AccountsRepository struct {
	pool *pgxpool.Pool
}

// NewAccountsRepository creates a new accounts repository
func NewAccountsRepository(pool *pgxpool.Pool) *AccountsRepository {
	return &AccountsRepository{pool: pool}
}

// CreateAccount creates a new account with SW-prefixed ID
func (r *AccountsRepository) CreateAccount(ctx context.Context, currency string, externalID *string) (*accounts.Account, error) {
	query := `
		INSERT INTO accounts (account_id, currency, external_id, balance, created_at, updated_at)
		VALUES ('SW' || nextval('account_id_seq')::text, $1, $2, 0.00, NOW(), NOW())
		RETURNING account_id, external_id, currency, balance, created_at, updated_at
	`

	var account accounts.Account
	err := r.pool.QueryRow(ctx, query, currency, externalID).Scan(
		&account.AccountID,
		&account.ExternalID,
		&account.Currency,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &account, nil
}

// GetAccount retrieves account by ID
func (r *AccountsRepository) GetAccount(ctx context.Context, accountID string) (*accounts.Account, error) {
	query := `
		SELECT account_id, external_id, currency, balance, created_at, updated_at
		FROM accounts
		WHERE account_id = $1
	`

	var account accounts.Account
	err := r.pool.QueryRow(ctx, query, accountID).Scan(
		&account.AccountID,
		&account.ExternalID,
		&account.Currency,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, common.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// DepositFunds calls deposit_funds stored procedure
func (r *AccountsRepository) DepositFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*accounts.DepositResult, error) {
	var txnID int64
	var newBalance decimal.Decimal
	var isDuplicate bool
	var txnTime time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT transaction_id, new_balance, is_duplicate, transaction_time
		FROM deposit_funds($1, $2, $3)
	`, accountID, amount, idempotencyID).Scan(&txnID, &newBalance, &isDuplicate, &txnTime)

	if err != nil {
		return nil, mapPgError(err)
	}

	// Build transaction response
	idempStr := idempotencyID.String()
	transaction := &accounts.Transaction{
		TransactionID:   txnID,
		AccountID:       accountID,
		Type:            accounts.TransactionTypeDeposit,
		Amount:          amount,
		IdempotencyID:   &idempStr,
		TransactionTime: txnTime,
	}

	return &accounts.DepositResult{
		Transaction: transaction,
		NewBalance:  newBalance,
	}, nil
}

// WithdrawFunds calls withdraw_funds stored procedure
func (r *AccountsRepository) WithdrawFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*accounts.WithdrawalResult, error) {
	var txnID int64
	var newBalance decimal.Decimal
	var isDuplicate bool
	var txnTime time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT transaction_id, new_balance, is_duplicate, transaction_time
		FROM withdraw_funds($1, $2, $3)
	`, accountID, amount, idempotencyID).Scan(&txnID, &newBalance, &isDuplicate, &txnTime)

	if err != nil {
		return nil, mapPgError(err)
	}

	// Build transaction response (amount is stored as negative in DB)
	idempStr := idempotencyID.String()
	transaction := &accounts.Transaction{
		TransactionID:   txnID,
		AccountID:       accountID,
		Type:            accounts.TransactionTypeWithdrawal,
		Amount:          amount.Neg(), // Negative for withdrawal
		IdempotencyID:   &idempStr,
		TransactionTime: txnTime,
	}

	return &accounts.WithdrawalResult{
		Transaction: transaction,
		NewBalance:  newBalance,
	}, nil
}
