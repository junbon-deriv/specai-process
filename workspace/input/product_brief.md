# New Financial Product Implementation Template

This template is designed to guide a non-technical team in defining a new financial product and to provide a structured specification for an AI agent to implement the corresponding Go microservice.

## Part 1: Product Definition

Digital Call/Put options is a binary option. For a Call option, client wins full payout if the symbol's exit price is strictly higher than the barrier. For a Put option, client wins full payout if the symbol's exit price is strictly lower than the barrier. Barrier can be defined by the client. The product accepts relative barrier, absolute barrier or null. If relative barrier is provided by the client, the barrier value can be calculated from the symbol's entry price of the contract. Entry price is the next price after contract start time. If barrier is not provided, the product uses the entry price as the barrier.

Client loses the stake if contract expire worthless. Client should be able to request for a single contract price or request for a stream of contract prices. Similarly, client should be able to request for the contract value after the contract is purchased. Client can sell the contract at market value before the contract expiry time.

### 1. Contract Request Parameters for Get & Stream endpoints (Inputs)
*Digital option parameters*
*   **Symbol**: The underlying asset (e.g. USD/JPY, BTC/USD). Required parameter.
*   **Currency**: The payout currency of the contract (e.g., USD, EUR). Required parameter.
*   **Stake**: The premium paid to enter into a contract. Required parameter.
*   **Duration**: A string that consists of duration amount and duration unit (e.g. '1m' is 1 minute, '30s' is 30 seconds, '2h' is 2 hours, '5d' is 5 days and '5t' is 5 ticks). Required parameter.
*   **Barrier**: Relative or absolute barrier. Optional parameter.
    **Start Time**: The start time of the contract. Only required for bid request.
*Pricing time*: Pricing request epoch time. Optional parameter.

### 1.1 Contract Lifecycle and Payout Determination

**CRITICAL**: The payout amount is calculated and fixed at the time of contract purchase (Ask request). Once a contract is purchased, the payout becomes a contractual obligation and MUST NOT be recalculated during the contract's lifetime.

**For Bid Requests**:
- The payout value MUST be included in the request parameters
- The payout value should be the same value that was returned in the original Ask response at purchase time
- The payout parameter ensures consistency and prevents recalculation errors

**Why This Matters**:
- Contract terms are legally binding at purchase time
- Market conditions (volatility, commission rates) may change after purchase
- Recalculating payout would violate contractual obligations
- Financial audits require consistency between purchase and valuation

### 2. Ask Response
*   **Current Spot**: Spot price of the underlying asset.
*   **Current Spot Time**: Time associated with the spot price.
*   **Payout**: Potential payout.
*   **Ask Price**: Ask price.
*   **Currency**: The contract quoted currency.
*   **Limits**: Contains option specific trading limits.

### 3. Bid Response
*   **Current Spot**: Spot price of the underlying asset at pricing time.
*   **Current Spot Time**: Time associated with the current spot price at pricing time.
*   **Entry Spot**: Entry price of the contract (first tick price after start time).
*   **Entry Spot Time**: **CRITICAL** - Timestamp of the entry tick, NOT the contract start time. This is the timestamp from the market feed when the entry spot was recorded. Must match the actual time the entry price was observed in the market.
*   **Exit Spot**: Exit price of the contract (for expired contracts only).
*   **Exit Spot Time**: **CRITICAL** - Timestamp of the exit tick, NOT the expiry time. This is the timestamp from the market feed when the exit spot was recorded.
*   **Bid Price**: Current market value of the contract.
*   **Currency**: The contract quoted currency.
*   **Barrier**: Contract barrier (resolved value, not the input specification).
*   **Start Time**: Contract start time (when contract was purchased).
*   **Expiry Time**: Contract end time (calculated from start time + duration for time-based contracts).
*   **Is Expired**: Boolean to indicate if contract is expired.

**Implementation Note**: Entry Spot Time and Exit Spot Time are market data timestamps, not calculated times. They must be preserved from the actual ticks received from the market feed.

### 4. Limits
*   **Max Payout**: Maximum payout per contract.
*   **Min Stake**: Minimum stake/ask price per contract.

### 5. Pricing Logic (The "Ask")
*How do we calculate the proposal before purchase?*
*   **Inputs**: Spot price, Barrier, Duration in years, Payout currency interest rate, Quanto drift, Volatility.
*   **Formula/Logic**:
    *   How are barriers calculated? If barrier is provided, check if it's absolute or relative barrier. Relative barrier is a string with '+' or '-' sign (E.g. '+0.0023'). Relative barrier value can be calculated from symbol's entry price. If barrier is not provided, use entry price as barrier.
    *   How is the potential payout calculated? Use standard black & scholes formula for payout calculation.
    *   Are there limits (Max Payout, Min Stake)? Min stake and max payout should be defined by symbol in yaml configuration file.
    *   Commission is deducted from potential payout. Commission should be defined by symbol in yaml configuration file.
    *   Volatility is set to 10%.
    *   Quanto drift and interest rate is set to 0.

