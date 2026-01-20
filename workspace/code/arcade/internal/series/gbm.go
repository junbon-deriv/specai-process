package series

import (
	"math"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"
)

// GBMGenerator generates OHLC candles using Geometric Brownian Motion
type GBMGenerator struct {
	rand *rand.Rand
}

// NewGBMGenerator creates a new GBM generator
func NewGBMGenerator() *GBMGenerator {
	return &GBMGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateCandles creates OHLC candles using GBM algorithm
// Formula: dS = μSdt + σSdW where:
// - S is the price
// - μ is the drift rate
// - σ is the volatility
// - dt is the time interval
// - dW is the random increment (normal distribution)
func (g *GBMGenerator) GenerateCandles(initialPrice decimal.Decimal, config *SeriesConfig, count int, startTime time.Time) ([]OHLC, error) {
	candles := make([]OHLC, count)
	currentPrice := initialPrice
	currentTime := startTime

	dt := float64(config.IntervalSeconds)
	drift := config.Drift
	volatility := config.Volatility
	precision := int32(3) // Default precision

	for i := 0; i < count; i++ {
		// Generate OHLC for this candle
		open := currentPrice

		// Generate 4 price movements within the candle to create realistic OHLC
		prices := []decimal.Decimal{open}
		tmpPrice := open

		for j := 0; j < 3; j++ {
			// Generate random number from standard normal distribution
			z := g.rand.NormFloat64()

			// Calculate price change using GBM formula
			// S(t+dt) = S(t) * exp((μ - σ²/2)dt + σ√dt*Z)
			priceFloat, _ := tmpPrice.Float64()
			exponent := (drift-volatility*volatility/2)*dt + volatility*math.Sqrt(dt)*z
			newPriceFloat := priceFloat * math.Exp(exponent)
			tmpPrice = decimal.NewFromFloat(newPriceFloat)

			prices = append(prices, tmpPrice)
		}

		// Determine high and low from all prices
		high := prices[0]
		low := prices[0]
		for _, p := range prices {
			if p.GreaterThan(high) {
				high = p
			}
			if p.LessThan(low) {
				low = p
			}
		}

		// Close is the last price
		close := prices[len(prices)-1]

		// Round to specified precision
		open = open.Round(precision)
		high = high.Round(precision)
		low = low.Round(precision)
		close = close.Round(precision)

		candles[i] = OHLC{
			Timestamp: currentTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
		}

		// Update for next candle
		currentPrice = close
		currentTime = currentTime.Add(time.Duration(config.IntervalSeconds) * time.Second)
	}

	return candles, nil
}
