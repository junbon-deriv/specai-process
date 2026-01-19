# Stories Preferences: Deriv Arcade

## Document Information
| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2026-01-16 |
| Last Updated | 2026-01-16 |
| Status | Draft |

---

## 1. User Type Definitions

### PD-UT-001: Trader as Primary User
| Attribute | Value |
|-----------|-------|
| **Decision** | Trader is the primary and most important user type |
| **Rationale** | The PRD explicitly identifies Trader as the primary user with full platform access. All core functionality revolves around trader interactions. |
| **Impact** | 16 out of 20 stories are Trader stories |
| **Alternative Considered** | None - PRD is clear on this |

### PD-UT-002: Broker as Integration Partner
| Attribute | Value |
|-----------|-------|
| **Decision** | Broker is a secondary user type for system integration only |
| **Rationale** | Broker's role is limited to account creation with external_id. No direct trading UI access. |
| **Impact** | Only 2 stories for Broker (US-AC-X2L, US-AC-E8P) |
| **Alternative Considered** | Could expand Broker capabilities in future phases |

### PD-UT-003: No Admin User Type
| Attribute | Value |
|-----------|-------|
| **Decision** | No administrative user type defined for Phase 1 |
| **Rationale** | PRD Section 6.3 explicitly excludes authentication/authorization. System is self-sufficient with no administrative functions. |
| **Impact** | No admin stories created |
| **Alternative Considered** | Could add admin for future monitoring/reporting needs |

---

## 2. Platform Hint Strategy

### PD-PH-001: Two-Service Model
| Attribute | Value |
|-----------|-------|
| **Decision** | Stories are grouped into two services: `accounts` and `trading` |
| **Rationale** | Direct alignment with the Domain Model bounded contexts (DOM-AC-K3M and DOM-TR-L8K) |
| **Impact** | Clear service boundaries, 9 accounts stories and 11 trading stories |
| **Alternative Considered** | Single monolith - rejected for maintainability |

### PD-PH-002: Accounts Service Scope
| Attribute | Value |
|-----------|-------|
| **Decision** | Accounts service handles all Account and Transaction entity operations |
| **Rationale** | Maps to Accounts Domain (DOM-AC-K3M) which owns Account and Transaction entities |
| **Impact** | Account creation, deposits, withdrawals, balance queries all in accounts service |
| **Stories** | US-AC-K3M, US-AC-M9J, US-AC-R3P, US-AC-K6L, US-AC-D4Q, US-AC-G9M, US-AC-F9L, US-AC-X2L, US-AC-E8P |

### PD-PH-003: Trading Service Scope
| Attribute | Value |
|-----------|-------|
| **Decision** | Trading service handles all PriceSeries and Contract entity operations |
| **Rationale** | Maps to Trading Domain (DOM-TR-L8K) which owns PriceSeries and Contract entities |
| **Impact** | Price preview, trade execution, trading history all in trading service |
| **Stories** | US-TR-V4N, US-TR-M5L, US-TR-W8P, US-TR-F7K, US-TR-J8N, US-TR-P7R, US-TR-Y5Q, US-TR-H3K, US-TR-Q4N, US-TR-B6N, US-TR-H2M |

### PD-PH-004: Cross-Service Balance Validation
| Attribute | Value |
|-----------|-------|
| **Decision** | Trading service validates balance via Accounts service before trade execution |
| **Rationale** | Maintains domain ownership while enabling atomic trade settlement |
| **Impact** | US-TR-B6N requires cross-service call |
| **Alternative Considered** | Duplicating balance in Trading - rejected for consistency |

---

## 3. Coverage Strategy

### PD-CV-001: Error Handling Stories Included
| Attribute | Value |
|-----------|-------|
| **Decision** | Explicit error handling stories created for all validation scenarios |
| **Rationale** | PRD Section 9.2 defines specific error codes that users experience |
| **Impact** | 8 error-related stories: US-AC-G9M, US-AC-F9L, US-AC-R6K, US-AC-N4F, US-TR-B6N, US-TR-H2M, US-TR-Q4N, US-TR-S9K |
| **Error Codes Covered** | INSUFFICIENT_BALANCE, INVALID_AMOUNT, INVALID_CURRENCY, ACCOUNT_NOT_FOUND, INVALID_STAKE, INVALID_SERIES_TYPE, INVALID_QUOTE, INVALID_SENTIMENT |

### PD-CV-002: Idempotency Stories for Financial Operations
| Attribute | Value |
|-----------|-------|
| **Decision** | Idempotency is captured as explicit user story for deposits/withdrawals |
| **Rationale** | PRD mandates 100% idempotency success rate; this affects user experience on retry |
| **Impact** | US-AC-D4Q captures idempotent deposit behavior, US-AC-W5N captures idempotent withdrawal behavior |
| **Note** | Both deposit and withdrawal idempotency have explicit stories |

