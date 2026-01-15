package pricer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

var (
	// ErrInvalidSymbol indicates the symbol is not supported
	ErrInvalidSymbol = errors.New("invalid symbol")
	// ErrSymbolDisabled indicates the symbol is disabled
	ErrSymbolDisabled = errors.New("symbol disabled")
	// ErrInvalidDuration indicates duration format is invalid
	ErrInvalidDuration = errors.New("invalid duration format")
	// ErrInvalidStake indicates stake is invalid
	ErrInvalidStake = errors.New("invalid stake")
	// ErrPayoutExceeded indicates payout exceeds maximum
	ErrPayoutExceeded = errors.New("payout exceeds maximum")
	// ErrMissingStartTime indicates start_time is required but missing
	ErrMissingStartTime = errors.New("missing start_time")
	// ErrMissingPayout indicates payout is required but missing
	ErrMissingPayout = errors.New("missing payout")
	// ErrMissingEntryTick indicates entry tick not found
	ErrMissingEntryTick = errors.New("missing entry tick")
	// ErrMarketDataUnavailable indicates market data service is unavailable
	ErrMarketDataUnavailable = errors.New("market data unavailable")
)

// Tick represents a market data point.
type Tick struct {
	Symbol string
	Time   int64  // Unix epoch seconds
	Quote  string // Price as string to preserve precision
}

// SymbolConfig contains symbol-specific configuration.
type SymbolConfig struct {
	Symbol         string
	CommissionRate float64
	MaxPayout      float64
	MinStake       float64
	Enabled        bool
}

// Duration represents a parsed duration.
type Duration struct {
	Value       int64
	Unit        string // "s", "m", "h", "d", "t"
	IsTickBased bool
}

// ToSeconds converts time-based duration to seconds.
func (d Duration) ToSeconds() int64 {
	if d.IsTickBased {
		return 0
	}
	switch d.Unit {
	case "s":
		return d.Value
	case "m":
		return d.Value * 60
	case "h":
		return d.Value * 3600
	case "d":
		return d.Value * 86400
	default:
		return 0
	}
}

// ToTicks returns tick count for tick-based durations.
func (d Duration) ToTicks() int64 {
	if d.IsTickBased {
		return d.Value
	}
	return 0
}

// ContractType represents the contract direction.
type ContractType int

const (
	ContractTypeUnspecified ContractType = iota
	ContractTypeRise
	ContractTypeFall
)

// AskRequest contains parameters for calculating ask price.
type AskRequest struct {
	Symbol         string
	ContractType   ContractType
	Currency       string
	FirstDuration  string
	SecondDuration string
	Stake          string
	PricingTime    int64 // 0 means use current time
}

// AskResult contains the calculated ask price and details.
type AskResult struct {
	AskPrice        string
	Currency        string
	CurrentSpot     string
	CurrentSpotTime int64
	Payout          string
	MaxPayout       string
	MinStake        string
}

// BidRequest contains parameters for evaluating bid price.
type BidRequest struct {
	Symbol         string
	ContractType   ContractType
	Currency       string
	FirstDuration  string
	SecondDuration string
	StartTime      int64
	Stake          string
	Payout         string
	PricingTime    int64 // 0 means use current time
}

// BidResult contains the evaluated bid price and details.
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
	Currency        string
	EvaluationTime  int64
}

// Subscription represents a tick subscription.
type Subscription struct {
	C   <-chan *Tick
	Err error
}

// AskSubscription provides a stream of ask price updates.
type AskSubscription struct {
	C      <-chan *AskResult
	Err    error
	cancel context.CancelFunc
}

// Close closes the subscription and releases resources.
func (s *AskSubscription) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

// BidSubscription provides a stream of bid price updates.
type BidSubscription struct {
	C      <-chan *BidResult
	Err    error
	cancel context.CancelFunc
}

// Close closes the subscription and releases resources.
func (s *BidSubscription) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

// ConfigProvider provides access to symbol configuration.
type ConfigProvider interface {
	GetSymbolConfig(symbol string) (*SymbolConfig, error)
}

