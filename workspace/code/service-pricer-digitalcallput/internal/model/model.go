package model

import (
	"errors"
	"time"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
)

var (
	ErrInvalidStake       = errors.New("stake must be greater than or equal to minimum stake")
	ErrPayoutExceedsLimit = errors.New("potential payout exceeds maximum limit")
)

// PricingRequest represents the internal request for pricing
type PricingRequest struct {
	Symbol       string
	ContractType pb.ContractType
	Currency     string
	Duration     string
	Barrier      *string
	StartTime    *int64
	Stake        string
	PricingTime  time.Time
}

// PricingResult represents the result of a pricing calculation
type PricingResult struct {
	Price           string
	Payout          string
	CurrentSpot     string
	CurrentSpotTime int64
	Limits          *pb.Limits
}

// ValuationRequest represents the internal request for valuation
type ValuationRequest struct {
	Symbol       string
	ContractType pb.ContractType
	Currency     string
	Duration     string
	Barrier      *string
	StartTime    int64
	Stake        string
	PricingTime  time.Time
}

// ValuationResult represents the result of a valuation calculation
type ValuationResult struct {
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
}
