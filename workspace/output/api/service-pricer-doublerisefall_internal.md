# Internal API Specification: service-pricer-doublerisefall

> **Version**: 1.0.0
> **Created**: 2026-01-15
> **API Type**: Internal
> **Status**: DRAFT

---

## Overview

This document specifies the internal gRPC API provided by `service-pricer-doublerisefall` for service-to-service communication. The API enables contract price calculation (Ask) and contract value evaluation (Bid) for the Double Rise/Fall digital binary option product.

### Purpose
Provide real-time pricing capabilities for path-dependent binary options that evaluate spot prices against a barrier at two distinct timestamps (t1 and t2).

### Scope
- Contract price calculation with commission markup
- Payout computation based on fair probability
- Contract value evaluation at expiry
- Real-time streaming of prices and values

### Target Consumers

| Consumer | Purpose | Priority |
|----------|---------|----------|
| `api-gateway-trading` | Request contract prices for client display and contract purchase | Primary |

---

## Authentication & Authorization

### Service-to-Service Authentication

| Field | Value |
|-------|-------|
| **Authentication Required** | No |
| **Method** | N/A |
| **Rationale** | Internal service consumed only by api-gateway-trading within trusted network |

### Network Security

| Aspect | Configuration |
|--------|---------------|
| **Network** | Internal Kubernetes cluster network |
| **TLS** | Optional (mTLS recommended for production) |
| **Access Control** | Kubernetes NetworkPolicy restricts access to api-gateway-trading |

---

## Base Configuration

| Field | Value |
|-------|-------|
| **Protocol** | gRPC (Protocol Buffers v3) |
| **Package** | `doublerisefall.v1` |
| **Service Name** | `DoubleRiseFallService` |
| **Default Port** | 50051 |
| **Health Check Port** | 50051 (gRPC health check protocol) |
| **Content Type** | `application/grpc` |
| **Versioning Strategy** | Package versioning (v1, v2, etc.) |

### Service Address

| Environment | Address |
|-------------|---------|
| Development | `localhost:50051` |
| Kubernetes | `service-pricer-doublerisefall.default.svc.cluster.local:50051` |

### Proto Import

```protobuf
syntax = "proto3";

package doublerisefall.v1;

option go_package = "github.com/regentmarkets/service-pricer-doublerisefall/api/";
```

---

## Table of Endpoints

| Endpoint ID | RPC Method | Pattern | Summary |
|-------------|------------|---------|---------|
| API-DF-A1K | GetAsk | Unary | Request single contract price |
| API-DF-A2S | StreamAsk | Server Streaming | Stream live contract prices |
| API-DF-B1V | GetBid | Unary | Request contract value/status |
| API-DF-B2T | StreamBid | Server Streaming | Stream live contract value |

---

## Endpoints & Methods

### API-DF-A1K: GetAsk

Request a single contract price (Ask) for the Double Rise/Fall product.

| Field | Value |
|-------|-------|
| **RPC Method** | `GetAsk` |
| **Pattern** | Unary |
| **Request Type** | `GetAskRequest` |
| **Response Type** | `GetAskResponse` |
| **Priority** | Critical |

#### Purpose

Calculate the premium (ask price) and potential payout for a Double Rise/Fall contract based on:
- Fair probability derived from the arcsin correlation formula
- Commission markup per symbol configuration
- Current spot price from market feed

#### Request Schema

```protobuf
message GetAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;      // Optional: For repricing historical
}
```

#### Response Schema

```protobuf
message GetAskResponse {
  string ask_price = 1;                 // Contract price (premium)
  string currency = 2;                  // Quote currency
  string current_spot = 3;              // Current market price
  int64 current_spot_time = 4;          // Timestamp of spot price
  string payout = 5;                    // Potential payout
  Limits limits = 6;                    // Trading limits
}
```

#### Success Response

| Status Code | Description |
|-------------|-------------|
| OK (0) | Price calculated successfully |

#### Error Responses