// FeedProvider provides access to market data.
type FeedProvider interface {
	GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error)
	GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*Tick, bool, error)
	Subscribe(ctx context.Context, symbol string, start int64) *Subscription
	Close() error
}

// ContractValidator validates contract parameters.
type ContractValidator interface {
	ValidateAskRequest(ctx context.Context, req *AskRequest) error
	ValidateBidRequest(ctx context.Context, req *BidRequest) error
	ParseDuration(s string) (Duration, error)
}

// Pricer calculates ask and bid prices for Double Rise/Fall contracts.
type Pricer struct {
	config   ConfigProvider
	feed     FeedProvider
	contract ContractValidator
}

// New creates a new Pricer.
func New(config ConfigProvider, feed FeedProvider, contract ContractValidator) *Pricer {
	return &Pricer{
		config:   config,
		feed:     feed,
		contract: contract,
	}
}

// CalculateAsk computes the ask price (payout) for a contract.
func (p *Pricer) CalculateAsk(ctx context.Context, req *AskRequest) (*AskResult, error) {
	// Validate request
	if err := p.contract.ValidateAskRequest(ctx, req); err != nil {
		return nil, err
	}

	// Get symbol configuration (GetSymbolConfig already checks if enabled)
	cfg, err := p.config.GetSymbolConfig(req.Symbol)
	if err != nil {
		return nil, err
	}

	// Parse durations
	d1, err := p.contract.ParseDuration(req.FirstDuration)
	if err != nil {
		return nil, fmt.Errorf("first_duration: %w", err)
	}

	d2, err := p.contract.ParseDuration(req.SecondDuration)
	if err != nil {
		return nil, fmt.Errorf("second_duration: %w", err)
	}

	// Get pricing time
	pricingTime := req.PricingTime
	if pricingTime == 0 {
		pricingTime = getCurrentTime()
	}

	// Get current spot price
	tick, _, err := p.feed.GetTickForEpoch(ctx, req.Symbol, pricingTime)
	if err != nil {
		return nil, fmt.Errorf("get current tick: %w", ErrMarketDataUnavailable)
	}
	if tick == nil {
		return nil, ErrMarketDataUnavailable
	}

	// Calculate fair probability using arcsin correlation formula
	fairProb := p.calculateFairProbability(d1, d2)

	// Apply commission markup
	clientPrice := fairProb + cfg.CommissionRate

	// Parse stake
	stake, err := strconv.ParseFloat(req.Stake, 64)
	if err != nil {
		return nil, ErrInvalidStake
	}

	// Calculate payout
	payout := stake / clientPrice

	// Check payout limits
	if payout > cfg.MaxPayout {
		return nil, ErrPayoutExceeded
	}

	return &AskResult{
		AskPrice:        req.Stake, // Premium equals stake
		Currency:        req.Currency,
		CurrentSpot:     tick.Quote,
		CurrentSpotTime: tick.Time,
		Payout:          fmt.Sprintf("%.2f", payout),
		MaxPayout:       fmt.Sprintf("%.2f", cfg.MaxPayout),
		MinStake:        fmt.Sprintf("%.2f", cfg.MinStake),
	}, nil
}

