# Development Preferences

> **Created**: 2026-01-15
> **Last Updated**: 2026-01-15
> **Scope**: All service implementations in workspace/code

---

## General Development Standards

### Code Quality
- Follow SOLID principles and clean code practices
- Write self-documenting code with clear naming
- Implement comprehensive error handling with context
- Use structured logging (slog) with request IDs
- Maintain high test coverage (>80% for business logic)

### Testing Strategy
- Unit tests for all business logic
- Mock external dependencies using interfaces
- Cover happy path and error scenarios
- Test files adjacent to implementation files
- Use table-driven tests for multiple scenarios

---

## Technology Stack

### [doublerisefall] Technology Stack
- **Language**: Go 1.21+
- **API Protocol**: gRPC with Protocol Buffers v3
- **Configuration**: YAML with Viper (hot-reload capable)
- **Logging**: slog (structured logging)
- **Testing**: Go testing + testify/mock
- **Build**: Makefile + buf for proto generation

---

## Implementation Patterns

### Architectural Patterns
- **Dependency Inversion**: Interfaces defined where consumed (in pricer package)
- **Single Responsibility**: One package per concern
- **Stateless Design**: All state passed via request context
- **Clean Boundaries**: gRPC handlers delegate to business logic

### Dependency Direction
```
grpcsvc → pricer → (config, feed, contract)
```
- grpcsvc depends on pricer
- pricer defines interfaces
- config, feed, contract implement interfaces defined in pricer
- No circular dependencies

### Error Handling
- Return domain errors with context
- Map domain errors to gRPC status codes in grpcsvc
- Use standard error codes (ERR-DF-XXX format)
- Log errors with structured fields

### Interface Design
- Define interfaces where consumed (not in separate package)
- Keep interfaces minimal and focused
- Use internal types in interfaces (not proto types)

---

## Code Organization

### Package Structure (Standard Complexity)
```
service-pricer-{product}/
├── api/                    # Generated code (do not edit)
├── cmd/{product}/          # Entry point
├── config/                 # Configuration files
├── internal/
│   ├── app/               # Application initialization
│   ├── config/            # Config loading
│   ├── contract/          # Business rules & validation
│   ├── feed/              # External service wrapper
│   ├── grpcsvc/           # gRPC handlers
│   ├── pricer/            # Core business logic
│   └── tools/             # Build tools
└── proto/{product}/v1/    # API definitions
```

### File Naming Conventions
- Package names: lowercase, single word
- Test files: `{filename}_test.go`
- Interface definitions: In consuming package
- Proto files: `{service}.proto` in versioned directory

---

## Integration Approach

### Service-Feed Integration
- **MUST** use `github.com/regentmarkets/service-feed/client`
- Create thin wrapper in `internal/feed` package
- Handle `is_final` flag appropriately
- Implement retry logic via client configuration
- Support streaming with context cancellation

### Configuration Management
- Load YAML config at startup using Viper
- Support hot-reload capability
- Validate configuration on load
- Return errors for missing/invalid config (no defaults)

### gRPC Service Implementation
- Implement standard gRPC health check protocol
- Support graceful shutdown
- Use context for cancellation
- Implement streaming with proper cleanup
- Return standard gRPC status codes

---

## Design Decisions

### [doublerisefall] Duration Handling
- **Decision**: Single Duration type with unit discriminator
- **Rationale**: Simplifies validation and expiry calculation
- **Implementation**: `Duration{Value int64, Unit string, IsTickBased bool}`

### [doublerisefall] Pricing Formula
- **Decision**: Implement arcsin correlation formula for fair probability
- **Formula**: `P_fair = 1/4 + arcsin(√(t₁/t₂))/(2π)`
- **Rationale**: Product specification requirement
- **Precision**: Use float64 for calculations, string for API

### [doublerisefall] Win/Loss Evaluation
- **Decision**: Short-circuit evaluation at t1 failure
- **Rationale**: Performance optimization (no need to check t2 if t1 fails)
- **Implementation**: Check barrier condition at t1 before fetching t2 tick

---

## Trade-offs

### Proto Generation vs Manual Code
- **Choice**: Use buf for proto generation
- **Trade-off**: Less control but consistent with ecosystem
- **Benefit**: Standard tooling, reproducible builds

### Configuration Hot-Reload
- **Choice**: Implement using Viper watch capability
- **Trade-off**: Additional complexity for configuration changes
- **Benefit**: No service restart required for symbol config updates

### Streaming Implementation
- **Choice**: Goroutine-per-stream with context cancellation
- **Trade-off**: Memory overhead for many concurrent streams
- **Benefit**: Simple implementation, good performance for target load (1000 streams)

---

## Deployment Standards

### Containerization
- Multi-stage Dockerfile (builder + runtime)
- Alpine Linux base for minimal size
- Include CA certificates for external HTTPS calls
- Copy config files into container

### Environment Variables
- `GRPC_PORT`: gRPC server port (default: 50051)
- `FEED_SERVICE_ADDR`: service-feed address (required)
- `CONFIG_PATH`: Path to symbols.yaml (default: /config/symbols.yaml)
- `LOG_LEVEL`: Logging level (default: info)

### Health Checks
- Implement gRPC health check protocol
- Return SERVING when ready
- Check feed service connectivity in health check

---

## Version History

| Date | Service | Changes |
|------|---------|---------|
| 2026-01-15 | doublerisefall | Initial service implementation preferences |
