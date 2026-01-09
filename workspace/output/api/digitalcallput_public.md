# Public API Specification
# Digital Call/Put Options Pricing Service

**Version**: 1.0  
**Date**: 2026-01-08  
**Status**: Draft  
**API Type**: Public (External Consumers)

---

## 1. Overview

The Digital Call/Put Options Pricing Service exposes a gRPC API for real-time pricing of digital (binary) options contracts. The API provides both unary (single request/response) and streaming endpoints for calculating Ask prices (contract purchase) and Bid prices (contract valuation).

### Target Consumers
- Trading platforms
- Mobile trading applications
- Third-party integrations

### API Protocol
- **Protocol**: gRPC
- **Serialization**: Protocol Buffers (proto3)
- **Transport**: HTTP/2

---

## 2. Authentication & Authorization

Authentication is handled by upstream services before requests reach this pricing service. The service itself does not implement authentication.

**Security Model**:
- Requests are expected from authenticated internal services
- No API keys or tokens required at this service level
- Rate limiting enforced by upstream API gateway

---

## 3. Base Configuration

| Configuration | Value |
|---------------|-------|
| Protocol | gRPC |
| Default Port | 50051 |
| Proto Package | `digitalcallput.v1` |
| Health Check | `/grpc.health.v1.Health/Check` |
| Content Type | `application/grpc` |

### Versioning Strategy
- Version embedded in proto package (`v1`, `v2`, etc.)
- Breaking changes require new major version
- Backward-compatible changes within same version

---

## 4. Table of Endpoints

| ID | RPC Type | Method | Summary |
|----|----------|--------|---------|
| API-DC-G1A | Unary | GetAsk | Calculate Ask price for proposed contract |
| API-DC-S2B | Server Stream | StreamAsk | Stream Ask prices as market changes |
| API-DC-G3C | Unary | GetBid | Get current market value of active contract |
| API-DC-S4D | Server Stream | StreamBid | Stream Bid prices for active contract |

---

## 5. Endpoints & Methods

### 5.1 GetAsk (API-DC-G1A)

**RPC Type**: Unary  
**Purpose**: Returns a single Ask price for a proposed digital option contract.

**Proto Definition**:
```protobuf
rpc GetAsk(GetAskRequest) returns (GetAskResponse);
```

**Request Schema**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| symbol | string | Yes | Underlying asset (e.g., "USD/JPY", "BTC/USD") |
| contract_type | ContractType | Yes | CALL or PUT |
| currency | string | Yes | Payout currency (e.g., "USD", "EUR") |
| duration | string | Yes | Duration string (e.g., "1m", "30s", "5t") |
| stake | string | Yes | Premium amount (decimal string) |
| barrier | string | No | Barrier value: absolute, relative (+/-), or omit for ATM |
| pricing_time | int64 | No | Epoch timestamp for pricing (default: now) |

**Response Schema**:
| Field | Type | Description |
|-------|------|-------------|
| ask_price | string | Calculated ask price (8 decimal precision) |
| currency | string | Contract currency |
| current_spot | string | Current spot price from market |
| current_spot_time | int64 | Spot price timestamp (epoch) |
| payout | string | Potential payout amount |
| limits | Limits | Trading limits (max_payout, min_stake) |

**Success Response**: gRPC status `OK` with GetAskResponse

**Error Responses**:
| gRPC Code | Error ID | Condition |
|-----------|----------|-----------|
| INVALID_ARGUMENT | ERR-DC-A1A | Missing required field |
| INVALID_ARGUMENT | ERR-DC-A2B | Invalid duration format |
| INVALID_ARGUMENT | ERR-DC-A3C | Stake below min_stake |
| INVALID_ARGUMENT | ERR-DC-A4D | Calculated payout exceeds max_payout |
| INVALID_ARGUMENT | ERR-DC-A5E | Invalid barrier format |
| NOT_FOUND | ERR-DC-A6F | Symbol not in configuration |
| UNAVAILABLE | ERR-DC-A7G | Market data service unavailable |

**Example Request**:
```json
{
  "symbol": "USD/JPY",
  "contract_type": "CALL",
  "currency": "USD",
  "duration": "1m",
  "stake": "100.00",
  "barrier": "+0.0023"
}
```

**Example Response**:
```json
{
  "ask_price": "45.23456789",
  "currency": "USD",
  "current_spot": "149.87654321",
  "current_spot_time": 1736326800,
  "payout": "210.50000000",
  "limits": {
    "max_payout": "50000.00000000",
    "min_stake": "1.00000000"
  }
}
```

