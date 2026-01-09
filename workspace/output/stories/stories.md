# User Stories
# Digital Call/Put Options Pricing Service

**Version**: 1.0  
**Date**: 2026-01-08  
**Status**: Draft

---

## 1. Executive Summary

This document defines the user stories for the Digital Call/Put Options Pricing Service, a stateless microservice that provides real-time pricing for digital (binary) options contracts. The service supports two contract types (Call and Put), three barrier modes (absolute, relative, at-the-money), and two duration types (time-based and tick-based).

**Primary User**: Trader - End user who makes trading decisions based on pricing information
**Service Scope**: Price generation only (Ask for purchase, Bid for valuation)
**Key Interactions**: Single price requests and streaming price updates

---

## 2. User Types

### 2.1 Trader

**Description**: An end user who interacts with the trading platform to evaluate and trade digital options. Traders need accurate, timely pricing information to make informed decisions about purchasing contracts and selling active positions.

**Key Characteristics**:
- Makes trading decisions based on real-time pricing
- Requires visibility into potential payout before purchase
- May hold active contracts requiring ongoing valuation
- Trades across multiple symbols and contract configurations

**Access Level**: Indirect access via upstream trading platform (authentication handled externally)

---

## 3. User Stories by Type

### 3.1 Trader Stories

#### Ask Pricing - Single Request

**US-PR-K3M** As a Trader, I want to request an Ask price for a Call option so that I can decide whether to purchase a contract that wins when the price goes up. (pricing)

**US-PR-P7R** As a Trader, I want to request an Ask price for a Put option so that I can decide whether to purchase a contract that wins when the price goes down. (pricing)

**US-PR-A1E** As a Trader, I want to see the potential payout along with the Ask price so that I can evaluate the risk/reward of the contract. (pricing)

**US-PR-B2F** As a Trader, I want to see my trading limits (minimum stake, maximum payout) so that I can structure my trade within allowed boundaries. (pricing)

**US-PR-C3G** As a Trader, I want to see the current spot price and timestamp with my Ask response so that I can verify the pricing is based on current market conditions. (pricing)

#### Ask Pricing - Streaming

**US-PR-D4H** As a Trader, I want to receive streaming Ask price updates so that I can trade at the optimal moment without repeatedly requesting prices. (pricing)

**US-PR-E5I** As a Trader, I want Ask stream updates whenever a new tick arrives so that I can react to market movements immediately. (pricing)

**US-PR-F6J** As a Trader, I want Ask stream updates at least every 5 seconds even without new ticks so that I know the stream is active and prices are current. (pricing)

#### Bid Pricing - Single Request

**US-PR-G7K** As a Trader, I want to request a Bid price for my active time-based contract so that I can decide whether to sell early. (pricing)

**US-PR-H8L** As a Trader, I want to see the resolved barrier value in the Bid response so that I can verify the win/loss threshold. (pricing)

**US-PR-I9M** As a Trader, I want to see the entry spot price and timestamp so that I can verify when my contract started. (pricing)

**US-PR-J1N** As a Trader, I want to see the exit spot and timestamp when my contract has expired so that I can verify the outcome. (pricing)

**US-PR-L2O** As a Trader, I want my original payout preserved exactly as purchased so that the contract terms remain binding. (pricing)

#### Bid Pricing - Streaming

**US-PR-M3P** As a Trader, I want to receive streaming Bid price updates for my active contract so that I can monitor its value and sell at the optimal moment. (pricing)

**US-PR-N4Q** As a Trader, I want the Bid stream to terminate automatically when my contract expires so that I receive the final outcome and the stream closes cleanly. (pricing)

**US-PR-O5R** As a Trader, I want time-based Bid streams to update every tick OR every 5 seconds so that I always have current valuations. (pricing)

**US-PR-Q6S** As a Trader, I want tick-based Bid streams to update only on new ticks so that I see exactly how many ticks have elapsed. (pricing)

#### Barrier Types

**US-CT-R7T** As a Trader, I want to specify an absolute barrier price so that my win/loss is determined against a fixed price level I choose. (contract)

**US-CT-S8U** As a Trader, I want to specify a relative barrier with "+" prefix so that my barrier is set above the entry price by my specified offset. (contract)

**US-CT-T9V** As a Trader, I want to specify a relative barrier with "-" prefix so that my barrier is set below the entry price by my specified offset. (contract)

