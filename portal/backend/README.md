<!--
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 -->

# OpenFGC Portal Backend (BFF)

Stateless backend-for-frontend (BFF) for OpenFGC Portal, responsible for handling portal-facing authentication flows and securely proxying API requests from `portal/frontend` to `/consent-server`.

## Commands

- `task fmt`
- `task fmt:check` (no edits. check only)
- `task lint`
- `task lint:install` (optional, installs golangci-lint to GOPATH/bin)
- `task test`
- `task build`
- `task run`
- `task run:env` (loads variables from `.env` for local development)

Install Task if needed: https://taskfile.dev/installation/

## Configuration

- Primary source: `BFF_` environment variables
- Optional file overlay: set `BFF_CONFIG_FILE` to a YAML config file path
- Final effective config is: defaults < file < env

### CORS for local frontend

When running `portal/frontend` on a different origin (for example Vite dev server), allow that origin with:

- `BFF_CORS__ALLOWED_ORIGINS` (comma-separated origins)
- `BFF_CORS__ALLOWED_METHODS` (comma-separated methods)
- `BFF_CORS__ALLOWED_HEADERS` (comma-separated request headers)
- `BFF_CORS__ALLOW_CREDENTIALS` (`true`/`false`)

Requests that include an `Origin` header from a non-allowlisted origin are rejected.
When credentials are enabled, origins must be explicitly allowlisted (wildcard origin is rejected).

## Health endpoints

- `GET /health`
- `GET /health/liveness`
- `GET /health/readiness`

## API endpoints

Portal-facing user endpoints:

- `GET /me/consents` -> upstream `GET /api/v1/consents` with forced `userIds=<placeholder>`
- `GET /me/consents/{consentId}` -> upstream `GET /api/v1/consents/{consentId}`
- `POST /me/consents/{consentId}/approve` -> BFF fetches current consent, merges selected optional approvals, uses the consent's `clientId` as the trusted upstream `TPP-client-id`, updates an existing authorization to approved for the trusted user (or creates one if none exist), and upstreams `PUT /api/v1/consents/{consentId}`
- `PUT /me/consents/{consentId}/revoke` -> upstream `PUT /api/v1/consents/{consentId}/revoke`

Proxy hardening:

- Path rewrite `/api/*` -> `/api/v1/*` with query preservation
- Deny-by-default allowlist for consent-server routes (unknown path -> `404`, known path wrong method -> `405`)
- Hop-by-hop header stripping and trusted-header override prevention (`org-id`, `TPP-client-id`)
- Correlation ID propagation/generation via `X-Correlation-ID`
- Request body limit enforcement (`BFF_PROXY__MAX_REQUEST_BYTES`) with `413`
- Deterministic upstream error mapping: timeout -> `503`, other connectivity failures -> `502`

Error contract for proxy-originated failures:

```json
{
	"code": "REQUEST_TOO_LARGE",
	"message": "request entity too large"
}
```

Common error codes:

- `REQUEST_TOO_LARGE`
- `METHOD_NOT_ALLOWED`
- `NOT_FOUND`
- `INVALID_PAYLOAD`
- `UPSTREAM_TIMEOUT`
- `UPSTREAM_UNAVAILABLE`

## Placeholder mode guardrails

- `BFF_PROXY__PLACEHOLDER_MODE_ENABLED=true` is blocked when `BFF_ENV=production`
- `BFF_PROXY__PLACEHOLDER_USER_ID` must be empty if placeholder mode is disabled

## AI Instructions

This repository uses VS Code Copilot instruction files to keep AI-generated changes aligned with project and organization standards.

- Backend standards: `portal/backend/AGENTS.md`
- Copilot workspace entrypoint (repo root): `.github/copilot-instructions.md`
- Scoped instructions folder (repo root): `.github/instructions/`
- Backend scope mapping: `portal/backend/**` -> `.github/instructions/portal-backend.instructions.md`

Copilot instructions are discovered automatically and scoped using `applyTo` patterns.
