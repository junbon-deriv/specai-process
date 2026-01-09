# Service Specifications Preferences

**Version**: 1.0  
**Date**: 2026-01-08  
**Status**: Active

---

## Overview

This document captures user preferences and decisions for all service specifications in this workspace. It serves as the authoritative source for service-wide standards and service-specific choices.

---

## 1. General Service Standards

### 1.1 Architectural Rules (STRICT)

These rules MUST be enforced across ALL services:

1. **Handler Delegation**: gRPC handlers receive requests and delegate to domain modules. Handlers MUST NOT fetch data from various sources or assemble responses directly.

2. **Standard Error Codes**: Always return gRPC standard error codes with descriptive messages.

3. **Dependency Direction**: All dependencies flow toward the core domain module. Other modules depend on core; core depends on nothing else within the service.

4. **Interface Location**: Interfaces are defined where they are consumed, not where they are implemented.

5. **No models/types Package**: No shared models package. Each module defines its own types as needed.

### 1.2 Code Standards

| Standard | Value | Rationale |
|----------|-------|-----------|
| Project Template | go-templates (https://github.com/junbon-deriv/go-templates) | Standard project scaffold |
| Code Style | golangci-lint defaults | Consistent formatting |
| Test Coverage Target | 80% | Balance of coverage and effort |
| Documentation | GoDoc comments for exported types | API documentation |

---

## 2. Service Structure

### 2.1 Directory Layout

All services follow the go-templates structure:

```
{service}/
├── cmd/{service}/main.go    # Entry point (go-templates)
├── internal/
│   ├── app/                  # Application setup, wiring
│   ├── grpcsvc/              # gRPC handlers (delegate only)
│   ├── {core}/               # Core domain module
│   └── {supporting}/         # Supporting modules
├── api/{service}/            # Generated protobuf code
├── proto/{service}/v1/       # Protobuf definitions
├── config/                   # Configuration files
├── Makefile                  # Build targets
├── Dockerfile                # Container definition
└── README.md                 # Documentation
```

### 2.2 Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Service Directory | lowercase, single word | `digitalcallput` |
| Module Directories | lowercase, descriptive | `pricing`, `contract`, `market` |
| Proto Package | `{service}.v1` | `digitalcallput.v1` |
| Go Package Path | `github.com/regentmarkets/service-pricer-{service}` | per preferences |

---

## 3. Technology Stack

### 3.1 Core Technologies

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go 1.21+ | Performance, concurrency, team expertise |
| API Protocol | gRPC | Streaming support, binary efficiency |
| Serialization | Protocol Buffers (proto3) | Schema evolution |
| Configuration | Viper + Cobra | Standard Go tools (go-templates) |
| Logging | log/slog | Structured logging, stdlib |
| Decimal Math | shopspring/decimal | Financial precision |
| Build | Make | Standard Go tooling |
| Containerization | Docker | Deployment consistency |

### 3.2 External Dependencies

| Dependency | Repository | Purpose |
|------------|------------|---------|
| service-feed | https://github.com/junbon-deriv/service-feed | Market data provider |

**CRITICAL**: Before integrating, clone and verify actual API from proto files.

---

## 4. Module Design

### 4.1 Module Patterns

| Pattern | When to Use |
|---------|-------------|
| Core Domain Module | Business logic with no internal dependencies |
| Supporting Module | Domain logic that depends on core |
| Infrastructure Module | External integrations (feeds, databases) |
| Configuration Module | Static configuration loading |

### 4.2 Interface Design

- Interfaces defined in consuming module, not provider
- Keep interfaces minimal (1-3 methods preferred)
- Use context.Context for all external calls
- Return errors, not panics

Example:
```go
// pricing/pricing.go (consumer defines interface)
type MarketDataProvider interface {
    GetLatestTick(ctx context.Context, symbol string) (*Tick, error)
}
```

---

## 5. Implementation Guidelines

### 5.1 Error Handling

| Domain Error | gRPC Code | Usage |
|--------------|-----------|-------|
| Validation errors | INVALID_ARGUMENT | Missing/invalid input |
| Not found | NOT_FOUND | Unknown resource |
| External unavailable | UNAVAILABLE | Dependency down |
| Calculation errors | INTERNAL | Unexpected failures |

### 5.2 Logging

| Level | Usage |
|-------|-------|
| DEBUG | Detailed troubleshooting |
| INFO | Normal operations, startup/shutdown |
| WARN | Recoverable issues |
| ERROR | Failures requiring attention |

### 5.3 Testing Strategy

| Test Type | Target | Tools |
|-----------|--------|-------|
| Unit | Individual functions | go test |
| Integration | Module interactions | testcontainers |
| E2E | Full service flow | grpcurl |

### 5.4 Test Case ID Format

Format: `TC-[SERVICE]-[3CHAR]`

| Service | Code | Example |
|---------|------|---------|
| digitalcallput | DC | TC-DC-A1B |

---

## 6. Deployment Standards

### 6.1 Environment Variables

All services use these standard environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| GRPC_ADDRESS | gRPC bind address | :8090 |
| HTTP_ADDRESS | HTTP gateway address | :8080 |
| LOG_LEVEL | Log level | INFO |
| LOG_TEXT_FORMAT | Text vs JSON logs | false |

### 6.2 Health Checks

- gRPC: `/grpc.health.v1.Health/Check`
- HTTP: `GET /health`

### 6.3 Resource Recommendations

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 0.5 cores | 2 cores |
| Memory | 256MB | 512MB |
| Replicas | 2 | 3+ |

---

## 7. Service-Specific Preferences

### 7.1 [digitalcallput] Service

| Decision | Value | Rationale |
|----------|-------|-----------|
| Core Module | pricing | Black-Scholes calculations |
| Supporting Modules | contract, market, config | Separation of concerns |
| Template | go-templates/service | gRPC service with gateway |
| Decimal Precision | 8 places | Financial requirement |
| Stateless Design | Yes | No persistent storage |

**User Input** (from checkpoint):
- Follow go-templates structure as specified in workspace/input/preferences.md
- Service code structure should match go-templates patterns

---

## 8. Changelog

| Date | Change | Author |
|------|--------|--------|
| 2026-01-08 | Initial preferences document | Service Architect |
| 2026-01-08 | Added digitalcallput preferences | Service Architect |
