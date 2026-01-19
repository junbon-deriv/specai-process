# Public API Specification: Deriv Arcade

## Document Information
| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2026-01-16 |
| Last Updated | 2026-01-16 |
| Status | Draft |
| Service | arcade (SVC-AR-K3M) |
| API Type | Public |
| PRD Reference | [workspace/output/requirements/prd.md](../requirements/prd.md) |
| Architecture Reference | [workspace/output/architecture/architecture.md](../architecture/architecture.md) |

---

## 1. Overview

The Deriv Arcade Public API provides REST endpoints for the arcade-style binary options trading platform. This API enables:

- **Account Management**: Create trading accounts, manage funds through deposits and withdrawals
- **Trading Operations**: Preview synthetic price series, place rise/fall binary options, and view trading history

**Target Consumers**:
- React JS Single Page Application (primary)
- External broker integrations (via external_id)

**Key Characteristics**:
- RESTful design with JSON request/response
- Idempotent financial operations (deposits, withdrawals)
- Atomic trade execution with immediate settlement
- No authentication required (Phase 1)

---

## 2. Authentication & Authorization

### Phase 1 (Current)
**No authentication required**. All endpoints are publicly accessible.

| Aspect | Configuration |
|--------|---------------|
| Authentication Method | None (Phase 1) |
| Authorization | None |
| Rate Limiting | Not implemented |

**Note**: This is appropriate for Phase 1 development. Production deployment should implement proper authentication mechanisms.

---

## 3. Base Configuration

| Configuration | Value |
|---------------|-------|
| Base URL | `http://localhost:8080` (development) |
| Protocol | HTTP/1.1, REST |
| Content-Type | `application/json` |
| Accept | `application/json` |
| Character Encoding | UTF-8 |
| Versioning Strategy | URL path (future: `/v1/`) |

### Request Headers
| Header | Required | Description |
|--------|----------|-------------|
| Content-Type | Yes (POST) | Must be `application/json` |
| Accept | No | Defaults to `application/json` |

### Response Headers
| Header | Always Present | Description |
|--------|----------------|-------------|
| Content-Type | Yes | Always `application/json` |

---

## 4. Table of Endpoints

| Method | Path | Summary | Module |
|--------|------|---------|--------|
| GET | `/health` | Health check endpoint | Operations |
| POST | `/accounts` | Create new trading account | Accounts |
| GET | `/accounts/{account_id}` | Get account details and balance | Accounts |
| POST | `/accounts/{account_id}/deposits` | Deposit funds (idempotent) | Accounts |
| POST | `/accounts/{account_id}/withdrawals` | Withdraw funds (idempotent) | Accounts |
| GET | `/swipe` | Get price preview (10 candles) | Trading |
| POST | `/swipe/buy` | Place rise/fall trade | Trading |
| GET | `/swipe/list` | List trading history | Trading |

---

## 5. Endpoints & Methods

### 5.1 Accounts Module

#### API-AC-K3M: Create Account

| Attribute | Value |
|-----------|-------|
| **Endpoint ID** | API-AC-K3M |
| **Method** | POST |
| **Path** | `/accounts` |
| **Purpose** | Create a new trading account with specified currency |
| **PRD Reference** | FEA-AC-T5N |
| **User Stories** | US-AC-K3M, US-AC-X2L, US-AC-E8P |

**Request Schema**:
```json
{
  "currency": "string",      // Required: 3-letter uppercase currency code (e.g., "USD")
  "external_id": "string"    // Optional: Broker reference identifier
}
```

**Response Schema** (201 Created):
```json
{
  "account_id": "string"     // SW-prefixed sequential ID (e.g., "SW1")
}
```

**Validation Rules**:
- `currency`: Required, exactly 3 uppercase letters (A-Z)
- `external_id`: Optional, stored as-is without validation

**Example**:
```
POST /accounts
Content-Type: application/json

{
  "currency": "USD",
  "external_id": "broker-ref-12345"
}

Response: 201 Created
{
  "account_id": "SW1"
}
```

**Error Responses**:
| Error Code | HTTP Status | Condition |
|------------|-------------|-----------|
| ERR-AC-R6K | 400 | Invalid currency format |

---

#### API-AC-K6L: Get Account

