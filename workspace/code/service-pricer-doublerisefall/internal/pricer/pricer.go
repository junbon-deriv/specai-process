// Package pricer implements the core pricing logic for Double Rise/Fall contracts.
package pricer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

// Domain errors
var (
	ErrInvalidSymbol         = errors.New("invalid symbol")
	ErrInvalidDuration       = errors.New("invalid duration format")
	ErrDurationOrder         = errors.New("duration order invalid: second duration must be greater than first")
	ErrDurationGap           = errors.New("duration gap invalid: minimum gap not met")
	ErrInvalidStake          = errors.New("stake below minimum")
	ErrPayoutExceeded        = errors.New("payout exceeds maximum")
	ErrPricingTimeFuture     = errors.New("pricing time cannot be in the future")
	ErrMissingStartTime      = errors.New("start_time required for bid requests")
	ErrMissingPayout         = errors.New("payout required for bid requests")
	ErrInvalidContractType   = errors.New("invalid contract type")
	ErrInvalidCurrency       = errors.New("invalid currency")
	ErrSymbolDisabled        = errors.New("symbol is disabled")
	ErrMissingEntryTick      = errors.New("missing entry tick")
	ErrMarketDataUnavailable = errors.New("market data unavailable")
	ErrStreamDisconnected    = errors.New("stream disconnected")
	ErrInternal              = errors.New("internal error")
)

// ContractType represents RISE or FALL.
type ContractType int

const (
	ContractTypeUnspecified ContractType = iota
	ContractTypeRise
	ContractTypeFall
)

// Duration represents a parsed duration with unit discriminator.
type Duration struct {
	Value int64
	Unit  DurationUnit
}

// DurationUnit discriminates between time and tick-based durations.
type DurationUnit string

const (
	DurationUnitSeconds DurationUnit = "s"
	DurationUnitMinutes DurationUnit = "m"
	DurationUnitHours   DurationUnit = "h"
	DurationUnitDays    DurationUnit = "d"
	DurationUnitTicks   DurationUnit = "t"
)

// ToSeconds converts a time-based duration to seconds.
func (d Duration) ToSeconds() (int64, error) {
	switch d.Unit {
	case DurationUnitSeconds:
		return d.Value, nil
	case DurationUnitMinutes:
		return d.Value * 60, nil
	case DurationUnitHours:
		return d.Value * 3600, nil
	case DurationUnitDays:
		return d.Value * 86400, nil
	case DurationUnitTicks:
		return 0, errors.New("tick duration cannot be converted to seconds")
	default:
		return 0, ErrInvalidDuration
	}
}

// SymbolConfig contains per-symbol configuration.
type SymbolConfig struct {
	Symbol     string
	Commission float64
	MaxPayout  float64
	MinStake   float64
	Enabled    bool
}

// Tick represents a market data tick.
type Tick struct {
	Symbol string
	Time   int64
	Quote  string
}

// AskRequest contains parameters for ask price calculation.
type AskRequest struct {
	Symbol         string
	ContractType   ContractType
	Currency       string
	FirstDuration  Duration
	SecondDuration Duration
	Stake          float64
	PricingTime    int64 // Optional, defaults to now
}

// AskResult contains the calculated ask price and metadata.
type AskResult struct {
	AskPrice        string
	Currency        string
	CurrentSpot     string
	CurrentSpotTime int64
	Payout          string
	MaxPayout       string
	MinStake        string
}

// BidRequest contains parameters for bid price calculation.
type BidRequest struct {
	Symbol         string
	ContractType   ContractType
	Currency       string
	FirstDuration  Duration
	SecondDuration Duration
	StartTime      int64
	Stake          float64
	Payout         float64
	PricingTime    int64
}

// BidResult contains the calculated bid price and contract state.
type BidResult struct {
	BidPrice        string
	IsExpired       bool
	CurrentSpot     string
	CurrentSpotTime int64
	EntrySpot       string
	EntrySpotTime   int64
	ExitSpot        string
	ExitSpotTime    int64
	Barrier         string
	StartTime       int64
	ExpiryTime      int64
	EvaluationTime  int64
	Currency        string
}

// ConfigProvider provides symbol configuration.
type ConfigProvider interface {
	GetSymbolConfig(symbol string) (*SymbolConfig, error)
}

