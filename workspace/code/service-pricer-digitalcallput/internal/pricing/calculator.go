package pricing

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FeedSubscriber defines the interface for market data feed (Rule 5.5: Interface at Consumer)
type FeedSubscriber interface {
	GetCurrentTick(ctx context.Context, symbol string) (*Tick, error)
	GetTickAfterTime(ctx context.Context, symbol string, afterTime time.Time) (*Tick, error)
	Subscribe(ctx context.Context, symbol string) (<-chan *Tick, error)
	StreamWithFallback(ctx context.Context, symbol string, fallbackDuration time.Duration) (<-chan *Tick, error)
}

// Tick represents a real-time market data point from service-feed
type Tick struct {
	Symbol    string
	Price     float64
	Timestamp time.Time
}

// OptionParams represents the internal option parameters
type OptionParams struct {
	Symbol       string
	ContractType pb.ContractType
	Currency     string
	Stake        float64
	Duration     Duration
	Barrier      *Barrier
	StartTime    *time.Time
	Payout       *float64 // Required for bid requests: fixed payout from purchase time
}

// AskQuote represents the result of ask price calculation
type AskQuote struct {
	AskPrice        float64
	Currency        string
	CurrentSpot     float64
	CurrentSpotTime time.Time
	Payout          float64
	Limits          *TradingLimits
}

// ToProto converts AskQuote to protobuf response
func (q *AskQuote) ToProto() *pb.GetAskResponse {
	return &pb.GetAskResponse{
		AskPrice:        formatDecimal(q.AskPrice),
		Currency:        q.Currency,
		CurrentSpot:     formatDecimal(q.CurrentSpot),
		CurrentSpotTime: q.CurrentSpotTime.Unix(),
		Payout:          formatDecimal(q.Payout),
		Limits: &pb.Limits{
			MaxPayout: formatDecimal(q.Limits.MaxPayout),
			MinStake:  formatDecimal(q.Limits.MinStake),
		},
	}
}

// BidQuote represents the result of bid price calculation
type BidQuote struct {
	BidPrice        float64
	IsExpired       bool
	CurrentSpot     float64
	CurrentSpotTime time.Time
	EntrySpot       float64
	EntrySpotTime   time.Time
	ExitSpot        *float64
	ExitSpotTime    *time.Time
	Barrier         float64
	StartTime       time.Time
	ExpiryTime      time.Time
	Currency        string
}

// ToProto converts BidQuote to protobuf response
func (q *BidQuote) ToProto() *pb.GetBidResponse {
	resp := &pb.GetBidResponse{
		BidPrice:        formatDecimal(q.BidPrice),
		IsExpired:       q.IsExpired,
		CurrentSpot:     formatDecimal(q.CurrentSpot),
		CurrentSpotTime: q.CurrentSpotTime.Unix(),
		EntrySpot:       formatDecimal(q.EntrySpot),
		EntrySpotTime:   q.EntrySpotTime.Unix(),
		Barrier:         formatDecimal(q.Barrier),
		StartTime:       q.StartTime.Unix(),
		ExpiryTime:      q.ExpiryTime.Unix(),
		Currency:        q.Currency,
	}

	if q.ExitSpot != nil {
		resp.ExitSpot = formatDecimal(*q.ExitSpot)
	}

	if q.ExitSpotTime != nil {
		resp.ExitSpotTime = q.ExitSpotTime.Unix()
	}

	return resp
}

// AskSubscription represents an active ask price stream
type AskSubscription struct {
	quotes chan *AskQuote
	err    error
	done   chan struct{}
}

// Quotes returns the read-only quote channel
func (s *AskSubscription) Quotes() <-chan *AskQuote {
	return s.quotes
}

// Err returns any error that occurred during streaming
func (s *AskSubscription) Err() error {
	return s.err
}

// Close terminates the subscription
func (s *AskSubscription) Close() {
	close(s.done)
}

// BidSubscription represents an active bid price stream
type BidSubscription struct {
	quotes chan *BidQuote
	err    error
	done   chan struct{}
	// AMENDMENT 3: State for tick-based contracts
	tickCount     int  // Number of ticks received after entry
	requiredTicks int  // Required number of ticks for expiry
	entryReceived bool // Whether entry tick has been received
}

