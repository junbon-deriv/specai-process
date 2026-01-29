-- Create accounts table with SW-prefixed sequential IDs
CREATE SEQUENCE account_id_seq START 1;

CREATE TABLE accounts (
    account_id VARCHAR(20) PRIMARY KEY,
    external_id VARCHAR(255),
    currency CHAR(3) NOT NULL,
    balance DECIMAL(18,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT chk_currency_format CHECK (currency ~ '^[A-Z]{3}$')
);

CREATE INDEX idx_accounts_external_id ON accounts(external_id) WHERE external_id IS NOT NULL;
