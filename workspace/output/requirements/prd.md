# Product Requirements Document: Deriv Arcade

## Document Information
| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2026-01-16 |
| Last Updated | 2026-01-16 |
| Status | Draft |
| Template | Standard PRD |
| Complexity Score | 4/10 |

---

## 1. Executive Summary

### 1.1 Product Vision
Deriv Arcade is an online arcade-like platform that offers a simple express rise/fall binary option trading experience based on randomly generated price series. The platform provides an engaging, game-like trading interface where users predict whether a synthetic index will rise or fall.

### 1.2 Product Goals
1. Deliver an intuitive single-page trading game with OHLC chart visualization
2. Provide fair 50/50 odds binary options with transparent payout structure
3. Enable account management without storing personal user information
4. Generate synthetic price data using Geometric Brownian Motion

### 1.3 Success Metrics
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| API Response Time | < 200ms | Server-side monitoring |
| Contract Evaluation Accuracy | 100% | Automated testing |
| Idempotency Success Rate | 100% | Transaction deduplication logs |
| Price Series Integrity | 100% | Full series storage verification |

---

## 2. Stakeholders

### 2.1 User Roles
| Role | Description | Capabilities |
|------|-------------|--------------|
| Trader | Primary user who trades on the platform | Create account, deposit/withdraw funds, place rise/fall trades, view trading history |
| Broker (External) | Integration partner using external_id | Reference user accounts via external_id |

### 2.2 System Actors
| Actor | Description |
|-------|-------------|
| Price Generator | Generates OHLC data using GBM algorithm |
| Contract Evaluator | Determines win/loss based on candle close values |
| Account Manager | Handles balance operations with idempotency |

---

## 3. Functional Requirements

### 3.1 Account Management

#### FEA-AC-T5N: Account Creation (P0 - Critical)
**User Story**: As a trader, I want to create an account so that I can start trading on the platform.

**Requirements**:
| ID | Requirement | Acceptance Criteria |
|----|-------------|-------------------|
| REQ-AC-K3M | System shall create accounts with currency and external_id | Account created with unique SW-prefixed ID |
| REQ-AC-P8R | System shall generate sequential account IDs with 'SW' prefix | Format: SW1, SW2, SW3... |
| REQ-AC-X2L | System shall accept 3-letter uppercase currency codes | Validation: exactly 3 uppercase letters |
| REQ-AC-N7Q | System shall not store personal information | No email, name, address fields exist |

**Business Rules**:
- BR-AC-J4M: IF account creation requested THEN generate next sequential ID with SW prefix
- BR-AC-R6K: IF currency code invalid THEN return error with code "INVALID_CURRENCY"

**API Specification**:
```
POST /accounts
Request:
{
  "currency": "USD",      // 3-letter uppercase code
  "external_id": "string" // broker reference (optional)
}
Response:
{
  "account_id": "SW1"     // SW-prefixed sequential ID
}
```

#### FEA-AC-M9J: Deposit Funds (P0 - Critical)
**User Story**: As a trader, I want to deposit funds so that I can have balance to trade.

**Requirements**:
| ID | Requirement | Acceptance Criteria |
|----|-------------|-------------------|
| REQ-AC-H5N | System shall credit deposit amount to account | Balance increases by exact deposit amount |
| REQ-AC-L2Q | System shall require deposit_id for idempotency | Duplicate deposit_id returns original result |
| REQ-AC-V8M | System shall return updated balance after deposit | Response includes new balance |
| REQ-AC-Y3K | System shall use 2 decimal precision for amounts | All amounts formatted as "0.00" |

**Business Rules**:
- BR-AC-Q2M: IF deposit_id already processed THEN return original transaction result
- BR-AC-F9L: IF amount <= 0 THEN return error "INVALID_AMOUNT"
- BR-AC-W4P: IF account not found THEN return error "ACCOUNT_NOT_FOUND"

**API Specification**:
```
POST /accounts/{account_id}/deposits
Request:
{
  "amount": "100.00",     // string, 2 decimal places
  "deposit_id": "uuid"    // idempotency key
}
Response:
{
  "balance": "100.00",
  "transaction_id": "integer",
  "transaction_time": "ISO8601"
}
```

#### FEA-AC-R3P: Withdraw Funds (P0 - Critical)
**User Story**: As a trader, I want to withdraw funds so that I can access my winnings.

**Requirements**:
| ID | Requirement | Acceptance Criteria |
|----|-------------|-------------------|
| REQ-AC-B7N | System shall debit withdrawal amount from account | Balance decreases by exact amount |
| REQ-AC-D4Q | System shall require withdrawal_id for idempotency | Duplicate withdrawal_id returns original result |
| REQ-AC-G9M | System shall validate sufficient balance | Withdrawal fails if amount > balance |
| REQ-AC-Z5K | System shall return updated balance after withdrawal | Response includes new balance |