// Quotes returns the read-only quote channel
func (s *BidSubscription) Quotes() <-chan *BidQuote {
	return s.quotes
}

// Err returns any error that occurred during streaming
func (s *BidSubscription) Err() error {
	return s.err
}

// Close terminates the subscription
func (s *BidSubscription) Close() {
	close(s.done)
}

// TradingLimits contains trading constraints
type TradingLimits struct {
	MinStake  float64
	MaxPayout float64
}

// PricingConfig contains global pricing parameters
type PricingConfig struct {
	Volatility   float64 // Annual volatility (σ)
	Commission   float64 // Commission rate
	InterestRate float64 // Risk-free rate (r)
	QuantoDrift  float64 // Quanto adjustment (q)
}

// formatDecimal formats a float64 as a decimal string with 2 decimal places for money
func formatDecimal(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

// Calculator orchestrates all pricing calculations
type Calculator struct {
	config *PricingConfig
	limits *TradingLimits
	feed   FeedSubscriber
}

// NewCalculator creates a new pricing calculator
func NewCalculator(config *PricingConfig, limits *TradingLimits, feed FeedSubscriber) *Calculator {
	return &Calculator{
		config: config,
		limits: limits,
		feed:   feed,
	}
}

// CalculateAsk calculates the ask price for a new contract proposal
func (c *Calculator) CalculateAsk(ctx context.Context, params *pb.OptionParameters) (*AskQuote, error) {
	// 1. Validate and parse parameters
	optionParams, err := c.parseAndValidateParams(params, false)
	if err != nil {
		return nil, err
	}

	// 2. Get current market tick
	tick, err := c.feed.GetCurrentTick(ctx, optionParams.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Market data feed unavailable: %v", err)
	}

	// 3. AMENDMENT 4: Pass tick to calculation function
	return c.calculateAskQuote(ctx, optionParams, tick)
}

// CalculateBid calculates the bid price for an active contract
func (c *Calculator) CalculateBid(ctx context.Context, params *pb.OptionParameters) (*BidQuote, error) {
	// 1. Validate and parse parameters (bid requires start_time and payout)
	optionParams, err := c.parseAndValidateParams(params, true)
	if err != nil {
		return nil, err
	}

	// 2. Get current market tick
	tick, err := c.feed.GetCurrentTick(ctx, optionParams.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Market data feed unavailable: %v", err)
	}

	// 3. AMENDMENT 4: Pass tick to calculation function
	return c.calculateBidQuote(ctx, optionParams, tick)
}

// StreamAsk creates a subscription that streams ask price updates
func (c *Calculator) StreamAsk(ctx context.Context, params *pb.OptionParameters) (*AskSubscription, error) {
	// 1. Validate parameters
	optionParams, err := c.parseAndValidateParams(params, false)
	if err != nil {
		return nil, err
	}

	// 2. Create subscription
	sub := &AskSubscription{
		quotes: make(chan *AskQuote, 10),
		done:   make(chan struct{}),
	}

	// 3. Subscribe to feed with 5-second fallback
	tickChan, err := c.feed.StreamWithFallback(ctx, optionParams.Symbol, 5*time.Second)
	if err != nil {
		return nil, err
	}

	// 4. Start goroutine to handle streaming
	go func() {
		defer close(sub.quotes)

		// Send initial quote immediately (fetch initial tick)
		initialTick, err := c.feed.GetCurrentTick(ctx, optionParams.Symbol)
		if err != nil {
			sub.err = err
			return
		}
		quote, err := c.calculateAskQuote(ctx, optionParams, initialTick)
		if err != nil {
			sub.err = err
			return
		}

		select {
		case sub.quotes <- quote:
		case <-ctx.Done():
			sub.err = ctx.Err()
			return
		case <-sub.done:
			return
		}

		// AMENDMENT 4: Stream updates on each tick - pass tick through pipeline
		for {
			select {
			case <-ctx.Done():
				sub.err = ctx.Err()
				return

			case <-sub.done:
				return

			case tick, ok := <-tickChan:
				if !ok {
					return
				}

				// AMENDMENT 4: Pass the received tick to calculation (no GetCurrentTick)
				quote, err := c.calculateAskQuote(ctx, optionParams, tick)
				if err != nil {
					sub.err = err
					return
				}

				select {
				case sub.quotes <- quote:
				case <-ctx.Done():
					sub.err = ctx.Err()
					return
				case <-sub.done:
					return
				}
			}
		}
	}()

	return sub, nil
}

// StreamBid creates a subscription that streams bid price updates
func (c *Calculator) StreamBid(ctx context.Context, params *pb.OptionParameters) (*BidSubscription, error) {
	// 1. Validate parameters (bid requires start_time)
	optionParams, err := c.parseAndValidateParams(params, true)
	if err != nil {
		return nil, err
	}

	// 2. Parse duration to check if tick-based
	duration, err := ParseDuration(params.Duration)
	if err != nil {
		return nil, err
	}

	// 3. Create subscription with tick counting state for tick-based contracts
	sub := &BidSubscription{
		quotes:        make(chan *BidQuote, 10),
		done:          make(chan struct{}),
		tickCount:     0,
		requiredTicks: 0,
		entryReceived: false,
	}

	// AMENDMENT 3: Initialize tick counting for tick-based durations
	if !duration.IsTimeBased() {
		sub.requiredTicks, err = duration.ToTicks()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to parse tick duration: %v", err)
		}
	}

	// 4. Subscribe to feed (with or without fallback)
	var tickChan <-chan *Tick
	if duration.IsTimeBased() {
		// Time-based: use 5-second fallback
		tickChan, err = c.feed.StreamWithFallback(ctx, optionParams.Symbol, 5*time.Second)
	} else {
		// Tick-based: NO fallback, only tick arrivals
		tickChan, err = c.feed.Subscribe(ctx, optionParams.Symbol)
	}
	if err != nil {
		return nil, err
	}

	// 5. Start goroutine to handle streaming
	go func() {
		defer close(sub.quotes)

		// Send initial quote immediately (fetch initial tick)
		initialTick, err := c.feed.GetCurrentTick(ctx, optionParams.Symbol)
		if err != nil {
			sub.err = err
			return
		}
		quote, err := c.calculateBidQuote(ctx, optionParams, initialTick)
		if err != nil {
			sub.err = err
			return
		}

		select {
		case sub.quotes <- quote:
		case <-ctx.Done():
			sub.err = ctx.Err()
			return
		case <-sub.done:
			return
		}

		// Check if already expired
		if quote.IsExpired {
			return
		}

		// AMENDMENT 4: Stream updates on each tick - pass tick through pipeline
		for {
			select {
			case <-ctx.Done():
				sub.err = ctx.Err()
				return

			case <-sub.done:
				return

			case tick, ok := <-tickChan:
				if !ok {
					return
				}

				// AMENDMENT 3: Update tick count for tick-based contracts
				if !duration.IsTimeBased() {
					if !sub.entryReceived {
						sub.entryReceived = true
						sub.tickCount = 0 // Reset on entry tick
					} else {
						sub.tickCount++
					}
				}

				// AMENDMENT 4: Pass the received tick and tick count to calculation
				quote, err := c.calculateBidQuoteWithState(ctx, optionParams, tick, sub.tickCount, sub.requiredTicks)
				if err != nil {
					sub.err = err
					return
				}

				select {
				case sub.quotes <- quote:
				case <-ctx.Done():
					sub.err = ctx.Err()
					return
				case <-sub.done:
					return
				}

				// Auto-terminate stream when contract expires
				if quote.IsExpired {
					return
				}
			}
		}
	}()

	return sub, nil
}

