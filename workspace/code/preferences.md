# Development Preferences

This document tracks development decisions and patterns for all service implementations in this workspace.

## General Development Standards

### Code Organization
- **Light modular organization** for complexity 6/10 services
- Clear separation of concerns: grpcsvc → pricer → implementations
- Interface-at-consumer pattern: interfaces defined where consumed (in pricer package)

### Dependency Direction
- All dependencies flow toward core logic (pricer)
- Core logic (pricer) defines interfaces
- Implementation packages (config, feed, contract) implement these interfaces
- gRPC layer depends on pricer, never the reverse

## Technology Stack

### Common Frameworks
- **Go 1.21+**: Primary language for all services
- **gRPC**: Internal service communication protocol
- **Protocol Buffers v3**: Message serialization
- **slog**: Structured logging (stdlib)
- **Viper/YAML**: Configuration management

### External Dependencies
- **gopkg.in/yaml.v3**: YAML parsing for configuration
- **google.golang.org/grpc**: Official gRPC implementation
- **google.golang.org/protobuf**: Protobuf runtime

## Implementation Patterns

### Service Structure (Complexity 6/10)
```
service-pricer-{product}/
├── cmd/{product}/          # Entry point
├── config/                 # Configuration files (YAML)
├── internal/
│   ├── app/               # DI wiring, lifecycle
│   ├── config/            # Config loading
│   ├── contract/          # Validation logic
│   ├── feed/              # External service wrappers
│   ├── grpcsvc/           # gRPC handlers (thin)
│   └── pricer/            # Core domain logic
└── api/proto/             # Generated protobuf code
```

### Interface Definition Pattern
- Define interfaces in the package that **consumes** them
- Example: `ConfigProvider`, `FeedProvider` defined in `pricer` package
- Implementation packages import from pricer, not vice versa
- No separate `interfaces` package

### Pricing Service Pattern
Components required for options pricing services:
1. **pricer**: Fair probability, commission, payout calculation
2. **contract**: Duration parsing, validation rules
3. **config**: Symbol configuration from YAML
4. **feed**: Market data client wrapper
5. **grpcsvc**: Thin gRPC handlers
6. **app**: Dependency injection

### Duration Handling
- Single `Duration` type with unit discriminator (s/m/h/d/t)
- Separate validation paths for time-based vs tick-based
- Parse once, validate constraints based on unit type

### Error Handling
- Domain errors defined in pricer package
- gRPC layer maps domain errors to status codes
- Use standard gRPC codes: INVALID_ARGUMENT, FAILED_PRECONDITION, UNAVAILABLE, INTERNAL
- Include error codes in messages (e.g., "ERR-DR-S1V: invalid symbol")

## Code Organization

### Package Responsibilities
| Package | Imports From | Exports | Purpose |
|---------|-------------|---------|---------|
| `pricer` | None | Pricing logic, interfaces | Core domain |
| `contract` | `pricer` | Validation implementation | Implements ContractValidator |
| `config` | `pricer` | Config management | Implements ConfigProvider |
| `feed` | `pricer` | Feed wrapper | Implements FeedProvider |
| `grpcsvc` | `pricer` | gRPC handlers | Protocol adapter |
| `app` | All | Application lifecycle | Wiring |

### No Circular Dependencies
- pricer MUST NOT import from contract, config, or feed
- grpcsvc MUST NOT be imported by any internal package
- Clean dependency graph ensures testability

## Integration Approach

### External Service Integration
- **Pattern**: Thin wrapper around external clients
- **Location**: `internal/feed/` (or service-specific name)
- **Interface**: Defined in consuming package (pricer)
- **Types**: Convert external types to domain types at boundary

### Service Feed Integration
- Use `github.com/regentmarkets/service-feed/client` directly
- Create wrapper implementing `FeedProvider` interface
- Convert proto Tick to domain Tick at wrapper boundary
- Handle reconnection logic in wrapper

## Configuration Management

### Configuration Files
- **Format**: YAML
- **Location**: `config/` directory
- **Loading**: At startup with hot-reload capability
- **Structure**: Per-symbol configuration with defaults

### Environment Variables
- Use for runtime configuration (ports, addresses)
- Document all variables in `.env.example`
- Provide sensible defaults in code

## Testing Strategy

### Test Organization
- Tests adjacent to implementation files
- Use table-driven tests for multiple scenarios
- Mock interfaces for unit tests
- Integration tests at package boundaries

### Coverage Goals
- High coverage for business logic (pricer, contract)
- Integration tests for gRPC handlers
- Mock external dependencies (feed client)

## Deployment Standards

### Containerization
- Multi-stage Docker builds
- Alpine base image for minimal size
- Non-root user for security
- Health check included in Dockerfile

### Build Automation
- Makefile for common tasks
- Targets: build, test, clean, docker-build, lint
- Simple and consistent across services

## Logging Standards

### Structured Logging
- Use `slog` for all logging
- JSON format for production
- Log levels: debug, info, warn, error
- Include context: operation, symbol, error details

### What to Log
- **Debug**: Detailed operation flow, pricing calculations
- **Info**: Startup, configuration loaded, major operations
- **Error**: All error conditions with context

## Service-Specific Decisions

### [service-pricer-doublerisefall] Double Rise/Fall Pricing

**Technology Choices**:
- Go 1.21+, gRPC, Protocol Buffers, slog, Viper
- service-feed client wrapper for market data

**Implementation Decisions**:
- Bivariate normal pricing formula in pricer package
- Duration type with unit discriminator for time/tick handling
- Separate evaluation logic for RISE vs FALL contract types
- Stream implementation: goroutine per stream with ticker for time-based updates

**Design Patterns**:
- Interface-at-consumer: pricer defines ConfigProvider, FeedProvider, ContractValidator
- Dependency injection in app package
- Stateless design: all state in request context

**Trade-offs**:
- Using float64 for calculations (acceptable for current precision requirements)
- No caching (market data must be real-time)
- Streaming: 5-second ticker + on-tick updates (balance between freshness and load)

**Validation Rules**:
- Duration gap: 10s minimum (time), 2t minimum (ticks)
- Duration range: 10s-1day (time), 2-10 ticks
- Payout validation against configured maximum

---

## Future Considerations

- Consider `shopspring/decimal` if higher precision needed
- Metrics/instrumentation for production monitoring
- Rate limiting at gRPC layer if needed
- Circuit breaker for external dependencies

---

**Last Updated**: 2026-01-15
**Services**: service-pricer-doublerisefall