// CalculateBid evaluates the bid price for an active contract.
func (p *Pricer) CalculateBid(ctx context.Context, req *BidRequest) (*BidResult, error) {
	// Validate request
	if err := p.contract.ValidateBidRequest(ctx, req); err != nil {
		return nil, err
	}

	// Get symbol configuration (GetSymbolConfig already checks if enabled)
	_, err := p.config.GetSymbolConfig(req.Symbol)
	if err != nil {
		return nil, err
	}

	// Parse durations
	d1, err := p.contract.ParseDuration(req.FirstDuration)
	if err != nil {
		return nil, fmt.Errorf("first_duration: %w", err)
	}

	d2, err := p.contract.ParseDuration(req.SecondDuration)
	if err != nil {
		return nil, fmt.Errorf("second_duration: %w", err)
	}

	// Get entry tick (barrier) - first tick AFTER start_time
	// Using start_time + 1 to ensure we get a tick after, not at start_time
	entryTicks, _, err := p.feed.GetTicksFromLimit(ctx, req.Symbol, req.StartTime+1, 1)
	if err != nil {
		return nil, fmt.Errorf("get entry tick: %w", ErrMarketDataUnavailable)
	}
	if len(entryTicks) == 0 {
		return nil, ErrMissingEntryTick
	}
	entryTick := entryTicks[0]

	barrier, err := strconv.ParseFloat(entryTick.Quote, 64)
	if err != nil {
		return nil, fmt.Errorf("parse barrier: %w", err)
	}

	// Get pricing time
	pricingTime := req.PricingTime
	if pricingTime == 0 {
		pricingTime = getCurrentTime()
	}

	// Determine evaluation time calculations based on duration type
	var t1, t2 int64

	if d1.IsTickBased {
		// For tick-based, we'll calculate times from the actual ticks
		// Fetch ticks first to determine evaluation times
		limit := d2.ToTicks() + 1 // +1 because entry tick is tick 0
		ticks, _, err := p.feed.GetTicksFromLimit(ctx, req.Symbol, req.StartTime+1, limit)
		if err != nil {
			return nil, fmt.Errorf("get ticks: %w", ErrMarketDataUnavailable)
		}
		if int64(len(ticks)) < limit {
			return nil, fmt.Errorf("insufficient ticks available: %w", ErrMarketDataUnavailable)
		}

		// ticks[0] is entry tick (already have this)
		// ticks[d1.Value] is first evaluation
		// ticks[d2.Value] is second evaluation
		spot1 := ticks[d1.ToTicks()]
		t1 = spot1.Time
		spot2 := ticks[d2.ToTicks()]
		t2 = spot2.Time

		// Evaluate bid with all ticks available
		return p.evaluateBidWithTicks(ctx, req, entryTick, spot1, spot2, barrier, t1, t2, pricingTime)
	}

	// For time-based contracts, calculate times and fetch ticks as needed
	t1 = req.StartTime + d1.ToSeconds()
	t2 = req.StartTime + d2.ToSeconds()

	// Get spot at t1
	spot1, _, err := p.feed.GetTickForEpoch(ctx, req.Symbol, t1)
	if err != nil || spot1 == nil {
		return nil, ErrMarketDataUnavailable
	}

	// Evaluate at t1 first to determine if we need t2
	price1, err := strconv.ParseFloat(spot1.Quote, 64)
	if err != nil {
		return nil, fmt.Errorf("parse spot1: %w", err)
	}

	t1Pass := false
	if req.ContractType == ContractTypeRise {
		t1Pass = price1 > barrier
	} else {
		t1Pass = price1 < barrier
	}

	var spot2 *Tick
	// Only fetch spot2 if t1 passed and contract has expired
	if t1Pass && pricingTime >= t2 {
		spot2, _, err = p.feed.GetTickForEpoch(ctx, req.Symbol, t2)
		if err != nil || spot2 == nil {
			return nil, ErrMarketDataUnavailable
		}
	}

	return p.evaluateBidWithTicks(ctx, req, entryTick, spot1, spot2, barrier, t1, t2, pricingTime)
}

