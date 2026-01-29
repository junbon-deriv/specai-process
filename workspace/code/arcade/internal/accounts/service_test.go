package accounts

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/deriv/arcade/internal/common"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// fakeAccountRepository is a fake in-memory implementation of Repository for testing
type fakeAccountRepository struct {
	mu           sync.RWMutex
	accounts     map[string]*Account
	transactions map[int64]*Transaction
	nextTxnID    int64
	err          error // For error injection
}

// newFakeAccountRepository creates a new fake repository
func newFakeAccountRepository() *fakeAccountRepository {
	return &fakeAccountRepository{
		accounts:     make(map[string]*Account),
		transactions: make(map[int64]*Transaction),
		nextTxnID:    1,
	}
}

func (f *fakeAccountRepository) CreateAccount(ctx context.Context, currency string, externalID *string) (*Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	accountID := fmt.Sprintf("SW%d", len(f.accounts)+1)
	account := &Account{
		AccountID:  accountID,
		ExternalID: externalID,
		Currency:   currency,
		Balance:    decimal.Zero,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	f.accounts[accountID] = account
	return account, nil
}

func (f *fakeAccountRepository) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	account, exists := f.accounts[accountID]
	if !exists {
		return nil, common.ErrAccountNotFound
	}

	// Return a copy
	return &Account{
		AccountID:  account.AccountID,
		ExternalID: account.ExternalID,
		Currency:   account.Currency,
		Balance:    account.Balance,
		CreatedAt:  account.CreatedAt,
		UpdatedAt:  account.UpdatedAt,
	}, nil
}

func (f *fakeAccountRepository) DepositFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*DepositResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	// Check idempotency
	idempStr := idempotencyID.String()
	for _, txn := range f.transactions {
		if txn.IdempotencyID != nil && *txn.IdempotencyID == idempStr && txn.Type == TransactionTypeDeposit {
			account := f.accounts[accountID]
			return &DepositResult{
				Transaction: txn,
				NewBalance:  account.Balance,
			}, nil
		}
	}

	account, exists := f.accounts[accountID]
	if !exists {
		return nil, common.ErrAccountNotFound
	}

	// Update balance
	account.Balance = account.Balance.Add(amount)
	account.UpdatedAt = time.Now()

	// Create transaction
	txnID := f.nextTxnID
	f.nextTxnID++

	transaction := &Transaction{
		TransactionID:   txnID,
		AccountID:       accountID,
		Type:            TransactionTypeDeposit,
		Amount:          amount,
		IdempotencyID:   &idempStr,
		TransactionTime: time.Now(),
	}

	f.transactions[txnID] = transaction

	return &DepositResult{
		Transaction: transaction,
		NewBalance:  account.Balance,
	}, nil
}

func (f *fakeAccountRepository) WithdrawFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*WithdrawalResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.err; err != nil {
		f.err = nil
		return nil, err
	}

	// Check idempotency
	idempStr := idempotencyID.String()
	for _, txn := range f.transactions {
		if txn.IdempotencyID != nil && *txn.IdempotencyID == idempStr && txn.Type == TransactionTypeWithdrawal {
			account := f.accounts[accountID]
			return &WithdrawalResult{
				Transaction: txn,
				NewBalance:  account.Balance,
			}, nil
		}
	}

	account, exists := f.accounts[accountID]
	if !exists {
		return nil, common.ErrAccountNotFound
	}

	// Check balance
	if account.Balance.LessThan(amount) {
		return nil, common.ErrInsufficientBalance
	}

	// Update balance
	account.Balance = account.Balance.Sub(amount)
	account.UpdatedAt = time.Now()

	// Create transaction (negative amount)
	txnID := f.nextTxnID
	f.nextTxnID++

	transaction := &Transaction{
		TransactionID:   txnID,
		AccountID:       accountID,
		Type:            TransactionTypeWithdrawal,
		Amount:          amount.Neg(),
		IdempotencyID:   &idempStr,
		TransactionTime: time.Now(),
	}

	f.transactions[txnID] = transaction

	return &WithdrawalResult{
		Transaction: transaction,
		NewBalance:  account.Balance,
	}, nil
}

