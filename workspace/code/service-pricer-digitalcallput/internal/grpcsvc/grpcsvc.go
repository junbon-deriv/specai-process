package grpcsvc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/contract"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// Service implements the DigitalCallPutService gRPC service
type Service struct {
	pb.UnimplementedDigitalCallPutServiceServer
	pricer pricing.Pricer
	market pricing.MarketDataProvider
	logger *slog.Logger
}

// New creates a new gRPC service instance
func New(pricer pricing.Pricer, market pricing.MarketDataProvider, logger *slog.Logger) *Service {
	return &Service{
		pricer: pricer,
		market: market,
		logger: logger,
	}
}

// GetAsk implements the GetAsk RPC method
func (s *Service) GetAsk(ctx context.Context, req *pb.GetAskRequest) (*pb.GetAskResponse, error) {
	s.logger.Info("GetAsk request",
		"symbol", req.Symbol,
		"contract_type", req.ContractType.String(),
		"duration", req.Duration,
		"stake", req.Stake)

	// Validate input
	if err := s.validateAskRequest(req); err != nil {
		s.logger.Error("GetAsk validation failed", "error", err)
		return nil, err
	}

	// Convert request to pricing input
	input := pricing.AskInput{
		Symbol:       req.Symbol,
		ContractType: convertContractType(req.ContractType),
		Currency:     req.Currency,
		Duration:     req.Duration,
		Stake:        req.Stake,
		Barrier:      req.Barrier,
		PricingTime:  req.PricingTime,
	}

	// Delegate to pricing module
	result, err := s.pricer.CalculateAsk(ctx, input)
	if err != nil {
		s.logger.Error("GetAsk calculation failed", "error", err)
		return nil, s.mapError(err)
	}

	s.logger.Debug("GetAsk calculated",
		"ask_price", result.AskPrice,
		"payout", result.Payout,
		"current_spot", result.CurrentSpot)

	// Return response
	return &pb.GetAskResponse{
		AskPrice:        result.AskPrice,
		Currency:        result.Currency,
		CurrentSpot:     result.CurrentSpot,
		CurrentSpotTime: result.CurrentSpotTime,
		Payout:          result.Payout,
		Limits: &pb.Limits{
			MaxPayout: result.Limits.MaxPayout.String(),
			MinStake:  result.Limits.MinStake.String(),
		},
	}, nil
}

// StreamAsk implements the StreamAsk RPC method with proper tick subscription
func (s *Service) StreamAsk(req *pb.GetAskRequest, stream pb.DigitalCallPutService_StreamAskServer) error {
	s.logger.Info("StreamAsk started",
		"symbol", req.Symbol,
		"contract_type", req.ContractType.String(),
		"duration", req.Duration,
		"stake", req.Stake)

	// Validate input
	if err := s.validateAskRequest(req); err != nil {
		s.logger.Error("StreamAsk validation failed", "error", err)
		return err
	}

	ctx := stream.Context()

	// Parse duration to determine if tick-based or time-based
	duration, err := contract.ParseDuration(req.Duration)
	if err != nil {
		s.logger.Error("StreamAsk invalid duration", "error", err)
		return status.Errorf(codes.InvalidArgument, "invalid duration: %v", err)
	}

	// Subscribe to tick stream
	tickCh, err := s.market.StreamTicks(ctx, req.Symbol)
	if err != nil {
		s.logger.Error("StreamAsk failed to subscribe to ticks", "error", err)
		return s.mapError(err)
	}

	// Create 5-second heartbeat timer ONLY for time-based durations (REQ-ST-T2Q)
	// Tick-based streams should ONLY update on new ticks
	var tickerCh <-chan time.Time
	if duration.IsTime() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		tickerCh = ticker.C
	} else {
		// For tick-based, use nil channel which blocks forever
		tickerCh = nil
	}

	// Convert request to pricing input
	input := pricing.AskInput{
		Symbol:       req.Symbol,
		ContractType: convertContractType(req.ContractType),
		Currency:     req.Currency,
		Duration:     req.Duration,
		Stake:        req.Stake,
		Barrier:      req.Barrier,
		PricingTime:  req.PricingTime,
	}

	// Send initial response immediately
	if err := s.sendAskResponse(ctx, &input, stream); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("StreamAsk context cancelled", "symbol", req.Symbol)
			return ctx.Err()

		case tick, ok := <-tickCh:
			if !ok {
				// Tick channel closed
				s.logger.Warn("StreamAsk tick channel closed", "symbol", req.Symbol)
				return status.Error(codes.Unavailable, "market data stream closed")
			}
			s.logger.Debug("StreamAsk received tick",
				"symbol", req.Symbol,
				"price", tick.Price.String(),
				"timestamp", tick.Timestamp)

			if err := s.sendAskResponse(ctx, &input, stream); err != nil {
				return err
			}

		case <-tickerCh:
			// 5-second heartbeat for time-based durations only
			// For tick-based, tickerCh is nil and this case never triggers
			s.logger.Debug("StreamAsk heartbeat", "symbol", req.Symbol)
			if err := s.sendAskResponse(ctx, &input, stream); err != nil {
				return err
			}
		}
	}
}

