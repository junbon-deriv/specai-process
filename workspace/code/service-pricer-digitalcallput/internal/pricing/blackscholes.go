package pricing

import (
	"math"

	"github.com/shopspring/decimal"
)

// BlackScholes implements digital option pricing using the Black-Scholes model
type BlackScholes struct {
	volatility float64 // Fixed at 10% (0.10)
	rate       float64 // Fixed at 0% (0.00)
}

// Calculate returns the probability of winning for a digital option
// durationYears is the time to expiry in years
func (bs *BlackScholes) Calculate(spot, barrier decimal.Decimal, durationYears float64) decimal.Decimal {
	if durationYears <= 0 {
		return decimal.Zero
	}

	// Standard Black-Scholes for digital options
	// d2 = (ln(S/K) + (r - σ²/2)T) / (σ√T)
	// Probability = N(d2) for calls, N(-d2) for puts
	S := spot.InexactFloat64()
	K := barrier.InexactFloat64()
	T := durationYears
	σ := bs.volatility
	r := bs.rate

	d2 := (math.Log(S/K) + (r-σ*σ/2)*T) / (σ * math.Sqrt(T))
	probability := normalCDF(d2)

	return decimal.NewFromFloat(probability)
}

// CalculateForCall returns the probability for a call option
func (bs *BlackScholes) CalculateForCall(spot, barrier decimal.Decimal, durationYears float64) decimal.Decimal {
	if durationYears <= 0 {
		// If already at expiry, check if it's a win
		if spot.GreaterThan(barrier) {
			return decimal.NewFromInt(1)
		}
		return decimal.Zero
	}

	S := spot.InexactFloat64()
	K := barrier.InexactFloat64()
	T := durationYears
	σ := bs.volatility
	r := bs.rate

	d2 := (math.Log(S/K) + (r-σ*σ/2)*T) / (σ * math.Sqrt(T))
	probability := normalCDF(d2)

	return decimal.NewFromFloat(probability)
}

// CalculateForPut returns the probability for a put option
func (bs *BlackScholes) CalculateForPut(spot, barrier decimal.Decimal, durationYears float64) decimal.Decimal {
	if durationYears <= 0 {
		// If already at expiry, check if it's a win
		if spot.LessThan(barrier) {
			return decimal.NewFromInt(1)
		}
		return decimal.Zero
	}

	S := spot.InexactFloat64()
	K := barrier.InexactFloat64()
	T := durationYears
	σ := bs.volatility
	r := bs.rate

	d2 := (math.Log(S/K) + (r-σ*σ/2)*T) / (σ * math.Sqrt(T))
	probability := normalCDF(-d2) // Note the negative sign for puts

	return decimal.NewFromFloat(probability)
}

// normalCDF returns the cumulative distribution function of the standard normal distribution
func normalCDF(x float64) float64 {
	// Using the approximation from Abramowitz and Stegun
	// Maximum error: 7.5e-8
	sign := 1.0
	if x < 0 {
		sign = -1.0
		x = -x
	}

	t := 1.0 / (1.0 + 0.2316419*x)
	y := t * (0.31938153 + t*(-0.356563782+t*(1.781477937+t*(-1.821255978+t*1.330274429))))
	cdf := 1.0 - (1.0/math.Sqrt(2.0*math.Pi))*math.Exp(-0.5*x*x)*y

	if sign < 0 {
		cdf = 1.0 - cdf
	}

	return cdf
}
