# Product Requirements Document (PRD)
# Digital Call/Put Options Pricing Service

**Version**: 1.0  
**Date**: 2026-01-08  
**Status**: Draft  
**Complexity Score**: 6/10 (Standard)

---

## 1. Executive Summary

### 1.1 Product Overview

The Digital Call/Put Options Pricing Service is a stateless Go microservice that provides real-time pricing for digital (binary) options contracts. The service calculates Ask prices (for contract purchase) and Bid prices (for contract valuation/sale) using the Black-Scholes pricing model.

Digital options are binary contracts where:
- **Call Option**: Client wins full payout if exit price > barrier
- **Put Option**: Client wins full payout if exit price < barrier

### 1.2 Business Objectives

| Objective | Success Metric |
|-----------|----------------|
| Enable real-time options pricing | Response time < 100ms for unary calls |
| Support streaming price updates | Stream latency < 500ms from tick receipt |
| Ensure pricing accuracy | Black-Scholes calculations accurate to 8 decimal places |
| Maintain contract integrity | Payout immutable after purchase |

### 1.3 Scope

**In Scope**:
- Digital Call/Put option pricing (Ask)
- Active contract valuation (Bid)
- Single request and streaming endpoints
- Time-based and tick-based contract durations
- Integration with service-feed for market data

**Out of Scope**:
- Contract storage/persistence (stateless service)
- User authentication (handled by upstream services)
- Trade execution (pricing only)
- Risk management systems

---

## 2. Functional Requirements

### 2.1 Contract Types

#### REQ-CT-K3M: Call Option Contract
**Priority**: P0 (Critical)

A Call option contract where the client wins full payout if the underlying asset's exit price is strictly higher than the barrier.

**Acceptance Criteria**:
- IF exit_price > barrier THEN payout = full_payout
- IF exit_price <= barrier THEN payout = 0

#### REQ-CT-P7R: Put Option Contract
**Priority**: P0 (Critical)

A Put option contract where the client wins full payout if the underlying asset's exit price is strictly lower than the barrier.

**Acceptance Criteria**:
- IF exit_price < barrier THEN payout = full_payout
- IF exit_price >= barrier THEN payout = 0

### 2.2 API Endpoints

#### REQ-AP-G1A: GetAsk Endpoint
**Priority**: P0 (Critical)

Unary gRPC endpoint that returns a single Ask price for a proposed contract.

**User Story**: As a trader, I want to request a contract price so that I can decide whether to purchase the option.

**Request Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| symbol | string | Yes | Underlying asset (e.g., USD/JPY, BTC/USD) |
| contract_type | enum | Yes | CALL or PUT |
| currency | string | Yes | Payout currency (e.g., USD, EUR) |
| duration | string | Yes | Duration string (e.g., "1m", "30s", "5t") |
| stake | string | Yes | Premium amount |
| barrier | string | No | Relative (+/-) or absolute value |
| pricing_time | int64 | No | Epoch time for pricing |

**Response Fields**:
| Field | Type | Description |
|-------|------|-------------|
| ask_price | string | Calculated ask price |
| currency | string | Contract currency |
| current_spot | string | Current spot price |
| current_spot_time | int64 | Spot price timestamp |
| payout | string | Potential payout amount |
| limits | Limits | Trading limits (max_payout, min_stake) |

**Acceptance Criteria**:
- Response returned within 100ms under normal load
- All monetary values returned as strings with 8 decimal precision
- Returns appropriate gRPC error codes on failure

#### REQ-AP-S2B: StreamAsk Endpoint
**Priority**: P0 (Critical)

Server-streaming gRPC endpoint that streams Ask prices as market conditions change.

**User Story**: As a trader, I want to receive continuous price updates so that I can trade at the optimal moment.

**Acceptance Criteria**:
- Stream initiates within 500ms of request
- Updates sent on each new tick OR every 5 seconds (whichever is first)
- Stream terminates gracefully on client disconnect

#### REQ-AP-G3C: GetBid Endpoint
**Priority**: P0 (Critical)

Unary gRPC endpoint that returns the current market value of an active contract.

**User Story**: As a trader, I want to know the current value of my contract so that I can decide whether to sell early.

**Additional Request Parameters** (beyond Ask parameters):
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| start_time | int64 | Yes | Contract start timestamp |
| payout | string | Yes | Fixed payout from purchase |

