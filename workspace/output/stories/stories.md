# User Stories: Deriv Arcade

## Document Information
| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2026-01-16 |
| Last Updated | 2026-01-16 |
| Status | Draft |
| PRD Reference | [workspace/output/requirements/prd.md](../requirements/prd.md) |
| Domain Model Reference | [workspace/output/domain/domain_model.md](../domain/domain_model.md) |

---

## 1. Executive Summary

Deriv Arcade is an arcade-style binary options trading platform where traders predict whether synthetic price indices will rise or fall. The system serves two primary user types:

1. **Trader** - The primary end user who creates accounts, manages funds, and places binary option trades
2. **Broker** - External integration partner who links accounts via external identifiers

The user interaction model is straightforward: traders manage their accounts through deposits and withdrawals, view price previews to make trading decisions, and place rise/fall contracts. The platform operates without authentication in Phase 1, focusing on core trading mechanics.

This document contains **25 user stories** distributed across **2 service domains** (accounts, trading), covering all PRD requirements and edge cases.

---

## 2. User Types

### 2.1 Trader

| Attribute | Description |
|-----------|-------------|
| **User Type Name** | Trader |
| **Description** | Primary end user who engages with the arcade-style trading platform. Creates accounts, manages funds through deposits and withdrawals, previews synthetic price series, and places rise/fall binary option contracts. |
| **Key Characteristics** | Direct platform user, interacts via React JS frontend, manages own balance, makes trading decisions based on 10-candle price previews, watches animated trade outcomes |
| **Access Level** | Full access to own account operations: create account, deposit/withdraw funds, view balance, preview prices, place trades, view trading history |

**Domain Entities Accessed**:
- [Account](../domain/domain_model.md) (ENT-AC-K3M) - Create, Read, Update balance
- [Transaction](../domain/domain_model.md) (ENT-AC-M9J) - Create via deposits/withdrawals
- [PriceSeries](../domain/domain_model.md) (ENT-TR-P7R) - Request price previews
- [Contract](../domain/domain_model.md) (ENT-TR-X2L) - Create trades, view history

### 2.2 Broker

| Attribute | Description |
|-----------|-------------|
| **User Type Name** | Broker |
| **Description** | External system integration partner who creates and references user accounts using external_id for correlation with their own systems. Does not directly interact with the trading UI. |
| **Key Characteristics** | System-to-system integration, uses external_id for account linking, programmatic API access |
| **Access Level** | Create accounts with external_id, reference accounts by external_id |

**Domain Entities Accessed**:
- [Account](../domain/domain_model.md) (ENT-AC-K3M) - Create with external_id

---

## 3. User Stories by Type

### 3.1 Trader Stories

#### Account Management

| ID | User Story | Service Hint |
|----|------------|--------------|
| US-AC-K3M | As a trader, I want to create an account with my preferred currency so that I can start trading on the platform | accounts |
| US-AC-M9J | As a trader, I want to deposit funds into my account so that I have balance available for trading | accounts |
| US-AC-R3P | As a trader, I want to withdraw funds from my account so that I can access my winnings | accounts |
| US-AC-K6L | As a trader, I want to view my account details so that I can see my current balance and currency | accounts |
| US-AC-D4Q | As a trader, I want duplicate deposit requests to return the same result so that network retries don't double-credit my account | accounts |
| US-AC-W5N | As a trader, I want duplicate withdrawal requests to return the same result so that network retries don't double-debit my account | accounts |
| US-AC-G9M | As a trader, I want to be notified when a withdrawal exceeds my balance so that I understand why the operation failed | accounts |
| US-AC-F9L | As a trader, I want to be notified when I enter an invalid amount so that I can correct my input | accounts |
| US-AC-R6K | As a trader, I want to be notified when I use an invalid currency code so that I can create my account with a valid currency | accounts |
| US-AC-N4F | As a trader, I want to receive a clear error when accessing a non-existent account so that I know the account ID is incorrect | accounts |

#### Trading Operations

