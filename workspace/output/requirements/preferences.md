# Requirements Phase Preferences

**Document Version**: 1.0  
**Date**: 2026-01-08  
**Phase**: Requirements Analysis

---

## Complexity Assessment

### Entry 1: Application Complexity Scoring
**Type**: Assessment  
**Date**: 2026-01-08

| Dimension | Score | Justification |
|-----------|-------|---------------|
| Technical Complexity | 2/3 | Multiple components (Ask/Bid pricing, streaming), complex Black-Scholes calculations, distinct handling for time-based vs tick-based contracts |
| User Complexity | 0/2 | Single user type (client/trader), no authentication within service |
| Integration Complexity | 1/2 | Single external integration with service-feed for market data |
| Regulatory/Compliance | 2/2 | Financial trading service, legally binding payout calculations, audit requirements |
| Scale/Performance | 1/1 | Real-time pricing required, streaming endpoints, performance critical |

**Total Score**: 6/10  
**Selected Template**: Standard PRD Template  
**Override**: None requested

---

## Scope Decisions

### Entry 2: Stateless Service Design
**Type**: Directive (from workspace preferences)  
**User Input**: "Service is stateless and will have no database access."  
**Context**: Architectural constraint from workspace preferences  
**Impact**: All contract state must be reconstructed from request parameters. No persistent tick counting across requests.  
**Date**: 2026-01-08

### Entry 3: Service Boundary
**Type**: Directive (from workspace preferences)  
**User Input**: "Pricer will be a production-ready service. Dependency integration must be done."  
**Context**: Service must integrate with service-feed, not mock it  
**Impact**: Architecture must include verified service-feed integration  
**Date**: 2026-01-08

---

## Technical Constraints

### Entry 4: Pricing Parameters (Fixed Values)
**Type**: Clarification (from product brief update)  
**Question**: Where do volatility, interest rate, and quanto drift values come from?  
**User Input**: Updated product brief specifies:
- Volatility = 10% (0.10)
- Interest Rate = 0%
- Quanto Drift = 0%
**Context**: These values are hardcoded for initial implementation  
**Impact**: Pricing formula uses fixed parameters, no external data source needed for these values  
**Date**: 2026-01-08

### Entry 5: Symbol Configuration Source
**Type**: Clarification (from product brief update)  
**Question**: Where does symbol-specific configuration (commission, limits) come from?  
**User Input**: "Min stake and max payout should be defined by symbol in yaml configuration file. Commission should be defined by symbol in yaml configuration file."  
**Context**: Configuration stored locally in YAML, not from external service  
**Impact**: Service reads YAML config at startup for symbol parameters  
**Date**: 2026-01-08

### Entry 6: Duration Ranges
**Type**: Clarification (from product brief update)  
**Question**: What are the valid ranges for durations?  
**User Input**: 
- Time-based: "A maximum of 1 year duration"
- Tick-based: "A maximum of 10 ticks duration"
**Context**: Validation bounds for duration parameter  
**Impact**: Input validation must enforce these limits  
**Date**: 2026-01-08

### Entry 7: Relative Barrier Format
**Type**: Clarification (from product brief update)  
**Question**: What's the format for relative barriers?  
**User Input**: "Relative barrier is a string with '+' or '-' sign (E.g. '+0.0023')"  
**Context**: Parsing logic for barrier parameter  
**Impact**: Barrier parsing must detect +/- prefix to determine relative vs absolute  
**Date**: 2026-01-08

---

## Feature Priorities

### Entry 8: Tick-Based Bid Pricing
**Type**: Clarification (from product brief update)  
**Question**: How should bid price be calculated for active tick-based contracts?  
**User Input**: "Bid price is not calculated for active tick-based until expiry."  
**Context**: No early exit for tick-based contracts  
**Impact**: Bid requests for active tick-based contracts return no meaningful bid price (0 or not applicable)  
**Date**: 2026-01-08

### Entry 9: Stream Behavior When No Ticks Arrive
**Type**: Clarification (from product brief update)  
**Question**: What should streams do when no ticks arrive for extended periods?  
**User Input**: "When no ticks arrive for extended periods, stream should only send price updates."  
**Context**: Time-based contracts update every 5 seconds even without new ticks  
**Impact**: 5-second timer sends price updates, not heartbeats  
**Date**: 2026-01-08