**Response Fields**:
| Field | Type | Description |
|-------|------|-------------|
| bid_price | string | Current market value |
| is_expired | bool | Contract expiration status |
| current_spot | string | Current spot price |
| current_spot_time | int64 | Spot timestamp |
| entry_spot | string | Entry tick price |
| entry_spot_time | int64 | Entry tick timestamp (market data time) |
| exit_spot | string | Exit tick price (expired only) |
| exit_spot_time | int64 | Exit tick timestamp (market data time) |
| barrier | string | Resolved barrier value |
| start_time | int64 | Contract start time |
| expiry_time | int64 | Contract expiry time |
| currency | string | Contract currency |

**Acceptance Criteria**:
- payout parameter is mandatory and used as-is (never recalculated)
- entry_spot_time reflects actual market feed timestamp
- exit_spot/exit_spot_time only populated when is_expired=true

#### REQ-AP-S4D: StreamBid Endpoint
**Priority**: P0 (Critical)

Server-streaming gRPC endpoint that streams Bid prices for active contracts.

**User Story**: As a trader, I want to monitor my contract value in real-time so that I can sell at the optimal moment.

**Acceptance Criteria**:
- Time-based: Updates on tick OR every 5 seconds
- Tick-based: Updates ONLY on new ticks
- Stream terminates when contract expires (final update includes exit data)

### 2.3 Barrier Logic

#### REQ-BR-A1E: Absolute Barrier
**Priority**: P0 (Critical)

When barrier parameter is a numeric value without +/- prefix, use it as the absolute barrier value.

**Business Rule BR-BR-Q2M**:
- IF barrier is numeric (e.g., "123.45") THEN barrier_value = barrier

#### REQ-BR-R2F: Relative Barrier
**Priority**: P0 (Critical)

When barrier parameter starts with '+' or '-', calculate barrier relative to entry price.

**Business Rule BR-BR-L8K**:
- IF barrier starts with '+' (e.g., "+0.0023") THEN barrier_value = entry_price + value
- IF barrier starts with '-' (e.g., "-0.0023") THEN barrier_value = entry_price - value

#### REQ-BR-N3G: Default Barrier (At-The-Money)
**Priority**: P0 (Critical)

When no barrier is provided, use entry price as the barrier.

**Business Rule BR-BR-T5N**:
- IF barrier is null/empty THEN barrier_value = entry_price

### 2.4 Duration Types

#### REQ-DU-T1H: Time-Based Duration
**Priority**: P0 (Critical)

Durations specified in seconds (s), minutes (m), hours (h), or days (d).

**Supported Formats**:
| Unit | Example | Meaning |
|------|---------|---------|
| s | "30s" | 30 seconds |
| m | "1m" | 1 minute |
| h | "2h" | 2 hours |
| d | "5d" | 5 days |

**Business Rule BR-DU-M9J**:
- expiry_time = start_time + duration
- IF now >= expiry_time THEN is_expired = true
- Maximum duration: 1 year (365 days)

**Acceptance Criteria**:
- Duration parsing validates format and range
- Invalid durations return INVALID_ARGUMENT error

#### REQ-DU-K2I: Tick-Based Duration
**Priority**: P0 (Critical)

Durations specified in ticks (t).

**Supported Format**:
| Unit | Example | Meaning |
|------|---------|---------|
| t | "5t" | 5 ticks after entry |

**Business Rule BR-DU-P7R**:
- Contract expires when tick_count >= required_ticks
- NO time-based expiry comparison
- Maximum duration: 10 ticks

**Acceptance Criteria**:
- Tick counter maintained internally per stream
- Expiry based purely on tick count, not time

### 2.5 Pricing Logic

#### REQ-PR-A1J: Ask Price Calculation
**Priority**: P0 (Critical)

Calculate Ask price using Black-Scholes formula.

**Feature FEA-PR-T5N**: Black-Scholes Pricing Engine
**Priority**: P0

**Inputs**:
| Input | Source | Value |
|-------|--------|-------|
| Spot Price | service-feed | Real-time |
| Barrier | Resolved from input | Calculated |
| Duration (years) | Input | Converted from duration string |
| Volatility | Configuration | 10% (0.10) |
| Interest Rate | Configuration | 0% (0.00) |
| Quanto Drift | Configuration | 0% (0.00) |

**Formula Application**:
- Use standard Black-Scholes for digital options
- Deduct commission from potential payout
- Commission rate from YAML configuration per symbol

**Acceptance Criteria**:
- Calculations accurate to 8 decimal places
- ask_price <= stake (premium cannot exceed stake)
- payout = (stake / probability) - commission

#### REQ-PR-B2K: Bid Price Calculation (Time-Based)
**Priority**: P0 (Critical)

Calculate Bid price for active time-based contracts using Black-Scholes with remaining time.