| Attribute | Value |
|-----------|-------|
| **Endpoint ID** | API-AC-K6L |
| **Method** | GET |
| **Path** | `/accounts/{account_id}` |
| **Purpose** | Retrieve account details including current balance |
| **PRD Reference** | FEA-AC-K6L |
| **User Stories** | US-AC-K6L, US-AC-N4F |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | string | Yes | Account identifier (e.g., "SW1") |

**Response Schema** (200 OK):
```json
{
  "account_id": "string",    // SW-prefixed ID
  "balance": "string",       // Current balance, 2 decimal places (e.g., "150.00")
  "currency": "string"       // 3-letter currency code
}
```

**Example**:
```
GET /accounts/SW1

Response: 200 OK
{
  "account_id": "SW1",
  "balance": "150.00",
  "currency": "USD"
}
```

**Error Responses**:
| Error Code | HTTP Status | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | 404 | Account not found |

---

#### API-AC-M9J: Deposit Funds

| Attribute | Value |
|-----------|-------|
| **Endpoint ID** | API-AC-M9J |
| **Method** | POST |
| **Path** | `/accounts/{account_id}/deposits` |
| **Purpose** | Credit funds to account balance (idempotent) |
| **PRD Reference** | FEA-AC-M9J |
| **User Stories** | US-AC-M9J, US-AC-D4Q, US-AC-F9L |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | string | Yes | Account identifier (e.g., "SW1") |

**Request Schema**:
```json
{
  "amount": "string",        // Required: Positive amount, 2 decimal places (e.g., "100.00")
  "deposit_id": "string"     // Required: UUID for idempotency
}
```

**Response Schema** (200 OK):
```json
{
  "balance": "string",           // Updated balance, 2 decimal places
  "transaction_id": "integer",   // Unique transaction identifier
  "transaction_time": "string"   // ISO8601 timestamp
}
```

**Idempotency Behavior**:
- If `deposit_id` was already processed, returns the original transaction result with HTTP 200
- Response includes indicator when returning duplicate: same `transaction_id` and `transaction_time`

**Validation Rules**:
- `amount`: Required, must be positive (> 0), 2 decimal precision
- `deposit_id`: Required, must be valid UUID format

**Example**:
```
POST /accounts/SW1/deposits
Content-Type: application/json

{
  "amount": "100.00",
  "deposit_id": "550e8400-e29b-41d4-a716-446655440000"
}

Response: 200 OK
{
  "balance": "100.00",
  "transaction_id": 1,
  "transaction_time": "2026-01-16T10:30:00Z"
}
```

**Error Responses**:
| Error Code | HTTP Status | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | 404 | Account not found |
| ERR-AC-F9L | 400 | Invalid amount (zero, negative, or invalid format) |

---

#### API-AC-R3P: Withdraw Funds

| Attribute | Value |
|-----------|-------|
| **Endpoint ID** | API-AC-R3P |
| **Method** | POST |
| **Path** | `/accounts/{account_id}/withdrawals` |
| **Purpose** | Debit funds from account balance (idempotent) |
| **PRD Reference** | FEA-AC-R3P |
| **User Stories** | US-AC-R3P, US-AC-W5N, US-AC-G9M, US-AC-F9L |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | string | Yes | Account identifier (e.g., "SW1") |

**Request Schema**:
```json
{
  "amount": "string",          // Required: Positive amount, 2 decimal places
  "withdrawal_id": "string"    // Required: UUID for idempotency
}
```

**Response Schema** (200 OK):
```json
{
  "balance": "string",           // Updated balance, 2 decimal places
  "transaction_id": "integer",   // Unique transaction identifier
  "transaction_time": "string"   // ISO8601 timestamp
}
```

**Idempotency Behavior**:
- If `withdrawal_id` was already processed, returns the original transaction result with HTTP 200
- Response includes indicator when returning duplicate: same `transaction_id` and `transaction_time`

**Validation Rules**:
- `amount`: Required, must be positive (> 0), 2 decimal precision
- `amount`: Must not exceed current balance
- `withdrawal_id`: Required, must be valid UUID format

**Example**:
```
POST /accounts/SW1/withdrawals
Content-Type: application/json

{
  "amount": "50.00",
  "withdrawal_id": "550e8400-e29b-41d4-a716-446655440001"
}

Response: 200 OK
{
  "balance": "50.00",
  "transaction_id": 2,
  "transaction_time": "2026-01-16T11:00:00Z"
}
```

