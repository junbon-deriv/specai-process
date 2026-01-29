package accounts

import (
	"context"
	"fmt"

	"github.com/deriv/arcade/internal/common"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Repository interface for account data operations
type Repository interface {
	// CreateAccount creates a new account with SW-prefixed ID
	CreateAccount(ctx context.Context, currency string, externalID *string) (*Account, error)

	// GetAccount retrieves account by ID
	GetAccount(ctx context.Context, accountID string) (*Account, error)

	// DepositFunds calls deposit_funds stored procedure
	DepositFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*DepositResult, error)

	// WithdrawFunds calls withdraw_funds stored procedure
	WithdrawFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*WithdrawalResult, error)
}

// Service implements account business logic
type Service struct {
	repo Repository
}

// NewService creates a new accounts service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
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

// Deposit credits funds to account with idempotency (uses deposit_funds stored procedure)
func (s *Service) Deposit(ctx context.Context, accountID string, amount decimal.Decimal, depositID string) (*DepositResult, error) {
	// Validate amount
	if err := common.ValidatePositiveAmount(amount); err != nil {
		return nil, err
	}

	// Parse deposit ID as UUID
	idempUUID, err := uuid.Parse(depositID)
	if err != nil {
		return nil, fmt.Errorf("invalid deposit_id format: %w", err)
	}

	// Call repository method (wraps stored procedure)
	return s.repo.DepositFunds(ctx, accountID, amount, idempUUID)
}

// Withdraw debits funds from account with idempotency (uses withdraw_funds stored procedure)
func (s *Service) Withdraw(ctx context.Context, accountID string, amount decimal.Decimal, withdrawalID string) (*WithdrawalResult, error) {
	// Validate amount
	if err := common.ValidatePositiveAmount(amount); err != nil {
		return nil, err
	}

	// Parse withdrawal ID as UUID
	idempUUID, err := uuid.Parse(withdrawalID)
	if err != nil {
		return nil, fmt.Errorf("invalid withdrawal_id format: %w", err)
	}

	// Call repository method (wraps stored procedure)
	return s.repo.WithdrawFunds(ctx, accountID, amount, idempUUID)
}
