# service-pricer-doublerisefall

A specialized pricing engine for the Double Rise/Fall digital binary option product. This service implements path-dependent contract pricing that evaluates spot prices against a barrier at two distinct timestamps (t1 and t2).

## Features

- **Real-time Pricing**: Calculate ask prices based on fair probability and commission
- **Contract Valuation**: Evaluate bid prices for active contracts
- **Streaming Support**: Real-time price updates via gRPC streaming
- **Path-Dependent Logic**: Win/loss evaluation at two time points (t1 and t2)
- **Flexible Durations**: Support for both time-based (s, m, h, d) and tick-based (t) durations
- **Hot-Reload Configuration**: Symbol configuration updates without restart

## Architecture

The service follows a modular architecture with clear separation of concerns:

```
grpcsvc → pricer → (config, feed, contract)
```

### Internal Packages

- **pricer**: Core pricing logic, defines interfaces for dependencies
- **grpcsvc**: gRPC handlers and streaming implementation
- **config**: Symbol configuration management with hot-reload
- **contract**: Duration parsing and validation
- **feed**: Wrapper for service-feed client
- **app**: Application initialization and dependency wiring

## Prerequisites

- Go 1.21 or higher
- Access to `service-feed` for market data
- Protocol Buffers compiler (for development)

## Installation

### Build from Source

```bash
# Clone the repository
git clone github.com/regentmarkets/service-pricer-doublerisefall
cd service-pricer-doublerisefall

# Download dependencies
go mod download

# Generate protobuf code
make proto

# Build the application
make build
```

### Docker

```bash
# Build Docker image
docker build -t service-pricer-doublerisefall:latest .

# Run container
docker run -p 50051:50051 \
  -e FEED_SERVICE_ADDR=service-feed:50051 \
  service-pricer-doublerisefall:latest
```

## Configuration

### Environment Variables

See [`.env.example`](.env.example) for all available configuration options:

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC server port | 50051 |
| `FEED_SERVICE_ADDR` | service-feed address | service-feed:50051 |
| `CONFIG_PATH` | Path to symbols.yaml | ./config/symbols.yaml |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info |
| `FEED_RETRY_ATTEMPTS` | Feed service retry attempts | 3 |
| `FEED_RETRY_DELAY_MS` | Retry delay in milliseconds | 1000 |

### Symbol Configuration

Edit [`config/symbols.yaml`](config/symbols.yaml) to configure trading parameters per symbol:

```yaml
symbols:
  R_100:
    symbol: R_100
    commission_rate: 0.05    # 5% commission
    max_payout: 1000.00      # Maximum payout limit
    min_stake: 1.00          # Minimum stake requirement
    enabled: true            # Enable/disable trading
```

## Usage

### Running Locally

```bash
# Set environment variables
export FEED_SERVICE_ADDR=localhost:50051
export CONFIG_PATH=./config/symbols.yaml

# Run the service
./bin/doublerisefall
```

### API Endpoints

The service exposes 4 gRPC endpoints:

1. **GetAsk**: Calculate single contract price
2. **StreamAsk**: Stream real-time contract prices
3. **GetBid**: Evaluate current contract value
4. **StreamBid**: Stream real-time contract values

See [API documentation](proto/doublerisefall/v1/doublerisefall.proto) for detailed request/response schemas.

## Development

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint
```

### Proto Generation

```bash
# Regenerate protobuf code
make proto
```

## Pricing Formula

The service uses the arcsin correlation formula for fair probability calculation:

**Correlation**: ρ = √(t₁/t₂)

**Fair Probability**: P_fair = 1/4 + arcsin(ρ)/(2π)

**Client Price**: P_client = P_fair + Commission

**Payout**: Payout = Stake / P_client

## Win/Loss Conditions

### RISE Contract
- **Win**: spot_t1 > barrier AND spot_t2 > barrier
- **Loss**: spot_t1 ≤ barrier OR spot_t2 ≤ barrier

### FALL Contract
- **Win**: spot_t1 < barrier AND spot_t2 < barrier
- **Loss**: spot_t1 ≥ barrier OR spot_t2 ≥ barrier

**Note**: If the condition fails at t1, the contract expires worthless immediately (short-circuit evaluation).

## Health Check

The service implements the standard gRPC health check protocol:

```bash
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

## Dependencies

- **service-feed**: Market data provider (required)
- **google.golang.org/grpc**: gRPC framework
- **github.com/spf13/viper**: Configuration management

## License

Copyright © 2026 Regent Markets

## Support

For issues or questions, contact the development team.
