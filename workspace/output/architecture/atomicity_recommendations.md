# Atomicity Recommendations for Options Trading Platform

## Executive Summary

This document provides architectural recommendations for ensuring atomicity across the `accounts`, `transactions`, and `contracts` tables in the options trading platform. After reviewing the existing codebase, the current implementation already demonstrates strong transactional patterns. This document outlines what's working well, identifies potential improvements, and provides additional safeguards.

**Two approaches are presented:**
1. **Application-level transactions** (current approach) - Transaction management in Go code
2. **Stored procedure approach** (recommended simplification) - Transaction logic encapsulated in PostgreSQL

---

## Current Implementation Assessment

### ✅ Already Implemented Correctly

#### 1. Database Transaction Wrapping

The [`ExecuteTrade()`](workspace/code/arcade/internal/trading/service.go:64) method properly wraps all operations in a single database transaction:

```go
tx, err := s.pool.Begin(ctx)
if err != nil {
    return nil, fmt.Errorf("failed to begin transaction: %w", err)
}
defer tx.Rollback(ctx)
// ... operations ...
if err := tx.Commit(ctx); err != nil {
    return nil, fmt.Errorf("failed to commit transaction: %w", err)
}
```

#### 2. Context-Based Transaction Propagation

The [`common.WithTx()`](workspace/code/arcade/internal/common/context.go:24) and [`common.GetTx()`](workspace/code/arcade/internal/common/context.go:18) pattern correctly propagates the transaction across service boundaries:

```go
txCtx := common.WithTx(ctx, tx)
// All subsequent operations use txCtx
```

#### 3. Pessimistic Locking with `SELECT FOR UPDATE`

The [`LockAccountForUpdate()`](workspace/code/arcade/internal/accounts/repository.go:125) method uses row-level locking to prevent race conditions:

```sql
SELECT balance FROM accounts WHERE account_id = $1 FOR UPDATE
```

#### 4. Two-Phase Contract Creation

The implementation uses a two-phase approach in [`CreateContractInitial()`](workspace/code/arcade/internal/trading/repository.go:128) and [`UpdateContractComplete()`](workspace/code/arcade/internal/trading/repository.go:173):
- **Phase 1**: Reserve contract ID before financial operations
- **Phase 2**: Finalize contract after trade execution

#### 5. Database-Level Constraints

The migrations include proper constraints in [`000001_create_accounts.up.sql`](workspace/code/arcade/migrations/000001_create_accounts.up.sql:12):

```sql
CONSTRAINT chk_balance_non_negative CHECK (balance >= 0)
```

---

## Recommended Improvements

### 1. Add Isolation Level Control

**Current Gap**: The transaction uses PostgreSQL's default isolation level (Read Committed), which may allow phantom reads in edge cases.

**Recommendation**: Explicitly set `SERIALIZABLE` or `REPEATABLE READ` isolation for critical financial operations:

```go
// In trading/service.go ExecuteTrade()
txOptions := pgx.TxOptions{
    IsoLevel:   pgx.Serializable,
    AccessMode: pgx.ReadWrite,
}
tx, err := s.pool.BeginTx(ctx, txOptions)
```

**Trade-off**: Higher isolation levels may increase contention and require retry logic for serialization failures.

### 2. Implement Serialization Failure Retry Logic

**Recommendation**: Add retry logic for `SQLSTATE 40001` (serialization failure):

```go
const maxRetries = 3

func (s *Service) ExecuteTradeWithRetry(ctx context.Context, req SwipeBuyRequest) (*SwipeBuyResponse, error) {
    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        resp, err := s.ExecuteTrade(ctx, req)
        if err == nil {
            return resp, nil
        }
        
        // Check for serialization failure
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "40001" {
            lastErr = err
            time.Sleep(time.Duration(attempt*10) * time.Millisecond)
            continue
        }
        return nil, err
    }
    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### 3. Add Foreign Key Constraint for Transaction-Contract Relationship

**Current Gap**: The `reference_id` in transactions table is not a formal foreign key to contracts.

**Recommendation**: Add a deferred foreign key constraint:

```sql
-- Add to migrations
ALTER TABLE transactions 
ADD CONSTRAINT fk_transactions_contract 
    FOREIGN KEY (reference_id) 
    REFERENCES contracts(contract_id) 
    DEFERRABLE INITIALLY DEFERRED;
