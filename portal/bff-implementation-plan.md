# Portal BFF Implementation Plan
**IdP React SDK · JWT Resource Server · OpenFGC Integration**

---

## 1. Purpose and Architecture

The portal uses an IdP React SDK, behind a portal authentication adapter, for OIDC login, token acquisition, renewal, session handling, and logout. The portal backend is not an OIDC relying party and does not create a browser session. It is a stateless protected API (OAuth 2.0 resource server).

```
┌────────────────────────────────────────────────────┐
│ React Portal                                       │
│ - IdP React SDK via AuthProvider adapter           │
│ - OAuth public client, Authorization Code + PKCE   │
│ - Sends Authorization: Bearer <access_token>       │
└───────────────────────────┬────────────────────────┘
                            │ HTTPS
                            ▼
┌──────────────────────────────────────────────────────┐
│ Portal Backend / BFF (Go, port 8080)                 │
│ - OAuth resource server                              │
│ - Validates JWT access tokens using IdP JWKS         │
│ - Enforces route, scope, organization, and ownership │
│ - Maps approved requests to OpenFGC                  │
└───────────────────────────┬──────────────────────────┘
                            │ private network
                            ▼
┌──────────────────────────────────────────────────────┐
│ OpenFGC server (port 8060)                           │
│ - Internal downstream API                            │
│ - Receives only BFF-derived trusted context headers  │
└──────────────────────────────────────────────────────┘

          ┌──────────────────────────────────────┐
          │ IdP                                  │
          │ Authorization server / OIDC provider │
          └──────────────────────────────────────┘
```

### Responsibilities

| Component | Responsibilities | Must not do |
|---|---|---|
| React Portal | Login callback, token acquisition/renewal, attaching access token, SDK logout | Store a BFF session or make authorization decisions from unvalidated UI data |
| Portal BFF | Validate access tokens, derive principal, authorize requests, protect OpenFGC boundary | Perform OIDC code exchange, store refresh tokens, issue browser auth cookies |
| IdP | Authenticate users and issue tokens | Be bypassed by the portal or BFF |
| OpenFGC | Execute internal consent operations | Be reachable directly from the browser/public network |

### Assumptions

- The selected IdP issues signed JWT access tokens for the portal API audience.
- The React application is configured as an OAuth public client using Authorization Code with PKCE. It has no client secret.
- The selected IdP SDK owns refresh/renewal according to its supported configuration.
- The validated token `sub` is the same identifier OpenFGC uses in `userIds`.
- The BFF and OpenFGC communicate on a private network. OpenFGC is available at `http://localhost:8060` in local development.
- OpenFGC trusts BFF-derived identity/context headers. If OpenFGC later validates user access tokens itself, that is a separate downstream-authentication design decision.

---

## 2. Core Design Decisions

| Decision | Rationale |
|---|---|
| React SDK adapter performs OIDC | Isolates provider SDK details behind a stable portal authentication contract. |
| BFF is a resource server | The BFF validates access tokens directly and remains horizontally scalable without session storage. |
| Bearer token in `Authorization` header | Avoids automatically attached authentication cookies and supports a clear API boundary. |
| JWT validation through issuer discovery/JWKS | Validates tokens from the configured IdP while supporting signing-key rotation. |
| Explicit claim-to-principal mapping | Prevents business authorization from depending on unstructured token payloads. |
| Allowlisted BFF routes | Prevents an unrestricted proxy to OpenFGC. |
| BFF-derived upstream headers only | Browser-provided identity, organization, and client headers are never trusted. |
| OpenFGC private-only exposure | Makes the BFF the mandatory policy-enforcement point. |

---

## 3. Authentication and Request Flows

### 3.1 Provider and SDK Abstraction

The portal must depend on its own `AuthProvider` interface, not directly on a particular provider SDK throughout application code. An IdP-specific adapter implements this interface and is the only frontend location that imports or configures the provider's React SDK.

```
AuthProvider
  - initialize()
  - isAuthenticated()
  - getAccessToken()
  - getUser()
  - login(returnTo?)
  - logout(returnTo?)
  - subscribe(listener)
```

The adapter is responsible for provider-specific initialization, redirect handling, renewal, logout, and claim normalization. Portal components and the API client use only `AuthProvider`. Replacing an IdP therefore requires a new adapter and configuration, rather than application-wide changes.

