package pricing

import (
	"context"
	"fmt"
	"time"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/contract"
	"github.com/shopspring/decimal"
)

// CalculateBid implements Bid price calculation
// CRITICAL: payout comes from input, never recalculated
func (p *pricer) CalculateBid(ctx context.Context, input BidInput) (*BidResult, error) {
	// 1. Get symbol config (for validation)
	_, err := p.config.GetSymbolConfig(input.Symbol)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnknownSymbol, err)
	}

	// 2. Parse payout (REQUIRED parameter)
	payout, err := decimal.NewFromString(input.Payout)
	if err != nil {
		return nil, fmt.Errorf("invalid payout format: %w", err)
	}

	// 3. Get entry tick (first tick after start_time)
	entryTick, err := p.market.GetTick(ctx, input.Symbol, input.StartTime)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get entry tick: %v", ErrMarketUnavailable, err)
	}

	// 4. Resolve barrier using entry price (contract module)
	barrierResult, err := contract.ResolveBarrier(input.Barrier, entryTick.Price)
	if err != nil {
		return nil, err
	}
	barrier := barrierResult.ResolvedValue

	// 5. Parse duration and calculate expiry (contract module)
	duration, err := contract.ParseDuration(input.Duration)
	if err != nil {
		return nil, err
	}

	var expiryTime int64
	var isTickBased bool
	if duration.IsTime() {
		expiryTime = duration.CalculateExpiry(input.StartTime)
		isTickBased = false
	} else {
		// For tick-based contracts, expiry is determined by tick count, not time
		// We'll need to count ticks from entry to determine expiry
		isTickBased = true
		expiryTime = 0 // No time-based expiry for tick-based contracts
	}

	// 6. Get current spot
	currentTick, err := p.market.GetLatestTick(ctx, input.Symbol)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get current spot: %v", ErrMarketUnavailable, err)
	}

	// 7. Determine if expired and calculate bid
	var bidPrice decimal.Decimal
	var exitSpot *string
	var exitSpotTime *int64
	var isExpired bool

	if isTickBased {
		// For tick-based contracts: no early exit allowed (REQ-PR-N3L)
		// Bid price for active tick-based contracts is always 0
		// We need to count actual ticks to determine expiry, which requires historical data
		// For now, check if we can get exit tick by counting from entry
		tickCount := 0
		requiredTicks := duration.Value

		// Try to count ticks from entry time
		// This is a simplified approach - in streaming, tick counting happens differently
		ticks, err := p.market.GetTicksInRange(ctx, input.Symbol, input.StartTime, currentTick.Timestamp)
		if err == nil && ticks != nil {
			tickCount = len(ticks)
		}

		isExpired = tickCount >= requiredTicks

		if isExpired {
			// Contract has expired - determine win/loss
			// Exit tick is the Nth tick after entry
			var exitTick *Tick
			if tickCount >= requiredTicks && ticks != nil {
				exitTick = ticks[requiredTicks-1] // Use the Nth tick as exit
			} else {
				exitTick = currentTick // Fallback
			}

			exitPrice := exitTick.Price.StringFixed(8)
			exitTime := exitTick.Timestamp
			exitSpot = &exitPrice
			exitSpotTime = &exitTime
			expiryTime = exitTime

			// Determine win/loss
			if isWin(input.ContractType, exitTick.Price, barrier) {
				bidPrice = payout
			} else {
				bidPrice = decimal.Zero
			}
		} else {
			// Active tick-based contract: no early exit (REQ-PR-N3L)
			bidPrice = decimal.Zero
			expiryTime = 0 // Unknown until all ticks counted
		}
	} else {
		// Time-based contract logic
		now := time.Now().Unix()
		isExpired = now >= expiryTime

		if isExpired {
			// Get exit tick at expiry time
			exitTick, err := p.market.GetTick(ctx, input.Symbol, expiryTime)
			if err != nil {
				// If we can't get the exact exit tick, use current tick as fallback
				exitTick = currentTick
			}

			exitPrice := exitTick.Price.StringFixed(8)
			exitTime := exitTick.Timestamp
			exitSpot = &exitPrice
			exitSpotTime = &exitTime

			// Determine win/loss
			if isWin(input.ContractType, exitTick.Price, barrier) {
				bidPrice = payout
			} else {
				bidPrice = decimal.Zero
			}
		} else {
			// Calculate bid using Black-Scholes with remaining time
			remaining := float64(expiryTime - now)
			remainingYears := remaining / (365.25 * 24 * 60 * 60)

			var probability decimal.Decimal
			if input.ContractType == ContractTypeCall {
				probability = p.blackScholes.CalculateForCall(currentTick.Price, barrier, remainingYears)
			} else {
				probability = p.blackScholes.CalculateForPut(currentTick.Price, barrier, remainingYears)
			}

			bidPrice = payout.Mul(probability)
		}
	}

	return &BidResult{
		BidPrice:        bidPrice.StringFixed(8),
		IsExpired:       isExpired,
		CurrentSpot:     currentTick.Price.StringFixed(8),
		CurrentSpotTime: currentTick.Timestamp,
		EntrySpot:       entryTick.Price.StringFixed(8),
		EntrySpotTime:   entryTick.Timestamp,
		ExitSpot:        exitSpot,
		ExitSpotTime:    exitSpotTime,
		Barrier:         barrier.StringFixed(8),
		StartTime:       input.StartTime,
		ExpiryTime:      expiryTime,
		Currency:        input.Currency,
	}, nil
}

// isWin determines if a contract is a winning contract based on type and exit price
// Call: wins if exit > barrier (strictly greater)
// Put: wins if exit < barrier (strictly less than)
func isWin(contractType ContractType, exitPrice, barrier decimal.Decimal) bool {
	if contractType == ContractTypeCall {
		return exitPrice.GreaterThan(barrier)
	}
	// Put
	return exitPrice.LessThan(barrier)
}
