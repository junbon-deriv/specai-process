# Internal API Specification: Double Rise/Fall Pricing Service

> **Service**: `service-pricer-doublerisefall`
> **API Type**: Internal (gRPC)
> **Version**: 1.0.0
> **Status**: Draft
> **Created**: 2026-01-15

---

## Overview

This document specifies the internal gRPC API for the Double Rise/Fall Pricing Service (`service-pricer-doublerisefall`). This service calculates ask (purchase) and bid (valuation) prices for Double Rise/Fall binary options contracts.

### Purpose
Calculate contract prices for Double Rise/Fall binary options using a bivariate normal distribution pricing model.

### Scope
- Ask price calculation (contract purchase)
- Bid price calculation (active contract valuation)
- Real-time price streaming

### Target Consumers
| Consumer | Purpose | Priority |
|----------|---------|----------|
| `api-gateway-trading` | Contract pricing for client display and purchase | **Critical** |

---

## Authentication & Authorization

### Service-to-Service Authentication
| Aspect | Value |
|--------|-------|
| **Method** | mTLS (mutual TLS) |
| **Certificate Management** | Kubernetes secrets |
| **Authorization** | Service mesh policy (allowed consumers list) |

### Allowed Consumers
| Service | Access Level |
|---------|--------------|
| `api-gateway-trading` | Full (all methods) |

> **Note**: This is an internal service with no external authentication. Access is controlled via service mesh policies.

---

## Base Configuration

| Property | Value |
|----------|-------|
| **Protocol** | gRPC / Protocol Buffers |
| **Proto Package** | `doublerisefall.v1` |
| **Service Name** | `DoubleRiseFallService` |
| **Default Port** | 50051 |
| **Health Check Port** | 8081 (HTTP) |
| **Content Type** | `application/grpc` |
| **Versioning** | Package versioning (`v1`, `v2`, etc.) |

### Service Discovery
| Property | Value |
|----------|-------|
| **Kubernetes Service** | `service-pricer-doublerisefall.default.svc.cluster.local:50051` |
| **Environment Variable** | `DOUBLERISEFALL_SERVICE_ADDR` |

---

## Table of Endpoints

| Method | RPC Type | Request | Response | Summary |
|--------|----------|---------|----------|---------|
| `GetAsk` | Unary | `GetAskRequest` | `GetAskResponse` | Calculate contract purchase price |
| `StreamAsk` | Server Stream | `StreamAskRequest` | `stream GetAskResponse` | Real-time price updates |
| `GetBid` | Unary | `GetBidRequest` | `GetBidResponse` | Value an active contract |
| `StreamBid` | Server Stream | `StreamBidRequest` | `stream GetBidResponse` | Real-time contract valuation |

---

## Endpoints & Methods

### API-DR-A1K: GetAsk

Calculate the ask (purchase) price for a Double Rise/Fall contract.

| Property | Value |
|----------|-------|
| **ID** | `API-DR-A1K` |
| **Method** | `GetAsk` |
| **RPC Type** | Unary |
| **Request** | `GetAskRequest` |
| **Response** | `GetAskResponse` |

#### Purpose
Calculate the contract purchase price (stake to payout ratio) for a new Double Rise/Fall contract based on current market conditions.

#### Request Schema

```protobuf
message GetAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;        // Unix epoch (seconds)
}
```

#### Response Schema

```protobuf
message GetAskResponse {
  string ask_price = 1;                   // Contract price (premium)
  string currency = 2;                    // Quote currency
  string current_spot = 3;                // Current market price
  int64 current_spot_time = 4;            // Unix epoch of spot price
  string payout = 5;                      // Potential payout
  Limits limits = 6;                      // Trading limits
}
```

#### Success Response
- **gRPC Status**: `OK`
- Returns `GetAskResponse` with calculated price

#### Error Responses
| gRPC Status | Error Code | Condition |
|-------------|------------|-----------|
| `INVALID_ARGUMENT` | `ERR-DR-S1V` | Invalid symbol |
| `INVALID_ARGUMENT` | `ERR-DR-D2U` | Invalid duration format |
| `INVALID_ARGUMENT` | `ERR-DR-D3O` | Duration order invalid (t2 ≤ t1) |
| `INVALID_ARGUMENT` | `ERR-DR-D4G` | Duration gap invalid |
| `INVALID_ARGUMENT` | `ERR-DR-K5S` | Stake below minimum |
| `INVALID_ARGUMENT` | `ERR-DR-P6X` | Payout exceeds maximum |
| `INVALID_ARGUMENT` | `ERR-DR-T2F` | Future pricing time |
| `FAILED_PRECONDITION` | `ERR-DR-Y7D` | Symbol disabled |
| `UNAVAILABLE` | `ERR-DR-M8E` | Market data unavailable |
| `INTERNAL` | `ERR-DR-I9N` | Internal server error |