// AMENDMENT 4: calculateAskQuote now accepts tick as parameter to avoid race conditions
// The tick that triggers repricing must be passed through the calculation pipeline
func (c *Calculator) calculateAskQuote(ctx context.Context, params *OptionParams, tick *Tick) (*AskQuote, error) {
	// Calculate time to expiry
	timeToExpiry, err := c.calculateTimeToExpiry(params.Duration, time.Now())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to calculate expiry: %v", err)
	}

	// Resolve barrier
	barrier := tick.Price // Default to entry spot
	if params.Barrier != nil {
		barrier, err = params.Barrier.ResolveBarrier(tick.Price)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to resolve barrier: %v", err)
		}
	}

	// Calculate probability using Black-Scholes
	probability := BlackScholesDigitalOption(
		tick.Price,
		barrier,
		timeToExpiry,
		c.config.Volatility,
		c.config.InterestRate,
		c.config.QuantoDrift,
		params.ContractType,
	)

	// Calculate payout
	payout := CalculateAskPayout(params.Stake, probability, c.config.Commission)

	// Build and return quote
	return &AskQuote{
		AskPrice:        params.Stake, // Ask price equals stake
		Currency:        params.Currency,
		CurrentSpot:     tick.Price,
		CurrentSpotTime: tick.Timestamp,
		Payout:          payout,
		Limits:          c.limits,
	}, nil
}