| gRPC Status | Error Code | Description |
|-------------|------------|-------------|
| INVALID_ARGUMENT (3) | ERR-DF-S1K | Invalid symbol |
| INVALID_ARGUMENT (3) | ERR-DF-D1N | Invalid duration format |
| INVALID_ARGUMENT (3) | ERR-DF-D2O | Duration order invalid (second ≤ first) |
| INVALID_ARGUMENT (3) | ERR-DF-D3G | Duration gap invalid (< 10s or < 2t) |
| INVALID_ARGUMENT (3) | ERR-DF-K1M | Invalid stake (below minimum) |
| INVALID_ARGUMENT (3) | ERR-DF-P1X | Payout exceeds maximum |
| INVALID_ARGUMENT (3) | ERR-DF-T1F | Pricing time in future |
| FAILED_PRECONDITION (9) | ERR-DF-S2D | Symbol disabled |
| UNAVAILABLE (14) | ERR-DF-M1E | Market data unavailable |
| INTERNAL (13) | ERR-DF-I1X | Internal server error |

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
  "current_spot_time": 1736916600,
  "payout": "30.12",
  "limits": {
    "max_payout": "1000.00",
    "min_stake": "1.00"
  }
}
```

---

### API-DF-A2S: StreamAsk

Stream live contract prices (Ask) with updates on market tick changes.

| Field | Value |
|-------|-------|
| **RPC Method** | `StreamAsk` |
| **Pattern** | Server Streaming |
| **Request Type** | `StreamAskRequest` |
| **Response Type** | `stream GetAskResponse` |
| **Priority** | High |

#### Purpose

Provide real-time price updates for a Double Rise/Fall contract as market data changes. The stream emits:
- On each tick update from market feed
- Every 5 seconds as keepalive (time-based contracts only)

#### Request Schema

```protobuf
message StreamAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}
```

#### Response Schema

Same as `GetAskResponse` (streamed repeatedly).

#### Stream Behavior

| Trigger | Time-Based Contract | Tick-Based Contract |
|---------|---------------------|---------------------|
| Market tick update | ✅ Emit response | ✅ Emit response |
| 5-second interval | ✅ Emit response | ❌ No emission |

#### Stream Termination

| Condition | Behavior |
|-----------|----------|
| Client cancellation | Graceful close |
| Server shutdown | UNAVAILABLE status |
| Market data disconnection | UNAVAILABLE status with reconnect attempt |
| Invalid request | Immediate termination with error |

#### Error Responses

Same error codes as GetAsk, plus:

| gRPC Status | Error Code | Description |
|-------------|------------|-------------|
| CANCELLED (1) | - | Client cancelled stream |
| UNAVAILABLE (14) | ERR-DF-M2R | Market data stream interrupted |

---

### API-DF-B1V: GetBid

Request the current value (Bid) of an active or expired contract.

| Field | Value |
|-------|-------|
| **RPC Method** | `GetBid` |
| **Pattern** | Unary |
| **Request Type** | `GetBidRequest` |
| **Response Type** | `GetBidResponse` |
| **Priority** | Critical |

#### Purpose

Evaluate the current value of a Double Rise/Fall contract by:
- Fetching entry tick (barrier) at start_time
- Fetching spot at evaluation times (t1, t2)
- Applying win/loss conditions (RISE: spot > barrier at BOTH t1 AND t2)
- Returning payout (win) or 0 (loss)

#### Request Schema

```protobuf
message GetBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}
```

**Required Fields for Bid:**
- `option_parameters.start_time` - Contract start time
- `option_parameters.payout` - Payout from original Ask response

#### Response Schema

```protobuf
message GetBidResponse {
  string bid_price = 1;                 // Current contract value
  bool is_expired = 2;                  // Expiry status
  string current_spot = 3;              // Current market price
  int64 current_spot_time = 4;          // Timestamp of current spot
  string entry_spot = 5;                // Entry price (barrier)
  int64 entry_spot_time = 6;            // Timestamp of entry tick
  string exit_spot = 7;                 // Exit price (if expired)
  int64 exit_spot_time = 8;             // Timestamp of exit tick
  string barrier = 9;                   // Resolved barrier value
  int64 start_time = 10;                // Contract start time
  int64 expiry_time = 11;               // Contract expiry time
  string currency = 12;                 // Quote currency
  int64 evaluation_time = 13;           // First evaluation time (t1)
}
```

#### Win/Loss Evaluation

| Contract Type | Win Condition | Result |
|---------------|---------------|--------|
| RISE | spot_t1 > barrier AND spot_t2 > barrier | bid_price = payout |
| RISE | spot_t1 ≤ barrier OR spot_t2 ≤ barrier | bid_price = 0 |
| FALL | spot_t1 < barrier AND spot_t2 < barrier | bid_price = payout |
| FALL | spot_t1 ≥ barrier OR spot_t2 ≥ barrier | bid_price = 0 |

**Critical**: If condition fails at t1, contract expires worthless immediately.

#### Success Response

| Status Code | Description |
|-------------|-------------|
| OK (0) | Bid evaluated successfully |

#### Error Responses

Same as GetAsk, plus:

| gRPC Status | Error Code | Description |
|-------------|------------|-------------|
| INVALID_ARGUMENT (3) | ERR-DF-R1S | Missing start_time (required for Bid) |
| INVALID_ARGUMENT (3) | ERR-DF-R2P | Missing payout (required for Bid) |
| FAILED_PRECONDITION (9) | ERR-DF-E1M | Missing entry tick at start_time |
| FAILED_PRECONDITION (9) | ERR-DF-E2T | Missing tick at evaluation time |

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
    "start_time": 1736916000,
    "stake": "10.00",
    "payout": "30.12"
  }
}
```

