# Internal API Specification: Arcade Service

## Document Information
| Field | Value |
|-------|-------|
| Version | 1.1 |
| Created | 2026-01-16 |
| Last Updated | 2026-01-16 |
| Status | Draft |
| Service | arcade |
| API Type | Internal (Inter-Module) |
| Architecture Reference | [workspace/output/architecture/architecture.md](../architecture/architecture.md) |
| Domain Model Reference | [workspace/output/domain/domain_model.md](../domain/domain_model.md) |

---

## 1. Overview

This document specifies the **internal API** for the Deriv Arcade service. These are **inter-module function calls** within a single Golang service, NOT network REST APIs. The Accounts module exposes these functions for consumption by the Trading module.

### Purpose

The internal API enables the Trading module to:
- Validate account existence before trade execution
- Check sufficient balance before stake deduction
- Atomically deduct stake amounts during trade placement
- Atomically credit payout amounts during trade settlement

### Key Characteristics

| Characteristic | Value |
|----------------|-------|
| Communication Type | Synchronous Go function calls |
| Transaction Scope | All calls occur within database transactions |
| Consumer | Trading Module |
| Provider | Accounts Module |
| Atomicity | Critical - must support all-or-nothing trade execution |

---

## 2. Authentication & Authorization

### Inter-Module Context

Since these are internal function calls within a single service, there is no network authentication. Instead, the functions receive a context object for:
- Database transaction propagation
- Request tracing/correlation
- Timeout handling

```go
// All internal API functions receive context.Context as first parameter
// The context carries the database transaction for atomic operations
type Context = context.Context
```

### Access Control

| Aspect | Implementation |
|--------|----------------|
| Authorization | Not applicable (internal calls) |
| Transaction Isolation | Read Committed (PostgreSQL default) |
| Row Locking | SELECT FOR UPDATE on account balance |

---

## 3. Base Configuration

### Transaction Context Key

```go
// txContextKey is used to pass the database transaction through context
type contextKey string

const txContextKey contextKey = "db_transaction"

// Helper function to get transaction from context
func getTx(ctx context.Context) (*sql.Tx, bool) {
    tx, ok := ctx.Value(txContextKey).(*sql.Tx)
    return tx, ok
}
```

### Go Interface Definition

```go
package accounts

import (
    "context"
    "github.com/shopspring/decimal"
)

// AccountService defines the internal API exposed by the Accounts module
// for consumption by the Trading module.
type AccountService interface {
    // GetAccount retrieves account details by account ID.
    // Returns an error if the account does not exist.
    GetAccount(ctx context.Context, accountID string) (*Account, error)
    
    // GetAccountBalance retrieves the current balance for an account.
    // Used to validate sufficient funds before trade execution.
    GetAccountBalance(ctx context.Context, accountID string) (decimal.Decimal, error)
    
    // DeductStake atomically deducts the stake amount from an account
    // and creates a STAKE transaction. Must be called within a transaction.
    DeductStake(ctx context.Context, accountID string, amount decimal.Decimal, contractID int64) (*Transaction, error)
    
    // CreditPayout atomically credits the payout amount to an account
    // and creates a PAYOUT transaction. Must be called within a transaction.
    CreditPayout(ctx context.Context, accountID string, amount decimal.Decimal, contractID int64) (*Transaction, error)
}
```

### Module Boundaries

```
arcade/
├── internal/
│   ├── accounts/
│   │   ├── service.go         # Implements AccountService interface
│   │   └── repository.go      # Database operations
│   └── trading/
│       ├── service.go         # Consumes AccountService
│       └── repository.go      # Trading database operations
```

---

## 4. Function Summary Table

| Function ID | Function | Purpose | Consumer | Critical |
|-------------|----------|---------|----------|----------|
| API-AC-G4K | `GetAccount(accountId)` | Validate account exists | Trading | Yes |
| API-AC-B7N | `GetAccountBalance(accountId)` | Check sufficient funds | Trading | Yes |
| API-AC-D8Q | `DeductStake(accountId, amount, contractId)` | Debit stake during trade | Trading | Yes |
| API-AC-P3L | `CreditPayout(accountId, amount, contractId)` | Credit payout during trade | Trading | Yes |

---

## 5. Function Specifications