**Business Rules**:
- BR-AC-M5K: IF withdrawal_id already processed THEN return original transaction result
- BR-AC-C8L: IF amount > balance THEN return error "INSUFFICIENT_BALANCE"
- BR-AC-T2P: IF amount <= 0 THEN return error "INVALID_AMOUNT"

**API Specification**:
```
POST /accounts/{account_id}/withdrawals
Request:
{
  "amount": "50.00",        // string, 2 decimal places
  "withdrawal_id": "uuid"   // idempotency key
}
Response:
{
  "balance": "50.00",
  "transaction_id": "integer",
  "transaction_time": "ISO8601"
}
```

#### FEA-AC-K6L: Get Account (P0 - Critical)
**User Story**: As a trader, I want to view my account details so that I can see my current balance.

**Requirements**:
| ID | Requirement | Acceptance Criteria |
|----|-------------|-------------------|
| REQ-AC-E2N | System shall return account balance | Balance returned with 2 decimal precision |
| REQ-AC-S8Q | System shall return account currency | 3-letter currency code returned |

**API Specification**:
```
GET /accounts/{account_id}
Response:
{
  "account_id": "SW1",
  "balance": "150.00",
  "currency": "USD"
}
```

---

### 3.2 Trading Operations

#### FEA-TR-V4N: Get Price Preview (SwipeGet) (P0 - Critical)
**User Story**: As a trader, I want to see the first 10 candles so that I can decide whether to bet rise or fall.

**Requirements**:
| ID | Requirement | Acceptance Criteria |
|----|-------------|-------------------|
| REQ-TR-A7M | System shall generate 10 OHLC candles on request | Returns exactly 10 candles |
| REQ-TR-F3Q | System shall use GBM algorithm per series type | Volatility matches series configuration |
| REQ-TR-J9K | System shall return OHLC data for each candle | Each candle has open, high, low, close |
| REQ-TR-N5L | System shall generate candles with 1-second interval | Timestamp increments by 1 second |

**Business Rules**:
- BR-TR-H2M: IF series_type invalid THEN return error "INVALID_SERIES_TYPE"
- BR-TR-L8K: IF SwipeGet requested THEN generate fresh random series from initial value

**API Specification**:
```
GET /swipe?series_type=Vol100
Response:
{
  "ohlcs": [
    {
      "timestamp": "ISO8601",
      "open": "50000.123",
      "high": "50010.456",
      "low": "49990.789",
      "close": "50005.234"
    }
    // ... 10 candles total
  ]
}
```

#### FEA-TR-W8P: Place Trade (SwipeBuy) (P0 - Critical)
**User Story**: As a trader, I want to buy a rise or fall contract so that I can profit from my prediction.

**Requirements**:
| ID | Requirement | Acceptance Criteria |
|----|-------------|-------------------|
| REQ-TR-B6N | System shall deduct stake from balance | Balance reduced by stake amount |
| REQ-TR-E2Q | System shall generate next 10 candles | Returns candles 11-20 from previous_quote |
| REQ-TR-I8M | System shall evaluate contract immediately | Win/loss determined on response |
| REQ-TR-M4K | System shall calculate payout per formula | Payout = stake / 0.53 if win, else 0 |
| REQ-TR-Q1L | System shall store full 20-candle series | All candles persisted for audit |
| REQ-TR-U7P | System shall credit payout to balance | Balance increased by payout amount |

**Business Rules**:
- BR-TR-K3M: IF sentiment=rise AND candle20.close > candle10.close THEN win
- BR-TR-P7R: IF sentiment=fall AND candle20.close < candle10.close THEN win
- BR-TR-X2N: IF candle20.close == candle10.close THEN loss (zero payout)
- BR-AC-Q8L: IF stake > balance THEN return error "INSUFFICIENT_BALANCE"
- BR-TR-R5M: IF stake <= 0 THEN return error "INVALID_STAKE"
- BR-TR-J4H: Full 20-candle series MUST be stored for each contract
- BR-TR-Q4N: IF previous_quote does not match 10th candle close from corresponding SwipeGet THEN return error "INVALID_QUOTE"

**Payout Calculation**:
- Base probability: 50%
- Commission: 3%
- Adjusted probability: 53%
- Payout multiplier: 1 / 0.53 ≈ 1.8868
- Win payout = stake × 1.8868 (rounded to 2 decimals)
- Loss payout = 0.00