**US-CT-U1W** As a Trader, I want to omit the barrier parameter so that my contract uses at-the-money pricing where barrier equals entry price. (contract)

#### Duration Types - Time-Based

**US-CT-V2X** As a Trader, I want to specify duration in seconds (e.g., "30s") so that I can trade ultra-short-term contracts. (contract)

**US-CT-W3Y** As a Trader, I want to specify duration in minutes (e.g., "5m") so that I can trade short-term contracts. (contract)

**US-CT-X4Z** As a Trader, I want to specify duration in hours (e.g., "2h") so that I can trade medium-term contracts. (contract)

**US-CT-Y5A** As a Trader, I want to specify duration in days (e.g., "5d") so that I can trade longer-term contracts. (contract)

**US-CT-Z6B** As a Trader, I want my time-based contract to expire based on clock time so that I have predictable expiry timing. (contract)

#### Duration Types - Tick-Based

**US-CT-A7C** As a Trader, I want to specify duration in ticks (e.g., "5t") so that my contract expires based on market activity rather than time. (contract)

**US-CT-B8D** As a Trader, I want tick-based contracts to count only ticks after my entry so that the entry tick is not counted toward expiry. (contract)

**US-CT-C9E** As a Trader, I want to understand that tick-based contracts cannot be sold early so that I set my expectations appropriately. (contract)

#### Contract Lifecycle

**US-MK-D1F** As a Trader, I want my entry price to be the first tick after my contract starts so that I enter at a fair market price. (market)

**US-MK-E2G** As a Trader, I want my exit price to be determined at expiry so that the win/loss outcome reflects the actual market at that moment. (market)

**US-MK-F3H** As a Trader, I want tick timestamps to come from the market feed so that I have accurate audit records. (market)

#### Input Validation & Error Handling

**US-VL-G4I** As a Trader, I want clear error messages when my symbol is not supported so that I know which assets are available for trading. (validation)

**US-VL-H5J** As a Trader, I want clear error messages when my stake is below the minimum so that I can adjust my trade amount. (validation)

**US-VL-I6K** As a Trader, I want clear error messages when my potential payout exceeds the maximum so that I can reduce my stake accordingly. (validation)

**US-VL-J7L** As a Trader, I want clear error messages when my duration format is invalid so that I can correct my input. (validation)

**US-VL-K8M** As a Trader, I want clear error messages when my barrier format is invalid so that I can correct my input. (validation)

**US-VL-L9N** As a Trader, I want to be notified when market data is unavailable so that I know pricing cannot proceed. (validation)

---

## 4. Cross-User Type Stories

Not applicable - single user type (Trader) interacting with a stateless pricing service.

---

## 5. Story Coverage Matrix

| PRD Section/Feature | Related Story IDs | User Type |
|---------------------|-------------------|-----------|
| REQ-CT-K3M: Call Option | US-PR-K3M | Trader |
| REQ-CT-P7R: Put Option | US-PR-P7R | Trader |
| REQ-AP-G1A: GetAsk Endpoint | US-PR-K3M, US-PR-P7R, US-PR-A1E, US-PR-B2F, US-PR-C3G | Trader |
| REQ-AP-S2B: StreamAsk Endpoint | US-PR-D4H, US-PR-E5I, US-PR-F6J | Trader |
| REQ-AP-G3C: GetBid Endpoint | US-PR-G7K, US-PR-H8L, US-PR-I9M, US-PR-J1N, US-PR-L2O | Trader |
| REQ-AP-S4D: StreamBid Endpoint | US-PR-M3P, US-PR-N4Q, US-PR-O5R, US-PR-Q6S | Trader |
| REQ-BR-A1E: Absolute Barrier | US-CT-R7T | Trader |
| REQ-BR-R2F: Relative Barrier | US-CT-S8U, US-CT-T9V | Trader |
| REQ-BR-N3G: Default Barrier (ATM) | US-CT-U1W | Trader |
| REQ-DU-T1H: Time-Based Duration | US-CT-V2X, US-CT-W3Y, US-CT-X4Z, US-CT-Y5A, US-CT-Z6B | Trader |
| REQ-DU-K2I: Tick-Based Duration | US-CT-A7C, US-CT-B8D, US-CT-C9E | Trader |
| REQ-LC-E1M: Entry Tick | US-MK-D1F | Trader |
| REQ-LC-X2N: Exit Tick | US-MK-E2G | Trader |
| REQ-LC-P3O: Payout Immutability | US-PR-L2O | Trader |
| REQ-ST-U1P: Time-Based Stream | US-PR-F6J, US-PR-O5R | Trader |
| REQ-ST-T2Q: Tick-Based Stream | US-PR-Q6S | Trader |
| Section 5.1: Input Validation | US-VL-G4I, US-VL-H5J, US-VL-I6K, US-VL-J7L, US-VL-K8M | Trader |
| Section 6.1: Error Handling | US-VL-L9N | Trader |