#### Example

**Request:**
```json
{
  "option_parameters": {
    "symbol": "R_100",
    "contract_type": "CONTRACT_TYPE_RISE",
    "currency": "USD",
    "first_duration": "1m",
    "second_duration": "2m",
    "stake": "10.00"
  }
}
```

**Response:**
```json
{
  "ask_price": "10.00",
  "currency": "USD",
  "current_spot": "1234.5678",
  "current_spot_time": 1736930731,
  "payout": "27.85",
  "limits": {
    "max_payout": "1000.00",
    "min_stake": "1.00"
  }
}
```

---

### API-DR-S2T: StreamAsk

Stream real-time ask price updates for a Double Rise/Fall contract.

| Property | Value |
|----------|-------|
| **ID** | `API-DR-S2T` |
| **Method** | `StreamAsk` |
| **RPC Type** | Server Streaming |
| **Request** | `StreamAskRequest` |
| **Response** | `stream GetAskResponse` |

#### Purpose
Provide continuous price updates for contract pricing display. Updates are sent on each market tick or every 5 seconds (whichever comes first) for time-based contracts.

#### Request Schema

```protobuf
message StreamAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}
```

#### Response Schema
Returns a stream of `GetAskResponse` messages (same as `GetAsk`).

#### Stream Behavior
| Contract Type | Update Trigger |
|---------------|----------------|
| Time-based | On tick OR every 5 seconds |
| Tick-based | On tick only |

#### Error Responses
Same as `GetAsk`, plus:
| gRPC Status | Error Code | Condition |
|-------------|------------|-----------|
| `UNAVAILABLE` | `ERR-DR-C1D` | Stream disconnected |

---

### API-DR-B3V: GetBid

Calculate the bid (valuation) price for an active Double Rise/Fall contract.

| Property | Value |
|----------|-------|
| **ID** | `API-DR-B3V` |
| **Method** | `GetBid` |
| **RPC Type** | Unary |
| **Request** | `GetBidRequest` |
| **Response** | `GetBidResponse` |

#### Purpose
Determine the current value of an active contract, including whether it has expired and the win/loss outcome.

#### Request Schema

```protobuf
message GetBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}
```

**Required Fields for Bid**:
- `option_parameters.start_time` - Contract start time (required)
- `option_parameters.payout` - Original payout from Ask (required)

#### Response Schema

```protobuf
message GetBidResponse {
  string bid_price = 1;                   // Current contract value
  bool is_expired = 2;                    // Expiry status
  string current_spot = 3;                // Current market price
  int64 current_spot_time = 4;            // Timestamp of current spot
  string entry_spot = 5;                  // Entry price (barrier)
  int64 entry_spot_time = 6;              // Timestamp of entry tick
  string exit_spot = 7;                   // Exit price (if expired)
  int64 exit_spot_time = 8;               // Timestamp of exit tick
  string barrier = 9;                     // Resolved barrier value
  int64 start_time = 10;                  // Contract start time
  int64 expiry_time = 11;                 // Contract expiry time (t2)
  string currency = 12;                   // Quote currency
  int64 evaluation_time = 13;             // First evaluation time (t1)
}
```

#### Bid Price Logic
| Contract State | `bid_price` | `is_expired` |
|----------------|-------------|--------------|
| Active (pre-t1) | Current spot info | `false` |
| Active (post-t1, pre-t2) | Depends on t1 result | `false` |
| Expired - Won | Original payout | `true` |
| Expired - Lost | `"0"` | `true` |

#### Success Response
- **gRPC Status**: `OK`
- Returns `GetBidResponse` with contract valuation

#### Error Responses
| gRPC Status | Error Code | Condition |
|-------------|------------|-----------|
| `INVALID_ARGUMENT` | `ERR-DR-S1V` | Invalid symbol |
| `INVALID_ARGUMENT` | `ERR-DR-D2U` | Invalid duration |
| `INVALID_ARGUMENT` | `ERR-DR-T1M` | Missing start_time |
| `INVALID_ARGUMENT` | `ERR-DR-P2Y` | Missing payout |
| `FAILED_PRECONDITION` | `ERR-DR-E3T` | Missing entry tick |
| `UNAVAILABLE` | `ERR-DR-M8E` | Market data unavailable |
| `INTERNAL` | `ERR-DR-I9N` | Internal server error |

#### Example

**Request:**
```json
{
  "option_parameters": {
    "symbol": "R_100",
    "contract_type": "CONTRACT_TYPE_RISE",
    "currency": "USD",
    "first_duration": "1m",
    "second_duration": "2m",
    "start_time": 1736930600,
    "stake": "10.00",
    "payout": "27.85"
  }
}
```