**API Specification**:
```
POST /swipe/buy
Request:
{
  "account_id": "SW1",
  "stake": "10.00",
  "series_type": "Vol100",
  "previous_quote": "50005.234",   // 10th candle close
  "sentiment": "rise"              // or "fall"
}
Response:
{
  "contract_id": 12345,
  "purchase_time": "ISO8601",
  "ohlcs": [
    // candles 11-20
  ],
  "payout": "18.87"    // or "0.00" if loss
}
```

#### FEA-TR-Y5Q: List Contracts (SwipeList) (P1 - Important)
**User Story**: As a trader, I want to see my trading history so that I can review my past trades.

**Requirements**:
| ID | Requirement | Acceptance Criteria |
|----|-------------|-------------------|
| REQ-TR-C9N | System shall return last 50 contracts | Maximum 50 contracts returned |
| REQ-TR-G5Q | System shall filter by series_type if provided | Only matching series returned |
| REQ-TR-K1M | System shall return full 20-candle series | All candle data included |
| REQ-TR-O7K | System shall order by purchase time descending | Most recent first |

**API Specification**:
```
GET /swipe/list?account_id=SW1&series_type=Vol100
Response:
{
  "contracts": [
    {
      "contract_id": 12345,
      "purchase_time": "ISO8601",
      "series_type": "Vol100",
      "sentiment": "rise",
      "stake": "10.00",
      "payout": "18.87",
      "ohlcs": [
        // all 20 candles
      ]
    }
  ]
}
```

---

### 3.3 Price Generation

#### FEA-PG-Z3L: Geometric Brownian Motion Generator (P0 - Critical)
**User Story**: As a system, I need to generate realistic price movements for the trading game.

**Requirements**:
| ID | Requirement | Acceptance Criteria |
|----|-------------|-------------------|
| REQ-PG-D8N | System shall implement GBM algorithm | Price follows dS = μSdt + σSdW |
| REQ-PG-H4Q | System shall support 4 volatility configurations | Vol50, Vol100, Vol200, Vol300 |
| REQ-PG-L0M | System shall maintain 50% rise/fall probability | Statistical distribution verified |
| REQ-PG-P6K | System shall use configured precision | 0.001 for all series types |

**Series Configuration**:
| Series | Initial Value | Volatility | Interest Rate | Quanto Drift | Interval | Precision |
|--------|---------------|------------|---------------|--------------|----------|-----------|
| Vol50 | 10000 | 50% | 0 | 0 | 1 second | 0.001 |
| Vol100 | 50000 | 100% | 0 | 0 | 1 second | 0.001 |
| Vol200 | 100000 | 200% | 0 | 0 | 1 second | 0.001 |
| Vol300 | 200000 | 300% | 0 | 0 | 1 second | 0.001 |

**OHLC Generation Logic**:
- For each 5-second candle interval, generate 5 price ticks (1-second interval)
- Open = first tick of interval
- High = maximum tick of interval
- Low = minimum tick of interval
- Close = last tick of interval

---

## 4. Non-Functional Requirements

### 4.1 Performance
| Requirement | Target | Rationale |
|-------------|--------|-----------|
| API Response Time | < 200ms (p95) | Smooth user experience |
| Contract Evaluation | < 50ms | Instant feedback on trade |
| Price Generation | < 100ms for 10 candles | Quick preview loading |

### 4.2 Reliability
| Requirement | Target | Rationale |
|-------------|--------|-----------|
| Idempotency Success | 100% | Prevent duplicate transactions |
| Data Integrity | 100% | Accurate financial records |
| Transaction Atomicity | All or nothing | No partial state |

### 4.3 Security
| Requirement | Description |
|-------------|-------------|
| No PII Storage | System must not store personal information |
| Input Validation | All inputs validated for type and range |
| SQL Injection Prevention | Parameterized queries only |

### 4.4 Data Retention
| Data Type | Retention Policy |
|-----------|-----------------|
| Account Data | Indefinite |
| Transaction History | Indefinite |
| Contract Data | Indefinite (full series stored) |
| Trading History Display | Last 50 per account |

---

## 5. User Interface Requirements

### 5.1 Single-Page Application Layout
**Framework**: React JS

**Main Components**:
1. **OHLC Chart Area** (Focal Point)
   - Displays candlestick chart
   - Shows first 10 candles during preview
   - Animates candles 11-20 after trade placement
   
2. **Trading Action Panel**
   - Rise button (upper-right, square)
   - Fall button (lower-right, square)
   - Disappears after selection to show remaining candles

3. **Trading History Sidebar** (Right)
   - Shows last 50 games for current account
   - Displays: timestamp, sentiment, stake, payout, result

