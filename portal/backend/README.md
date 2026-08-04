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

# OpenFGC Portal Backend

The OpenFGC Portal Backend handles portal-facing authentication flows and securely proxies allowlisted API requests from `portal/frontend` to OpenFGC.

## Quickstart

Prerequisites:

- Go 1.26.3
- [Task](https://taskfile.dev/installation/)
- A running OpenFGC server

From `portal/backend`:

1. Copy `.env.example` to `.env` and set `BFF_PROXY__OPENFGC_API_URL` for your OpenFGC server.
2. To access protected routes locally without an identity provider, enable placeholder mode and set non-empty placeholder user and organization IDs in `.env`.
3. Start the Portal Backend:

   ```shell
   task run:env
   ```

The server listens on `http://localhost:8080` by default. Verify it with `GET /health`.

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

- `GET /me/consents` -> upstream `GET /api/v1/consents` with forced `userIds=<validated subject>` and `details=true`
- `GET /me/consents/{consentId}` -> one upstream `GET /api/v1/consents/{consentId}?details=true`; the detailed consent snapshot is returned unchanged
- `POST /me/consents/{consentId}/approve` -> the Portal Backend fetches the current consent with `details=true`, validates selected optional elements against its embedded stable IDs and versions, automatically approves mandatory elements, preserves existing approvals, and sends a full upstream consent update with the consent's immutable `groupId` as the trusted `group-id` header
- `POST /me/consents/{consentId}/revoke` -> upstream `POST /api/v1/consents/{consentId}/revoke`

Proxy hardening:

- Path rewrite `/api/*` -> `/api/v1/*` with query preservation
- Deny-by-default allowlist for consent-server routes (unknown path -> `404`, known path wrong method -> `405`)
- Hop-by-hop header stripping and trusted-header override prevention (`org-id`, `group-id`, and the rejected legacy client header)
- Correlation ID propagation/generation via `X-Correlation-ID`
- Request body limit enforcement (`BFF_PROXY__MAX_REQUEST_BYTES`) with `413`
- Deterministic upstream error mapping: timeout -> `504`, other connectivity failures -> `502`

Authentication endpoints:

- `GET /auth/login` starts confidential-client OIDC login
- `GET /auth/callback` validates returned tokens through discovery/JWKS and issues split cookies
- `POST /auth/refresh` reconstructs the split refresh token and returns `204`
- `POST /auth/logout` clears cookies and returns the configured or discovered logout URL

Set `BFF_AUTH__ENABLED=true` to use OIDC authentication. The client secret is read only from
`BFF_AUTH__CLIENT_SECRET`; see `.env.example` for the complete configuration contract.

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
- `BFF_PROXY__PLACEHOLDER_ORG_ID` must be empty if placeholder mode is disabled
- `BFF_PROXY__PLACEHOLDER_SCOPES` is a space-separated list of portal scope. Must be empty if placeholder mode is disabled
- Empty placeholder scopes are allowed and produce an authenticated principal with no portal permissions

## AI Instructions

This repository uses AGENTS.md files to keep AI-generated changes aligned with project and organization standards.

- Backend standards: `portal/backend/AGENTS.md`
