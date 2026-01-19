-- Create price_series table for quote preview storage
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