// sendAskResponse calculates and sends an Ask response
func (s *Service) sendAskResponse(ctx context.Context, input *pricing.AskInput, stream pb.DigitalCallPutService_StreamAskServer) error {
	result, err := s.pricer.CalculateAsk(ctx, *input)
	if err != nil {
		s.logger.Error("StreamAsk calculation failed", "error", err)
		return s.mapError(err)
	}

	resp := &pb.GetAskResponse{
		AskPrice:        result.AskPrice,
		Currency:        result.Currency,
		CurrentSpot:     result.CurrentSpot,
		CurrentSpotTime: result.CurrentSpotTime,
		Payout:          result.Payout,
		Limits: &pb.Limits{
			MaxPayout: result.Limits.MaxPayout.String(),
			MinStake:  result.Limits.MinStake.String(),
		},
	}

	if err := stream.Send(resp); err != nil {
		s.logger.Error("StreamAsk send failed", "error", err)
		return err
	}

	return nil
}

// GetBid implements the GetBid RPC method
func (s *Service) GetBid(ctx context.Context, req *pb.GetBidRequest) (*pb.GetBidResponse, error) {
	s.logger.Info("GetBid request",
		"symbol", req.Symbol,
		"contract_type", req.ContractType.String(),
		"duration", req.Duration,
		"start_time", req.StartTime,
		"payout", req.Payout)

	// Validate input
	if err := s.validateBidRequest(req); err != nil {
		s.logger.Error("GetBid validation failed", "error", err)
		return nil, err
	}

	// Convert request to pricing input
	input := pricing.BidInput{
		Symbol:       req.Symbol,
		ContractType: convertContractType(req.ContractType),
		Currency:     req.Currency,
		Duration:     req.Duration,
		Barrier:      req.Barrier,
		StartTime:    req.StartTime,
		Payout:       req.Payout,
	}

	// Delegate to pricing module
	result, err := s.pricer.CalculateBid(ctx, input)
	if err != nil {
		s.logger.Error("GetBid calculation failed", "error", err)
		return nil, s.mapError(err)
	}

	s.logger.Debug("GetBid calculated",
		"bid_price", result.BidPrice,
		"is_expired", result.IsExpired,
		"current_spot", result.CurrentSpot)

	// Return response
	return &pb.GetBidResponse{
		BidPrice:        result.BidPrice,
		IsExpired:       result.IsExpired,
		CurrentSpot:     result.CurrentSpot,
		CurrentSpotTime: result.CurrentSpotTime,
		EntrySpot:       result.EntrySpot,
		EntrySpotTime:   result.EntrySpotTime,
		ExitSpot:        result.ExitSpot,
		ExitSpotTime:    result.ExitSpotTime,
		Barrier:         result.Barrier,
		StartTime:       result.StartTime,
		ExpiryTime:      result.ExpiryTime,
		Currency:        result.Currency,
	}, nil
}

// StreamBid implements the StreamBid RPC method with proper tick subscription
func (s *Service) StreamBid(req *pb.GetBidRequest, stream pb.DigitalCallPutService_StreamBidServer) error {
	s.logger.Info("StreamBid started",
		"symbol", req.Symbol,
		"contract_type", req.ContractType.String(),
		"duration", req.Duration,
		"start_time", req.StartTime)

	// Validate input
	if err := s.validateBidRequest(req); err != nil {
		s.logger.Error("StreamBid validation failed", "error", err)
		return err
	}

	ctx := stream.Context()

	// Parse duration to determine if tick-based or time-based
	duration, err := contract.ParseDuration(req.Duration)
	if err != nil {
		s.logger.Error("StreamBid invalid duration", "error", err)
		return status.Errorf(codes.InvalidArgument, "invalid duration: %v", err)
	}

	// Subscribe to tick stream
	tickCh, err := s.market.StreamTicks(ctx, req.Symbol)
	if err != nil {
		s.logger.Error("StreamBid failed to subscribe to ticks", "error", err)
		return s.mapError(err)
	}

	// Create 5-second heartbeat timer ONLY for time-based durations (REQ-ST-T2Q)
	// Tick-based streams should ONLY update on new ticks
	var tickerCh <-chan time.Time
	if duration.IsTime() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		tickerCh = ticker.C
	} else {
		// For tick-based, use nil channel which blocks forever
		tickerCh = nil
	}

	// Convert request to pricing input
	input := pricing.BidInput{
		Symbol:       req.Symbol,
		ContractType: convertContractType(req.ContractType),
		Currency:     req.Currency,
		Duration:     req.Duration,
		Barrier:      req.Barrier,
		StartTime:    req.StartTime,
		Payout:       req.Payout,
	}

	// Send initial response immediately
	result, err := s.sendBidResponse(ctx, &input, stream)
	if err != nil {
		return err
	}
	if result.IsExpired {
		s.logger.Info("StreamBid terminated - contract expired", "symbol", req.Symbol)
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("StreamBid context cancelled", "symbol", req.Symbol)
			return ctx.Err()

		case tick, ok := <-tickCh:
			if !ok {
				// Tick channel closed
				s.logger.Warn("StreamBid tick channel closed", "symbol", req.Symbol)
				return status.Error(codes.Unavailable, "market data stream closed")
			}
			s.logger.Debug("StreamBid received tick",
				"symbol", req.Symbol,
				"price", tick.Price.String(),
				"timestamp", tick.Timestamp)

			result, err := s.sendBidResponse(ctx, &input, stream)
			if err != nil {
				return err
			}
			if result.IsExpired {
				s.logger.Info("StreamBid terminated - contract expired", "symbol", req.Symbol)
				return nil
			}

		case <-tickerCh:
			// 5-second heartbeat for time-based durations only
			// For tick-based, tickerCh is nil and this case never triggers
			s.logger.Debug("StreamBid heartbeat", "symbol", req.Symbol)
			result, err := s.sendBidResponse(ctx, &input, stream)
			if err != nil {
				return err
			}
			if result.IsExpired {
				s.logger.Info("StreamBid terminated - contract expired", "symbol", req.Symbol)
				return nil
			}
		}
	}
}

