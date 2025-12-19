# Preferences

## Complexity Assessment
- **Technical Complexity**: 2 (Microservice, pricing logic, streaming)
- **User Complexity**: 0 (No auth handled in service)
- **Integration Complexity**: 1 (Feed service)
- **Regulatory/Compliance**: 2 (Financial product)
- **Scale/Performance**: 1 (High frequency updates)
- **Total Score**: 6
- **Selected Template**: Standard (implied)

## Project Preferences
- **Language**: Golang
- **Communication**: gRPC
- **Database**: None (Stateless/In-memory)
- **Dependencies**:
  - `github.com/regentmarkets/service-feed`
  - `github.com/regentmarkets/go-templates`
- **Repository**: `github.com/regentmarkets/service-pricer-digitalcallput`

## User Directives
- Follow defensive programming.
- Do not truncate files.