```

Using `DEFERRABLE INITIALLY DEFERRED` allows the constraint to be checked at commit time, supporting the two-phase contract creation pattern.

### 4. Implement Idempotency for Trade Execution

**Current Gap**: Unlike `Deposit` and `Withdraw`, `ExecuteTrade` lacks idempotency protection.

**Recommendation**: Add an optional `trade_request_id` field for idempotency:

```go
type SwipeBuyRequest struct {
    AccountID     string  `json:"account_id"`
    SeriesType    string  `json:"series_type"`
    Sentiment     string  `json:"sentiment"`
    Stake         string  `json:"stake"`
    PreviousQuote string  `json:"previous_quote"`
    TradeRequestID *string `json:"trade_request_id,omitempty"` // Optional idempotency key
}
```

Store in contracts table or use price_series deletion as implicit idempotency guard.

### 5. Add Balance Audit Trail Validation

**Recommendation**: Add a database trigger or periodic validation to ensure balance consistency:

```sql
-- Create balance audit function
CREATE OR REPLACE FUNCTION validate_account_balance(p_account_id VARCHAR(20))
RETURNS BOOLEAN AS $$
DECLARE
    calc_balance DECIMAL(18,2);
    actual_balance DECIMAL(18,2);
BEGIN
    SELECT 
        COALESCE(SUM(CASE 
            WHEN type IN ('DEPOSIT', 'PAYOUT') THEN amount
            WHEN type IN ('WITHDRAWAL', 'STAKE') THEN -amount
        END), 0)
    INTO calc_balance
    FROM transactions
    WHERE account_id = p_account_id;
    
    SELECT balance INTO actual_balance
    FROM accounts WHERE account_id = p_account_id;
    
    RETURN calc_balance = actual_balance;
END;
$$ LANGUAGE plpgsql;
```

### 6. Add Transaction Status Field

**Recommendation**: For future rollback/reversal support, consider adding transaction status:

```sql
ALTER TABLE transactions ADD COLUMN status VARCHAR(20) DEFAULT 'COMPLETED'
    CHECK (status IN ('PENDING', 'COMPLETED', 'REVERSED'));
```

---

## Sequence Diagram: Atomic Contract Purchase

```
┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────────┐
│ Client  │          │TradingService│         │AccountService│          │  PostgreSQL  │
└────┬────┘          └──────┬──────┘          └──────┬───────┘          └──────┬───────┘
     │   SwipeBuy Request   │                        │                         │
     │─────────────────────>│                        │                         │
     │                      │                        │                         │
     │                      │  BEGIN TRANSACTION     │                         │
     │                      │─────────────────────────────────────────────────>│
     │                      │                        │                         │
     │                      │  1. SELECT ... FOR UPDATE (accounts)             │
     │                      │─────────────────────────────────────────────────>│
     │                      │                        │         balance         │
     │                      │<─────────────────────────────────────────────────│
     │                      │                        │                         │
     │                      │  2. Validate balance >= stake                    │
     │                      │  3. INSERT contracts (Phase 1)                   │
     │                      │─────────────────────────────────────────────────>│
     │                      │                        │       contract_id       │
     │                      │<─────────────────────────────────────────────────│
     │                      │                        │                         │
     │                      │  4. DeductStake(ctx)   │                         │
     │                      │───────────────────────>│                         │
     │                      │                        │  UPDATE accounts        │
     │                      │                        │  INSERT transactions    │
     │                      │                        │────────────────────────>│
     │                      │<───────────────────────│                         │
     │                      │                        │                         │
     │                      │  5. Generate candles 11-20, evaluate outcome     │
     │                      │                        │                         │
     │                      │  6. UPDATE contracts (Phase 2)                   │
     │                      │─────────────────────────────────────────────────>│
     │                      │                        │                         │
     │                      │  7. CreditPayout(ctx)  │                         │
     │                      │───────────────────────>│                         │
     │                      │                        │  UPDATE accounts        │
     │                      │                        │  INSERT transactions    │
     │                      │                        │────────────────────────>│
     │                      │<───────────────────────│                         │
     │                      │                        │                         │
     │                      │  8. DELETE price_series                          │
     │                      │─────────────────────────────────────────────────>│
     │                      │                        │                         │
     │                      │  COMMIT                │                         │
     │                      │─────────────────────────────────────────────────>│
     │   SwipeBuyResponse   │                        │                         │
     │<─────────────────────│                        │                         │