**PRD Reference**: REQ-AP-G1A

---

### 5.2 StreamAsk (API-DC-S2B)

**RPC Type**: Server Streaming  
**Purpose**: Streams Ask prices continuously as market conditions change.

**Proto Definition**:
```protobuf
rpc StreamAsk(GetAskRequest) returns (stream GetAskResponse);
```

**Request Schema**: Same as GetAsk

**Response Stream**: Continuous GetAskResponse messages

**Stream Behavior**:
| Duration Type | Update Trigger |
|---------------|----------------|
| Time-based (s/m/h/d) | On new tick OR every 5 seconds |
| Tick-based (t) | ONLY on new ticks |

**Stream Termination**:
- Client disconnects
- Error condition occurs
- Server-side timeout (configurable)

**Error Responses**: Same as GetAsk

**PRD Reference**: REQ-AP-S2B, REQ-ST-U1P, REQ-ST-T2Q

---

### 5.3 GetBid (API-DC-G3C)

**RPC Type**: Unary  
**Purpose**: Returns the current market value of an active digital option contract.

**Proto Definition**:
```protobuf
rpc GetBid(GetBidRequest) returns (GetBidResponse);
```

**Request Schema**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| symbol | string | Yes | Underlying asset |
| contract_type | ContractType | Yes | CALL or PUT |
| currency | string | Yes | Contract currency |
| duration | string | Yes | Original duration string |
| barrier | string | No | Barrier value (resolved if relative) |
| start_time | int64 | Yes | Contract start timestamp (epoch) |
| payout | string | Yes | Fixed payout from purchase (IMMUTABLE) |

**Response Schema**:
| Field | Type | Description |
|-------|------|-------------|
| bid_price | string | Current market value (8 decimal precision) |
| is_expired | bool | Contract expiration status |
| current_spot | string | Current spot price |
| current_spot_time | int64 | Current spot timestamp |
| entry_spot | string | Entry tick price |
| entry_spot_time | int64 | Entry tick timestamp (market feed time) |
| exit_spot | string | Exit tick price (only when expired) |
| exit_spot_time | int64 | Exit tick timestamp (only when expired) |
| barrier | string | Resolved barrier value |
| start_time | int64 | Contract start time |
| expiry_time | int64 | Contract expiry time |
| currency | string | Contract currency |

**Critical Business Rule**: The `payout` parameter is MANDATORY and used as-is. The service NEVER recalculates payout - it was fixed at purchase time.

**Success Response**: gRPC status `OK` with GetBidResponse

**Error Responses**:
| gRPC Code | Error ID | Condition |
|-----------|----------|-----------|
| INVALID_ARGUMENT | ERR-DC-B1A | Missing required field |
| INVALID_ARGUMENT | ERR-DC-B2B | Missing payout parameter |
| INVALID_ARGUMENT | ERR-DC-B3C | Invalid start_time (future or invalid) |
| INVALID_ARGUMENT | ERR-DC-B4D | Invalid duration format |
| NOT_FOUND | ERR-DC-B5E | Symbol not in configuration |
| UNAVAILABLE | ERR-DC-B6F | Market data service unavailable |

**Example Request**:
```json
{
  "symbol": "USD/JPY",
  "contract_type": "CALL",
  "currency": "USD",
  "duration": "1m",
  "barrier": "+0.0023",
  "start_time": 1736326740,
  "payout": "210.50000000"
}
```

**Example Response (Active Contract)**:
```json
{
  "bid_price": "125.34567890",
  "is_expired": false,
  "current_spot": "149.89012345",
  "current_spot_time": 1736326780,
  "entry_spot": "149.87654321",
  "entry_spot_time": 1736326741,
  "barrier": "149.87884321",
  "start_time": 1736326740,
  "expiry_time": 1736326800,
  "currency": "USD"
}
```

**Example Response (Expired Contract - Win)**:
```json
{
  "bid_price": "210.50000000",
  "is_expired": true,
  "current_spot": "149.92000000",
  "current_spot_time": 1736326801,
  "entry_spot": "149.87654321",
  "entry_spot_time": 1736326741,
  "exit_spot": "149.92000000",
  "exit_spot_time": 1736326800,
  "barrier": "149.87884321",
  "start_time": 1736326740,
  "expiry_time": 1736326800,
  "currency": "USD"
}
```

