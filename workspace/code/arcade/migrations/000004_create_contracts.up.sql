-- Create contracts table for trading history
CREATE TABLE contracts (
    contract_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(20) NOT NULL REFERENCES accounts(account_id),
    series_type VARCHAR(10) NOT NULL,
    sentiment VARCHAR(10) NOT NULL,
    stake DECIMAL(18,2) NOT NULL,
    payout DECIMAL(18,2) NOT NULL,
    ohlcs JSONB NOT NULL,
    purchase_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_series_type_valid CHECK (series_type IN ('Vol50', 'Vol100', 'Vol200', 'Vol300')),
    CONSTRAINT chk_sentiment_valid CHECK (sentiment IN ('rise', 'fall')),
    CONSTRAINT chk_stake_positive CHECK (stake > 0),
    CONSTRAINT chk_payout_non_negative CHECK (payout >= 0)
);

-- Index for account trading history (most recent first)
CREATE INDEX idx_contracts_account_time 
    ON contracts(account_id, purchase_time DESC);

-- Index for filtered trading history by series type
CREATE INDEX idx_contracts_account_series 
    ON contracts(account_id, series_type, purchase_time DESC);
