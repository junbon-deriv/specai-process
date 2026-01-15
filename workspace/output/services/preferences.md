# Service Specifications Preferences

> **Last Updated**: 2026-01-15
> **Scope**: All service specifications in this workspace

---

## General Service Standards

### Language & Framework

| Preference | Value | Rationale |
|------------|-------|-----------|
| **Primary Language** | Go 1.21+ | Performance, gRPC support, type safety |
| **API Protocol** | gRPC + Protocol Buffers v3 | Type-safe contracts, streaming support |
| **Configuration** | YAML + Viper | Human-readable, hot-reload capable |
| **Logging** | slog (structured) | Standard library, JSON output |
| **Testing** | Go testing + testify | Native framework with assertions |

### Code Conventions

| Convention | Standard |
|------------|----------|
| **Package Naming** | Lowercase, single word (e.g., `pricer`, `grpcsvc`) |
| **Interface Location** | Defined where consumed, not in shared packages |
| **Error Handling** | Wrap errors with context, use domain error types |
| **ID Format** | `TC-[SERVICE]-[3CHAR]` for test cases |

---

## Service Structure

### Module Organization

| Preference | Value |
|------------|-------|
| **Interface Definition** | Interfaces defined where consumed (not centralized) |
| **Package Isolation** | Each package handles one concern |
| **Proto Types** | Never expose proto types in interfaces; use internal types |

### Directory Structure Pattern

```
service-name/
├── api/           # Generated gRPC code (do not edit)
├── cmd/           # Application entry points
├── config/        # Configuration files (YAML)
├── internal/      # Private packages
│   ├── app/       # Application initialization
│   ├── grpcsvc/   # gRPC handlers
│   └── ...        # Domain-specific packages
├── proto/         # Proto definitions
├── Dockerfile
├── Makefile
└── README.md
```

---

## Technology Stack

### Service-Specific Preferences

#### [doublerisefall] Technology Stack

| Component | Choice | Notes |
|-----------|--------|-------|
| Language | Go 1.21+ | Standard for pricing services |
| API | gRPC | Server streaming for real-time prices |
| Config | YAML + Viper | Hot-reload for symbol configuration |
| External Client | service-feed/client | **MANDATORY**: Use provided client, not direct gRPC |

---

## Module Design

### Dependency Patterns

| Pattern | Description |
|---------|-------------|
| **Dependency Inversion** | Core logic defines interfaces, infrastructure implements |
| **Thin Wrappers** | External clients wrapped for testability |
| **No Circular Dependencies** | Clear directional flow between packages |

### Common Package Responsibilities

| Package | Typical Responsibility |
|---------|------------------------|
| `grpcsvc` | Request handling, response formatting, error mapping |
| `config` | Configuration loading, validation, accessor methods |
| `feed` | External service client wrappers |
| Core logic | Business rules, calculations, domain validation |

---

## Implementation Guidelines

### Error Handling

| Guideline | Description |
|-----------|-------------|
| **Error Codes** | Use service-specific prefix (e.g., `ERR-DF-` for doublerisefall) |
| **gRPC Mapping** | Map domain errors to appropriate gRPC status codes |
| **Logging** | Log errors with structured context, no PII |

### Testing

| Guideline | Description |
|-----------|-------------|
| **Unit Tests** | Test business logic with mocked dependencies |
| **Coverage Target** | 80%+ for business logic packages |
| **Mock Pattern** | Use interface-based mocks with testify |

### Deployment

| Guideline | Description |
|-----------|-------------|
| **Container** | Multi-stage Docker build (builder + alpine) |
| **Health Check** | gRPC health check protocol |
| **Probes** | readinessProbe and livenessProbe on gRPC port |

---

## Changelog

| Date | Change | Service |
|------|--------|---------|
| 2026-01-15 | Initial preferences created | doublerisefall |
