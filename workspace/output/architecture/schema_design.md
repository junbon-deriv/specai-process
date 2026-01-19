# Database Schema Design - Options Trading Platform

## Overview

This document defines the database schema for an options trading platform with atomic operations across accounts, transactions, and contracts tables.

---

## Schema Diagram

```
┌─────────────────────┐       ┌─────────────────────┐
│    series_types     │       │      accounts       │
├─────────────────────┤       ├─────────────────────┤
│ series_type (PK)    │       │ account_id (PK)     │
│ config (JSONB)      │       │ external_id         │
│ created_at          │       │ currency            │
│                     │       │ balance >= 0        │
└─────────────────────┘       │ created_at          │
         │                    │ updated_at          │
         │                    └─────────────────────┘
         │                              │
         ▼                              │
┌─────────────────────┐                 │
│    price_series     │                 │
├─────────────────────┤                 │
│ series_id (PK)      │                 │
│ account_id (FK)─────┼─────────────────┘
│ series_type (FK)────┘
│ candles (JSONB)     │
│ quote_value         │
│ created_at          │
│ expires_at          │
└─────────────────────┘
         │
         │ (consumed on open_trade)
         ▼
┌─────────────────────┐       ┌─────────────────────┐
│     contracts       │       │    transactions     │
├─────────────────────┤       ├─────────────────────┤
│ contract_id (PK)    │◄──────┤ contract_id (FK)    │
│ account_id (FK)     │       │ transaction_id (PK) │
│ series_type (FK)    │       │ account_id (FK)     │
│ sentiment           │       │ type (DEPOSIT/      │
│ buy_price (stake)   │       │   WITHDRAWAL/BUY/   │
│ buy_time            │       │   SELL)             │
│ buy_ohlcs (JSONB)   │       │ amount (+/-)        │
│ sell_price (payout) │       │ idempotency_id      │
│ sell_time (NULL=open)       │ transaction_time    │
│ sell_ohlcs (JSONB)  │       └─────────────────────┘
└─────────────────────┘
```

**Contract Status Logic**: A contract is OPEN if `sell_time IS NULL`, SETTLED if `sell_time IS NOT NULL`.

**Price Semantics**:
- `buy_price` = stake (amount deducted from balance on BUY)
- `sell_price` = payout (amount credited to balance on SELL, 0 for loss)

---

## Table Definitions

### 1. `accounts` - Client Balance Storage

```sql
-- Migration: 000001_create_accounts.up.sql

CREATE SEQUENCE account_id_seq START 1;

CREATE TABLE accounts (
    account_id VARCHAR(20) PRIMARY KEY,
    external_id VARCHAR(255),
    currency CHAR(3) NOT NULL,
    balance DECIMAL(18,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Balance can never go negative
    CONSTRAINT chk_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT chk_currency_format CHECK (currency ~ '^[A-Z]{3}$')
);

CREATE INDEX idx_accounts_external_id ON accounts(external_id) WHERE external_id IS NOT NULL;

-- Function to generate SW-prefixed account IDs
CREATE OR REPLACE FUNCTION generate_account_id()
RETURNS VARCHAR(20) AS $$
BEGIN
    RETURN 'SW' || nextval('account_id_seq')::text;
END;
$$ LANGUAGE plpgsql;
```

### 2. `series_types` - Trading Series Configuration