// evaluateBidWithTicks evaluates bid price given all the tick data.
func (p *Pricer) evaluateBidWithTicks(
	ctx context.Context,
	req *BidRequest,
	entryTick, spot1, spot2 *Tick,
	barrier float64,
	t1, t2, pricingTime int64,
) (*BidResult, error) {
	var bidPrice string
	var isExpired bool
	var exitSpot string
	var exitSpotTime int64

	if pricingTime < t1 {
		// Before first evaluation, bid is 0, not expired
		bidPrice = "0.00"
		isExpired = false
	} else {
		// Evaluate at t1
		price1, err := strconv.ParseFloat(spot1.Quote, 64)
		if err != nil {
			return nil, fmt.Errorf("parse spot1: %w", err)
		}

		t1Pass := false
		if req.ContractType == ContractTypeRise {
			t1Pass = price1 > barrier
		} else {
			t1Pass = price1 < barrier
		}

		if !t1Pass {
			// Contract fails at t1 - expires immediately
			bidPrice = "0.00"
			isExpired = true
			exitSpot = spot1.Quote
			exitSpotTime = spot1.Time
		} else if pricingTime >= t2 {
			// Contract reached t2, t1 passed, evaluate final result
			if spot2 == nil {
				return nil, fmt.Errorf("missing spot2 for expired contract")
			}

			price2, err := strconv.ParseFloat(spot2.Quote, 64)
			if err != nil {
				return nil, fmt.Errorf("parse spot2: %w", err)
			}

			t2Pass := false
			if req.ContractType == ContractTypeRise {
				t2Pass = price2 > barrier
			} else {
				t2Pass = price2 < barrier
			}

			if t2Pass {
				bidPrice = req.Payout
			} else {
				bidPrice = "0.00"
			}
			isExpired = true
			exitSpot = spot2.Quote
			exitSpotTime = spot2.Time
		} else {
			// Between t1 and t2, t1 passed but not yet expired
			bidPrice = "0.00"
			isExpired = false
		}
	}

	// Get current tick for response
	currentTick, _, _ := p.feed.GetTickForEpoch(ctx, req.Symbol, pricingTime)
	if currentTick == nil {
		currentTick = entryTick
	}

	return &BidResult{
		BidPrice:        bidPrice,
		IsExpired:       isExpired,
		CurrentSpot:     currentTick.Quote,
		CurrentSpotTime: currentTick.Time,
		EntrySpot:       entryTick.Quote,
		EntrySpotTime:   entryTick.Time,
		ExitSpot:        exitSpot,
		ExitSpotTime:    exitSpotTime,
		Barrier:         entryTick.Quote,
		StartTime:       req.StartTime,
		ExpiryTime:      t2,
		Currency:        req.Currency,
		EvaluationTime:  t1,
	}, nil
}

// calculateFairProbability computes fair probability using arcsin correlation formula.
// Formula: P_fair = 1/4 + arcsin(ρ)/(2π)
// Where: ρ = √(t₁/t₂)
func (p *Pricer) calculateFairProbability(d1, d2 Duration) float64 {
	var t1, t2 float64

	if d1.IsTickBased {
		t1 = float64(d1.Value)
		t2 = float64(d2.Value)
	} else {
		t1 = float64(d1.ToSeconds())
		t2 = float64(d2.ToSeconds())
	}

	// Calculate correlation: ρ = √(t₁/t₂)
	rho := math.Sqrt(t1 / t2)

	// Calculate fair probability: P_fair = 1/4 + arcsin(ρ)/(2π)
	fairProb := 0.25 + math.Asin(rho)/(2*math.Pi)

	return fairProb
}

// SubscribeAsk creates a subscription that emits ask price updates.
// For tick-based contracts: emits on each tick.
// For time-based contracts: emits on each tick OR every 5 seconds.
func (p *Pricer) SubscribeAsk(ctx context.Context, req *AskRequest) (*AskSubscription, error) {
	// Validate request
	if err := p.contract.ValidateAskRequest(ctx, req); err != nil {
		return nil, err
	}

	// Parse durations to determine if tick-based
	d1, err := p.contract.ParseDuration(req.FirstDuration)
	if err != nil {
		return nil, fmt.Errorf("first_duration: %w", err)
	}

	// Create context for subscription lifecycle
	subCtx, cancel := context.WithCancel(ctx)

	// Create channel for ask results
	askChan := make(chan *AskResult, 10)

	sub := &AskSubscription{
		C:      askChan,
		cancel: cancel,
	}

	// Start goroutine to handle streaming
	go func() {
		defer close(askChan)
		defer cancel()

		// Send initial response
		result, err := p.CalculateAsk(subCtx, req)
		if err != nil {
			sub.Err = err
			return
		}
		select {
		case <-subCtx.Done():
			return
		case askChan <- result:
		}

		if d1.IsTickBased {
			// Tick-based: subscribe to feed, emit on each tick
			feedSub := p.feed.Subscribe(subCtx, req.Symbol, time.Now().Unix())

			for {
				select {
				case <-subCtx.Done():
					return
				case tick := <-feedSub.C:
					if tick == nil {
						if feedSub.Err != nil {
							sub.Err = feedSub.Err
						}
						return
					}
					// Recalculate ask with updated market data
					result, err := p.CalculateAsk(subCtx, req)
					if err != nil {
						sub.Err = err
						return
					}
					select {
					case <-subCtx.Done():
						return
					case askChan <- result:
					}
				}
			}
		} else {
			// Time-based: subscribe to feed AND use 5-second ticker
			feedSub := p.feed.Subscribe(subCtx, req.Symbol, time.Now().Unix())
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-subCtx.Done():
					return
				case tick := <-feedSub.C:
					if tick == nil {
						if feedSub.Err != nil {
							sub.Err = feedSub.Err
						}
						return
					}
					result, err := p.CalculateAsk(subCtx, req)
					if err != nil {
						sub.Err = err
						return
					}
					select {
					case <-subCtx.Done():
						return
					case askChan <- result:
					}
				case <-ticker.C:
					result, err := p.CalculateAsk(subCtx, req)
					if err != nil {
						sub.Err = err
						return
					}
					select {
					case <-subCtx.Done():
						return
					case askChan <- result:
					}
				}
			}
		}
	}()

	return sub, nil
}