---

## 6. Service Distribution Summary

| Service Hint | Story Count | Stories |
|--------------|-------------|---------|
| pricing | 17 | US-PR-K3M, US-PR-P7R, US-PR-A1E, US-PR-B2F, US-PR-C3G, US-PR-D4H, US-PR-E5I, US-PR-F6J, US-PR-G7K, US-PR-H8L, US-PR-I9M, US-PR-J1N, US-PR-L2O, US-PR-M3P, US-PR-N4Q, US-PR-O5R, US-PR-Q6S |
| contract | 12 | US-CT-R7T, US-CT-S8U, US-CT-T9V, US-CT-U1W, US-CT-V2X, US-CT-W3Y, US-CT-X4Z, US-CT-Y5A, US-CT-Z6B, US-CT-A7C, US-CT-B8D, US-CT-C9E |
| market | 3 | US-MK-D1F, US-MK-E2G, US-MK-F3H |
| validation | 6 | US-VL-G4I, US-VL-H5J, US-VL-I6K, US-VL-J7L, US-VL-K8M, US-VL-L9N |

**Total Stories**: 38

### Service Boundary Observations

Given this is a stateless pricing microservice, all service hints map to a single service with internal boundaries:

1. **Core Pricing (pricing)**: Black-Scholes calculations, Ask/Bid price generation - aligns with DOM-PR-H8L
2. **Contract Handling (contract)**: Barrier resolution, duration parsing - aligns with DOM-CT-I9M  
3. **Market Integration (market)**: Tick handling, entry/exit determination - aligns with DOM-MK-J1N
4. **Input Validation (validation)**: Request validation, error responses - cross-cutting concern

---

## 7. Domain Entity Alignment

| Domain Entity | Related Stories | Usage Context |
|---------------|-----------------|---------------|
| Contract (ENT-CT-K3M) | US-PR-K3M, US-PR-P7R, US-CT-* | Central entity for all pricing operations |
| Tick (ENT-MK-T5N) | US-MK-D1F, US-MK-E2G, US-MK-F3H, US-PR-E5I | Market price events driving pricing |
| Symbol (ENT-SY-L8K) | US-VL-G4I, US-PR-B2F | Asset configuration and limits |
| Barrier (VO-CT-P7R) | US-CT-R7T, US-CT-S8U, US-CT-T9V, US-CT-U1W, US-PR-H8L | Win/loss threshold |
| Duration (VO-CT-M9J) | US-CT-V2X through US-CT-C9E | Contract lifecycle specification |
| Price (VO-PR-Q2M) | All US-PR-* stories | Ask/Bid price responses |
| Limits (VO-SY-T5N) | US-PR-B2F, US-VL-H5J, US-VL-I6K | Trading constraints |

---

## 8. Glossary

| Term | Definition | Related Stories |
|------|------------|-----------------|
| Ask | The price to purchase (enter) a contract | US-PR-K3M through US-PR-F6J |
| Bid | The current market value of an active contract | US-PR-G7K through US-PR-Q6S |
| Barrier | The price level that determines win/loss outcome | US-CT-R7T through US-CT-U1W |
| Call Option | Contract winning if exit price > barrier | US-PR-K3M |
| Put Option | Contract winning if exit price < barrier | US-PR-P7R |
| Entry Spot | First tick price after contract start | US-MK-D1F |
| Exit Spot | Final tick price at contract expiry | US-MK-E2G |
| Payout | Fixed amount paid on winning | US-PR-A1E, US-PR-L2O |
| Stake | Premium paid to enter the contract | US-VL-H5J |
| Tick | A single price update from market feed | US-MK-F3H |
| ATM | At-the-money: barrier equals entry price | US-CT-U1W |

---

## 9. Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-08 | AI Architect | Initial user stories creation |