```sql
-- Migration: 000002_create_series_types.up.sql

CREATE TABLE series_types (
    series_type VARCHAR(10) PRIMARY KEY,
    config JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Validate required config fields exist
    CONSTRAINT chk_config_required_fields CHECK (
        config ? 'initial_value' AND
        config ? 'volatility' AND
        config ? 'drift' AND
        config ? 'interval_seconds' AND
        config ? 'display_name'
    )
);

-- Seed default series types
INSERT INTO series_types (series_type, config) VALUES
    ('Vol50', '{
        "display_name": "Volatility 50",
        "initial_value": 1000.00,
        "volatility": 0.50,
        "drift": 0.0,
        "interval_seconds": 1,
        "payout_multiplier": 1.8868
    }'::jsonb),
    ('Vol100', '{
        "display_name": "Volatility 100",
        "initial_value": 1000.00,
        "volatility": 1.00,
        "drift": 0.0,
        "interval_seconds": 1,
        "payout_multiplier": 1.8868
    }'::jsonb),
    ('Vol200', '{
        "display_name": "Volatility 200",
        "initial_value": 1000.00,
        "volatility": 2.00,
        "drift": 0.0,
        "interval_seconds": 1,
        "payout_multiplier": 1.8868
    }'::jsonb),
    ('Vol300', '{
        "display_name": "Volatility 300",
        "initial_value": 1000.00,
        "volatility": 3.00,
        "drift": 0.0,
        "interval_seconds": 1,
        "payout_multiplier": 1.8868
    }'::jsonb);

CREATE INDEX idx_series_types_active ON series_types(is_active) WHERE is_active = TRUE;
```

### 3. `transactions` - Financial Movement Ledger

```sql
-- Migration: 000003_create_transactions.up.sql

CREATE TABLE transactions (
    transaction_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(20) NOT NULL REFERENCES accounts(account_id),
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(18,2) NOT NULL,  -- Positive for DEPOSIT/SELL, Negative for WITHDRAWAL/BUY
    contract_id BIGINT,              -- FK to contracts (nullable for DEPOSIT/WITHDRAWAL)
    idempotency_id UUID,             -- For DEPOSIT/WITHDRAWAL deduplication
    transaction_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_type_valid CHECK (type IN ('DEPOSIT', 'WITHDRAWAL', 'BUY', 'SELL')),
    
    -- Amount sign validation based on type
    CONSTRAINT chk_amount_sign CHECK (
        (type = 'DEPOSIT' AND amount > 0) OR
        (type = 'WITHDRAWAL' AND amount < 0) OR
        (type = 'BUY' AND amount < 0) OR
        (type = 'SELL' AND amount >= 0)
    ),
    
    -- Contract ID required for BUY/SELL transactions
    CONSTRAINT chk_contract_required CHECK (
        (type IN ('DEPOSIT', 'WITHDRAWAL') AND contract_id IS NULL) OR
        (type IN ('BUY', 'SELL') AND contract_id IS NOT NULL)
    ),
    
    -- Idempotency ID required for DEPOSIT/WITHDRAWAL
    CONSTRAINT chk_idempotency_required CHECK (
        (type IN ('BUY', 'SELL') AND idempotency_id IS NULL) OR
        (type IN ('DEPOSIT', 'WITHDRAWAL') AND idempotency_id IS NOT NULL)
    )
);

-- Foreign key to contracts (added after contracts table is created)
-- See migration 000005

-- Unique index for idempotency (only for DEPOSIT/WITHDRAWAL)
CREATE UNIQUE INDEX idx_transactions_idempotency 
    ON transactions(account_id, idempotency_id) 
    WHERE idempotency_id IS NOT NULL;

-- Index for account transactions lookup
CREATE INDEX idx_transactions_account_time ON transactions(account_id, transaction_time DESC);

-- Index for contract transactions lookup
CREATE INDEX idx_transactions_contract ON transactions(contract_id) WHERE contract_id IS NOT NULL;
```

### 4. `price_series` - Quote Preview Storage

```sql
-- Migration: 000004_create_price_series.up.sql

CREATE TABLE price_series (
    series_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(20) NOT NULL REFERENCES accounts(account_id),
    series_type VARCHAR(10) NOT NULL REFERENCES series_types(series_type),
    candles JSONB NOT NULL,
    quote_value VARCHAR(30) NOT NULL,  -- 10th candle close as string for matching
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for series lookup by ID (primary use case)
CREATE INDEX idx_price_series_id ON price_series(series_id);

-- Index for account's active series
CREATE INDEX idx_price_series_account ON price_series(account_id, created_at DESC);
```

**Note on price_series querying**: Querying by `series_id` alone is sufficient because:
1. The `series_id` is returned to the client in the quote response
2. Client submits `series_id` with the trade request
3. The stored procedure validates that the series belongs to the requesting account
4. No need for composite keys - single column PK provides optimal performance