**PRD Reference**: REQ-AP-G3C, REQ-PR-B2K, REQ-LC-P3O

---

### 5.4 StreamBid (API-DC-S4D)

**RPC Type**: Server Streaming  
**Purpose**: Streams Bid prices for active contracts until expiry.

**Proto Definition**:
```protobuf
rpc StreamBid(GetBidRequest) returns (stream GetBidResponse);
```

**Request Schema**: Same as GetBid

**Response Stream**: Continuous GetBidResponse messages

**Stream Behavior**:
| Duration Type | Update Trigger |
|---------------|----------------|
| Time-based (s/m/h/d) | On new tick OR every 5 seconds |
| Tick-based (t) | ONLY on new ticks (no time fallback) |

**Stream Termination**:
- Contract expires (final message includes exit data with `is_expired: true`)
- Client disconnects
- Error condition occurs

**Tick-Based Contract Note**: Per REQ-PR-N3L, tick-based contracts do not support early exit. The `bid_price` for active tick-based contracts will be 0 or indicate non-tradeable status.

**Error Responses**: Same as GetBid

**PRD Reference**: REQ-AP-S4D, REQ-ST-U1P, REQ-ST-T2Q, REQ-PR-N3L

---

## 6. Data Models & Schemas

### 6.1 Enumerations

#### ContractType
```protobuf
enum ContractType {
  CONTRACT_TYPE_UNSPECIFIED = 0;
  CONTRACT_TYPE_CALL = 1;
  CONTRACT_TYPE_PUT = 2;
}
```

**Validation**: Requests with `CONTRACT_TYPE_UNSPECIFIED` return INVALID_ARGUMENT.

### 6.2 Message Types

#### Limits
```protobuf
message Limits {
  string max_payout = 1;  // Maximum allowed payout
  string min_stake = 2;   // Minimum required stake
}
```

### 6.3 Duration Format

**Time-Based Durations**:
| Unit | Format | Example | Range |
|------|--------|---------|-------|
| Seconds | `[n]s` | "30s" | 1s - 86400s |
| Minutes | `[n]m` | "5m" | 1m - 1440m |
| Hours | `[n]h` | "2h" | 1h - 24h |
| Days | `[n]d` | "7d" | 1d - 365d |

**Tick-Based Durations**:
| Unit | Format | Example | Range |
|------|--------|---------|-------|
| Ticks | `[n]t` | "5t" | 1t - 10t |

### 6.4 Barrier Format

| Type | Format | Example | Resolution |
|------|--------|---------|------------|
| Absolute | Numeric string | "149.50" | Used as-is |
| Relative Positive | `+[offset]` | "+0.0023" | entry_price + offset |
| Relative Negative | `-[offset]` | "-0.0023" | entry_price - offset |
| Default (ATM) | Omit parameter | - | entry_price |

### 6.5 Monetary Values

All monetary values (prices, stake, payout, barrier) are represented as strings with 8 decimal place precision to avoid floating-point errors.

**Format**: `"[integer].[8_decimals]"`  
**Examples**: `"100.00000000"`, `"149.87654321"`, `"0.00000001"`

### 6.6 Timestamps

All timestamps are Unix epoch time in seconds (int64).

---

## 7. Error Handling

### 7.1 Standard Error Format

All errors follow gRPC standard status codes with detailed messages.

**Error Response Structure**:
```protobuf
// gRPC Status
status {
  code: [gRPC_CODE]
  message: "[Human-readable error message]"
  details: [
    {
      "@type": "type.googleapis.com/google.rpc.ErrorInfo"
      "reason": "[ERROR_ID]"
      "domain": "digitalcallput.v1"
      "metadata": {
        "field": "[field_name]",
        "value": "[provided_value]",
        "constraint": "[validation_rule]"
      }
    }
  ]
}
```

### 7.2 Error Code Reference

