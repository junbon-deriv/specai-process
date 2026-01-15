# Mode

**This task should be run in `roo-verify` custom mode.**

# Goal

We have generated a Service Architecture document that translates our domain model into concrete service boundaries and defines the overall system architecture.
This architecture will guide all subsequent service implementation phases.
I want you to thoroughly check this Service Architecture and verify it before we proceed.

# Read

Read all the content related to service architecture:
* `workspace/output/architecture/architecture.md`: The service architecture document.
* `workspace/output/domain/domain_model.md`: The domain model with business entities and boundaries.
* `workspace/output/requirements/prd.md`: The comprehensive product requirements document.
* `workspace/output/stories/stories.md`: The user stories document.
* `workspace/output/requirements/preferences.md`: The document with user directives and clarifications (if it exists).
* `workspace/output/domain/preferences.md`: Domain modeling preferences (if it exists).
* `workspace/dependency/service_template_guide.md`: The service template guide (if it exists) - **CRITICAL for service structure verification**.

**Important**: Focus your verification on the service architecture and its alignment with domain model and requirements. Do not read any other verify.md files.

# Verify

Read all of these at once and then check if the service architecture is complete and ready:

## Domain Alignment
* Do service boundaries directly map to domain boundaries from the domain model?
* Are aggregate roots properly owned by single services?
* Is there any violation of domain boundaries?
* Are bounded contexts properly represented as services?

## Service Coverage
* Are all PRD requirements addressed by at least one service?
* Are there any requirements that fall between service boundaries?
* Is each requirement clearly owned by a specific service?
* Are all user stories covered by services?

## Service Design
* Are service boundaries clear and well-defined?
* Is data ownership clearly established for each service?
* Are there any circular dependencies between services?
* Do services have appropriate levels of independence?
* Are service names meaningful and domain-aligned?

## API Structure
* Are public API offerings clearly defined where needed?
* Are internal API dependencies between services reasonable?
* Is the separation between public and internal APIs clear?
* Does each service own its public API (no orchestrator pattern)?

## Data Strategy
* Is the data ownership strategy clear and consistent?
* Are transaction boundaries well-defined?
* Is the approach to eventual consistency documented?
* Are there any data integrity concerns?

## Implementation Feasibility
* Can each service be developed independently?
* Are the dependencies manageable?
* Is the suggested development order logical?
* Are technology choices appropriate?

## Communication Patterns
* Is the Inter-Service Communication Matrix complete?
* Are all service dependencies captured?
* Are communication patterns (sync/async) appropriate?
* Are there any missing integration points?

## Specific Checks
* Does each service have a single-word name?
* Are all business capabilities properly distributed?
* Is the Requirements Coverage Matrix complete and accurate?
* Is the User Story Coverage Matrix comprehensive?
* Are there any missing services that should be added?
* Are there any services that should be merged or split?

## Service Template Compliance (If `workspace/dependency/service_template_guide.md` exists)
**CRITICAL**: If the service template guide exists, verify compliance with its patterns:

### Naming Conventions
* Do service names follow the template naming convention (e.g., `service-pricer-{product}`)?
* Are module paths structured correctly (e.g., `github.com/regentmarkets/service-pricer-{product}`)?

### Required Components
* Does the architecture specify the required internal packages based on the component matrix?
  - Feed Client (`internal/feed/`) - if product needs market data
  - Config Manager (`internal/config/`) - if product has configuration
  - Pricer (`internal/pricer/`) - if product has pricing logic
  - Contract Manager (`internal/contract/`) - if product has business rules
* Are the required interfaces defined for each component (ConfigProvider, FeedProvider)?

### Dependency Direction Rules
* Does the architecture follow the dependency graph: `grpcsvc → pricer → (config, feed, contract)`?
* Does the pricer define interfaces that other packages implement?
* Is the pricer free from imports of config, feed, or contract packages?
* Are all packages free from importing grpcsvc?

### API Patterns
* Is gRPC specified as the primary API protocol?
* Is the Ask/Bid API pattern implemented where applicable?
* Are gRPC standard error codes used (not custom errors)?
* Do gRPC handlers only receive and pass requests (not assembling data from various sources)?

### Strict Rules Compliance
* Is there NO models, types, or interfaces package? (Dependencies should flow toward core/pricing)
* Is the service designed to be stateless where possible?
* Is the configuration structure following the specified YAML format?

# Output

Create a thorough report at `workspace/cache/verify_architecture_architecture.md`. This should explain:
* What is working well in the service architecture
* Any issues, gaps, or concerns that need to be addressed
* Specific recommendations for improvements
* Assessment of alignment with domain model
* Assessment of requirements and story coverage
* Overall readiness to proceed with service development
* **Service template compliance status** (if service template guide exists)

Structure the report with these sections:
- **Summary**: Overall assessment (Ready/Not Ready)
- **Strengths**: What aspects of the architecture are well done
- **Critical Issues**: Must-fix problems before proceeding
- **Recommendations**: Specific improvements needed
- **Domain Alignment Analysis**: How well services map to domain boundaries
- **Coverage Analysis**: Completeness of requirement and story coverage
- **Dependencies Review**: Assessment of service dependencies and communication patterns
- **Service Template Compliance** (if `workspace/dependency/service_template_guide.md` exists): Assessment of compliance with service template patterns, including:
  - Naming convention compliance
  - Required component coverage
  - Dependency direction adherence
  - API pattern compliance
  - Strict rules violations (if any)