**Response (Win):**
```json
{
  "bid_price": "30.12",
  "is_expired": true,
  "current_spot": "1235.0000",
  "current_spot_time": 1736916120,
  "entry_spot": "1234.0000",
  "entry_spot_time": 1736916001,
  "exit_spot": "1235.0000",
  "exit_spot_time": 1736916120,
  "barrier": "1234.0000",
  "start_time": 1736916000,
  "expiry_time": 1736916120,
  "currency": "USD",
  "evaluation_time": 1736916060
}
```

---

### API-DF-B2T: StreamBid

Stream live contract value (Bid) with updates as contract progresses toward expiry.

| Field | Value |
|-------|-------|
| **RPC Method** | `StreamBid` |
| **Pattern** | Server Streaming |
| **Request Type** | `StreamBidRequest` |
| **Response Type** | `stream GetBidResponse` |
| **Priority** | High |

#### Purpose

Provide real-time updates on contract value and status. Stream terminates automatically when contract expires.

#### Request Schema

```protobuf
message StreamBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}
```

#### Response Schema

Same as `GetBidResponse` (streamed repeatedly).

#### Stream Behavior

| Trigger | Time-Based Contract | Tick-Based Contract |
|---------|---------------------|---------------------|
| Market tick update | ✅ Emit response | ✅ Emit response |
| 5-second interval | ✅ Emit response | ❌ No emission |
| Contract expiry | ✅ Final response + close | ✅ Final response + close |

**Critical**: Tick-based contracts have NO time-based fallback for StreamBid.

#### Stream Termination

| Condition | Behavior |
|-----------|----------|
| Contract expired | Final response with `is_expired=true`, then close |
| Client cancellation | Graceful close |
| Server shutdown | UNAVAILABLE status |
| Market data disconnection | UNAVAILABLE status |

#### Error Responses

Same error codes as GetBid.

---

## Data Models & Schemas

### OptionParameters

Core contract parameters used across all endpoints.

```protobuf
message OptionParameters {
  string symbol = 1;                    // Required: Underlying asset
  ContractType contract_type = 2;       // Required: Contract direction
  string currency = 3;                  // Required: Payout currency
  string first_duration = 4;            // Required: e.g., "1m", "30s", "5t"
  string second_duration = 5;           // Required: e.g., "2m", "60s", "10t"
  optional int64 start_time = 6;        // Required for Bid requests
  string stake = 7;                     // Required: Premium amount
  optional string payout = 8;           // Required for Bid requests (from Ask response)
}
```