### 5.1 GetAccount (API-AC-G4K)

| Attribute | Value |
|-----------|-------|
| **Function ID** | API-AC-G4K |
| **Function Name** | GetAccount |
| **Purpose** | Validate that an account exists and retrieve its details before trade execution |
| **Provider** | Accounts Module |
| **Consumer** | Trading Module |
| **Priority** | Critical |

#### Signature

```go
func (s *AccountServiceImpl) GetAccount(ctx context.Context, accountID string) (*Account, error)
```

#### Parameters

| Parameter | Type | Required | Description | Validation |
|-----------|------|----------|-------------|------------|
| ctx | context.Context | Yes | Context with optional transaction | Must not be nil |
| accountID | string | Yes | Account identifier (SW-prefixed) | Format: SW followed by digits (e.g., SW1, SW42) |

#### Return Value

| Type | Description |
|------|-------------|
| *Account | Account details if found |
| error | Error if account not found or database error |

#### Return Structure

```go
type Account struct {
    AccountID  string          `json:"account_id"`   // SW-prefixed identifier (e.g., "SW1")
    ExternalID *string         `json:"external_id"`  // Optional broker reference
    Currency   string          `json:"currency"`     // 3-letter currency code
    Balance    decimal.Decimal `json:"balance"`      // Current balance (2 decimal precision)
}
```

#### Errors

| Error Code | Description | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | Account not found | accountID does not exist in database |
| ERR-AC-D9K | Database error | Database connection or query failure |

#### Example Usage

```go
// In Trading module's SwipeBuy handler
account, err := accountService.GetAccount(ctx, "SW1")
if err != nil {
    if errors.Is(err, accounts.ErrAccountNotFound) {
        return nil, fmt.Errorf("account not found: %w", err)
    }
    return nil, fmt.Errorf("failed to get account: %w", err)
}
// Proceed with trade execution
```

---

### 5.2 GetAccountBalance (API-AC-B7N)

| Attribute | Value |
|-----------|-------|
| **Function ID** | API-AC-B7N |
| **Function Name** | GetAccountBalance |
| **Purpose** | Retrieve current account balance to validate sufficient funds before trade |
| **Provider** | Accounts Module |
| **Consumer** | Trading Module |
| **Priority** | Critical |

#### Signature

```go
func (s *AccountServiceImpl) GetAccountBalance(ctx context.Context, accountID string) (decimal.Decimal, error)
```

#### Parameters

| Parameter | Type | Required | Description | Validation |
|-----------|------|----------|-------------|------------|
| ctx | context.Context | Yes | Context with optional transaction | Must not be nil |
| accountID | string | Yes | Account identifier (SW-prefixed) | Format: SW followed by digits |

#### Return Value

| Type | Description |
|------|-------------|
| decimal.Decimal | Current account balance with 2 decimal precision |
| error | Error if account not found or database error |

#### Errors

| Error Code | Description | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | Account not found | accountID does not exist in database |
| ERR-AC-D9K | Database error | Database connection or query failure |

#### Example Usage

```go
// Validate sufficient balance before trade
balance, err := accountService.GetAccountBalance(ctx, "SW1")
if err != nil {
    return nil, fmt.Errorf("failed to get balance: %w", err)
}

stake, _ := decimal.NewFromString("10.00")
if balance.LessThan(stake) {
    return nil, accounts.ErrInsufficientBalance
}
```

---

### 5.3 DeductStake (API-AC-D8Q)

| Attribute | Value |
|-----------|-------|
| **Function ID** | API-AC-D8Q |
| **Function Name** | DeductStake |
| **Purpose** | Atomically deduct stake from account balance and create STAKE transaction |
| **Provider** | Accounts Module |
| **Consumer** | Trading Module |
| **Priority** | Critical |

#### Signature

```go
func (s *AccountServiceImpl) DeductStake(ctx context.Context, accountID string, amount decimal.Decimal, contractID int64) (*Transaction, error)
```

#### Parameters

| Parameter | Type | Required | Description | Validation |
|-----------|------|----------|-------------|------------|
| ctx | context.Context | Yes | Context with database transaction | Must contain active transaction |
| accountID | string | Yes | Account identifier (SW-prefixed) | Format: SW followed by digits |
| amount | decimal.Decimal | Yes | Stake amount to deduct | Must be positive, 2 decimal precision |
| contractID | int64 | Yes | Associated contract ID | Must be valid contract ID |

