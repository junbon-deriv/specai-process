package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/regentmarkets/service-feed/client"
	"github.com/regentmarkets/service-pricer-doublerisefall/internal/pricer"
)

// Client wraps the service-feed client.
type Client struct {
	feedClient *client.Client
}

// NewClient creates a new feed client connected to the specified address.
func NewClient(addr string, retryAttempts int, retryDelay time.Duration) (*Client, error) {
	c, err := client.New(addr, retryAttempts, retryDelay)
	if err != nil {
		return nil, fmt.Errorf("create feed client: %w", err)
	}

	return &Client{
		feedClient: c,
	}, nil
}

// GetTickForEpoch retrieves a tick for a specific epoch.
// Returns the tick, is_final flag, and error.
func (c *Client) GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*pricer.Tick, bool, error) {
	tick, isFinal, err := c.feedClient.GetTickForEpoch(ctx, symbol, epoch)
	if err != nil {
		return nil, false, fmt.Errorf("get tick for epoch: %w", err)
	}

	if tick == nil {
		return nil, isFinal, nil
	}

	return &pricer.Tick{
		Symbol: tick.Symbol,
		Time:   tick.Time.Seconds,
		Quote:  tick.Quote,
	}, isFinal, nil
}

// GetTicksFromLimit retrieves N ticks starting from a specific epoch.
// Returns the ticks, is_final flag, and error.
func (c *Client) GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*pricer.Tick, bool, error) {
	ticks, isFinal, err := c.feedClient.GetTicksFromLimit(ctx, symbol, start, limit)
	if err != nil {
		return nil, false, fmt.Errorf("get ticks from limit: %w", err)
	}

	if ticks == nil {
		return nil, isFinal, nil
	}

	result := make([]*pricer.Tick, 0, len(ticks))
	for _, t := range ticks {
		result = append(result, &pricer.Tick{
			Symbol: t.Symbol,
			Time:   t.Time.Seconds,
			Quote:  t.Quote,
		})
	}

	return result, isFinal, nil
}

// Subscribe creates a real-time subscription to tick updates.
// Returns a Subscription that provides a channel of ticks.
func (c *Client) Subscribe(ctx context.Context, symbol string, start int64) *pricer.Subscription {
	sub := c.feedClient.Subscribe(ctx, symbol, start)

	// Create a channel to forward ticks
	tickChan := make(chan *pricer.Tick, 10)

	// Start goroutine to convert and forward ticks
	go func() {
		defer close(tickChan)
		for tick := range sub.C() {
			select {
			case <-ctx.Done():
				return
			case tickChan <- &pricer.Tick{
				Symbol: tick.Symbol,
				Time:   tick.Time.Seconds,
				Quote:  tick.Quote,
			}:
			}
		}
	}()

	return &pricer.Subscription{
		C:   tickChan,
		Err: sub.Err(),
	}
}

// Close closes the underlying feed client connection.
func (c *Client) Close() error {
	if c.feedClient != nil {
		return c.feedClient.Close()
	}
	return nil
}