---

## Business Rules

### Entry 10: Payout Immutability
**Type**: Directive (from product brief)  
**User Input**: "The payout amount is calculated and fixed at the time of contract purchase (Ask request). Once a contract is purchased, the payout becomes a contractual obligation and MUST NOT be recalculated during the contract's lifetime."  
**Context**: Critical financial compliance requirement  
**Impact**: 
- Payout calculated once during Ask
- Bid requests MUST include payout parameter
- Service never recalculates payout
**Date**: 2026-01-08

### Entry 11: Entry/Exit Spot Time Semantics
**Type**: Directive (from product brief)  
**User Input**: "Entry Spot Time and Exit Spot Time are market data timestamps, not calculated times. They must be preserved from the actual ticks received from the market feed."  
**Context**: Audit trail accuracy requirement  
**Impact**: 
- entry_spot_time comes from tick.timestamp, not calculated from start_time
- exit_spot_time comes from tick.timestamp, not calculated from expiry_time
**Date**: 2026-01-08

### Entry 12: Win/Loss Conditions - Strict Comparison
**Type**: Directive (from product brief)  
**User Input**: 
- Call wins: "exit price is strictly higher than the barrier"
- Put wins: "exit price is strictly lower than the barrier"
**Context**: Equality goes to the house (client loses)  
**Impact**: 
- Call: exit > barrier = win, exit <= barrier = loss
- Put: exit < barrier = win, exit >= barrier = loss
**Date**: 2026-01-08

---

## Architecture Rules

### Entry 13: Handler Responsibility
**Type**: Directive (from workspace preferences)  
**User Input**: "grpc handlers should receive requests from client and pass them to higher level modules, they should not be fetching data from various sources and assembling it together."  
**Context**: Clean architecture principle  
**Impact**: Handlers delegate to service layer; handlers don't orchestrate  
**Date**: 2026-01-08

### Entry 14: No models/types Package
**Type**: Directive (from workspace preferences)  
**User Input**: "do not ever create models/types package. All the dependencies should be directed towards the core (in this case, pricing). Other packages should depend on pricing, pricing should not depend on anything else."  
**Context**: Dependency inversion principle  
**Impact**: Pricing is the core domain; all dependencies point toward it  
**Date**: 2026-01-08

### Entry 15: Interface Location
**Type**: Directive (from workspace preferences)  
**User Input**: "interface should be defined where it is consumed."  
**Context**: Go interface best practice  
**Impact**: Interfaces defined in consumer package, not provider package  
**Date**: 2026-01-08

### Entry 16: Error Codes
**Type**: Directive (from workspace preferences)  
**User Input**: "always return gRPC standard error codes."  
**Context**: API consistency requirement  
**Impact**: All errors wrapped in appropriate gRPC status codes  
**Date**: 2026-01-08

---

## Dependency Verification

### Entry 17: service-feed Verification Requirement
**Type**: Directive (from workspace preferences)  
**User Input**: "Before implementing any integration with external services: Clone the Dependency Repository, Verify the Actual API, Document What Actually Exists, Use Correct Endpoints."  
**Context**: Prevents implementing against assumed APIs  
**Impact**: Architecture phase must clone service-feed and verify actual endpoints before documenting integration  
**Date**: 2026-01-08

---

## Open Items

### Entry 18: Service Name
**Type**: Pending Question  
**Question**: What is the service name for the repository?  
**User Input**: Not yet provided  
**Context**: Required for service instantiation: `service-pricer-${SERVICE_NAME}`  
**Default**: Based on product brief proto, appears to be "digitalcallput"  
**Impact**: Affects repository name, module path, package names  
**Date**: 2026-01-08

---

## Summary

This document captures all decisions, directives, and clarifications made during the requirements phase for the Digital Call/Put Options Pricing Service. Key decisions include:

1. **Complexity**: Rated 6/10, using Standard PRD template
2. **Fixed Pricing Parameters**: Volatility 10%, rates 0%
3. **Configuration**: YAML file for symbol-specific limits and commission
4. **Tick-Based Contracts**: No early exit, no bid calculation until expiry
5. **Payout Immutability**: Critical business rule - payout fixed at purchase
6. **Architecture**: Clean architecture with pricing as core domain