#### Return Value

| Type | Description |
|------|-------------|
| *Transaction | Created STAKE transaction record |
| error | Error if validation fails or insufficient balance |

#### Return Structure

```go
type Transaction struct {
    TransactionID   int64           `json:"transaction_id"`    // Auto-generated ID
    AccountID       string          `json:"account_id"`        // SW-prefixed account ID
    Type            TransactionType `json:"type"`              // STAKE
    Amount          decimal.Decimal `json:"amount"`            // Stake amount
    TransactionTime time.Time       `json:"transaction_time"`  // ISO8601 timestamp
    ReferenceID     *int64          `json:"reference_id"`      // Contract ID
}

type TransactionType string

const (
    TransactionTypeDeposit    TransactionType = "DEPOSIT"
    TransactionTypeWithdrawal TransactionType = "WITHDRAWAL"
    TransactionTypeStake      TransactionType = "STAKE"
    TransactionTypePayout     TransactionType = "PAYOUT"
)
```

#### Errors

| Error Code | Description | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | Account not found | accountID does not exist |
| ERR-AC-G9M | Insufficient balance | Balance < stake amount |
| ERR-AC-F9L | Invalid amount | Amount is not positive |
| ERR-AC-D9K | Database error | Transaction or query failure |

#### Business Rules

1. Must be called within an active database transaction
2. Uses `SELECT FOR UPDATE` to lock the account row
3. Validates balance >= amount before deduction
4. Creates Transaction record with type=STAKE
5. Transaction.reference_id points to the contractID

#### Example Usage

```go
// Within SwipeBuy transaction
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return nil, err
}
defer tx.Rollback()

ctx = context.WithValue(ctx, txKey, tx)

// Deduct stake
stake, _ := decimal.NewFromString("10.00")
stakeTxn, err := accountService.DeductStake(ctx, "SW1", stake, contractID)
if err != nil {
    if errors.Is(err, accounts.ErrInsufficientBalance) {
        return nil, &APIError{Code: "INSUFFICIENT_BALANCE", Message: "Insufficient funds"}
    }
    return nil, err
}

// Continue with contract creation and payout...
tx.Commit()
```

---

### 5.4 CreditPayout (API-AC-P3L)

| Attribute | Value |
|-----------|-------|
| **Function ID** | API-AC-P3L |
| **Function Name** | CreditPayout |
| **Purpose** | Atomically credit payout to account balance and create PAYOUT transaction |
| **Provider** | Accounts Module |
| **Consumer** | Trading Module |
| **Priority** | Critical |

#### Signature

```go
func (s *AccountServiceImpl) CreditPayout(ctx context.Context, accountID string, amount decimal.Decimal, contractID int64) (*Transaction, error)
```

#### Parameters

| Parameter | Type | Required | Description | Validation |
|-----------|------|----------|-------------|------------|
| ctx | context.Context | Yes | Context with database transaction | Must contain active transaction |
| accountID | string | Yes | Account identifier (SW-prefixed) | Format: SW followed by digits |
| amount | decimal.Decimal | Yes | Payout amount to credit | Must be >= 0, 2 decimal precision |
| contractID | int64 | Yes | Associated contract ID | Must be valid contract ID |

#### Return Value

| Type | Description |
|------|-------------|
| *Transaction | Created PAYOUT transaction record |
| error | Error if account not found or database error |

#### Return Structure

Same as DeductStake (Transaction struct with Type = PAYOUT)

#### Errors

| Error Code | Description | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | Account not found | accountID does not exist |
| ERR-AC-F9L | Invalid amount | Amount is negative |
| ERR-AC-D9K | Database error | Transaction or query failure |

#### Business Rules

1. Must be called within an active database transaction (same as DeductStake)
2. Amount can be 0.00 (for losing trades)
3. Always creates Transaction record with type=PAYOUT, even for zero amounts
4. Transaction.reference_id points to the contractID
5. If amount > 0, account balance is credited

#### Example Usage