// AMENDMENT 1, 2, 3, 4: calculateBidQuote with all fixes
// - Accepts tick as parameter (no GetCurrentTick race condition)
// - Uses provided payout instead of recalculating
// - Uses tick timestamps for entry/exit spot time
// - Simple version without tick counting (for CalculateBid unary call)
func (c *Calculator) calculateBidQuote(ctx context.Context, params *OptionParams, tick *Tick) (*BidQuote, error) {
	return c.calculateBidQuoteWithState(ctx, params, tick, 0, 0)
}

// calculateBidQuoteWithState performs bid calculation with tick counting state
// tickCount: current number of ticks received (for tick-based contracts)
// requiredTicks: required number of ticks for expiry (for tick-based contracts)
func (c *Calculator) calculateBidQuoteWithState(ctx context.Context, params *OptionParams, tick *Tick, tickCount int, requiredTicks int) (*BidQuote, error) {
	// AMENDMENT 1: Use provided payout instead of recalculating
	if params.Payout == nil {
		return nil, status.Error(codes.Internal, "payout is required for bid calculation")
	}
	payout := *params.Payout

	// AMENDMENT 2: Fetch entry tick (first tick after start_time)
	entryTick, err := c.feed.GetTickAfterTime(ctx, params.Symbol, *params.StartTime)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Failed to get entry tick: %v", err)
	}
	entrySpot := entryTick.Price
	entrySpotTime := entryTick.Timestamp // AMENDMENT 2: Use tick timestamp, not start_time

	// Resolve barrier based on entry spot
	barrier := entrySpot // Default
	if params.Barrier != nil {
		barrier, err = params.Barrier.ResolveBarrier(entrySpot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to resolve barrier: %v", err)
		}
	}

	// Calculate expiry time and check expiry
	var expiryTime time.Time
	var isExpired bool

	if params.Duration.IsTimeBased() {
		// AMENDMENT 3: Time-based duration - use time comparison
		expiryTime, err = params.Duration.CalculateExpiryTime(*params.StartTime)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to calculate expiry: %v", err)
		}
		now := time.Now()
		isExpired = now.After(expiryTime) || now.Equal(expiryTime)
	} else {
		// AMENDMENT 3: Tick-based duration - use tick counting
		expiryTime = time.Now().Add(365 * 24 * time.Hour) // Far future for display
		isExpired = tickCount >= requiredTicks
	}

	// Determine bid price
	var bidPrice float64
	var exitSpot *float64
	var exitSpotTime *time.Time

	if isExpired {
		// Contract expired - check win condition
		exitSpotVal := tick.Price
		exitSpot = &exitSpotVal
		// AMENDMENT 2: Use tick timestamp for exit spot time
		exitSpotTimeVal := tick.Timestamp
		exitSpotTime = &exitSpotTimeVal

		hasWon := c.checkWinCondition(tick.Price, barrier, params.ContractType)
		// Use probability = 1.0 for expired contracts (certainty)
		bidPrice = CalculateBidPrice(payout, 1.0, true, hasWon)
	} else {
		// Contract active - calculate current value for early exit
		// Only time-based contracts support early exit
		if params.Duration.IsTimeBased() {
			now := time.Now()
			remainingTime := expiryTime.Sub(now).Seconds() / (365.25 * 24 * 60 * 60)
			currentProbability := BlackScholesDigitalOption(
				tick.Price,
				barrier,
				remainingTime,
				c.config.Volatility,
				c.config.InterestRate,
				c.config.QuantoDrift,
				params.ContractType,
			)
			bidPrice = CalculateBidPrice(payout, currentProbability, false, false)
		} else {
			// Tick-based contracts: NO early exit support
			// Bid price not applicable for partial tick completion
			bidPrice = 0
		}
	}

	// Build and return quote
	return &BidQuote{
		BidPrice:        bidPrice,
		IsExpired:       isExpired,
		CurrentSpot:     tick.Price,
		CurrentSpotTime: tick.Timestamp,
		EntrySpot:       entrySpot,
		EntrySpotTime:   entrySpotTime,
		ExitSpot:        exitSpot,
		ExitSpotTime:    exitSpotTime,
		Barrier:         barrier,
		StartTime:       *params.StartTime,
		ExpiryTime:      expiryTime,
		Currency:        params.Currency,
	}, nil
}

