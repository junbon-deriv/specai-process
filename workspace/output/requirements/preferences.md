# Requirements Phase Preferences

## Overview
This document captures all decisions, directives, and clarifications made during the requirements analysis phase for the Deriv Arcade project.

---

## Complexity Assessment

### Assessment Date: 2026-01-16

| Dimension | Score | Justification |
|-----------|-------|---------------|
| Technical Complexity | 2/3 | Golang REST API + PostgreSQL + React, GBM price generation with 4 series types, OHLC chart rendering, real-time game mechanics |
| User Complexity | 1/2 | Multi-user platform with account-based operations, authentication deferred to phase 2 |
| Integration Complexity | 0/2 | Self-sufficient, no external dependencies as per strict rules |
| Regulatory/Compliance | 1/2 | Financial transactions with idempotency requirements, trading records storage, audit trail |
| Scale/Performance | 0/1 | Standard requirements, 50-game history limit per account |

**Total Score: 4/10**

### Template Selection
- **Selected Template**: Standard PRD Template
- **Rationale**: Score of 4 falls within the 4-7 range for standard business applications
- **User Override**: None

---

## Workspace Preferences (from workspace/input/preferences.md)

### Entry WP-1: Backend Technology
**Type**: Directive
**User Input**: Backend is a Golang service exposing REST API with JSON request/response
**Context**: Initial workspace configuration
**Impact**: Defines backend architecture and API design approach

### Entry WP-2: Database Technology
**Type**: Directive
**User Input**: PostgreSQL as database
**Context**: Initial workspace configuration
**Impact**: Defines data persistence layer and query capabilities

### Entry WP-3: Frontend Technology
**Type**: Directive
**User Input**: Frontend to use React JS
**Context**: Initial workspace configuration
**Impact**: Defines frontend framework for the single-page application

---

## Phase Decisions

### Entry PD-1: Payout Structure
**Type**: From Product Brief
**Decision**: Payout = Stake / Probability, where probability includes 3% commission
**Context**: Base probability is 50% for rise/fall outcomes
**Impact**: Payout multiplier = stake / 0.53 ≈ 1.887x on win

### Entry PD-2: Series Types
**Type**: From Product Brief
**Decision**: Four series types using Geometric Brownian Motion:
- Vol50: initial=10000, volatility=50%, precision=0.001
- Vol100: initial=50000, volatility=100%, precision=0.001
- Vol200: initial=100000, volatility=200%, precision=0.001
- Vol300: initial=200000, volatility=300%, precision=0.001
**Context**: Each series has interval of 1 second, zero interest rates and quanto drift
**Impact**: Defines price generation algorithm parameters

### Entry PD-3: Stake Validation
**Type**: Assumption (User Approved)
**Decision**: Stake must be positive amount, validated against account balance
**Context**: No explicit min/max stake limits specified
**Impact**: SwipeBuy returns error if stake > balance

### Entry PD-4: Quote Validation
**Type**: Assumption (User Approved)
**Decision**: SwipeBuy's previous_quote should match 10th candle close value
**Context**: Ensures game continuity and prevents manipulation
**Impact**: Server validates quote consistency

### Entry PD-5: Game Flow
**Type**: From Product Brief
**Decision**: Next game continues from 20th candle of previous game
**Context**: Creates continuous trading experience
**Impact**: Frontend maintains state, backend validates continuity

### Entry PD-6: Trading History
**Type**: From Product Brief
**Decision**: Last 50 games displayed per account
**Context**: Account-specific history, not global
**Impact**: SwipeList returns max 50 recent contracts

### Entry PD-7: Monetary Precision
**Type**: From Product Brief
**Decision**: 2 decimal places for all monetary amounts (stake, balance, payout)
**Context**: String representation with 2-decimal point format
**Impact**: Consistent financial calculations across system

### Entry PD-8: Account ID Format
**Type**: From Product Brief
**Decision**: Integer increment with 'SW' prefix (e.g., SW1, SW2)
**Context**: External-facing account identifier
**Impact**: Database uses integer internally, prefix added for display

### Entry PD-9: Insufficient Balance Handling
**Type**: Assumption (User Approved)
**Decision**: Return error if stake exceeds available balance
**Context**: Standard financial validation
**Impact**: SwipeBuy validates balance before processing

### Entry PD-10: Authentication Scope
**Type**: From Product Brief
**Decision**: Authentication deferred to phase 2
**Context**: Multi-user platform but auth not in initial scope
**Impact**: APIs accessible without authentication in phase 1

---

## Business Rules Summary

| Rule ID | Rule Description |
|---------|-----------------|
| BR-TR-K3M | Rise contract wins if 20th candle close > 10th candle close |
| BR-TR-P7R | Fall contract wins if 20th candle close < 10th candle close |
| BR-TR-X2N | Tie (equal close values) results in zero payout |
| BR-AC-Q8L | Stake must be positive and ≤ account balance |
| BR-AC-M5K | Deposit/Withdraw require idempotency ID for deduplication |
| BR-TR-J4H | Full 20-candle series must be stored for each contract |

---

## Technical Constraints

| Constraint | Description |
|------------|-------------|
| TC-1 | No personal user information stored (no email, name, address) |
| TC-2 | Self-sufficient system with no external service dependencies |
| TC-3 | Random series generated on-the-fly per SwipeGet request |
| TC-4 | Standard REST API error handling |

---

## Out of Scope (Phase 1)

1. User authentication/authorization
2. Deployment configuration
3. Rate limiting
4. Advanced analytics
5. Multi-currency conversion
