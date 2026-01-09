# Digital Call/Put Options Pricing Service

A stateless Go microservice that provides real-time pricing for digital (binary) options contracts using the Black-Scholes pricing model.

## Overview

This service calculates Ask prices (for purchasing new contracts) and Bid prices (for valuing active contracts) for digital call and put options. It supports both single-request and streaming scenarios for time-based and tick-based contracts.

## Features

- **Ask Price Calculation**: Calculate purchase price for new digital options
- **Bid Price Calculation**: Calculate current value of active contracts
- **Streaming Pricing**: Real-time price updates via server streaming
- **Barrier Support**: Absolute, relative (+/-), and ATM (at-the-money) barriers
- **Duration Handling**: Time-based (seconds, minutes, hours, days) and tick-based durations
- **Black-Scholes Model**: Industry-standard pricing with configurable volatility and rate

## Tech Stack

- **Language**: Go 1.21+
- **API Protocol**: gRPC (Protocol Buffers v3)
- **Configuration**: YAML
- **Logging**: log/slog (structured JSON logging)
- **Dependencies**: service-feed (market data provider)

## Project Structure

```
digitalcallput/
├── cmd/digitalcallput/     # Application entry point
├── internal/
│   ├── app/                # Application setup and wiring
│   ├── grpcsvc/            # gRPC handler implementations
│   ├── pricing/            # Core pricing logic (Black-Scholes)
│   ├── config/             # Configuration management
│   └── market/             # Market data integration (service-feed)
├── proto/                  # Protocol buffer definitions
├── api/                    # Generated protobuf code
├── config/                 # Configuration files
├── Dockerfile              # Container definition
└── Makefile               # Build targets
```

## Prerequisites

- Go 1.21 or higher
- Protocol Buffer compiler (protoc)
- buf (for protobuf generation)
- service-feed running (for market data)

## Installation

### Using go-templates (Recommended)

```bash
# Clone go-templates
git clone git@github.com:junbon-deriv/go-templates.git
cd go-templates
go install

# Generate service
cd ..
go-templates --template service \
  --module-path github.com/regentmarkets/service-pricer-digitalcallput \
  --module-name digitalcallput
```

### Manual Installation

```bash
# Clone repository
git clone git@github.com:regentmarkets/service-pricer-digitalcallput.git
cd service-pricer-digitalcallput

# Install dependencies
go mod download

# Generate protobuf code
make grpc-generate

# Build
go build -o digitalcallput ./cmd/digitalcallput
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_ADDRESS` | gRPC server bind address | `:8090` |
| `FEED_HOST` | service-feed hostname | `localhost` |
| `FEED_PORT` | service-feed port | `50052` |
| `CONFIG_PATH` | Path to symbols.yaml | `./config/symbols.yaml` |
| `LOG_LEVEL` | Logging level (DEBUG, INFO, WARN, ERROR) | `INFO` |

### Symbol Configuration

Edit `config/symbols.yaml` to configure trading symbols:

```yaml
symbols:
  - symbol: "USD/JPY"
    min_stake: "1.00000000"
    max_payout: "50000.00000000"
    commission: "0.02000000"
```

## Usage

### Running Locally

```bash
# Set environment variables
export FEED_HOST=localhost
export FEED_PORT=50052
export CONFIG_PATH=./config/symbols.yaml

# Run service
./digitalcallput
```

### Running with Docker

```bash
# Build image
docker build -t digitalcallput:latest .

# Run container
docker run -p 8090:8090 \
  -e FEED_HOST=service-feed \
  -e FEED_PORT=50052 \
  digitalcallput:latest
```

### Example gRPC Calls

#### GetAsk - Calculate Ask Price

```bash
grpcurl -plaintext \
  -d '{
    "symbol": "USD/JPY",
    "contract_type": "CONTRACT_TYPE_CALL",
    "currency": "USD",
    "duration": "1m",
    "stake": "100.00000000",
    "barrier": "+0.0023"
  }' \
  localhost:8090 \
  digitalcallput.v1.DigitalCallPutService/GetAsk
```

#### GetBid - Calculate Bid Price

```bash
grpcurl -plaintext \
  -d '{
    "symbol": "USD/JPY",
    "contract_type": "CONTRACT_TYPE_CALL",
    "currency": "USD",
    "duration": "1m",
    "start_time": 1736326740,
    "payout": "210.50000000",
    "barrier": "+0.0023"
  }' \
  localhost:8090 \
  digitalcallput.v1.DigitalCallPutService/GetBid
```

## API Reference

### Endpoints

| Method | Type | Description |
|--------|------|-------------|
| `GetAsk` | Unary | Single Ask price calculation |
| `StreamAsk` | Server Stream | Continuous Ask price updates |
| `GetBid` | Unary | Single Bid price calculation |
| `StreamBid` | Server Stream | Continuous Bid price updates |

### Contract Types

- `CONTRACT_TYPE_CALL`: Call option (wins if exit > barrier)
- `CONTRACT_TYPE_PUT`: Put option (wins if exit < barrier)

### Duration Formats

- **Time-based**: `30s`, `5m`, `2h`, `7d`
- **Tick-based**: `5t` (1-10 ticks)

### Barrier Formats

- **Absolute**: `"149.50"` - Fixed price level
- **Relative Positive**: `"+0.0023"` - Entry price + offset
- **Relative Negative**: `"-0.0023"` - Entry price - offset
- **ATM**: Omit parameter - Equals entry price

## Development

### Build Commands

```bash
# Generate protobuf code
make grpc-generate

# Build binary
go build -o digitalcallput ./cmd/digitalcallput

# Run tests
go test ./...

# Run tests with coverage
make test

# Run linter
make lint
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run specific module tests
go test -v ./internal/pricing/...
go test -v ./internal/config/...

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Architecture

### Design Principles

1. **Stateless**: All contract state reconstructed from request parameters
2. **Dependency Inversion**: Pricing core has no dependencies
3. **Handler Delegation**: gRPC handlers delegate to domain modules
4. **Interface Location**: Interfaces defined where consumed

### Module Dependencies

```
grpcsvc → pricing (core)
pricing ← market (implements pricing.MarketDataProvider)
pricing ← config (implements pricing.ConfigProvider)
market → service-feed (external)
```

## Performance

- **GetAsk/GetBid**: < 100ms p95
- **Stream First Response**: < 500ms
- **Stream Update Latency**: < 200ms from tick receipt
- **Concurrent Streams**: 10,000 per instance
- **Throughput**: 5,000 RPS per instance

## Error Handling

| gRPC Code | Condition |
|-----------|-----------|
| `INVALID_ARGUMENT` | Missing required field, invalid format |
| `NOT_FOUND` | Unknown symbol |
| `UNAVAILABLE` | Market data service unavailable |
| `INTERNAL` | Internal calculation error |

## Deployment

### Resource Recommendations

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 0.5 cores | 2 cores |
| Memory | 256MB | 512MB |
| Replicas | 2 | 3+ |

### Health Check

- **Endpoint**: gRPC Health Checking Protocol
- **Port**: 8090

## Dependencies

- [service-feed](https://github.com/junbon-deriv/service-feed) - Market data provider
- [shopspring/decimal](https://github.com/shopspring/decimal) - High-precision decimal arithmetic

## License

Copyright © 2026 Regent Markets

## Support

For issues and questions, please contact the development team.
