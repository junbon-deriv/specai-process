-- Create price_series table for temporary preview storage
CREATE TABLE price_series (
    series_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(20) NOT NULL REFERENCES accounts(account_id),
    series_type VARCHAR(10) NOT NULL,
    candles JSONB NOT NULL,
    quote_value VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_series_type_valid CHECK (series_type IN ('Vol50', 'Vol100', 'Vol200', 'Vol300'))
);

-- Index for quote lookup during SwipeBuy
CREATE INDEX idx_price_series_lookup 
    ON price_series(account_id, series_type, quote_value);

-- Index for cleanup of old records
CREATE INDEX idx_price_series_cleanup 
    ON price_series(created_at);
