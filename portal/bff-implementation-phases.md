## Plan: Phased BFF Implementation

The phases below include your requirement that AI guidance files are created in Phase 1, while deployment happens only after all phases complete.

**Phase 1: Bootstrap, standards, and tooling (blocks all later phases)**
Goal: establish a runnable project foundation with quality controls.

1. Create BFF project skeleton and core package layout.
2. Add startup, health endpoint, config loader, logging, router wiring, and graceful shutdown.
3. Set up linting and code quality checks in local scripts and CI.
4. Add AI guidance artifacts as mandatory project assets:
   - AGENTS.md
   - .github/copilot-instructions.md
5. Add initial test scaffolding for unit and integration tests.
6. Add initial OpenAPI contract placeholders for BFF routes.

Exit criteria:
1. Service starts and health endpoint returns success.
2. Lint, format, and baseline tests pass in CI.
3. AI guidance files exist and are referenced in contributor docs.

**Phase 2: Proxy layer with config-driven identity placeholders (depends on Phase 1)**
Goal: validate proxy behavior independently from auth.

1. Implement proxy path mapping, method/path allowlist, timeout policy, and body limits.
2. Introduce portal user endpoints in proxy mode (for example, `/me/consents`, `/me/consents/{consentId}`, `/me/consents/{consentId}/approve`, `/me/consents/{consentId}/revoke`) with explicit route mappings to upstream `/api/v1/*` contracts.
3. Add secure header handling:
   - strip hop-by-hop headers
   - ignore client-supplied trusted headers
   - propagate/generate correlation id
4. Inject currently unavailable auth-dependent values from config for test mode.
5. Add hard guardrails so placeholder mode cannot run in production.
6. Add integration tests for rewrite logic, header safety, query preservation, error mapping, and portal user endpoint route mappings in placeholder mode.
7. Defer user ownership enforcement and principal-derived scoping for `/me/*` endpoints to Phase 3 when auth/session context is available.

Exit criteria:
1. Proxy behavior is fully testable without auth integration.
2. Security checks on headers and route allowlisting pass.
3. Portal user endpoint mappings are implemented and validated in placeholder mode.
4. Placeholder identity mode is explicitly restricted to non-production use.

**Phase 3: Authentication and session implementation (depends on Phase 2)**
Goal: replace placeholders with real identity from auth context.

1. Implement login, callback, me, refresh, and logout flows.
2. Implement split-cookie security model and CSRF protections.
3. Add auth/session middleware and context propagation.
4. Replace config-based identity injection with validated principal-derived values.
5. Add end-to-end auth + proxy integration tests.

Exit criteria:
1. Full authenticated request lifecycle works end to end.
2. Refresh/logout error mappings and cookie behaviors match spec.
3. Config placeholder mode is off by default and not used in normal runtime.

**Phase 4: Hardening and release readiness (depends on Phase 3)**
Goal: finalize operational safety and production readiness.

1. Validate key rotation, session cookie size limits, refresh race behavior, and timeout handling.
2. Complete deployment and operations documentation.
3. Run full regression suite in local compose and staging-like setup.
4. Finalize release packaging and CI gates.

Exit criteria:
1. All tests and quality gates are green.
2. Security and operational checks are complete.
3. Release artifact is ready.