### 5.2 Game Flow Visualization
```
[Preview Phase]
┌─────────────────────┬───────┐
│                     │ RISE  │
│  Candles 1-10       ├───────┤
│                     │ FALL  │
└─────────────────────┴───────┘

[After Selection]
┌─────────────────────────────┐
│                             │
│  Candles 1-20 (animated)    │
│                             │
└─────────────────────────────┘
```

### 5.3 Chart Specifications
- Chart type: OHLC Candlestick
- Candle interval: 5 seconds display (derived from 1-second ticks)
- Game duration: 20 candles × 5 seconds = 100 seconds total
- Progression: Left to right
- Next game continues from candle 20 close value

---

## 6. Technical Constraints

### 6.1 Mandatory Constraints
| ID | Constraint | Rationale |
|----|------------|-----------|
| TC-1 | No personal user information storage | Privacy by design |
| TC-2 | Self-sufficient with no external dependencies | Standalone operation |
| TC-3 | Random series generated on-the-fly | Fresh data per request |
| TC-4 | Full series stored per contract | Audit and replay capability |

### 6.2 Technology Stack (from Workspace Preferences)
| Layer | Technology |
|-------|------------|
| Backend | Golang |
| API | REST with JSON |
| Database | PostgreSQL |
| Frontend | React JS |

### 6.3 Out of Scope (Phase 1)
- User authentication/authorization
- Deployment configuration
- Rate limiting
- Multi-currency conversion
- Advanced analytics

---

## 7. Data Model Overview

### 7.1 Core Entities
| Entity | Description | Key Fields |
|--------|-------------|------------|
| Account | Trading account | id (bigint), external_id (string), currency (char(3)), balance (decimal(18,2)) |
| Transaction | Deposit/Withdrawal record | id (bigint), account_id (bigint), type (enum), amount (decimal(18,2)), idempotency_id (uuid) |
| Contract | Trade record | id (bigint), account_id (bigint), series_type (string), sentiment (enum), stake (decimal(18,2)), payout (decimal(18,2)), ohlcs (jsonb) |

### 7.2 Relationships
- Account 1:N Transaction (deposits and withdrawals)
- Account 1:N Contract (trades placed)

---

## 8. API Summary

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/accounts` | POST | Create new account |
| `/accounts/{id}` | GET | Get account details |
| `/accounts/{id}/deposits` | POST | Deposit funds |
| `/accounts/{id}/withdrawals` | POST | Withdraw funds |
| `/swipe` | GET | Get price preview (10 candles) |
| `/swipe/buy` | POST | Place rise/fall trade |
| `/swipe/list` | GET | List trading history |

---

## 9. Error Handling

### 9.1 Standard Error Response Format
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

### 9.2 Error Codes
| Code | HTTP Status | Description |
|------|-------------|-------------|
| ACCOUNT_NOT_FOUND | 404 | Account ID does not exist |
| INVALID_CURRENCY | 400 | Currency code format invalid |
| INVALID_AMOUNT | 400 | Amount is not positive |
| INVALID_STAKE | 400 | Stake is not positive |
| INSUFFICIENT_BALANCE | 400 | Balance < requested amount |
| INVALID_SERIES_TYPE | 400 | Series type not recognized |
| INVALID_SENTIMENT | 400 | Sentiment not rise/fall |
| INVALID_QUOTE | 400 | previous_quote does not match expected 10th candle close |
| DUPLICATE_TRANSACTION | 200 | Idempotency key already used (returns original) |

---

## 10. Glossary

| Term | Definition |
|------|------------|
| OHLC | Open-High-Low-Close candlestick data |
| GBM | Geometric Brownian Motion - stochastic process for modeling prices |
| Rise Contract | Bet that close of candle 20 > close of candle 10 |
| Fall Contract | Bet that close of candle 20 < close of candle 10 |
| Payout | Amount returned to trader on winning trade |
| Stake | Amount wagered on a trade |
| Series Type | Configuration for price generation (Vol50, Vol100, Vol200, Vol300) |
| Idempotency | Ensuring duplicate requests produce same result |
| External ID | Broker-provided reference for account linking |

---

## Appendix A: Preference References

This PRD incorporates decisions documented in [workspace/output/requirements/preferences.md](preferences.md):
- PD-1: Payout Structure
- PD-2: Series Types
- PD-3: Stake Validation
- PD-4: Quote Validation
- PD-5: Game Flow
- PD-6: Trading History
- PD-7: Monetary Precision
- PD-8: Account ID Format
- PD-9: Insufficient Balance Handling
- PD-10: Authentication Scope

---

## Appendix B: Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-16 | Archi | Initial PRD creation |
| 1.1 | 2026-01-16 | Archi | Added BR-TR-Q4N for quote validation, explicit data types in data model, INVALID_QUOTE error code |