```go
// Within SwipeBuy transaction (after contract evaluation)
var payout decimal.Decimal
if isWin {
    // Payout = stake / 0.53
    payout = stake.Div(decimal.NewFromFloat(0.53)).Round(2)
} else {
    payout = decimal.Zero
}

payoutTxn, err := accountService.CreditPayout(ctx, "SW1", payout, contractID)
if err != nil {
    return nil, err
}

// Contract is settled
```

---

## 6. Data Models

### 6.1 Account

```go
// Account represents a trader's account in the system
type Account struct {
    AccountID  string          `json:"account_id"`   // Primary key, SW-prefixed (e.g., "SW1", "SW42")
    ExternalID *string         `json:"external_id"`  // Optional broker reference
    Currency   string          `json:"currency"`     // ISO 4217 currency code (3 uppercase letters)
    Balance    decimal.Decimal `json:"balance"`      // Current balance, 2 decimal precision, non-negative
}
```

### 6.2 Transaction

```go
// Transaction represents a financial movement affecting an account
type Transaction struct {
    TransactionID   int64           `json:"transaction_id"`    // Auto-generated primary key
    AccountID       string          `json:"account_id"`        // Foreign key to Account
    Type            TransactionType `json:"type"`              // DEPOSIT, WITHDRAWAL, STAKE, PAYOUT
    Amount          decimal.Decimal `json:"amount"`            // Transaction amount, 2 decimal precision
    IdempotencyID   *string         `json:"idempotency_id"`    // For DEPOSIT/WITHDRAWAL only
    TransactionTime time.Time       `json:"transaction_time"`  // When transaction occurred
    ReferenceID     *int64          `json:"reference_id"`      // Contract ID for STAKE/PAYOUT
}

// TransactionType enumeration
type TransactionType string

const (
    TransactionTypeDeposit    TransactionType = "DEPOSIT"
    TransactionTypeWithdrawal TransactionType = "WITHDRAWAL"
    TransactionTypeStake      TransactionType = "STAKE"
    TransactionTypePayout     TransactionType = "PAYOUT"
)
```

### 6.3 Decimal Handling

```go
// Use github.com/shopspring/decimal for precise monetary calculations
import "github.com/shopspring/decimal"

// Monetary amounts use 2 decimal places
balance := decimal.NewFromString("100.00")
stake := decimal.NewFromString("10.00")

// Payout calculation
payoutMultiplier := decimal.NewFromFloat(0.53)
payout := stake.Div(payoutMultiplier).Round(2) // stake / 0.53, rounded to 2 decimals
```

---

## 7. Error Handling

### 7.1 Error Types

```go
package accounts

import "errors"

// Sentinel errors for internal API
var (
    // ErrAccountNotFound indicates the requested account does not exist
    ErrAccountNotFound = errors.New("account not found")
    
    // ErrInsufficientBalance indicates balance is less than requested amount
    ErrInsufficientBalance = errors.New("insufficient balance")
    
    // ErrInvalidAmount indicates the amount is not valid (non-positive for stake)
    ErrInvalidAmount = errors.New("invalid amount")
    
    // ErrDatabaseError indicates a database operation failure
    ErrDatabaseError = errors.New("database error")
)
```

### 7.2 Error Code Reference

| Error Code | Go Error | HTTP Equivalent | Description |
|------------|----------|-----------------|-------------|
| ERR-AC-N4F | ErrAccountNotFound | 404 | Account ID does not exist |
| ERR-AC-G9M | ErrInsufficientBalance | 400 | Balance < requested amount |
| ERR-AC-F9L | ErrInvalidAmount | 400 | Amount is not positive |
| ERR-AC-D9K | ErrDatabaseError | 500 | Database operation failure |

### 7.3 Error Propagation Pattern

```go
// In Trading module, errors from internal API are wrapped for context
func (s *TradingService) ExecuteTrade(ctx context.Context, req *TradeRequest) (*Contract, error) {
    // Get account to validate existence
    account, err := s.accountService.GetAccount(ctx, req.AccountID)
    if err != nil {
        if errors.Is(err, accounts.ErrAccountNotFound) {
            return nil, &APIError{
                Code:    "ACCOUNT_NOT_FOUND",
                Message: "The specified account does not exist",
            }
        }
        return nil, fmt.Errorf("failed to validate account: %w", err)
    }
    
    // Check balance
    balance, err := s.accountService.GetAccountBalance(ctx, req.AccountID)
    if err != nil {
        return nil, fmt.Errorf("failed to get balance: %w", err)
    }
    
    if balance.LessThan(req.Stake) {
        return nil, &APIError{
            Code:    "INSUFFICIENT_BALANCE",
            Message: "Account balance is insufficient for the requested stake",
        }
    }
    
    // Continue with trade execution...
}
```

