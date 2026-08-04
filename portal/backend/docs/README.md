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

# Portal Architecture and Operations

This guide describes the implemented OpenFGC Portal architecture. For commands and component-specific setup, use the [backend README](../README.md) and [frontend README](../../frontend/README.md). The complete HTTP contract is defined in the [OpenAPI specification](../openapi/portal-backend.yaml).

## Architecture

```text
React frontend ─────► Portal Backend ─────► OpenFGC
                          │
                          └──── OIDC discovery and flows ────► Identity provider
```

### Responsibilities

| Component | Responsibility |
| --- | --- |
| React frontend | Starts login and logout, sends the readable access-token half, refreshes expired sessions, renders ID-token claims for display, and uses the effective scopes returned by `GET /me` for route and element visibility. These UI decisions improve the user experience; Portal Backend scope enforcement remains the security boundary. |
| Portal Backend | Acts as the OIDC confidential client and protected API boundary. It validates tokens, creates the trusted principal, enforces route policy, and maps allowed requests to OpenFGC. |
| Identity provider | Authenticates users, exposes OIDC discovery and JWKS metadata, and issues signed ID, access, and refresh tokens. |
| OpenFGC | Performs consent operations behind the Portal Backend. It receives identity and organization context derived by the Portal Backend. |

The Portal Backend is stateless: login correlation is held in short-lived browser cookies, and authenticated request context is reconstructed from validated tokens. OpenFGC should not be exposed directly to browsers or the public network.

## Authentication

Authentication is enabled with `BFF_AUTH__ENABLED=true`. The Portal Backend performs OIDC discovery during startup and uses Authorization Code flow as a confidential client.

### Login

1. The frontend navigates to `GET /auth/login`.
2. The Portal Backend generates random OAuth state and a PKCE verifier, stores each in a short-lived `HttpOnly` cookie, and redirects to the identity provider with an S256 code challenge.
3. The identity provider redirects to `GET /auth/callback` with one authorization code and state value.
4. The Portal Backend consumes the transaction cookies, compares state, validates the verifier, and exchanges the code using the client credentials and PKCE verifier.
5. The returned access token is validated for the configured API audience; the ID token is validated for the OIDC client. Both use issuer discovery and JWKS verification.
6. The Portal Backend issues split-token cookies and redirects to `BFF_AUTH__PORTAL_URL`.

Invalid, incomplete, duplicate, or replayed callback data is rejected with a generic login failure. Transaction cookies are cleared before the code exchange so they cannot be reused.

### Split-token session

Tokens are divided at a deterministic midpoint and constrained by configured per-part and reconstructed-token size limits.

| Token | JavaScript-readable portion | `HttpOnly` portion | Use |
| --- | --- | --- | --- |
| Access token | Part 1 cookie | Part 2 cookie | Part 1 is sent as `Authorization: Bearer <part-1>`; the browser sends part 2 with credentials. |
| Refresh token | Part 1 cookie | Part 2 cookie | Part 1 is submitted in the form body to `/auth/refresh`; the browser sends part 2 with credentials. |
| ID token | Parts 1 and 2 | None | The frontend reconstructs and decodes it for profile display only. |

For each protected request, the Portal Backend requires exactly one partial Bearer value and exactly one access-token part-2 cookie. It reconstructs the JWT, verifies its signature, issuer, resource audience, expiry, not-before claim when present, subject, organization claim, and configured token type when enabled. The validated identity is normalized into a [`Principal`](../internal/system/context/user.go) containing `UserID`, `OrgID`, and `Scopes`, then stored in request context. Downstream authorization and proxy code use this principal rather than raw token claims.

The split-token design prevents JavaScript from reading a complete access or refresh token, but it does not eliminate the impact of same-origin script injection. The frontend therefore emits a restrictive Content Security Policy and performs production security checks as part of its build workflow.

### Refresh and request retry

`POST /auth/refresh` accepts only `application/x-www-form-urlencoded` with one `refresh_token` field containing refresh-token part 1. Cookie-only refresh is rejected. The Portal Backend reconstructs the refresh token, exchanges it at the identity provider, validates the returned access token and optional ID token, and returns `204 No Content` after replacing the applicable cookies. A rotated refresh token replaces both refresh cookies; otherwise the existing refresh cookies remain in use. Identity-provider exchange or returned-token validation failures clear the token cookies.

The frontend deduplicates concurrent refresh attempts within the current browser runtime. After a protected request returns `401`, it performs at most one refresh and retries that request once. A failed refresh starts login. A `403` is treated as an authorization failure and does not trigger refresh.

### Logout

`POST /auth/logout` is protected by the same split access-token authentication as other protected routes. It clears all local token cookies and returns a logout URL. When OIDC discovery provides an end-session endpoint, the Portal Backend uses it with the configured post-logout redirect and includes the reconstructed ID token only if it validates successfully. The frontend navigates only to an explicitly allowed HTTP or HTTPS origin.

## Routes and Authorization

### Public and authentication routes

| Routes | Access |
| --- | --- |
| `GET /health`, `/health/liveness`, `/health/readiness` | Public |
| `GET /auth/login`, `GET /auth/callback`, `POST /auth/refresh` | Public protocol endpoints; login and refresh still require their corresponding split transaction or token material. |
| `POST /auth/logout` | Authenticated |
| `GET /me` | Authenticated without a business-scope requirement; returns the validated user ID, organization ID, and effective portal scopes. |

