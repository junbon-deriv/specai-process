# API Specification Preferences

> **Version**: 1.0.0
> **Created**: 2026-01-15
> **Last Updated**: 2026-01-15

---

## Overview

This document tracks API design decisions and user preferences for all API specifications across services in this workspace.

---

## General API Standards

| Decision | Value | Rationale |
|----------|-------|-----------|
| **Protocol** | gRPC (Protocol Buffers v3) | Type-safe contracts, streaming support, performance |
| **Versioning Strategy** | Package versioning (v1, v2) | Clean proto organization, backward compatibility |
| **Field Naming** | snake_case | Proto3 standard convention |
| **Numeric Precision** | String representation | Preserve decimal precision for financial values |

---

## Endpoint Design Patterns

| Pattern | Standard | Notes |
|---------|----------|-------|
| **RPC Naming** | Verb + Noun (GetAsk, StreamBid) | Clarity of operation |
| **Streaming** | Server streaming for real-time updates | Unary for single requests |
| **Endpoint IDs** | API-[SERVICE]-[3CHAR] format | Unique identification |

---

## Data Model Strategy

| Decision | Standard | Example |
|----------|----------|---------|
| **Financial Values** | String type | `"10.00"` not `10.00` |
| **Timestamps** | Unix epoch (int64) | `1736916600` |
| **Durations** | Human-readable strings | `"1m"`, `"30s"`, `"5t"` |
| **Enums** | UNSPECIFIED as value 0 | `CONTRACT_TYPE_UNSPECIFIED = 0` |

---

## Error Handling

| Decision | Standard |
|----------|----------|
| **Error Format** | gRPC status codes with detailed messages |
| **Error Code Format** | ERR-[SERVICE]-[3CHAR] |
| **Validation Errors** | INVALID_ARGUMENT (code 3) |
| **Precondition Failures** | FAILED_PRECONDITION (code 9) |
| **Availability Errors** | UNAVAILABLE (code 14) |
| **Internal Errors** | INTERNAL (code 13) |

---

## Service-Specific Preferences

### [service-pricer-doublerisefall] Internal API

| Decision | Value | Date |
|----------|-------|------|
| **Authentication** | None (internal service) | 2026-01-15 |
| **Service Code** | DF (DoubleRiseFall) | 2026-01-15 |
| **Default Port** | 50051 | 2026-01-15 |
| **Health Check** | gRPC health protocol | 2026-01-15 |

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-15 | Initial preferences document for service-pricer-doublerisefall internal API |
