# Architecture Phase Preferences

**Document Version**: 1.0  
**Date**: 2026-01-08  
**Phase**: Service Architecture

---

## Service Boundaries

### Entry 1: Single Service Architecture
**Type**: Decision  
**Date**: 2026-01-08

**Decision**: The Digital Call/Put Options Pricing Service is designed as a single microservice with internal module boundaries, not as multiple separate microservices.

**Rationale**:
- Stateless service with no persistent data - no need for separate data services
- All functionality serves a single business capability (options pricing)
- Single data flow pattern for all requests
- Single external dependency (service-feed)
- Simpler deployment and operational overhead

**Impact**: 
- Domain boundaries (Pricing, Contract, Market, Configuration) become internal modules
- Single deployment unit
- Shared configuration

### Entry 2: Internal Module Boundaries
**Type**: Decision  
**Date**: 2026-01-08

**Decision**: Internal modules align with domain model bounded contexts:
- **pricing**: Core domain - Black-Scholes calculations
- **contract**: Barrier resolution, duration parsing
- **market**: service-feed integration, tick handling
- **config**: YAML symbol configuration

**Rationale**: Domain-driven design principles applied at module level

---

## Communication Patterns

### Entry 3: gRPC for External API
**Type**: Directive (from workspace preferences)  
**Date**: 2026-01-08

**Decision**: Use gRPC for all external API endpoints

**Rationale**:
- Native streaming support for real-time price updates
- Efficient binary protocol for high-throughput scenarios
- Strong typing with Protocol Buffers
- Consistent with service-feed protocol

### Entry 4: Synchronous Communication with service-feed
**Type**: Decision  
**Date**: 2026-01-08

**Decision**: Use synchronous gRPC calls for unary requests and gRPC streaming for continuous updates

**Required Endpoints from service-feed**:
- `GetLatestTick`: Single tick retrieval (unary pricing)
- `StreamTicks`: Continuous tick stream (streaming pricing)
- `GetTicks`: Historical tick retrieval (entry/exit determination)

**Note**: Per workspace preferences, these endpoints must be verified against actual service-feed proto file before implementation.

### Entry 5: Stream Update Patterns
**Type**: Directive (from PRD)  
**Date**: 2026-01-08

**Patterns**:
- Time-based contracts: Update on tick OR every 5 seconds
- Tick-based contracts: Update ONLY on new ticks (no time fallback)

---

## Data Strategy

### Entry 6: Stateless Architecture
**Type**: Directive (from workspace preferences)  
**Date**: 2026-01-08

**Decision**: Service maintains no persistent state

**Implications**:
- No database access
- All contract state reconstructed from request parameters
- Stream state maintained only for connection duration
- Symbol configuration loaded from YAML at startup

### Entry 7: Data Ownership
**Type**: Decision  
**Date**: 2026-01-08

| Data | Ownership | Source |
|------|-----------|--------|
| Contract parameters | Transient | Request |
| Pricing results | Computed | Calculation |
| Symbol configuration | Static | YAML file |
| Tick data | External | service-feed |

### Entry 8: Payout Immutability
**Type**: Directive (from PRD)  
**Date**: 2026-01-08

**Rule**: Payout is fixed at purchase time and MUST NOT be recalculated.
- Bid requests MUST include payout parameter
- Service uses provided payout, never recalculates

---

## Technology Preferences

### Entry 9: Language and Framework
**Type**: Directive (from workspace preferences)  
**Date**: 2026-01-08

**Stack**:
- Language: Go
- API: gRPC
- Serialization: Protocol Buffers
- Configuration: YAML

### Entry 10: Project Template
**Type**: Directive (from workspace preferences)  
**Date**: 2026-01-08

**Source**: https://github.com/junbon-deriv/go-templates

**Setup**:
```bash
cd workspace/code
git clone git@github.com:junbon-deriv/go-templates.git
cd go-templates && go install && cd ../
git clone git@github.com:regentmarkets/service-pricer-digitalcallput.git
go-templates --template service --module-path github.com/regentmarkets/service-pricer-digitalcallput --module-name digitalcallput
```

### Entry 11: Architecture Rules
**Type**: Directive (from workspace preferences)  
**Date**: 2026-01-08

**Mandatory Rules**:
1. gRPC handlers delegate to higher-level modules - no data fetching/assembly
2. Always return gRPC standard error codes
3. No models/types package - dependencies flow toward pricing core
4. Interfaces defined where consumed, not implemented

### Entry 12: Internal Structure Complexity
**Type**: Decision (based on PRD complexity score)  
**Date**: 2026-01-08

**PRD Complexity Score**: 6/10 (Standard)

**Structure Approach**: Light modular organization
- Separate files for API handlers, business logic, data access
- Simple separation of concerns
- Logical component boundaries without over-engineering

---

## Orchestration

### Entry 13: Port Allocation
**Type**: Decision  
**Date**: 2026-01-08

| Service | Port |
|---------|------|
| digitalcallput | 50051 |
| service-feed (external) | 50052 |

### Entry 14: Health Checking
**Type**: Decision  
**Date**: 2026-01-08

**Protocol**: gRPC Health Checking Protocol
**Endpoint**: `/grpc.health.v1.Health/Check`

### Entry 15: Environment Configuration
**Type**: Decision  
**Date**: 2026-01-08

| Variable | Description | Default |
|----------|-------------|---------|
| GRPC_PORT | gRPC server port | 50051 |
| FEED_HOST | service-feed hostname | localhost |
| FEED_PORT | service-feed port | 50052 |
| CONFIG_PATH | Symbol config path | ./config/symbols.yaml |
| LOG_LEVEL | Logging verbosity | info |

---

## Open Items

### Entry 16: service-feed API Verification
**Type**: Pending Action  
**Date**: 2026-01-08

**Action Required**: Before implementation, clone service-feed repository and verify actual API endpoints:
```bash
cd workspace/code
git clone git@github.com:junbon-deriv/service-feed.git
# Verify proto/grpcfeed/v1/ticks.proto
# Check client/client.go for usage patterns
```

**Impact**: Architecture assumes GetLatestTick, StreamTicks, GetTicks endpoints exist. Verification required.

### Entry 17: Service Name Confirmation
**Type**: Pending Confirmation  
**Date**: 2026-01-08

**Assumed Name**: digitalcallput
**Source**: Proto package name from product brief

**Impact**: Affects repository name, module path, package names

---

## Summary

Key architectural decisions for the Digital Call/Put Options Pricing Service:

1. **Single Service**: All domain boundaries as internal modules
2. **Stateless**: No persistent data, reconstruct from requests
3. **gRPC**: All external APIs use gRPC with streaming support
4. **Dependency Direction**: All internal dependencies flow toward pricing core
5. **Standard Complexity**: Light modular structure per score 6/10
6. **External Dependency**: service-feed for market data (requires verification)