| Field | Type | Required (Ask) | Required (Bid) | Description |
|-------|------|----------------|----------------|-------------|
| symbol | string | ✅ | ✅ | Underlying asset (R_10, R_25, R_50, R_75, R_100) |
| contract_type | ContractType | ✅ | ✅ | RISE or FALL |
| currency | string | ✅ | ✅ | Payout currency (e.g., USD) |
| first_duration | string | ✅ | ✅ | Duration to first evaluation (t1) |
| second_duration | string | ✅ | ✅ | Duration to second evaluation/expiry (t2) |
| start_time | int64 | ❌ | ✅ | Contract start time (Unix epoch) |
| stake | string | ✅ | ✅ | Premium amount |
| payout | string | ❌ | ✅ | Payout from original Ask response |

### ContractType

```protobuf
enum ContractType {
  CONTRACT_TYPE_UNSPECIFIED = 0;
  CONTRACT_TYPE_RISE = 1;
  CONTRACT_TYPE_FALL = 2;
}
```

| Value | Description |
|-------|-------------|
| CONTRACT_TYPE_UNSPECIFIED | Invalid/unset |
| CONTRACT_TYPE_RISE | Win if spot > barrier at both t1 and t2 |
| CONTRACT_TYPE_FALL | Win if spot < barrier at both t1 and t2 |

### Limits

Trading limits for the symbol.

```protobuf
message Limits {
  string max_payout = 1;
  string min_stake = 2;
}
```

### Duration Format

Duration strings support two formats:

| Format | Unit | Examples | Validation |
|--------|------|----------|------------|
| Time-based | seconds (s), minutes (m), hours (h), days (d) | "30s", "1m", "2h", "1d" | Max: 1 day |
| Tick-based | ticks (t) | "5t", "10t" | Min: 2t, Max: 10t |

**Validation Rules:**
- second_duration > first_duration
- Gap ≥ 10 seconds (time-based) or ≥ 2 ticks (tick-based)

### Supported Symbols

| Symbol | Commission | Max Payout | Min Stake | Enabled |
|--------|------------|------------|-----------|---------|
| R_10 | 5% (0.05) | 1000 USD | 1 USD | ✅ |
| R_25 | 5% (0.05) | 1000 USD | 1 USD | ✅ |
| R_50 | 5% (0.05) | 1000 USD | 1 USD | ✅ |
| R_75 | 5% (0.05) | 1000 USD | 1 USD | ✅ |
| R_100 | 5% (0.05) | 1000 USD | 1 USD | ✅ |

---

## Error Handling

### Standard Error Format

All errors are returned using standard gRPC status codes with detailed error information in the status message.

```go
// Error response structure
status.Errorf(codes.InvalidArgument, "ERR-DF-S1K: invalid symbol: %s", symbol)
```

### Error Code Reference