**Error Responses**:
| Error Code | HTTP Status | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | 404 | Account not found |
| ERR-AC-F9L | 400 | Invalid amount (zero, negative, or invalid format) |
| ERR-AC-G9M | 400 | Insufficient balance |

---

### 5.2 Trading Module

#### API-TR-V4N: Get Price Preview (SwipeGet)

| Attribute | Value |
|-----------|-------|
| **Endpoint ID** | API-TR-V4N |
| **Method** | GET |
| **Path** | `/swipe` |
| **Purpose** | Generate and return 10-candle OHLC price preview for trading decision |
| **PRD Reference** | FEA-TR-V4N |
| **User Stories** | US-TR-V4N, US-TR-M5L, US-TR-H2M |

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| series_type | string | Yes | Volatility series: Vol50, Vol100, Vol200, Vol300 |
| account_id | string | Yes | Account identifier used to associate this price preview with the account. The generated PriceSeries is stored temporarily and matched during SwipeBuy using account_id + series_type + quote_value. |

**Response Schema** (200 OK):
```json
{
  "ohlcs": [
    {
      "timestamp": "string",   // ISO8601 timestamp
      "open": "string",        // Opening price (3 decimal precision)
      "high": "string",        // Highest price (3 decimal precision)
      "low": "string",         // Lowest price (3 decimal precision)
      "close": "string"        // Closing price (3 decimal precision)
    }
    // ... 10 candles total
  ]
}
```

**Business Rules**:
- Generates fresh random series using GBM algorithm from series initial value
- Series stored temporarily for quote validation during SwipeBuy
- 10th candle close value serves as `previous_quote` for subsequent trade
- Candles have 1-second interval timestamps
- Multiple concurrent SwipeGet calls are supported; each generates a unique PriceSeries identified by the combination of account_id + series_type + quote_value (10th candle close)

**Series Configuration**:
| Series | Initial Value | Volatility | Precision |
|--------|---------------|------------|-----------|
| Vol50 | 10000 | 50% | 0.001 |
| Vol100 | 50000 | 100% | 0.001 |
| Vol200 | 100000 | 200% | 0.001 |
| Vol300 | 200000 | 300% | 0.001 |

**Example**:
```
GET /swipe?series_type=Vol100&account_id=SW1

Response: 200 OK
{
  "ohlcs": [
    {
      "timestamp": "2026-01-16T12:00:00Z",
      "open": "50000.000",
      "high": "50012.345",
      "low": "49988.765",
      "close": "50005.234"
    },
    {
      "timestamp": "2026-01-16T12:00:01Z",
      "open": "50005.234",
      "high": "50018.456",
      "low": "49995.123",
      "close": "50010.789"
    }
    // ... 8 more candles
  ]
}
```

**Error Responses**:
| Error Code | HTTP Status | Condition |
|------------|-------------|-----------|
| ERR-TR-H2M | 400 | Invalid series type |
| ERR-AC-N4F | 404 | Account not found |

---

#### API-TR-W8P: Place Trade (SwipeBuy)

| Attribute | Value |
|-----------|-------|
| **Endpoint ID** | API-TR-W8P |
| **Method** | POST |
| **Path** | `/swipe/buy` |
| **Purpose** | Execute rise/fall binary option trade with immediate settlement |
| **PRD Reference** | FEA-TR-W8P |
| **User Stories** | US-TR-W8P, US-TR-F7K, US-TR-J8N, US-TR-P7R, US-TR-Q4N, US-TR-B6N, US-TR-R5M, US-TR-S9K |

**Request Schema**:
```json
{
  "account_id": "string",       // Required: Account identifier (e.g., "SW1")
  "stake": "string",            // Required: Positive amount, 2 decimal places
  "series_type": "string",      // Required: Vol50, Vol100, Vol200, Vol300
  "previous_quote": "string",   // Required: 10th candle close from SwipeGet
  "sentiment": "string"         // Required: "rise" or "fall"
}
```

**Response Schema** (201 Created):
```json
{
  "contract_id": "integer",      // Unique contract identifier
  "purchase_time": "string",     // ISO8601 timestamp
  "ohlcs": [
    {
      "timestamp": "string",
      "open": "string",
      "high": "string",
      "low": "string",
      "close": "string"
    }
    // ... candles 11-20 (10 candles)
  ],
  "payout": "string"             // Payout amount (e.g., "18.87" or "0.00")
}
```