```

---

## Failure Scenarios & Recovery

| Failure Point | Current Behavior | Status |
|--------------|------------------|--------|
| Before `tx.Begin()` | No changes made | ✅ Safe |
| After balance lock, before contract insert | `defer tx.Rollback()` restores state | ✅ Safe |
| After contract insert, before stake deduction | `defer tx.Rollback()` removes partial contract | ✅ Safe |
| After stake deduction, before payout | `defer tx.Rollback()` restores balance + removes contract | ✅ Safe |
| After payout, before commit | `defer tx.Rollback()` restores all | ✅ Safe |
| During commit | PostgreSQL WAL ensures atomic commit | ✅ Safe |
| Network failure after commit | Client may not receive response; needs idempotency | ⚠️ Needs idempotency key |

---

## Concurrency Considerations

### Race Condition: Same Account Concurrent Trades

**Scenario**: Two concurrent trade requests for the same account.

**Current Protection**: `SELECT ... FOR UPDATE` serializes access at row level.

**Behavior**:
1. Request A acquires lock on account row
2. Request B waits for lock
3. Request A commits/rollbacks
4. Request B proceeds with updated balance

### Race Condition: Quote Staleness

**Scenario**: Client submits trade with stale `previous_quote`.

**Current Protection**: Price series lookup validates quote; invalid quote returns error.

---

## Recommended Transaction Boundary Pattern

For maximum safety, follow this pattern for any operation involving multiple tables:

```go
func (s *Service) AtomicOperation(ctx context.Context, ...) error {
    // 1. Validate inputs BEFORE starting transaction
    if err := validate(input); err != nil {
        return err
    }
    
    // 2. Start transaction with explicit isolation
    txOpts := pgx.TxOptions{IsoLevel: pgx.RepeatableRead}
    tx, err := s.pool.BeginTx(ctx, txOpts)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx)
    
    txCtx := common.WithTx(ctx, tx)
    
    // 3. Lock resources in consistent order (prevents deadlocks)
    _, err = s.repo.LockAccountForUpdate(txCtx, accountID)
    if err != nil {
        return err
    }
    
    // 4. Perform all mutations
    // ... insert/update operations ...
    
    // 5. Commit
    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("commit: %w", err)
    }
    
    return nil
}
```

---

## Alternative: Stored Procedure Approach (Recommended)

A stored procedure can significantly simplify the application code by encapsulating all atomicity logic within the database. This approach has several advantages for financial operations.

### Stored Procedure: `execute_trade`

```sql
-- Migration: 000005_create_execute_trade_procedure.up.sql

CREATE OR REPLACE FUNCTION execute_trade(
    p_account_id VARCHAR(20),
    p_series_type VARCHAR(10),
    p_sentiment VARCHAR(10),
    p_stake DECIMAL(18,2),
    p_initial_candles JSONB,      -- Candles 1-10 from price preview
    p_execution_candles JSONB,    -- Candles 11-20 generated by app
    p_payout DECIMAL(18,2),       -- Pre-calculated payout (0 for loss)
    p_price_series_id BIGINT      -- ID of price_series to delete
)
RETURNS TABLE (
    contract_id BIGINT,
    stake_transaction_id BIGINT,
    payout_transaction_id BIGINT,
    new_balance DECIMAL(18,2),
    purchase_time TIMESTAMP WITH TIME ZONE
) AS $$
DECLARE
    v_current_balance DECIMAL(18,2);
    v_contract_id BIGINT;
    v_stake_txn_id BIGINT;
    v_payout_txn_id BIGINT;
    v_new_balance DECIMAL(18,2);
    v_purchase_time TIMESTAMP WITH TIME ZONE := NOW();
    v_all_candles JSONB;
