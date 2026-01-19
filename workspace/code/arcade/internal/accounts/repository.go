package accounts

import (
	"context"
	"fmt"

	"github.com/deriv/arcade/internal/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Repository handles database operations for accounts
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new accounts repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateAccount creates a new account with SW-prefixed ID
func (r *Repository) CreateAccount(ctx context.Context, currency string, externalID *string) (*Account, error) {
	query := `
		INSERT INTO accounts (account_id, currency, external_id, balance, created_at, updated_at)
		VALUES ('SW' || nextval('account_id_seq')::text, $1, $2, 0.00, NOW(), NOW())
		RETURNING account_id, external_id, currency, balance, created_at, updated_at
	`

	var account Account
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
func (r *Repository) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	query := `
		SELECT account_id, external_id, currency, balance, created_at, updated_at
		FROM accounts
		WHERE account_id = $1
	`

	var account Account
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

// GetAccountBalance retrieves account balance
func (r *Repository) GetAccountBalance(ctx context.Context, accountID string) (decimal.Decimal, error) {
	query := `SELECT balance FROM accounts WHERE account_id = $1`

	var balance decimal.Decimal
	err := r.pool.QueryRow(ctx, query, accountID).Scan(&balance)

	if err == pgx.ErrNoRows {
		return decimal.Zero, common.ErrAccountNotFound
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}