// FeedProvider provides market data access.
type FeedProvider interface {
	GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error)
	GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*Tick, bool, error)
	Subscribe(ctx context.Context, symbol string, start int64) Subscription
}

// Subscription represents a real-time tick subscription.
type Subscription interface {
	C() <-chan *Tick
	Err() error
	Close()
}

// ContractValidator validates contract parameters.
type ContractValidator interface {
	ParseDuration(s string) (Duration, error)
	ValidateAskRequest(req *AskRequest, config *SymbolConfig) error
	ValidateBidRequest(req *BidRequest, config *SymbolConfig) error
}

// Pricer calculates ask and bid prices for Double Rise/Fall contracts.
type Pricer struct {
	config    ConfigProvider
	feed      FeedProvider
	validator ContractValidator
}

// NewPricer creates a new pricer with the given dependencies.
func NewPricer(config ConfigProvider, feed FeedProvider, validator ContractValidator) *Pricer {
	return &Pricer{
		config:    config,
		feed:      feed,
		validator: validator,
	}
}

// CalculateAsk computes the ask price and payout for a contract.
func (p *Pricer) CalculateAsk(ctx context.Context, req *AskRequest) (*AskResult, error) {
	// Get symbol configuration
	cfg, err := p.config.GetSymbolConfig(req.Symbol)
	if err != nil {
		return nil, err
	}

	// Validate request
	if err := p.validator.ValidateAskRequest(req, cfg); err != nil {
		return nil, err
	}

	// Determine pricing time
	pricingTime := req.PricingTime
	if pricingTime == 0 {
		pricingTime = time.Now().Unix()
	}

	// Get current spot price
	tick, _, err := p.feed.GetTickForEpoch(ctx, req.Symbol, pricingTime)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketDataUnavailable, err)
	}
	if tick == nil {
		return nil, ErrMissingEntryTick
	}

	// Calculate durations in seconds based on duration type
	var t1Seconds, t2Seconds int64

	if req.FirstDuration.Unit == DurationUnitTicks {
		// Tick-based contract: retrieve actual ticks and calculate time span
		t1Seconds, t2Seconds, err = p.calculateTickBasedDurations(ctx, req.Symbol, pricingTime, req.FirstDuration, req.SecondDuration)
		if err != nil {
			return nil, err
		}
	} else {
		// Time-based contract: convert directly to seconds
		t1Seconds, err = req.FirstDuration.ToSeconds()
		if err != nil {
			return nil, err
		}
		t2Seconds, err = req.SecondDuration.ToSeconds()
		if err != nil {
			return nil, err
		}
	}

	// Calculate fair probability
	pFair := calculateFairProbability(t1Seconds, t2Seconds)

	// Apply commission
	pClient := applyCommission(pFair, cfg.Commission)

	// Calculate payout
	payout := calculatePayout(req.Stake, pClient)

	// Validate payout doesn't exceed maximum
	if payout > cfg.MaxPayout {
		return nil, ErrPayoutExceeded
	}

	return &AskResult{
		AskPrice:        formatPrice(pClient, 4),
		Currency:        req.Currency,
		CurrentSpot:     tick.Quote,
		CurrentSpotTime: tick.Time,
		Payout:          formatPrice(payout, 2),
		MaxPayout:       formatPrice(cfg.MaxPayout, 2),
		MinStake:        formatPrice(cfg.MinStake, 2),
	}, nil
}

// calculateTickBasedDurations retrieves ticks and calculates actual time durations for tick-based contracts.
func (p *Pricer) calculateTickBasedDurations(ctx context.Context, symbol string, startTime int64, firstDuration, secondDuration Duration) (int64, int64, error) {
	// Get ticks for first duration (t1)
	ticksT1, _, err := p.feed.GetTicksFromLimit(ctx, symbol, startTime, firstDuration.Value)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: failed to get ticks for t1", ErrMarketDataUnavailable)
	}
	if int64(len(ticksT1)) < firstDuration.Value {
		return 0, 0, fmt.Errorf("%w: insufficient ticks for t1", ErrMissingEntryTick)
	}

	// Get ticks for second duration (t2)
	ticksT2, _, err := p.feed.GetTicksFromLimit(ctx, symbol, startTime, secondDuration.Value)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: failed to get ticks for t2", ErrMarketDataUnavailable)
	}
	if int64(len(ticksT2)) < secondDuration.Value {
		return 0, 0, fmt.Errorf("%w: insufficient ticks for t2", ErrMissingEntryTick)
	}

	// Calculate actual time spans from tick timestamps
	// t1 time = time from start tick to Nth tick
	t1Seconds := ticksT1[len(ticksT1)-1].Time - startTime

	// t2 time = time from start tick to Mth tick
	t2Seconds := ticksT2[len(ticksT2)-1].Time - startTime

	return t1Seconds, t2Seconds, nil
}

