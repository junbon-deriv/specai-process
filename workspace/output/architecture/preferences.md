# Preferences

## Service Boundaries
- **Architecture Style**: Microservices (Single service focus).
- **Service Granularity**: One service (`service-pricer-digitalcallput`) as defined in PRD.
- **Guiding Principles**: Stateless, Defensive Programming.

## Communication Patterns
- **Synchronous**: gRPC for all client-facing and internal feed interactions.
- **Asynchronous**: None specified (Streaming is over gRPC).

## Data Strategy
- **Ownership**: Service is stateless; owns no persistent data.
- **Consistency**: Relies on `service-feed` for data truth.

## Technology Preferences
- **Language**: Golang.
- **Framework**: Standard Go templates (`github.com/regentmarkets/go-templates`).
- **Protocol**: gRPC (Protobuf).
- **Dependencies**: `service-feed`.