The BFF remains provider-agnostic by using OIDC discovery, JWKS validation, and configurable claim mappings. A replacement IdP must issue compatible JWT access tokens for the configured audience; provider-specific claim names are translated into the BFF `Principal` model through configuration.

### 3.2 Login and Portal Bootstrap

```
1. User opens the React Portal.
2. The `AuthProvider` adapter checks its SDK-managed authentication state.
3. If required, the adapter starts OIDC Authorization Code + PKCE login with the selected IdP.
4. The user authenticates at the IdP.
5. The IdP redirects to the React application's registered redirect URI.
6. The SDK validates the OIDC response and manages the resulting token lifecycle.
7. The React API client obtains a current access token from the SDK.
8. Each BFF API request includes:
   Authorization: Bearer <access_token>
```

The BFF has no `/auth/login`, `/auth/callback`, `/auth/refresh`, cookie issuance, encrypted state, PKCE verifier, or refresh-token handling.

### 3.3 Protected BFF Request

```
React Portal
  Authorization: Bearer <access_token>
             │
             ▼
Portal BFF
  1. Parse exactly one Bearer Authorization header.
  2. Obtain verification keys from IdP discovery/JWKS cache.
  3. Validate token signature and registered claims.
  4. Map validated claims to an authenticated principal.
  5. Enforce route-level scope and organization policy.
  6. Enforce object ownership for self-service routes.
  7. Strip client-supplied trusted headers and inject BFF-derived context.
  8. Map the allowlisted route to OpenFGC.
             │
             ▼
OpenFGC server :8060
```

### 3.4 Token Validation Requirements

For every protected request, the BFF must validate:

- JWT signature using the configured IdP JWKS and an explicitly allowed signing algorithm.
- `iss` equals `BFF_AUTH__ISSUER_URL`.
- `aud` contains the configured portal API audience.
- `exp`, `nbf` (when present), and `iat` (when required by policy), allowing only configured clock skew.
- A non-empty `sub` claim.
- Token type/purpose only when `BFF_AUTH__REQUIRE_ACCESS_TOKEN_TYPE=true`, so an ID token cannot be accepted as an API access token when the IdP provides a reliable distinguishing claim.
- Required scopes and organization claims for the target route.

The BFF must reject query-string tokens, cookies used as credentials, malformed headers, multiple authorization values, unsigned tokens, unsupported algorithms, unknown issuers/audiences, and expired tokens.

### 3.5 Logout and Expiry

- Logout is performed by the React application through `AuthProvider`.
- The adapter clears SDK-managed state and invokes IdP logout/end-session when supported by the selected SDK/provider configuration.
- The BFF has no session to clear and must not expose a BFF logout endpoint.
- On BFF `401`, the frontend obtains a fresh token through the SDK when possible; otherwise it starts login. It must avoid retry loops.
- On BFF `403`, the frontend shows an authorization error and must not initiate login repeatedly.

---

## 4. API Routes and Authorization

All routes below require a valid bearer access token unless marked otherwise. Authorization is evaluated from the validated principal, never from browser-supplied user or organization values. The BFF authorizes capabilities through scopes; IdP roles are used only to grant scope bundles and are not evaluated by the BFF.

### 4.1 User Endpoints

| Endpoint | Method | Description | Required scope | Additional authorization |
|---|---|---|---|---|
| `/me/consents` | GET | List the authenticated user's consents. | `portal:consents:read:self` | Self-scoped to `UserIdentity.UserID` |
| `/me/consents/{consentId}` | GET | Get an owned consent. | `portal:consents:read:self` | Ownership validation |
| `/me/consents/{consentId}/approve` | POST | Approve an owned consent. | `portal:consents:write:self` | Ownership validation |
| `/me/consents/{consentId}/revoke` | PUT | Revoke an owned consent. | `portal:consents:write:self` | Ownership validation |

Rules:

- `GET /me/consents` maps to `GET /api/v1/consents?userIds=<UserIdentity.UserID>` on OpenFGC.
- Ignore or overwrite every browser-supplied `userIds` value on `/me/*` routes.
- Before reading, approving, or revoking a specific consent, verify ownership against the authenticated `sub`.
- Ignore any request body `userId` that differs from `UserIdentity.UserID`.
- Purpose data fetched internally to enrich `/me/consents` responses is covered by the parent consent scope; it does not grant independent purpose browsing or management.

