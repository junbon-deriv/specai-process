-- Create transactions table for financial movements
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

-- Unique index for idempotency (only for DEPOSIT/WITHDRAWAL)
CREATE UNIQUE INDEX idx_transactions_idempotency 
    ON transactions(account_id, idempotency_id) 
    WHERE idempotency_id IS NOT NULL;

-- Index for account transactions lookup
CREATE INDEX idx_transactions_account_time ON transactions(account_id, transaction_time DESC);

-- Index for contract transactions lookup
CREATE INDEX idx_transactions_contract ON transactions(contract_id) WHERE contract_id IS NOT NULL;
