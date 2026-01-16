// Package feed provides a wrapper for the service-feed client.
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
	inner *client.Client
}

// NewClient creates a new feed client wrapper.
func NewClient(addr string) (*Client, error) {
	c, err := client.New(addr, 3, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create feed client: %w", err)
	}
	return &Client{inner: c}, nil
}

// GetTickForEpoch retrieves the tick at or after the specified epoch.
func (c *Client) GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*pricer.Tick, bool, error) {
	tick, isFinal, err := c.inner.GetTickForEpoch(ctx, symbol, epoch)
	if err != nil {
		return nil, false, fmt.Errorf("feed error: %w", err)
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

// GetTicksFromLimit retrieves N ticks starting from the specified epoch.
func (c *Client) GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*pricer.Tick, bool, error) {
	ticks, isFinal, err := c.inner.GetTicksFromLimit(ctx, symbol, start, limit)
	if err != nil {
		return nil, false, fmt.Errorf("feed error: %w", err)
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

// Subscribe creates a real-time tick subscription.
func (c *Client) Subscribe(ctx context.Context, symbol string, start int64) pricer.Subscription {
	sub := c.inner.Subscribe(ctx, symbol, start)
	return &subscription{
		inner: sub,
		ch:    make(chan *pricer.Tick, 10),
		ctx:   ctx,
	}
}

// Close closes the feed client connection.
func (c *Client) Close() error {
	return c.inner.Close()
}

// subscription wraps the feed subscription.
type subscription struct {
	inner *client.Subscription
	ch    chan *pricer.Tick
	ctx   context.Context
	once  bool
}

// C returns the channel for receiving ticks.
func (s *subscription) C() <-chan *pricer.Tick {
	if !s.once {
		s.once = true
		go s.forward()
	}
	return s.ch
}

// Err returns any error that occurred during subscription.
func (s *subscription) Err() error {
	return s.inner.Err()
}

// Close closes the subscription.
func (s *subscription) Close() {
	s.inner.Close()
}

// forward converts ticks from the inner subscription to our domain type.
func (s *subscription) forward() {
	defer close(s.ch)
	for {
		select {
		case <-s.ctx.Done():
			return
		case tick, ok := <-s.inner.C():
			if !ok {
				return
			}
			domainTick := &pricer.Tick{
				Symbol: tick.Symbol,
				Time:   tick.Time.Seconds,
				Quote:  tick.Quote,
			}
			select {
			case s.ch <- domainTick:
			case <-s.ctx.Done():
				return
			}
		}
	}
}