| Error Code | gRPC Status | HTTP Equiv | Description | Client Action |
|------------|-------------|------------|-------------|---------------|
| ERR-DF-S1K | INVALID_ARGUMENT | 400 | Symbol not supported | Check supported symbol list |
| ERR-DF-S2D | FAILED_PRECONDITION | 400 | Symbol trading disabled | Choose different symbol |
| ERR-DF-D1N | INVALID_ARGUMENT | 400 | Invalid duration format | Use valid format (s, m, h, d, t) |
| ERR-DF-D2O | INVALID_ARGUMENT | 400 | Duration order invalid | Ensure second > first |
| ERR-DF-D3G | INVALID_ARGUMENT | 400 | Duration gap invalid | Increase gap (≥10s or ≥2t) |
| ERR-DF-K1M | INVALID_ARGUMENT | 400 | Stake below minimum | Increase stake |
| ERR-DF-P1X | INVALID_ARGUMENT | 400 | Payout exceeds maximum | Reduce stake |
| ERR-DF-T1F | INVALID_ARGUMENT | 400 | Pricing time in future | Use current or past time |
| ERR-DF-R1S | INVALID_ARGUMENT | 400 | Missing start_time | Provide start_time for Bid |
| ERR-DF-R2P | INVALID_ARGUMENT | 400 | Missing payout | Provide payout for Bid |
| ERR-DF-E1M | FAILED_PRECONDITION | 400 | Missing entry tick | Wait for market data |
| ERR-DF-E2T | FAILED_PRECONDITION | 400 | Missing evaluation tick | Wait for market data |
| ERR-DF-M1E | UNAVAILABLE | 503 | Market data unavailable | Retry with backoff |
| ERR-DF-M2R | UNAVAILABLE | 503 | Market stream interrupted | Reconnect stream |
| ERR-DF-I1X | INTERNAL | 500 | Internal server error | Contact support |

### Error Handling Best Practices

**For INVALID_ARGUMENT errors:**
- Do not retry automatically
- Fix request parameters and retry

**For FAILED_PRECONDITION errors:**
- May retry after waiting for precondition to be met
- Check market data availability

**For UNAVAILABLE errors:**
- Retry with exponential backoff
- Maximum 3 retries recommended
- Consider circuit breaker pattern

**For INTERNAL errors:**
- Log error details
- Alert operations team
- May retry with caution

---

## Service Discovery

### Kubernetes Service Configuration

```yaml
apiVersion: v1
kind: Service
metadata:
  name: service-pricer-doublerisefall
  namespace: default
  labels:
    app: service-pricer-doublerisefall
spec:
  type: ClusterIP
  ports:
    - name: grpc
      port: 50051
      targetPort: 50051
      protocol: TCP
  selector:
    app: service-pricer-doublerisefall
```

### Health Check

The service implements the standard gRPC health check protocol:

```protobuf
service Health {
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);
  rpc Watch(HealthCheckRequest) returns (stream HealthCheckResponse);
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC server port | 50051 |
| `FEED_SERVICE_ADDR` | service-feed address | service-feed:50051 |
| `CONFIG_PATH` | Path to symbol config | /config/symbols.yaml |
| `LOG_LEVEL` | Logging level | info |

### Connection Configuration

**Recommended client configuration:**

```go
conn, err := grpc.Dial(
    "service-pricer-doublerisefall:50051",
    grpc.WithInsecure(), // or grpc.WithTransportCredentials(creds)
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                10 * time.Second,
        Timeout:             3 * time.Second,
        PermitWithoutStream: true,
    }),
)
```

---

## SLA & Reliability

### Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| GetAsk latency (p50) | < 20ms | Excluding network |
| GetAsk latency (p99) | < 50ms | Excluding network |
| StreamAsk update latency | < 100ms | From tick to client |
| GetBid latency (p99) | < 100ms | May require multiple feed calls |
| Concurrent streams | 1000 | Per instance |

### Availability

| Metric | Target |
|--------|--------|
| Uptime | 99.9% |
| Recovery time | < 30s |

### Capacity

| Metric | Capacity |
|--------|----------|
| Requests per second | 10,000 |
| Active streams | 1,000 per instance |
| Memory usage | < 500MB |

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-15 | Initial internal API specification |

---

## Quality Checklist

- [x] All required capabilities from Inter-Service Communication Matrix addressed
- [x] Authentication/authorization clearly defined (internal, no auth required)
- [x] Base configuration complete (gRPC, package, port)
- [x] Endpoint table provides quick reference
- [x] Each endpoint has complete documentation
- [x] Data models are comprehensive (OptionParameters, ContractType, Limits)
- [x] Error handling standardized with unique error codes
- [x] Service discovery documented (Kubernetes, health check)
- [x] SLAs and reliability guarantees documented
- [x] API follows consistent design principles
- [x] No functionality is duplicated
- [x] Service boundaries respected
