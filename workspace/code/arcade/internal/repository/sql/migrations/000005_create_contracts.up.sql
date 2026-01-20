-- Create contracts table for trading history
CREATE TABLE contracts (
    contract_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(20) NOT NULL REFERENCES accounts(account_id),
    series_type VARCHAR(10) NOT NULL REFERENCES series_types(series_type),
    sentiment VARCHAR(10) NOT NULL,
    
    -- Buy details (populated at open_trade)
    buy_price DECIMAL(18,6) NOT NULL,           -- Stake amount (entry price)
    buy_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    buy_ohlcs JSONB NOT NULL,                   -- Candles 1-10 from preview
    
    -- Sell details (NULL = contract is OPEN, populated at close_trade)
    sell_price DECIMAL(18,6),                   -- Payout amount (exit price)
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