### 4.2 Controlled `/api/*` Passthrough Endpoints

`/api/{path...}` is the existing controlled passthrough route. It is bearer-authenticated and restricted by the explicit method/path allowlist; it is not an unrestricted proxy.

| BFF route | Methods | Required scope |
|---|---|---|
| `/api/consents` | GET | `portal:consents:read:any` |
| `/api/consents` | POST | `portal:consents:write:any` |
| `/api/consents/attributes` | GET | `portal:consents:read:any` |
| `/api/consents/validate` | POST | `portal:consents:read:any` |
| `/api/consents/{consentId}` | GET | `portal:consents:read:any` |
| `/api/consents/{consentId}` | PUT | `portal:consents:write:any` |
| `/api/consents/{consentId}/revoke` | PUT | `portal:consents:write:any` |
| `/api/consents/{consentId}/authorizations` | GET, POST | `portal:consents:read:any` for GET; `portal:consents:write:any` for POST |
| `/api/consents/{consentId}/authorizations/{authorizationId}` | GET, PUT | `portal:consents:read:any` for GET; `portal:consents:write:any` for PUT |
| `/api/consent-elements` | GET, POST | `portal:elements:read` for GET; `portal:elements:write` for POST |
| `/api/consent-elements/{elementId}` | GET, PUT, DELETE | `portal:elements:read` for GET; `portal:elements:write` for PUT/DELETE |
| `/api/consent-elements/validate` | POST | `portal:elements:read` |
| `/api/consent-purposes` | GET, POST | `portal:purposes:read` for GET; `portal:purposes:write` for POST |
| `/api/consent-purposes/{purposeId}` | GET, PUT, DELETE | `portal:purposes:read` for GET; `portal:purposes:write` for PUT/DELETE |

Rules:

- `:any` permits cross-user consent access only within `UserIdentity.OrgID`; the BFF preserves authorized `userIds` filters and always injects `org-id`.
- `:self` is intentionally available only through `/me/*`; the BFF overwrites the target user with `UserIdentity.UserID`.
- IdP role bundles are: end user receives `:self` scopes; DPO receives consent `:any` scopes; admin receives consent `:any` scopes plus element and purpose scopes.
- Deny unknown `/api/*` paths and methods before proxying.

### 4.3 Scope Enforcement Rules

- The BFF reads the space-delimited `scope` claim only after JWT signature and registered-claim validation.
- Missing, invalid, expired, or wrong-audience tokens return `401`; a valid token missing a required route scope returns `403`.
- Scope values use `:` within a scope name and spaces only to separate distinct scopes, for example: `portal:consents:read:self portal:consents:write:self`.
- The BFF never accepts scopes, user IDs, or organization context from request headers, paths, query parameters, or request bodies.

---

## 5. Middleware and Router Structure

### 5.1 Middleware Order

```
SecurityHeaders
  → CORS
    → RequestID / Logger
      → BearerAuth
        → UserIdentity
          → AuthorizationPolicy
            → Route Handler / OpenFGC Proxy
```

`BearerAuth` must run before `UserIdentity` and all protected route handlers. `UserIdentity` is the only resolver of effective user and organization identifiers; it stores both values in request context. `AuthorizationPolicy` uses the validated principal and route metadata to enforce scopes, organization rules, and ownership requirements.

### 5.2 Bearer Authentication Middleware

1. Require exactly one `Authorization` header matching `Bearer <JWT>`.
2. Fetch provider metadata from the configured issuer discovery document at startup and use its `jwks_uri`.
3. Cache JWKS keys with bounded TTL and refresh the cache when an otherwise valid token references an unknown `kid`.
4. Validate signature, algorithm, issuer, audience, time claims, and subject.
5. Map claims into `Principal` only after successful validation.
6. Store `Principal` in request context; never store the raw access token in context unless a downstream-token-forwarding design explicitly requires it.
7. Return `401` with a generic error response and `WWW-Authenticate: Bearer` for failed authentication.

### 5.3 User Identity Middleware

