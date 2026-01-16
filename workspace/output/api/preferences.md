# API Preferences

> **Phase**: API Specification
> **Last Updated**: 2026-01-15
> **Status**: Active

---

## General API Standards

### Protocol Preferences
| Preference | Value | Rationale |
|------------|-------|-----------|
| **Internal API Protocol** | gRPC / Protocol Buffers | High-performance binary protocol for service-to-service communication |
| **Public API Protocol** | REST / JSON | N/A - no public APIs in this project |
| **Versioning Strategy** | Package versioning (v1, v2) | Standard gRPC versioning pattern |

### Naming Conventions
| Element | Convention | Example |
|---------|------------|---------|
| **Package Name** | `{service}.v{version}` | `doublerisefall.v1` |
| **Service Name** | `{Product}Service` | `DoubleRiseFallService` |
| **RPC Methods** | PascalCase, verb prefix | `GetAsk`, `StreamBid` |
| **Message Names** | PascalCase | `GetAskRequest`, `GetAskResponse` |
| **Field Names** | snake_case | `current_spot`, `start_time` |
| **Enum Values** | SCREAMING_SNAKE_CASE with prefix | `CONTRACT_TYPE_RISE` |

### Field Format Standards
| Data Type | Format | Example |
|-----------|--------|---------|
| **Monetary Values** | String (decimal) | `"10.00"`, `"1234.5678"` |
| **Timestamps** | int64 (Unix epoch seconds) | `1736930731` |
| **Durations** | String with unit suffix | `"30s"`, `"1m"`, `"5t"` |
| **Prices** | String, 4 decimal precision | `"0.3589"` |
| **Payouts** | String, 2 decimal precision | `"27.85"` |

---

## Endpoint Design Patterns

### Request/Response Patterns
| Pattern | Usage | Rationale |
|---------|-------|-----------|
| **Unary RPC** | Single request/response | Simple queries like `GetAsk`, `GetBid` |
| **Server Streaming** | Real-time updates | Price streaming via `StreamAsk`, `StreamBid` |
| **Client Streaming** | N/A | Not used in this project |
| **Bidirectional** | N/A | Not used in this project |

### Endpoint ID Format
| Component | Format | Example |
|-----------|--------|---------|
| **Prefix** | `API-` | `API-` |
| **Service Code** | 2 uppercase letters | `DR` (Double Rise/Fall) |
| **Unique ID** | 3 alphanumeric chars | `A1K`, `S2T` |
| **Full Format** | `API-{SVC}-{ID}` | `API-DR-A1K` |

---

## Data Model Strategy

### Message Structure
| Aspect | Preference | Rationale |
|--------|------------|-----------|
| **Nesting** | Flat with shared messages | Reusable `OptionParameters` message |
| **Optional Fields** | Use `optional` keyword | Explicit optionality for proto3 |
| **Default Values** | Avoid relying on defaults | Always validate required fields |

### Shared Messages
| Message | Usage |
|---------|-------|
| `OptionParameters` | Common contract parameters for all RPCs |
| `Limits` | Trading limits returned with Ask responses |

---

## Error Handling

### Error Code Format
| Component | Format | Example |
|-----------|--------|---------|
| **Prefix** | `ERR-` | `ERR-` |
| **Service Code** | 2 uppercase letters | `DR` (Double Rise/Fall) |
| **Unique ID** | 3 alphanumeric chars | `S1V`, `D2U` |
| **Full Format** | `ERR-{SVC}-{ID}` | `ERR-DR-S1V` |

### gRPC Status Code Mapping
| Scenario | gRPC Status |
|----------|-------------|
| Validation errors | `INVALID_ARGUMENT` |
| Missing prerequisites | `FAILED_PRECONDITION` |
| External service unavailable | `UNAVAILABLE` |
| Internal errors | `INTERNAL` |
| Not found | `NOT_FOUND` |

### Error Response Structure
```protobuf
message ErrorDetail {
  string code = 1;              // ERR-{SVC}-{ID}
  string message = 2;           // Human-readable message
  map<string, string> metadata = 3;  // Additional context
}
```

---

## Authentication (Internal APIs)

### Service-to-Service Auth
| Aspect | Preference |
|--------|------------|
| **Method** | mTLS (mutual TLS) |
| **Certificate Management** | Kubernetes secrets |
| **Authorization** | Service mesh policies |

---

## Service-Specific Preferences

### [doublerisefall] Internal API

| Preference | Value | Notes |
|------------|-------|-------|
| **Protocol** | gRPC only | No REST gateway |
| **Port** | 50051 | Standard gRPC port |
| **Health Port** | 8081 | HTTP health/metrics |
| **Stream Update Frequency** | On tick OR every 5s (time-based), on tick only (tick-based) | Per product requirements |
| **Allowed Consumers** | api-gateway-trading | Single consumer |

---

## Decision Log

| Date | Decision | Context | Made By |
|------|----------|---------|---------|
| 2026-01-15 | Use string for monetary values | Avoid floating-point precision issues | AI Generated |
| 2026-01-15 | Use Unix epoch seconds for timestamps | Simplicity and language-agnostic | AI Generated |
| 2026-01-15 | Use shared `OptionParameters` message | DRY principle, consistent validation | AI Generated |
| 2026-01-15 | Stream terminates after expiry | Automatic cleanup, resource efficiency | AI Generated |

---

> **Version**: 1.0.0
> **Created**: 2026-01-15
