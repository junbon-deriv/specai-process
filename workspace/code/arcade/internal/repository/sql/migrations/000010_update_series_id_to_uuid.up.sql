-- Migration to change series_id from BIGSERIAL to UUID
-- This is a breaking change that requires data migration

BEGIN;

-- Step 1: Drop the open_trade function (depends on series_id type)
DROP FUNCTION IF EXISTS open_trade(VARCHAR, BIGINT, VARCHAR, DECIMAL);

-- Step 2: Drop existing data from price_series (since we can't convert BIGSERIAL to UUID)
-- WARNING: This will delete all existing price series previews
TRUNCATE TABLE price_series CASCADE;

-- Step 3: Drop the series_id column and recreate as UUID
ALTER TABLE price_series DROP COLUMN series_id;
ALTER TABLE price_series ADD COLUMN series_id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- Step 4: Recreate open_trade function with UUID parameter
CREATE OR REPLACE FUNCTION open_trade(
    p_account_id VARCHAR(20),
    p_series_id UUID,               -- Changed from BIGINT to UUID
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
  p_series_id - Price series UUID from quote
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

COMMIT;