// TestDeposit_Success tests successful deposit
func TestDeposit_Success(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	// Create account first
	account, err := service.CreateAccount(context.Background(), CreateAccountRequest{Currency: "USD"})
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	depositID := uuid.New().String()
	result, err := service.Deposit(context.Background(), account.AccountID, decimal.NewFromInt(100), depositID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.NewBalance.Cmp(decimal.NewFromInt(100)) != 0 {
		t.Errorf("Expected balance 100, got %v", result.NewBalance)
	}

	if result.Transaction.Type != TransactionTypeDeposit {
		t.Errorf("Expected DEPOSIT type, got %v", result.Transaction.Type)
	}

	if result.Transaction.Amount.Cmp(decimal.NewFromInt(100)) != 0 {
		t.Errorf("Expected amount 100, got %v", result.Transaction.Amount)
	}
}

// TestDeposit_Idempotency tests deposit idempotency
func TestDeposit_Idempotency(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	account, _ := service.CreateAccount(context.Background(), CreateAccountRequest{Currency: "USD"})

	depositID := uuid.New().String()
	result1, _ := service.Deposit(context.Background(), account.AccountID, decimal.NewFromInt(100), depositID)

	// Deposit again with same ID
	result2, err := service.Deposit(context.Background(), account.AccountID, decimal.NewFromInt(100), depositID)

	if err != nil {
		t.Fatalf("Expected no error on idempotent deposit, got %v", err)
	}

	if result1.Transaction.TransactionID != result2.Transaction.TransactionID {
		t.Errorf("Idempotent deposit should return same transaction ID")
	}

	if result2.NewBalance.Cmp(decimal.NewFromInt(100)) != 0 {
		t.Errorf("Balance should be 100 (not doubled), got %v", result2.NewBalance)
	}
}

// TestWithdraw_InsufficientBalance tests withdrawal with insufficient balance
func TestWithdraw_InsufficientBalance(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	account, _ := service.CreateAccount(context.Background(), CreateAccountRequest{Currency: "USD"})

	withdrawalID := uuid.New().String()
	_, err := service.Withdraw(context.Background(), account.AccountID, decimal.NewFromInt(50), withdrawalID)

	if err != common.ErrInsufficientBalance {
		t.Errorf("Expected ErrInsufficientBalance, got %v", err)
	}

	// Verify balance unchanged
	acct, _ := service.GetAccount(context.Background(), account.AccountID)
	if acct.Balance.Cmp(decimal.Zero) != 0 {
		t.Errorf("Balance should be 0, got %v", acct.Balance)
	}
}

// TestWithdraw_Success tests successful withdrawal
func TestWithdraw_Success(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	account, _ := service.CreateAccount(context.Background(), CreateAccountRequest{Currency: "USD"})
	depositID := uuid.New().String()
	service.Deposit(context.Background(), account.AccountID, decimal.NewFromInt(100), depositID)

	withdrawalID := uuid.New().String()
	result, err := service.Withdraw(context.Background(), account.AccountID, decimal.NewFromInt(50), withdrawalID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.NewBalance.Cmp(decimal.NewFromInt(50)) != 0 {
		t.Errorf("Expected balance 50, got %v", result.NewBalance)
	}

	if result.Transaction.Type != TransactionTypeWithdrawal {
		t.Errorf("Expected WITHDRAWAL type, got %v", result.Transaction.Type)
	}

	// Amount should be negative for withdrawal
	if result.Transaction.Amount.Cmp(decimal.NewFromInt(-50)) != 0 {
		t.Errorf("Expected amount -50, got %v", result.Transaction.Amount)
	}
}

// TestCreateAccount_InvalidCurrency tests account creation with invalid currency
func TestCreateAccount_InvalidCurrency(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	_, err := service.CreateAccount(context.Background(), CreateAccountRequest{Currency: "INVALID"})

	if err != common.ErrInvalidCurrency {
		t.Errorf("Expected ErrInvalidCurrency, got %v", err)
	}
}

// TestGetAccount_NotFound tests account retrieval when not found
func TestGetAccount_NotFound(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	_, err := service.GetAccount(context.Background(), "SW999")

	if err != common.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

// TestDeposit_AccountNotFound tests deposit to non-existent account
func TestDeposit_AccountNotFound(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	depositID := uuid.New().String()
	_, err := service.Deposit(context.Background(), "SW999", decimal.NewFromInt(100), depositID)

	if err != common.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

// TestDeposit_InvalidAmount tests deposit with invalid amount
func TestDeposit_InvalidAmount(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	depositID := uuid.New().String()
	_, err := service.Deposit(context.Background(), "SW123", decimal.NewFromInt(-100), depositID)

	if err == nil {
		t.Fatal("Expected error for negative amount")
	}
}

// TestDeposit_InvalidDepositID tests deposit with invalid deposit ID
func TestDeposit_InvalidDepositID(t *testing.T) {
	store := newFakeAccountRepository()
	service := NewService(store)

	_, err := service.Deposit(context.Background(), "SW123", decimal.NewFromInt(100), "invalid-uuid")

	if err == nil {
		t.Fatal("Expected error for invalid deposit ID")
	}
}