### 6. Lifecycle & State Machine (The "Bid" / Active Contract)
*What happens after purchase?*

#### Contract Start
*   **Start Condition**: The contract starts at the specified start_time and begins waiting for the entry tick.
*   **Entry Determination**: The contract receives its entry price from the first tick AFTER the start_time.

#### Duration Types and Expiry Logic

**Time-Based Durations (s, m, h, d)**:
*   **Update Frequency**: Contract price should be updated when it receives a new tick OR every 5 seconds (whichever comes first). When no ticks arrive for extended periods, stream should only send price updates.
*   **Expiry Condition**: The contract expires at `start_time + duration`. Use time comparison: `now >= expiry_time`.
*   **Update Stops**: After contract expires (time-based check).
*   **Range**: A maximum of 1 year duration.

**Tick-Based Durations (t)**:
*   **Update Frequency**: Contract price should be updated ONLY when a new tick is received. NO time-based fallback.
*   **Expiry Condition**: The contract expires when exactly N ticks have been received after the entry tick. Track tick count internally.
*   **NO Time-Based Expiry**: Do NOT use time comparisons for tick-based contracts. The contract remains active until the tick count is reached, regardless of elapsed time.
*   **Implementation Requirement**: Maintain a tick counter that increments on each received tick after entry. When counter reaches N, mark contract as expired.
*   **Range**: A maximum of 10 ticks duration.

**CRITICAL DISTINCTION**: Time-based and tick-based contracts have fundamentally different expiry mechanisms. Implementation must use separate code paths:
- Time-based: `if (now >= expiryTime) { expired = true }`
- Tick-based: `if (tickCount >= requiredTicks) { expired = true }`

#### Win/Loss Conditions
*   **Winning Condition**: For a Call option, client wins when the exit price is strictly higher than the barrier. For a Put option, client wins when the exit price is strictly lower than the barrier.
*   **Losing Condition**: For a Call option, client loses when the exit price is lower or equal to the barrier. For a Put option, client loses when the exit price is higher or equal to the barrier.

#### Early Exit (Time-Based Contracts Only)
*   **Time-Based Contracts**: The contract can be sold at market value before expiry. Bid price is calculated using Black-Scholes with remaining time to expiry.
*   **Tick-Based Contracts**: NO early exit supported. Contract must complete all required ticks. Bid price is not calculated for active tick-based until expiry.

---

## Part 2: Technical Implementation Guide (For AI Agent)

### 1. Proto Definition

In service-pricer-digitalcallput repository, update protobuf with the following definitions.

```protobuf
syntax = "proto3";

package digitalcallput.v1;

option go_package="github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput";

service DigitalcallputService {
  // Request a single contract price (Ask)
  rpc GetAsk (GetAskRequest) returns (GetAskResponse);
  // Stream contract prices (Ask Stream)
  rpc StreamAsk (StreamAskRequest) returns (stream GetAskResponse);
  // Request value of an active contract (Bid/Sell)
  rpc GetBid (GetBidRequest) returns (GetBidResponse);
  // Stream value of an active contract
  rpc StreamBid (StreamBidRequest) returns (stream GetBidResponse);
}

enum ContractType {
  CONTRACT_TYPE_UNSPECIFIED = 0;
  CONTRACT_TYPE_CALL = 1;
  CONTRACT_TYPE_PUT = 2;
}

message OptionParameters {
  string symbol = 1;
  ContractType contract_type = 2;
  string currency = 3;
  string duration = 4; // e.g. "1m", "30s", "5t"
  optional string barrier = 5; // Relative (+/-) or absolute. Optional.
  optional int64 start_time = 6; // Required for active contract
  string stake = 7; // Premium paid. Required.
  optional string payout = 8; // REQUIRED for bid requests: Fixed payout from purchase time
}

message GetAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2; // Epoch time
}

message StreamAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}

message GetAskResponse {
  string ask_price = 1;
  string currency = 2;
  string current_spot = 3;
  int64 current_spot_time = 4;
  string payout = 5;
  Limits limits = 6;
}

message Limits {
  string max_payout = 1;
  string min_stake = 2;
}

message GetBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}

message StreamBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}

message GetBidResponse {
  string bid_price = 1;
  bool is_expired = 2;
  string current_spot = 3;
  int64 current_spot_time = 4;
  string entry_spot = 5;
  int64 entry_spot_time = 6;
  string exit_spot = 7;
  int64 exit_spot_time = 8;
  string barrier = 9;
  int64 start_time = 10;
  int64 expiry_time = 11;
  string currency = 12;
}
```