| Error ID | gRPC Code | Description | Resolution |
|----------|-----------|-------------|------------|
| ERR-DC-A1A | INVALID_ARGUMENT | Missing required field | Include all required fields |
| ERR-DC-A2B | INVALID_ARGUMENT | Invalid duration format | Use format: [n][s\|m\|h\|d\|t] |
| ERR-DC-A3C | INVALID_ARGUMENT | Stake below minimum | Increase stake to min_stake |
| ERR-DC-A4D | INVALID_ARGUMENT | Payout exceeds maximum | Reduce stake |
| ERR-DC-A5E | INVALID_ARGUMENT | Invalid barrier format | Use absolute or +/- relative |
| ERR-DC-A6F | NOT_FOUND | Unknown symbol | Use configured symbol |
| ERR-DC-A7G | UNAVAILABLE | Market data unavailable | Retry later |
| ERR-DC-B1A | INVALID_ARGUMENT | Missing required field (Bid) | Include all required fields |
| ERR-DC-B2B | INVALID_ARGUMENT | Missing payout (Bid) | Payout is required for Bid |
| ERR-DC-B3C | INVALID_ARGUMENT | Invalid start_time | Use valid past timestamp |
| ERR-DC-B4D | INVALID_ARGUMENT | Invalid duration format | Use format: [n][s\|m\|h\|d\|t] |
| ERR-DC-B5E | NOT_FOUND | Unknown symbol | Use configured symbol |
| ERR-DC-B6F | UNAVAILABLE | Market data unavailable | Retry later |
| ERR-DC-I1A | INTERNAL | Internal calculation error | Contact support |

### 7.3 Error Message Guidelines

Error messages include:
- Which field caused the error
- The invalid value (if applicable)
- The expected format or constraint
- Suggested resolution

**Example**:
```
"Duration '15t' exceeds maximum tick count. Valid range: 1t-10t"
```

---

## 8. Integration Guide

### 8.1 Getting Started

#### 1. Proto File Location
```
proto/digitalcallput/v1/digitalcallput.proto
```

#### 2. Generate Client Code
```bash
# Go
protoc --go_out=. --go-grpc_out=. proto/digitalcallput/v1/digitalcallput.proto

# Node.js
npx grpc_tools_node_protoc \
  --js_out=import_style=commonjs,binary:./generated \
  --grpc_out=grpc_js:./generated \
  proto/digitalcallput/v1/digitalcallput.proto
```

#### 3. Connect to Service
```go
// Go example
conn, err := grpc.Dial("pricing-service:50051", grpc.WithInsecure())
client := digitalcallputv1.NewDigitalCallPutServiceClient(conn)
```

### 8.2 Best Practices

#### Unary Requests (GetAsk, GetBid)
- Use for single, one-time price queries
- Implement proper timeout handling (recommended: 5s)
- Handle all error codes appropriately

#### Streaming Requests (StreamAsk, StreamBid)
- Use for real-time price monitoring
- Implement reconnection logic for disconnects
- Process messages asynchronously
- For StreamBid, handle `is_expired: true` as stream end signal

#### Monetary Values
- Parse all monetary strings as decimal types (not float)
- Maintain 8 decimal precision throughout calculations
- Never perform floating-point arithmetic on prices

#### Barrier Handling
- For new contracts: Use relative barriers (+/-) for strike relative to entry
- For expired contracts: Barrier is already resolved to absolute value

### 8.3 Sample Client Code

**Go - GetAsk Request**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.GetAsk(ctx, &digitalcallputv1.GetAskRequest{
    Symbol:       "USD/JPY",
    ContractType: digitalcallputv1.ContractType_CONTRACT_TYPE_CALL,
    Currency:     "USD",
    Duration:     "1m",
    Stake:        "100.00000000",
    Barrier:      "+0.0023",
})
if err != nil {
    st, _ := status.FromError(err)
    log.Printf("Error: %s - %s", st.Code(), st.Message())
    return
}
log.Printf("Ask Price: %s, Payout: %s", resp.AskPrice, resp.Payout)
```

**Go - StreamBid Request**:
```go
stream, err := client.StreamBid(context.Background(), &digitalcallputv1.GetBidRequest{
    Symbol:       "USD/JPY",
    ContractType: digitalcallputv1.ContractType_CONTRACT_TYPE_CALL,
    Currency:     "USD",
    Duration:     "1m",
    StartTime:    startTime,
    Payout:       "210.50000000",
})
if err != nil {
    log.Fatal(err)
}