| ID | User Story | Service Hint |
|----|------------|--------------|
| US-TR-V4N | As a trader, I want to see a price preview of 10 OHLC candles so that I can analyze the trend before placing a trade | trading |
| US-TR-M5L | As a trader, I want to select a volatility series type (Vol50, Vol100, Vol200, Vol300) so that I can trade on different synthetic indices | trading |
| US-TR-W8P | As a trader, I want to buy a rise contract so that I can profit when the 20th candle closes higher than the 10th | trading |
| US-TR-F7K | As a trader, I want to buy a fall contract so that I can profit when the 20th candle closes lower than the 10th | trading |
| US-TR-J8N | As a trader, I want to see candles 11-20 animate after placing a trade so that I can watch the outcome unfold | trading |
| US-TR-P7R | As a trader, I want to see my payout immediately after the trade resolves so that I know my winnings or losses | trading |
| US-TR-Y5Q | As a trader, I want to view my last 50 trades so that I can review my trading history | trading |
| US-TR-H3K | As a trader, I want to filter my trading history by series type so that I can analyze my performance per index | trading |
| US-TR-Q4N | As a trader, I want quote validation to prevent stale trades so that I always trade on the series I previewed | trading |

#### Error Handling

| ID | User Story | Service Hint |
|----|------------|--------------|
| US-TR-B6N | As a trader, I want to be notified when my stake exceeds my balance so that I can adjust my trade size | trading |
| US-TR-R5M | As a trader, I want to be notified when my stake is zero or negative so that I can enter a valid trade amount | trading |
| US-TR-H2M | As a trader, I want to be notified when I select an invalid series type so that I can choose a valid option | trading |
| US-TR-S9K | As a trader, I want to be notified when I use an invalid sentiment so that I can choose rise or fall correctly | trading |

### 3.2 Broker Stories

| ID | User Story | Service Hint |
|----|------------|--------------|
| US-AC-X2L | As a broker, I want to create accounts with an external_id so that I can link trader accounts to my external system | accounts |
| US-AC-E8P | As a broker, I want the external_id to be stored but not validated so that I have flexibility in my reference format | accounts |

---

## 4. Cross-User Type Stories

No direct cross-user interactions are defined for Phase 1. The Broker creates accounts that Traders subsequently use, but this is an indirect relationship with no runtime collaboration.

---

## 5. Story Coverage Matrix

| PRD Feature | Feature Description | Related Story IDs | User Types |
|-------------|---------------------|-------------------|------------|
| FEA-AC-T5N | Account Creation | US-AC-K3M, US-AC-X2L | Trader, Broker |
| FEA-AC-M9J | Deposit Funds | US-AC-M9J, US-AC-D4Q, US-AC-F9L | Trader |
| FEA-AC-R3P | Withdraw Funds | US-AC-R3P, US-AC-W5N, US-AC-G9M, US-AC-F9L | Trader |
| FEA-AC-K6L | Get Account | US-AC-K6L, US-AC-N4F | Trader |
| FEA-TR-V4N | Price Preview (SwipeGet) | US-TR-V4N, US-TR-M5L, US-TR-H2M | Trader |
| FEA-TR-W8P | Place Trade (SwipeBuy) | US-TR-W8P, US-TR-F7K, US-TR-J8N, US-TR-P7R, US-TR-Q4N, US-TR-B6N, US-TR-R5M, US-TR-S9K | Trader |
| FEA-TR-Y5Q | List Contracts (SwipeList) | US-TR-Y5Q, US-TR-H3K | Trader |
| FEA-PG-Z3L | GBM Price Generation | US-TR-V4N (implied) | Trader |

---

## 6. Service Distribution Summary

### Service: accounts
| Metric | Value |
|--------|-------|
| Story Count | 12 |
| Primary User Type | Trader |
| Secondary User Type | Broker |
| Domain | DOM-AC-K3M (Accounts Domain) |

**Stories**: US-AC-K3M, US-AC-M9J, US-AC-R3P, US-AC-K6L, US-AC-D4Q, US-AC-W5N, US-AC-G9M, US-AC-F9L, US-AC-R6K, US-AC-N4F, US-AC-X2L, US-AC-E8P

**Capabilities**:
- Account creation with SW-prefixed sequential IDs
- Balance management (deposits, withdrawals)
- Idempotent financial operations
- Balance validation and error reporting

### Service: trading
| Metric | Value |
|--------|-------|
| Story Count | 13 |
| Primary User Type | Trader |
| Secondary User Type | None |
| Domain | DOM-TR-L8K (Trading Domain) |

**Stories**: US-TR-V4N, US-TR-M5L, US-TR-W8P, US-TR-F7K, US-TR-J8N, US-TR-P7R, US-TR-Y5Q, US-TR-H3K, US-TR-Q4N, US-TR-B6N, US-TR-R5M, US-TR-H2M, US-TR-S9K

**Capabilities**:
- Price series generation with GBM algorithm
- Rise/fall contract execution
- Trade outcome evaluation and payout calculation
- Trading history display with filtering

