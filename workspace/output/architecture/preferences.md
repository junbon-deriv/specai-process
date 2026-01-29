# Architecture Phase Preferences

## Overview
This document captures all decisions, directives, and clarifications made during the service architecture design phase for the Deriv Arcade project.

---

## Inherited Preferences

### From Workspace Preferences (workspace/input/preferences.md)

| ID | Type | Preference | Impact |
|----|------|------------|--------|
| WP-1 | Directive | Backend is a Golang service exposing REST API with JSON | Single Golang backend service |
| WP-2 | Directive | PostgreSQL as database | Shared database for all modules |
| WP-3 | Directive | Frontend to use React JS | Direct API calls to backend |

### From Requirements Preferences (workspace/output/requirements/preferences.md)

| ID | Type | Preference | Impact |
|----|------|------------|--------|
| PD-1 | Decision | Payout = Stake / 0.53 | Implemented in Trading module |
| PD-2 | Decision | Four series types (Vol50, Vol100, Vol200, Vol300) | GBM configuration |
| PD-7 | Decision | 2 decimal precision for monetary amounts | Database decimal(18,2) |
| PD-8 | Decision | Account ID format: SW prefix | Sequential ID generation |
| PD-10 | Decision | No authentication in Phase 1 | APIs accessible without auth |

### Complexity Score

| Metric | Value | Impact |
|--------|-------|--------|
| PRD Complexity Score | 4/10 | Light modular organization (not full enterprise modules) |
| Internal Structure | Score 4-7 template | Separate files for routes, business logic, data access |

---

## Architecture Decisions

### AD-1: Modular Monolith Architecture

**Type**: User Approved  
**Decision**: Single Golang service with two internal modules (accounts, trading)  
**Context**: User confirmed preference for single service approach when presented with options  
**Alternatives Considered**:
- Two separate microservices (rejected due to distributed transaction complexity)

**Rationale**:
1. Atomic trade execution requires transactions spanning both domains
2. Complexity score of 4/10 doesn't warrant microservices overhead
3. Aligns with workspace preference "Backend is a Golang service" (singular)
4. Simplifies deployment and operational complexity

### AD-2: Shared Database

**Type**: Derived from AD-1  
**Decision**: Single PostgreSQL database serving both modules  
**Context**: Required for atomic cross-module transactions  

**Rationale**:
1. Trade execution must be atomic (stake + payout in single transaction)
2. Eliminates distributed transaction complexity
3. PostgreSQL JSONB supports OHLC data storage

### AD-3: Direct Service APIs

**Type**: Assumption  
**Decision**: No API gateway; Golang service exposes APIs directly to React frontend  
**Context**: Single service architecture doesn't require API aggregation  

**Rationale**:
1. Single service means no routing complexity
2. Reduces infrastructure requirements
3. Appropriate for Phase 1 without authentication

### AD-4: Quote Validation via previous_quote

**Type**: Derived from PRD  
**Decision**: Use `previous_quote` parameter to match PriceSeries for validation  
**Context**: SwipeBuy must validate against correct SwipeGet response  

**Implementation**:
- SwipeGet stores PriceSeries with `quote_value` = 10th candle close
- SwipeBuy finds PriceSeries by `account_id` + `series_type` + `quote_value`
- Matched PriceSeries deleted after successful trade

### AD-5: PriceSeries Cleanup

**Type**: Assumption  
**Decision**: Background cleanup of orphaned PriceSeries records older than 5 minutes  
**Context**: PriceSeries may be created but never consumed (user abandons session)  

**Implementation**: Simple cron job or database scheduled task

---

## Service Boundaries

### Module: accounts

| Attribute | Value |
|-----------|-------|
| **Domain Alignment** | DOM-AC-K3M (Accounts Domain) |
| **Entities Owned** | Account, Transaction |
| **Core Responsibilities** | Account creation, balance management, idempotent financial operations |
| **Story Count** | 12 user stories |

### Module: trading

