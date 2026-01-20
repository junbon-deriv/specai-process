package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/deriv/arcade/internal/common"
	"github.com/deriv/arcade/internal/series"
	"github.com/deriv/arcade/internal/trading"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// TradingRepository implements trading.Repository interface
type TradingRepository struct {
	pool *pgxpool.Pool
}

// NewTradingRepository creates a new trading repository
func NewTradingRepository(pool *pgxpool.Pool) *TradingRepository {
	return &TradingRepository{pool: pool}
}

// CreatePriceSeries stores a price preview series
func (r *TradingRepository) CreatePriceSeries(ctx context.Context, accountID, seriesType string, candles []series.OHLC, quoteValue string) (*trading.PriceSeries, error) {
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

	var ps trading.PriceSeries
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
func (r *TradingRepository) FindPriceSeries(ctx context.Context, accountID, seriesType, quoteValue string) (*trading.PriceSeries, error) {
	query := `
		SELECT series_id, account_id, series_type, candles, quote_value, created_at
		FROM price_series
		WHERE account_id = $1 AND series_type = $2 AND quote_value = $3
		LIMIT 1
	`

	var ps trading.PriceSeries
	var candlesJSONB []byte

	err := r.pool.QueryRow(ctx, query, accountID, seriesType, quoteValue).Scan(
		&ps.SeriesID,
		&ps.AccountID,
		&ps.SeriesType,
		&candlesJSONB,
		&ps.QuoteValue,
		&ps.CreatedAt,
	)
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

// ListContracts retrieves contracts with optional series filter
func (r *TradingRepository) ListContracts(ctx context.Context, accountID string, seriesType *string) ([]trading.Contract, error) {
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

	var contracts []trading.Contract
	for rows.Next() {
		var contract trading.Contract
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

// OpenTrade calls open_trade stored procedure
func (r *TradingRepository) OpenTrade(ctx context.Context, accountID string, seriesID int64, sentiment string, buyPrice decimal.Decimal) (*trading.OpenTradeResult, error) {
	var contractID int64
	var buyTxnID int64
	var newBalance decimal.Decimal
	var buyOHLCsJSON []byte
	var buyTime time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT contract_id, buy_transaction_id, new_balance, buy_ohlcs, buy_time
		FROM open_trade($1, $2, $3, $4)
	`, accountID, seriesID, sentiment, buyPrice).Scan(
		&contractID,
		&buyTxnID,
		&newBalance,
		&buyOHLCsJSON,
		&buyTime,
	)
	if err != nil {
		return nil, mapPgError(err)
	}

	// Unmarshal buy candles
	var buyOHLCs []series.OHLC
	if err := json.Unmarshal(buyOHLCsJSON, &buyOHLCs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal buy candles: %w", err)
	}

	return &trading.OpenTradeResult{
		ContractID:       contractID,
		BuyTransactionID: buyTxnID,
		NewBalance:       newBalance,
		BuyOHLCs:         buyOHLCs,
		BuyTime:          buyTime,
	}, nil
}

// CloseTrade calls close_trade stored procedure
func (r *TradingRepository) CloseTrade(ctx context.Context, accountID string, contractID int64, sellPrice decimal.Decimal, sellOHLCs []series.OHLC) (*trading.CloseTradeResult, error) {
	// Marshal sell candles
	sellCandlesJSON, err := json.Marshal(sellOHLCs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sell candles: %w", err)
	}

	var sellTxnID int64
	var newBalance decimal.Decimal
	var sellTime time.Time

	err = r.pool.QueryRow(ctx, `
		SELECT sell_transaction_id, new_balance, sell_time
		FROM close_trade($1, $2, $3, $4)
	`, accountID, contractID, sellPrice, sellCandlesJSON).Scan(
		&sellTxnID,
		&newBalance,
		&sellTime,
	)
	if err != nil {
		return nil, mapPgError(err)
	}

	return &trading.CloseTradeResult{
		SellTransactionID: sellTxnID,
		NewBalance:        newBalance,
		SellTime:          sellTime,
	}, nil
}
