# Service Specifications Preferences

## Document Information
| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2026-01-16 |
| Last Updated | 2026-01-16 |
| Status | Active |

---

## Purpose

This document records user decisions and preferences that guide service specification creation. These preferences are applied consistently across all services in the Deriv Arcade project.

---

## 1. General Service Standards

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Service Name | lowercase, single word | arcade |
| File Names | snake_case | accounts_handler.go |
| Package Names | lowercase, no underscores | accounts |
| Interface Names | PascalCase with suffix | AccountService |
| Error Codes | UPPER_SNAKE_CASE | ACCOUNT_NOT_FOUND |

### ID Formats

| ID Type | Format | Example |
|---------|--------|---------|
| Service ID | SVC-[2CHAR]-[3CHAR] | SVC-AR-K3M |
| Test Case ID | TC-[SERVICE]-[3CHAR] | TC-AC-J6Q |
| Error Code ID | ERR-[SERVICE]-[3CHAR] | ERR-AC-N4F |

---

## 2. Service Structure

### Architecture Pattern

| Decision | Value | Rationale |
|----------|-------|-----------|
| Architecture Style | Modular Monolith | Single service with internal modules; appropriate for complexity score 4/10 |
| Internal Organization | Layered (API → Service → Repository) | Clear separation of concerns; enables testability |
| Module Communication | Direct function calls | No network overhead; within single process |

### Directory Structure

| Decision | Value | Rationale |
|----------|-------|-----------|
| Root Layout | `cmd/`, `internal/`, `migrations/`, `config/` | Standard Go project layout |
| Module Separation | Separate packages per module | Clear boundaries; independent testing |
| Handler Location | `internal/api/` | All HTTP handlers in one place |
| Business Logic | `internal/{module}/service.go` | Clear service layer per module |
| Data Access | `internal/{module}/repository.go` | Repository pattern per module |

---

## 3. Technology Stack

### [arcade] Backend Technology

| Layer | Technology | Version | Rationale |
|-------|------------|---------|-----------|
| Language | Golang | 1.21+ | Workspace preference; strong concurrency |
| HTTP Router | Chi | v5.x | Lightweight, idiomatic Go |
| Database | PostgreSQL | 14+ | Workspace preference; JSONB support |
| DB Driver | pgx | v5.x | Native PostgreSQL driver; better performance |
| Decimal | shopspring/decimal | v1.x | Precise monetary calculations |
| Migrations | golang-migrate | v4.x | Version-controlled migrations |
| Config | Viper | v1.x | Environment variable management |
| Logging | zerolog | v1.x | High-performance structured logging |
| Testing | testify | v1.x | Assertion helpers |

---

## 4. Module Design

### Inter-Module Communication

| Decision | Value | Rationale |
|----------|-------|-----------|
| Pattern | Interface-based | Enables mocking for tests; clear contracts |
| Transaction Scope | Single database transaction | Atomic operations across modules |
| Context Propagation | context.Context with tx | Standard Go pattern; carries transaction |

### Module Dependencies

| Consumer Module | Provider Module | Purpose |
|-----------------|-----------------|---------|
| Trading | Accounts | Balance validation, stake/payout operations |

---

## 5. Implementation Guidelines

### Error Handling

| Decision | Value | Rationale |
|----------|-------|-----------|
| Error Pattern | Sentinel errors + wrapping | Clear error types; stack trace preservation |
| API Error Format | `{"error": {"code": "...", "message": "..."}}` | Consistent client-side handling |
| HTTP Status Codes | Semantic (400, 404, 500) | RESTful conventions |

### Testing Strategy

| Test Type | Approach | Tools |
|-----------|----------|-------|
| Unit Tests | Mock interfaces | testify/mock |
| Integration Tests | Testcontainers | testcontainers-go |
| API Tests | Full HTTP requests | httptest |

### Database Practices

| Practice | Implementation | Rationale |
|----------|----------------|-----------|
| Migrations | Versioned SQL files | Reproducible schema changes |
| Connection Pooling | pgx pool (5-25 connections) | Efficient resource usage |
| Row Locking | SELECT FOR UPDATE | Prevent race conditions on balance |
| Idempotency | Unique constraint on idempotency_id | Safe retries |

### Deployment Approach

| Aspect | Approach | Rationale |
|--------|----------|-----------|
| Containerization | Docker multi-stage build | Small image size |
| Configuration | Environment variables | 12-factor app compliance |
| Health Check | GET /health endpoint | Kubernetes readiness/liveness |
| Database Init | Migrations at deploy time | Automated schema setup |

---

## 6. Service-Specific Preferences

### [arcade] Service Decisions

| Decision | Value | Date | Rationale |
|----------|-------|------|-----------|
| Module Count | 2 (Accounts, Trading) | 2026-01-16 | Aligns with domain boundaries |
| Shared Database | Single PostgreSQL | 2026-01-16 | Enables atomic cross-module transactions |
| Authentication | None (Phase 1) | 2026-01-16 | Phase 1 scope exclusion per PRD |
| API Gateway | None required | 2026-01-16 | Single service architecture |
| Caching | None (Phase 1) | 2026-01-16 | Simple initial implementation |

---

## 7. Pending Decisions

No pending decisions at this time.

---

## Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-16 | Archi | Initial preferences for arcade service |
