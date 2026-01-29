# Development Phase Preferences

## Overview
This document captures all decisions, directives, and clarifications made during the development phase for the Deriv Arcade project.

---

## General Development Standards

### Code Organization
- **Pattern**: Layered modular architecture (API → Service → Repository)
- **Rationale**: Clear separation of concerns for complexity score 4-7 services
- **Applied to**: All services

### Error Handling
- **Pattern**: Sentinel errors + structured API errors
- **Implementation**: 
  - Common errors defined in `internal/common/errors.go`
  - API errors with code + message structure
  - HTTP status code mapping in response handlers
- **Rationale**: Consistent error handling across all endpoints

### Logging
- **Library**: zerolog
- **Format**: Structured JSON logging
- **Configuration**: Environment-variable controlled log level
- **Rationale**: High-performance structured logging for observability

---

## Technology Stack

### [arcade] Backend Language
- **Technology**: Golang 1.21
- **Framework**: Chi v5 (lightweight HTTP router)
- **Rationale**: Workspace preference; strong concurrency support; excellent for REST APIs

### [arcade] Database
- **Technology**: PostgreSQL 14+
- **Driver**: pgx v5 (native PostgreSQL driver)
- **Connection**: Connection pooling (min: 5, max: 25)
- **Rationale**: Workspace preference; JSONB support for OHLC data; strong transactional guarantees

### [arcade] Decimal Handling
- **Library**: shopspring/decimal v1.3.1
- **Precision**: 
  - Monetary amounts: 2 decimal places
  - OHLC prices: 3 decimal places
- **Rationale**: Precise monetary calculations avoiding floating-point errors

### [arcade] Migrations
- **Tool**: golang-migrate v4
- **Location**: `migrations/` directory
- **Format**: Numbered SQL files (up/down pairs)
- **Rationale**: Version-controlled, database-agnostic migrations

---

## Implementation Patterns

### [arcade] Modular Monolith Architecture
- **Decision**: Single service with two internal modules (Accounts, Trading)
- **Modules**:
  - Accounts: Account management, deposits, withdrawals, balance tracking
  - Trading: GBM price generation, trade execution, history
- **Rationale**: Complexity score 4/10 → light modular organization; atomic transactions required across domains

### [arcade] Internal API Pattern
- **Implementation**: Go function calls between modules
- **Functions**: GetAccount, GetAccountBalance, DeductStake, CreditPayout
- **Transaction Propagation**: Context-based transaction passing
- **Rationale**: Enables atomic trade execution without distributed transactions

### [arcade] Atomic Trade Execution
- **Pattern**: Single database transaction spanning multiple operations
- **Flow**:
  1. Validate account
  2. Validate quote
  3. Check balance
  4. Deduct stake
  5. Generate candles 11-20
  6. Evaluate outcome
  7. Credit payout
  8. Create contract
  9. Delete price series
- **Rationale**: All-or-nothing execution for financial integrity

### [arcade] Idempotency Pattern
- **Implementation**: UUID-based idempotency keys for deposits/withdrawals
- **Storage**: Unique index on (account_id, idempotency_id)
- **Behavior**: Return original transaction if duplicate key
- **Rationale**: Safe retry behavior for financial operations

### [arcade] Row-Level Locking
- **Implementation**: SELECT FOR UPDATE on balance updates
- **Usage**: All balance modifications (deposits, withdrawals, stakes, payouts)
- **Rationale**: Prevents race conditions in concurrent balance updates

---

## Design Decisions

### [arcade] Account ID Format
- **Format**: SW-prefixed sequential integers (SW1, SW2, SW3...)
- **Implementation**: PostgreSQL sequence + string concatenation
- **Rationale**: Human-readable, sequential, matches PRD requirements

### [arcade] GBM Algorithm Implementation
- **Formula**: dS = μSdt + σSdW
- **Parameters**:
  - Vol50: initial=10000, volatility=50%
  - Vol100: initial=50000, volatility=100%
  - Vol200: initial=100000, volatility=200%
  - Vol300: initial=200000, volatility=300%
- **Implementation**: Custom generator with math/rand for normal distribution
- **Rationale**: Realistic price movements with configurable volatility

### [arcade] Quote Validation Mechanism
- **Pattern**: Store 10th candle close as quote_value in price_series
- **Lookup**: Match by (account_id + series_type + quote_value)
- **Lifecycle**: Created on SwipeGet, matched on SwipeBuy, deleted after trade
- **Rationale**: Implicit matching without explicit series IDs; supports concurrent previews

### [arcade] Payout Calculation
- **Formula**: Win payout = stake / 0.53 (≈ stake × 1.8868)
- **Loss Payout**: 0.00
- **Rounding**: 2 decimal places using standard rounding
- **Transaction**: PAYOUT transaction created even for losses
- **Rationale**: Matches PRD requirements; maintains audit trail

### [arcade] Zero Payout Transactions
- **Decision**: Create PAYOUT transaction even when amount is 0.00
- **Rationale**: Complete audit trail for all trades; consistency in transaction history

---

## Trade-offs

### [arcade] Modular Monolith vs Microservices
- **Chosen**: Modular monolith
- **Alternative**: Separate microservices for Accounts and Trading
- **Trade-off**: 
  - ✅ Simpler deployment
  - ✅ Atomic transactions without distributed coordination
  - ✅ No network latency between modules
  - ❌ Cannot scale modules independently
  - ❌ All code deployed together
