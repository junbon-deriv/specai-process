# Service Template Generation Guide

This document provides a precise, step-by-step guide for generating Go-based gRPC services using the [`go-templates`](https://github.com/junbon-deriv/go-templates) tool. This process was successfully executed for the `derivatives` service and should be used as the standard approach for all future service development.

---

## Overview

The service template tool creates a production-ready Go service with:
- gRPC server with Protocol Buffers
- HTTP gateway (grpc-gateway) for REST compatibility
- Structured logging with slog
- Configuration management with Cobra and Viper
- Built-in testing infrastructure
- CI/CD ready Makefile

---

## Prerequisites

### 1. System Requirements
- Go 1.21 or higher installed
- Git configured with SSH access to private repositories
- Access to `github.com/junbon-deriv` and `github.com/regentmarkets` organizations

### 2. Required Go Tools
Install the following tools before running the template generator:

```bash
# Protobuf code generators
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
```

### 3. Private Repository Configuration
**CRITICAL**: Configure Git and Go for private repositories before any operations:

```bash
# Configure Go to treat these as private (bypass proxy.golang.org)
export GOPRIVATE=github.com/junbon-deriv,github.com/regentmarkets

# Configure Git to use SSH instead of HTTPS for GitHub
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

**Verification**:
```bash
# Test SSH access
ssh -T git@github.com

# Should output: Hi <username>! You've successfully authenticated...
```

---

## Service Template Generation Process

### Step 1: Setup Working Directory
Create a dedicated workspace for code generation:

```bash
# From project root
mkdir -p workspace/code
cd workspace/code
```

### Step 2: Clone go-templates Repository
Clone the template repository:

```bash
git clone git@github.com:junbon-deriv/go-templates.git
cd go-templates
```

### Step 3: Install go-templates Tool
Build and install the template generator:

```bash
# Ensure private repo access is configured
export GOPRIVATE=github.com/junbon-deriv,github.com/regentmarkets
git config --global url."git@github.com:".insteadOf "https://github.com/"

# Install the tool to ~/go/bin
go install
```

### Step 4: Generate Service Template
Navigate back to the workspace directory and generate the service:

```bash
cd ..  # Back to workspace/code

# Add Go bin to PATH for protoc plugins
export PATH=$PATH:~/go/bin

# Generate service (replace SERVICE_NAME with your service name)
~/go/bin/go-templates \
  --template service \
  --module-path github.com/regentmarkets/service-pricer-${SERVICE_NAME} \
  --module-name ${SERVICE_NAME}
```

**Example for derivatives service**:
```bash
~/go/bin/go-templates \
  --template service \
  --module-path github.com/regentmarkets/service-pricer-derivatives \
  --module-name derivatives
```

**Expected Output**: The tool will create a complete service structure at [`service-pricer-${SERVICE_NAME}`](workspace/code/service-pricer-derivatives) directory.

---

## Generated Service Structure

The template creates the following structure:

```
service-pricer-derivatives/
├── .clinerules              # CLI tool rules/config
├── .gitignore              # Git ignore patterns
├── .golangci.yml           # Go linting configuration
├── buf.gen.yaml            # Buf code generation config
├── buf.yaml                # Buf proto linting config
├── Dockerfile              # Container build definition
├── go.mod                  # Go module definition
├── go.sum                  # Go dependencies checksums
├── Makefile               # Build automation
├── README.md              # Service documentation
├── api/                   # Generated gRPC code (do not edit manually)
│   └── derivatives/
│       ├── derivatives.pb.go         # Protobuf messages
│       ├── derivatives_grpc.pb.go    # gRPC server/client
│       └── derivatives.pb.gw.go      # HTTP gateway
├── cmd/                   # Application entrypoints
│   └── derivatives/
│       └── main.go        # Main application entry
├── internal/              # Private application code
│   ├── app/
│   │   ├── app.go         # Application initialization
│   │   └── app_test.go    # Application tests
│   ├── grpcsvc/
│   │   ├── grpcsvc.go     # gRPC service implementation
│   │   └── grpcsvc_test.go
│   └── tools/
│       └── tools.go       # Build tools dependencies
└── proto/                 # Protocol buffer definitions
    └── derivatives/
        └── v1/
            └── derivatives.proto  # Service API definition
```

---

## Common Issues and Solutions

### Issue: `protoc-gen-grpc-gateway: executable file not found`
**Solution**: Ensure Go bin is in PATH:
```bash
export PATH=$PATH:~/go/bin
```

### Issue: `Repository not found` when cloning
**Solution**: Verify SSH access and organization permissions:
```bash
ssh -T git@github.com
```

### Issue: `go mod download` fails for private repos
**Solution**: Configure GOPRIVATE:
```bash
export GOPRIVATE=github.com/junbon-deriv,github.com/regentmarkets
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

### Issue: Service directory already exists
**Solution**: Remove and regenerate:
```bash
rm -rf service-pricer-${SERVICE_NAME}
# Run go-templates command again
```

---


## Options Product-Specific Patterns

### Pricing Service Pattern
For every options product (e.g., Double Rise/Fall, Digital Call/Put):

1. **Single Responsibility**: One service per product type
2. **Pricing Module**: Implement fair probability and commission logic
3. **Contract Module**: Duration validation, parameter validation
4. **Configuration Module**: Symbol-specific settings
5. **MarketData Module**: A wrapper for market data dependency. (E.g. service-feed)

### Contract Lifecycle Pattern
1. **Ask Phase**: Calculate payout based on stake, fair probability, and commission
2. **Bid Phase**: Use fixed payout from purchase, evaluate win/loss conditions
3. **Expiry Handling**: Time-based vs tick-based expiry must be separate code paths
4. **Contract**: A conceptual entity representing the contract specification during API request processing. Ask and Bid operate on the same Contract entity at different lifecycle stages.

### Duration Handling Pattern
- **MUST** support both time-based (s, m, h, d) and tick-based (t) durations if defined in product specification
- Parse duration strings into structured duration objects
- Validate duration constraints per product specification
- Duration **MUST** be modeled as a single entity with a type discriminator

---

## Required Components Based on Product Specification

When generating a pricing service, the following components **MUST** be created based on what the product specification includes. These are **MANDATORY** requirements that ensure consistent service architecture.

### Component Matrix

| Specification Contains | Required Component | Path | Purpose |
|------------------------|-------------------|------|---------|
| Market data dependency | Feed Client | `internal/feed/client.go` | Wrapper for service-feed gRPC client |
| Product configuration | Config Manager | `internal/config/config.go` | Symbol-specific settings management |
| Pricing formula/logic | Pricer | `internal/pricer/pricer.go` | Fair probability and payout calculation |
| Contract business rules | Contract Manager | `internal/contract/contract.go` | Validation, lifecycle, and domain logic |

### 1. Feed Client (`internal/feed/client.go`)

**When Required**: Product specification mentions dependency on market data, tick data, or service-feed.

**MUST Implement**:
```go
// Package feed provides a client wrapper for the market data feed service.
package feed

import (
    "context"
    
    feedapi "github.com/regentmarkets/service-feed/api"
)

// Client wraps the service-feed gRPC client.
type Client struct {
    conn   *grpc.ClientConn
    client feedapi.TickServiceClient
}

// NewClient creates a new feed client connected to the specified address.
func NewClient(ctx context.Context, address string) (*Client, error) {
    // Implementation
}

// GetTick retrieves the latest tick for a symbol.
func (c *Client) GetTick(ctx context.Context, symbol string) (*Tick, error) {
    // Implementation
}

// StreamTicks subscribes to real-time tick updates.
func (c *Client) StreamTicks(ctx context.Context, symbol string) (<-chan *Tick, error) {
    // Implementation
}

// Close closes the underlying connection.
func (c *Client) Close() error {
    // Implementation
}
```

**Key Requirements**:
- **MUST** define its own `Tick` type (do not expose proto types)
- **MUST** handle connection lifecycle (connect, reconnect, close)
- **MUST** implement context cancellation
- **MUST** wrap errors with context

### 2. Config Manager (`internal/config/config.go`)

**When Required**: Product specification includes symbol-specific parameters, commission rates, duration limits, or any configurable values.

**MUST Implement**:
```go
// Package config manages product configuration.
package config

import (
    "context"
)

// Manager handles loading and accessing configuration.
type Manager struct {
    configs map[string]*SymbolConfig
}

// SymbolConfig contains symbol-specific configuration.
type SymbolConfig struct {
    Symbol           string
    CommissionRate   float64
    MinStake         float64
    MaxStake         float64
    MinDuration      Duration
    MaxDuration      Duration
    AllowedDurations []DurationType
    // Add product-specific fields
}

// NewManager creates a config manager and loads configuration.
func NewManager(configPath string) (*Manager, error) {
    // Implementation
}

// GetSymbolConfig returns configuration for a specific symbol.
func (m *Manager) GetSymbolConfig(symbol string) (*SymbolConfig, error) {
    // Implementation
}

// Reload reloads configuration from disk.
func (m *Manager) Reload(ctx context.Context) error {
    // Implementation
}
```

**Key Requirements**:
- **MUST** support YAML configuration files
- **MUST** validate configuration on load
- **MUST** return errors for missing symbols (not defaults)
- **MUST** support hot-reload capability

### 3. Pricer (`internal/pricer/pricer.go`)

**When Required**: Product specification includes pricing formula, fair probability calculation, or payout logic.

**MUST Implement**:
```go
// Package pricer implements pricing calculations for the product.
package pricer

import (
    "context"
)

// Pricer calculates ask and bid prices for contracts.
type Pricer struct {
    config ConfigProvider
    feed   FeedProvider
}

// ConfigProvider defines the interface for accessing configuration.
type ConfigProvider interface {
    GetSymbolConfig(symbol string) (*SymbolConfig, error)
}

// FeedProvider defines the interface for accessing market data.
type FeedProvider interface {
    GetTick(ctx context.Context, symbol string) (*Tick, error)
}

// NewPricer creates a new pricer with the given dependencies.
func NewPricer(config ConfigProvider, feed FeedProvider) *Pricer {
    return &Pricer{config: config, feed: feed}
}

// CalculateAsk computes the ask price (payout) for a contract.
func (p *Pricer) CalculateAsk(ctx context.Context, req *AskRequest) (*AskResult, error) {
    // 1. Get current market data
    // 2. Calculate fair probability
    // 3. Apply commission
    // 4. Return payout
}

// CalculateBid computes the bid price for an open contract.
func (p *Pricer) CalculateBid(ctx context.Context, req *BidRequest) (*BidResult, error) {
    // 1. Get current market data
    // 2. Evaluate contract state against exit conditions
    // 3. Calculate current value
    // 4. Return bid price
}
```

**Key Requirements**:
- **MUST** define interfaces for dependencies (ConfigProvider, FeedProvider)
- **MUST** separate fair probability calculation from commission application
- **MUST** return structured results (not just numbers)
- **MUST** be stateless (all state passed via request)
- **MUST NOT** import from other internal packages (others depend on pricer)

### 4. Contract Manager (`internal/contract/contract.go`)

**When Required**: Product specification includes contract validation rules, duration constraints, or lifecycle state management.

**MUST Implement**:
```go
// Package contract handles contract validation and lifecycle.
package contract

import (
    "context"
    "time"
)

// Manager handles contract validation and state.
type Manager struct {
    config ConfigProvider
}

// ConfigProvider defines the interface for accessing configuration.
type ConfigProvider interface {
    GetSymbolConfig(symbol string) (*SymbolConfig, error)
}

// Contract represents a contract instance.
type Contract struct {
    Symbol      string
    Stake       float64
    Duration    Duration
    StartTime   time.Time
    StartSpot   float64
    Payout      float64
    // Product-specific fields
}

// Duration represents a parsed duration with type.
type Duration struct {
    Value int
    Unit  DurationUnit
}

// DurationUnit discriminates between duration types.
type DurationUnit string

const (
    DurationUnitSeconds DurationUnit = "s"
    DurationUnitMinutes DurationUnit = "m"
    DurationUnitHours   DurationUnit = "h"
    DurationUnitDays    DurationUnit = "d"
    DurationUnitTicks   DurationUnit = "t"
)

// NewManager creates a contract manager.
func NewManager(config ConfigProvider) *Manager {
    return &Manager{config: config}
}

// ValidateAskRequest validates parameters for an ask request.
func (m *Manager) ValidateAskRequest(ctx context.Context, req *AskRequest) error {
    // Validate symbol, stake, duration, barriers, etc.
}

// ValidateBidRequest validates parameters for a bid request.
func (m *Manager) ValidateBidRequest(ctx context.Context, req *BidRequest) error {
    // Validate contract state, timing, etc.
}

// ParseDuration parses a duration string (e.g., "5m", "10t").
func ParseDuration(s string) (Duration, error) {
    // Implementation
}

// IsExpired checks if a contract has expired.
func (c *Contract) IsExpired(now time.Time, tickCount int) bool {
    // Handle both time-based and tick-based expiry
}
```

**Key Requirements**:
- **MUST** handle both time-based and tick-based durations
- **MUST** define Duration as a value type with unit discriminator
- **MUST** validate all business rules from product specification
- **MUST** return descriptive validation errors
- **MUST** be independent of transport layer (no gRPC types)

---

## Generated Service Structure (Extended)

With all required components, the complete structure becomes:

```
service-pricer-{product}/
├── .clinerules
├── .gitignore
├── .golangci.yml
├── buf.gen.yaml
├── buf.yaml
├── Dockerfile
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── api/
│   └── {product}/
│       ├── {product}.pb.go
│       ├── {product}_grpc.pb.go
│       └── {product}.pb.gw.go
├── cmd/
│   └── {product}/
│       └── main.go
├── config/
│   └── symbols.yml              # Symbol-specific configuration
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── app_test.go
│   ├── config/                  # REQUIRED if product has configuration
│   │   ├── config.go
│   │   └── config_test.go
│   ├── contract/                # REQUIRED if product has business rules
│   │   ├── contract.go
│   │   └── contract_test.go
│   ├── feed/                    # REQUIRED if product needs market data
│   │   ├── client.go
│   │   └── client_test.go
│   ├── grpcsvc/
│   │   ├── grpcsvc.go
│   │   └── grpcsvc_test.go
│   ├── pricer/                  # REQUIRED if product has pricing logic
│   │   ├── pricer.go
│   │   └── pricer_test.go
│   └── tools/
│       └── tools.go
└── proto/
    └── {product}/
        └── v1/
            └── {product}.proto
```

---

## Dependency Direction Rules

The following dependency graph **MUST** be maintained:

```
┌─────────────────────────────────────────────────────────────┐
│                         grpcsvc                             │
│              (gRPC handlers, entry point)                   │
└──────────────────────────┬──────────────────────────────────┘
                           │ depends on
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                         pricer                              │
│            (core pricing logic, defines interfaces)         │
└──────────────────────────┬──────────────────────────────────┘
                           │ interfaces implemented by
              ┌────────────┼────────────┐
              ▼            ▼            ▼
┌─────────────────┐ ┌───────────┐ ┌───────────┐
│    contract     │ │   feed    │ │  config   │
│ (validation)    │ │ (client)  │ │ (loader)  │
└─────────────────┘ └───────────┘ └───────────┘
```

**Rules**:
1. `grpcsvc` depends on `pricer` (calls pricing methods)
2. `pricer` defines interfaces (`ConfigProvider`, `FeedProvider`)
3. `config`, `feed`, `contract` implement interfaces defined in `pricer`
4. `pricer` **MUST NOT** import from `config`, `feed`, or `contract`
5. All packages **MUST NOT** import from `grpcsvc`

---

## STRICT RULES

When generating architecture at `workspace/output/architecture/architecture.md`:

1. **MUST** follow the service naming conventions defined above
2. **MUST** use gRPC as the primary API protocol
3. **MUST** implement the Ask/Bid API pattern
4. **MUST** use the specified configuration structure
5. **MUST** maintain service statelessness where possible
6. grpc handlers **MUST** receive requests from client and pass them to higher level modules, they should not be fetching data from various sources and assembling it together.
7. **MUST** return gRPC standard error codes and not custom error.
8. **MUST NOT**  ever create models, types or interfaces package. All the dependencies should be directed towards the core (in this case, pricing). Other packages should depend on pricing, pricing should not depend on anything else.
9. **MUST** define interface where it is consumed.