| Attribute | Value |
|-----------|-------|
| **Domain Alignment** | DOM-TR-L8K (Trading Domain) |
| **Entities Owned** | PriceSeries, Contract |
| **Core Responsibilities** | GBM price generation, contract execution, history display |
| **Story Count** | 13 user stories |

### Cross-Module Interaction

| Pattern | Description |
|---------|-------------|
| **Internal Function Calls** | Trading module calls Accounts module functions directly |
| **Single Transaction** | SwipeBuy spans both modules within one DB transaction |
| **No Message Queue** | Synchronous calls only (appropriate for atomic requirements) |

---

## Communication Patterns

### Inter-Module Communication

| Communication Type | Usage | Rationale |
|--------------------|-------|-----------|
| Synchronous function calls | All inter-module calls | Single service, no network latency |
| Database transactions | Trade execution | Atomic consistency required |

### External Communication (Frontend to Backend)

| Pattern | Endpoints | Description |
|---------|-----------|-------------|
| REST/JSON | All /accounts/*, /swipe/* | Standard HTTP APIs |
| Stateless | All endpoints | No server-side session |

---

## Data Strategy

### Database Selection

| Attribute | Value |
|-----------|-------|
| **Database** | PostgreSQL |
| **Version** | 14+ (recommended) |
| **Features Used** | JSONB for OHLC data, unique constraints for idempotency |

### Data Ownership Matrix

| Entity | Owner Module | Read Access | Write Access |
|--------|--------------|-------------|--------------|
| Account | accounts | accounts, trading | accounts |
| Transaction | accounts | accounts | accounts |
| PriceSeries | trading | trading | trading |
| Contract | trading | trading | trading |

### Consistency Guarantees

| Operation | Consistency Level | Implementation |
|-----------|------------------|----------------|
| Deposit/Withdrawal | Strong (ACID) | Single table transaction |
| Trade Execution | Strong (ACID) | Multi-table transaction |
| Price Preview | Eventual | Can be regenerated |

---

## Technology Preferences

### Backend Stack

| Component | Technology | Version/Notes |
|-----------|------------|---------------|
| Language | Golang | Latest stable |
| HTTP Framework | Standard library / Chi / Gin | No preference (any is acceptable) |
| Database Driver | pgx or database/sql | No preference |
| JSON | encoding/json | Standard library |

### Database Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| RDBMS | PostgreSQL | 14+ recommended |
| Migrations | golang-migrate / goose | No preference |
| ORM | None (raw SQL preferred) | Complexity score doesn't warrant ORM |

### Infrastructure

| Component | Specification | Notes |
|-----------|--------------|-------|
| Port | 8080 | Default, configurable via ENV |
| Health Check | GET /health | Returns {"status": "healthy"} |
| Logging | Structured JSON logs | No specific framework required |

---

## Scalability Considerations

### Current Architecture (Phase 1)

| Aspect | Approach |
|--------|----------|
| Horizontal Scaling | Single instance sufficient |
| Database | Single PostgreSQL instance |
| Caching | None required (low latency requirements met) |

### Future Considerations (Phase 2+)

| Aspect | Potential Approach |
|--------|-------------------|
| Horizontal Scaling | Stateless design supports multiple instances |
| Database | Read replicas for SwipeList if needed |
| Caching | Redis for session/rate limiting if authentication added |

---

## Constraints and Boundaries

### In Scope (Phase 1)

- Account CRUD operations
- Idempotent deposits/withdrawals
- GBM price generation
- Rise/fall contract execution
- Trading history display
- REST API for React frontend

### Out of Scope (Phase 1)

- User authentication/authorization
- API rate limiting
- WebSocket real-time updates
- Multi-region deployment
- Analytics/reporting
- Admin interfaces

---

## Open Questions (Resolved)

| Question | Resolution | Date |
|----------|------------|------|
| Single service vs microservices? | Single service with modules | 2026-01-16 |
| API gateway needed? | No, direct service APIs | 2026-01-16 |
| How to validate quotes? | previous_quote parameter matching | 2026-01-16 |

---

## Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-16 | Archi | Initial architecture preferences creation |
