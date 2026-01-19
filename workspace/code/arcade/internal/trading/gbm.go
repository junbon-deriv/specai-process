package trading

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
// - μ is the drift rate (interest rate - quanto drift)
// - σ is the volatility
// - dt is the time interval (1 second)
// - dW is the random increment (normal distribution)
func (g *GBMGenerator) GenerateCandles(initialPrice decimal.Decimal, config SeriesConfig, count int, startTime time.Time) []OHLC {
	candles := make([]OHLC, count)
	currentPrice := initialPrice
	currentTime := startTime

	dt := 1.0 // 1 second interval in seconds
	drift := config.InterestRate - config.QuantoDrift
	volatility := config.Volatility

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
		open = roundToPrecision(open, config.Precision)
		high = roundToPrecision(high, config.Precision)
		low = roundToPrecision(low, config.Precision)
		close = roundToPrecision(close, config.Precision)

		candles[i] = OHLC{
			Timestamp: currentTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
		}

		// Update for next candle
		currentPrice = close
		currentTime = currentTime.Add(config.Interval)
	}

	return candles
}

// roundToPrecision rounds decimal to specified number of decimal places
func roundToPrecision(d decimal.Decimal, precision int32) decimal.Decimal {
	return d.Round(precision)
}