// sendBidResponse calculates and sends a Bid response
func (s *Service) sendBidResponse(ctx context.Context, input *pricing.BidInput, stream pb.DigitalCallPutService_StreamBidServer) (*pricing.BidResult, error) {
	result, err := s.pricer.CalculateBid(ctx, *input)
	if err != nil {
		s.logger.Error("StreamBid calculation failed", "error", err)
		return nil, s.mapError(err)
	}

	resp := &pb.GetBidResponse{
		BidPrice:        result.BidPrice,
		IsExpired:       result.IsExpired,
		CurrentSpot:     result.CurrentSpot,
		CurrentSpotTime: result.CurrentSpotTime,
		EntrySpot:       result.EntrySpot,
		EntrySpotTime:   result.EntrySpotTime,
		ExitSpot:        result.ExitSpot,
		ExitSpotTime:    result.ExitSpotTime,
		Barrier:         result.Barrier,
		StartTime:       result.StartTime,
		ExpiryTime:      result.ExpiryTime,
		Currency:        result.Currency,
	}

	if err := stream.Send(resp); err != nil {
		s.logger.Error("StreamBid send failed", "error", err)
		return nil, err
	}

	return result, nil
}

// validateAskRequest validates the GetAsk/StreamAsk request
func (s *Service) validateAskRequest(req *pb.GetAskRequest) error {
	if req.Symbol == "" {
		return status.Error(codes.InvalidArgument, "symbol is required")
	}
	if req.ContractType == pb.ContractType_CONTRACT_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "contract_type must be CALL or PUT")
	}
	if req.Currency == "" {
		return status.Error(codes.InvalidArgument, "currency is required")
	}
	if req.Duration == "" {
		return status.Error(codes.InvalidArgument, "duration is required")
	}
	if req.Stake == "" {
		return status.Error(codes.InvalidArgument, "stake is required")
	}
	return nil
}

// validateBidRequest validates the GetBid/StreamBid request
func (s *Service) validateBidRequest(req *pb.GetBidRequest) error {
	if req.Symbol == "" {
		return status.Error(codes.InvalidArgument, "symbol is required")
	}
	if req.ContractType == pb.ContractType_CONTRACT_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "contract_type must be CALL or PUT")
	}
	if req.Currency == "" {
		return status.Error(codes.InvalidArgument, "currency is required")
	}
	if req.Duration == "" {
		return status.Error(codes.InvalidArgument, "duration is required")
	}
	if req.StartTime <= 0 {
		return status.Error(codes.InvalidArgument, "start_time is required and must be positive")
	}
	if req.Payout == "" {
		return status.Error(codes.InvalidArgument, "payout is required")
	}
	return nil
}

// mapError maps domain errors to gRPC status codes
func (s *Service) mapError(err error) error {
	switch {
	case err == pricing.ErrStakeBelowMinimum:
		return status.Errorf(codes.InvalidArgument, "stake below minimum: %v", err)
	case err == pricing.ErrPayoutExceedsMaximum:
		return status.Errorf(codes.InvalidArgument, "payout exceeds maximum: %v", err)
	case err == pricing.ErrUnknownSymbol:
		return status.Errorf(codes.NotFound, "unknown symbol: %v", err)
	case err == pricing.ErrInvalidDuration:
		return status.Errorf(codes.InvalidArgument, "invalid duration format: %v", err)
	case err == pricing.ErrInvalidBarrier:
		return status.Errorf(codes.InvalidArgument, "invalid barrier format: %v", err)
	case err == pricing.ErrMarketUnavailable:
		return status.Errorf(codes.Unavailable, "market data unavailable: %v", err)
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}

// convertContractType converts protobuf ContractType to pricing.ContractType
func convertContractType(ct pb.ContractType) pricing.ContractType {
	switch ct {
	case pb.ContractType_CONTRACT_TYPE_CALL:
		return pricing.ContractTypeCall
	case pb.ContractType_CONTRACT_TYPE_PUT:
		return pricing.ContractTypePut
	default:
		return 0 // This should not happen due to validation
	}
}
