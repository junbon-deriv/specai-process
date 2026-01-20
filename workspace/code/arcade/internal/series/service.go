package series

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Repository interface for series data operations
type Repository interface {
	// GetSeriesConfig retrieves and parses series configuration
	GetSeriesConfig(ctx context.Context, seriesType string) (*SeriesConfig, error)

	// ListActiveSeries retrieves all active series types
	ListActiveSeries(ctx context.Context) ([]SeriesType, error)
}

// Service handles series-related business logic
type Service struct {
	repo    Repository
	factory *GeneratorFactory
}

// NewService creates a new series service
func NewService(repo Repository) *Service {
	return &Service{
		repo:    repo,
		factory: NewGeneratorFactory(),
	}
}

// GetConfig retrieves configuration for a series type
func (s *Service) GetConfig(ctx context.Context, seriesType string) (*SeriesConfig, error) {
	return s.repo.GetSeriesConfig(ctx, seriesType)
}

// GenerateCandles generates price candles for a series type
func (s *Service) GenerateCandles(ctx context.Context, seriesType string, initialValue decimal.Decimal, count int, startTime time.Time) ([]OHLC, error) {
	// Get configuration from database
	config, err := s.repo.GetSeriesConfig(ctx, seriesType)
	if err != nil {
		return nil, err
	}

	// Get appropriate generator based on config
	generator, err := s.factory.GetGenerator(config.GeneratorType)
	if err != nil {
		return nil, err
	}

	// Generate candles using the configured algorithm
	candles, err := generator.GenerateCandles(initialValue, config, count, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to generate candles: %w", err)
	}

	return candles, nil
}

// ListActiveSeries retrieves all active series types
func (s *Service) ListActiveSeries(ctx context.Context) ([]SeriesType, error) {
	return s.repo.ListActiveSeries(ctx)
}
