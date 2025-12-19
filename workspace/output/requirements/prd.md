# Product Requirements Document: Digital Call/Put Options Pricer

## 1. Introduction

### 1.1 Purpose
The purpose of this document is to define the requirements for the `service-pricer-digitalcallput` microservice. This service acts as a **stateless pricing engine** for Digital Call/Put (Binary) options. It calculates prices for new contracts (Ask) and current values for active contracts (Bid), supporting both single requests and streaming updates.

### 1.2 Scope
- **In Scope**:
  - Calculation of Ask prices (premium) for new contracts.
  - Calculation of Bid prices (sell value) for active contracts.
  - Streaming of Ask and Bid prices.
  - Validation of contract parameters (duration, barriers, stake).
  - Integration with `service-feed` for real-time and historical market data.
- **Out of Scope**:
  - Trade execution or order management.
  - User account management or authentication (handled upstream).
  - Persistent storage of contracts (Service is stateless).
  - Payment processing.

### 1.3 Definitions
- **Digital Option**: A binary option where the payout is fixed if the condition is met, and zero otherwise.
- **Call**: Client wins if Exit Price > Barrier.
- **Put**: Client wins if Exit Price < Barrier.
- **Barrier**: The price level that determines the outcome. Can be Absolute (specific price) or Relative (offset from Entry Price).
- **Entry Price**: The market price at the contract start time.
- **Exit Price**: The market price at the contract expiry time (or current time for active valuation).
- **Spot Price**: The current market price of the underlying asset.

## 2. Product Overview

### 2.1 User Stories
- **As a Client**, I want to get a quote (Ask Price) for a Digital Call/Put option so that I know the cost to enter the trade.
- **As a Client**, I want to see a stream of updated quotes so that I can time my entry.
- **As a Client**, I want to know the current value (Bid Price) of my active contract so that I can decide whether to sell it early.
- **As a System**, I need to validate that requested contracts are within defined limits (Min Stake, Max Payout) to manage risk.

### 2.2 Use Cases
1.  **Get Ask Price**: Client requests a price for a potential contract (e.g., "Buy USD/JPY Call, 1m duration, $10 stake"). Service returns the potential payout and current spot.
2.  **Stream Ask Price**: Client subscribes to price updates for the same parameters. Service pushes updates as the spot price changes.
3.  **Get Bid Price**: Client requests the value of an *active* contract. Service calculates the sell price based on current market conditions and time to expiry.
4.  **Stream Bid Price**: Client subscribes to value updates for an active contract.

## 3. Functional Requirements

### 3.1 Pricing Logic (The "Ask")
The service must calculate the proposal (Ask) using the Black & Scholes model.
- **Inputs**:
  - Spot Price (from `service-feed`)
  - Barrier (Client provided or default to Spot)
  - Duration (Client provided)
  - Volatility (Configurable/Feed)
  - Interest Rate (Configurable)
  - Quanto Drift (Configurable)
- **Barrier Calculation**:
  - If `barrier` is explicitly provided as **Absolute**, use as is.
  - If `barrier` is **Relative** (e.g., "+0.05"), `Barrier = Spot Price + Relative Value`.
  - If `barrier` is **Null**, `Barrier = Spot Price`.
- **Limits**:
  - Validate `Stake >= Min Stake`.
  - Validate `Potential Payout <= Max Payout`.
  - Apply Commission (deducted from payout).

### 3.2 Valuation Logic (The "Bid" / Active Contract)
The service must calculate the current value (Bid) of an active contract.
- **State Reconstruction**:
  - Since the service is stateless, it must reconstruct the contract state using the `start_time` provided in the request.
  - **Entry Spot**: Fetch the market price at `start_time` from `service-feed`.
  - **Barrier**:
    - If Relative: `Barrier = Entry Spot + Relative Value`.
    - If Null: `Barrier = Entry Spot`.
- **Winning Condition**:
  - **Call**: `Current Spot > Barrier` -> Potential Win.
  - **Put**: `Current Spot < Barrier` -> Potential Win.
- **Pricing**:
  - Calculate the probability of winning at expiry using the remaining duration and current spot.
  - `Bid Price = Potential Payout * Probability * Discount Factor`.

### 3.3 Lifecycle Management
- **Start**: Contract starts at `start_time`.
- **Expiry**: Contract expires at `start_time + duration`.
- **Updates**:
  - For Streams: Push update on every new tick OR every 5 seconds (heartbeat), whichever is first.
  - Stop streaming when `Current Time >= Expiry Time`.

## 4. Non-Functional Requirements

### 4.1 Performance
- **Latency**: Pricing calculations should be performed in < 10ms (excluding network/feed latency).
- **Throughput**: Support high-frequency updates for streaming.

### 4.2 Reliability
- **Statelessness**: The service must not maintain internal state between requests. All necessary context must be derived from the request or external feeds.
- **Defensive Programming**: Handle missing feed data or invalid parameters gracefully without crashing.

### 4.3 Compliance
- **Audit**: All pricing requests and responses should be logged (masked for PII if applicable, though this service is likely backend-only).

## 5. Technical Architecture

### 5.1 Dependencies
- **`service-feed`**:
  - Source for real-time Spot Prices (for Ask/Bid calculations).
  - Source for historical Spot Prices (to determine `Entry Spot` based on `start_time`).
- **`go-templates`**: Project structure and boilerplate.

### 5.2 Data Flow
1.  **Request**: Client sends gRPC request (Ask/Bid).
2.  **Data Fetch**:
    - Pricer requests current Spot from `service-feed`.
    - (For Bid) Pricer requests historical Spot (at `start_time`) from `service-feed`.
3.  **Calculation**: Pricer applies Black-Scholes logic.
4.  **Response**: Pricer returns calculated price/payout.

## 6. Interface Specifications (gRPC)

### 6.1 Service Definition
```protobuf
syntax = "proto3";

package digitalcallput.v1;

option go_package="github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput";

service DigitalcallputService {
  rpc GetAsk (GetAskRequest) returns (GetAskResponse);
  rpc StreamAsk (StreamAskRequest) returns (stream GetAskResponse);
  rpc GetBid (GetBidRequest) returns (GetBidResponse);
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
  optional int64 start_time = 6; // Required for active contract (Bid)
  string stake = 7; // Premium paid. Required.
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

## 7. Assumptions & Constraints
- **Pricing Parameters**: Volatility, Interest Rate, and Drift are assumed to be static configuration or fetched from a simple provider for V1, as no specific source was defined.
- **Feed Availability**: `service-feed` is assumed to be available and capable of serving both real-time and historical requests with low latency.
- **Currency**: Payout currency is assumed to be supported by the system.
