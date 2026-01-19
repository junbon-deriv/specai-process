# Arcade Service

The **arcade** service is the single backend service for Deriv Arcade, an arcade-style binary options trading platform. It implements a modular monolith architecture containing two internal modules: Accounts and Trading.

## Overview

- **Service ID**: SVC-AR-K3M
- **Tech Stack**: Golang, Chi router, PostgreSQL 14+, pgx driver
- **Architecture**: Modular monolith with light separation (complexity score 4/10)
- **Modules**: Accounts (account management, deposits, withdrawals) and Trading (SwipeGet, SwipeBuy, SwipeList)

## Features

### Accounts Module
- Create trading accounts with SW-prefixed sequential IDs
- Idempotent deposits and withdrawals
- Balance management with row-level locking
- Internal API for trading module

### Trading Module
- GBM price generation for 4 series types (Vol50, Vol100, Vol200, Vol300)
- Price preview (SwipeGet) - generates 10-candle OHLC series
- Atomic trade execution (SwipeBuy) - immediate settlement with payout
- Trading history (SwipeList) - last 50 contracts per account

## API Endpoints

### Accounts
- `POST /accounts` - Create new trading account
- `GET /accounts/{account_id}` - Get account details and balance
- `POST /accounts/{account_id}/deposits` - Deposit funds (idempotent)
- `POST /accounts/{account_id}/withdrawals` - Withdraw funds (idempotent)

### Trading
- `GET /swipe?series_type={type}&account_id={id}` - Get price preview
- `POST /swipe/buy` - Place rise/fall trade
- `GET /swipe/list?account_id={id}&series_type={type}` - List trading history

### Operations
- `GET /health` - Health check endpoint

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Go 1.21+ (for local development)
- PostgreSQL 14+ (for local development without Docker)

### Using Docker Compose (Recommended)

```bash
# Start all services
docker-compose up

# Start in detached mode
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

The service will be available at `http://localhost:8080`.

### Local Development

```bash
# Install dependencies
go mod download

# Start PostgreSQL (if not using Docker)
# Update DATABASE_URL in .env

# Run migrations
make migrate-up

# Run the service
make run

# Or use the dev command (starts db + migrations + run)
make dev
```

## Configuration

Environment variables (see `.env.example`):

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `PORT` | No | 8080 | HTTP server port |
| `LOG_LEVEL` | No | info | Logging level (debug, info, warn, error) |
| `DB_MAX_CONNS` | No | 25 | Maximum database connections |
| `DB_MIN_CONNS` | No | 5 | Minimum database connections |

## Database Migrations

```bash
# Apply all migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create NAME=migration_name
```

Migrations are located in the `migrations/` directory.

## Development Commands

```bash
make help          # Show all available commands
make build         # Build the application
make run           # Run locally
make test          # Run tests
make test-coverage # Run tests with coverage report
make docker-build  # Build Docker image
make docker-up     # Start with docker-compose
make docker-down   # Stop docker-compose services
make lint          # Run linter
make fmt           # Format code
```

## Architecture

### Directory Structure

```
arcade/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   ├── router.go            # HTTP router setup
│   │   ├── accounts_handler.go  # Accounts API handlers
│   │   ├── trading_handler.go   # Trading API handlers
│   │   └── response.go          # Response utilities
│   ├── accounts/
│   │   ├── service.go           # Account business logic
│   │   ├── repository.go        # Account data access
│   │   └── model.go             # Account domain models
│   ├── trading/
│   │   ├── service.go           # Trading business logic
│   │   ├── repository.go        # Trading data access
│   │   ├── model.go             # Trading domain models
│   │   └── gbm.go               # GBM price generator
│   └── common/
│       ├── errors.go            # Standard error definitions
│       ├── decimal.go           # Decimal handling utilities
│       ├── validation.go        # Input validation helpers
│       └── context.go           # Context utilities
├── migrations/                  # Database migrations
├── config/
│   └── config.go                # Configuration management
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

### Database Schema

- **accounts**: Trading accounts with SW-prefixed IDs
- **transactions**: Financial movements (DEPOSIT, WITHDRAWAL, STAKE, PAYOUT)
- **price_series**: Temporary price previews (deleted after use)
- **contracts**: Completed trades with full 20-candle OHLC series

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test -v ./internal/accounts/...
go test -v ./internal/trading/...
```

## Trading Flow Example

### 1. Create Account
```bash
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"currency": "USD"}'

# Response: {"account_id": "SW1"}
```

### 2. Deposit Funds
```bash
curl -X POST http://localhost:8080/accounts/SW1/deposits \
  -H "Content-Type: application/json" \
  -d '{
    "amount": "100.00",
    "deposit_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

### 3. Get Price Preview
```bash
curl "http://localhost:8080/swipe?series_type=Vol100&account_id=SW1"

# Note the 10th candle's close value as previous_quote
```

### 4. Place Trade
```bash
curl -X POST http://localhost:8080/swipe/buy \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "SW1",
    "stake": "10.00",
    "series_type": "Vol100",
    "previous_quote": "50005.234",
    "sentiment": "rise"
  }'

# Returns candles 11-20 and payout amount
```

### 5. View Trading History
```bash
curl "http://localhost:8080/swipe/list?account_id=SW1"
```

## GBM Price Generation

The service uses Geometric Brownian Motion (GBM) to generate realistic price movements:

**Formula**: dS = μSdt + σSdW

Where:
- S is the price
- μ is the drift rate (interest rate - quanto drift)
- σ is the volatility
- dt is the time interval (1 second)
- dW is the random increment (normal distribution)

**Series Configurations**:
- Vol50: initial=10000, volatility=50%
- Vol100: initial=50000, volatility=100%
- Vol200: initial=100000, volatility=200%
- Vol300: initial=200000, volatility=300%

## Payout Calculation

- Base probability: 50% (rise/fall)
- Commission: 3%
- Adjusted probability: 53%
- Win payout: stake / 0.53 ≈ stake × 1.8868
- Loss payout: 0.00

## Error Handling

All errors follow a standard format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

Common error codes:
- `ACCOUNT_NOT_FOUND` (404)
- `INSUFFICIENT_BALANCE` (400)
- `INVALID_QUOTE` (400)
- `INVALID_STAKE` (400)
- `INVALID_SERIES_TYPE` (400)

## Performance Targets

- API Response Time: < 200ms (p95)
- Contract Evaluation: < 50ms
- Price Generation: < 100ms for 10 candles

## Security Notes

**Phase 1**: No authentication required. All endpoints are publicly accessible.

**Production Considerations**:
- Implement proper authentication mechanisms
- Add rate limiting
- Enable HTTPS/TLS
- Implement audit logging

## License

Copyright © 2026 Deriv. All rights reserved.
