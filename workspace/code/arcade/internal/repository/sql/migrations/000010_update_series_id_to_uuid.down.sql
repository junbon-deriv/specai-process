-- Rollback migration: Revert series_id from UUID back to BIGSERIAL
-- WARNING: This will delete all existing price series data

BEGIN;

-- Step 1: Drop the UUID version of open_trade function
DROP FUNCTION IF EXISTS open_trade(VARCHAR, UUID, VARCHAR, DECIMAL);

-- Step 2: Clear price_series table (UUID data cannot be converted back to BIGSERIAL)
TRUNCATE TABLE price_series CASCADE;

-- Step 3: Drop UUID column and recreate as BIGSERIAL
ALTER TABLE price_series DROP COLUMN series_id;
ALTER TABLE price_series ADD COLUMN series_id BIGSERIAL PRIMARY KEY;

-- Step 4: Recreate open_trade function with BIGINT parameter
CREATE OR REPLACE FUNCTION open_trade(
    p_account_id VARCHAR(20),
    p_series_id BIGINT,             -- Reverted from UUID to BIGINT
    p_sentiment VARCHAR(10),
    p_buy_price DECIMAL(18,6)
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
    IF p_sentiment NOT IN ('rise', 'fall') THEN
        RAISE EXCEPTION 'Invalid sentiment: %. Must be rise or fall', p_sentiment
            USING ERRCODE = 'P0004';
    END IF;
    
    IF p_buy_price <= 0 THEN
        RAISE EXCEPTION 'Buy price must be positive: %', p_buy_price
            USING ERRCODE = 'P0003';
    END IF;

    SELECT balance INTO v_current_balance
    FROM accounts
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_account_id
            USING ERRCODE = 'P0001';
    END IF;
    
    IF v_current_balance < p_buy_price THEN
        RAISE EXCEPTION 'Insufficient balance. Required: %, Available: %', p_buy_price, v_current_balance
            USING ERRCODE = 'P0002';
    END IF;
    
    SELECT ps.account_id, ps.series_type, ps.candles
    INTO v_series_account, v_series_type, v_series_candles
    FROM price_series ps
    WHERE ps.series_id = p_series_id
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Price series not found: %', p_series_id
            USING ERRCODE = 'P0005';
    END IF;
    
    IF v_series_account != p_account_id THEN
        RAISE EXCEPTION 'Price series does not belong to account'
            USING ERRCODE = 'P0006';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM series_types WHERE series_type = v_series_type AND is_active = TRUE) THEN
        RAISE EXCEPTION 'Series type not active: %', v_series_type
            USING ERRCODE = 'P0007';
    END IF;
    
    INSERT INTO contracts (
        account_id, series_type, sentiment,
        buy_price, buy_time, buy_ohlcs
    )
    VALUES (
        p_account_id, v_series_type, p_sentiment,
        p_buy_price, v_buy_time, v_series_candles
    )
    RETURNING contracts.contract_id INTO v_contract_id;
    
    v_new_balance := v_current_balance - p_buy_price;
    
    UPDATE accounts 
    SET balance = v_new_balance, updated_at = v_buy_time
    WHERE account_id = p_account_id;
    
    INSERT INTO transactions (account_id, type, amount, contract_id, transaction_time)
    VALUES (p_account_id, 'BUY', -p_buy_price, v_contract_id, v_buy_time)
    RETURNING transaction_id INTO v_txn_id;
    
    DELETE FROM price_series WHERE series_id = p_series_id;
    
    RETURN QUERY SELECT v_contract_id, v_txn_id, v_new_balance, v_series_candles, v_buy_time;
END;
$$ LANGUAGE plpgsql;

COMMIT;