### Cross-Service Patterns

| Pattern | Stories Involved | Description |
|---------|------------------|-------------|
| Balance Check | US-TR-W8P, US-TR-F7K, US-TR-B6N | Trading domain validates balance via Accounts domain before trade execution |
| Atomic Trade Settlement | US-TR-W8P, US-TR-F7K, US-TR-P7R | Trade execution creates STAKE and PAYOUT transactions atomically |

---

## 7. Domain Entity Alignment

| Story ID | Primary Entity | Related Entities | Operation |
|----------|----------------|------------------|-----------|
| US-AC-K3M | Account | - | Create |
| US-AC-M9J | Transaction | Account | Create (DEPOSIT) |
| US-AC-R3P | Transaction | Account | Create (WITHDRAWAL) |
| US-AC-K6L | Account | - | Read |
| US-AC-D4Q | Transaction | - | Idempotent Create |
| US-AC-W5N | Transaction | - | Idempotent Create |
| US-AC-G9M | Account | - | Validate |
| US-AC-R6K | Account | - | Validate |
| US-AC-N4F | Account | - | Validate |
| US-AC-F9L | Transaction | - | Validate |
| US-AC-X2L | Account | - | Create |
| US-AC-E8P | Account | - | Create |
| US-TR-V4N | PriceSeries | - | Create |
| US-TR-M5L | PriceSeries | - | Create |
| US-TR-W8P | Contract | Transaction, Account, PriceSeries | Create |
| US-TR-F7K | Contract | Transaction, Account, PriceSeries | Create |
| US-TR-J8N | Contract | - | Read (UI) |
| US-TR-P7R | Contract | Transaction | Read |
| US-TR-Y5Q | Contract | - | Read (List) |
| US-TR-H3K | Contract | - | Read (Filter) |
| US-TR-Q4N | PriceSeries | Contract | Validate |
| US-TR-B6N | Account | - | Validate |
| US-TR-R5M | Contract | - | Validate |
| US-TR-H2M | PriceSeries | - | Validate |
| US-TR-S9K | Contract | - | Validate |

---

## 8. User Journey Summary

### 8.1 Trader Journey

```
[Onboarding]
  └─ US-AC-K3M: Create account
       └─ US-AC-M9J: Deposit funds

[Core Trading Loop]
  └─ US-TR-V4N: View price preview (10 candles)
       └─ US-TR-M5L: Select series type
            └─ US-TR-W8P / US-TR-F7K: Place rise/fall trade
                 └─ US-TR-J8N: Watch candles 11-20 animate
                      └─ US-TR-P7R: See payout result
                           └─ [Repeat trading loop]

[Account Management]
  └─ US-AC-K6L: Check balance
  └─ US-AC-R3P: Withdraw funds

[History Review]
  └─ US-TR-Y5Q: View trading history
  └─ US-TR-H3K: Filter by series type
```

### 8.2 Broker Journey

```
[Integration Setup]
  └─ US-AC-X2L: Create account with external_id
       └─ US-AC-E8P: Flexible external_id format
            └─ [Account available for Trader use]
```

---

## Appendix A: Story ID Reference

| ID | Short Description |
|----|-------------------|
| US-AC-K3M | Create account |
| US-AC-M9J | Deposit funds |
| US-AC-R3P | Withdraw funds |
| US-AC-K6L | View account |
| US-AC-D4Q | Idempotent deposits |
| US-AC-W5N | Idempotent withdrawals |
| US-AC-G9M | Insufficient balance error |
| US-AC-F9L | Invalid amount error |
| US-AC-R6K | Invalid currency error |
| US-AC-N4F | Account not found error |
| US-AC-X2L | Broker account creation |
| US-AC-E8P | Flexible external_id |
| US-TR-V4N | Price preview |
| US-TR-M5L | Select series type |
| US-TR-W8P | Buy rise contract |
| US-TR-F7K | Buy fall contract |
| US-TR-J8N | Watch candle animation |
| US-TR-P7R | See payout result |
| US-TR-Y5Q | View trading history |
| US-TR-H3K | Filter history by series |
| US-TR-Q4N | Quote validation |
| US-TR-B6N | Stake exceeds balance error |
| US-TR-R5M | Invalid stake error |
| US-TR-H2M | Invalid series type error |
| US-TR-S9K | Invalid sentiment error |

---

## Appendix B: Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-16 | Archi | Initial user stories creation |
| 1.1 | 2026-01-16 | Archi | Added US-TR-R5M for INVALID_STAKE error per verification feedback |
