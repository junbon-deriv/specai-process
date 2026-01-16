package grpcsvc

import (
	"context"
	"testing"

	pb "github.com/regentmarkets/service-pricer-doublerisefall/api/proto/doublerisefall/v1"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Test mapProtoContractType
func TestMapProtoContractType(t *testing.T) {
	tests := []struct {
		name  string
		input pb.ContractType
		want  pricer.ContractType
	}{
		{
			name:  "RISE",
			input: pb.ContractType_CONTRACT_TYPE_RISE,
			want:  pricer.ContractTypeRise,
		},
		{
			name:  "FALL",
			input: pb.ContractType_CONTRACT_TYPE_FALL,
			want:  pricer.ContractTypeFall,
		},
		{
			name:  "UNSPECIFIED",
			input: pb.ContractType_CONTRACT_TYPE_UNSPECIFIED,
			want:  pricer.ContractTypeUnspecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapProtoContractType(tt.input)
			if got != tt.want {
				t.Errorf("mapProtoContractType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test mapDomainErrorToGRPC
func TestMapDomainErrorToGRPC(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "ErrInvalidSymbol",
			err:      pricer.ErrInvalidSymbol,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrInvalidDuration",
			err:      pricer.ErrInvalidDuration,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrDurationOrder",
			err:      pricer.ErrDurationOrder,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrDurationGap",
			err:      pricer.ErrDurationGap,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrInvalidStake",
			err:      pricer.ErrInvalidStake,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrPayoutExceeded",
			err:      pricer.ErrPayoutExceeded,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrSymbolDisabled",
			err:      pricer.ErrSymbolDisabled,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "ErrMissingEntryTick",
			err:      pricer.ErrMissingEntryTick,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "ErrMarketDataUnavailable",
			err:      pricer.ErrMarketDataUnavailable,
			wantCode: codes.Unavailable,
		},
		{
			name:     "ErrStreamDisconnected",
			err:      pricer.ErrStreamDisconnected,
			wantCode: codes.Unavailable,
		},
		{
			name:     "ErrInternal",
			err:      pricer.ErrInternal,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcErr := mapDomainErrorToGRPC(tt.err)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("mapDomainErrorToGRPC() did not return a gRPC status error")
			}
			if st.Code() != tt.wantCode {
				t.Errorf("mapDomainErrorToGRPC() code = %v, want %v", st.Code(), tt.wantCode)
			}
		})
	}
}

// Test mapAskResultToProto
func TestMapAskResultToProto(t *testing.T) {
	result := &pricer.AskResult{
		AskPrice:        "0.4667",
		Currency:        "USD",
		CurrentSpot:     "1234.5678",
		CurrentSpotTime: 1736930731,
		Payout:          "21.43",
		MaxPayout:       "1000.00",
		MinStake:        "1.00",
	}

	proto := mapAskResultToProto(result)

	if proto.AskPrice != result.AskPrice {
		t.Errorf("AskPrice = %v, want %v", proto.AskPrice, result.AskPrice)
	}
	if proto.Currency != result.Currency {
		t.Errorf("Currency = %v, want %v", proto.Currency, result.Currency)
	}
	if proto.CurrentSpot != result.CurrentSpot {
		t.Errorf("CurrentSpot = %v, want %v", proto.CurrentSpot, result.CurrentSpot)
	}
	if proto.CurrentSpotTime != result.CurrentSpotTime {
		t.Errorf("CurrentSpotTime = %v, want %v", proto.CurrentSpotTime, result.CurrentSpotTime)
	}
	if proto.Payout != result.Payout {
		t.Errorf("Payout = %v, want %v", proto.Payout, result.Payout)
	}
	if proto.Limits == nil {
		t.Fatal("Limits is nil")
	}
	if proto.Limits.MaxPayout != result.MaxPayout {
		t.Errorf("MaxPayout = %v, want %v", proto.Limits.MaxPayout, result.MaxPayout)
	}
	if proto.Limits.MinStake != result.MinStake {
		t.Errorf("MinStake = %v, want %v", proto.Limits.MinStake, result.MinStake)
	}
}

// Test mapBidResultToProto
func TestMapBidResultToProto(t *testing.T) {
	result := &pricer.BidResult{
		BidPrice:        "27.85",
		IsExpired:       true,
		CurrentSpot:     "1235.1234",
		CurrentSpotTime: 1736930731,
		EntrySpot:       "1234.5678",
		EntrySpotTime:   1736930601,
		ExitSpot:        "1235.1234",
		ExitSpotTime:    1736930720,
		Barrier:         "1234.5678",
		StartTime:       1736930600,
		ExpiryTime:      1736930720,
		EvaluationTime:  1736930660,
		Currency:        "USD",
	}

	proto := mapBidResultToProto(result)

	if proto.BidPrice != result.BidPrice {
		t.Errorf("BidPrice = %v, want %v", proto.BidPrice, result.BidPrice)
	}
	if proto.IsExpired != result.IsExpired {
		t.Errorf("IsExpired = %v, want %v", proto.IsExpired, result.IsExpired)
	}
	if proto.CurrentSpot != result.CurrentSpot {
		t.Errorf("CurrentSpot = %v, want %v", proto.CurrentSpot, result.CurrentSpot)
	}
	if proto.Currency != result.Currency {
		t.Errorf("Currency = %v, want %v", proto.Currency, result.Currency)
	}
}

// MockPricer for testing gRPC handlers
type MockPricer struct {
	askResult *pricer.AskResult
	bidResult *pricer.BidResult
	err       error
}

func (m *MockPricer) CalculateAsk(ctx context.Context, req *pricer.AskRequest) (*pricer.AskResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.askResult, nil
}

func (m *MockPricer) CalculateBid(ctx context.Context, req *pricer.BidRequest) (*pricer.BidResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.bidResult, nil
}

func (m *MockPricer) StreamAsk(ctx context.Context, req *pricer.AskRequest) (<-chan *pricer.AskResult, <-chan error) {
	resultCh := make(chan *pricer.AskResult, 1)
	errCh := make(chan error, 1)
	if m.err != nil {
		errCh <- m.err
	} else if m.askResult != nil {
		resultCh <- m.askResult
	}
	close(resultCh)
	close(errCh)
	return resultCh, errCh
}

func (m *MockPricer) StreamBid(ctx context.Context, req *pricer.BidRequest) (<-chan *pricer.BidResult, <-chan error) {
	resultCh := make(chan *pricer.BidResult, 1)
	errCh := make(chan error, 1)
	if m.err != nil {
		errCh <- m.err
	} else if m.bidResult != nil {
		resultCh <- m.bidResult
	}
	close(resultCh)
	close(errCh)
	return resultCh, errCh
}

// MockValidator for testing
type MockValidator struct{}

func (m *MockValidator) ParseDuration(s string) (pricer.Duration, error) {
	return pricer.Duration{Value: 60, Unit: pricer.DurationUnitSeconds}, nil
}