// SubscribeBid creates a subscription that emits bid price updates until contract expires.
// For tick-based contracts: emits on each tick ONLY (no time-based fallback).
// For time-based contracts: emits on each tick OR every 5 seconds.
func (p *Pricer) SubscribeBid(ctx context.Context, req *BidRequest) (*BidSubscription, error) {
	// Validate request
	if err := p.contract.ValidateBidRequest(ctx, req); err != nil {
		return nil, err
	}

	// Parse durations to determine if tick-based
	d1, err := p.contract.ParseDuration(req.FirstDuration)
	if err != nil {
		return nil, fmt.Errorf("first_duration: %w", err)
	}

	// Create context for subscription lifecycle
	subCtx, cancel := context.WithCancel(ctx)

	// Create channel for bid results
	bidChan := make(chan *BidResult, 10)

	sub := &BidSubscription{
		C:      bidChan,
		cancel: cancel,
	}

	// Start goroutine to handle streaming
	go func() {
		defer close(bidChan)
		defer cancel()

		// Send initial response
		result, err := p.CalculateBid(subCtx, req)
		if err != nil {
			sub.Err = err
			return
		}
		select {
		case <-subCtx.Done():
			return
		case bidChan <- result:
		}

		// If already expired, close immediately
		if result.IsExpired {
			return
		}

		if d1.IsTickBased {
			// Tick-based: subscribe to feed, emit on each tick ONLY
			feedSub := p.feed.Subscribe(subCtx, req.Symbol, req.StartTime)

			for {
				select {
				case <-subCtx.Done():
					return
				case tick := <-feedSub.C:
					if tick == nil {
						if feedSub.Err != nil {
							sub.Err = feedSub.Err
						}
						return
					}
					// Recalculate bid with updated market data
					result, err := p.CalculateBid(subCtx, req)
					if err != nil {
						sub.Err = err
						return
					}
					select {
					case <-subCtx.Done():
						return
					case bidChan <- result:
					}
					// Close if contract expired
					if result.IsExpired {
						return
					}
				}
			}
		} else {
			// Time-based: subscribe to feed AND use 5-second ticker
			feedSub := p.feed.Subscribe(subCtx, req.Symbol, req.StartTime)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-subCtx.Done():
					return
				case tick := <-feedSub.C:
					if tick == nil {
						if feedSub.Err != nil {
							sub.Err = feedSub.Err
						}
						return
					}
					result, err := p.CalculateBid(subCtx, req)
					if err != nil {
						sub.Err = err
						return
					}
					select {
					case <-subCtx.Done():
						return
					case bidChan <- result:
					}
					if result.IsExpired {
						return
					}
				case <-ticker.C:
					result, err := p.CalculateBid(subCtx, req)
					if err != nil {
						sub.Err = err
						return
					}
					select {
					case <-subCtx.Done():
						return
					case bidChan <- result:
					}
					if result.IsExpired {
						return
					}
				}
			}
		}
	}()

	return sub, nil
}

// getCurrentTime returns current Unix timestamp.
// This is a separate function to make testing easier.
func getCurrentTime() int64 {
	return time.Now().Unix()
}
