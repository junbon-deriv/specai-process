// Package grpcsvc implements the gRPC service handlers.
package grpcsvc

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	pb "github.com/regentmarkets/service-pricer-doublerisefall/api/proto/doublerisefall/v1"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Pricer defines the pricing operations required by the gRPC service.
type Pricer interface {
	CalculateAsk(ctx context.Context, req *pricer.AskRequest) (*pricer.AskResult, error)
	CalculateBid(ctx context.Context, req *pricer.BidRequest) (*pricer.BidResult, error)
	StreamAsk(ctx context.Context, req *pricer.AskRequest) (<-chan *pricer.AskResult, <-chan error)
	StreamBid(ctx context.Context, req *pricer.BidRequest) (<-chan *pricer.BidResult, <-chan error)
}

// ContractValidator validates contract parameters.
type ContractValidator interface {
	ParseDuration(s string) (pricer.Duration, error)
}

// Service implements the DoubleRiseFallService gRPC service.
type Service struct {
	pb.UnimplementedDoubleRiseFallServiceServer
	pricer    Pricer
	validator ContractValidator
	logger    *slog.Logger
}

// NewService creates a new gRPC service.
func NewService(pricer Pricer, validator ContractValidator, logger *slog.Logger) *Service {
	return &Service{
		pricer:    pricer,
		validator: validator,
		logger:    logger,
	}
}

// GetAsk calculates the ask price for a contract.
func (s *Service) GetAsk(ctx context.Context, req *pb.GetAskRequest) (*pb.GetAskResponse, error) {
	s.logger.Debug("GetAsk called", "symbol", req.GetOptionParameters().GetSymbol())

	// Map proto to domain
	askReq, err := s.mapProtoToAskRequest(req)
	if err != nil {
		s.logger.Error("failed to map request", "error", err)
		return nil, mapDomainErrorToGRPC(err)
	}

	// Call pricer
	result, err := s.pricer.CalculateAsk(ctx, askReq)
	if err != nil {
		s.logger.Error("failed to calculate ask", "error", err)
		return nil, mapDomainErrorToGRPC(err)
	}

	// Map domain to proto
	return mapAskResultToProto(result), nil
}

// StreamAsk streams real-time ask price updates.
func (s *Service) StreamAsk(req *pb.StreamAskRequest, stream pb.DoubleRiseFallService_StreamAskServer) error {
	s.logger.Debug("StreamAsk called", "symbol", req.GetOptionParameters().GetSymbol())

	// Map proto to domain
	askReq, err := s.mapProtoToAskRequestFromStream(req)
	if err != nil {
		s.logger.Error("failed to map stream request", "error", err)
		return mapDomainErrorToGRPC(err)
	}

	// Start streaming
	resultCh, errCh := s.pricer.StreamAsk(stream.Context(), askReq)

	// Forward results to client
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case err := <-errCh:
			if err != nil {
				s.logger.Error("stream error", "error", err)
				return mapDomainErrorToGRPC(err)
			}
			return nil
		case result, ok := <-resultCh:
			if !ok {
				return nil
			}
			resp := mapAskResultToProto(result)
			if err := stream.Send(resp); err != nil {
				s.logger.Error("failed to send result", "error", err)
				return err
			}
		}
	}
}

// GetBid calculates the bid price for an active contract.
func (s *Service) GetBid(ctx context.Context, req *pb.GetBidRequest) (*pb.GetBidResponse, error) {
	s.logger.Debug("GetBid called", "symbol", req.GetOptionParameters().GetSymbol())

	// Map proto to domain
	bidReq, err := s.mapProtoToBidRequest(req)
	if err != nil {
		s.logger.Error("failed to map request", "error", err)
		return nil, mapDomainErrorToGRPC(err)
	}

	// Call pricer
	result, err := s.pricer.CalculateBid(ctx, bidReq)
	if err != nil {
		s.logger.Error("failed to calculate bid", "error", err)
		return nil, mapDomainErrorToGRPC(err)
	}

	// Map domain to proto
	return mapBidResultToProto(result), nil
}