**Atomic Execution Flow**:
1. Validate `previous_quote` matches stored PriceSeries
2. Validate account balance >= stake
3. Deduct stake from balance (create STAKE transaction)
4. Generate candles 11-20 from previous_quote
5. Evaluate outcome (candle 20 close vs candle 10 close)
6. Calculate payout (stake / 0.53 if win, 0 if loss)
7. Credit payout to balance (create PAYOUT transaction)
8. Create Contract with full 20 candles
9. Delete PriceSeries

**Win/Loss Evaluation**:
| Sentiment | Win Condition | Loss Condition |
|-----------|---------------|----------------|
| rise | candle20.close > candle10.close | candle20.close <= candle10.close |
| fall | candle20.close < candle10.close | candle20.close >= candle10.close |

**Payout Calculation**:
- Win payout: `stake / 0.53` (approximately stake × 1.8868)
- Loss payout: `0.00`
- All payouts rounded to 2 decimal places using standard rounding (round half up)

**Example**:
```
POST /swipe/buy
Content-Type: application/json

{
  "account_id": "SW1",
  "stake": "10.00",
  "series_type": "Vol100",
  "previous_quote": "50005.234",
  "sentiment": "rise"
}

Response: 201 Created
{
  "contract_id": 12345,
  "purchase_time": "2026-01-16T12:01:00Z",
  "ohlcs": [
    {
      "timestamp": "2026-01-16T12:00:10Z",
      "open": "50005.234",
      "high": "50020.123",
      "low": "50000.456",
      "close": "50015.789"
    }
    // ... 9 more candles (11-20)
  ],
  "payout": "18.87"
}
```

**Error Responses**:
| Error Code | HTTP Status | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | 404 | Account not found |
| ERR-TR-Q4N | 400 | Invalid quote (previous_quote mismatch) |
| ERR-TR-B6N | 400 | Insufficient balance (stake > balance) |
| ERR-TR-R5M | 400 | Invalid stake (zero, negative, or invalid format) |
| ERR-TR-H2M | 400 | Invalid series type |
| ERR-TR-S9K | 400 | Invalid sentiment (not "rise" or "fall") |

---

#### API-TR-Y5Q: List Contracts (SwipeList)

| Attribute | Value |
|-----------|-------|
| **Endpoint ID** | API-TR-Y5Q |
| **Method** | GET |
| **Path** | `/swipe/list` |
| **Purpose** | Retrieve trading history with optional series type filter |
| **PRD Reference** | FEA-TR-Y5Q |
| **User Stories** | US-TR-Y5Q, US-TR-H3K |

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | string | Yes | Account identifier |
| series_type | string | No | Filter by series type (Vol50, Vol100, Vol200, Vol300) |

**Response Schema** (200 OK):
```json
{
  "contracts": [
    {
      "contract_id": "integer",
      "purchase_time": "string",     // ISO8601 timestamp
      "series_type": "string",
      "sentiment": "string",         // "rise" or "fall"
      "stake": "string",             // 2 decimal places
      "payout": "string",            // 2 decimal places
      "ohlcs": [
        {
          "timestamp": "string",
          "open": "string",
          "high": "string",
          "low": "string",
          "close": "string"
        }
        // ... all 20 candles
      ]
    }
  ]
}
```

**Business Rules**:
- Returns maximum 50 contracts
- Ordered by `purchase_time` descending (most recent first)
- Each contract includes full 20-candle OHLC series
- Optional `series_type` filter limits results to matching series

**Example**:
```
GET /swipe/list?account_id=SW1&series_type=Vol100

Response: 200 OK
{
  "contracts": [
    {
      "contract_id": 12345,
      "purchase_time": "2026-01-16T12:01:00Z",
      "series_type": "Vol100",
      "sentiment": "rise",
      "stake": "10.00",
      "payout": "18.87",
      "ohlcs": [
        // ... 20 candles
      ]
    },
    {
      "contract_id": 12344,
      "purchase_time": "2026-01-16T11:55:00Z",
      "series_type": "Vol100",
      "sentiment": "fall",
      "stake": "5.00",
      "payout": "0.00",
      "ohlcs": [
        // ... 20 candles
      ]
    }
  ]
}
```