// CalculateBid computes the bid price for an active contract.
func (p *Pricer) CalculateBid(ctx context.Context, req *BidRequest) (*BidResult, error) {
	// Get symbol configuration
	cfg, err := p.config.GetSymbolConfig(req.Symbol)
	if err != nil {
		return nil, err
	}

	// Validate request
	if err := p.validator.ValidateBidRequest(req, cfg); err != nil {
		return nil, err
	}

	// Determine pricing time
	pricingTime := req.PricingTime
	if pricingTime == 0 {
		pricingTime = time.Now().Unix()
	}

	// Get entry tick (barrier)
	entryTick, _, err := p.feed.GetTickForEpoch(ctx, req.Symbol, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketDataUnavailable, err)
	}
	if entryTick == nil {
		return nil, ErrMissingEntryTick
	}

	barrier, err := strconv.ParseFloat(entryTick.Quote, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid entry spot", ErrInternal)
	}

	// Calculate evaluation and expiry times based on duration type
	var evaluationTime, expiryTime int64

	if req.FirstDuration.Unit == DurationUnitTicks {
		// Tick-based contract: retrieve actual ticks to determine times
		ticksT1, _, err := p.feed.GetTicksFromLimit(ctx, req.Symbol, req.StartTime, req.FirstDuration.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to get ticks for t1", ErrMarketDataUnavailable)
		}
		if int64(len(ticksT1)) < req.FirstDuration.Value {
			return nil, fmt.Errorf("%w: insufficient ticks for t1", ErrMissingEntryTick)
		}
		evaluationTime = ticksT1[len(ticksT1)-1].Time

		ticksT2, _, err := p.feed.GetTicksFromLimit(ctx, req.Symbol, req.StartTime, req.SecondDuration.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to get ticks for t2", ErrMarketDataUnavailable)
		}
		if int64(len(ticksT2)) < req.SecondDuration.Value {
			return nil, fmt.Errorf("%w: insufficient ticks for t2", ErrMissingEntryTick)
		}
		expiryTime = ticksT2[len(ticksT2)-1].Time
	} else {
		// Time-based contract: calculate from durations
		t1Seconds, err := req.FirstDuration.ToSeconds()
		if err != nil {
			return nil, err
		}
		t2Seconds, err := req.SecondDuration.ToSeconds()
		if err != nil {
			return nil, err
		}
		evaluationTime = req.StartTime + t1Seconds
		expiryTime = req.StartTime + t2Seconds
	}

	// Get current spot
	currentTick, _, err := p.feed.GetTickForEpoch(ctx, req.Symbol, pricingTime)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketDataUnavailable, err)
	}
	if currentTick == nil {
		return nil, ErrMissingEntryTick
	}

	result := &BidResult{
		Currency:        req.Currency,
		EntrySpot:       entryTick.Quote,
		EntrySpotTime:   entryTick.Time,
		Barrier:         entryTick.Quote,
		StartTime:       req.StartTime,
		ExpiryTime:      expiryTime,
		EvaluationTime:  evaluationTime,
		CurrentSpot:     currentTick.Quote,
		CurrentSpotTime: currentTick.Time,
	}

	// Check if contract has expired
	if pricingTime >= expiryTime {
		// Contract expired - evaluate win/loss
		result.IsExpired = true

		// Get t1 spot
		t1Tick, _, err := p.feed.GetTickForEpoch(ctx, req.Symbol, evaluationTime)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMarketDataUnavailable, err)
		}
		if t1Tick == nil {
			return nil, ErrMissingEntryTick
		}

		spotT1, err := strconv.ParseFloat(t1Tick.Quote, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid t1 spot", ErrInternal)
		}

		// Get t2 spot (exit spot)
		t2Tick, _, err := p.feed.GetTickForEpoch(ctx, req.Symbol, expiryTime)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMarketDataUnavailable, err)
		}
		if t2Tick == nil {
			return nil, ErrMissingEntryTick
		}

		result.ExitSpot = t2Tick.Quote
		result.ExitSpotTime = t2Tick.Time

		spotT2, err := strconv.ParseFloat(t2Tick.Quote, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid t2 spot", ErrInternal)
		}

		// Evaluate contract
		won := evaluateContract(req.ContractType, barrier, spotT1, spotT2)
		if won {
			result.BidPrice = formatPrice(req.Payout, 2)
		} else {
			result.BidPrice = "0.00"
		}
	} else {
		// Contract still active - calculate value based on current state
		result.IsExpired = false
		result.ExitSpot = ""
		result.ExitSpotTime = 0

		// Calculate bid price based on contract state
		bidPrice, err := p.calculateActiveBidPrice(ctx, req, barrier, evaluationTime, expiryTime, pricingTime, currentTick)
		if err != nil {
			return nil, err
		}
		result.BidPrice = formatPrice(bidPrice, 2)
	}

	return result, nil
}