`GET /me` returns only scopes in the canonical portal registry, preserving registry order. OIDC scopes such as `openid`, `profile`, and `email` are excluded. An authenticated principal without portal permissions receives `200 OK` with an empty `scopes` array.

### Current-user consent routes

| Routes | Required scope |
| --- | --- |
| `GET /me/consents`, `GET /me/consents/{consentId}` | `portal:consents:read:self` |
| `POST /me/consents/{consentId}/approve` | `portal:consents:write:self` |
| `POST /me/consents/{consentId}/reject` | `portal:consents:write:self` |
| `POST /me/consents/{consentId}/revoke` | `portal:consents:write:self` |

The Portal Backend replaces browser-supplied user filters with the validated subject. Object routes fetch the consent and verify ownership before returning details or applying a mutation. Approval and rejection payloads are built from the current consent snapshot, and trusted `actionBy` and `group-id` values are derived by the Portal Backend rather than accepted from the browser.

### Controlled API passthrough

`/api/*` is a deny-by-default passthrough to OpenFGC `/api/v1/*`. Unknown paths return `404`; unsupported methods on known paths return `405`. The allowlist covers consent, consent-authorization, consent-element, and consent-purpose operations documented in OpenAPI. The canonical portal scope names and method/path-to-scope policies are defined in [`scopes.go`](../internal/system/auth/scopes.go).

| Operation family | Read scope | Write scope |
| --- | --- | --- |
| Consents and authorizations | `portal:consents:read:any` | `portal:consents:write:any` |
| Consent elements | `portal:elements:read` | `portal:elements:write` |
| Consent purposes | `portal:purposes:read` | `portal:purposes:write` |

Authentication failures return `401` with `WWW-Authenticate: Bearer`. An authenticated principal without the required scope receives `403` without a bearer challenge.

### Trusted upstream context

Before forwarding, the Portal Backend removes hop-by-hop headers and browser-supplied identity headers. It sets `org-id` from the validated principal, adds a consent-bound `group-id` only for self-service mutations that require it, and propagates or generates `X-Correlation-ID`. The raw access token is not forwarded to OpenFGC.

## Configuration and Runtime

The Portal Backend loads configuration with the following precedence:

```text
defaults < optional YAML file < BFF_ environment variables
```

Set the optional file with `BFF_CONFIG_FILE`. `BFF_AUTH__CLIENT_SECRET` is intentionally read only from the process environment and is not loaded from YAML. The complete development contract and defaults are in [`.env.example`](../.env.example).

Configuration is grouped into:

- `BFF_SERVER__*`: listener and server timeouts.
- `BFF_LOG__*`: log level.
- `BFF_CORS__*`: exact browser origins, methods, headers, and credential support.
- `BFF_AUTH__*`: OIDC client, token validation, claims, timeouts, cookies, and token size limits.
- `BFF_PROXY__*`: OpenFGC URL, request and response limits, timeout, passthrough methods, and local placeholder identity and scopes.

Production startup requires authentication, HTTPS authentication URLs, and secure cookies. Placeholder identity cannot run in production and cannot be enabled together with OIDC authentication. In permitted local environments, placeholder mode supplies the configured user and organization without an identity provider. `BFF_PROXY__PLACEHOLDER_SCOPES` is a space-separated list of canonical portal scopes granted to that principal. Unknown or duplicate scopes are rejected, and an empty list creates an authenticated principal without portal permissions. Placeholder identity and scope values must be empty when placeholder mode is disabled.

For authenticated deployments, the organization normally comes from the access-token claim named by `BFF_AUTH__ORG_ID_CLAIM`. Set `BFF_AUTH__ORG_ID_OVERRIDE` to force a fixed organization ID instead; a non-empty override takes precedence and the token does not need the organization claim. Leave the override empty to keep claim-based resolution.

### Local topology

| Service | Typical local address |
| --- | --- |
| React development server | `http://localhost:5173` |
| Portal Backend | `http://localhost:8080` |
| OpenFGC | `http://localhost:8060` when configured through `.env.example` |
| Identity provider | Deployment-specific `BFF_AUTH__ISSUER_URL` |

When the frontend and Portal Backend use different origins, requests include credentials and the frontend origin must be explicitly listed in `BFF_CORS__ALLOWED_ORIGINS`. Credentialed CORS does not allow a wildcard origin. Matching frontend and backend cookie-name configuration is required.

The backend Docker Compose file builds and publishes the Portal Backend on port 8080. OpenFGC and the identity provider must be started or supplied separately.

## Testing

Backend unit and integration tests cover configuration guardrails, OIDC transactions, token splitting and validation, cookie handling, the `GET /me` session response, placeholder-scope filtering, authorization scopes, CORS, current-user ownership, proxy mappings, header sanitization, limits, and upstream failures. Frontend tests cover the `/me` session gate, scope-based routes and elements, sidebar visibility, authorization redirects, auth-client refresh behavior, profile rendering, API integration, and production CSP generation. A frontend test parses the backend OpenAPI document and detects missing, extra, empty, or duplicate portal scopes relative to the centralized `PORTAL_SCOPES` registry.

Use the commands in the [backend README](../README.md) and [frontend README](../../frontend/README.md) to run their respective checks. Changes to routes or scopes must also update and validate the [OpenAPI specification](../openapi/portal-backend.yaml).
