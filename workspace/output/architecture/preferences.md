# Architecture Preferences: Double Rise/Fall Pricing Service

> **Version**: 1.0.0
> **Created**: 2026-01-15
> **Phase**: Architecture

---

## Overview

This document captures user preferences and architectural decisions for the `service-pricer-doublerisefall` service. These preferences guide the architecture design and should be consulted during implementation.

---

## Decision Categories

### 1. Service Boundaries

| Decision | Value | Rationale | User Confirmed |
|----------|-------|-----------|----------------|
| **Service Count** | Single service | Focused product scope, clear domain boundary | ✅ Yes |
| **Service Name** | `service-pricer-doublerisefall` | Follows template guide naming convention | ✅ Yes |
| **Boundary Principle** | One service per product type | Standard pricing service pattern | ✅ Yes |

### 2. Communication Patterns

| Decision | Value | Rationale | User Confirmed |
|----------|-------|-----------|----------------|
| **Primary Protocol** | gRPC | Internal service, high performance required | ✅ Yes |
| **REST API** | Not required | Internal consumption only | ✅ Yes |
| **Streaming** | Server-side streaming | Real-time price updates | ✅ Yes |
| **Feed Integration** | Use `service-feed/client` wrapper | Per dependency specification | ✅ Yes |

### 3. Data Strategy

| Decision | Value | Rationale | User Confirmed |
|----------|-------|-----------|----------------|
| **Database** | None required | Stateless pricing service | ✅ Yes |
| **Configuration** | YAML files | Simple, hot-reloadable | ✅ Yes |
| **Market Data** | External (`service-feed`) | Single source of truth | ✅ Yes |
| **Caching** | None | Always fetch fresh market data | ✅ Yes |

### 4. Technology Preferences

| Decision | Value | Rationale | User Confirmed |
|----------|-------|-----------|----------------|
| **Language** | Go 1.21+ | Team standard | ✅ Yes |
| **Proto Toolchain** | buf | Modern, standard | ✅ Yes |
| **Logging** | slog | Go standard library | ✅ Yes |
| **Configuration** | Viper | Hot-reload capability | ✅ Yes |

---

## Step-Specific Sections

### Service Boundaries

**Guiding Principles**:
- One pricing service per derivative product type
- Service owns all pricing logic, validation, and configuration
- External dependencies accessed through wrapper interfaces
- No shared databases with other services

**Special Cases**:
- None identified for this product

### Communication Patterns

**Synchronous Communication**:
- gRPC for all service interactions
- Unary calls: `GetAsk`, `GetBid`
- Used for: Single price requests

**Asynchronous/Streaming Communication**:
- Server streaming: `StreamAsk`, `StreamBid`
- Update triggers: On tick OR every 5 seconds (time-based), on tick only (tick-based)

**Event-Driven Patterns**:
- Not applicable for this service

### Data Strategy

**Data Ownership**:
| Entity | Owner | Storage |
|--------|-------|---------|
| SymbolConfig | This service | YAML |
| Contract (request scope) | This service | Memory |
| Market Data | service-feed | External |

**Consistency Patterns**:
- Strong consistency for configuration reads
- Eventual consistency acceptable for market data (always fetch fresh)

**Replication Strategy**:
- Not applicable (stateless service)

### Technology Preferences

**Stack Choices**:
| Layer | Choice | Alternative Considered |
|-------|--------|----------------------|
| Language | Go | N/A (team standard) |
| Protocol | gRPC | REST (not needed for internal) |
| Config | YAML + Viper | JSON (less readable) |
| Logging | slog | zerolog (external dep) |

**API Standards**:
- gRPC with Protocol Buffers v3
- Standard gRPC error codes (no custom errors)
- Decimal values as strings for precision

**Infrastructure Preferences**:
- Default to service template patterns
- No specific deployment requirements captured

---

## User Input Summary

| Checkpoint | Question | Response | Date |
|------------|----------|----------|------|
| Initial | Service boundaries, communication patterns, technology preferences | No special preferences - proceed with standard approach | 2026-01-15 |

---

## Change Log

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-15 | Initial preferences document |

---

> **Last Updated**: 2026-01-15