**Error Responses**:
| Error Code | HTTP Status | Condition |
|------------|-------------|-----------|
| ERR-AC-N4F | 404 | Account not found |
| ERR-TR-H2M | 400 | Invalid series type (if filter provided) |

---

### 5.3 Operations Module

#### API-OP-H1K: Health Check

| Attribute | Value |
|-----------|-------|
| **Endpoint ID** | API-OP-H1K |
| **Method** | GET |
| **Path** | `/health` |
| **Purpose** | Service health check for orchestration and monitoring |
| **Architecture Reference** | Section 3.1 Orchestration Requirements |

**Response Schema** (200 OK):
```json
{
  "status": "string"    // Always "healthy" when service is operational
}
```

**Business Rules**:
- Returns 200 OK with status "healthy" when the service is operational
- Used by orchestration systems for readiness and liveness checks
- No authentication required

**Example**:
```
GET /health

Response: 200 OK
{
  "status": "healthy"
}
```

**Error Responses**:
- If the service is unhealthy or unavailable, the endpoint may not respond or return a 500 status code

---

## 6. Data Models & Schemas

### 6.1 Account
```json
{
  "account_id": "string",      // SW-prefixed sequential ID (e.g., "SW1", "SW2")
  "balance": "string",         // Decimal with 2 decimal places (e.g., "150.00")
  "currency": "string"         // 3-letter uppercase code (e.g., "USD")
}
```

### 6.2 Transaction Response
```json
{
  "balance": "string",           // Updated balance after transaction
  "transaction_id": "integer",   // Auto-generated sequential ID
  "transaction_time": "string"   // ISO8601 timestamp (e.g., "2026-01-16T10:30:00Z")
}
```

### 6.3 OHLC Candle
```json
{
  "timestamp": "string",    // ISO8601 timestamp
  "open": "string",         // Opening price, 3 decimal precision
  "high": "string",         // Highest price, 3 decimal precision
  "low": "string",          // Lowest price, 3 decimal precision
  "close": "string"         // Closing price, 3 decimal precision
}
```

### 6.4 Contract
```json
{
  "contract_id": "integer",      // Auto-generated sequential ID
  "purchase_time": "string",     // ISO8601 timestamp
  "series_type": "string",       // "Vol50" | "Vol100" | "Vol200" | "Vol300"
  "sentiment": "string",         // "rise" | "fall"
  "stake": "string",             // Decimal with 2 decimal places
  "payout": "string",            // Decimal with 2 decimal places (0.00 for losses)
  "ohlcs": "OHLC[]"              // Array of 20 OHLC candles
}
```

### 6.5 Enumerations

**Series Type**:
| Value | Description |
|-------|-------------|
| Vol50 | Low volatility series (50%), initial value 10000 |
| Vol100 | Medium volatility series (100%), initial value 50000 |
| Vol200 | High volatility series (200%), initial value 100000 |
| Vol300 | Very high volatility series (300%), initial value 200000 |

**Sentiment**:
| Value | Description |
|-------|-------------|
| rise | Predict candle 20 close > candle 10 close |
| fall | Predict candle 20 close < candle 10 close |

---

## 7. Error Handling

### 7.1 Standard Error Response Format
```json
{
  "error": {
    "code": "string",        // Error code (e.g., "ERR-AC-N4F")
    "message": "string"      // Human-readable message
  }
}
```

### 7.2 Error Codes

#### Accounts Module Errors

| Error ID | Code | HTTP Status | Message | Condition |
|----------|------|-------------|---------|-----------|
| ERR-AC-N4F | ACCOUNT_NOT_FOUND | 404 | Account not found | Account ID does not exist |
| ERR-AC-R6K | INVALID_CURRENCY | 400 | Invalid currency code | Currency not 3 uppercase letters |
| ERR-AC-F9L | INVALID_AMOUNT | 400 | Invalid amount | Amount is zero, negative, or invalid format |
| ERR-AC-G9M | INSUFFICIENT_BALANCE | 400 | Insufficient balance | Withdrawal amount > balance |

#### Trading Module Errors

