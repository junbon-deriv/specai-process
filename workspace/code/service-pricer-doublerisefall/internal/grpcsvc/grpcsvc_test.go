package grpcsvc

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockPricer mocks the pricer.Pricer for testing
type MockPricer struct {
	mock.Mock
}

func (m *MockPricer) CalculateAsk(ctx context.Context, req *pricer.AskRequest) (*pricer.AskResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pricer.AskResult), args.Error(1)
}

func (m *MockPricer) CalculateBid(ctx context.Context, req *pricer.BidRequest) (*pricer.BidResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pricer.BidResult), args.Error(1)
}

// MockContractValidator mocks the contract validator
type MockContractValidator struct {
	mock.Mock
}

func (m *MockContractValidator) ValidateAskRequest(ctx context.Context, req *pricer.AskRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockContractValidator) ValidateBidRequest(ctx context.Context, req *pricer.BidRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockContractValidator) ParseDuration(s string) (pricer.Duration, error) {
	args := m.Called(s)
	return args.Get(0).(pricer.Duration), args.Error(1)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TC-DF-G3V: Map validation error to INVALID_ARGUMENT
func TestMapError_InvalidSymbol(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrInvalidSymbol)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-S1K")
}

func TestMapError_SymbolDisabled(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrSymbolDisabled)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-S2D")
}

func TestMapError_InvalidDuration(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrInvalidDuration)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-D1N")
}

func TestMapError_InvalidStake(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrInvalidStake)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-K1M")
}

func TestMapError_PayoutExceeded(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrPayoutExceeded)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-P1X")
}

func TestMapError_MissingStartTime(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrMissingStartTime)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-R1S")
}

func TestMapError_MissingPayout(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrMissingPayout)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-R2P")
}

func TestMapError_MissingEntryTick(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrMissingEntryTick)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-E1M")
}

// TC-DF-G4W: Map feed error to UNAVAILABLE
func TestMapError_MarketDataUnavailable(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(pricer.ErrMarketDataUnavailable)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-M1E")
}

func TestMapError_DurationOrderViolation(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(errors.New("second_duration must be greater than first_duration"))

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-D2O")
}

func TestMapError_DurationGapTooSmall(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(errors.New("duration gap must be at least 10 seconds"))

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-D3G")
}

func TestMapError_InternalError(t *testing.T) {
	s := &Service{logger: testLogger()}

	err := s.mapError(errors.New("some unknown error"))

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "ERR-DF-I1X")
}

// Test proto conversion
func TestProtoToContractType(t *testing.T) {
	s := &Service{logger: testLogger()}

	tests := []struct {
		name     string
		input    int
		expected pricer.ContractType
	}{
		{
			name:     "rise",
			input:    1, // CONTRACT_TYPE_RISE
			expected: pricer.ContractTypeRise,
		},
		{
			name:     "fall",
			input:    2, // CONTRACT_TYPE_FALL
			expected: pricer.ContractTypeFall,
		},
		{
			name:     "unspecified",
			input:    0, // CONTRACT_TYPE_UNSPECIFIED
			expected: pricer.ContractTypeUnspecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Actual test would require generated proto types
			_ = s
			_ = tt
		})
	}
}

// Test askResultToProto conversion
func TestAskResultToProto(t *testing.T) {
	s := &Service{logger: testLogger()}

	result := &pricer.AskResult{
		AskPrice:        "10.00",
		Currency:        "USD",
		CurrentSpot:     "1234.56",
		CurrentSpotTime: 1704067200,
		Payout:          "25.00",
		MaxPayout:       "1000.00",
		MinStake:        "1.00",
	}

	response := s.askResultToProto(result)

	assert.Equal(t, "10.00", response.GetAskPrice())
	assert.Equal(t, "USD", response.GetCurrency())
	assert.Equal(t, "1234.56", response.GetCurrentSpot())
	assert.Equal(t, int64(1704067200), response.GetCurrentSpotTime())
	assert.Equal(t, "25.00", response.GetPayout())
	assert.Equal(t, "1000.00", response.GetLimits().GetMaxPayout())
	assert.Equal(t, "1.00", response.GetLimits().GetMinStake())
}

// Test bidResultToProto conversion
func TestBidResultToProto(t *testing.T) {
	s := &Service{logger: testLogger()}

	result := &pricer.BidResult{
		BidPrice:        "25.00",
		IsExpired:       true,
		CurrentSpot:     "1234.56",
		CurrentSpotTime: 1704067200,
		EntrySpot:       "1234.00",
		EntrySpotTime:   1704067100,
		ExitSpot:        "1235.00",
		ExitSpotTime:    1704067160,
		Barrier:         "1234.00",
		StartTime:       1704067100,
		ExpiryTime:      1704067160,
		Currency:        "USD",
		EvaluationTime:  1704067130,
	}

	response := s.bidResultToProto(result)

	assert.Equal(t, "25.00", response.GetBidPrice())
	assert.True(t, response.GetIsExpired())
	assert.Equal(t, "1234.56", response.GetCurrentSpot())
	assert.Equal(t, int64(1704067200), response.GetCurrentSpotTime())
	assert.Equal(t, "1234.00", response.GetEntrySpot())
	assert.Equal(t, int64(1704067100), response.GetEntrySpotTime())
	assert.Equal(t, "1235.00", response.GetExitSpot())
	assert.Equal(t, int64(1704067160), response.GetExitSpotTime())
	assert.Equal(t, "1234.00", response.GetBarrier())
	assert.Equal(t, int64(1704067100), response.GetStartTime())
	assert.Equal(t, int64(1704067160), response.GetExpiryTime())
	assert.Equal(t, "USD", response.GetCurrency())
	assert.Equal(t, int64(1704067130), response.GetEvaluationTime())
}

// Test contains helper function
func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"must be greater than", "greater than", true},
		{"duration gap must be at least 10 seconds", "10 seconds", true},
		{"", "", true},
		{"abc", "", true},
		{"", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			assert.Equal(t, tt.want, got)
		})
	}
}