// parseAndValidateParams validates and converts protobuf parameters to internal types
func (c *Calculator) parseAndValidateParams(params *pb.OptionParameters, requireStartTime bool) (*OptionParams, error) {
	// Validate symbol
	if params.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	// Validate contract type
	if params.ContractType == pb.ContractType_CONTRACT_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "contract_type is required")
	}

	// Validate currency
	if params.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}

	// Parse and validate stake
	stake, err := strconv.ParseFloat(params.Stake, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid stake format: %s", params.Stake)
	}
	if stake <= 0 {
		return nil, status.Error(codes.OutOfRange, "Stake must be positive")
	}
	if stake < c.limits.MinStake {
		return nil, status.Errorf(codes.FailedPrecondition, "Stake below minimum: %.2f", c.limits.MinStake)
	}

	// Parse duration
	duration, err := ParseDuration(params.Duration)
	if err != nil {
		return nil, err
	}

	// Parse barrier (optional)
	var barrier *Barrier
	if params.Barrier != nil && *params.Barrier != "" {
		barrier, err = ParseBarrier(*params.Barrier)
		if err != nil {
			return nil, err
		}
	} else {
		barrier = &Barrier{Type: BarrierTypeNone}
	}

	// Validate start_time for bid requests
	var startTime *time.Time
	if requireStartTime {
		if params.StartTime == nil {
			return nil, status.Error(codes.InvalidArgument, "start_time is required for bid requests")
		}
		st := time.Unix(*params.StartTime, 0)
		if st.After(time.Now()) {
			return nil, status.Error(codes.InvalidArgument, "start_time cannot be in the future")
		}
		startTime = &st
	}

	// AMENDMENT: Validate and parse payout for bid requests
	// Payout must be provided for bid requests and must not be recalculated
	var payout *float64
	if requireStartTime {
		if params.Payout == nil || *params.Payout == "" {
			return nil, status.Error(codes.InvalidArgument, "payout is required for bid requests")
		}
		payoutVal, err := strconv.ParseFloat(*params.Payout, 64)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid payout format: %s", *params.Payout)
		}
		if payoutVal <= 0 {
			return nil, status.Error(codes.OutOfRange, "Payout must be positive")
		}
		payout = &payoutVal
	}

	return &OptionParams{
		Symbol:       params.Symbol,
		ContractType: params.ContractType,
		Currency:     params.Currency,
		Stake:        stake,
		Duration:     *duration,
		Barrier:      barrier,
		StartTime:    startTime,
		Payout:       payout,
	}, nil
}

// calculateTimeToExpiry calculates time to expiry in years
func (c *Calculator) calculateTimeToExpiry(duration Duration, fromTime time.Time) (float64, error) {
	seconds, err := duration.ToSeconds()
	if err != nil {
		return 0, err
	}

	return TimeToExpiryInYears(seconds), nil
}

// checkWinCondition determines if a contract has won
func (c *Calculator) checkWinCondition(exitSpot, barrier float64, contractType pb.ContractType) bool {
	if contractType == pb.ContractType_CONTRACT_TYPE_CALL {
		return exitSpot > barrier
	}
	return exitSpot < barrier
}
