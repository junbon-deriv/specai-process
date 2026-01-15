package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestConfig(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "symbols.yaml")
	content := `
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_50:
    symbol: R_50
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)
	return configPath
}

// TestNew_ValidConfig tests app creation with valid configuration
func TestNew_ValidConfig(t *testing.T) {
	t.Skip("Requires service-feed to be running")

	configPath := createTestConfig(t)

	cfg := &Config{
		GRPCPort:          50051,
		FeedServiceAddr:   "localhost:50051",
		ConfigPath:        configPath,
		LogLevel:          "info",
		FeedRetryAttempts: 3,
		FeedRetryDelay:    time.Second,
	}

	app, err := New(cfg)

	require.NoError(t, err)
	assert.NotNil(t, app)
	assert.NotNil(t, app.cfg)
	assert.NotNil(t, app.pricer)
	assert.NotNil(t, app.grpcService)
}

// TestNew_InvalidConfigPath tests app creation with invalid config path
func TestNew_InvalidConfigPath(t *testing.T) {
	t.Skip("Requires service-feed to be running")

	cfg := &Config{
		GRPCPort:          50051,
		FeedServiceAddr:   "localhost:50051",
		ConfigPath:        "/nonexistent/config.yaml",
		LogLevel:          "info",
		FeedRetryAttempts: 3,
		FeedRetryDelay:    time.Second,
	}

	app, err := New(cfg)

	assert.Nil(t, app)
	assert.Error(t, err)
}

// TestNew_InvalidFeedAddress tests app creation with invalid feed address
func TestNew_InvalidFeedAddress(t *testing.T) {
	t.Skip("Requires valid service-feed address")

	configPath := createTestConfig(t)

	cfg := &Config{
		GRPCPort:          50051,
		FeedServiceAddr:   "invalid-address",
		ConfigPath:        configPath,
		LogLevel:          "info",
		FeedRetryAttempts: 1,
		FeedRetryDelay:    time.Millisecond,
	}

	app, err := New(cfg)

	assert.Nil(t, app)
	assert.Error(t, err)
}

// TestApp_Shutdown tests graceful shutdown
func TestApp_Shutdown(t *testing.T) {
	t.Skip("Requires service-feed to be running")

	configPath := createTestConfig(t)

	cfg := &Config{
		GRPCPort:          50051,
		FeedServiceAddr:   "localhost:50051",
		ConfigPath:        configPath,
		LogLevel:          "info",
		FeedRetryAttempts: 3,
		FeedRetryDelay:    time.Second,
	}

	app, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = app.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestApp_Run tests server startup
func TestApp_Run(t *testing.T) {
	t.Skip("Requires service-feed to be running")

	configPath := createTestConfig(t)

	cfg := &Config{
		GRPCPort:          50052,
		FeedServiceAddr:   "localhost:50051",
		ConfigPath:        configPath,
		LogLevel:          "info",
		FeedRetryAttempts: 3,
		FeedRetryDelay:    time.Second,
	}

	app, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(ctx, 50052) // Use different port
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel to trigger shutdown
	cancel()

	// Wait for shutdown
	select {
	case err := <-errCh:
		// Context cancellation is expected
		if err != nil && err != context.Canceled {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for server shutdown")
	}
}

// Integration test: Full round trip with mock feed
func TestIntegration_AskRequest(t *testing.T) {
	t.Skip("Requires service-feed to be running")

	// This test would:
	// 1. Create app with test config
	// 2. Make GetAsk request via gRPC client
	// 3. Verify response format and values

	// Example test structure:
	/*
		configPath := createTestConfig(t)
		app, err := New(configPath, "localhost:50051")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server
		go app.Run(ctx, 50053)
		time.Sleep(100 * time.Millisecond)

		// Create gRPC client
		conn, err := grpc.Dial("localhost:50053", grpc.WithInsecure())
		require.NoError(t, err)
		defer conn.Close()

		client := pb.NewDoubleRiseFallServiceClient(conn)

		// Make request
		resp, err := client.GetAsk(context.Background(), &pb.GetAskRequest{
			OptionParameters: &pb.AskOptionParameters{
				Symbol:         "R_100",
				ContractType:   pb.ContractType_CONTRACT_TYPE_RISE,
				Currency:       "USD",
				FirstDuration:  "30s",
				SecondDuration: "60s",
				Stake:          "10.00",
			},
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetAskPrice())
		assert.Equal(t, "USD", resp.GetCurrency())
	*/
}

// Integration test: Bid evaluation
func TestIntegration_BidRequest(t *testing.T) {
	t.Skip("Requires service-feed to be running")

	// This test would:
	// 1. Create app with test config
	// 2. Make GetBid request via gRPC client
	// 3. Verify win/loss evaluation
}

// Integration test: Stream ask updates
func TestIntegration_StreamAsk(t *testing.T) {
	t.Skip("Requires service-feed to be running")

	// This test would:
	// 1. Create app with test config
	// 2. Start StreamAsk
	// 3. Verify multiple updates received
	// 4. Cancel and verify cleanup
}

// Integration test: Health check
func TestIntegration_HealthCheck(t *testing.T) {
	t.Skip("Requires service-feed to be running")

	// This test would:
	// 1. Create app and start server
	// 2. Call gRPC health check
	// 3. Verify SERVING status
}

// Benchmark: Ask calculation throughput
func BenchmarkCalculateAsk(b *testing.B) {
	b.Skip("Requires service-feed to be running")

	// This benchmark would measure:
	// - Ask calculations per second
	// - p50/p99 latency
}

// Benchmark: Bid evaluation throughput
func BenchmarkCalculateBid(b *testing.B) {
	b.Skip("Requires service-feed to be running")

	// This benchmark would measure:
	// - Bid evaluations per second
	// - p50/p99 latency
}
