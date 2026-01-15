# Service Specification: doublerisefall

> **Version**: 1.0.0
> **Created**: 2026-01-15
> **Service ID**: SVC-DRF-P7K
> **Status**: DRAFT

---

## Overview

The `doublerisefall` service is a specialized pricing engine for the Double Rise/Fall digital binary option product. It implements a path-dependent contract pricing model that evaluates spot prices against a barrier at two distinct timestamps (t1 and t2). The service provides real-time and streaming price calculation (Ask) and contract value evaluation (Bid) to the trading gateway.

---

## Tech Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **Language** | Go 1.21+ | High performance, excellent gRPC support, strong type safety, mature concurrency primitives |
| **API Protocol** | gRPC + Protocol Buffers v3 | Type-safe contracts, streaming support, efficient binary serialization |
| **Configuration** | YAML + Viper | Human-readable configuration, hot-reload capability for symbol settings |
| **Logging** | slog (structured) | Standard library package, JSON output, structured logging best practices |
| **Build** | Makefile + buf | Reproducible builds, standardized proto generation |
| **Testing** | Go testing + testify | Native testing framework with assertion library |

---

## User Story Coverage

| Story ID | Summary | Implementation |
|----------|---------|----------------|
| **US-001** | Trader requests contract price for display | [`GetAsk`](../../api/service-pricer-doublerisefall_internal.md) |
| **US-002** | Trader streams live prices during trade setup | [`StreamAsk`](../../api/service-pricer-doublerisefall_internal.md) |
| **US-003** | Trader purchases contract (uses Ask response) | [`GetAsk`](../../api/service-pricer-doublerisefall_internal.md) → external purchase |
| **US-004** | System evaluates contract at expiry | [`GetBid`](../../api/service-pricer-doublerisefall_internal.md) |
| **US-005** | Trader views live contract value | [`StreamBid`](../../api/service-pricer-doublerisefall_internal.md) |
| **US-006** | System validates trade parameters | [`GetAsk`](../../api/service-pricer-doublerisefall_internal.md) validation |
| **US-007** | Trader sees potential payout before purchase | [`GetAsk`](../../api/service-pricer-doublerisefall_internal.md) response.payout |
| **US-008** | System enforces trading limits | [`GetAsk`](../../api/service-pricer-doublerisefall_internal.md) validation |

---

## Architecture Design

### Design Philosophy

The service follows a **light modular architecture** (complexity score 5) with clear separation of concerns while avoiding over-engineering. The design prioritizes:

1. **Single Responsibility**: Each package handles one concern
2. **Dependency Inversion**: Interfaces defined where consumed (in pricer package)
3. **Stateless Design**: All state passed via request, configuration loaded at startup
4. **Clean Boundaries**: gRPC handlers delegate to business logic without data assembly

### Dependency Flow

```
grpcsvc → pricer → (config, feed, contract)
```