for {
    resp, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Printf("Stream error: %v", err)
        break
    }
    
    log.Printf("Bid: %s, Expired: %v", resp.BidPrice, resp.IsExpired)
    
    if resp.IsExpired {
        log.Printf("Contract expired. Exit: %s at %d", resp.ExitSpot, resp.ExitSpotTime)
        break
    }
}
```

---

## 9. Performance & Limits

### 9.1 Response Time SLAs

| Endpoint | Target | Measurement |
|----------|--------|-------------|
| GetAsk | < 100ms p95 | End-to-end |
| GetBid | < 100ms p95 | End-to-end |
| StreamAsk (first response) | < 500ms | From request |
| StreamBid (first response) | < 500ms | From request |
| Stream update latency | < 200ms | From tick receipt |

### 9.2 Throughput Limits

| Metric | Limit |
|--------|-------|
| Concurrent streams per client | 100 |
| Requests per second per client | 500 RPS |
| Total concurrent streams | 10,000 per instance |
| Total RPS | 5,000 per instance |

### 9.3 Validation Limits

| Parameter | Constraint |
|-----------|------------|
| Time-based duration | 1 second - 365 days |
| Tick-based duration | 1 - 10 ticks |
| Stake | >= min_stake (symbol config) |
| Payout | <= max_payout (symbol config) |

---

## 10. Full Proto Definition

```protobuf
syntax = "proto3";

package digitalcallput.v1;

option go_package = "github.com/regentmarkets/service-pricer-digitalcallput/proto/digitalcallput/v1;digitalcallputv1";

// Digital Call/Put Options Pricing Service
service DigitalCallPutService {
  // GetAsk returns a single Ask price for a proposed contract
  rpc GetAsk(GetAskRequest) returns (GetAskResponse);
  
  // StreamAsk streams Ask prices as market conditions change
  rpc StreamAsk(GetAskRequest) returns (stream GetAskResponse);
  
  // GetBid returns current market value of an active contract
  rpc GetBid(GetBidRequest) returns (GetBidResponse);
  
  // StreamBid streams Bid prices for active contracts
  rpc StreamBid(GetBidRequest) returns (stream GetBidResponse);
}

enum ContractType {
  CONTRACT_TYPE_UNSPECIFIED = 0;
  CONTRACT_TYPE_CALL = 1;
  CONTRACT_TYPE_PUT = 2;
}

message GetAskRequest {
  string symbol = 1;
  ContractType contract_type = 2;
  string currency = 3;
  string duration = 4;
  string stake = 5;
  optional string barrier = 6;
  optional int64 pricing_time = 7;
}

message GetAskResponse {
  string ask_price = 1;
  string currency = 2;
  string current_spot = 3;
  int64 current_spot_time = 4;
  string payout = 5;
  Limits limits = 6;
}

message GetBidRequest {
  string symbol = 1;
  ContractType contract_type = 2;
  string currency = 3;
  string duration = 4;
  optional string barrier = 5;
  int64 start_time = 6;
  string payout = 7;
}

message GetBidResponse {
  string bid_price = 1;
  bool is_expired = 2;
  string current_spot = 3;
  int64 current_spot_time = 4;
  string entry_spot = 5;
  int64 entry_spot_time = 6;
  optional string exit_spot = 7;
  optional int64 exit_spot_time = 8;
  string barrier = 9;
  int64 start_time = 10;
  int64 expiry_time = 11;
  string currency = 12;
}

message Limits {
  string max_payout = 1;
  string min_stake = 2;
}
```

---

## 11. Requirements Traceability

| API Endpoint | PRD Requirements | User Stories |
|--------------|------------------|--------------|
| GetAsk | REQ-AP-G1A, REQ-PR-A1J | US-PR-K3M, US-PR-P7R, US-PR-A1E, US-PR-B2F, US-PR-C3G |
| StreamAsk | REQ-AP-S2B, REQ-ST-U1P, REQ-ST-T2Q | US-PR-D4H, US-PR-E5I, US-PR-F6J |
| GetBid | REQ-AP-G3C, REQ-PR-B2K, REQ-PR-N3L, REQ-LC-P3O | US-PR-G7K, US-PR-H8L, US-PR-I9M, US-PR-J1N, US-PR-L2O |
| StreamBid | REQ-AP-S4D, REQ-ST-U1P, REQ-ST-T2Q | US-PR-M3P, US-PR-N4Q, US-PR-O5R, US-PR-Q6S |

**Cross-cutting Requirements**:
- All endpoints: REQ-BR-A1E, REQ-BR-R2F, REQ-BR-N3G (barriers)
- All endpoints: REQ-DU-T1H, REQ-DU-K2I (durations)
- All endpoints: REQ-LC-E1M, REQ-LC-X2N (tick handling)
- All endpoints: NFR-PR-D1E (8 decimal precision)

---

## 12. Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-08 | API Architect | Initial API specification |
