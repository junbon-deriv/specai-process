package pricer

import (
	"math"
	"strconv"
	"time"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/model"
)

// Pricer handles the core pricing logic
type Pricer struct {
	// Configuration for static parameters (volatility, rate, etc.)
	Volatility   float64
	InterestRate float64
}

func NewPricer() *Pricer {
	return &Pricer{
		Volatility:   0.1,  // Default 10%
		InterestRate: 0.05, // Default 5%
	}
}

// CalculateAsk calculates the Ask price (premium) for a new contract
func (p *Pricer) CalculateAsk(req model.PricingRequest, currentSpot float64) (*model.PricingResult, error) {
	// 1. Parse inputs
	stake, err := strconv.ParseFloat(req.Stake, 64)
	if err != nil {
		return nil, err
	}

	// Validate Min Stake
	minStake := 1.0
	if stake < minStake {
		return nil, model.ErrInvalidStake // We need to define this or just return generic error
	}

	// 2. Determine Barrier
	barrier := currentSpot
	if req.Barrier != nil {
		barrierStr := *req.Barrier
		isRelative := false
		if len(barrierStr) > 0 && (barrierStr[0] == '+' || barrierStr[0] == '-') {
			isRelative = true
		}

		b, err := strconv.ParseFloat(barrierStr, 64)
		if err == nil {
			if isRelative {
				barrier = currentSpot + b
			} else {
				barrier = b
			}
		}
	}

	// 3. Calculate Probability (Simplified Black-Scholes for Digital)
	// Digital Call Price = e^(-rT) * N(d2)
	// Digital Put Price = e^(-rT) * N(-d2)
	// d2 = (ln(S/K) + (r - 0.5*sigma^2)*T) / (sigma * sqrt(T))

	// Parse Duration (simplified)
	duration, _ := time.ParseDuration(req.Duration)
	T := duration.Hours() / 24 / 365 // Time in years

	d2 := (math.Log(currentSpot/barrier) + (p.InterestRate-0.5*p.Volatility*p.Volatility)*T) / (p.Volatility * math.Sqrt(T))

	var probability float64
	if req.ContractType == pb.ContractType_CONTRACT_TYPE_CALL {
		probability = normalCDF(d2)
	} else {
		probability = normalCDF(-d2)
	}

	// Discount factor
	df := math.Exp(-p.InterestRate * T)

	// Price (per unit payout)
	pricePerUnit := df * probability

	// Calculate Payout based on Stake (Premium)
	// If Stake is the cost, then Payout = Stake / PricePerUnit?
	// Or usually, Price = Payout * Probability * DF.
	// Here we are given Stake (Premium). So Payout = Stake / (Probability * DF)
	// But we need to be careful about limits.

	payout := stake / pricePerUnit

	// Validate Max Payout
	maxPayout := 10000.0
	if payout > maxPayout {
		// If payout exceeds max, we can either reject or cap (usually reject in this context)
		// For now, let's reject.
		// return nil, fmt.Errorf("payout %.2f exceeds limit %.2f", payout, maxPayout)
		// But to keep it simple and avoid importing fmt if not needed:
		// Actually, let's just return the result but maybe the client checks?
		// PRD says "Validate... to manage risk". So we should probably error.
	}
	// Re-reading PRD: "Validate that requested contracts are within defined limits".
	// Let's enforce it.

	if payout > maxPayout {
		return nil, model.ErrPayoutExceedsLimit
	}

	return &model.PricingResult{
		Price:           req.Stake, // The ask price is the stake the user pays
		Payout:          strconv.FormatFloat(payout, 'f', 2, 64),
		CurrentSpot:     strconv.FormatFloat(currentSpot, 'f', 2, 64),
		CurrentSpotTime: req.PricingTime.Unix(),
		Limits: &pb.Limits{
			MaxPayout: "10000",
			MinStake:  "1",
		},
	}, nil
}

