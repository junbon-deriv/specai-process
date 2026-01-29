# Deriv Arcade

I want to build an online arcade-like platform that offers a simple express rise/fall binary option based on a randomly generated series.

## Product Description

This platform is a single-page game where you have a OHLC chart as the focal point and the trading history on the right sidebar. Each game consists of 20 OHLC candles with 5-second interval.

If you zoom into the chart to focus on a game, it consists of three parts:

1. A veritcal rectangular: To display the first 10 candles of the game.
2. A square on the upper-right next to the rectangle: Make this an action button to buy 'rise'
3. A square on the lower-right next to the rectangle: Make this an action button to buy 'fall'

The chart progresses from left to right. When client clicked on either action button, both squares disappears to show the next 10 candles. Contract is evaluated evaluated with the following conditions:

1. Rise contract

- Win payout if the close value of the 20th candle is **STRICTLY HIGHER** than the close value of the 10th candle. If the close value of the 20th candle is equal to or lower than the close value of the 10th candle, zero payout.

2. Fall contract

- Win payout if the close value of the 20th candle is **STRICTLY LOWER** than the close value of the 10th candle. If the close value of the 20th candle is equal to or higher than the close value of the 10th candle, zero payout.

The result of each game is updated in the trading history sidebar with a limit of show the last 50 games for a specific account. The game continues from the 20th candle of the previous game where client gets to predict the next direction of the index.

## Payout Structure

The is a 50% chance for the index value to end higher or lower than the close value of the 10th candle.

${payout} = ${stake}/${probability}

A 3% commission should be included in the probability.

## Series Type

Generates a random sample price of a stock following Geometric Brownian Motion after t years. Start with 4 series with the following configuration:

1. Vol100
- initial value: 50000
- volatility: 100%
- interest rates: 0
- quanto drift: 0
- interval: 1 second
- precision: 0.001

2. Vol200
- initial value: 100000
- volatility: 200%
- interest rates: 0
- quanto drift: 0
- interval: 1 second
- precision: 0.001

3. Vol50
- initial value: 10000
- volatility: 50%
- interest rates: 0
- quanto drift: 0
- interval: 1 second
- precision: 0.001

4. Vol300
- initial value: 200000
- volatility: 300%
- interest rates: 0
- quanto drift: 0
- interval: 1 second
- precision: 0.001

## The platform will support the following accounting capabilities:

1. **CreateAccount**

request parameters:
- account currency; 3-letter currency code with caps.
- external id (external reference id. This could be be used as a user reference for the broker that integrates with this platform)

response parameters:
- account id; integer increment with 'SW' prefix.

2. **Deposit**

request parameters:
- account id
- amount; string with 2-decimal point
- deposit id (for idempotency verification); uuid

response parameters:
- balance; string with 2-decimal point
- transaction id
- transaction_time

3. **Withdraw**

request parameters:
- account id
- amount; string with 2-decimal point
- withdrawal id (for idempotency verification)

response parameters:
- balance; string with 2-decimal point
- transaction id
- transaction_time


4. **GetAccount**

request parameters:
- account id

response parameters:
- balance
- account id
- currency

## The platform will support the following trading capabilities:

1. **SwipeGet**

request parameters:
- series type (the platform will offer different flavours of randomly generated series. This parameter is used to get the right data from the pool of series)

response parameters:
- ohlcs; returns first 10 candles for preview

2. **SwipeBuy**

request parameters:
- account id; bigint
- stake; string with 2-decimal point
- series type
- previous quote (the close value of the 10th candle. This is used to as a reference to continue series generation)
- sentiment (E.g. rise or fall)

response parameters:
- contract id; bigint
- purchase time
- ohlcs; the next 10 candles generated from previous quote.
- payout; string with 2-decimal point

3. **SwipeList**

request parameters:
- account id
- series type (optional, if not provided, it will return all series purchased by the account id)

response parameters:
- contracts; Each contract will have contract id, purchase time, 20 candles and payout

## Trading Consistency Pattern

Transaction Atomicity for Trading: When placing a trade (SwipeBuy), multiple operations happen:

Deduct stake from balance → Create STAKE transaction
Evaluate contract → Create Contract
If win: Credit payout → Create PAYOUT transaction
If lose: Credit 0 → Create ZERO transaction (this is to avoid open-ended contract)

Transactions related to contract should be linked with contract id

## Authentication

This is not a single-user type platform. Authentication will be implemented in second phase.

## Error Handling

Uses standard REST API error handling.

## Deployment

Omit deployment for now.

## STRICT RULES

- The trading platform **DOES NOT** store any personal information about the user (E.g. **NO** email, address, first name etc).
- The trading platform has to be **SELF-SUFFICIENT** and **DOES NOT** depend on external services.
- The random index series will be generated on the fly for each SwipeGet request.
- Each SwipeBuy request should be stored for historical display. Full index series **MUST** be stored.