BEGIN
    -- 1. Lock account row and get current balance (prevents concurrent modifications)
    SELECT balance INTO v_current_balance
    FROM accounts
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_account_id
            USING ERRCODE = 'P0001';
    END IF;
    
    -- 2. Validate sufficient balance
    IF v_current_balance < p_stake THEN
        RAISE EXCEPTION 'Insufficient balance. Required: %, Available: %', p_stake, v_current_balance
            USING ERRCODE = 'P0002';
    END IF;
    
    -- 3. Combine all candles (1-20)
    v_all_candles := p_initial_candles || p_execution_candles;
    
    -- 4. Insert contract record
    INSERT INTO contracts (account_id, series_type, sentiment, stake, payout, ohlcs, purchase_time)
    VALUES (p_account_id, p_series_type, p_sentiment, p_stake, p_payout, v_all_candles, v_purchase_time)
    RETURNING contracts.contract_id INTO v_contract_id;
    
    -- 5. Deduct stake from balance
    v_new_balance := v_current_balance - p_stake;
    
    -- 6. Record STAKE transaction
    INSERT INTO transactions (account_id, type, amount, reference_id, transaction_time)
    VALUES (p_account_id, 'STAKE', p_stake, v_contract_id, v_purchase_time)
    RETURNING transaction_id INTO v_stake_txn_id;
    
    -- 7. Credit payout (even if 0)
    v_new_balance := v_new_balance + p_payout;
    
    -- 8. Record PAYOUT transaction
    INSERT INTO transactions (account_id, type, amount, reference_id, transaction_time)
    VALUES (p_account_id, 'PAYOUT', p_payout, v_contract_id, v_purchase_time)
    RETURNING transaction_id INTO v_payout_txn_id;
    
    -- 9. Update account balance (single update with final value)
    UPDATE accounts
    SET balance = v_new_balance, updated_at = v_purchase_time
    WHERE account_id = p_account_id;
    
    -- 10. Delete used price series
    DELETE FROM price_series WHERE series_id = p_price_series_id;
    
    -- Return results
    RETURN QUERY SELECT v_contract_id, v_stake_txn_id, v_payout_txn_id, v_new_balance, v_purchase_time;
END;
$$ LANGUAGE plpgsql;

-- Custom error codes for application error handling
COMMENT ON FUNCTION execute_trade IS 'Atomic trade execution procedure.
Error codes:
  P0001 - Account not found
  P0002 - Insufficient balance';
```

### Stored Procedure: `deposit_funds`

```sql
CREATE OR REPLACE FUNCTION deposit_funds(
    p_account_id VARCHAR(20),
    p_amount DECIMAL(18,2),
    p_idempotency_id UUID
)
RETURNS TABLE (
    transaction_id BIGINT,
    new_balance DECIMAL(18,2),
    is_duplicate BOOLEAN
) AS $$
DECLARE
    v_existing_txn_id BIGINT;
    v_current_balance DECIMAL(18,2);
    v_new_balance DECIMAL(18,2);
    v_txn_id BIGINT;
BEGIN
    -- Check idempotency first (outside lock for performance)
    SELECT t.transaction_id INTO v_existing_txn_id
    FROM transactions t
    WHERE t.account_id = p_account_id AND t.idempotency_id = p_idempotency_id;
    
    IF FOUND THEN
        -- Return existing transaction (idempotent response)
        SELECT a.balance INTO v_current_balance FROM accounts a WHERE a.account_id = p_account_id;
        RETURN QUERY SELECT v_existing_txn_id, v_current_balance, TRUE;
        RETURN;
    END IF;
    
    -- Lock account
    SELECT balance INTO v_current_balance
    FROM accounts
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_account_id USING ERRCODE = 'P0001';
    END IF;
    
    -- Calculate and update balance
    v_new_balance := v_current_balance + p_amount;
    
    UPDATE accounts SET balance = v_new_balance, updated_at = NOW()
    WHERE account_id = p_account_id;
    
    -- Create transaction record
    INSERT INTO transactions (account_id, type, amount, idempotency_id, transaction_time)
    VALUES (p_account_id, 'DEPOSIT', p_amount, p_idempotency_id, NOW())
    RETURNING transactions.transaction_id INTO v_txn_id;
    
    RETURN QUERY SELECT v_txn_id, v_new_balance, FALSE;
