-- Create series_types table with JSONB configuration
CREATE TABLE series_types (
    series_type VARCHAR(10) PRIMARY KEY,
    config JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Validate required config fields exist
    CONSTRAINT chk_config_required_fields CHECK (
        config ? 'initial_value' AND
        config ? 'volatility' AND
        config ? 'drift' AND
        config ? 'interval_seconds' AND
        config ? 'display_name' AND
        config ? 'payout_multiplier'
    )
);

-- Seed default series types
INSERT INTO series_types (series_type, config) VALUES
    ('Vol50', '{
        "display_name": "Volatility 50",
        "initial_value": 1000.00,
        "volatility": 0.50,
        "drift": 0.0,
        "interval_seconds": 1,
        "payout_multiplier": 1.8868,
        "generator_type": "gbm"
    }'::jsonb),
    ('Vol100', '{
        "display_name": "Volatility 100",
        "initial_value": 1000.00,
        "volatility": 1.00,
        "drift": 0.0,
        "interval_seconds": 1,
        "payout_multiplier": 1.8868,
        "generator_type": "gbm"
    }'::jsonb),
    ('Vol200', '{
        "display_name": "Volatility 200",
        "initial_value": 1000.00,
        "volatility": 2.00,
        "drift": 0.0,
        "interval_seconds": 1,
        "payout_multiplier": 1.8868,
        "generator_type": "gbm"
    }'::jsonb),
    ('Vol300', '{
        "display_name": "Volatility 300",
        "initial_value": 1000.00,
        "volatility": 3.00,
        "drift": 0.0,
        "interval_seconds": 1,
        "payout_multiplier": 1.8868,
        "generator_type": "gbm"
    }'::jsonb);

CREATE INDEX idx_series_types_active ON series_types(is_active) WHERE is_active = TRUE;