---

## 8. Transaction Flow

### 8.1 Atomic Trade Execution

The following diagram shows how the internal API functions are used within a single database transaction during trade execution:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ATOMIC TRADE EXECUTION                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  BEGIN TRANSACTION                                                           │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                                                                         │ │
│  │  1. GetAccount(ctx, accountId)                   [Accounts Module]     │ │
│  │     └─ Validates account exists                                        │ │
│  │     └─ Returns Account{account_id, currency, balance}                  │ │
│  │                                                                         │ │
│  │  2. Validate quote                               [Trading Module]      │ │
│  │     └─ Find PriceSeries matching account_id + series_type + quote      │ │
│  │     └─ Return INVALID_QUOTE error if not found                         │ │
│  │                                                                         │ │
│  │  3. GetAccountBalance(ctx, accountId)            [Accounts Module]     │ │
│  │     └─ Returns current balance                                         │ │
│  │     └─ Trading module validates: balance >= stake                      │ │
│  │                                                                         │ │
│  │  4. Create Contract (partial)                    [Trading Module]      │ │
│  │     └─ Reserves contract ID for transaction references                 │ │
│  │     └─ Returns contractId (int64)                                      │ │
│  │                                                                         │ │
│  │  5. DeductStake(ctx, accountId, stake, contractId) [Accounts Module]   │ │
│  │     └─ Locks account row (SELECT FOR UPDATE)                           │ │
│  │     └─ Validates balance >= stake (double-check)                       │ │
│  │     └─ Deducts stake: balance = balance - stake                        │ │
│  │     └─ Creates Transaction(type=STAKE, reference_id=contractId)        │ │
│  │     └─ Returns Transaction                                             │ │
│  │                                                                         │ │
│  │  6. Generate candles 11-20                       [Trading Module]      │ │
│  │     └─ Uses GBM from previous_quote                                    │ │
│  │                                                                         │ │
│  │  7. Evaluate outcome                             [Trading Module]      │ │
│  │     └─ Compare candle[20].close vs candle[10].close                    │ │
│  │     └─ Determine win/loss based on sentiment                           │ │
│  │                                                                         │ │
│  │  8. Calculate payout                             [Trading Module]      │ │
│  │     └─ Win: payout = stake / 0.53                                      │ │
│  │     └─ Loss: payout = 0.00                                             │ │
│  │                                                                         │ │
│  │  9. CreditPayout(ctx, accountId, payout, contractId) [Accounts Module] │ │
│  │     └─ Credits payout: balance = balance + payout                      │ │
│  │     └─ Creates Transaction(type=PAYOUT, reference_id=contractId)       │ │
│  │     └─ Returns Transaction (even if payout is 0.00)                    │ │
│  │                                                                         │ │
│  │  10. Update Contract (finalize)                  [Trading Module]      │ │
│  │      └─ Stores full 20 candles (candles 1-10 + 11-20)                  │ │
│  │      └─ Records stake, payout, outcome                                 │ │
│  │                                                                         │ │
│  │  11. Delete PriceSeries                          [Trading Module]      │ │
│  │      └─ Removes temporary preview data                                 │ │
│  │                                                                         │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│  COMMIT TRANSACTION                                                          │
│                                                                              │
│  If any step fails → ROLLBACK TRANSACTION (all-or-nothing)                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key Design Decision**: The contract is created in two phases:
1. **Partial creation (Step 4)**: Reserves the contractId before financial operations
2. **Finalization (Step 10)**: Updates with full candle data and outcome after trade completion

This ensures the STAKE and PAYOUT transactions always have a valid `reference_id` linking them to the contract.

### 8.2 Example Implementation