**Note**: Price series do not expire. They are deleted when consumed by `open_trade`.

### 5. `contracts` - Trading Contracts

```sql
-- Migration: 000005_create_contracts.up.sql

CREATE TABLE contracts (
    contract_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(20) NOT NULL REFERENCES accounts(account_id),
    series_type VARCHAR(10) NOT NULL REFERENCES series_types(series_type),
    sentiment VARCHAR(10) NOT NULL,
    
    -- Buy details (populated at open_trade)
    buy_price DECIMAL(18,6) NOT NULL,           -- Entry price (stake amount)
    buy_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    buy_ohlcs JSONB NOT NULL,                   -- Candles 1-10 from preview
    
    -- Sell details (NULL = contract is OPEN, populated at close_trade)
    sell_price DECIMAL(18,6),                   -- Exit price (payout amount)
    sell_time TIMESTAMP WITH TIME ZONE,         -- Settlement timestamp
    sell_ohlcs JSONB,                           -- Candles 11-20 from execution
    
    CONSTRAINT chk_sentiment_valid CHECK (sentiment IN ('rise', 'fall')),
    CONSTRAINT chk_buy_price_positive CHECK (buy_price > 0),
    CONSTRAINT chk_sell_price_non_negative CHECK (sell_price IS NULL OR sell_price >= 0),
    
    -- Contract is OPEN if sell_time IS NULL, SETTLED if sell_time IS NOT NULL
    CONSTRAINT chk_settlement_consistency CHECK (
        (sell_time IS NULL AND sell_price IS NULL AND sell_ohlcs IS NULL) OR
        (sell_time IS NOT NULL AND sell_price IS NOT NULL AND sell_ohlcs IS NOT NULL)
    )
);

-- Add foreign key from transactions to contracts
ALTER TABLE transactions
ADD CONSTRAINT fk_transactions_contract
    FOREIGN KEY (contract_id)
    REFERENCES contracts(contract_id)
    DEFERRABLE INITIALLY DEFERRED;

-- Index for account trading history (most recent first)
CREATE INDEX idx_contracts_account_time ON contracts(account_id, buy_time DESC);

-- Index for open contracts (sell_time IS NULL means OPEN)
CREATE INDEX idx_contracts_open ON contracts(account_id) WHERE sell_time IS NULL;

-- Index for filtered trading history by series type
CREATE INDEX idx_contracts_account_series ON contracts(account_id, series_type, buy_time DESC);
```

**Contract Status**: No explicit `status` column. A contract is:
- **OPEN**: `sell_time IS NULL`
- **SETTLED**: `sell_time IS NOT NULL`

**Price Semantics**:
- `buy_price` = stake amount (what client paid to open the contract)
- `sell_price` = payout amount (what client received on settlement, 0 for loss)

---

## Stored Procedures

### 1. `deposit_funds` - Atomic Deposit with Idempotency

```sql
-- Migration: 000010_create_deposit_funds.up.sql

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
    -- Validate amount is positive
    IF p_amount <= 0 THEN
        RAISE EXCEPTION 'Deposit amount must be positive: %', p_amount
            USING ERRCODE = 'P0003';
    END IF;

    -- Check idempotency first (read-only, no lock needed)
    SELECT t.transaction_id INTO v_existing_txn_id
    FROM transactions t
    WHERE t.account_id = p_account_id 
      AND t.idempotency_id = p_idempotency_id
      AND t.type = 'DEPOSIT';
    
    IF FOUND THEN
        -- Return existing transaction (idempotent response)
        SELECT a.balance INTO v_current_balance 
        FROM accounts a 
        WHERE a.account_id = p_account_id;
        
        RETURN QUERY SELECT v_existing_txn_id, v_current_balance, TRUE;
        RETURN;
    END IF;
    
    -- Lock account row
    SELECT balance INTO v_current_balance
    FROM accounts
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_account_id 
            USING ERRCODE = 'P0001';
    END IF;
    
    -- Calculate new balance
    v_new_balance := v_current_balance + p_amount;
    
    -- Update account balance
    UPDATE accounts 
    SET balance = v_new_balance, updated_at = NOW()
    WHERE account_id = p_account_id;
    
    -- Create transaction record (positive amount for DEPOSIT)
    INSERT INTO transactions (account_id, type, amount, idempotency_id, transaction_time)
    VALUES (p_account_id, 'DEPOSIT', p_amount, p_idempotency_id, NOW())
    RETURNING transactions.transaction_id INTO v_txn_id;
    
    RETURN QUERY SELECT v_txn_id, v_new_balance, FALSE;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION deposit_funds IS 'Atomic deposit with idempotency.
Parameters:
  p_account_id - Target account
  p_amount - Positive deposit amount
  p_idempotency_id - UUID for deduplication
  
Error codes:
  P0001 - Account not found
  P0003 - Invalid amount';
```