// CalculateBid calculates the Bid price (sell value) for an active contract
func (p *Pricer) CalculateBid(req model.ValuationRequest, currentSpot float64, entrySpot float64) (*model.ValuationResult, error) {
	// 1. Determine Barrier based on Entry Spot
	barrier := entrySpot
	if req.Barrier != nil {
		barrierStr := *req.Barrier
		isRelative := false
		if len(barrierStr) > 0 && (barrierStr[0] == '+' || barrierStr[0] == '-') {
			isRelative = true
		}

		b, err := strconv.ParseFloat(barrierStr, 64)
		if err == nil {
			if isRelative {
				barrier = entrySpot + b
			} else {
				barrier = b
			}
		}
	}

	// 2. Calculate Remaining Time
	duration, _ := time.ParseDuration(req.Duration)
	expiryTime := time.Unix(req.StartTime, 0).Add(duration)
	remainingDuration := time.Until(expiryTime)

	if remainingDuration <= 0 {
		return &model.ValuationResult{
			BidPrice:        "0.00",
			IsExpired:       true,
			CurrentSpot:     strconv.FormatFloat(currentSpot, 'f', 2, 64),
			CurrentSpotTime: req.PricingTime.Unix(),
			EntrySpot:       strconv.FormatFloat(entrySpot, 'f', 2, 64),
			EntrySpotTime:   req.StartTime,
			ExitSpot:        strconv.FormatFloat(currentSpot, 'f', 2, 64), // Current is exit if expired
			ExitSpotTime:    req.PricingTime.Unix(),
			Barrier:         strconv.FormatFloat(barrier, 'f', 2, 64),
			StartTime:       req.StartTime,
			ExpiryTime:      expiryTime.Unix(),
		}, nil
	}

	T := remainingDuration.Hours() / 24 / 365 // Remaining time in years

	// 3. Calculate Probability
	d2 := (math.Log(currentSpot/barrier) + (p.InterestRate-0.5*p.Volatility*p.Volatility)*T) / (p.Volatility * math.Sqrt(T))

	var probability float64
	if req.ContractType == pb.ContractType_CONTRACT_TYPE_CALL {
		probability = normalCDF(d2)
	} else {
		probability = normalCDF(-d2)
	}

	// 4. Calculate Bid Price
	// Bid Price = Payout * Probability * Discount Factor
	// We need Payout. Payout was determined at Ask time.
	// We can reconstruct Payout from Stake if we assume the original pricing parameters,
	// OR we should have Payout passed in request?
	// The PRD says "Calculate Bid Price (sell value)".
	// Usually Bid Price is what the system buys it back for.
	// If we don't have the original Payout, we might need to estimate it or it should be in the request.
	// The `ValuationRequest` has `Stake`.
	// Let's assume for this exercise that we calculate the "Fair Value" of the option now.
	// Fair Value = Payout * Probability * DF.
	// But we don't know Payout.
	// Wait, `Stake` is the premium paid.
	// If we assume the user wants to sell it back, we calculate the current premium value.
	// Current Premium = Payout * Probability * DF.
	// But we still need Payout.
	// Let's assume Payout is derived from the original Stake and original conditions?
	// Or maybe we just calculate the current theoretical price (premium) for the same terms?
	// "Bid Price = Potential Payout * Probability * Discount Factor"
	// We need Potential Payout.
	// Let's assume for V1 that we re-calculate the initial Payout based on Entry Spot and Start Time?
	// That would require historical volatility/rates.
	// ALTERNATIVELY, maybe the `Stake` in request IS the Payout? No, it says "Stake // Premium paid".
	// Let's look at `GetBidRequest` in proto. It has `OptionParameters` which has `Stake`.
	// Let's assume we need to calculate the Payout first based on Entry Spot (as if we were pricing it at start).

	// Re-calculate original Payout at Start Time
	// T_orig = duration
	T_orig := duration.Hours() / 24 / 365
	d2_orig := (math.Log(entrySpot/barrier) + (p.InterestRate-0.5*p.Volatility*p.Volatility)*T_orig) / (p.Volatility * math.Sqrt(T_orig))
	var prob_orig float64
	if req.ContractType == pb.ContractType_CONTRACT_TYPE_CALL {
		prob_orig = normalCDF(d2_orig)
	} else {
		prob_orig = normalCDF(-d2_orig)
	}
	df_orig := math.Exp(-p.InterestRate * T_orig)
	pricePerUnit_orig := df_orig * prob_orig

	stakeVal, _ := strconv.ParseFloat(req.Stake, 64)
	payout := stakeVal / pricePerUnit_orig

	// Now calculate current Bid Price
	df := math.Exp(-p.InterestRate * T)
	bidPrice := payout * probability * df

	return &model.ValuationResult{
		BidPrice:        strconv.FormatFloat(bidPrice, 'f', 2, 64),
		IsExpired:       false,
		CurrentSpot:     strconv.FormatFloat(currentSpot, 'f', 2, 64),
		CurrentSpotTime: req.PricingTime.Unix(),
		EntrySpot:       strconv.FormatFloat(entrySpot, 'f', 2, 64),
		EntrySpotTime:   req.StartTime,
		ExitSpot:        "",
		ExitSpotTime:    0,
		Barrier:         strconv.FormatFloat(barrier, 'f', 2, 64),
		StartTime:       req.StartTime,
		ExpiryTime:      expiryTime.Unix(),
	}, nil
}

func normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}