### PD-CV-003: UI Animation Story Included
| Attribute | Value |
|-----------|-------|
| **Decision** | Candle animation is captured as user story US-TR-J8N |
| **Rationale** | PRD Section 5.2 describes game flow with animated candles 11-20 |
| **Impact** | Frontend must implement candle-by-candle animation |
| **Alternative Considered** | Could omit UI-specific story - included for completeness |

### PD-CV-004: Full 20-Candle Series Storage
| Attribute | Value |
|-----------|-------|
| **Decision** | Trading history stories include full candle data |
| **Rationale** | Domain Model BR-TR-J4H requires full 20-candle series storage |
| **Impact** | US-TR-Y5Q returns complete OHLC data for each contract |

---

## 4. Story Priorities

### PD-SP-001: P0 Critical Stories
| Attribute | Value |
|-----------|-------|
| **Decision** | Stories aligned with P0 PRD features are implicitly critical |
| **Stories** | US-AC-K3M, US-AC-M9J, US-AC-R3P, US-AC-K6L, US-AC-D4Q, US-AC-W5N, US-TR-V4N, US-TR-W8P, US-TR-F7K |
| **Rationale** | Direct mapping to PRD P0 features (Account Management, Trading Operations) |

### PD-SP-002: P1 Important Stories
| Attribute | Value |
|-----------|-------|
| **Decision** | Trading history stories are P1 Important |
| **Stories** | US-TR-Y5Q, US-TR-H3K |
| **Rationale** | PRD FEA-TR-Y5Q is marked P1 - Important |

### PD-SP-003: Error Stories Priority
| Attribute | Value |
|-----------|-------|
| **Decision** | Error handling stories are implicit P0 as they gate core functionality |
| **Stories** | US-AC-G9M, US-AC-F9L, US-AC-R6K, US-AC-N4F, US-TR-B6N, US-TR-H2M, US-TR-Q4N, US-TR-S9K |
| **Rationale** | Without proper error handling, core features are incomplete |

---

## 5. Excluded Scope

### PD-EX-001: No Authentication Stories
| Attribute | Value |
|-----------|-------|
| **Decision** | No authentication/authorization stories for Phase 1 |
| **Rationale** | PRD Section 6.3 explicitly excludes authentication |
| **Future Consideration** | May add login/session stories in Phase 2 |

### PD-EX-002: No Rate Limiting Stories
| Attribute | Value |
|-----------|-------|
| **Decision** | No rate limiting or abuse prevention stories |
| **Rationale** | PRD Section 6.3 explicitly excludes rate limiting |
| **Future Consideration** | May add throttling stories in Phase 2 |

### PD-EX-003: No Multi-Currency Conversion Stories
| Attribute | Value |
|-----------|-------|
| **Decision** | No currency conversion stories |
| **Rationale** | PRD Section 6.3 excludes multi-currency conversion |
| **Note** | Each account operates in single currency |

---

## 6. Domain Terminology Alignment

| Story Term | Domain Model Entity | Entity ID |
|------------|---------------------|-----------|
| Account | Account | ENT-AC-K3M |
| Deposit/Withdrawal | Transaction | ENT-AC-M9J |
| Price Preview | PriceSeries | ENT-TR-P7R |
| Rise/Fall Contract | Contract | ENT-TR-X2L |
| Balance | Account.balance | - |
| Series Type | Contract.series_type / PriceSeries.series_type | - |
| Stake | Contract.stake | - |
| Payout | Contract.payout | - |
| OHLC Candles | Contract.ohlcs / PriceSeries.candles | - |

---

## Appendix: Decision Log

| Date | Decision ID | Description |
|------|-------------|-------------|
| 2026-01-16 | PD-UT-001 | Trader defined as primary user |
| 2026-01-16 | PD-UT-002 | Broker defined as integration partner |
| 2026-01-16 | PD-UT-003 | No admin user for Phase 1 |
| 2026-01-16 | PD-PH-001 | Two-service model adopted |
| 2026-01-16 | PD-PH-002 | Accounts service scope defined |
| 2026-01-16 | PD-PH-003 | Trading service scope defined |
| 2026-01-16 | PD-PH-004 | Cross-service balance validation pattern |
| 2026-01-16 | PD-CV-001 | Error handling stories included |
| 2026-01-16 | PD-CV-002 | Idempotency stories for financial ops |
| 2026-01-16 | PD-CV-003 | UI animation story included |
| 2026-01-16 | PD-CV-004 | Full candle series in history |
| 2026-01-16 | PD-SP-001 | P0 critical stories identified |
| 2026-01-16 | PD-SP-002 | P1 important stories identified |
| 2026-01-16 | PD-SP-003 | Error stories priority set |
| 2026-01-16 | PD-EX-001 | No auth stories |
| 2026-01-16 | PD-EX-002 | No rate limiting stories |
| 2026-01-16 | PD-EX-003 | No multi-currency stories |