### 2. `withdraw_funds` - Atomic Withdrawal with Balance Check

```sql
-- Migration: 000011_create_withdraw_funds.up.sql

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
    -- Validate amount is positive (will be stored as negative)
    IF p_amount <= 0 THEN
        RAISE EXCEPTION 'Withdrawal amount must be positive: %', p_amount
            USING ERRCODE = 'P0003';
    END IF;

    -- Check idempotency first
    SELECT t.transaction_id INTO v_existing_txn_id
    FROM transactions t
    WHERE t.account_id = p_account_id 
      AND t.idempotency_id = p_idempotency_id
      AND t.type = 'WITHDRAWAL';
    
    IF FOUND THEN
        SELECT a.balance INTO v_current_balance 
        FROM accounts a 
        WHERE a.account_id = p_account_id;
        
        RETURN QUERY SELECT v_existing_txn_id, v_current_balance, TRUE;
        RETURN;
    END IF;
    
    -- Lock account row
    SELECT balance INTO v_current_balance
    FROM accounts
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_account_id 
            USING ERRCODE = 'P0001';
    END IF;
    
    -- Validate sufficient balance
    IF v_current_balance < p_amount THEN
        RAISE EXCEPTION 'Insufficient balance. Required: %, Available: %', p_amount, v_current_balance
            USING ERRCODE = 'P0002';
    END IF;
    
    -- Calculate new balance
    v_new_balance := v_current_balance - p_amount;
    
    -- Update account balance
    UPDATE accounts 
    SET balance = v_new_balance, updated_at = NOW()
    WHERE account_id = p_account_id;
    
    -- Create transaction record (NEGATIVE amount for WITHDRAWAL)
    INSERT INTO transactions (account_id, type, amount, idempotency_id, transaction_time)
    VALUES (p_account_id, 'WITHDRAWAL', -p_amount, p_idempotency_id, NOW())
    RETURNING transactions.transaction_id INTO v_txn_id;
    
    RETURN QUERY SELECT v_txn_id, v_new_balance, FALSE;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION withdraw_funds IS 'Atomic withdrawal with balance validation.
Parameters:
  p_account_id - Source account
  p_amount - Positive withdrawal amount (stored as negative)
  p_idempotency_id - UUID for deduplication
  
Error codes:
  P0001 - Account not found
  P0002 - Insufficient balance
  P0003 - Invalid amount';
```

### 3. `open_trade` - Buy Contract (Atomic)

