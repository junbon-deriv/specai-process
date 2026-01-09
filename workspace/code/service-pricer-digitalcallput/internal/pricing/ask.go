package pricing

import (
	"context"
	"fmt"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/contract"
	"github.com/shopspring/decimal"
)

// CalculateAsk implements Ask price calculation
func (p *pricer) CalculateAsk(ctx context.Context, input AskInput) (*AskResult, error) {
	// 1. Get current spot
	tick, err := p.market.GetLatestTick(ctx, input.Symbol)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketUnavailable, err)
	}

	// 2. Get symbol config
	symbolConfig, err := p.config.GetSymbolConfig(input.Symbol)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnknownSymbol, err)
	}

	// 3. Parse and validate stake
	stake, err := decimal.NewFromString(input.Stake)
	if err != nil {
		return nil, fmt.Errorf("invalid stake format: %w", err)
	}
	if stake.LessThan(symbolConfig.MinStake) {
		return nil, ErrStakeBelowMinimum
	}

	// 4. Parse duration using contract module
	duration, err := contract.ParseDuration(input.Duration)
	if err != nil {
		return nil, err
	}

	// 5. Resolve barrier using contract module (using current spot as entry price for Ask)
	barrierResult, err := contract.ResolveBarrier(input.Barrier, tick.Price)
	if err != nil {
		return nil, err
	}
	barrier := barrierResult.ResolvedValue

	// 6. Calculate probability using Black-Scholes
	var probability decimal.Decimal
	durationYears := duration.ToYears()

	if input.ContractType == ContractTypeCall {
		probability = p.blackScholes.CalculateForCall(tick.Price, barrier, durationYears)
	} else {
		probability = p.blackScholes.CalculateForPut(tick.Price, barrier, durationYears)
	}

	// Ensure probability is not zero to avoid division by zero
	if probability.IsZero() {
		probability = decimal.NewFromFloat(0.0001)
	}

	// 7. Calculate payout (stake / probability)
	payout := stake.Div(probability)

	// 8. Check payout limit
	if payout.GreaterThan(symbolConfig.MaxPayout) {
		return nil, ErrPayoutExceedsMaximum
	}

	// 9. Calculate ask price (stake plus commission)
	// Note: The ask price is what the buyer pays, which includes the commission
	// The stake is what goes into the contract
	commissionAmount := stake.Mul(symbolConfig.Commission)
	askPrice := stake.Add(commissionAmount)

	return &AskResult{
		AskPrice:        askPrice.StringFixed(8),
		Payout:          payout.StringFixed(8),
		CurrentSpot:     tick.Price.StringFixed(8),
		CurrentSpotTime: tick.Timestamp,
		Currency:        input.Currency,
		Limits: Limits{
			MaxPayout: symbolConfig.MaxPayout,
			MinStake:  symbolConfig.MinStake,
		},
	}, nil
}
