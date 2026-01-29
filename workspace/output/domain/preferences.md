# Domain Model Preferences: Deriv Arcade

## Document Information
| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2026-01-16 |
| Last Updated | 2026-01-16 |
| Domain Model | [workspace/output/domain/domain_model.md](domain_model.md) |

---

## Overview

This document captures all user decisions and preferences made during the domain modeling process for Deriv Arcade. These preferences guide the domain model structure and should be consulted for future updates.

---

## Decision Categories

### 1. Entity Boundaries

#### PD-DOM-01: Series Configuration as Application Config
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-01 |
| **Question** | Should Series Configuration be stored as a database entity or application-level configuration? |
| **Decision** | Application-level configuration (not a database entity) |
| **Rationale** | Series types (Vol50, Vol100, Vol200, Vol300) are static and don't require database persistence |
| **Impact** | No Series Configuration entity in domain model; values hard-coded or config-driven |

#### PD-DOM-02: PriceSeries Entity
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-02 |
| **Question** | Should preview price series (SwipeGet 10 candles) be tracked/persisted? |
| **Decision** | Yes, persisted temporarily for quote validation |
| **Rationale** | Required to validate previous_quote in SwipeBuy matches actual generated 10th candle close |
| **Impact** | PriceSeries entity added to Trading domain with temporary lifecycle |

#### PD-DOM-03: PriceSeries Account Linking
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-03 |
| **Question** | Should PriceSeries be linked to an account or anonymous? |
| **Decision** | Linked to account |
| **Rationale** | Need to validate quote belongs to requesting account during SwipeBuy |
| **Impact** | PriceSeries has account_id foreign key; queried by account |

---

### 2. Domain Groupings

#### PD-DOM-04: Accounts and Trading Separation
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-04 |
| **Question** | Should Accounts (balance/transactions) and Trading (contracts/price series) be separate bounded contexts? |
| **Decision** | Keep as separate bounded contexts |
| **Rationale** | Clear separation of concerns; Accounts manages financial state, Trading manages game logic |
| **Impact** | Two domains defined: DOM-AC-K3M (Accounts), DOM-TR-L8K (Trading) |

#### PD-DOM-05: Domain Entities Assignment
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-05 |
| **Question** | Which entities belong to which domain? |
| **Decision** | Accounts: Account, Transaction; Trading: PriceSeries, Contract |
| **Rationale** | Group by primary responsibility and lifecycle management |
| **Impact** | Clear ownership boundaries; cross-domain operations via defined interfaces |

---

### 3. Data Ownership Strategy

#### PD-DOM-06: Transaction Types Extended
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-06 |
| **Question** | Should contract settlements (stake/payout) be recorded as transactions? |
| **Decision** | Yes, include STAKE and PAYOUT transaction types |
| **Rationale** | Complete audit trail of all balance-affecting operations |
| **Impact** | Transaction entity has 4 types: DEPOSIT, WITHDRAWAL, STAKE, PAYOUT |

#### PD-DOM-07: Zero Payout Transactions
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-07 |
| **Question** | Should PAYOUT transactions be created for losing contracts? |
| **Decision** | Yes, create PAYOUT transaction with amount 0.00 for losses |
| **Rationale** | Avoids open-ended contracts; every contract has corresponding stake and payout transactions |
| **Impact** | Contract always has exactly 2 transactions: 1 STAKE, 1 PAYOUT |

#### PD-DOM-08: PriceSeries Lifecycle
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-08 |
| **Question** | Should PriceSeries be kept after contract creation? |
| **Decision** | Delete PriceSeries after contract creation |
| **Rationale** | Full 20-candle series stored in Contract; no need to keep preview separately |
| **Impact** | PriceSeries is short-lived; cleaned up as part of trade execution |

#### PD-DOM-09: Contract Candle Storage
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-09 |
| **Question** | Should Contract reference PriceSeries or store full candles? |
| **Decision** | Store full 20 candles in Contract (no reference needed) |
| **Rationale** | Complete self-contained record; no need to join tables for history display |
| **Impact** | Contract has JSONB ohlcs field with all 20 candles |

---

### 4. Consistency Patterns

#### PD-DOM-10: Trade Execution Atomicity
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-10 |
| **Question** | Should trade execution be atomic or eventual consistency? |
| **Decision** | Full atomic transaction (all-or-nothing) |
| **Rationale** | Financial integrity; stake deduction, contract creation, and payout must be consistent |
| **Impact** | Single database transaction for SwipeBuy operation |

#### PD-DOM-11: Cross-Domain Transaction Handling
| Attribute | Value |
|-----------|-------|
| **Decision ID** | PD-DOM-11 |
| **Question** | How to handle transactions spanning Accounts and Trading domains? |
| **Decision** | Single atomic transaction despite separate bounded contexts |
| **Rationale** | Simplicity for MVP; both domains use same database |
| **Impact** | Trade execution wraps all operations in single transaction |

---

## Step-Specific Sections

### Domain Boundaries Details

| Boundary Aspect | Decision |
|----------------|----------|
| Separation Criteria | By business capability (financial management vs. trading) |
| Cross-Domain References | Via account_id foreign key only |
| Shared Concepts | None (each domain has distinct entities) |
| Integration Pattern | Synchronous within same transaction |

### Data Ownership Strategy Details

| Ownership Aspect | Decision |
|------------------|----------|
| Approach | Domain-based ownership (each entity owned by one domain) |
| Ownership Assignment | Based on primary lifecycle responsibility |
| Shared Data Handling | No shared entities; references by ID only |
| Reference Data | Series Configuration as application config |

### Consistency Patterns Details

| Consistency Aspect | Decision |
|-------------------|----------|
| Strong Consistency Required | All financial operations (deposit, withdrawal, trade) |
| Eventual Consistency Acceptable | None for MVP |
| Transaction Boundaries | Single DB transaction per user operation |
| Saga Patterns | Not required for MVP (single database) |

---

## Decision Changelog

| Date | Decision ID | Change | Reason |
|------|-------------|--------|--------|
| 2026-01-16 | PD-DOM-01 | Initial decision | User preference |
| 2026-01-16 | PD-DOM-02 | Initial decision | User preference |
| 2026-01-16 | PD-DOM-03 | Initial decision | User clarification |
| 2026-01-16 | PD-DOM-04 | Initial decision | User preference |
| 2026-01-16 | PD-DOM-05 | Initial decision | Derived from PD-DOM-04 |
| 2026-01-16 | PD-DOM-06 | Initial decision | User preference |
| 2026-01-16 | PD-DOM-07 | Initial decision | User preference |
| 2026-01-16 | PD-DOM-08 | Initial decision | User preference |
| 2026-01-16 | PD-DOM-09 | Initial decision | User preference |
| 2026-01-16 | PD-DOM-10 | Initial decision | User preference |
| 2026-01-16 | PD-DOM-11 | Initial decision | Derived from PD-DOM-10 |
