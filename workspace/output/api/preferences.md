# API Specification Preferences

## Document Information
| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2026-01-16 |
| Last Updated | 2026-01-16 |
| Status | Active |

---

## Purpose

This document captures API design decisions and preferences that apply across all service API specifications in the Deriv Arcade project.

---

## General API Standards

### Protocol & Format
| Decision | Value | Rationale |
|----------|-------|-----------|
| Protocol | REST/HTTP | Workspace preference, simplicity for single-service architecture |
| Content Type | application/json | Workspace preference, universal support |
| Character Encoding | UTF-8 | Industry standard |

### Versioning Strategy
| Decision | Value | Rationale |
|----------|-------|-----------|
| Strategy | URL path versioning (future: /v1/) | Simple, explicit, easy to understand |
| Current Version | Unversioned (Phase 1) | Simplicity for initial development |

---

## Endpoint Design Patterns

### Naming Conventions
| Pattern | Example | Rationale |
|---------|---------|-----------|
| Resource-based paths | `/accounts`, `/swipe` | RESTful clarity |
| Plural nouns for collections | `/accounts`, not `/account` | Standard REST convention |
| Nested resources for actions | `/accounts/{id}/deposits` | Clear resource relationships |
| Query parameters for filters | `?series_type=Vol100` | Standard filtering pattern |

### HTTP Methods
| Method | Usage | Idempotent |
|--------|-------|------------|
| GET | Read operations | Yes |
| POST | Create resources, execute actions | Depends (idempotent for deposits/withdrawals) |
| PUT | Not used (Phase 1) | N/A |
| DELETE | Not used (Phase 1) | N/A |
| PATCH | Not used (Phase 1) | N/A |

### URL Parameter Style
| Decision | Value | Rationale |
|----------|-------|-----------|
| Path parameters | For resource identification (`{account_id}`) | Clear resource addressing |
| Query parameters | For filtering and optional data | Standard for GET requests |

---

## Data Model Strategy

### Field Naming
| Decision | Value | Example |
|----------|-------|---------|
| Case | snake_case | `account_id`, `series_type` |
| Clarity | Full descriptive names | `transaction_time`, not `tx_time` |

### Numeric Representation
| Type | Format | Precision | Rationale |
|------|--------|-----------|-----------|
| Monetary amounts | String | 2 decimals | Avoid floating-point precision issues |
| OHLC prices | String | 3 decimals | Per series configuration requirement |
| IDs (account) | String | N/A | SW-prefixed format |
| IDs (other) | Integer | N/A | Simple sequential integers |

### Timestamp Format
| Decision | Value | Example |
|----------|-------|---------|
| Format | ISO8601 | `2026-01-16T10:30:00Z` |
| Timezone | UTC (Z suffix) | Always UTC for consistency |

### Null Handling
| Decision | Value | Rationale |
|----------|-------|-----------|
| Optional fields | Omit if null | Cleaner responses |
| Required fields | Always present | Clear contract |

---

## Error Handling

### Error Response Structure
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

### Error Code Format
| Decision | Value | Example |
|----------|-------|---------|
| Format | ERR-[SERVICE]-[3CHAR] | ERR-AC-N4F, ERR-TR-Q4N |
| Service codes | AC (Accounts), TR (Trading) | Two-letter service identifier |

### HTTP Status Usage
| Status | Usage |
|--------|-------|
| 200 | Success, including idempotent duplicate requests |
| 201 | Resource created (accounts, contracts) |
| 400 | Client errors (validation, business rules) |
| 404 | Resource not found |
| 500 | Server errors (unexpected) |

---

## Authentication (Phase 1)

| Decision | Value | Rationale |
|----------|-------|-----------|
| Authentication | None | Phase 1 scope excludes auth |
| Rate Limiting | None | Phase 1 scope excludes rate limiting |
| API Keys | Not implemented | Future consideration |

---

## Service-Specific Preferences

### [arcade] Public API

| Aspect | Decision | Notes |
|--------|----------|-------|
| Account ID format | SW-prefixed sequential | SW1, SW2, SW3... |
| Idempotency mechanism | Client-provided UUID | deposit_id, withdrawal_id fields |
| Trade execution | Atomic, synchronous | All-or-nothing within single request |
| Price preview | Temporary storage | Deleted after contract creation |
| Quote validation | previous_quote matching | 10th candle close value |
| History limit | 50 contracts maximum | Performance consideration |
| OHLC storage | Full 20 candles per contract | Audit and replay capability |

---

## Future Considerations

| Topic | Consideration | Priority |
|-------|---------------|----------|
| API versioning | Implement /v1/ prefix | Medium |
| Authentication | JWT or API keys | High for production |
| Rate limiting | Per-account or per-IP limits | High for production |
| Pagination | Cursor-based for history | Medium |
| WebSocket | Real-time price updates | Low |

---

## Decision Log

| Date | Decision | Context |
|------|----------|---------|
| 2026-01-16 | Use string format for monetary amounts | Avoid floating-point precision issues |
| 2026-01-16 | Client-provided idempotency keys | Simple, transparent deduplication |
| 2026-01-16 | Synchronous trade execution | Phase 1 simplicity, no message queues |
| 2026-01-16 | 50-contract history limit | Balance between completeness and performance |
| 2026-01-16 | No authentication in Phase 1 | Per PRD scope definition |
