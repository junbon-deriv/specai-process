package series

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for series types
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new series repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetSeriesType retrieves series type configuration from database
func (r *Repository) GetSeriesType(ctx context.Context, seriesType string) (*SeriesType, error) {
	query := `
		SELECT series_type, config, is_active, created_at
		FROM series_types
		WHERE series_type = $1
	`

	var st SeriesType
	var configJSON []byte

	err := r.pool.QueryRow(ctx, query, seriesType).Scan(
		&st.SeriesType,
		&configJSON,
		&st.IsActive,
		&st.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("series type not found: %s", seriesType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get series type: %w", err)
	}

	// Unmarshal config
	if err := json.Unmarshal(configJSON, &st.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &st, nil
}

// GetSeriesConfig retrieves and parses series configuration
func (r *Repository) GetSeriesConfig(ctx context.Context, seriesType string) (*SeriesConfig, error) {
	st, err := r.GetSeriesType(ctx, seriesType)
	if err != nil {
		return nil, err
	}

	if !st.IsActive {
		return nil, fmt.Errorf("series type is not active: %s", seriesType)
	}

	return ParseConfig(st.Config)
}

// ListActiveSeries retrieves all active series types
func (r *Repository) ListActiveSeries(ctx context.Context) ([]SeriesType, error) {
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

	var seriesTypes []SeriesType
	for rows.Next() {
		var st SeriesType
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