```sql
-- Migration: 000012_create_open_trade.up.sql

CREATE OR REPLACE FUNCTION open_trade(
    p_account_id VARCHAR(20),
    p_series_id BIGINT,             -- Price series ID from quote
    p_sentiment VARCHAR(10),        -- 'rise' or 'fall'
    p_buy_price DECIMAL(18,6)       -- Stake amount (entry price)
)
RETURNS TABLE (
    contract_id BIGINT,
    buy_transaction_id BIGINT,
    new_balance DECIMAL(18,2),
    buy_ohlcs JSONB,
    buy_time TIMESTAMP WITH TIME ZONE
) AS $$
DECLARE
    v_current_balance DECIMAL(18,2);
    v_series_account VARCHAR(20);
    v_series_type VARCHAR(10);
    v_series_candles JSONB;
    v_contract_id BIGINT;
    v_txn_id BIGINT;
    v_new_balance DECIMAL(18,2);
    v_buy_time TIMESTAMP WITH TIME ZONE := NOW();
BEGIN
    -- Validate sentiment
    IF p_sentiment NOT IN ('rise', 'fall') THEN
        RAISE EXCEPTION 'Invalid sentiment: %. Must be rise or fall', p_sentiment
            USING ERRCODE = 'P0004';
    END IF;
    
    -- Validate buy price (stake)
    IF p_buy_price <= 0 THEN
        RAISE EXCEPTION 'Buy price must be positive: %', p_buy_price
            USING ERRCODE = 'P0003';
    END IF;

    -- Lock account and get balance
    SELECT balance INTO v_current_balance
    FROM accounts
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_account_id
            USING ERRCODE = 'P0001';
    END IF;
    
    -- Validate sufficient balance
    IF v_current_balance < p_buy_price THEN
        RAISE EXCEPTION 'Insufficient balance. Required: %, Available: %', p_buy_price, v_current_balance
            USING ERRCODE = 'P0002';
    END IF;
    
    -- Get and validate price series (no expiry check)
    SELECT ps.account_id, ps.series_type, ps.candles
    INTO v_series_account, v_series_type, v_series_candles
    FROM price_series ps
    WHERE ps.series_id = p_series_id
    FOR UPDATE;  -- Lock to prevent double-use
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Price series not found: %', p_series_id
            USING ERRCODE = 'P0005';
    END IF;
    
    -- Verify series belongs to this account
    IF v_series_account != p_account_id THEN
        RAISE EXCEPTION 'Price series does not belong to account'
            USING ERRCODE = 'P0006';
    END IF;
    
    -- Validate series type exists and is active
    IF NOT EXISTS (SELECT 1 FROM series_types WHERE series_type = v_series_type AND is_active = TRUE) THEN
        RAISE EXCEPTION 'Series type not active: %', v_series_type
            USING ERRCODE = 'P0007';
    END IF;
    
    -- Create contract with buy details (sell fields remain NULL = OPEN)
    INSERT INTO contracts (
        account_id, series_type, sentiment,
        buy_price, buy_time, buy_ohlcs
    )
    VALUES (
        p_account_id, v_series_type, p_sentiment,
        p_buy_price, v_buy_time, v_series_candles
    )
    RETURNING contracts.contract_id INTO v_contract_id;
    
    -- Deduct buy_price (stake) from balance
    v_new_balance := v_current_balance - p_buy_price;
    
    UPDATE accounts
    SET balance = v_new_balance, updated_at = v_buy_time
    WHERE account_id = p_account_id;
    
    -- Create BUY transaction (NEGATIVE amount)
    INSERT INTO transactions (account_id, type, amount, contract_id, transaction_time)
    VALUES (p_account_id, 'BUY', -p_buy_price, v_contract_id, v_buy_time)
    RETURNING transaction_id INTO v_txn_id;
    
    -- Delete used price series (one-time use)
    DELETE FROM price_series WHERE series_id = p_series_id;
    
    -- Return results
    RETURN QUERY SELECT v_contract_id, v_txn_id, v_new_balance, v_series_candles, v_buy_time;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION open_trade IS 'Atomic contract purchase (BUY).
Parameters:
  p_account_id - Buyer account
  p_series_id - Price series ID from quote
  p_sentiment - Trade direction (rise/fall)
  p_buy_price - Stake amount (deducted from balance)
  
Returns:
  contract_id - New contract ID
  buy_transaction_id - BUY transaction ID
  new_balance - Updated account balance
  buy_ohlcs - Candles 1-10
  buy_time - Contract purchase timestamp
  
Error codes:
  P0001 - Account not found
  P0002 - Insufficient balance
  P0003 - Invalid amount
  P0004 - Invalid sentiment
  P0005 - Price series not found
  P0006 - Price series account mismatch
  P0007 - Series type not active';
```

### 4. `close_trade` - Settle Contract (Atomic)

The `close_trade` procedure receives the sell_price (payout) from the application layer. The application is responsible for:
1. Generating execution candles (11-20)
2. Determining win/loss outcome
3. Calculating sell_price (payout) based on series configuration