- **Rationale**: Complexity score 4/10 and atomic transaction requirement favor monolith

### [arcade] JSONB for OHLC Storage
- **Chosen**: PostgreSQL JSONB columns
- **Alternative**: Separate OHLC table with foreign keys
- **Trade-off**:
  - ✅ Simple schema
  - ✅ Easy to retrieve full contract with candles
  - ✅ No JOIN queries needed
  - ❌ Cannot query individual candles efficiently
  - ❌ Slightly larger storage
- **Rationale**: OHLC data always accessed as complete array; query simplicity

### [arcade] Chi Router vs Standard Library
- **Chosen**: Chi router
- **Alternative**: Go standard library net/http only
- **Trade-off**:
  - ✅ URL parameter extraction (account_id)
  - ✅ Middleware support
  - ✅ RESTful routing patterns
  - ✅ Minimal overhead
  - ❌ Additional dependency
- **Rationale**: Lightweight with better ergonomics than stdlib

### [arcade] Context-Based Transaction Propagation
- **Chosen**: Pass transactions through context.Context
- **Alternative**: Pass transaction explicitly as parameter
- **Trade-off**:
  - ✅ Idiomatic Go pattern
  - ✅ Works with existing service interfaces
  - ✅ Transparent to most code
  - ❌ Slightly less explicit
- **Rationale**: Standard Go pattern for request-scoped values

---

## Code Organization

### [arcade] Directory Structure
```
arcade/
├── cmd/server/              # Application entry point
├── internal/
│   ├── api/                 # HTTP handlers and routing
│   ├── accounts/            # Accounts module (service, repo, model)
│   ├── trading/             # Trading module (service, repo, model, gbm)
│   └── common/              # Shared utilities
├── migrations/              # Database migrations
├── config/                  # Configuration management
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

**Rationale**: Light modular organization appropriate for complexity 4/10; clear module boundaries without over-engineering

### [arcade] File Naming Convention
- `service.go`: Business logic layer
- `repository.go`: Data access layer
- `model.go`: Domain models and DTOs
- `errors.go`: Module-specific errors
- `*_handler.go`: API handlers per module

---

## Integration Approach

### [arcade] Database Connection
- **Pattern**: Shared connection pool across all modules
- **Lifecycle**: Created on startup, closed on shutdown
- **Configuration**: Externalized via environment variables

### [arcade] Health Check
- **Endpoint**: GET /health
- **Response**: `{"status": "healthy"}`
- **Purpose**: Container orchestration readiness/liveness probes

### [arcade] Docker Configuration
- **Multi-stage Build**: Builder + runtime stages
- **Base Images**: golang:1.21-alpine (build), alpine:3.18 (runtime)
- **Health Check**: Built into Dockerfile (wget to /health)
- **Rationale**: Minimal image size; production-ready

### [arcade] Development Environment
- **Tool**: docker-compose
- **Services**: arcade (app) + db (PostgreSQL)
- **Features**: Auto-restart, health checks, volume persistence
- **Rationale**: Complete local development environment

---

## Testing Strategy

### [arcade] Unit Testing
- **Framework**: Go testing + testify
- **Coverage**: Service and repository layers
- **Mocking**: Interface-based mocking for dependencies
- **Pattern**: Table-driven tests

### [arcade] Integration Testing
- **Scope**: End-to-end API tests
- **Database**: testcontainers with PostgreSQL
- **Focus**: Atomic transactions, idempotency, error cases

---

## Security Implementation

### [arcade] Phase 1 Security
- **Authentication**: None (as per PRD)
- **Authorization**: None (as per PRD)
- **Input Validation**: All inputs validated at API layer
- **SQL Injection**: Prevented via parameterized queries (pgx)
- **Note**: Production deployment requires auth implementation

### [arcade] Data Protection
- **PII**: No personal information stored
- **Decimal Precision**: shopspring/decimal for financial accuracy
- **Balance Validation**: Non-negative constraint at database level

---

## Performance Considerations

### [arcade] Optimization Decisions
- **Connection Pooling**: Configured min/max connections
- **Indexing**: Strategic indexes on lookup patterns
  - price_series: (account_id, series_type, quote_value)
  - contracts: (account_id, purchase_time DESC)
- **Query Optimization**: SELECT FOR UPDATE for balance locking
- **Rationale**: Meet performance targets (< 200ms p95)

---

## Changelog

| Date | Service | Change | Reason |
|------|---------|--------|--------|
| 2026-01-16 | arcade | Initial implementation | New service development |
| 2026-01-16 | arcade | Modular monolith architecture | Complexity 4/10 + atomic transactions |
| 2026-01-16 | arcade | Context-based transaction propagation | Idiomatic Go pattern |
| 2026-01-16 | arcade | JSONB for OHLC storage | Simplicity over query flexibility |
| 2026-01-16 | arcade | Chi router selection | Ergonomics with minimal overhead |

---

## Future Considerations

### [arcade] Potential Enhancements
- Add authentication/authorization layer
- Implement rate limiting
- Add caching for series configurations
- Implement background job for orphaned price_series cleanup
- Add performance monitoring/metrics
- Implement audit logging
- Add integration with external broker systems

### [arcade] Migration Path
If future requirements demand microservices:
1. Extract internal interfaces to gRPC/REST APIs
2. Separate accounts and trading databases
3. Implement saga pattern for distributed transactions
4. Deploy services independently

**Current Decision**: Defer until complexity justifies the overhead