// StreamBid streams real-time bid price updates.
func (s *Service) StreamBid(req *pb.StreamBidRequest, stream pb.DoubleRiseFallService_StreamBidServer) error {
	s.logger.Debug("StreamBid called", "symbol", req.GetOptionParameters().GetSymbol())

	// Map proto to domain
	bidReq, err := s.mapProtoToBidRequestFromStream(req)
	if err != nil {
		s.logger.Error("failed to map stream request", "error", err)
		return mapDomainErrorToGRPC(err)
	}

	// Start streaming
	resultCh, errCh := s.pricer.StreamBid(stream.Context(), bidReq)

	// Forward results to client
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case err := <-errCh:
			if err != nil {
				s.logger.Error("stream error", "error", err)
				return mapDomainErrorToGRPC(err)
			}
			return nil
		case result, ok := <-resultCh:
			if !ok {
				return nil
			}
			resp := mapBidResultToProto(result)
			if err := stream.Send(resp); err != nil {
				s.logger.Error("failed to send result", "error", err)
				return err
			}
		}
	}
}

// mapProtoToAskRequest maps GetAskRequest proto to domain AskRequest
func (s *Service) mapProtoToAskRequest(req *pb.GetAskRequest) (*pricer.AskRequest, error) {
	params := req.GetOptionParameters()

	// Parse durations
	firstDuration, err := s.validator.ParseDuration(params.GetFirstDuration())
	if err != nil {
		return nil, err
	}

	secondDuration, err := s.validator.ParseDuration(params.GetSecondDuration())
	if err != nil {
		return nil, err
	}

	// Parse stake
	stake, err := strconv.ParseFloat(params.GetStake(), 64)
	if err != nil {
		return nil, pricer.ErrInvalidStake
	}

	return &pricer.AskRequest{
		Symbol:         params.GetSymbol(),
		ContractType:   mapProtoContractType(params.GetContractType()),
		Currency:       params.GetCurrency(),
		FirstDuration:  firstDuration,
		SecondDuration: secondDuration,
		Stake:          stake,
		PricingTime:    req.GetPricingTime(),
	}, nil
}

// mapProtoToAskRequestFromStream maps StreamAskRequest proto to domain AskRequest
func (s *Service) mapProtoToAskRequestFromStream(req *pb.StreamAskRequest) (*pricer.AskRequest, error) {
	params := req.GetOptionParameters()

	// Parse durations
	firstDuration, err := s.validator.ParseDuration(params.GetFirstDuration())
	if err != nil {
		return nil, err
	}

	secondDuration, err := s.validator.ParseDuration(params.GetSecondDuration())
	if err != nil {
		return nil, err
	}

	// Parse stake
	stake, err := strconv.ParseFloat(params.GetStake(), 64)
	if err != nil {
		return nil, pricer.ErrInvalidStake
	}

	return &pricer.AskRequest{
		Symbol:         params.GetSymbol(),
		ContractType:   mapProtoContractType(params.GetContractType()),
		Currency:       params.GetCurrency(),
		FirstDuration:  firstDuration,
		SecondDuration: secondDuration,
		Stake:          stake,
		PricingTime:    req.GetPricingTime(),
	}, nil
}

// mapProtoToBidRequest maps GetBidRequest proto to domain BidRequest
func (s *Service) mapProtoToBidRequest(req *pb.GetBidRequest) (*pricer.BidRequest, error) {
	params := req.GetOptionParameters()

	// Parse durations
	firstDuration, err := s.validator.ParseDuration(params.GetFirstDuration())
	if err != nil {
		return nil, err
	}

	secondDuration, err := s.validator.ParseDuration(params.GetSecondDuration())
	if err != nil {
		return nil, err
	}

	// Parse stake
	stake, err := strconv.ParseFloat(params.GetStake(), 64)
	if err != nil {
		return nil, pricer.ErrInvalidStake
	}

	// Parse payout
	payout, err := strconv.ParseFloat(params.GetPayout(), 64)
	if err != nil {
		return nil, pricer.ErrMissingPayout
	}

	return &pricer.BidRequest{
		Symbol:         params.GetSymbol(),
		ContractType:   mapProtoContractType(params.GetContractType()),
		Currency:       params.GetCurrency(),
		FirstDuration:  firstDuration,
		SecondDuration: secondDuration,
		StartTime:      params.GetStartTime(),
		Stake:          stake,
		Payout:         payout,
		PricingTime:    req.GetPricingTime(),
	}, nil
}