`internal/system/middleware/identity.go` is the single, central resolver of effective request identity for `/me/*` handlers, services, and proxy transforms.

- It writes one `UserIdentity` value into `internal/system/context`:

  ```
  UserIdentity {
    UserID string // OpenFGC userIds; derived from token sub
    OrgID  string // OpenFGC org-id; derived from token org_id claim
  }
  ```

- In normal operation, it requires the `Principal` placed in request context by `BearerAuth`, derives `UserIdentity` from the validated `Principal.Subject` and `Principal.OrgID`, and writes it through `system/context.WithUserIdentity`.
- In explicit placeholder mode, it derives the same `UserIdentity` from `PlaceholderUserID` and `PlaceholderOrgID`; handlers and services therefore use the identical contract in tests and local development.
- Placeholder mode is permitted only in local/test environments and is rejected during startup when `BFF_ENV=production`; this production safeguard already exists in `internal/system/config/config.go`.
- Placeholder mode must be an explicit configuration choice; it must never activate because authentication failed or because a bearer token is missing.
- Production routes always require bearer authentication; test/local route wiring may use `UserIdentity` with placeholder mode instead of `BearerAuth` only when the explicit non-production configuration is enabled.

Application handlers, services, and proxy transforms read `UserIdentityFromContext` only. `UserIDFromContext` may remain as a convenience wrapper over `UserIdentity.UserID` while existing code is migrated. They must not read JWT claims or placeholder configuration directly.

### 5.4 Authorization Policy

The principal model must contain only normalized, validated information:

```
Principal {
  Subject string
  Scopes  []string
  OrgID   string
}
```

Claim names and mappings are configuration-driven because IdP claim conventions vary. Route policy must distinguish authentication failures (`401`) from authenticated-but-forbidden requests (`403`).

### 5.5 CORS

- Allow only origins in `BFF_CORS__ALLOWED_ORIGINS`; never use `*`.
- Allow `Authorization`, `Content-Type`, and `X-Correlation-ID` request headers.
- Allow only the BFF's supported HTTP methods.
- Return `Vary: Origin`.
- Set `Access-Control-Allow-Credentials: false`; this design does not use browser cookies for authentication.
- Handle preflight `OPTIONS` before bearer authentication.

---

## 6. OpenFGC Proxy Boundary

### 6.1 Request Transformation

For an authorized route, the proxy must:

- Map only approved BFF routes to their corresponding OpenFGC `/api/v1/*` route.
- Preserve approved query parameters and request bodies while applying user-scoping rules.
- Strip hop-by-hop headers: `Connection`, `Keep-Alive`, `Proxy-Authenticate`, `Proxy-Authorization`, `TE`, `Trailer`, `Transfer-Encoding`, and `Upgrade`.
- Remove any browser-supplied `org-id`, `TPP-client-id`, user, or identity headers.
- Inject only the trusted context explicitly defined in §6.2 from the validated principal and route policy.
- Propagate a validated `X-Correlation-ID`, or generate one when absent.
- Enforce request-size limits, downstream timeouts, and a conservative retry policy appropriate for idempotency.

### 6.2 Trusted Context Mapping

The BFF strips any browser-supplied values for the trusted headers below, then derives them only from validated request context.

| BFF source | OpenFGC target | Rule |
|---|---|---|
| `UserIdentity.UserID` | `userIds` query parameter | Set only for `/me/*` routes; overwrite all browser-supplied `userIds`. It is also used for user-owned authorization and action payloads. |
| `UserIdentity.OrgID` | `org-id` header | Set on every OpenFGC request. Reject the request if the required organization identity is absent. |
| Fetched OpenFGC consent `clientId` | `TPP-client-id` header | Strip any browser value. The `/me/consents/{consentId}/approve` flow may set this header only from the consent fetched from OpenFGC; it is never derived from a token, placeholder identity, or request header. |
| Validated incoming correlation ID, or generated ID | `X-Correlation-ID` header | Propagate only after format/length validation; otherwise generate a new ID. |

The BFF never forwards the browser `Authorization` header or a browser-supplied trusted-context header to OpenFGC.

### 6.3 Downstream Authentication