**Response (Expired - Won):**
```json
{
  "bid_price": "27.85",
  "is_expired": true,
  "current_spot": "1235.1234",
  "current_spot_time": 1736930731,
  "entry_spot": "1234.5678",
  "entry_spot_time": 1736930601,
  "exit_spot": "1235.1234",
  "exit_spot_time": 1736930720,
  "barrier": "1234.5678",
  "start_time": 1736930600,
  "expiry_time": 1736930720,
  "currency": "USD",
  "evaluation_time": 1736930660
}
```

---

### API-DR-S4B: StreamBid

Stream real-time bid price updates for an active Double Rise/Fall contract.

| Property | Value |
|----------|-------|
| **ID** | `API-DR-S4B` |
| **Method** | `StreamBid` |
| **RPC Type** | Server Streaming |
| **Request** | `StreamBidRequest` |
| **Response** | `stream GetBidResponse` |

#### Purpose
Provide continuous contract valuation updates until contract expiry.

#### Request Schema

```protobuf
message StreamBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}
```

**Required Fields**:
- `option_parameters.start_time` - Contract start time
- `option_parameters.payout` - Original payout from Ask

#### Response Schema
Returns a stream of `GetBidResponse` messages (same as `GetBid`).

#### Stream Behavior
| Contract Type | Update Trigger | Termination |
|---------------|----------------|-------------|
| Time-based | On tick OR every 5 seconds | After expiry response |
| Tick-based | On tick only | After expiry response |

#### Stream Termination
The stream terminates automatically after sending the final expiry response (`is_expired: true`).

#### Error Responses
Same as `GetBid`, plus:
| gRPC Status | Error Code | Condition |
|-------------|------------|-----------|
| `UNAVAILABLE` | `ERR-DR-C1D` | Stream disconnected |

---

## Data Models & Schemas

### ContractType (Enum)

```protobuf
enum ContractType {
  CONTRACT_TYPE_UNSPECIFIED = 0;
  CONTRACT_TYPE_RISE = 1;
  CONTRACT_TYPE_FALL = 2;
}
```

| Value | Description |
|-------|-------------|
| `CONTRACT_TYPE_UNSPECIFIED` | Invalid/unset |
| `CONTRACT_TYPE_RISE` | Win if spot > barrier at both t1 AND t2 |
| `CONTRACT_TYPE_FALL` | Win if spot < barrier at both t1 AND t2 |

### OptionParameters (Message)

```protobuf
message OptionParameters {
  string symbol = 1;                      // Required: Underlying asset (e.g., "R_100")
  ContractType contract_type = 2;         // Required: Contract direction
  string currency = 3;                    // Required: Payout currency (e.g., "USD")
  string first_duration = 4;              // Required: Duration to t1 (e.g., "1m", "5t")
  string second_duration = 5;             // Required: Duration to t2 (e.g., "2m", "10t")
  optional int64 start_time = 6;          // Required for Bid: Contract start time
  string stake = 7;                       // Required: Premium amount
  optional string payout = 8;             // Required for Bid: Original payout
}
```

#### Field Validation Rules

| Field | Validation | Error Code |
|-------|------------|------------|
| `symbol` | Must be in: `R_10`, `R_25`, `R_50`, `R_75`, `R_100` | `ERR-DR-S1V` |
| `contract_type` | Must be `RISE` or `FALL` | `ERR-DR-C2T` |
| `currency` | Must be supported currency | `ERR-DR-C3U` |
| `first_duration` | Valid format: `\d+(s|m|h|d|t)` | `ERR-DR-D2U` |
| `second_duration` | Valid format: `\d+(s|m|h|d|t)`, must be > first_duration | `ERR-DR-D3O` |
| `stake` | Must be ≥ min_stake (1.00 USD) | `ERR-DR-K5S` |

#### Duration Format

| Unit | Format | Example | Range |
|------|--------|---------|-------|
| Seconds | `{n}s` | `30s` | 10s - 86400s |
| Minutes | `{n}m` | `5m` | 1m - 1440m |
| Hours | `{n}h` | `2h` | 1h - 24h |
| Days | `{n}d` | `1d` | 1d only |
| Ticks | `{n}t` | `5t` | 2t - 10t |

#### Duration Gap Constraints

| Type | Minimum Gap |
|------|-------------|
| Time-based | 10 seconds |
| Tick-based | 2 ticks |

### Limits (Message)

```protobuf
message Limits {
  string max_payout = 1;                  // Maximum payout limit
  string min_stake = 2;                   // Minimum stake limit
}
```

---