The [`grpcsvc`](#grpcsvc---grpc-handlers) package serves as the entry point, delegating to the [`pricer`](#pricer---core-pricing-logic) package which defines interfaces that are implemented by [`config`](#config---symbol-configuration), [`feed`](#feed---market-data-wrapper), and [`contract`](#contract---validation--lifecycle) packages.

### Why This Structure?

| Decision | Rationale |
|----------|-----------|
| **Single service** | One product type = one service (Single Responsibility) |
| **Interfaces in pricer** | Pricer is the consumer; defines what it needs (Interface Segregation) |
| **No shared interfaces package** | Avoids artificial abstraction layer |
| **Thin feed wrapper** | Encapsulates service-feed client, enables testing |
| **Separate contract package** | Duration parsing/validation is distinct from pricing math |

---

## Directory Structure

```
service-pricer-doublerisefall/
├── api/                           # Generated gRPC code (do not edit)
│   └── doublerisefall/
│       ├── doublerisefall.pb.go
│       ├── doublerisefall_grpc.pb.go
│       └── doublerisefall.pb.gw.go
├── cmd/
│   └── doublerisefall/
│       └── main.go                # Application entry point
├── config/
│   └── symbols.yaml               # Symbol-specific configuration
├── internal/
│   ├── app/
│   │   ├── app.go                 # Application initialization
│   │   └── app_test.go
│   ├── grpcsvc/
│   │   ├── grpcsvc.go             # gRPC handlers (entry point)
│   │   └── grpcsvc_test.go
│   ├── pricer/
│   │   ├── pricer.go              # Core pricing logic, defines interfaces
│   │   ├── pricer_test.go
│   │   ├── ask.go                 # Ask price calculation
│   │   └── bid.go                 # Bid price evaluation
│   ├── config/
│   │   ├── config.go              # Configuration loading
│   │   └── config_test.go
│   ├── contract/
│   │   ├── contract.go            # Contract validation
│   │   ├── contract_test.go
│   │   └── duration.go            # Duration parsing
│   ├── feed/
│   │   ├── client.go              # Wrapper for service-feed
│   │   └── client_test.go
│   └── tools/
│       └── tools.go               # Build tool dependencies
├── proto/
│   └── doublerisefall/
│       └── v1/
│           └── doublerisefall.proto  # API definition
├── Dockerfile
├── Makefile
├── buf.yaml
├── buf.gen.yaml
├── go.mod
├── go.sum
└── README.md
```

---

## Module Specifications

### grpcsvc - gRPC Handlers

| Field | Value |
|-------|-------|
| **Package** | `internal/grpcsvc` |
| **Purpose** | Request handling, response formatting, streaming orchestration |
| **Depends On** | `pricer` |

#### File Ownership

| File | Responsibility |
|------|----------------|
| [`grpcsvc.go`](.) | gRPC service implementation, handler methods |
| [`grpcsvc_test.go`](.) | Unit tests with mocked pricer |

#### Core Functionality

**gRPC Service Implementation**:
```go
type Service struct {
    doublerisefall.UnimplementedDoubleRiseFallServiceServer
    pricer *pricer.Pricer
}

func New(p *pricer.Pricer) *Service
```

**Handler Methods**:

| Method | Description | Stream Type |
|--------|-------------|-------------|
| `GetAsk(ctx, *GetAskRequest) (*GetAskResponse, error)` | Single ask price calculation | Unary |
| `StreamAsk(*StreamAskRequest, stream) error` | Real-time ask updates | Server streaming |
| `GetBid(ctx, *GetBidRequest) (*GetBidResponse, error)` | Single bid evaluation | Unary |
| `StreamBid(*StreamBidRequest, stream) error` | Real-time bid updates | Server streaming |

**Streaming Behavior**:

| Contract Type | StreamAsk Trigger | StreamBid Trigger |
|---------------|-------------------|-------------------|
| Time-based | Tick update OR 5s interval | Tick update OR 5s interval |
| Tick-based | Tick update ONLY | Tick update ONLY (NO time-based fallback) |

**Error Mapping**:
- Convert domain errors to gRPC status codes
- Use standard error format: `status.Errorf(codes.InvalidArgument, "ERR-DF-S1K: invalid symbol: %s", symbol)`

#### Interfaces Consumed

```go
// From pricer package
type Pricer interface {
    CalculateAsk(ctx context.Context, req *AskRequest) (*AskResult, error)
    CalculateBid(ctx context.Context, req *BidRequest) (*BidResult, error)
    SubscribeAsk(ctx context.Context, req *AskRequest) (*AskSubscription, error)
    SubscribeBid(ctx context.Context, req *BidRequest) (*BidSubscription, error)
}
```

---

### pricer - Core Pricing Logic

| Field | Value |
|-------|-------|
| **Package** | `internal/pricer` |
| **Purpose** | Fair probability calculation, commission application, payout computation, win/loss evaluation |
| **Depends On** | `config`, `feed`, `contract` (via interfaces) |

#### File Ownership

| File | Responsibility |
|------|----------------|
| [`pricer.go`](.) | Interface definitions, Pricer struct, constructor |
| [`ask.go`](.) | Ask price calculation logic |
| [`bid.go`](.) | Bid price evaluation logic |
| [`pricer_test.go`](.) | Unit tests with mock implementations |

#### Core Functionality

**Interface Definitions** (defined in pricer, implemented by other packages):

```go
// Internal types (NOT proto types)
type Tick struct {
    Symbol string
    Time   int64   // Unix epoch seconds
    Quote  string  // Price as string (preserve precision)
}

type SymbolConfig struct {
    Symbol         string
    CommissionRate float64
    MaxPayout      float64
    MinStake       float64
    Enabled        bool
}

type Duration struct {
    Value    int64
    Unit     string  // "s", "m", "h", "d", "t"
    IsTickBased bool
}

// ConfigProvider provides access to symbol configuration
type ConfigProvider interface {
    GetSymbolConfig(symbol string) (*SymbolConfig, error)
}

// FeedProvider provides access to market data
type FeedProvider interface {
    GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error)
    GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*Tick, bool, error)
    Subscribe(ctx context.Context, symbol string, start int64) *Subscription
    Close() error
}

// ContractValidator validates contract parameters
type ContractValidator interface {
    ValidateAskRequest(ctx context.Context, req *AskRequest) error
    ValidateBidRequest(ctx context.Context, req *BidRequest) error
    ParseDuration(s string) (Duration, error)
}
```

**Pricer Struct**:

```go
type Pricer struct {
    config   ConfigProvider
    feed     FeedProvider
    contract ContractValidator
}

func New(cfg ConfigProvider, feed FeedProvider, contract ContractValidator) *Pricer
```

**Ask Calculation** ([`ask.go`](.)):

1. Validate request via `ContractValidator`
2. Get symbol config via `ConfigProvider`
3. Get current spot via `FeedProvider.GetTickForEpoch`
4. Calculate fair probability using arcsin formula
5. Apply commission markup
6. Calculate payout from stake

**Pricing Formulas**:

| Formula | Expression |
|---------|------------|
| **Correlation** | `ρ = √(t₁/t₂)` |
| **Fair Probability** | `P_fair = 1/4 + arcsin(ρ)/(2π)` |
| **Client Price (Unit)** | `P_client = P_fair + Commission` |
| **Payout** | `Payout = Stake / P_client` |

**Bid Evaluation** ([`bid.go`](.)):

1. Validate request via `ContractValidator`
2. Get entry tick (barrier) at `start_time`
3. Calculate evaluation times: `t1 = start_time + first_duration`, `t2 = start_time + second_duration`
4. Get spot at t1 and t2 via `FeedProvider.GetTickForEpoch`
5. Apply win/loss conditions

**Win/Loss Conditions**:

| Contract Type | Win Condition | Bid Result |
|---------------|---------------|------------|
| RISE | `spot_t1 > barrier AND spot_t2 > barrier` | `payout` |
| RISE | `spot_t1 ≤ barrier OR spot_t2 ≤ barrier` | `0` |
| FALL | `spot_t1 < barrier AND spot_t2 < barrier` | `payout` |
| FALL | `spot_t1 ≥ barrier OR spot_t2 ≥ barrier` | `0` |

**Critical**: If condition fails at t1, contract expires worthless immediately (no need to check t2).

---

### config - Symbol Configuration

| Field | Value |
|-------|-------|
| **Package** | `internal/config` |
| **Purpose** | YAML configuration loading, symbol settings access |
| **Implements** | `pricer.ConfigProvider` |

#### File Ownership

| File | Responsibility |
|------|----------------|
| [`config.go`](.) | Configuration loading, Viper setup, accessor methods |
| [`config_test.go`](.) | Unit tests for config loading and validation |

#### Core Functionality

**Configuration Structure**:

```go
type Config struct {
    Symbols map[string]SymbolConfig
}

type SymbolConfig struct {
    Symbol         string  `yaml:"symbol"`
    CommissionRate float64 `yaml:"commission_rate"`
    MaxPayout      float64 `yaml:"max_payout"`
    MinStake       float64 `yaml:"min_stake"`
    Enabled        bool    `yaml:"enabled"`
}
```

**YAML Schema** ([`config/symbols.yaml`](.)):

```yaml
symbols:
  R_10:
    symbol: R_10
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_25:
    symbol: R_25
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
  R_75:
    symbol: R_75
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
  R_100:
    symbol: R_100
    commission_rate: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
```

**Methods**:

```go
func Load(path string) (*Config, error)
func (c *Config) GetSymbolConfig(symbol string) (*SymbolConfig, error)
func (c *Config) Watch(onChange func())  // Hot-reload support
```

**Error Handling**:
- Return `ErrSymbolNotFound` for unknown symbols
- Return `ErrSymbolDisabled` for disabled symbols
- Validate configuration on load (commission_rate > 0, max_payout > 0, min_stake > 0)

---

### contract - Validation & Lifecycle

| Field | Value |
|-------|-------|
| **Package** | `internal/contract` |
| **Purpose** | Duration parsing, contract parameter validation |
| **Implements** | `pricer.ContractValidator` |

#### File Ownership

| File | Responsibility |
|------|----------------|
| [`contract.go`](.) | Validation logic for Ask and Bid requests |
| [`duration.go`](.) | Duration string parsing |
| [`contract_test.go`](.) | Unit tests for validation and parsing |

#### Core Functionality

**Duration Parsing** ([`duration.go`](.)):

```go
// ParseDuration parses duration strings like "30s", "1m", "2h", "1d", "5t"
func ParseDuration(s string) (Duration, error)

type Duration struct {
    Value       int64
    Unit        string  // "s", "m", "h", "d", "t"
    IsTickBased bool
}

// ToSeconds converts time-based duration to seconds
func (d Duration) ToSeconds() int64

// ToTicks returns tick count for tick-based durations
func (d Duration) ToTicks() int64
```

**Supported Duration Formats**:

| Format | Unit | Examples | Max Value |
|--------|------|----------|-----------|
| Time-based | seconds (s), minutes (m), hours (h), days (d) | "30s", "1m", "2h", "1d" | 1 day (86400s) |
| Tick-based | ticks (t) | "5t", "10t" | 10 ticks |

**Validation Rules** ([`contract.go`](.)):

| Rule | Validation | Error Code |
|------|------------|------------|
| Symbol valid | Symbol in supported list | ERR-DF-S1K |
| Duration format | Parseable duration string | ERR-DF-D1N |
| Duration order | `second_duration > first_duration` | ERR-DF-D2O |
| Duration gap | Gap ≥ 10s (time) or ≥ 2t (tick) | ERR-DF-D3G |
| Stake minimum | `stake >= min_stake` | ERR-DF-K1M |
| Payout maximum | `payout <= max_payout` | ERR-DF-P1X |
| Pricing time | `pricing_time <= now` | ERR-DF-T1F |
| Start time (Bid) | `start_time` provided | ERR-DF-R1S |
| Payout (Bid) | `payout` provided | ERR-DF-R2P |

**Methods**:

```go
type Validator struct {
    config ConfigProvider
}

func NewValidator(cfg ConfigProvider) *Validator
func (v *Validator) ValidateAskRequest(ctx context.Context, req *AskRequest) error
func (v *Validator) ValidateBidRequest(ctx context.Context, req *BidRequest) error
func (v *Validator) ParseDuration(s string) (Duration, error)
```

---

### feed - Market Data Wrapper

| Field | Value |
|-------|-------|
| **Package** | `internal/feed` |
| **Purpose** | Thin wrapper around service-feed client |
| **Implements** | `pricer.FeedProvider` |

#### File Ownership

| File | Responsibility |
|------|----------------|
| [`client.go`](.) | Wrapper implementation around service-feed client |
| [`client_test.go`](.) | Unit tests with mock client |

#### Core Functionality

**⚠️ MANDATORY**: Import and use `github.com/regentmarkets/service-feed/client` - do NOT implement direct gRPC calls.

**Wrapper Implementation**:

```go
import (
    "github.com/regentmarkets/service-feed/client"
)

type Client struct {
    feedClient *client.Client
}

func NewClient(addr string, retryAttempts int, retryDelay time.Duration) (*Client, error) {
    c, err := client.New(addr, retryAttempts, retryDelay)
    if err != nil {
        return nil, fmt.Errorf("create feed client: %w", err)
    }
    return &Client{feedClient: c}, nil
}

func (c *Client) GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error) {
    tick, isFinal, err := c.feedClient.GetTickForEpoch(ctx, symbol, epoch)
    if err != nil {
        return nil, false, err
    }
    if tick == nil {
        return nil, isFinal, nil
    }
    return &Tick{
        Symbol: tick.Symbol,
        Time:   tick.Time.Seconds,
        Quote:  tick.Quote,
    }, isFinal, nil
}

func (c *Client) GetTicksFromLimit(ctx context.Context, symbol string, start int64, limit int64) ([]*Tick, bool, error)
func (c *Client) Subscribe(ctx context.Context, symbol string, start int64) *Subscription
func (c *Client) Close() error
```

**Subscription Wrapper**:

```go
type Subscription struct {
    sub *client.Subscription
    ch  chan *Tick
}

func (s *Subscription) C() <-chan *Tick
func (s *Subscription) Err() error
func (s *Subscription) Close()
```

**Error Handling**:

| Scenario | Handling |
|----------|----------|
| `tick == nil` | Return `nil, isFinal, nil` (no tick at epoch) |
| `isFinal == false` | Data may be incomplete; caller decides handling |
| Connection error | Return wrapped error for retry |

---

### app - Application Initialization

| Field | Value |
|-------|-------|
| **Package** | `internal/app` |
| **Purpose** | Wire up dependencies, start gRPC server |

#### File Ownership

| File | Responsibility |
|------|----------------|
| [`app.go`](.) | Dependency injection, server lifecycle |
| [`app_test.go`](.) | Integration tests |

#### Core Functionality

```go
type App struct {
    cfg      *config.Config
    feed     *feed.Client
    pricer   *pricer.Pricer
    grpcSvc  *grpcsvc.Service
    server   *grpc.Server
}

func New(configPath, feedAddr string) (*App, error) {
    // 1. Load configuration
    cfg, err := config.Load(configPath)
    
    // 2. Create feed client
    feedClient, err := feed.NewClient(feedAddr, 3, time.Second)
    
    // 3. Create contract validator
    validator := contract.NewValidator(cfg)
    
    // 4. Create pricer
    pricer := pricer.New(cfg, feedClient, validator)
    
    // 5. Create gRPC service
    grpcSvc := grpcsvc.New(pricer)
    
    return &App{...}, nil
}

func (a *App) Run(ctx context.Context, port int) error
func (a *App) Shutdown(ctx context.Context) error
```

---

## Data Architecture

### Data Models

The service uses request-scoped data only. No persistent storage.

**Request Types** (internal, converted from proto):

```go
type AskRequest struct {
    Symbol         string
    ContractType   ContractType
    Currency       string
    FirstDuration  string
    SecondDuration string
    Stake          string
    PricingTime    int64  // Optional
}

type BidRequest struct {
    Symbol         string
    ContractType   ContractType
    Currency       string
    FirstDuration  string
    SecondDuration string
    StartTime      int64   // Required
    Stake          string
    Payout         string  // Required
    PricingTime    int64   // Optional
}
```

**Result Types**:

```go
type AskResult struct {
    AskPrice        string
    Currency        string
    CurrentSpot     string
    CurrentSpotTime int64
    Payout          string
    MaxPayout       string
    MinStake        string
}

type BidResult struct {
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
    Currency        string
    EvaluationTime  int64
}
```

### Persistence Strategy

| Data Type | Strategy |
|-----------|----------|
| **Symbol Configuration** | Local YAML file, loaded at startup |
| **Contract State** | Not persisted (owned by trading gateway) |
| **Market Data** | Not persisted (fetched from service-feed) |

### Data Access Patterns

1. **Ask Flow**: Read config → Fetch current spot → Calculate → Return
2. **Bid Flow**: Read config → Fetch entry tick → Fetch spots at t1, t2 → Evaluate → Return
3. **Stream Flow**: Subscribe to ticks → Calculate on each tick → Stream to client

---

## Security Architecture

### Authentication

| Field | Value |
|-------|-------|
| **Service-to-Service Auth** | None (internal network) |
| **Rationale** | Consumed only by api-gateway-trading within Kubernetes cluster |

### Network Security

| Aspect | Configuration |
|--------|---------------|
| **Network** | Internal Kubernetes cluster network |
| **TLS** | Optional (mTLS recommended for production) |
| **Access Control** | Kubernetes NetworkPolicy restricts access to api-gateway-trading |

### API Security

| Measure | Implementation |
|---------|----------------|
| **Input Validation** | Contract package validates all inputs |
| **Rate Limiting** | Not implemented (handled by gateway) |
| **Request Size** | gRPC default limits |

### Data Protection

| Aspect | Implementation |
|--------|----------------|
| **Sensitive Data** | None persisted |
| **Logging** | No PII logged |
| **Configuration** | No secrets in symbols.yaml |

---

## Performance & Scalability

### Performance Requirements

| Metric | Target | Notes |
|--------|--------|-------|
| GetAsk latency (p50) | < 20ms | Excluding network |
| GetAsk latency (p99) | < 50ms | Excluding network |
| StreamAsk update latency | < 100ms | From tick to client |
| GetBid latency (p99) | < 100ms | May require multiple feed calls |

### Scalability Strategy

| Aspect | Strategy |
|--------|----------|
| **Horizontal Scaling** | Stateless design enables multiple replicas |
| **Concurrent Streams** | 1000 per instance (goroutine-per-stream) |
| **Memory Target** | < 500MB per instance |
| **CPU** | Linear scaling with request volume |

### Caching Strategy

| Data | Caching |
|------|---------|
| **Symbol Config** | Loaded once at startup, hot-reload via Viper watch |
| **Market Data** | No caching (service-feed handles caching) |
| **Pricing Results** | No caching (real-time calculation required) |

### Resource Management

| Resource | Management |
|----------|------------|
| **gRPC Connections** | Connection pooling to service-feed |
| **Goroutines** | Context cancellation for cleanup |
| **Memory** | No tick caching, stream cleanup on close |

---

## Error Handling & Resilience

### Error Categories

| Category | gRPC Status | Handling |
|----------|-------------|----------|
| **Validation Errors** | INVALID_ARGUMENT | Return immediately, no retry |
| **Precondition Errors** | FAILED_PRECONDITION | May retry after condition met |
| **Market Data Errors** | UNAVAILABLE | Retry with backoff |
| **Internal Errors** | INTERNAL | Log, alert, may retry |

### Error Code Reference

| Error Code | gRPC Status | Description |
|------------|-------------|-------------|
| ERR-DF-S1K | INVALID_ARGUMENT | Invalid symbol |
| ERR-DF-S2D | FAILED_PRECONDITION | Symbol disabled |
| ERR-DF-D1N | INVALID_ARGUMENT | Invalid duration format |
| ERR-DF-D2O | INVALID_ARGUMENT | Duration order invalid |
| ERR-DF-D3G | INVALID_ARGUMENT | Duration gap invalid |
| ERR-DF-K1M | INVALID_ARGUMENT | Stake below minimum |
| ERR-DF-P1X | INVALID_ARGUMENT | Payout exceeds maximum |
| ERR-DF-T1F | INVALID_ARGUMENT | Pricing time in future |
| ERR-DF-R1S | INVALID_ARGUMENT | Missing start_time |
| ERR-DF-R2P | INVALID_ARGUMENT | Missing payout |
| ERR-DF-E1M | FAILED_PRECONDITION | Missing entry tick |
| ERR-DF-E2T | FAILED_PRECONDITION | Missing evaluation tick |
| ERR-DF-M1E | UNAVAILABLE | Market data unavailable |
| ERR-DF-M2R | UNAVAILABLE | Market stream interrupted |
| ERR-DF-I1X | INTERNAL | Internal server error |

### Retry Strategy

| Error Type | Client Retry | Backoff |
|------------|--------------|---------|
| INVALID_ARGUMENT | No | N/A |
| FAILED_PRECONDITION | Wait for condition | N/A |
| UNAVAILABLE | Yes, max 3 | Exponential (1s, 2s, 4s) |
| INTERNAL | Yes, with caution | Linear (1s) |

### Circuit Breaker

| Dependency | Strategy |
|------------|----------|
| **service-feed** | service-feed client has built-in retry logic |
| **Failure Threshold** | 3 consecutive failures |
| **Recovery** | Automatic reconnection by client |

### Monitoring Points

| Metric | Purpose |
|--------|---------|
| `pricer_ask_latency_seconds` | Ask calculation latency |
| `pricer_bid_latency_seconds` | Bid evaluation latency |
| `pricer_stream_count` | Active stream count |
| `pricer_error_total` | Error count by code |
| `feed_request_total` | Feed requests by method |
| `feed_error_total` | Feed errors |

---

## External Dependencies

### Services

| Service | Purpose | Protocol | Criticality |
|---------|---------|----------|-------------|
| **service-feed** | Market data (spot prices, tick history) | gRPC | Critical |

### Dependency Details

| Aspect | Value |
|--------|-------|
| **Repository** | `github.com/regentmarkets/service-feed` |
| **Client Package** | `github.com/regentmarkets/service-feed/client` |
| **Version Compatibility** | v1.x |
| **Integration Pattern** | Thin wrapper in `internal/feed` |

### Required Capabilities from service-feed

| Method | Signature | Use Case |
|--------|-----------|----------|
| `GetTickForEpoch` | `(ctx, symbol, epoch) (*Tick, bool, error)` | Entry tick, spot at t1/t2 |
| `GetTicksFromLimit` | `(ctx, symbol, start, limit) ([]*Tick, bool, error)` | Tick-based evaluation |
| `Subscribe` | `(ctx, symbol, start) *Subscription` | Real-time streaming |

### Key Libraries

| Library | Version | Purpose |
|---------|---------|---------|
| `google.golang.org/grpc` | v1.60+ | gRPC framework |
| `google.golang.org/protobuf` | v1.32+ | Protocol Buffers |
| `github.com/spf13/viper` | v1.18+ | Configuration |
| `github.com/stretchr/testify` | v1.8+ | Testing assertions |

---

## Deployment Requirements

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `GRPC_PORT` | gRPC server port | 50051 | No |
| `FEED_SERVICE_ADDR` | service-feed address | service-feed:50051 | Yes |
| `CONFIG_PATH` | Path to symbols.yaml | /config/symbols.yaml | No |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info | No |

### Health Check Endpoint

| Field | Value |
|-------|-------|
| **Protocol** | gRPC Health Check Protocol |
| **Service** | `grpc.health.v1.Health` |
| **Endpoint** | `/grpc.health.v1.Health/Check` |
| **Response** | `SERVING`, `NOT_SERVING`, `UNKNOWN` |

### Startup Dependencies

| Dependency | Requirement |
|------------|-------------|
| **service-feed** | Must be running and reachable |

### Database Requirements

None - stateless service.

### Container Configuration

**Dockerfile**:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /doublerisefall ./cmd/doublerisefall

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /doublerisefall .
COPY config/symbols.yaml /config/symbols.yaml
EXPOSE 50051
ENTRYPOINT ["/app/doublerisefall"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: service-pricer-doublerisefall
spec:
  replicas: 2
  selector:
    matchLabels:
      app: service-pricer-doublerisefall
  template:
    metadata:
      labels:
        app: service-pricer-doublerisefall
    spec:
      containers:
        - name: doublerisefall
          image: service-pricer-doublerisefall:latest
          ports:
            - containerPort: 50051
          env:
            - name: FEED_SERVICE_ADDR
              value: "service-feed:50051"
            - name: CONFIG_PATH
              value: "/config/symbols.yaml"
            - name: LOG_LEVEL
              value: "info"
          readinessProbe:
            grpc:
              port: 50051
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            grpc:
              port: 50051
            initialDelaySeconds: 10
            periodSeconds: 20
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
```

---

## Development Guidelines

### Code Organization

| Principle | Implementation |
|-----------|----------------|
| **Package Naming** | Lowercase, single word (e.g., `pricer`, `grpcsvc`) |
| **Interface Location** | Defined where consumed (in `pricer`) |
| **Error Handling** | Wrap errors with context, return domain errors |
| **Logging** | Use structured logging (slog) with request ID |

### Testing Strategy

#### Test Coverage Targets

| Package | Coverage Target |
|---------|-----------------|
| `pricer` | 90% (business logic) |
| `contract` | 95% (validation) |
| `grpcsvc` | 80% (handler logic) |
| `config` | 85% (loading/access) |
| `feed` | 70% (wrapper) |

#### Test Cases

| Test ID | Package | Description |
|---------|---------|-------------|
| TC-DF-A1B | pricer | Calculate ask for RISE contract with valid parameters |
| TC-DF-A2C | pricer | Calculate ask for FALL contract with valid parameters |
| TC-DF-A3D | pricer | Reject ask with stake below minimum |
| TC-DF-A4E | pricer | Reject ask with payout exceeding maximum |
| TC-DF-B1F | pricer | Evaluate bid for winning RISE contract |
| TC-DF-B2G | pricer | Evaluate bid for losing RISE contract (fail at t1) |
| TC-DF-B3H | pricer | Evaluate bid for losing RISE contract (fail at t2) |
| TC-DF-B4J | pricer | Evaluate bid for winning FALL contract |
| TC-DF-D1K | contract | Parse valid time-based durations |
| TC-DF-D2L | contract | Parse valid tick-based durations |
| TC-DF-D3M | contract | Reject invalid duration format |
| TC-DF-D4N | contract | Reject duration order violation |
| TC-DF-D5P | contract | Reject duration gap violation |
| TC-DF-C1Q | config | Load valid configuration |
| TC-DF-C2R | config | Return error for unknown symbol |
| TC-DF-C3S | config | Return error for disabled symbol |
| TC-DF-G1T | grpcsvc | Handle GetAsk request |
| TC-DF-G2U | grpcsvc | Handle GetBid request |
| TC-DF-G3V | grpcsvc | Map validation error to INVALID_ARGUMENT |
| TC-DF-G4W | grpcsvc | Map feed error to UNAVAILABLE |

#### Testing with Mocks

```go
// Example mock for FeedProvider
type MockFeedProvider struct {
    mock.Mock
}

func (m *MockFeedProvider) GetTickForEpoch(ctx context.Context, symbol string, epoch int64) (*Tick, bool, error) {
    args := m.Called(ctx, symbol, epoch)
    tick, _ := args.Get(0).(*Tick)
    return tick, args.Bool(1), args.Error(2)
}
```

### Deployment Model

| Aspect | Configuration |
|--------|---------------|
| **Containerization** | Docker |
| **Orchestration** | Kubernetes |
| **Replicas** | 2+ for HA |
| **Rolling Update** | maxSurge=1, maxUnavailable=0 |

### Configuration Management

| Environment | Config Source |
|-------------|---------------|
| Development | Local `config/symbols.yaml` |
| Staging | ConfigMap mounted at `/config` |
| Production | ConfigMap with hot-reload |

---

## Implementation Order

### Phase 1: Foundation (Week 1)

| Order | Component | Deliverable |
|-------|-----------|-------------|
| 1 | Project Setup | Generate service template, configure go.mod |
| 2 | Proto Definition | Define proto files, generate Go code |
| 3 | Config Package | YAML loading, SymbolConfig struct, validation |
| 4 | Feed Package | Wrapper around service-feed/client |

### Phase 2: Core Logic (Week 1-2)

| Order | Component | Deliverable |
|-------|-----------|-------------|
| 5 | Contract Package | Duration parsing, request validation |
| 6 | Pricer Package | Interface definitions |
| 7 | Pricer - Ask | Fair probability, commission, payout calculation |
| 8 | Pricer - Bid | Win/loss evaluation logic |

### Phase 3: API Layer (Week 2)

| Order | Component | Deliverable |
|-------|-----------|-------------|
| 9 | gRPC Handlers | GetAsk, GetBid handlers |
| 10 | gRPC Streaming | StreamAsk, StreamBid handlers |
| 11 | Error Mapping | Domain errors to gRPC codes |
| 12 | App Package | Dependency wiring, server startup |

### Phase 4: Integration (Week 2-3)

| Order | Component | Deliverable |
|-------|-----------|-------------|
| 13 | Unit Tests | Test all packages |
| 14 | Integration Tests | End-to-end with mock feed |
| 15 | Performance Tests | Benchmarks for latency targets |
| 16 | Deployment | Dockerfile, Kubernetes manifests |

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-15 | Initial service specification |

---

## Quality Checklist

- [x] Service purpose and boundaries are clear
- [x] Tech stack is appropriate and justified
- [x] Internal architecture is well-reasoned
- [x] Directory structure reflects the design
- [x] All major components are specified
- [x] Data architecture is comprehensive
- [x] Security measures are defined
- [x] Performance requirements are addressed
- [x] Error handling strategies are clear
- [x] Development guidelines match the architecture
- [x] External dependencies are documented
- [x] Implementation order is logical
- [x] Deployment requirements are specified (env vars, health checks, dependencies)
