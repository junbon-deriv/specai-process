# Architecture Phase Preferences

> **Phase**: Architecture Design
> **Service**: service-pricer-doublerisefall
> **Created**: 2026-01-15
> **Status**: Active

---

## Decision Log

### User-Provided Preferences

| Decision | Value | Source | Date |
|----------|-------|--------|------|
| Service Template Patterns | Standard patterns from service_template_guide.md | User confirmation | 2026-01-15 |
| Additional Requirements | None specified | User confirmation | 2026-01-15 |

---

## Service Boundaries

### Guiding Principles

| Principle | Application |
|-----------|-------------|
| **Single Product Focus** | One pricer service per product type (Double Rise/Fall) |
| **Domain Alignment** | Service boundary matches contract pricing bounded context |
| **Stateless Design** | No persistent state in pricer, all data from feed/config |

### Boundary Decisions

| Decision | Rationale |
|----------|-----------|
| No internal microservices | Product complexity doesn't warrant service splitting |
| Modular internal packages | Clean separation of concerns without deployment overhead |
| Leaf service position | No downstream internal dependencies, only feeds external gateway |

---

## Communication Patterns

### Protocol Selection

| Pattern | When Used | Rationale |
|---------|-----------|-----------|
| **gRPC (Sync)** | GetAsk, GetBid | Type-safe, low latency, streaming support |
| **gRPC (Stream)** | StreamAsk, StreamBid | Built-in streaming, efficient bidirectional |
| **REST (Gateway)** | Optional HTTP access | grpc-gateway for debugging/compatibility |

### Error Handling

| Pattern | Application |
|---------|-------------|
| **gRPC Status Codes** | All errors mapped to standard codes |
| **No Custom Errors** | Domain errors mapped to gRPC codes only |
| **Fail Fast** | Invalid requests rejected immediately |

---

## Data Strategy

### Ownership Model

| Data Type | Owner | Access Pattern |
|-----------|-------|----------------|
| Symbol Configuration | service-pricer-doublerisefall | Load at startup, hot-reload |
| Market Data (Ticks) | service-feed | On-demand fetch |
| Contract Parameters | Request-scoped | Transient |

### Consistency Decisions

| Decision | Rationale |
|----------|-----------|
| No caching of ticks | Feed service handles caching |
| Payout immutability | Fixed at Ask time, never recalculated |
| Config hot-reload | Support runtime updates without restart |

---

## Technology Preferences

### Stack Choices

| Component | Choice | Rationale |
|-----------|--------|-----------|
| **Language** | Go 1.21+ | Service template standard |
| **API** | gRPC + Protocol Buffers | Type safety, streaming |
| **Config** | YAML + Viper | Human-readable, standard |
| **Logging** | slog | Standard library, structured |
| **Build** | Makefile + buf | Reproducible, proto tooling |

### Coding Standards

| Standard | Application |
|----------|-------------|
| **Interface Location** | Defined where consumed (in pricer) |
| **No shared types** | No models/types/interfaces package |
| **Dependency Direction** | grpcsvc → pricer → (config, feed, contract) |
| **gRPC Handlers** | Delegate to business logic, no data assembly |

---

## Step-Specific Sections

### Service Boundaries (Architecture Phase)

| Aspect | Decision |
|--------|----------|
| **Number of Services** | Single service (service-pricer-doublerisefall) |
| **Internal Packages** | 5 packages (grpcsvc, pricer, config, contract, feed) |
| **API Surface** | 4 gRPC methods (GetAsk, StreamAsk, GetBid, StreamBid) |

### Communication Patterns (Architecture Phase)

| Pattern | Usage |
|---------|-------|
| **Synchronous** | GetAsk, GetBid - single request/response |
| **Streaming** | StreamAsk, StreamBid - server-side streaming |
| **Client Pattern** | Import and wrap service-feed/client |

### Data Strategy (Architecture Phase)

| Strategy | Application |
|----------|-------------|
| **Configuration** | Local YAML, no external config service |
| **Market Data** | Direct feed dependency via wrapper |
| **State Management** | Stateless - compute on each request |

### Technology Preferences (Architecture Phase)

| Preference | Value |
|------------|-------|
| **Template** | go-templates service template |
| **Proto Package** | doublerisefall.v1 |
| **Module Path** | github.com/regentmarkets/service-pricer-doublerisefall |

---

## Open Questions

None - all architectural decisions resolved.

---

## Change History

| Date | Change | Rationale |
|------|--------|-----------|
| 2026-01-15 | Initial preferences captured | New architecture phase |
