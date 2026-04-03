# MSA Refactoring Guidelines

Rules for decomposing monoliths into microservices and managing service boundaries.

Priority: Tier 1 on-demand. Read when: monolith-to-microservices refactoring, service decomposition, bounded context analysis.

## Scope

This rule governs the architectural transition from monolithic applications to distributed microservices. It covers boundary definition, migration patterns, data isolation, and API contract management. Apply these rules when extracting logic from a monolith or designing new services in a multi-service environment.

## Inputs/constraints

- Monolith source code is the primary input for extraction.
- Existing database schemas constrain data ownership transitions.
- Organizational structure dictates service boundaries through Conway's Law.
- Infrastructure capabilities define the minimum viable service size.
- Performance requirements limit the depth of service call chains.

## Decision/rules

### Migration patterns

- Apply the Strangler Fig pattern for incremental migration. Wrap the legacy monolith with a new API surface. Route traffic to the new service in small batches. Retire monolith components only after the new service handles 100% of the specific traffic.
- Use Branch by Abstraction for logic-heavy refactors. Create an abstraction layer in the monolith first. Implement the new service logic behind this abstraction. Switch the implementation to the external service. Remove the legacy code once verified.
- Implement UI-First Separation to decouple deployment cycles early. Extract the frontend application from the monolith. Connect the standalone frontend to the monolith via API. This enables independent UI releases while backends are being split.

### Domain-Driven Design (DDD) decomposition

- Identify bounded contexts by analyzing domain events. Group related events that occur within the same logical boundary.
- Classify subdomains to prioritize effort. Focus on the Core domain where competitive advantage lives. Treat Supporting subdomains as necessary but secondary. Use Generic subdomains for commodity features like auth or logging.
- Use an Anti-Corruption Layer (ACL) between bounded contexts. This prevents legacy models or external service models from leaking into the new service's internal domain model.
- Align service boundaries with team boundaries. Each service should have a single owning team to minimize cross-team coordination overhead.

### Service decomposition heuristics

- Group components that change at the same cadence. If two modules always require simultaneous deployment, keep them in the same service.
- Split services based on scaling requirements. Isolate resource-heavy components that need independent scaling from the rest of the system.
- Minimize the blast radius through failure isolation. Separate critical path components from non-essential features so that failure in one doesn't crash the entire system.
- Enforce strict data ownership. Each service must own its data store. No service should reach into another service's database.

### API contract design

- Use the API Gateway pattern to handle cross-cutting concerns. Centralize authentication, rate limiting, and routing at the gateway level.
- Follow contract-first design. Define the API using OpenAPI for REST or Protobuf for gRPC before writing any implementation code.
- Apply semantic versioning to all API endpoints. Communicate breaking changes through major version bumps.
- Maintain backward compatibility by making additive changes only. Use a deprecation period before removing any existing fields or endpoints.

### Data management patterns

- Implement a database per service. Prevent shared database tables across service boundaries to ensure independent deployments.
- Use the Saga pattern for distributed transactions. Prefer event-driven choreography for simple flows. Use orchestration with a central coordinator for complex multi-step processes.
- Apply Command Query Responsibility Segregation (CQRS) when read and write patterns diverge. Separate the data models for updates from the models for queries.
- Use event sourcing for audit-critical domains. Store the state as a sequence of events rather than just the current snapshot.

### Anti-patterns

- Avoid nano-services. Don't create services so small that the operational overhead of managing them outweighs the benefits of separation.
- Don't share business logic libraries. Shared libraries couple services together and force synchronized updates. Prefer duplicating small amounts of code or using service calls.
- Prevent database integration. Never allow two services to share the same database tables.
- Avoid the distributed monolith. If all services must be deployed together to work, the system is a monolith with more failure points.
- Minimize chatty inter-service communication. Prefer coarse-grained APIs that return all needed data in one call rather than many small requests.

### Testing strategy

- Use Consumer-Driven Contract testing. Implement tools like Pact to ensure the service provider doesn't break the expectations of its consumers.
- Apply service virtualization for integration tests. Mock live dependencies to test service behavior in isolation without needing the entire environment running.
- Use canary deployments for production validation. Route a small percentage of traffic to the new service version to detect issues before a full rollout.

## Verification

- Run a dependency analysis to ensure no circular dependencies exist between the new service and the monolith.
- Validate that the new service has its own independent database schema and no shared tables.
- Verify API contract compliance by running contract tests against the provider and consumer.
- Check that all inter-service communication uses the defined API Gateway or message broker.

## Rollback/safety

- Maintain the legacy code path in the monolith until the new service is proven stable.
- Use feature flags to toggle between legacy logic and the new service.
- Implement a kill switch at the API Gateway to route traffic back to the monolith if the new service fails.
- Perform side-by-side testing where both legacy and new paths run, comparing results without affecting the user.

## Reference composition

Tier 1 on-demand rule. Read when monolith decomposition, service boundary analysis, or MSA migration is in scope.
Defers to `00-hard-autonomy-no-questions.md` on execution posture.
Defers to `00-code-modularization.md` for file size governance during service extraction.
