# Service Feed Dependency Specification

> **Repository**: `git@github.com:junbon-deriv/service-feed.git`
> **Proto File**: `proto/grpcfeed/v1/ticks.proto`
> **Preferred Client**: `client/client.go`

---

### 1. Private Repository Configuration
**CRITICAL**: Configure Git and Go for private repositories before any operations:

```bash
# Configure Go to treat these as private (bypass proxy.golang.org)
export GOPRIVATE=github.com/junbon-deriv,github.com/regentmarkets

# Configure Git to use SSH instead of HTTPS for GitHub
git config --global url."git@github.com:".insteadOf "https://github.com/"
```


## 2. Integration Requirement

### ⚠️ MANDATORY: Use the Preferred Client

**DO NOT** implement direct gRPC client calls. Instead:

1. **Import the existing client**: `github.com/regentmarkets/service-feed/client`
2. **Create a wrapper** around the client for clean dependency separation
3. **Use the client's methods** (not raw proto methods)

### Why Use the Preferred Client?

The [`client/client.go`](../code/service-feed/client/client.go) provides:
- Built-in retry logic with configurable attempts and backoff
- Automatic reconnection for streaming subscriptions
- Convenient epoch-based methods (no timestamp conversion needed)
- Proper error handling and `is_final` flag management

---

## 3. Available Client Methods

### 3.1 Client Initialization

```go
import "github.com/regentmarkets/service-feed/client"

// Create client with retry policy
feedClient, err := client.New(
    "feed-service:50051",  // address
    3,                      // retryAttempts
    time.Second,            // retryDelay
)
defer feedClient.Close()
```

### 3.2 Single Tick Retrieval

| Method | Signature | Use Case |
|--------|-----------|----------|
| `GetTickForEpoch` | `(ctx, symbol, epoch int64) (*Tick, bool, error)` | Get tick at or after specific time |

**Usage**: Entry tick, spot at evaluation time (t1, t2)

```go
// Get entry tick (first tick after start_time)
tick, isFinal, err := feedClient.GetTickForEpoch(ctx, "R_100", startTimeEpoch)
```

### 3.3 Multiple Ticks Retrieval

| Method | Signature | Use Case |
|--------|-----------|----------|
| `GetTicksForInterval` | `(ctx, symbol, start, end int64) ([]*Tick, bool, error)` | Get ticks in time range |
| `GetTicksFromLimit` | `(ctx, symbol, start, limit int64) ([]*Tick, bool, error)` | Get N ticks from time |

**Usage**: Historical data, tick-based contract evaluation

```go
// Get ticks for interval
ticks, isFinal, err := feedClient.GetTicksForInterval(ctx, "R_100", startEpoch, endEpoch)

// Get specific number of ticks
ticks, isFinal, err := feedClient.GetTicksFromLimit(ctx, "R_100", startEpoch, 10)
```

### 3.4 Real-Time Streaming

| Method | Signature | Use Case |
|--------|-----------|----------|
| `Subscribe` | `(ctx, symbol, start int64) *Subscription` | Real-time tick stream |

**Usage**: StreamAsk, StreamBid operations

```go
// Subscribe to real-time ticks
sub := feedClient.Subscribe(ctx, "R_100", time.Now().Unix())
defer sub.Close()

// Read from channel
for tick := range sub.C() {
    // Process tick
    price := tick.Quote
    timestamp := tick.Time.Seconds
}

// Check for errors
if err := sub.Err(); err != nil {
    // Handle subscription error
}
```

---

## 4. Wrapper Implementation Pattern

### 4.1 Define Interface Where Consumed

```go
// In your pricing package (NOT in a separate interfaces package)
package pricing

// FeedClient defines the market data operations needed by the pricer
type FeedClient interface {
    GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error)
    GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*Tick, bool, error)
    Subscribe(ctx context.Context, symbol string, start int64) *Subscription
    Close() error
}
```

### 4.2 Create Thin Wrapper

```go
// In your feed package
package feed

import (
    "github.com/regentmarkets/service-feed/client"
)

// Wrapper wraps the service-feed client
type Wrapper struct {
    client *client.Client
}

func NewWrapper(addr string) (*Wrapper, error) {
    c, err := client.New(addr, 3, time.Second)
    if err != nil {
        return nil, err
    }
    return &Wrapper{client: c}, nil
}

// Implement the FeedClient interface by delegating to the underlying client
func (w *Wrapper) GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error) {
    return w.client.GetTickForEpoch(ctx, symbol, epoch)
}
// ... other methods
```

---

## 5. Error Handling

### 5.1 Handle `is_final` Flag

The client methods return an `is_final` boolean. When `is_final=false`:
- The server may not have all data for the requested range
- Make additional requests if complete data is required

```go
tick, isFinal, err := feedClient.GetTickForEpoch(ctx, symbol, epoch)
if err != nil {
    return fmt.Errorf("market data error: %w", err)
}
if tick == nil {
    return errors.New("no tick found at specified time")
}
if !isFinal {
    // Data may be incomplete - handle according to business logic
}
```

### 5.2 Subscription Errors

```go
sub := feedClient.Subscribe(ctx, symbol, start)
// ... use subscription
if err := sub.Err(); err != nil {
    // Subscription closed due to error (e.g., max retries exceeded)
}
```

---

## 6. Tick Data Structure

```go
type Tick struct {
    Symbol string                    // e.g., "R_100"
    Time   *timestamppb.Timestamp    // Use .Seconds for epoch
    Quote  string                    // Price as string (preserve precision)
}
```

---

## 7. DO NOT

- ❌ Do NOT implement direct gRPC calls to `TickService`
- ❌ Do NOT create your own retry/reconnection logic
- ❌ Do NOT convert timestamps manually (client handles this)
- ❌ Do NOT ignore the `is_final` flag in responses

## 8. DO

- ✅ Import and use `github.com/regentmarkets/service-feed/client`
- ✅ Create a thin wrapper for dependency isolation
- ✅ Define interfaces where they are consumed (in pricing package)
- ✅ Handle `is_final=false` appropriately
- ✅ Use `GetTickForEpoch` for single tick retrieval (not `Subscribe`)
