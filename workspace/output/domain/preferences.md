# Domain Modeling Preferences
# Digital Call/Put Options Pricing Service

**Version**: 1.0  
**Date**: 2026-01-08  
**Last Updated**: 2026-01-08

---

## Step Information

| Property | Value |
|----------|-------|
| Step Name | Domain Modeling |
| Input Document | workspace/output/requirements/prd.md |
| Output Document | workspace/output/domain/domain_model.md |
| Mode | New (initial creation) |

---

## User Decisions

### Decision Categories

#### Entity Boundaries

| Decision ID | Question | User Response | Impact |
|-------------|----------|---------------|--------|
| DEC-EB-001 | Should Contract be modeled as single entity or split into ProposedContract/ActiveContract? | Single Contract entity with states - simpler and reflects business concept | Contract entity represents both Ask (proposed) and Bid (active) contexts |
| DEC-EB-002 | Should Barrier be modeled with separate resolved/unresolved states? | Single value object - resolution is computation, not state | Barrier is a value object with resolution logic, not separate entities |

#### Domain Groupings

| Decision ID | Domain | Entities | Rationale |
|-------------|--------|----------|-----------|
| DEC-DG-001 | Pricing Domain (DOM-PR-H8L) | Price (VO) | Core Black-Scholes calculations and price generation |
| DEC-DG-002 | Contract Domain (DOM-CT-I9M) | Contract, Barrier, Duration | Contract representation and parameter resolution |
| DEC-DG-003 | Market Domain (DOM-MK-J1N) | Tick | Market data handling and tick determination |
| DEC-DG-004 | Configuration Domain (DOM-CF-K2O) | Symbol, Limits | Static configuration from YAML |

#### Data Ownership Strategy

| Decision ID | Decision | Value | Rationale |
|-------------|----------|-------|-----------|
| DEC-DO-001 | Ownership Approach | Transient | Service is stateless; no persistent data ownership |
| DEC-DO-002 | Configuration Ownership | Static/Service-Owned | Symbol config loaded at startup from YAML |
| DEC-DO-003 | Market Data Ownership | External | Ticks owned by service-feed; consumed only |
| DEC-DO-004 | Contract State Ownership | Request-Scoped | Contract data reconstructed from request parameters |

#### Consistency Patterns

| Decision ID | Area | Pattern | Rationale |
|-------------|------|---------|-----------|
| DEC-CP-001 | Single Request | Strong Consistency | Each calculation must be atomic and consistent |
| DEC-CP-002 | Payout Immutability | Strict Enforcement | Payout from purchase MUST be used as-is per BR-LC-M9J |
| DEC-CP-003 | Stream Updates | Eventual Consistency | Each update is consistent snapshot; < 500ms latency acceptable |
| DEC-CP-004 | Tick Counter | Stream-Scoped | Tick counts maintained per-stream only (tick-based contracts) |

---

## Domain-Specific Preferences

### Domain Boundaries

| Preference | Value | Notes |
|------------|-------|-------|
| Boundary Definition Criteria | Business capability alignment | Domains separated by distinct business capabilities |
| Cross-Domain Communication | Direct method calls | Stateless service; no async messaging needed |
| Shared Concepts | Tick data, Symbol config | Passed by reference within request context |

### Data Ownership Strategy

| Preference | Value | Notes |
|------------|-------|-------|
| Ownership Assignment Rule | Functional responsibility | Domain owns entities it has primary responsibility for |
| Shared Data Handling | Pass-by-reference | No data duplication; shared access to immutable config |
| External Data Strategy | Read-only consumption | Market data (Ticks) consumed, not modified |

### Consistency Patterns

| Preference | Value | Notes |
|------------|-------|-------|
| Strong Consistency Areas | Pricing calculations, Barrier resolution | Must be atomic and deterministic |
| Eventual Consistency Areas | Stream updates | Acceptable lag per performance requirements |
| Transaction Boundaries | Per-request | No cross-request transactions (stateless) |

---

## Architecture Alignment

### Workspace Preferences Alignment

| Workspace Preference | Domain Model Alignment |
|---------------------|------------------------|
| Golang service with gRPC endpoint | Domain model is language-agnostic; supports gRPC patterns |
| Service is stateless, no database | Data ownership is transient; no persistent entities |
| Dependencies toward core (pricing) | Pricing Domain is central; other domains depend on it |
| Interfaces where consumed | Domain boundaries support this pattern |

### PRD Alignment

| PRD Section | Domain Model Coverage |
|-------------|----------------------|
| Contract Types (CALL/PUT) | ENT-CT-K3M: Contract with contract_type attribute |
| Barrier Logic | VO-CT-P7R: Barrier with resolution rules |
| Duration Types | VO-CT-M9J: Duration with time/tick types |
| Pricing Logic | DOM-PR-H8L: Pricing Domain |
| Contract Lifecycle | ENT-CT-K3M: lifecycle states |
| Stream Behavior | Consistency patterns per duration type |

---

## Quality Checklist

- [x] All major business concepts from PRD are represented
- [x] Entities use consistent business terminology
- [x] Relationships accurately reflect business rules
- [x] Domain boundaries align with business capabilities
- [x] Data ownership is clearly established
- [x] Consistency requirements are documented
- [x] No implementation details leak into conceptual model
- [x] Model supports all PRD requirements
- [x] Glossary includes all domain-specific terms

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-01-08 | Initial preferences document created |
