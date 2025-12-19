# Phase Preferences

## Technology Stack
- **Language**: Golang
- **Framework**: gRPC
- **Protocol**: Protobuf
- **Dependencies**:
  - `github.com/regentmarkets/service-feed` (Mocked/Interface for now)

## Implementation Patterns
- **Architecture**: Modular Monolith (Internal packages)
- **Error Handling**: Defensive programming, explicit error returns
- **Testing**: Unit tests for Pricer, Integration tests for Handlers

## Code Organization
- `api/`: Protobuf definitions
- `cmd/`: Entry points
- `internal/`: Private application code
  - `pricer/`: Core domain logic (Black-Scholes)
  - `feed/`: Feed service client wrapper
  - `grpcsvc/`: gRPC service implementation
  - `model/`: Domain types