## Error Handling

### Standard Error Format

All errors use standard gRPC status codes with error details in the `google.rpc.Status` message.

```protobuf
// Error details included in gRPC status
message ErrorDetail {
  string code = 1;                        // Error code (e.g., "ERR-DR-S1V")
  string message = 2;                     // Human-readable message
  map<string, string> metadata = 3;       // Additional context
}
```

### Error Codes

| Error ID | gRPC Status | Code | Description | Client Action |
|----------|-------------|------|-------------|---------------|
| `ERR-DR-S1V` | `INVALID_ARGUMENT` | `INVALID_SYMBOL` | Symbol not supported | Check supported symbols list |
| `ERR-DR-D2U` | `INVALID_ARGUMENT` | `INVALID_DURATION` | Invalid duration format | Fix duration format |
| `ERR-DR-D3O` | `INVALID_ARGUMENT` | `DURATION_ORDER_INVALID` | second_duration ≤ first_duration | Ensure t2 > t1 |
| `ERR-DR-D4G` | `INVALID_ARGUMENT` | `DURATION_GAP_INVALID` | Gap < 10s (time) or < 2 ticks | Increase duration gap |
| `ERR-DR-K5S` | `INVALID_ARGUMENT` | `INVALID_STAKE` | Stake below minimum | Increase stake to ≥ 1.00 |
| `ERR-DR-P6X` | `INVALID_ARGUMENT` | `PAYOUT_EXCEEDED` | Payout exceeds maximum | Reduce stake |
| `ERR-DR-T2F` | `INVALID_ARGUMENT` | `PRICING_TIME_FUTURE` | Future pricing time not allowed | Use current or past time |
| `ERR-DR-Y7D` | `FAILED_PRECONDITION` | `SYMBOL_DISABLED` | Symbol trading disabled | Choose different symbol |
| `ERR-DR-E3T` | `FAILED_PRECONDITION` | `MISSING_ENTRY_TICK` | No tick at entry/evaluation time | Wait for market data |
| `ERR-DR-M8E` | `UNAVAILABLE` | `MARKET_DATA_ERROR` | Cannot fetch market data | Retry with exponential backoff |
| `ERR-DR-C1D` | `UNAVAILABLE` | `STREAM_DISCONNECTED` | Stream connection lost | Reconnect and resume |
| `ERR-DR-T1M` | `INVALID_ARGUMENT` | `MISSING_START_TIME` | start_time required for Bid | Provide start_time |
| `ERR-DR-P2Y` | `INVALID_ARGUMENT` | `MISSING_PAYOUT` | payout required for Bid | Provide original payout |
| `ERR-DR-C2T` | `INVALID_ARGUMENT` | `INVALID_CONTRACT_TYPE` | Invalid contract type | Use RISE or FALL |
| `ERR-DR-C3U` | `INVALID_ARGUMENT` | `INVALID_CURRENCY` | Unsupported currency | Use supported currency |
| `ERR-DR-I9N` | `INTERNAL` | `INTERNAL_ERROR` | Unexpected server error | Contact support |

### Stream Error Behavior

| Scenario | Behavior |
|----------|----------|
| Validation error | Terminate stream with error status immediately |
| Market data disconnect | Send error, attempt reconnect (3 retries), terminate if failed |
| Server shutdown | Graceful termination with `UNAVAILABLE` status |
| Client cancellation | Clean stream termination |

---

## Service Level Agreements

### Performance Requirements

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Latency (P50)** | < 5ms | Unary calls |
| **Latency (P99)** | < 10ms | Unary calls |
| **Stream Setup** | < 50ms | Time to first message |
| **Availability** | 99.9% | Monthly uptime |
| **Error Rate** | < 0.1% | Non-validation errors |

### Rate Limiting

| Consumer | Limit | Window |
|----------|-------|--------|
| `api-gateway-trading` | 10,000 requests/second | Per service instance |

### Timeouts

| Operation | Timeout |
|-----------|---------|
| Unary call | 5 seconds |
| Stream keepalive | 30 seconds |
| Connection establishment | 10 seconds |

---

## Proto Definition Reference

The complete proto definition is located at:
[`workspace/code/service-pricer-doublerisefall/api/proto/doublerisefall/v1/doublerisefall.proto`](../../../code/service-pricer-doublerisefall/api/proto/doublerisefall/v1/doublerisefall.proto)

---

## Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.1 | 2026-01-15 | AI Generated | Added missing error code `ERR-DR-T2F` (PRICING_TIME_FUTURE) per Product Brief §7.2 |
| 1.0.0 | 2026-01-15 | AI Generated | Initial API specification |

---

> **Document Version**: 1.0.1
> **Last Updated**: 2026-01-15