This plan uses a header-trust boundary: OpenFGC trusts the BFF's injected context, not client-provided headers. The BFF does not forward the browser access token to OpenFGC by default.

If OpenFGC must validate access tokens in the future, explicitly add one of these designs before forwarding credentials:

- token forwarding with OpenFGC validation of the same audience/token contract; or
- OAuth token exchange / service-to-service credentials for the OpenFGC audience.

Do not forward a token merely because it is available; its audience and downstream authorization meaning must be correct.

---

## 7. Security Requirements

### 7.1 Browser and Token Handling

- The frontend must obtain tokens through `AuthProvider` and attach them only to trusted BFF origins.
- Do not put access tokens in URLs, application logs, analytics events, error messages, or telemetry attributes.
- Do not accept tokens from query parameters, form fields, or cookies.
- Use HTTPS in staging and production for both portal-to-BFF and BFF-to-IdP traffic.
- Keep access-token lifetime, renewal, and storage behavior aligned with the selected SDK guidance and IdP policy.

### 7.2 JWT/JWKS Safety

- Pin the expected issuer and audience; do not infer either from untrusted token claims.
- Permit only explicitly configured asymmetric signing algorithms.
- Use the issuer's discovered JWKS URI; do not accept a JWKS URL from the token.
- Bound JWKS cache lifetime and apply HTTP timeouts. Retain and use the last known-good keys during transient discovery/JWKS retrieval failures, provided they still validate the token; do not discard a usable cache merely because refresh failed.
- Refresh JWKS on an unknown `kid` before rejecting a token, supporting normal IdP key rotation.
- Fail closed if discovery/JWKS integrity cannot be established and no valid cached key can validate the token.

### 7.3 CSRF and XSS

CSRF middleware and CSRF cookies are not required for bearer authentication because browsers do not automatically attach the `Authorization` header cross-site. The portal must still maintain a strong CSP and XSS protections: script injection can misuse an in-memory token.

### 7.4 Security Headers and Caching

Apply to BFF responses:

```
Content-Security-Policy: default-src 'none'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Cache-Control: no-store
Pragma: no-cache
```

The BFF returns JSON and redirects only; the React portal has its own CSP.

---

## 8. Project Structure

```
portal/backend/
├── cmd/server/
│   ├── main.go
│   └── servicemanager.go
├── internal/
│   ├── me/
│   │   ├── init.go             # module initialization and /me route registration
│   │   ├── handler.go          # /me request handlers
│   │   └── service.go          # self-service consent operations
│   ├── proxy/
│   │   ├── init.go             # module initialization and /api route registration
│   │   ├── handler.go          # proxy HTTP handler
│   │   ├── service.go          # OpenFGC request transformation/proxying
│   │   └── service_test.go
│   └── system/
│       ├── auth/
│       │   ├── validator.go    # issuer discovery, JWKS cache, JWT validation
│       │   ├── principal.go    # validated claim-to-Principal mapping
│       │   └── policy.go       # scope, organization, and ownership helpers
│       ├── config/
│       │   ├── config.go       # configuration loading, defaults, validation
│       │   └── config_test.go
│       ├── context/
│       │   └── identity.go     # request-scoped Principal and UserIdentity accessors
│       ├── correlation/
│       │   └── id.go           # correlation-ID helpers
│       ├── healthcheck/
│       │   └── handler.go
│       ├── middleware/
│       │   ├── bearer_auth.go  # invokes system/auth validation and stores Principal
│       │   ├── correlationid.go
│       │   ├── correlationid_test.go
│       │   ├── cors.go
│       │   ├── cors_test.go
│       │   ├── identity.go      # central UserIdentity resolver: token claims or test placeholders
│       │   └── identity_test.go
│       └── log/
│           ├── log.go
│           └── log_test.go
├── tests/
│   └── integration/
│       ├── health_test.go
│       ├── proxy_invalid_consent_id_test.go
│       ├── proxy_phase2_test.go
│       └── server_test.go
├── openapi/
│   └── bff.yaml
├── README.md
├── Taskfile.yml
├── go.mod
├── Dockerfile
├── docker-compose.yml
└── go.sum
```