// calculateActiveBidPrice calculates the bid price for an active contract based on its current state.
func (p *Pricer) calculateActiveBidPrice(ctx context.Context, req *BidRequest, barrier float64, evaluationTime, expiryTime, pricingTime int64, currentTick *Tick) (float64, error) {
	currentSpot, err := strconv.ParseFloat(currentTick.Quote, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid current spot", ErrInternal)
	}

	// Check if we're before or after t1 (evaluation time)
	if pricingTime < evaluationTime {
		// Before t1: Contract value is based on current position and probability
		// Simplified approach: return current spot as proxy for contract value
		// This could be enhanced with proper probability-based valuation
		return currentSpot, nil
	}

	// After t1, before t2: Check t1 evaluation result
	t1Tick, _, err := p.feed.GetTickForEpoch(ctx, req.Symbol, evaluationTime)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrMarketDataUnavailable, err)
	}
	if t1Tick == nil {
		return 0, ErrMissingEntryTick
	}

	spotT1, err := strconv.ParseFloat(t1Tick.Quote, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid t1 spot", ErrInternal)
	}

	// Evaluate t1 condition
	var t1ConditionMet bool
	switch req.ContractType {
	case ContractTypeRise:
		t1ConditionMet = spotT1 > barrier
	case ContractTypeFall:
		t1ConditionMet = spotT1 < barrier
	default:
		return 0, ErrInvalidContractType
	}

	if !t1ConditionMet {
		// First condition failed - contract will lose, value approaches 0
		// Return a small residual value based on remaining time
		timeToExpiry := float64(expiryTime - pricingTime)
		totalTime := float64(expiryTime - req.StartTime)
		residualValue := req.Payout * 0.01 * (timeToExpiry / totalTime) // 1% residual * time factor
		if residualValue < 0.01 {
			return 0.01, nil // Minimum bid
		}
		return residualValue, nil
	}

	// First condition met, still need to meet second condition
	// Contract has value between stake and payout based on current position and remaining time
	var currentConditionMet bool
	switch req.ContractType {
	case ContractTypeRise:
		currentConditionMet = currentSpot > barrier
	case ContractTypeFall:
		currentConditionMet = currentSpot < barrier
	default:
		return 0, ErrInvalidContractType
	}

	if currentConditionMet {
		// Both conditions currently met - higher value
		// Value approaches payout as we get closer to expiry
		timeToExpiry := float64(expiryTime - pricingTime)
		totalTime := float64(expiryTime - req.StartTime)
		timeDecayFactor := 1.0 - (timeToExpiry / totalTime) // 0 at start, 1 at expiry

		// Value ranges from stake to payout as we approach expiry
		bidValue := req.Stake + (req.Payout-req.Stake)*timeDecayFactor*0.8 // 80% confidence factor
		return bidValue, nil
	}

	// First condition met but second currently not met
	// Contract has uncertain value - return stake value as it's still viable
	return req.Stake, nil
}

