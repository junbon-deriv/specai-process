package grpcsvc

import (
	"context"
	"fmt"
	"time"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/feed"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/model"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricer"
)

type Server struct {
	pb.UnimplementedDigitalcallputServiceServer
	pricer *pricer.Pricer
	feed   feed.FeedClient
}

func NewServer(p *pricer.Pricer, f feed.FeedClient) *Server {
	return &Server{
		pricer: p,
		feed:   f,
	}
}

func (s *Server) GetAsk(ctx context.Context, req *pb.GetAskRequest) (*pb.GetAskResponse, error) {
	// 1. Get Spot
	spot, spotTime, err := s.feed.GetSpot(ctx, req.OptionParameters.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get spot: %w", err)
	}

	// 2. Map to Internal Model
	internalReq := model.PricingRequest{
		Symbol:       req.OptionParameters.Symbol,
		ContractType: req.OptionParameters.ContractType,
		Currency:     req.OptionParameters.Currency,
		Duration:     req.OptionParameters.Duration,
		Barrier:      req.OptionParameters.Barrier,
		StartTime:    req.OptionParameters.StartTime,
		Stake:        req.OptionParameters.Stake,
		PricingTime:  time.Now(),
	}

	// 3. Calculate
	res, err := s.pricer.CalculateAsk(internalReq, spot)
	if err != nil {
		return nil, fmt.Errorf("pricing failed: %w", err)
	}

	// 4. Map Response
	return &pb.GetAskResponse{
		AskPrice:        res.Price,
		Currency:        req.OptionParameters.Currency,
		CurrentSpot:     res.CurrentSpot,
		CurrentSpotTime: spotTime,
		Payout:          res.Payout,
		Limits:          res.Limits,
	}, nil
}

func (s *Server) StreamAsk(req *pb.StreamAskRequest, stream pb.DigitalcallputService_StreamAskServer) error {
	// Simple streaming implementation: send one update then wait/loop
	// In production, this would subscribe to feed updates

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			// Re-use GetAsk logic
			spot, spotTime, err := s.feed.GetSpot(stream.Context(), req.OptionParameters.Symbol)
			if err != nil {
				continue // Log error
			}

			internalReq := model.PricingRequest{
				Symbol:       req.OptionParameters.Symbol,
				ContractType: req.OptionParameters.ContractType,
				Currency:     req.OptionParameters.Currency,
				Duration:     req.OptionParameters.Duration,
				Barrier:      req.OptionParameters.Barrier,
				StartTime:    req.OptionParameters.StartTime,
				Stake:        req.OptionParameters.Stake,
				PricingTime:  time.Now(),
			}

			res, err := s.pricer.CalculateAsk(internalReq, spot)
			if err != nil {
				continue
			}

			if err := stream.Send(&pb.GetAskResponse{
				AskPrice:        res.Price,
				Currency:        req.OptionParameters.Currency,
				CurrentSpot:     res.CurrentSpot,
				CurrentSpotTime: spotTime,
				Payout:          res.Payout,
				Limits:          res.Limits,
			}); err != nil {
				return err
			}
		}
	}
}

func (s *Server) GetBid(ctx context.Context, req *pb.GetBidRequest) (*pb.GetBidResponse, error) {
	// 1. Get Current Spot
	currentSpot, currentSpotTime, err := s.feed.GetSpot(ctx, req.OptionParameters.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get current spot: %w", err)
	}

	// 2. Get Entry Spot (Historical)
	if req.OptionParameters.StartTime == nil {
		return nil, fmt.Errorf("start_time is required for Bid")
	}
	entrySpot, entrySpotTime, err := s.feed.GetHistoricalSpot(ctx, req.OptionParameters.Symbol, *req.OptionParameters.StartTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry spot: %w", err)
	}

	// 3. Map to Internal Model
	internalReq := model.ValuationRequest{
		Symbol:       req.OptionParameters.Symbol,
		ContractType: req.OptionParameters.ContractType,
		Currency:     req.OptionParameters.Currency,
		Duration:     req.OptionParameters.Duration,
		Barrier:      req.OptionParameters.Barrier,
		StartTime:    *req.OptionParameters.StartTime,
		Stake:        req.OptionParameters.Stake,
		PricingTime:  time.Now(),
	}

	// 4. Calculate
	res, err := s.pricer.CalculateBid(internalReq, currentSpot, entrySpot)
	if err != nil {
		return nil, fmt.Errorf("valuation failed: %w", err)
	}

	// 5. Map Response
	return &pb.GetBidResponse{
		BidPrice:        res.BidPrice,
		IsExpired:       res.IsExpired,
		CurrentSpot:     res.CurrentSpot,
		CurrentSpotTime: currentSpotTime,
		EntrySpot:       res.EntrySpot,
		EntrySpotTime:   entrySpotTime,
		ExitSpot:        res.ExitSpot,
		ExitSpotTime:    res.ExitSpotTime,
		Barrier:         res.Barrier,
		StartTime:       res.StartTime,
		ExpiryTime:      res.ExpiryTime,
		Currency:        req.OptionParameters.Currency,
	}, nil
}

func (s *Server) StreamBid(req *pb.StreamBidRequest, stream pb.DigitalcallputService_StreamBidServer) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Entry spot is constant for the stream
	if req.OptionParameters.StartTime == nil {
		return fmt.Errorf("start_time is required for Bid")
	}
	entrySpot, entrySpotTime, err := s.feed.GetHistoricalSpot(stream.Context(), req.OptionParameters.Symbol, *req.OptionParameters.StartTime)
	if err != nil {
		return fmt.Errorf("failed to get entry spot: %w", err)
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			currentSpot, currentSpotTime, err := s.feed.GetSpot(stream.Context(), req.OptionParameters.Symbol)
			if err != nil {
				continue
			}

			internalReq := model.ValuationRequest{
				Symbol:       req.OptionParameters.Symbol,
				ContractType: req.OptionParameters.ContractType,
				Currency:     req.OptionParameters.Currency,
				Duration:     req.OptionParameters.Duration,
				Barrier:      req.OptionParameters.Barrier,
				StartTime:    *req.OptionParameters.StartTime,
				Stake:        req.OptionParameters.Stake,
				PricingTime:  time.Now(),
			}

			res, err := s.pricer.CalculateBid(internalReq, currentSpot, entrySpot)
			if err != nil {
				continue
			}

			if err := stream.Send(&pb.GetBidResponse{
				BidPrice:        res.BidPrice,
				IsExpired:       res.IsExpired,
				CurrentSpot:     res.CurrentSpot,
				CurrentSpotTime: currentSpotTime,
				EntrySpot:       res.EntrySpot,
				EntrySpotTime:   entrySpotTime,
				ExitSpot:        res.ExitSpot,
				ExitSpotTime:    res.ExitSpotTime,
				Barrier:         res.Barrier,
				StartTime:       res.StartTime,
				ExpiryTime:      res.ExpiryTime,
				Currency:        req.OptionParameters.Currency,
			}); err != nil {
				return err
			}

			if res.IsExpired {
				return nil // End stream on expiry
			}
		}
	}
}
