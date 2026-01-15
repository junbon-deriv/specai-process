package grpcsvc

import (
	"context"
	"fmt"
	"log/slog"

	pb "github.com/regentmarkets/service-pricer-doublerisefall/api/proto/doublerisefall/v1"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the DoubleRiseFallService gRPC service.
type Service struct {
	pb.UnimplementedDoubleRiseFallServiceServer
	pricer *pricer.Pricer
	logger *slog.Logger
}

// New creates a new gRPC service.
func New(p *pricer.Pricer, logger *slog.Logger) *Service {
	return &Service{
		pricer: p,
		logger: logger,
	}
}

// GetAsk handles a single ask price request.
func (s *Service) GetAsk(ctx context.Context, req *pb.GetAskRequest) (*pb.GetAskResponse, error) {
	s.logger.InfoContext(ctx, "GetAsk request",
		"symbol", req.GetOptionParameters().GetSymbol(),
		"contract_type", req.GetOptionParameters().GetContractType().String(),
	)

	// Convert proto request to internal request
	askReq, err := s.protoToAskRequest(req)
	if err != nil {
		return nil, s.mapError(err)
	}

	// Calculate ask price
	result, err := s.pricer.CalculateAsk(ctx, askReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to calculate ask", "error", err)
		return nil, s.mapError(err)
	}

	// Convert result to proto response
	return s.askResultToProto(result), nil
}

// StreamAsk handles streaming ask price requests.
func (s *Service) StreamAsk(req *pb.StreamAskRequest, stream pb.DoubleRiseFallService_StreamAskServer) error {
	ctx := stream.Context()
	s.logger.InfoContext(ctx, "StreamAsk request",
		"symbol", req.GetOptionParameters().GetSymbol(),
	)

	// Convert proto request to internal request
	askReq, err := s.protoToAskRequestFromStream(req)
	if err != nil {
		return s.mapError(err)
	}

	// Subscribe to ask price updates
	sub, err := s.pricer.SubscribeAsk(ctx, askReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to subscribe to ask", "error", err)
		return s.mapError(err)
	}
	defer sub.Close()

	// Stream ask results
	for {
		select {
		case <-ctx.Done():
			return nil
		case result, ok := <-sub.C:
			if !ok {
				// Channel closed, check for error
				if sub.Err != nil {
					s.logger.ErrorContext(ctx, "ask subscription error", "error", sub.Err)
					return s.mapError(sub.Err)
				}
				return nil
			}
			// Send result to client
			if err := stream.Send(s.askResultToProto(result)); err != nil {
				return err
			}
		}
	}
}

// GetBid handles a single bid price request.
func (s *Service) GetBid(ctx context.Context, req *pb.GetBidRequest) (*pb.GetBidResponse, error) {
	s.logger.InfoContext(ctx, "GetBid request",
		"symbol", req.GetOptionParameters().GetSymbol(),
		"start_time", req.GetOptionParameters().GetStartTime(),
	)

	// Convert proto request to internal request
	bidReq, err := s.protoToBidRequest(req)
	if err != nil {
		return nil, s.mapError(err)
	}

	// Calculate bid price
	result, err := s.pricer.CalculateBid(ctx, bidReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to calculate bid", "error", err)
		return nil, s.mapError(err)
	}

	// Convert result to proto response
	return s.bidResultToProto(result), nil
}

// StreamBid handles streaming bid price requests.
func (s *Service) StreamBid(req *pb.StreamBidRequest, stream pb.DoubleRiseFallService_StreamBidServer) error {
	ctx := stream.Context()
	s.logger.InfoContext(ctx, "StreamBid request",
		"symbol", req.GetOptionParameters().GetSymbol(),
		"start_time", req.GetOptionParameters().GetStartTime(),
	)

	// Convert proto request to internal request
	bidReq, err := s.protoToBidRequestFromStream(req)
	if err != nil {
		return s.mapError(err)
	}

	// Subscribe to bid price updates
	sub, err := s.pricer.SubscribeBid(ctx, bidReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to subscribe to bid", "error", err)
		return s.mapError(err)
	}
	defer sub.Close()

	// Stream bid results
	for {
		select {
		case <-ctx.Done():
			return nil
		case result, ok := <-sub.C:
			if !ok {
				// Channel closed, check for error
				if sub.Err != nil {
					s.logger.ErrorContext(ctx, "bid subscription error", "error", sub.Err)
					return s.mapError(sub.Err)
				}
				return nil
			}
			// Send result to client
			if err := stream.Send(s.bidResultToProto(result)); err != nil {
				return err
			}
		}
	}
}

// protoToAskRequest converts proto GetAskRequest to internal AskRequest.
func (s *Service) protoToAskRequest(req *pb.GetAskRequest) (*pricer.AskRequest, error) {
	params := req.GetOptionParameters()
	if params == nil {
		return nil, fmt.Errorf("missing option_parameters")
	}

	return &pricer.AskRequest{
		Symbol:         params.GetSymbol(),
		ContractType:   s.protoToContractType(params.GetContractType()),
		Currency:       params.GetCurrency(),
		FirstDuration:  params.GetFirstDuration(),
		SecondDuration: params.GetSecondDuration(),
		Stake:          params.GetStake(),
		PricingTime:    req.GetPricingTime(),
	}, nil
}

// protoToAskRequestFromStream converts proto StreamAskRequest to internal AskRequest.
func (s *Service) protoToAskRequestFromStream(req *pb.StreamAskRequest) (*pricer.AskRequest, error) {
	params := req.GetOptionParameters()
	if params == nil {
		return nil, fmt.Errorf("missing option_parameters")
	}

	return &pricer.AskRequest{
		Symbol:         params.GetSymbol(),
		ContractType:   s.protoToContractType(params.GetContractType()),
		Currency:       params.GetCurrency(),
		FirstDuration:  params.GetFirstDuration(),
		SecondDuration: params.GetSecondDuration(),
		Stake:          params.GetStake(),
		PricingTime:    req.GetPricingTime(),
	}, nil
}

// protoToBidRequest converts proto GetBidRequest to internal BidRequest.
func (s *Service) protoToBidRequest(req *pb.GetBidRequest) (*pricer.BidRequest, error) {
	params := req.GetOptionParameters()
	if params == nil {
		return nil, fmt.Errorf("missing option_parameters")
	}

	return &pricer.BidRequest{
		Symbol:         params.GetSymbol(),
		ContractType:   s.protoToContractType(params.GetContractType()),
		Currency:       params.GetCurrency(),
		FirstDuration:  params.GetFirstDuration(),
		SecondDuration: params.GetSecondDuration(),
		StartTime:      params.GetStartTime(),
		Stake:          params.GetStake(),
		Payout:         params.GetPayout(),
		PricingTime:    req.GetPricingTime(),
	}, nil
}

// protoToBidRequestFromStream converts proto StreamBidRequest to internal BidRequest.
func (s *Service) protoToBidRequestFromStream(req *pb.StreamBidRequest) (*pricer.BidRequest, error) {
	params := req.GetOptionParameters()
	if params == nil {
		return nil, fmt.Errorf("missing option_parameters")
	}

	return &pricer.BidRequest{
		Symbol:         params.GetSymbol(),
		ContractType:   s.protoToContractType(params.GetContractType()),
		Currency:       params.GetCurrency(),
		FirstDuration:  params.GetFirstDuration(),
		SecondDuration: params.GetSecondDuration(),
		StartTime:      params.GetStartTime(),
		Stake:          params.GetStake(),
		Payout:         params.GetPayout(),
		PricingTime:    req.GetPricingTime(),
	}, nil
}

// protoToContractType converts proto ContractType to internal ContractType.
func (s *Service) protoToContractType(ct pb.ContractType) pricer.ContractType {
	switch ct {
	case pb.ContractType_CONTRACT_TYPE_RISE:
		return pricer.ContractTypeRise
	case pb.ContractType_CONTRACT_TYPE_FALL:
		return pricer.ContractTypeFall
	default:
		return pricer.ContractTypeUnspecified
	}
}

// askResultToProto converts internal AskResult to proto GetAskResponse.
func (s *Service) askResultToProto(result *pricer.AskResult) *pb.GetAskResponse {
	return &pb.GetAskResponse{
		AskPrice:        result.AskPrice,
		Currency:        result.Currency,
		CurrentSpot:     result.CurrentSpot,
		CurrentSpotTime: result.CurrentSpotTime,
		Payout:          result.Payout,
		Limits: &pb.Limits{
			MaxPayout: result.MaxPayout,
			MinStake:  result.MinStake,
		},
	}
}

// bidResultToProto converts internal BidResult to proto GetBidResponse.
func (s *Service) bidResultToProto(result *pricer.BidResult) *pb.GetBidResponse {
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
		EvaluationTime:  result.EvaluationTime,
	}
}

