# Double Rise/Fall Pricing Service

Internal gRPC service for calculating Double Rise/Fall binary options contract prices.

## Overview

The Double Rise/Fall Pricing Service implements a closed-form analytical pricing model using bivariate normal distribution to determine fair probabilities for path-dependent contracts that require spot price to breach a barrier at two distinct evaluation times (t1 and t2).

## Features

- **Ask Price Calculation**: Calculate contract purchase price (stake to payout ratio)
- **Bid Price Calculation**: Value active contracts and determine win/loss at expiry
- **Real-time Streaming**: Continuous price updates via gRPC streams
- **Duration Support**: Both time-based (s/m/h/d) and tick-based (t) durations
- **Symbol Configuration**: Per-symbol commission rates and limits

## Architecture

### Dependency Direction

```
grpcsvc (entry point)
    ↓
pricer (core logic, defines interfaces)
    ↑ implements
    ├── contract (validation)
    ├── config (configuration)
    └── feed (market data)
```

### Components

- **pricer**: Core pricing logic implementing bivariate normal formula
- **contract**: Duration parsing and validation
- **config**: Symbol configuration management from YAML
- **feed**: Wrapper for service-feed client
- **grpcsvc**: gRPC handlers (thin layer)
- **app**: Dependency injection and lifecycle management

## Prerequisites

- Go 1.21 or higher
- Access to `service-feed` for market data
- Protocol Buffer compiler and Go plugins

## Configuration

Configuration is loaded from `config/symbols.yml`:

```yaml
symbols:
  R_100:
    commission: 0.05
    max_payout: 1000.00
    min_stake: 1.00
    enabled: true
```

## Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Required
GRPC_PORT=50051
FEED_SERVICE_ADDR=service-feed:50051
CONFIG_PATH=config/symbols.yml

# Optional
LOG_LEVEL=info
```

## Building

### Local Build

```bash
# Build binary
make build

# Run tests
make test

# Run with coverage
make test-coverage
```

### Docker Build

```bash
make docker-build
```

## Running

### Local Development

```bash
# Set environment variables
export FEED_SERVICE_ADDR=localhost:50051
export CONFIG_PATH=config/symbols.yml

# Run service
make run
```

### Docker

```bash
docker run -p 50051:50051 \
  -e FEED_SERVICE_ADDR=service-feed:50051 \
  -e CONFIG_PATH=config/symbols.yml \
  service-pricer-doublerisefall:latest
```

## API

### gRPC Endpoints

| Method | Type | Description |
|--------|------|-------------|
| `GetAsk` | Unary | Calculate contract purchase price |
| `StreamAsk` | Server Stream | Real-time price updates |
| `GetBid` | Unary | Value active contract |
| `StreamBid` | Server Stream | Real-time contract valuation |

### Proto Definition

See [`api/proto/doublerisefall/v1/doublerisefall.proto`](api/proto/doublerisefall/v1/doublerisefall.proto)

### Example Request

```bash
# Using grpcurl
grpcurl -plaintext \
  -d '{
    "option_parameters": {
      "symbol": "R_100",
      "contract_type": "CONTRACT_TYPE_RISE",
      "currency": "USD",
      "first_duration": "1m",
      "second_duration": "2m",
      "stake": "10.00"
    }
  }' \
  localhost:50051 \
  doublerisefall.v1.DoubleRiseFallService/GetAsk
```

## Pricing Formula

**Fair Probability**:
```
P_fair = 1/4 + arcsin(√(t1/t2)) / (2π)
```

**Client Price (Ask)**:
```
P_client = P_fair + commission
```

**Payout**:
```
Payout = Stake / P_client
```

## Development

### Project Structure

```
.
├── api/                    # Generated proto code
├── cmd/doublerisefall/     # Main entry point
├── config/                 # Configuration files
├── internal/
│   ├── app/               # Application lifecycle
│   ├── config/            # Config management
│   ├── contract/          # Validation
│   ├── feed/              # Market data client
│   ├── grpcsvc/           # gRPC handlers
│   └── pricer/            # Core pricing logic
├── Dockerfile
├── Makefile
└── README.md
```

### Testing

```bash
# Run all tests
make test

# Run with race detector
go test -race ./...

# Run specific package tests
go test ./internal/pricer/...
```

### Code Style

```bash
# Format code
make fmt

# Run linter
make lint
```

## Health Check

The service implements gRPC health check protocol:

```bash
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

## Monitoring

The service logs structured JSON output using `slog`:

- **Debug**: Detailed operation logs
- **Info**: Normal operation events
- **Error**: Error conditions

Set `LOG_LEVEL` environment variable to control verbosity.

## Error Handling

All errors use standard gRPC status codes with descriptive error codes:

| Error Code | gRPC Status | Description |
|------------|-------------|-------------|
| `ERR-DR-S1V` | `INVALID_ARGUMENT` | Invalid symbol |
| `ERR-DR-D2U` | `INVALID_ARGUMENT` | Invalid duration format |
| `ERR-DR-M8E` | `UNAVAILABLE` | Market data unavailable |
| `ERR-DR-I9N` | `INTERNAL` | Internal error |

See API specification for complete error code list.

## Dependencies

- **service-feed**: Market data (critical)
- **google.golang.org/grpc**: gRPC implementation
- **gopkg.in/yaml.v3**: Configuration parsing

## Contributing

1. Follow Go standard project layout
2. Write tests for new features
3. Update documentation
4. Ensure all tests pass before submitting

## License

Internal service - Regent Markets Group