END;
$$ LANGUAGE plpgsql;
```

### Stored Procedure: `withdraw_funds`

```sql
CREATE OR REPLACE FUNCTION withdraw_funds(
    p_account_id VARCHAR(20),
    p_amount DECIMAL(18,2),
    p_idempotency_id UUID
)
RETURNS TABLE (
    transaction_id BIGINT,
    new_balance DECIMAL(18,2),
    is_duplicate BOOLEAN
) AS $$
DECLARE
    v_existing_txn_id BIGINT;
    v_current_balance DECIMAL(18,2);
    v_new_balance DECIMAL(18,2);
    v_txn_id BIGINT;
BEGIN
    -- Check idempotency first
    SELECT t.transaction_id INTO v_existing_txn_id
    FROM transactions t
    WHERE t.account_id = p_account_id AND t.idempotency_id = p_idempotency_id;
    
    IF FOUND THEN
        SELECT a.balance INTO v_current_balance FROM accounts a WHERE a.account_id = p_account_id;
        RETURN QUERY SELECT v_existing_txn_id, v_current_balance, TRUE;
        RETURN;
    END IF;
    
    -- Lock account
    SELECT balance INTO v_current_balance
    FROM accounts
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_account_id USING ERRCODE = 'P0001';
    END IF;
    
    -- Validate balance
    IF v_current_balance < p_amount THEN
        RAISE EXCEPTION 'Insufficient balance' USING ERRCODE = 'P0002';
    END IF;
    
    -- Calculate and update balance
    v_new_balance := v_current_balance - p_amount;
    
    UPDATE accounts SET balance = v_new_balance, updated_at = NOW()
    WHERE account_id = p_account_id;
    
    -- Create transaction record
    INSERT INTO transactions (account_id, type, amount, idempotency_id, transaction_time)
    VALUES (p_account_id, 'WITHDRAWAL', p_amount, p_idempotency_id, NOW())
    RETURNING transactions.transaction_id INTO v_txn_id;
    
    RETURN QUERY SELECT v_txn_id, v_new_balance, FALSE;
END;
$$ LANGUAGE plpgsql;
```

### Simplified Go Application Code

With stored procedures, the application code becomes dramatically simpler:

```go
// trading/service.go - Simplified ExecuteTrade
func (s *Service) ExecuteTrade(ctx context.Context, req SwipeBuyRequest) (*SwipeBuyResponse, error) {
    // 1. Validate inputs (unchanged)
    if err := common.ValidateSeriesType(req.SeriesType); err != nil {
        return nil, err
    }
    if err := common.ValidateSentiment(req.Sentiment); err != nil {
        return nil, err
    }
    stake, err := common.ParseAmount(req.Stake)
    if err != nil {
        return nil, common.NewAPIError(common.ErrCodeInvalidStake, "Invalid stake")
    }

    // 2. Find price series (read-only, no transaction needed)
    priceSeries, err := s.repo.FindPriceSeries(ctx, req.AccountID, req.SeriesType, req.PreviousQuote)
    if err != nil {
        return nil, common.NewAPIError(common.ErrCodeInvalidQuote, "Quote invalid")
    }

    // 3. Generate execution candles (business logic in app)
    config := GetSeriesConfig(req.SeriesType)
    lastCandle := priceSeries.Candles[9]
    startTime := lastCandle.Timestamp.Add(config.Interval)
    executionCandles := s.gbmGenerator.GenerateCandles(lastCandle.Close, config, 10, startTime)

    // 4. Evaluate outcome & calculate payout
    isWin := s.evaluateOutcome(req.Sentiment, lastCandle.Close, executionCandles[9].Close)
    var payout decimal.Decimal
    if isWin {
        payout = stake.Div(decimal.NewFromFloat(0.53)).Round(2)
    }

    // 5. Marshal candles to JSON
    initialCandlesJSON, _ := json.Marshal(priceSeries.Candles)
    executionCandlesJSON, _ := json.Marshal(executionCandles)

    // 6. Call stored procedure (ALL atomicity handled by DB)
    var result struct {
        ContractID   int64
        StakeTxnID   int64
        PayoutTxnID  int64
        NewBalance   decimal.Decimal
        PurchaseTime time.Time
    }
    
    err = s.pool.QueryRow(ctx, `
        SELECT * FROM execute_trade($1, $2, $3, $4, $5, $6, $7, $8)
    `, req.AccountID, req.SeriesType, req.Sentiment, stake,
       initialCandlesJSON, executionCandlesJSON, payout, priceSeries.SeriesID,
    ).Scan(&result.ContractID, &result.StakeTxnID, &result.PayoutTxnID,
           &result.NewBalance, &result.PurchaseTime)
    
    if err != nil {
        return nil, s.mapPgError(err)
    }

    return &SwipeBuyResponse{
        ContractID:   result.ContractID,
        PurchaseTime: result.PurchaseTime,
        OHLCs:        executionCandles,
        Payout:       common.FormatAmount(payout),
    }, nil
}