| Error ID | Code | HTTP Status | Message | Condition |
|----------|------|-------------|---------|-----------|
| ERR-TR-H2M | INVALID_SERIES_TYPE | 400 | Invalid series type | Series type not Vol50/Vol100/Vol200/Vol300 |
| ERR-TR-S9K | INVALID_SENTIMENT | 400 | Invalid sentiment | Sentiment not "rise" or "fall" |
| ERR-TR-R5M | INVALID_STAKE | 400 | Invalid stake amount | Stake is zero, negative, or invalid format |
| ERR-TR-B6N | INSUFFICIENT_BALANCE | 400 | Insufficient balance for trade | Stake amount > account balance |
| ERR-TR-Q4N | INVALID_QUOTE | 400 | Quote does not match preview | previous_quote mismatch with stored series |

#### Idempotent Operation Responses

| Response ID | Code | HTTP Status | Message | Condition |
|-------------|------|-------------|---------|-----------|
| - | DUPLICATE_TRANSACTION | 200 | Transaction already processed | Idempotency key (deposit_id/withdrawal_id) was already used; returns original transaction result |

**Note**: DUPLICATE_TRANSACTION is a success response (HTTP 200), not an error. When an idempotent operation is retried with the same idempotency key, the original successful result is returned. This ensures safe retry behavior for deposits and withdrawals.

### 7.3 HTTP Status Code Summary

| Status Code | Meaning | Usage |
|-------------|---------|-------|
| 200 | OK | Successful GET, successful idempotent POST (duplicate) |
| 201 | Created | Successful resource creation (account, contract) |
| 400 | Bad Request | Validation errors, business rule violations |
| 404 | Not Found | Resource not found (account) |
| 500 | Internal Server Error | Unexpected server errors |

---

## 8. Integration Guide

### 8.1 Getting Started

**Step 1: Create an Account**
```bash
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"currency": "USD"}'
```
Response: `{"account_id": "SW1"}`

**Step 2: Deposit Funds**
```bash
curl -X POST http://localhost:8080/accounts/SW1/deposits \
  -H "Content-Type: application/json" \
  -d '{"amount": "100.00", "deposit_id": "550e8400-e29b-41d4-a716-446655440000"}'
```
Response: `{"balance": "100.00", "transaction_id": 1, "transaction_time": "..."}`

**Step 3: Get Price Preview**
```bash
curl "http://localhost:8080/swipe?series_type=Vol100&account_id=SW1"
```
Response: 10 OHLC candles - note the 10th candle's close value as `previous_quote`

**Step 4: Place a Trade**
```bash
curl -X POST http://localhost:8080/swipe/buy \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "SW1",
    "stake": "10.00",
    "series_type": "Vol100",
    "previous_quote": "50005.234",
    "sentiment": "rise"
  }'
```
Response: Contract with candles 11-20 and payout result

### 8.2 Trading Flow Best Practices

1. **Always use the latest preview**: Call SwipeGet immediately before placing a trade
2. **Store the 10th candle close**: This becomes `previous_quote` for SwipeBuy
3. **Handle quote expiration**: If SwipeBuy returns INVALID_QUOTE, fetch a new preview
4. **Idempotency for deposits/withdrawals**: Always generate unique UUIDs for each financial operation
5. **Check balance before trading**: Use GET `/accounts/{id}` to verify sufficient funds

### 8.3 UI Animation Pattern

For the React frontend, implement the following animation flow:

1. **Preview Phase**: Display candles 1-10 from SwipeGet response
2. **Trading Decision**: Show RISE/FALL buttons alongside stake input
3. **Execution Phase**: After SwipeBuy response, animate candles 11-20 sequentially
4. **Result Display**: Show WIN/LOSS and payout amount after final candle

### 8.4 Error Handling Patterns

```javascript
// Example error handling
async function placeTrade(request) {
  try {
    const response = await fetch('/swipe/buy', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request)
    });
    
    if (!response.ok) {
      const error = await response.json();
      switch (error.error.code) {
        case 'INVALID_QUOTE':
          // Refresh price preview
          return refreshPreview();
        case 'INSUFFICIENT_BALANCE':
          // Show balance error to user
          return showInsufficientBalance();
        default:
          return showError(error.error.message);
      }
    }
    
    return response.json();
  } catch (e) {
    return showNetworkError();
  }
}
```

---

