# Service Specifications Preferences

> **Phase**: Services Specification
> **Last Updated**: 2026-01-15
> **Status**: Active

This document captures user directives and decisions for service specifications across all services in this project.

---

## General Service Standards

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Service name | `service-pricer-{product}` | `service-pricer-doublerisefall` |
| Package name | `{product}.v1` | `doublerisefall.v1` |
| Module path | `github.com/regentmarkets/service-pricer-{product}` | `github.com/regentmarkets/service-pricer-doublerisefall` |
| Internal packages | lowercase, single word | `pricer`, `contract`, `config`, `feed` |

### File Organization

| Pattern | Application |
|---------|-------------|
| Interface-at-Consumer | Interfaces defined where consumed (in `pricer` package) |
| No interfaces/models package | Dependencies flow toward core, not shared packages |
| Test files co-located | `*_test.go` next to implementation files |

---

## Technology Stack

### Default Technology Choices

| Category | Technology | Rationale |
|----------|------------|-----------|
| **Language** | Go 1.21+ | Team standard, excellent concurrency |
| **Protocol** | gRPC/Protocol Buffers | High-performance internal communication |
| **Configuration** | YAML + Viper | Human-readable, hot-reload support |
| **Logging** | slog (stdlib) | Standard library structured logging |
| **Build Tooling** | buf | Modern protobuf toolchain |
| **Testing** | stdlib testing | Table-driven tests pattern |

### Service-Specific Overrides

_No overrides recorded yet._

---

## Module Design

### Dependency Direction

```
grpcsvc → pricer ← (contract, config, feed)
```

**Rules**:
1. `grpcsvc` depends on `pricer`
2. `pricer` defines interfaces
3. `contract`, `config`, `feed` implement interfaces
4. `pricer` MUST NOT import from other internal packages

### Standard Modules for Pricing Services

| Module | Purpose | Required When |
|--------|---------|---------------|
| `grpcsvc` | gRPC handlers | Always |
| `pricer` | Core pricing logic, interfaces | Always |
| `contract` | Duration parsing, validation | Product has business rules |
| `config` | Symbol configuration | Product has per-symbol config |
| `feed` | Market data wrapper | Product needs market data |
| `app` | DI wiring, lifecycle | Always |

---

## Implementation Guidelines

### Error Handling

| Pattern | Application |
|---------|-------------|
| Domain errors | Custom error types in each package |
| gRPC mapping | Convert domain errors to gRPC status in `grpcsvc` |
| Error codes | Format: `ERR-{SVC}-{CODE}` (e.g., `ERR-DR-S1V`) |

### Logging

| Level | Usage |
|-------|-------|
| `DEBUG` | Detailed execution flow |
| `INFO` | Request/response summaries |
| `WARN` | Recoverable issues |
| `ERROR` | Failures requiring attention |

### Testing

| Level | Coverage Target | Tools |
|-------|-----------------|-------|
| Unit | 80%+ | stdlib `testing`, table-driven |
| Integration | Key paths | Mock interfaces |
| E2E | Happy paths | `grpcurl`, test containers |

### Test ID Convention

Format: `TC-{SVC}-{PKG}{NUM}{CHAR}`

- SVC: Two-letter service code (DR for doublerisefall)
- PKG: Package indicator (P=pricer, C=contract, F=config, E=feed, G=grpcsvc)
- NUM: Test number
- CHAR: Random alphanumeric

Examples: `TC-DR-P1A`, `TC-DR-C3C`, `TC-DR-E2B`

---

## Service-Specific Preferences

### [service-pricer-doublerisefall]

| Category | Decision | Rationale |
|----------|----------|-----------|
| **Complexity Score** | 6/10 (Standard) | Multiple components but single product |
| **Internal Structure** | Light modular | Organized components without over-engineering |
| **REST Gateway** | Not included | Internal service, no external consumers |
| **Database** | None | Stateless, all data from feed/config |
| **Streaming** | Server-side only | Client subscribes, server pushes updates |

---

## User Decisions Log

| Date | Decision | Context |
|------|----------|---------|
| 2026-01-15 | Initial preferences created | First service specification (doublerisefall) |

---

## Pending Decisions

_No pending decisions._

---

> **Note**: Update this file when making technology or design decisions that should apply to future services.