This keeps business logic (outcome determination, payout calculation) in the application layer while the database handles atomicity.

```sql
-- Migration: 000013_create_close_trade.up.sql

CREATE OR REPLACE FUNCTION close_trade(
    p_account_id VARCHAR(20),
    p_contract_id BIGINT,
    p_sell_price DECIMAL(18,6),     -- Payout amount (0 for loss)
    p_sell_ohlcs JSONB              -- Candles 11-20 generated by application
)
RETURNS TABLE (
    sell_transaction_id BIGINT,
    new_balance DECIMAL(18,2),
    sell_time TIMESTAMP WITH TIME ZONE
) AS $$
DECLARE
    v_contract RECORD;
    v_current_balance DECIMAL(18,2);
    v_new_balance DECIMAL(18,2);
    v_txn_id BIGINT;
    v_sell_time TIMESTAMP WITH TIME ZONE := NOW();
BEGIN
    -- Validate sell_price (payout) is non-negative
    IF p_sell_price < 0 THEN
        RAISE EXCEPTION 'Sell price cannot be negative: %', p_sell_price
            USING ERRCODE = 'P0003';
    END IF;

    -- Lock account
    SELECT balance INTO v_current_balance
    FROM accounts
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_account_id
            USING ERRCODE = 'P0001';
    END IF;
    
    -- Get and lock contract
    SELECT c.contract_id, c.account_id, c.sell_time
    INTO v_contract
    FROM contracts c
    WHERE c.contract_id = p_contract_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Contract not found: %', p_contract_id
            USING ERRCODE = 'P0008';
    END IF;
    
    -- Verify contract belongs to account
    IF v_contract.account_id != p_account_id THEN
        RAISE EXCEPTION 'Contract does not belong to account'
            USING ERRCODE = 'P0009';
    END IF;
    
    -- Verify contract is OPEN (sell_time IS NULL)
    IF v_contract.sell_time IS NOT NULL THEN
        RAISE EXCEPTION 'Contract is already settled'
            USING ERRCODE = 'P0010';
    END IF;
    
    -- Update contract with sell details
    UPDATE contracts
    SET sell_price = p_sell_price,
        sell_time = v_sell_time,
        sell_ohlcs = p_sell_ohlcs
    WHERE contract_id = p_contract_id;
    
    -- Credit sell_price (payout) to balance
    v_new_balance := v_current_balance + p_sell_price;
    
    UPDATE accounts
    SET balance = v_new_balance, updated_at = v_sell_time
    WHERE account_id = p_account_id;
    
    -- Create SELL transaction (positive or zero amount)
    INSERT INTO transactions (account_id, type, amount, contract_id, transaction_time)
    VALUES (p_account_id, 'SELL', p_sell_price, p_contract_id, v_sell_time)
    RETURNING transaction_id INTO v_txn_id;
    
    RETURN QUERY SELECT v_txn_id, v_new_balance, v_sell_time;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION close_trade IS 'Atomic contract settlement (SELL).
Parameters:
  p_account_id - Owner account
  p_contract_id - Contract to settle
  p_sell_price - Payout amount (0 for loss, calculated by application)
  p_sell_ohlcs - Candles 11-20 as JSONB
  
Returns:
  sell_transaction_id - SELL transaction ID
  new_balance - Updated account balance
  sell_time - Settlement timestamp
  
Note: Outcome determination and payout calculation is done by the application.
The procedure only handles atomic balance update and record creation.
  
Error codes:
  P0001 - Account not found
  P0003 - Invalid sell_price (negative)
  P0008 - Contract not found
  P0009 - Contract account mismatch
  P0010 - Contract already settled';
```

---

## Error Code Reference

| Code | Description | Procedure(s) |
|------|-------------|--------------|
| P0001 | Account not found | All |
| P0002 | Insufficient balance | withdraw_funds, open_trade |
| P0003 | Invalid amount / payout | deposit_funds, withdraw_funds, open_trade, close_trade |
| P0004 | Invalid sentiment | open_trade |
| P0005 | Price series not found or expired | open_trade |
| P0006 | Price series account mismatch | open_trade |
| P0007 | Series type not active | open_trade |
| P0008 | Contract not found | close_trade |
| P0009 | Contract account mismatch | close_trade |
| P0010 | Contract already settled | close_trade |