// mapError maps internal errors to gRPC status errors.
func (s *Service) mapError(err error) error {
	// Handle wrapped errors by checking error message content
	errMsg := err.Error()

	// Check for base error types first
	switch err {
	case pricer.ErrInvalidSymbol:
		return status.Errorf(codes.InvalidArgument, "ERR-DF-S1K: %v", err)
	case pricer.ErrSymbolDisabled:
		return status.Errorf(codes.FailedPrecondition, "ERR-DF-S2D: %v", err)
	case pricer.ErrInvalidDuration:
		return status.Errorf(codes.InvalidArgument, "ERR-DF-D1N: %v", err)
	case pricer.ErrInvalidStake:
		return status.Errorf(codes.InvalidArgument, "ERR-DF-K1M: %v", err)
	case pricer.ErrPayoutExceeded:
		return status.Errorf(codes.InvalidArgument, "ERR-DF-P1X: %v", err)
	case pricer.ErrMissingStartTime:
		return status.Errorf(codes.InvalidArgument, "ERR-DF-R1S: %v", err)
	case pricer.ErrMissingPayout:
		return status.Errorf(codes.InvalidArgument, "ERR-DF-R2P: %v", err)
	case pricer.ErrMissingEntryTick:
		return status.Errorf(codes.FailedPrecondition, "ERR-DF-E1M: %v", err)
	case pricer.ErrMarketDataUnavailable:
		return status.Errorf(codes.Unavailable, "ERR-DF-M1E: %v", err)
	}

	// Check for wrapped errors by examining error message
	if err != nil {
		// Duration order errors
		if contains(errMsg, "must be greater than") {
			return status.Errorf(codes.InvalidArgument, "ERR-DF-D2O: %v", err)
		}
		// Duration gap errors
		if contains(errMsg, "gap") && (contains(errMsg, "10 seconds") || contains(errMsg, "2 ticks")) {
			return status.Errorf(codes.InvalidArgument, "ERR-DF-D3G: %v", err)
		}
		// Pricing time in future
		if contains(errMsg, "pricing_time") && contains(errMsg, "future") {
			return status.Errorf(codes.InvalidArgument, "ERR-DF-T1F: %v", err)
		}
		// Missing evaluation tick
		if contains(errMsg, "insufficient ticks") || contains(errMsg, "evaluation tick") {
			return status.Errorf(codes.FailedPrecondition, "ERR-DF-E2T: %v", err)
		}
	}

	return status.Errorf(codes.Internal, "ERR-DF-I1X: internal error")
}

// contains checks if a string contains a substring (case-insensitive helper).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findInString(s, substr)))
}

// findInString searches for substring in string.
func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