// mapProtoToBidRequestFromStream maps StreamBidRequest proto to domain BidRequest
func (s *Service) mapProtoToBidRequestFromStream(req *pb.StreamBidRequest) (*pricer.BidRequest, error) {
	params := req.GetOptionParameters()

	// Parse durations
	firstDuration, err := s.validator.ParseDuration(params.GetFirstDuration())
	if err != nil {
		return nil, err
	}

	secondDuration, err := s.validator.ParseDuration(params.GetSecondDuration())
	if err != nil {
		return nil, err
	}

	// Parse stake
	stake, err := strconv.ParseFloat(params.GetStake(), 64)
	if err != nil {
		return nil, pricer.ErrInvalidStake
	}

	// Parse payout
	payout, err := strconv.ParseFloat(params.GetPayout(), 64)
	if err != nil {
		return nil, pricer.ErrMissingPayout
	}

	return &pricer.BidRequest{
		Symbol:         params.GetSymbol(),
		ContractType:   mapProtoContractType(params.GetContractType()),
		Currency:       params.GetCurrency(),
		FirstDuration:  firstDuration,
		SecondDuration: secondDuration,
		StartTime:      params.GetStartTime(),
		Stake:          stake,
		Payout:         payout,
		PricingTime:    req.GetPricingTime(),
	}, nil
}

// mapProtoContractType maps proto ContractType to domain ContractType
func mapProtoContractType(ct pb.ContractType) pricer.ContractType {
	switch ct {
	case pb.ContractType_CONTRACT_TYPE_RISE:
		return pricer.ContractTypeRise
	case pb.ContractType_CONTRACT_TYPE_FALL:
		return pricer.ContractTypeFall
	default:
		return pricer.ContractTypeUnspecified
	}
}

// mapAskResultToProto maps domain AskResult to proto GetAskResponse
func mapAskResultToProto(result *pricer.AskResult) *pb.GetAskResponse {
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

// mapBidResultToProto maps domain BidResult to proto GetBidResponse
func mapBidResultToProto(result *pricer.BidResult) *pb.GetBidResponse {
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

// mapDomainErrorToGRPC maps domain errors to gRPC status codes
func mapDomainErrorToGRPC(err error) error {
	switch err {
	case pricer.ErrInvalidSymbol:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-S1V: %v", err))
	case pricer.ErrInvalidDuration:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-D2U: %v", err))
	case pricer.ErrDurationOrder:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-D3O: %v", err))
	case pricer.ErrDurationGap:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-D4G: %v", err))
	case pricer.ErrInvalidStake:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-K5S: %v", err))
	case pricer.ErrPayoutExceeded:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-P6X: %v", err))
	case pricer.ErrPricingTimeFuture:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-T2F: %v", err))
	case pricer.ErrMissingStartTime:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-T1M: %v", err))
	case pricer.ErrMissingPayout:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-P2Y: %v", err))
	case pricer.ErrInvalidContractType:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-C2T: %v", err))
	case pricer.ErrInvalidCurrency:
		return status.Error(codes.InvalidArgument, fmt.Sprintf("ERR-DR-C3U: %v", err))
	case pricer.ErrSymbolDisabled:
		return status.Error(codes.FailedPrecondition, fmt.Sprintf("ERR-DR-Y7D: %v", err))
	case pricer.ErrMissingEntryTick:
		return status.Error(codes.FailedPrecondition, fmt.Sprintf("ERR-DR-E3T: %v", err))
	case pricer.ErrMarketDataUnavailable:
		return status.Error(codes.Unavailable, fmt.Sprintf("ERR-DR-M8E: %v", err))
	case pricer.ErrStreamDisconnected:
		return status.Error(codes.Unavailable, fmt.Sprintf("ERR-DR-C1D: %v", err))
	case pricer.ErrInternal:
		return status.Error(codes.Internal, fmt.Sprintf("ERR-DR-I9N: %v", err))
	default:
		return status.Error(codes.Internal, fmt.Sprintf("ERR-DR-I9N: %v", err))
	}
}