Authentication is added within the existing `internal/system` foundation. `internal/system/auth` owns provider discovery, JWKS/JWT validation, normalized principal creation, and authorization helpers. `internal/system/middleware/bearer_auth.go` applies that logic to HTTP requests and stores the validated `Principal` in `internal/system/context`. `internal/system/middleware/identity.go` then resolves and stores the single `UserIdentity` (`UserID` + `OrgID`) from either validated claims or explicit non-production placeholders.

Add unit tests beside the package under test and extend `tests/integration` for end-to-end bearer-authentication and proxy scenarios.

---

## 9. Configuration

Use Koanf with environment variables as the primary runtime source and optional file overlays for local development.

```bash
# Server
BFF_SERVER__PORT=8080
BFF_ENV=development
BFF_LOG__LEVEL=info

# IdP resource-server validation
BFF_AUTH__ISSUER_URL=https://localhost:9443
BFF_AUTH__RESOURCE_AUDIENCE=portal-api
BFF_AUTH__ALLOWED_SIGNING_ALGORITHMS=RS256
BFF_AUTH__JWKS_CACHE_TTL=15m
BFF_AUTH__JWKS_HTTP_TIMEOUT=5s
BFF_AUTH__JWT_CLOCK_SKEW_SECONDS=60
# Disabled unless the IdP emits a reliable access-token-type claim.
BFF_AUTH__REQUIRE_ACCESS_TOKEN_TYPE=false
BFF_AUTH__ACCESS_TOKEN_TYPE_CLAIM=token_type
BFF_AUTH__ACCESS_TOKEN_TYPE_VALUE=access_token

# Claim mapping
BFF_AUTH__SCOPE_CLAIM=scope
BFF_AUTH__ORG_ID_CLAIM=org_id

# OpenFGC server
BFF_PROXY__OPENFGC_API_URL=http://localhost:8060
BFF_PROXY__OPENFGC_API_TIMEOUT=10s

# CORS
BFF_CORS__ALLOWED_ORIGINS=http://localhost:3000
BFF_CORS__ALLOWED_HEADERS=Authorization,Content-Type,X-Correlation-ID
BFF_CORS__ALLOW_CREDENTIALS=false

# Proxy safety
BFF_PROXY__MAX_REQUEST_BYTES=1048576

# Test/local identity only — startup must reject this in production
BFF_PROXY__PLACEHOLDER_MODE_ENABLED=false
BFF_PROXY__PLACEHOLDER_USER_ID=
```

Do not configure a BFF OIDC client secret, callback URI, cookie names, cookie keys, CSRF key, state-encryption key, refresh-token settings, or BFF-generated signing keys.

The React application's IdP SDK configuration belongs in `portal/frontend`, inside the selected `AuthProvider` adapter. It includes the IdP issuer, public client ID, registered React redirect URI, requested scopes, and API audience/resource configuration. No portal component outside the adapter should import the provider SDK directly.

---

## 10. Deployment Notes

### 10.1 Network Boundary

- Expose the React portal and BFF through the intended ingress only.
- Do not expose OpenFGC port `8060` to browsers or public networks.
- Permit inbound OpenFGC traffic only from the BFF/private network identity.
- Strip trusted context headers at ingress so only the BFF can set them.
- Prefer mTLS or equivalent service-to-service authentication between BFF and OpenFGC.

### 10.2 Availability and Rate Limiting

- Rate-limit protected API routes and malformed/failed-authentication requests at the gateway or reverse proxy.
- Apply stricter limits to expensive mutation endpoints.
- Configure timeouts for IdP discovery/JWKS and OpenFGC calls.
- Monitor JWKS refresh failures, authentication failure reason codes, authorization denials, and OpenFGC latency without recording token material.

### 10.3 Key Rotation

The IdP owns signing-key rotation. The BFF supports it by respecting JWKS cache lifetime and refreshing JWKS when it encounters an unknown `kid`. There are no BFF JWT-signing, JWE, HMAC, or cookie-encryption keys to rotate.

### 10.4 Local Development

- Portal: `http://localhost:3000`
- BFF: `http://localhost:8080`
- IdP: `https://localhost:9443`
- OpenFGC: `http://localhost:8060`

Trust the local IdP certificate through the container/host trust store. Do not use `InsecureSkipVerify` outside an explicitly isolated local-development setup.

---

## 11. Implementation Checklist

### React Portal