## 9. Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-16 | Archi | Initial API specification |
| 1.1 | 2026-01-16 | Archi | Self-review: Enhanced documentation clarity for SwipeGet account_id parameter, payout rounding behavior, and concurrent request handling |
| 1.2 | 2026-01-16 | Archi | Verification fixes: Added GET /health endpoint (API-OP-H1K), added DUPLICATE_TRANSACTION to error table (Section 7.2) |

---

## Appendix A: PRD Requirement Traceability

| PRD Requirement | Endpoint | Implementation Notes |
|-----------------|----------|---------------------|
| REQ-AC-K3M | API-AC-K3M | Account creation with currency |
| REQ-AC-P8R | API-AC-K3M | SW-prefixed sequential IDs |
| REQ-AC-X2L | API-AC-K3M | 3-letter currency validation |
| REQ-AC-H5N | API-AC-M9J | Credit deposit amount |
| REQ-AC-L2Q | API-AC-M9J | Idempotency via deposit_id |
| REQ-AC-B7N | API-AC-R3P | Debit withdrawal amount |
| REQ-AC-D4Q | API-AC-R3P | Idempotency via withdrawal_id |
| REQ-AC-G9M | API-AC-R3P | Balance validation |
| REQ-AC-E2N | API-AC-K6L | Return balance |
| REQ-AC-S8Q | API-AC-K6L | Return currency |
| REQ-TR-A7M | API-TR-V4N | Generate 10 candles |
| REQ-TR-F3Q | API-TR-V4N | GBM per series type |
| REQ-TR-J9K | API-TR-V4N | OHLC data structure |
| REQ-TR-B6N | API-TR-W8P | Deduct stake |
| REQ-TR-E2Q | API-TR-W8P | Generate candles 11-20 |
| REQ-TR-I8M | API-TR-W8P | Immediate evaluation |
| REQ-TR-M4K | API-TR-W8P | Payout calculation |
| REQ-TR-Q1L | API-TR-W8P | Store full 20 candles |
| REQ-TR-C9N | API-TR-Y5Q | Return last 50 |
| REQ-TR-G5Q | API-TR-Y5Q | Filter by series_type |
| REQ-TR-K1M | API-TR-Y5Q | Return full 20 candles |
| REQ-TR-O7K | API-TR-Y5Q | Order by purchase_time desc |

---

## Appendix B: User Story Coverage

| Story ID | Endpoint | Coverage |
|----------|----------|----------|
| US-AC-K3M | API-AC-K3M | ✓ Create account |
| US-AC-M9J | API-AC-M9J | ✓ Deposit funds |
| US-AC-R3P | API-AC-R3P | ✓ Withdraw funds |
| US-AC-K6L | API-AC-K6L | ✓ View account |
| US-AC-D4Q | API-AC-M9J | ✓ Idempotent deposits |
| US-AC-W5N | API-AC-R3P | ✓ Idempotent withdrawals |
| US-AC-G9M | API-AC-R3P | ✓ Insufficient balance error |
| US-AC-F9L | API-AC-M9J, API-AC-R3P | ✓ Invalid amount error |
| US-AC-R6K | API-AC-K3M | ✓ Invalid currency error |
| US-AC-N4F | All /accounts/* | ✓ Account not found error |
| US-AC-X2L | API-AC-K3M | ✓ Broker account creation |
| US-AC-E8P | API-AC-K3M | ✓ Flexible external_id |
| US-TR-V4N | API-TR-V4N | ✓ Price preview |
| US-TR-M5L | API-TR-V4N | ✓ Select series type |
| US-TR-W8P | API-TR-W8P | ✓ Buy rise contract |
| US-TR-F7K | API-TR-W8P | ✓ Buy fall contract |
| US-TR-J8N | API-TR-W8P | ✓ Candles 11-20 in response |
| US-TR-P7R | API-TR-W8P | ✓ Payout in response |
| US-TR-Y5Q | API-TR-Y5Q | ✓ View trading history |
| US-TR-H3K | API-TR-Y5Q | ✓ Filter by series type |
| US-TR-Q4N | API-TR-W8P | ✓ Quote validation |
| US-TR-B6N | API-TR-W8P | ✓ Stake exceeds balance error |
| US-TR-R5M | API-TR-W8P | ✓ Invalid stake error |
| US-TR-H2M | API-TR-V4N, API-TR-W8P | ✓ Invalid series type error |
| US-TR-S9K | API-TR-W8P | ✓ Invalid sentiment error |
