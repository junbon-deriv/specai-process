# Preferences

- Golang service with gRPC endpoint.
- Service is stateless and will have no database access.

# Service Template
- Request ${SERIVCE_NAME} from user.
- Project should follow https://github.com/junbon-deriv/go-templates
- Follow these instructions to instantiate a service:
    *   cd workspace/code
    *   git clone git@github.com:junbon-deriv/go-templates.git
    *   cd go-templates
    *   go install
    *   cd ../
    *   git clone git@github.com:regentmarkets/service-pricer-${SERVICE_NAME}.git
    *   go-templates --template service --module-path github.com/regentmarkets/service-pricer-${SERVICE_NAME} --module-name ${SERVICE_NAME}

# Service Dependency
- Pricer service is dependent on https://github.com/junbon-deriv/service-feed for data feed.
- Follow these instructions to explore service feed API:
    *   git clone git@github.com:junbon-deriv/service-feed.git
    *   service API is defined in protobuf file located at service-feed/proto/grpcfeed/v1/ticks.proto
    *   preferred service client implementation is defined in service-feed/client/client.go
- Pricer will be a production-ready service. Dependency integration must be done.

## CRITICAL: Dependency Verification Requirements
**MANDATORY**: Before implementing any integration with external services:

1. **Clone the Dependency Repository**:
   - Always clone the actual dependency repository to workspace/code
   - Do NOT assume API contracts without verification
   - Example: `cd workspace/code && git clone git@github.com:junbon-deriv/service-feed.git`

2. **Verify the Actual API**:
   - Read the actual proto file to see available endpoints
   - Check the preferred client implementation for usage patterns
   - Do NOT create integration code based on assumptions

3. **Document What Actually Exists**:
   - In architecture documents, specify ACTUAL endpoints (not desired/assumed ones)
   - Use verified endpoint names from the proto file
   - Mark any missing functionality as enhancement requests

4. **Use Correct Endpoints**:
   - Match endpoint usage to their intended purpose:
     - Unary RPCs for single requests (e.g., GetLatestTick)
     - Streaming RPCs for continuous data (e.g., StreamTicks)
   - Never misuse streaming endpoints for single value retrieval

**Example - service-feed Integration**:
- ✅ Clone service-feed first
- ✅ Read proto/grpcfeed/v1/ticks.proto to see: GetLatestTick, StreamTicks, GetTicks, etc.
- ✅ Use GetLatestTick for single tick retrieval (not StreamTicks)
- ✅ Document verified API in architecture specs

**Why This Matters**:
- Prevents implementing against assumed/imaginary APIs
- Ensures efficient endpoint usage
- Avoids workarounds for non-existent problems
- Creates accurate technical documentation

# Architecture rules

In the architecture document, please highlight the following strict rules that must be obeyed by roo-pe:
- grpc handlers should receive requests from client and pass them to higher level modules, they should not be fetching data from various sources and assembling it together.
- always return gRPC standard error codes.
- do not ever create models/types package. All the dependencies should be directed towards the core (in this case, pricing). Other packages should depend on pricing, pricing should not depend on anything else.
- interface should be defined where it is consumed.