- [x] Define the portal `AuthProvider` interface and use it from all portal components and API clients.
- [x] Implement the selected IdP React SDK adapter as an OAuth public client using Authorization Code + PKCE.
- [x] Register the React redirect URI and post-logout redirect URI with the selected IdP.
- [x] Configure the API audience/resource and requested scopes.
- [x] Add an API client/interceptor that obtains a current `AuthProvider` access token and sets `Authorization: Bearer <token>`.
- [x] Handle `401` by using `AuthProvider` token renewal/login without infinite retries.
- [x] Handle `403` as an authorization failure.
- [x] Use `AuthProvider` logout; remove calls to BFF login, refresh, callback, and logout endpoints.

### BFF Authentication and Authorization

- [x] Implement issuer discovery and JWKS caching.
- [x] Implement strict bearer-header parsing.
- [x] Validate signature, allowed algorithm, issuer, audience, time claims, and subject; validate access-token type only when explicitly enabled and supported by the IdP.
- [x] Refresh JWKS on unknown `kid` and support normal key rotation.
- [x] Map validated claims to the normalized `Principal` model.
- [x] Add `UserIdentity` context accessors and update identity middleware to resolve both `Principal.Subject` and `Principal.OrgID` into one request-scoped `UserIdentity`.
- [x] Retain explicit placeholder identity mode only for test/local use; reject it in production and never use it as an authentication fallback.
- [x] Configure and enforce the §4 scope, organization, and ownership policy for both `/me/*` and `/api/*` routes.
- [x] Return `401` with `WWW-Authenticate: Bearer` for authentication failures and `403` for authorization failures.

### BFF Routes and Proxy

- [x] Enforce self-scoping and object ownership on `/me/*` routes.
- [x] Enforce the §4 scope-to-route policy for the existing controlled `/api/*` passthrough route.
- [x] Update `openapi/bff.yaml`: define the OAuth2 security scheme and all scope descriptions, then declare each operation's required scope with OpenAPI `security` requirements.
- [x] Maintain explicit method/path mappings to OpenFGC `/api/v1/*`.
- [x] Point `BFF_PROXY__OPENFGC_API_URL` to port `8060`.
- [x] Strip client-supplied trusted headers; inject user and organization context only from `UserIdentity`.
- [x] Apply body limits, timeouts, correlation IDs, and hop-by-hop header stripping.

### Security and Operations

- [x] Configure explicit CORS origins and permit the `Authorization` header.
- [x] Set CORS credentials to false.
- [ ] Apply security and no-store cache headers.
- [ ] Ensure token redaction in application, proxy, and observability logs.
- [ ] Make OpenFGC port `8060` private-only and enforce BFF-only access.
- [ ] Add infrastructure rate limiting and monitoring for failed authentication/JWKS availability.

### Testing

- [x] Unit tests: valid token, missing/malformed bearer header, invalid signature, unsupported algorithm, expired/not-yet-valid token, wrong issuer, wrong audience, missing subject, and wrong token type.
- [x] Unit tests: JWKS cache hits, unknown-`kid` refresh, rotation, and transient JWKS failure behavior.
- [x] Unit tests: claim mapping, scope policy, organization policy, and ownership policy.
- [x] Integration tests: portal-style bearer request to every protected route; no cookies required.
- [x] Integration tests: `401` versus `403` behavior and `WWW-Authenticate` response.
- [x] Integration tests: CORS preflight and `Authorization` header allowance.
- [x] Integration tests: OpenFGC path mapping, query preservation, port-8060 target, and header override prevention.
- [x] Integration tests: `/me/consents` always injects `UserIdentity.UserID` and ignores client user filters; `/api/*` rejects missing `:any` scopes and preserves authorized filters only within the injected organization.

---

## 12. Standards

- HTTP stack: Go `net/http` with `ServeMux` and composable middleware.
- Configuration: centralized Koanf loading and startup validation.
- API design: contract-first route and payload definitions; deny-by-default proxy routing.
- Security model: standards-based JWT resource-server validation, least-privilege scope capabilities, private downstream boundary, and externalized secrets/configuration.
- Testing model: unit tests for validation/policy logic and integration tests for browser-to-BFF bearer authentication and BFF-to-OpenFGC transformations.