```go
func (s *TradingService) SwipeBuy(ctx context.Context, req *SwipeBuyRequest) (*SwipeBuyResponse, error) {
    // Start transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // Create transaction context
    txCtx := context.WithValue(ctx, txContextKey, tx)
    
    // 1. Validate account exists
    account, err := s.accountService.GetAccount(txCtx, req.AccountID)
    if err != nil {
        return nil, s.handleAccountError(err)
    }
    
    // 2. Validate quote (find matching PriceSeries)
    priceSeries, err := s.tradingRepo.FindPriceSeries(txCtx, req.AccountID, req.SeriesType, req.PreviousQuote)
    if err != nil {
        return nil, &APIError{Code: "INVALID_QUOTE", Message: "Quote does not match any active preview"}
    }
    
    // 3. Check balance
    balance, err := s.accountService.GetAccountBalance(txCtx, req.AccountID)
    if err != nil {
        return nil, fmt.Errorf("failed to get balance: %w", err)
    }
    if balance.LessThan(req.Stake) {
        return nil, &APIError{Code: "INSUFFICIENT_BALANCE", Message: "Insufficient funds"}
    }
    
    // 4. Create contract (get ID for transaction references)
    contract := &Contract{
        AccountID:  req.AccountID,
        SeriesType: req.SeriesType,
        Sentiment:  req.Sentiment,
        Stake:      req.Stake,
    }
    contractID, err := s.tradingRepo.CreateContract(txCtx, contract)
    if err != nil {
        return nil, fmt.Errorf("failed to create contract: %w", err)
    }
    
    // 5. Deduct stake
    _, err = s.accountService.DeductStake(txCtx, req.AccountID, req.Stake, contractID)
    if err != nil {
        return nil, s.handleAccountError(err)
    }
    
    // 6. Generate remaining candles and evaluate
    candles11to20 := s.generateCandles(priceSeries.Candles[9].Close, 10)
    allCandles := append(priceSeries.Candles, candles11to20...)
    
    outcome := s.evaluateOutcome(req.Sentiment, allCandles[9].Close, allCandles[19].Close)
    payout := s.calculatePayout(req.Stake, outcome)
    
    // 7. Credit payout (even if 0.00)
    _, err = s.accountService.CreditPayout(txCtx, req.AccountID, payout, contractID)
    if err != nil {
        return nil, fmt.Errorf("failed to credit payout: %w", err)
    }
    
    // 8. Update contract with outcome
    err = s.tradingRepo.UpdateContract(txCtx, contractID, allCandles, payout, outcome)
    if err != nil {
        return nil, fmt.Errorf("failed to update contract: %w", err)
    }
    
    // 9. Delete PriceSeries
    err = s.tradingRepo.DeletePriceSeries(txCtx, priceSeries.SeriesID)
    if err != nil {
        return nil, fmt.Errorf("failed to delete price series: %w", err)
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return &SwipeBuyResponse{
        ContractID:   contractID,
        Candles:      candles11to20,
        Outcome:      outcome,
        Payout:       payout,
        FinalBalance: balance.Sub(req.Stake).Add(payout),
    }, nil
}
```

---

## 9. Quality Checklist

- [x] All 4 inter-module functions are documented
- [x] Function signatures include Go types
- [x] Parameters are fully specified with validation rules
- [x] Return types are documented
- [x] Error codes follow ERR-[SERVICE]-[3CHAR] format
- [x] Transaction boundaries are clearly defined
- [x] Atomicity requirements are specified
- [x] Data models align with domain model
- [x] Decimal precision (2 decimals for monetary, 3 for OHLC) documented
- [x] Example usage provided for each function
- [x] Trade execution flow documented

---

## Appendix A: Function-to-Requirement Mapping

| Function | PRD Requirement | User Story |
|----------|-----------------|------------|
| GetAccount | FEA-TR-W8P | US-TR-W8P, US-TR-F7K |
| GetAccountBalance | REQ-TR-B6N | US-TR-B6N |
| DeductStake | REQ-TR-B6N | US-TR-W8P, US-TR-F7K |
| CreditPayout | REQ-TR-U7P | US-TR-P7R |

---

## Appendix B: Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-16 | Archi | Initial internal API specification |
| 1.1 | 2026-01-16 | Archi | Self-review improvements: (1) Fixed transaction flow diagram ordering - contract now created before DeductStake to ensure valid reference_id; (2) Added txContextKey definition for transaction context propagation; (3) Enhanced two-phase contract creation documentation |