// mapPgError converts PostgreSQL error codes to API errors
func (s *Service) mapPgError(err error) error {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "P0001":
            return common.ErrAccountNotFound
        case "P0002":
            return common.ErrInsufficientBalance
        }
    }
    return fmt.Errorf("database error: %w", err)
}
```

---

## Comparison: Application vs Stored Procedure Approach

| Aspect | Application Transactions | Stored Procedures |
|--------|------------------------|-------------------|
| **Code Complexity** | Complex - manage tx lifecycle, context propagation | Simple - single function call |
| **Network Round-trips** | Multiple (BEGIN, queries, COMMIT) | Single (procedure call) |
| **Atomicity Guarantee** | Requires careful coding | Automatic - procedure is atomic |
| **Error Handling** | Manual rollback on each error path | Automatic rollback on RAISE |
| **Testing** | Can mock repositories | Requires database for testing |
| **Debugging** | Easy - step through Go code | Harder - need PL/pgSQL debugging |
| **Portability** | Database-agnostic | PostgreSQL-specific |
| **Business Logic Location** | Application layer | Split: validation in app, financial ops in DB |
| **Deployment** | Code deploy only | Code + migration deploy |
| **Version Control** | Standard Git workflow | Migrations for procedure changes |

---

## Recommendation

**For an options trading platform handling real money, the stored procedure approach is recommended because:**

1. **Simpler correctness** - Atomicity is guaranteed by the database engine, not developer discipline
2. **Performance** - Fewer network round-trips, especially important for high-frequency trading
3. **Consistency** - All financial operations go through the same validated procedure
4. **Audit trail** - Procedure changes are versioned through migrations
5. **Defense in depth** - Even if application code has bugs, the procedure enforces invariants

**Keep in application layer:**
- Input validation (series type, sentiment, stake format)
- Price generation (GBM algorithm)
- Outcome evaluation (win/loss determination)

**Move to stored procedures:**
- Balance checks
- Contract creation
- Transaction recording
- Balance updates

---

## Summary

The current implementation in [`trading/service.go`](workspace/code/arcade/internal/trading/service.go) demonstrates **correct transactional patterns** for ensuring atomicity across the three tables. Two approaches are viable:

### Option 1: Keep Current Approach + Improvements

| Priority | Recommendation | Impact |
|----------|---------------|--------|
| High | Add serialization failure retry logic | Handles edge cases under high concurrency |
| Medium | Add idempotency for trade execution | Protects against network failures |
| Medium | Use explicit isolation level | Makes guarantees explicit |
| Low | Add foreign key for reference_id | Enforces referential integrity |
| Low | Add balance audit trigger | Provides additional safety net |

### Option 2: Migrate to Stored Procedures (Recommended)

| Priority | Action |
|----------|--------|
| High | Create `execute_trade` stored procedure |
| High | Create `deposit_funds` and `withdraw_funds` procedures |
| Medium | Simplify Go service layer to call procedures |
| Low | Add procedure for balance audit |

The stored procedure approach provides **stronger guarantees with simpler code** and is the recommended path for production financial systems.