// StreamAsk provides continuous ask price updates.
func (p *Pricer) StreamAsk(ctx context.Context, req *AskRequest) (<-chan *AskResult, <-chan error) {
	resultCh := make(chan *AskResult)
	errCh := make(chan error, 1)

	go func() {
		defer close(resultCh)
		defer close(errCh)

		// Subscribe to tick updates
		sub := p.feed.Subscribe(ctx, req.Symbol, time.Now().Unix())
		defer sub.Close()

		// Send initial price
		result, err := p.CalculateAsk(ctx, req)
		if err != nil {
			errCh <- err
			return
		}
		select {
		case resultCh <- result:
		case <-ctx.Done():
			return
		}

		// Stream updates
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case tick := <-sub.C():
				if tick != nil {
					result, err := p.CalculateAsk(ctx, req)
					if err != nil {
						errCh <- err
						return
					}
					select {
					case resultCh <- result:
					case <-ctx.Done():
						return
					}
				}
			case <-ticker.C:
				result, err := p.CalculateAsk(ctx, req)
				if err != nil {
					errCh <- err
					return
				}
				select {
				case resultCh <- result:
				case <-ctx.Done():
					return
				}
			}

			if sub.Err() != nil {
				errCh <- fmt.Errorf("%w: %v", ErrStreamDisconnected, sub.Err())
				return
			}
		}
	}()

	return resultCh, errCh
}

// StreamBid provides continuous bid price updates.
func (p *Pricer) StreamBid(ctx context.Context, req *BidRequest) (<-chan *BidResult, <-chan error) {
	resultCh := make(chan *BidResult)
	errCh := make(chan error, 1)

	go func() {
		defer close(resultCh)
		defer close(errCh)

		// Subscribe to tick updates
		sub := p.feed.Subscribe(ctx, req.Symbol, req.StartTime)
		defer sub.Close()

		// Send initial bid
		result, err := p.CalculateBid(ctx, req)
		if err != nil {
			errCh <- err
			return
		}
		select {
		case resultCh <- result:
		case <-ctx.Done():
			return
		}

		// If already expired, stop streaming
		if result.IsExpired {
			return
		}

		// Stream updates
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case tick := <-sub.C():
				if tick != nil {
					result, err := p.CalculateBid(ctx, req)
					if err != nil {
						errCh <- err
						return
					}
					select {
					case resultCh <- result:
					case <-ctx.Done():
						return
					}
					// Stop streaming after expiry
					if result.IsExpired {
						return
					}
				}
			case <-ticker.C:
				result, err := p.CalculateBid(ctx, req)
				if err != nil {
					errCh <- err
					return
				}
				select {
				case resultCh <- result:
				case <-ctx.Done():
					return
				}
				// Stop streaming after expiry
				if result.IsExpired {
					return
				}
			}

			if sub.Err() != nil {
				errCh <- fmt.Errorf("%w: %v", ErrStreamDisconnected, sub.Err())
				return
			}
		}
	}()

	return resultCh, errCh
}

// calculateFairProbability computes P_fair using bivariate normal distribution.
// Formula: P_fair = 1/4 + arcsin(sqrt(t1/t2)) / (2π)
func calculateFairProbability(t1Seconds, t2Seconds int64) float64 {
	// Calculate correlation
	rho := math.Sqrt(float64(t1Seconds) / float64(t2Seconds))

	// Calculate fair probability
	pFair := 0.25 + math.Asin(rho)/(2*math.Pi)

	return pFair
}

// applyCommission adds commission to fair probability.
// Returns client price (P_client = P_fair + commission)
func applyCommission(pFair, commission float64) float64 {
	return pFair + commission
}

// calculatePayout computes payout from stake and client price.
// Formula: Payout = Stake / P_client
func calculatePayout(stake, pClient float64) float64 {
	if pClient <= 0 || pClient > 1 {
		return 0
	}
	return stake / pClient
}

// evaluateContract determines win/loss at expiry.
func evaluateContract(contractType ContractType, barrier, spotT1, spotT2 float64) bool {
	switch contractType {
	case ContractTypeRise:
		// Win if spot > barrier at BOTH t1 AND t2
		return spotT1 > barrier && spotT2 > barrier
	case ContractTypeFall:
		// Win if spot < barrier at BOTH t1 AND t2
		return spotT1 < barrier && spotT2 < barrier
	default:
		return false
	}
}

// formatPrice formats a price to the specified number of decimal places.
func formatPrice(value float64, decimals int) string {
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, value)
}
