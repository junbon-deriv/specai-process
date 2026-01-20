-- Create deposit_funds stored procedure
CREATE OR REPLACE FUNCTION deposit_funds(
    p_account_id VARCHAR(20),
    p_amount DECIMAL(18,2),
    p_idempotency_id UUID
)
RETURNS TABLE (
    transaction_id BIGINT,
    new_balance DECIMAL(18,2),
    is_duplicate BOOLEAN,
    transaction_time TIMESTAMP WITH TIME ZONE
) AS $$
DECLARE
    v_existing_txn_id BIGINT;
    v_current_balance DECIMAL(18,2);
    v_new_balance DECIMAL(18,2);
    v_txn_id BIGINT;
    v_txn_time TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Validate amount is positive
    IF p_amount <= 0 THEN
        RAISE EXCEPTION 'Deposit amount must be positive: %', p_amount
            USING ERRCODE = 'P0003';
    END IF;

    -- Check idempotency first (read-only, no lock needed)
    SELECT t.transaction_id, t.transaction_time INTO v_existing_txn_id, v_txn_time
    FROM transactions t
    WHERE t.account_id = p_account_id
      AND t.idempotency_id = p_idempotency_id
      AND t.type = 'DEPOSIT';
    
    IF FOUND THEN
        -- Return existing transaction (idempotent response)
        SELECT a.balance INTO v_current_balance
        FROM accounts a
        WHERE a.account_id = p_account_id;
        
        RETURN QUERY SELECT v_existing_txn_id, v_current_balance, TRUE, v_txn_time;
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
    v_txn_time := NOW();
    INSERT INTO transactions (account_id, type, amount, idempotency_id, transaction_time)
    VALUES (p_account_id, 'DEPOSIT', p_amount, p_idempotency_id, v_txn_time)
    RETURNING transactions.transaction_id INTO v_txn_id;
    
    RETURN QUERY SELECT v_txn_id, v_new_balance, FALSE, v_txn_time;
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