---

## Transaction Amount Convention

| Type | Amount Sign | Description |
|------|-------------|-------------|
| DEPOSIT | Positive (+) | Funds added to account |
| WITHDRAWAL | Negative (-) | Funds removed from account |
| BUY | Negative (-) | Stake deducted for contract purchase |
| SELL | Positive/Zero (+/0) | Payout credited (0 for loss) |

**Invariant**: `SUM(amount) FROM transactions WHERE account_id = X` = `balance FROM accounts WHERE account_id = X`

---

## Migration Order

1. `000001_create_accounts.up.sql` - Accounts table
2. `000002_create_series_types.up.sql` - Series configuration
3. `000003_create_transactions.up.sql` - Transactions (partial, FK added later)
4. `000004_create_price_series.up.sql` - Quote storage
5. `000005_create_contracts.up.sql` - Contracts + FK constraint
6. `000010_create_deposit_funds.up.sql` - Deposit procedure
7. `000011_create_withdraw_funds.up.sql` - Withdrawal procedure
8. `000012_create_open_trade.up.sql` - Buy procedure
9. `000013_create_close_trade.up.sql` - Sell procedure

---

## Usage Examples

### Deposit Funds
```sql
SELECT * FROM deposit_funds('SW123', 100.00, 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid);
-- Returns: transaction_id, new_balance, is_duplicate
```

### Withdraw Funds
```sql
SELECT * FROM withdraw_funds('SW123', 50.00, 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid);
-- Returns: transaction_id, new_balance, is_duplicate
```

### Open Trade (Buy)
```sql
SELECT * FROM open_trade('SW123', 42, 'rise', 10.00);  -- 10.00 = buy_price (stake)
-- Returns: contract_id, buy_transaction_id, new_balance, buy_ohlcs, buy_time
```

### Close Trade (Sell)
```sql
-- Application determines outcome and calculates sell_price (payout):
-- 1. Generate execution candles (11-20)
-- 2. Determine win/loss based on sentiment and price movement
-- 3. Calculate sell_price: win ? buy_price * payout_multiplier : 0

SELECT * FROM close_trade(
    'SW123',                                    -- account_id
    1001,                                       -- contract_id
    18.87,                                      -- sell_price (payout, calculated by app)
    '[{"open":1002.0, "high":1006.0, ...}]'::jsonb   -- sell_ohlcs (candles 11-20)
);
-- Returns: sell_transaction_id, new_balance, sell_time
```

### Query Open Contracts
```sql
SELECT contract_id, series_type, sentiment, buy_price, buy_time
FROM contracts
WHERE account_id = 'SW123' AND sell_time IS NULL;
```

### Query Settled Contracts
```sql
SELECT contract_id, series_type, sentiment, buy_price, sell_price,
       CASE WHEN sell_price > 0 THEN 'WIN' ELSE 'LOSS' END AS outcome
FROM contracts
WHERE account_id = 'SW123' AND sell_time IS NOT NULL
ORDER BY sell_time DESC
LIMIT 50;
```

---

## Cleanup Jobs

```sql
-- Validate balance consistency (audit)
-- Sum of all transactions should equal account balance
SELECT a.account_id, a.balance AS stored_balance,
       COALESCE(SUM(t.amount), 0) AS calculated_balance,
       a.balance - COALESCE(SUM(t.amount), 0) AS discrepancy
FROM accounts a
LEFT JOIN transactions t ON a.account_id = t.account_id
GROUP BY a.account_id, a.balance
HAVING a.balance != COALESCE(SUM(t.amount), 0);

-- Note: Price series do not expire. They are deleted when consumed by open_trade.
-- Orphaned price series (from abandoned quotes) can be cleaned up based on age:
DELETE FROM price_series WHERE created_at < NOW() - INTERVAL '24 hours';
```