**Business Rule BR-PR-Q2M**:
- remaining_time = expiry_time - now
- Apply Black-Scholes with remaining_time
- Use original payout from request (never recalculate)

**Acceptance Criteria**:
- Bid price reflects current market conditions
- Original payout preserved throughout contract lifecycle

#### REQ-PR-N3L: Bid Price for Tick-Based Contracts
**Priority**: P0 (Critical)

Tick-based contracts do not support early exit.

**Business Rule BR-PR-L8K**:
- IF is_expired = false AND duration_type = tick THEN bid_price = 0 (or not applicable)
- IF is_expired = true THEN bid_price = payout (if won) OR 0 (if lost)

**Acceptance Criteria**:
- No bid price calculated for active tick-based contracts
- Final bid price reflects win/loss outcome

### 2.6 Contract Lifecycle

#### REQ-LC-E1M: Entry Tick Determination
**Priority**: P0 (Critical)

Entry price is the first tick received AFTER start_time.

**Business Rule BR-LC-T5N**:
- Wait for tick where tick_time > start_time
- entry_spot = tick.price
- entry_spot_time = tick.timestamp (from market feed, not calculated)

**Acceptance Criteria**:
- entry_spot_time is the actual market data timestamp
- Entry price used for barrier calculation if relative/default barrier

#### REQ-LC-X2N: Exit Tick Determination
**Priority**: P0 (Critical)

Exit price is determined at contract expiry.

**Time-Based Exit**:
- exit_spot = latest tick at or after expiry_time
- exit_spot_time = tick.timestamp from market feed

**Tick-Based Exit**:
- exit_spot = Nth tick after entry
- exit_spot_time = tick.timestamp of Nth tick

**Acceptance Criteria**:
- exit_spot_time is the actual market data timestamp
- exit_spot/exit_spot_time only populated when is_expired = true

#### REQ-LC-P3O: Payout Immutability
**Priority**: P0 (Critical)

Payout amount is fixed at purchase time and MUST NOT be recalculated.

**Business Rule BR-LC-M9J**:
- Payout calculated once during GetAsk/StreamAsk
- Bid requests MUST include payout as parameter
- Service uses provided payout, never recalculates

**Rationale**:
- Contract terms legally binding at purchase
- Market conditions may change post-purchase
- Financial audit requirements demand consistency

**Acceptance Criteria**:
- GetBid/StreamBid requires payout parameter
- Service returns error if payout missing from Bid request

### 2.7 Stream Behavior

#### REQ-ST-U1P: Time-Based Stream Updates
**Priority**: P1 (Important)

Stream updates for time-based contracts.

**Business Rule BR-ST-Q2M**:
- Send update on new tick arrival OR
- Send update every 5 seconds if no ticks arrive

**Acceptance Criteria**:
- 5-second timer resets on each tick
- Updates contain current pricing, not heartbeats

#### REQ-ST-T2Q: Tick-Based Stream Updates
**Priority**: P1 (Important)

Stream updates for tick-based contracts.

**Business Rule BR-ST-L8K**:
- Send update ONLY on new tick arrival
- NO time-based fallback updates

**Acceptance Criteria**:
- Stream silent between ticks (no periodic updates)
- Each tick increments internal counter

---

## 3. Non-Functional Requirements

### 3.1 Performance Requirements

#### NFR-PF-L1A: Response Latency
| Metric | Target | Measurement |
|--------|--------|-------------|
| GetAsk response time | < 100ms p95 | End-to-end |
| GetBid response time | < 100ms p95 | End-to-end |
| Stream first response | < 500ms | From request to first message |
| Stream update latency | < 200ms | From tick receipt to client delivery |

#### NFR-PF-T2B: Throughput
| Metric | Target |
|--------|--------|
| Concurrent streams | 10,000 per instance |
| Requests per second | 5,000 RPS per instance |

### 3.2 Reliability Requirements

#### NFR-RL-A1C: Availability
- Target: 99.9% uptime
- Graceful degradation on service-feed unavailability

#### NFR-RL-F2D: Fault Tolerance
- Return appropriate gRPC error codes on failures
- No partial responses (all-or-nothing)

### 3.3 Precision Requirements

#### NFR-PR-D1E: Numerical Precision
| Data Type | Precision |
|-----------|-----------|
| Monetary values | 8 decimal places |
| Prices (spot, barrier) | 8 decimal places |
| Duration (years) | 10 decimal places |

**Rationale**: Financial calculations require high precision to prevent rounding errors that accumulate across transactions.

---

## 4. External Dependencies

### 4.1 service-feed Integration

