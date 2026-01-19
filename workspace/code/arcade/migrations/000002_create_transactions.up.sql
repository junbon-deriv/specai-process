-- Create transactions table for financial movements
CREATE TABLE transactions (
    transaction_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(20) NOT NULL REFERENCES accounts(account_id),
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(18,2) NOT NULL,
    idempotency_id UUID,
    reference_id BIGINT,
    transaction_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_type_valid CHECK (type IN ('DEPOSIT', 'WITHDRAWAL', 'STAKE', 'PAYOUT')),
    CONSTRAINT chk_amount_non_negative CHECK (amount >= 0)
);

-- Unique index for idempotency (only for DEPOSIT/WITHDRAWAL)
CREATE UNIQUE INDEX idx_transactions_idempotency 
    ON transactions(account_id, idempotency_id) 
    WHERE idempotency_id IS NOT NULL;

-- Index for account transactions lookup
CREATE INDEX idx_transactions_account ON transactions(account_id);

-- Index for contract reference lookup
CREATE INDEX idx_transactions_reference ON transactions(reference_id) WHERE reference_id IS NOT NULL;
