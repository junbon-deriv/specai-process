package market

import (
	"context"
	"fmt"

	"github.com/regentmarkets/service-feed/client"
	"github.com/shopspring/decimal"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// Feed implements pricing.MarketDataProvider interface
// This wraps the service-feed client
type Feed struct {
	client *client.Client
}

// NewFeed creates a new Feed client
func NewFeed(feedClient *client.Client) *Feed {
	return &Feed{
		client: feedClient,
	}
}

// GetLatestTick implements pricing.MarketDataProvider
func (f *Feed) GetLatestTick(ctx context.Context, symbol string) (*pricing.Tick, error) {
	// Use the GetTicksFromLimit method to get the latest tick
	// by requesting 1 tick from current time
	ticks, _, err := f.client.GetTicksFromLimit(ctx, symbol, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("service-feed unavailable: %w", err)
	}

	if len(ticks) == 0 {
		return nil, fmt.Errorf("no tick data available for symbol %s", symbol)
	}

	tick := ticks[0]
	price, err := decimal.NewFromString(tick.Quote)
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	return &pricing.Tick{
		Symbol:    symbol,
		Price:     price,
		Timestamp: tick.Time.Seconds,
	}, nil
}

// GetTick retrieves tick at or after specific timestamp
func (f *Feed) GetTick(ctx context.Context, symbol string, timestamp int64) (*pricing.Tick, error) {
	// Use GetTickForEpoch to get the tick at or after the specified timestamp
	tick, _, err := f.client.GetTickForEpoch(ctx, symbol, timestamp)
	if err != nil {
		return nil, fmt.Errorf("service-feed unavailable: %w", err)
	}

	if tick == nil {
		return nil, fmt.Errorf("no tick found at timestamp %d for symbol %s", timestamp, symbol)
	}

	price, err := decimal.NewFromString(tick.Quote)
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	return &pricing.Tick{
		Symbol:    symbol,
		Price:     price,
		Timestamp: tick.Time.Seconds,
	}, nil
}

// GetTicksInRange retrieves all ticks in a time range (for tick counting)
func (f *Feed) GetTicksInRange(ctx context.Context, symbol string, from, to int64) ([]*pricing.Tick, error) {
	// Calculate a reasonable limit based on time range
	// For tick-based contracts (max 10 ticks), we'll request up to 50 ticks
	// to ensure we capture enough data even with high frequency ticks
	limit := int64(50)

	// Use GetTicksFromLimit to get ticks starting from 'from' timestamp
	ticks, _, err := f.client.GetTicksFromLimit(ctx, symbol, from, limit)
	if err != nil {
		return nil, fmt.Errorf("service-feed unavailable: %w", err)
	}

	result := make([]*pricing.Tick, 0, len(ticks))
	for _, tick := range ticks {
		// Only include ticks within the range
		if tick.Time.Seconds < from || tick.Time.Seconds > to {
			continue
		}

		price, err := decimal.NewFromString(tick.Quote)
		if err != nil {
			continue // Skip invalid ticks
		}

		result = append(result, &pricing.Tick{
			Symbol:    symbol,
			Price:     price,
			Timestamp: tick.Time.Seconds,
		})
	}

	return result, nil
}

// StreamTicks returns a channel of tick updates
func (f *Feed) StreamTicks(ctx context.Context, symbol string) (<-chan *pricing.Tick, error) {
	// Subscribe to tick stream starting from current time (0)
	sub := f.client.Subscribe(ctx, symbol, 0)

	ch := make(chan *pricing.Tick, 100)
	go func() {
		defer close(ch)
		for {
			select {
			case tick, ok := <-sub.C():
				if !ok {
					// Channel closed, check for errors
					if err := sub.Err(); err != nil {
						// Log error but don't send on channel
						return
					}
					return
				}

				price, err := decimal.NewFromString(tick.Quote)
				if err != nil {
					// Skip invalid ticks
					continue
				}

				pricingTick := &pricing.Tick{
					Symbol:    symbol,
					Price:     price,
					Timestamp: tick.Time.Seconds,
				}

				select {
				case ch <- pricingTick:
				case <-ctx.Done():
					sub.Close()
					return
				}
			case <-ctx.Done():
				sub.Close()
				return
			}
		}
	}()

	return ch, nil
}