**Dependency**: Market data feed service
**Repository**: https://github.com/junbon-deriv/service-feed
**Proto Location**: proto/grpcfeed/v1/ticks.proto
**Client Implementation**: client/client.go

**Required Endpoints** (to be verified from actual proto):
| Endpoint | Usage |
|----------|-------|
| GetLatestTick | Single tick retrieval for unary pricing |
| StreamTicks | Continuous tick stream for streaming endpoints |
| GetTicks | Historical ticks for entry/exit determination |

**Integration Requirements**:
- Clone and verify actual API before implementation
- Use GetLatestTick for single requests (not StreamTicks)
- Use StreamTicks for streaming price updates

### 4.2 Symbol Configuration

**Source**: YAML configuration file
**Location**: To be defined during architecture phase

**Configuration Parameters per Symbol**:
| Parameter | Type | Description |
|-----------|------|-------------|
| min_stake | decimal | Minimum stake amount |
| max_payout | decimal | Maximum payout amount |
| commission | decimal | Commission rate |

---

## 5. Data Requirements

### 5.1 Input Validation Rules

| Field | Validation Rules |
|-------|------------------|
| symbol | Non-empty, must exist in configuration |
| contract_type | Must be CALL or PUT (not UNSPECIFIED) |
| currency | Valid 3-letter currency code |
| duration | Format: [number][unit], unit in {s,m,h,d,t} |
| stake | Positive decimal, >= min_stake |
| barrier | Optional; if present, valid format (+/-prefix or absolute) |
| start_time | Required for Bid; must be past timestamp |
| payout | Required for Bid; positive decimal |

### 5.2 Duration Validation

| Type | Range |
|------|-------|
| Time-based (s,m,h,d) | 1 second to 365 days |
| Tick-based (t) | 1 tick to 10 ticks |

---

## 6. Error Handling

### 6.1 gRPC Error Codes

| Scenario | gRPC Code | Description |
|----------|-----------|-------------|
| Missing required field | INVALID_ARGUMENT | Specify which field |
| Invalid duration format | INVALID_ARGUMENT | Include valid format hint |
| Stake below minimum | INVALID_ARGUMENT | Include min_stake value |
| Payout above maximum | INVALID_ARGUMENT | Include max_payout value |
| Unknown symbol | NOT_FOUND | Symbol not in configuration |
| service-feed unavailable | UNAVAILABLE | Upstream dependency down |
| Internal calculation error | INTERNAL | Log details, generic message |
| Missing payout in Bid | INVALID_ARGUMENT | Payout required for Bid |

---

## 7. Architecture Constraints

### 7.1 Service Design Rules

Per workspace preferences, the following rules MUST be followed:

1. **Handler Responsibility**: gRPC handlers receive requests and delegate to higher-level modules. Handlers MUST NOT fetch data from various sources or assemble responses directly.

2. **Error Codes**: Always return standard gRPC error codes with descriptive messages.

3. **No models/types Package**: Dependencies flow toward core (pricing). Other packages depend on pricing; pricing depends on nothing else.

4. **Interface Location**: Interfaces defined where consumed, not where implemented.

### 7.2 Stateless Design

- Service maintains no persistent state
- No database access
- All contract state reconstructed from request parameters
- Tick counts for streaming maintained per-stream only

---

## 8. Glossary

| Term | Definition |
|------|------------|
| Ask | The price to purchase (enter) a contract |
| Bid | The current market value of an active contract |
| Barrier | The price level that determines win/loss |
| Digital Option | Binary option with fixed payout on win |
| Entry Spot | First tick price after contract start |
| Exit Spot | Final tick price at contract expiry |
| Payout | Fixed amount client receives on winning |
| Stake | Premium paid to enter the contract |
| Tick | A single price update from market feed |

---

## 9. Appendix

### 9.1 Proto Definition Reference

The complete protobuf definition is provided in the product brief at:
[`workspace/input/product_brief.md`](../../input/product_brief.md:113) - Part 2, Section 1.

### 9.2 Complexity Assessment

| Dimension | Score | Justification |
|-----------|-------|---------------|
| Technical Complexity | 2/3 | Multiple components, Black-Scholes, dual duration types |
| User Complexity | 0/2 | Single user type, no auth in service |
| Integration Complexity | 1/2 | Single external integration (service-feed) |
| Regulatory/Compliance | 2/2 | Financial trading, payout immutability, audit requirements |
| Scale/Performance | 1/1 | Real-time pricing, streaming, performance critical |
| **Total** | **6/10** | Standard complexity |

---

## 10. Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-08 | AI Architect | Initial PRD creation |
