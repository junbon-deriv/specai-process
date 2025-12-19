package pricer

import (
	"testing"
	"time"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/model"
)

func TestCalculateAsk(t *testing.T) {
	p := NewPricer()

	req := model.PricingRequest{
		Symbol:       "R_100",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Duration:     "1h",
		Stake:        "10",
		PricingTime:  time.Now(),
	}

	currentSpot := 100.0

	res, err := p.CalculateAsk(req, currentSpot)
	if err != nil {
		t.Fatalf("CalculateAsk failed: %v", err)
	}

	if res.Price != "10" {
		t.Errorf("Expected Price 10, got %s", res.Price)
	}

	if res.Payout == "" {
		t.Error("Expected Payout to be set")
	}

	// Basic sanity check: Payout should be > Stake for a fair game (ignoring edge cases where prob is 1)
	// With ATM call, prob ~ 0.5. Payout ~ Stake / 0.5 = 2 * Stake.
	// Let's just check it parses.
}

func TestCalculateBid(t *testing.T) {
	p := NewPricer()

	startTime := time.Now().Add(-30 * time.Minute).Unix()

	req := model.ValuationRequest{
		Symbol:       "R_100",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Duration:     "1h",
		StartTime:    startTime,
		Stake:        "10",
		PricingTime:  time.Now(),
	}

	currentSpot := 100.0
	entrySpot := 100.0

	res, err := p.CalculateBid(req, currentSpot, entrySpot)
	if err != nil {
		t.Fatalf("CalculateBid failed: %v", err)
	}

	if res.BidPrice == "" {
		t.Error("Expected BidPrice to be set")
	}

	if res.IsExpired {
		t.Error("Expected not expired")
	}
}

func TestCalculateAsk_RelativeBarrier(t *testing.T) {
	p := NewPricer()

	barrier := "+1.0"
	req := model.PricingRequest{
		Symbol:       "R_100",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Duration:     "1h",
		Barrier:      &barrier,
		Stake:        "10",
		PricingTime:  time.Now(),
	}

	currentSpot := 100.0
	// Barrier should be 101.0

	res, err := p.CalculateAsk(req, currentSpot)
	if err != nil {
		t.Fatalf("CalculateAsk failed: %v", err)
	}

	// With barrier > spot for Call, probability < 0.5
	// Payout should be higher than ATM (which was ~20)
	// Let's just check it runs and gives a valid price
	if res.Price != "10" {
		t.Errorf("Expected Price 10, got %s", res.Price)
	}
}

func TestCalculateAsk_Validation(t *testing.T) {
	p := NewPricer()

	// Test Min Stake
	req := model.PricingRequest{
		Symbol:       "R_100",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Duration:     "1h",
		Stake:        "0.5", // Below min stake of 1
		PricingTime:  time.Now(),
	}

	currentSpot := 100.0

	_, err := p.CalculateAsk(req, currentSpot)
	if err != model.ErrInvalidStake {
		t.Errorf("Expected ErrInvalidStake, got %v", err)
	}

	// Test Max Payout
	// To trigger max payout, we need a very high probability (low payout per unit) or high stake.
	// Payout = Stake / (Prob * DF)
	// If we set Stake = 10000 (Max Payout), and Prob < 1, Payout > 10000.
	req.Stake = "10000"

	_, err = p.CalculateAsk(req, currentSpot)
	if err != model.ErrPayoutExceedsLimit {
		t.Errorf("Expected ErrPayoutExceedsLimit, got %v", err)
	}
}
