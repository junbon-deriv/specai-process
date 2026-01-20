package series

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Generator interface for price generation algorithms
type Generator interface {
	// GenerateCandles generates OHLC candles based on configuration
	GenerateCandles(initialValue decimal.Decimal, config *SeriesConfig, count int, startTime time.Time) ([]OHLC, error)
}

// GeneratorFactory creates generators based on type
type GeneratorFactory struct {
	generators map[string]Generator
}

// NewGeneratorFactory creates a new generator factory
func NewGeneratorFactory() *GeneratorFactory {
	factory := &GeneratorFactory{
		generators: make(map[string]Generator),
	}

	// Register default generators
	factory.Register("gbm", NewGBMGenerator())

	return factory
}

// Register adds a generator to the factory
func (f *GeneratorFactory) Register(generatorType string, generator Generator) {
	f.generators[generatorType] = generator
}

// GetGenerator returns a generator by type
func (f *GeneratorFactory) GetGenerator(generatorType string) (Generator, error) {
	gen, ok := f.generators[generatorType]
	if !ok {
		return nil, fmt.Errorf("unknown generator type: %s", generatorType)
	}
	return gen, nil
}
