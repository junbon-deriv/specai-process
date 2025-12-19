package feed

import (
	"context"
	"time"
)

// FeedClient defines the interface for interacting with the feed service
type FeedClient interface {
	GetSpot(ctx context.Context, symbol string) (float64, int64, error)
	GetHistoricalSpot(ctx context.Context, symbol string, timestamp int64) (float64, int64, error)
}

// MockFeedClient is a temporary implementation until the real service-feed is available
type MockFeedClient struct{}

func NewMockFeedClient() *MockFeedClient {
	return &MockFeedClient{}
}

func (m *MockFeedClient) GetSpot(ctx context.Context, symbol string) (float64, int64, error) {
	// Return a dummy spot price
	return 100.0, time.Now().Unix(), nil
}

func (m *MockFeedClient) GetHistoricalSpot(ctx context.Context, symbol string, timestamp int64) (float64, int64, error) {
	// Return a dummy historical spot price
	return 99.0, timestamp, nil
}

// RealFeedClient would wrap the gRPC client for service-feed
// type RealFeedClient struct {
//     client pb.FeedServiceClient
// }
// ... implementation ...
