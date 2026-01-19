package trading

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deriv/arcade/internal/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Repository handles database operations for trading
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new trading repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreatePriceSeries stores a price preview series
func (r *Repository) CreatePriceSeries(ctx context.Context, accountID, seriesType string, candles []OHLC, quoteValue string) (*PriceSeries, error) {
	// Convert candles to JSON
	candlesJSON, err := json.Marshal(candles)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal candles: %w", err)
	}

	query := `
		INSERT INTO price_series (account_id, series_type, candles, quote_value, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING series_id, account_id, series_type, candles, quote_value, created_at
	`

	var ps PriceSeries
	var candlesJSONB []byte

	err = r.pool.QueryRow(ctx, query, accountID, seriesType, candlesJSON, quoteValue).Scan(
		&ps.SeriesID,
		&ps.AccountID,
		&ps.SeriesType,
		&candlesJSONB,
		&ps.QuoteValue,
		&ps.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create price series: %w", err)
	}

	// Unmarshal candles
	if err := json.Unmarshal(candlesJSONB, &ps.Candles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal candles: %w", err)
	}

	return &ps, nil
}

// FindPriceSeries finds price series by account, series type, and quote value
func (r *Repository) FindPriceSeries(ctx context.Context, accountID, seriesType, quoteValue string) (*PriceSeries, error) {
	query := `
		SELECT series_id, account_id, series_type, candles, quote_value, created_at
		FROM price_series
		WHERE account_id = $1 AND series_type = $2 AND quote_value = $3
		LIMIT 1
	`

	tx, inTx := common.GetTx(ctx)

	var ps PriceSeries
	var candlesJSONB []byte
	var err error

	if inTx {
		err = tx.QueryRow(ctx, query, accountID, seriesType, quoteValue).Scan(
			&ps.SeriesID,
			&ps.AccountID,
			&ps.SeriesType,
			&candlesJSONB,
			&ps.QuoteValue,
			&ps.CreatedAt,
		)
	} else {
		err = r.pool.QueryRow(ctx, query, accountID, seriesType, quoteValue).Scan(
			&ps.SeriesID,
			&ps.AccountID,
			&ps.SeriesType,
			&candlesJSONB,
			&ps.QuoteValue,
			&ps.CreatedAt,
		)
	}

	if err == pgx.ErrNoRows {
		return nil, common.ErrInvalidQuote
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find price series: %w", err)
	}

	// Unmarshal candles
	if err := json.Unmarshal(candlesJSONB, &ps.Candles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal candles: %w", err)
	}

	return &ps, nil
}

// DeletePriceSeries removes a price series
func (r *Repository) DeletePriceSeries(ctx context.Context, seriesID int64) error {
	tx, ok := common.GetTx(ctx)
	if !ok {
		return fmt.Errorf("DeletePriceSeries must be called within a transaction")
	}

	query := `DELETE FROM price_series WHERE series_id = $1`

	_, err := tx.Exec(ctx, query, seriesID)
	if err != nil {
		return fmt.Errorf("failed to delete price series: %w", err)
	}

	return nil
}

// CreateContractInitial creates a new contract with initial data (Phase 1: reserve ID)
// DEPRECATED: Use open_trade stored procedure instead
func (r *Repository) CreateContractInitial(ctx context.Context, accountID, seriesType, sentiment string, buyPrice decimal.Decimal, initialCandles []OHLC) (*Contract, error) {
	tx, ok := common.GetTx(ctx)
	if !ok {
		return nil, fmt.Errorf("CreateContractInitial must be called within a transaction")
	}

	// Convert initial candles to JSON (only candles 1-10)
	ohlcsJSON, err := json.Marshal(initialCandles)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initial candles: %w", err)
	}

	// Create contract with buy details (sell fields NULL = OPEN)
	query := `
		INSERT INTO contracts (account_id, series_type, sentiment, buy_price, buy_ohlcs, buy_time)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING contract_id, account_id, series_type, sentiment, buy_price, buy_time
	`

	var contract Contract

	err = tx.QueryRow(ctx, query, accountID, seriesType, sentiment, buyPrice, ohlcsJSON).Scan(
		&contract.ContractID,
		&contract.AccountID,
		&contract.SeriesType,
		&contract.Sentiment,
		&contract.BuyPrice,
		&contract.BuyTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial contract: %w", err)
	}

	contract.BuyOHLCs = initialCandles

	return &contract, nil
}

// UpdateContractComplete updates contract with full data (Phase 2: finalize)
// DEPRECATED: Use close_trade stored procedure instead
func (r *Repository) UpdateContractComplete(ctx context.Context, contractID int64, sellCandles []OHLC, sellPrice decimal.Decimal) error {
	tx, ok := common.GetTx(ctx)
	if !ok {
		return fmt.Errorf("UpdateContractComplete must be called within a transaction")
	}

	// Convert sell candles (11-20) to JSON
	ohlcsJSON, err := json.Marshal(sellCandles)
	if err != nil {
		return fmt.Errorf("failed to marshal sell candles: %w", err)
	}

	query := `
		UPDATE contracts
		SET sell_ohlcs = $1, sell_price = $2, sell_time = NOW()
		WHERE contract_id = $3
	`

	_, err = tx.Exec(ctx, query, ohlcsJSON, sellPrice, contractID)
	if err != nil {
		return fmt.Errorf("failed to update contract: %w", err)
	}

	return nil
}

// ListContracts retrieves contracts with optional series filter
func (r *Repository) ListContracts(ctx context.Context, accountID string, seriesType *string) ([]Contract, error) {
	var query string
	var args []interface{}

	if seriesType != nil {
		query = `
			SELECT contract_id, account_id, series_type, sentiment,
			       buy_price, buy_time, buy_ohlcs,
			       sell_price, sell_time, sell_ohlcs
			FROM contracts
			WHERE account_id = $1 AND series_type = $2
			ORDER BY buy_time DESC
			LIMIT 50
		`
		args = []interface{}{accountID, *seriesType}
	} else {
		query = `
			SELECT contract_id, account_id, series_type, sentiment,
			       buy_price, buy_time, buy_ohlcs,
			       sell_price, sell_time, sell_ohlcs
			FROM contracts
			WHERE account_id = $1
			ORDER BY buy_time DESC
			LIMIT 50
		`
		args = []interface{}{accountID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list contracts: %w", err)
	}
	defer rows.Close()

	var contracts []Contract
	for rows.Next() {
		var contract Contract
		var buyOHLCsJSON []byte
		var sellOHLCsJSON []byte

		err := rows.Scan(
			&contract.ContractID,
			&contract.AccountID,
			&contract.SeriesType,
			&contract.Sentiment,
			&contract.BuyPrice,
			&contract.BuyTime,
			&buyOHLCsJSON,
			&contract.SellPrice,
			&contract.SellTime,
			&sellOHLCsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contract: %w", err)
		}

		// Unmarshal buy OHLCs
		if err := json.Unmarshal(buyOHLCsJSON, &contract.BuyOHLCs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal buy ohlcs: %w", err)
		}

		// Unmarshal sell OHLCs if present
		if sellOHLCsJSON != nil {
			if err := json.Unmarshal(sellOHLCsJSON, &contract.SellOHLCs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal sell ohlcs: %w", err)
			}
		}

		contracts = append(contracts, contract)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating contracts: %w", err)
	}

	return contracts, nil
}
