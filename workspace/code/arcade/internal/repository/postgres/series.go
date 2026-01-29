package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deriv/arcade/internal/series"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeriesRepository implements series.Repository interface
type SeriesRepository struct {
	pool *pgxpool.Pool
}

// NewSeriesRepository creates a new series repository
func NewSeriesRepository(pool *pgxpool.Pool) *SeriesRepository {
	return &SeriesRepository{pool: pool}
}

// GetSeriesConfig retrieves and parses series configuration
func (r *SeriesRepository) GetSeriesConfig(ctx context.Context, seriesType string) (*series.SeriesConfig, error) {
	query := `
		SELECT config, is_active
		FROM series_types
		WHERE series_type = $1
	`

	var configJSON []byte
	var isActive bool

	err := r.pool.QueryRow(ctx, query, seriesType).Scan(&configJSON, &isActive)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("series type not found: %s", seriesType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get series type: %w", err)
	}

	if !isActive {
		return nil, fmt.Errorf("series type is not active: %s", seriesType)
	}

	// Parse config
	var configMap map[string]interface{}
	if err := json.Unmarshal(configJSON, &configMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return series.ParseConfig(configMap)
}

// ListActiveSeries retrieves all active series types
func (r *SeriesRepository) ListActiveSeries(ctx context.Context) ([]series.SeriesType, error) {
	query := `
		SELECT series_type, config, is_active, created_at
		FROM series_types
		WHERE is_active = TRUE
		ORDER BY series_type
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list series types: %w", err)
	}
	defer rows.Close()

	var seriesTypes []series.SeriesType
	for rows.Next() {
		var st series.SeriesType
		var configJSON []byte

		err := rows.Scan(
			&st.SeriesType,
			&configJSON,
			&st.IsActive,
			&st.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan series type: %w", err)
		}

		// Unmarshal config
		if err := json.Unmarshal(configJSON, &st.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		seriesTypes = append(seriesTypes, st)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating series types: %w", err)
	}

	return seriesTypes, nil
}
