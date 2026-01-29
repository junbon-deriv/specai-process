-- Create price_series table for quote preview storage
CREATE TABLE price_series (
    series_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id VARCHAR(20) REFERENCES accounts(account_id),  -- Nullable, not associated until trade
    series_type VARCHAR(10) NOT NULL REFERENCES series_types(series_type),
    candles JSONB NOT NULL,
    quote_value VARCHAR(30) NOT NULL,  -- Last candle close as string for reference
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for series lookup by ID (primary use case)
CREATE INDEX idx_price_series_id ON price_series(series_id);

-- Index for account's active series
CREATE INDEX idx_price_series_account ON price_series(account_id, created_at DESC);
