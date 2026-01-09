# API Phase Preferences

**Document Version**: 1.0  
**Date**: 2026-01-08  
**Phase**: API Specification

---

## General API Standards

### Entry 1: API Protocol
**Type**: Directive (from workspace preferences)  
**Date**: 2026-01-08

**Decision**: Use gRPC for all API endpoints

**Rationale**:
- Native streaming support for real-time price updates
- Efficient binary protocol for high-throughput scenarios
- Strong typing with Protocol Buffers
- Consistent with service-feed dependency

### Entry 2: Versioning Strategy
**Type**: Decision  
**Date**: 2026-01-08

**Decision**: Version embedded in proto package (`v1`, `v2`, etc.)

**Rules**:
- Breaking changes require new major version
- Backward-compatible changes within same version
- Package format: `{service}.v{n}` (e.g., `digitalcallput.v1`)

---

## Endpoint Design Patterns

### Entry 3: RPC Naming Convention
**Type**: Decision  
**Date**: 2026-01-08

**Pattern**: `{Action}{Resource}` or `Stream{Action}{Resource}`

**Examples**:
- `GetAsk` - Unary request for Ask price
- `StreamAsk` - Server streaming for Ask price
- `GetBid` - Unary request for Bid price
- `StreamBid` - Server streaming for Bid price

### Entry 4: Endpoint ID Format
**Type**: Decision  
**Date**: 2026-01-08

**Format**: `API-{SERVICE}-{REF}` where:
- SERVICE: Two-letter service code (DC for digitalcallput)
- REF: Reference from PRD (e.g., G1A for GetAsk from REQ-AP-G1A)

**Examples**: API-DC-G1A, API-DC-S2B, API-DC-G3C, API-DC-S4D

### Entry 5: Unary vs Streaming
**Type**: Decision  
**Date**: 2026-01-08

**Guidelines**:
- Use **Unary RPC** for single, one-time price queries
- Use **Server Streaming RPC** for real-time price monitoring
- Client streaming and bidirectional streaming not used in this service

---

## Data Model Strategy

### Entry 6: Monetary Value Representation
**Type**: Decision  
**Date**: 2026-01-08

**Format**: String with 8 decimal precision

**Rationale**:
- Avoids floating-point precision errors
- Consistent with financial calculation requirements (NFR-PR-D1E)
- Easy parsing across language clients

**Examples**: `"100.00000000"`, `"149.87654321"`

### Entry 7: Timestamp Format
**Type**: Decision  
**Date**: 2026-01-08

**Format**: Unix epoch time in seconds (int64)

**Rationale**:
- Unambiguous timezone handling
- Efficient storage and transmission
- Consistent with service-feed timestamps

### Entry 8: Optional Fields
**Type**: Decision  
**Date**: 2026-01-08

**Usage**: Use `optional` keyword for fields that may not be present

**Examples**:
- `barrier` in requests (optional for ATM)
- `pricing_time` in GetAsk (optional, defaults to now)
- `exit_spot` in GetBid response (only when expired)

### Entry 9: Enum Design
**Type**: Decision  
**Date**: 2026-01-08

**Rules**:
- Always include `UNSPECIFIED = 0` as first value
- Prefix enum values with enum name (e.g., `CONTRACT_TYPE_CALL`)
- Reject requests with UNSPECIFIED values

---

## Error Handling

### Entry 10: Error Code Strategy
**Type**: Decision  
**Date**: 2026-01-08

**Format**: `ERR-{SERVICE}-{CATEGORY}{SEQ}`
- SERVICE: Two-letter service code (DC)
- CATEGORY: A for Ask errors, B for Bid errors, I for Internal
- SEQ: Sequential identifier (1A, 2B, etc.)

**Examples**: ERR-DC-A1A, ERR-DC-B2B, ERR-DC-I1A

### Entry 11: gRPC Status Codes
**Type**: Decision  
**Date**: 2026-01-08

**Mapping**:
| Condition | gRPC Code |
|-----------|-----------|
| Missing/invalid field | INVALID_ARGUMENT |
| Unknown symbol | NOT_FOUND |
| Market data unavailable | UNAVAILABLE |
| Internal calculation error | INTERNAL |

### Entry 12: Error Message Structure
**Type**: Decision  
**Date**: 2026-01-08

**Requirements**:
- Include field name that caused error
- Include invalid value (if applicable)
- Include expected format or constraint
- Include suggested resolution

**Example**:
```
"Duration '15t' exceeds maximum tick count. Valid range: 1t-10t"
```

---

## Authentication Methods

### Entry 13: Service-Level Authentication
**Type**: Decision  
**Date**: 2026-01-08

**Approach**: None at this service level

**Rationale**:
- Authentication handled by upstream API gateway
- This service receives pre-authenticated requests
- Simplifies service implementation

**Security Model**:
- Rate limiting at gateway level
- Service-to-service trust within cluster
- No API keys or tokens at pricing service

---

## Stream Behavior

### Entry 14: Update Patterns
**Type**: Directive (from PRD)  
**Date**: 2026-01-08

| Duration Type | Update Trigger |
|---------------|----------------|
| Time-based (s/m/h/d) | On new tick OR every 5 seconds |
| Tick-based (t) | ONLY on new ticks |

### Entry 15: Stream Termination
**Type**: Decision  
**Date**: 2026-01-08

**Termination Conditions**:
1. Client disconnect
2. Error condition
3. StreamBid: Contract expiry (`is_expired: true`)
4. Server-side timeout (configurable)

---

## Service-Specific Preferences

### [digitalcallput] Public API: Proto Package
**Type**: Decision  
**Date**: 2026-01-08

**Package**: `digitalcallput.v1`  
**Go Package**: `github.com/regentmarkets/service-pricer-digitalcallput/proto/digitalcallput/v1;digitalcallputv1`

### [digitalcallput] Public API: Service Name
**Type**: Decision  
**Date**: 2026-01-08

**Service**: `DigitalCallPutService`

### [digitalcallput] Public API: Critical Business Rules
**Type**: Directive (from PRD)  
**Date**: 2026-01-08

1. **Payout Immutability**: Payout is fixed at purchase. Bid requests MUST include payout parameter; service never recalculates.

2. **Win/Loss Determination**:
   - Call: exit > barrier = win
   - Put: exit < barrier = win
   - Equality goes to the house

3. **Tick-Based Early Exit**: Tick-based contracts do not support early exit. bid_price = 0 for active tick-based contracts.

---

## Open Items

### Entry 16: Client Library Generation
**Type**: Pending Decision  
**Date**: 2026-01-08

**Question**: Should official client libraries be generated and published?

**Options**:
1. Provide proto files only, clients generate their own
2. Publish Go client library
3. Publish multi-language clients (Go, Node.js, Python)

**Current**: Option 1 (proto files only)

---

## Summary

Key API design decisions for Digital Call/Put Options Pricing Service:

1. **Protocol**: gRPC with Protocol Buffers
2. **Versioning**: Package-level versioning (v1, v2)
3. **Naming**: `{Action}{Resource}` pattern
4. **Data**: String monetary values with 8 decimal precision
5. **Timestamps**: Unix epoch seconds (int64)
6. **Errors**: gRPC standard codes with structured messages
7. **Streaming**: Server streaming for real-time updates
8. **Auth**: None at service level (gateway responsibility)
