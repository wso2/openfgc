# Stateless BFF Implementation Plan
**OAuth/OIDC · Portal · OpenFGC Integration (IdP-Agnostic)**

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Core Design Decisions](#2-core-design-decisions)
3. [Split Cookie Strategy](#3-split-cookie-strategy)
4. [Authentication Flows](#4-authentication-flows)
5. [Endpoints & Routes](#5-endpoints--routes)
6. [Middleware Stack](#6-middleware-stack)
7. [Core Functions & Signatures](#7-core-functions--signatures)
8. [Security](#8-security)
9. [Project Structure](#9-project-structure)
10. [Configuration](#10-configuration)
11. [Deployment Notes](#11-deployment-notes)
12. [Implementation Checklist](#12-implementation-checklist)
13. [Patterns, Practices, and Standards](#13-patterns-practices-and-standards)

---

## 1. Architecture Overview

```
┌──────────────────────────┐
│   React Portal           │
│   (portal/frontend)      │
└──────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────────┐
│  Stateless BFF (Go)                          Port: 8080  │
│  (portal/backend)                                        │
│                                                          │
│  ✅ No sessions / no state storage                      │
│  ✅ Cookie 1 — auth_token    (JWS, RS256)               │
│  ✅ Cookie 2 — session_token (JWE, AES-256-GCM)         │
│  ✅ HttpOnly + SameSite=Strict on both cookies          │
│  ✅ Stateless CSRF (HMAC double-submit)                 │
│  ✅ OIDC flow with external IdP                         │
│  ✅ Request proxying to OpenFGC                         │
└──────────────────────────────────────────────────────────┘
           │
     ┌─────┴──────┐
     ▼            ▼
┌──────────┐  ┌────────────────────┐
│  OIDC IdP│  │  OpenFGC Backend   │
│          │  │  (/consent-server) │
│          │  │  Port: 9090        │
└──────────┘  └────────────────────┘
```

### Assumptions

- Repository paths used by this plan:
  - React portal: `portal/frontend`
  - Stateless BFF: `portal/backend`
  - OpenFGC backend: `/consent-server`

- The selected OIDC provider is configured to issue **JWT access tokens** (not opaque). This must be confirmed before implementation — see [§8.6](#86-provider-token-format-assumption).
- The selected OIDC provider is configured to issue **refresh tokens** to this client (typically requires `offline_access` scope and provider policy permitting refresh issuance).
- Rate limiting on auth endpoints is enforced at the **infrastructure layer** (gateway / reverse proxy), not in the BFF — see [§11.1](#111-rate-limiting).
- Refresh token race conditions across BFF replicas use a single v1 policy: accept `401` on refresh collision and force re-login — see [§11.2](#112-refresh-race-condition).
- Local development runs all browser-facing services on `localhost` with different ports:
  - Portal: `http://localhost:3000`
  - BFF: `http://localhost:8080`
  - OIDC IdP: `https://localhost:9443`

---

## 2. Core Design Decisions

| Decision | Rationale |
|----------|-----------|
| Split cookies (JWS + JWE) | Encrypt only what needs to be secret. Identity claims are not sensitive; bearer tokens are. |
| RS256 for auth_token | Asymmetric signing — verification uses public key only. |
| AES-256-GCM for session_token | Hardware-accelerated (~1µs/op). Authenticated encryption — confidentiality and integrity in one operation. |
| HttpOnly on both cookies | JS cannot access auth_token or session_token regardless of content. |
| SameSite=Strict on all cookies | Primary CSRF defence. Blocks cross-site cookie submission without server state. |
| HMAC-signed CSRF token | Stronger than random double-submit — prevents subdomain cookie injection. No server storage required. |
| Encrypted OIDC state (AEAD) | Keeps PKCE verifier and nonce confidential in the front channel while staying stateless. |
| RP-initiated logout | Terminates IdP session on logout when provider supports end-session. Prevents silent re-authentication. |
| 15-minute token expiry | Limits blast radius of cookie theft. Short enough to matter, long enough not to force constant refresh. |
| Versioned encryption keys | Allows key rotation without mass logout — old keys retained for one TTL window during transition. |
| No server-side session store | Enables horizontal scaling with no coordination. Tradeoff: no per-token revocation (see [§8.5](#85-known-tradeoff-revocation)). |
| Go 1.22 stdlib mux | Go 1.22 stdlib ServeMux supports method routing and path parameters natively. |

---

## 3. Split Cookie Strategy

### 3.1 Why Split?

The original design stored the OpenFGC bearer token inside a single JWE cookie. While encrypted, this created a single high-value target and made per-token revocation impossible. Splitting separates concerns: identity (low sensitivity, needs integrity) from token material (high sensitivity, needs confidentiality).

### 3.2 Cookie 1 — `auth_token` (JWS)

Contains user identity claims only. The payload is not secret — the user already knows their own identity. The security requirement is **integrity**, not confidentiality: we must prevent the user from tampering with their roles or subject identifier. A signed JWT (RS256) achieves this without encryption overhead.

```
Format:  JWS (signed JWT, RS256)
Storage: HttpOnly cookie
Expiry:  15 minutes

Payload:
{
  "sub":   "user123",
  "email": "user@example.com",
  "name":  "John Doe",
  "roles": ["user", "admin"],
  "iat":   1711836000,
  "exp":   1711836900,
  "iss":   "http://localhost:8080",
  "aud":   "portal"
}

Cookie config:
  Name:     auth_token
  Path:     /
  HttpOnly: true
  Secure:   true
  SameSite: Strict
  MaxAge:   900
```

### 3.3 Cookie 2 — `session_token` (JWE)

Contains high-sensitivity token material used by the BFF: OIDC refresh token (and optional provider session metadata). OIDC ID token is intentionally not stored in `session_token` to keep cookie size within browser-safe limits. If a user could read this cookie and extract these values, they could mint fresh provider tokens outside the BFF. **Encryption is required here.** The `kid` header field carries the key version to support rotation without mass logout.

```
Format:  JWE (AES-256-GCM encrypted, kid-versioned)
Storage: HttpOnly cookie
Expiry:  Matches OpenFGC token TTL (15 minutes)

Payload:
{
  "refresh_token":  "idp_refresh_token_abc...",
  "exp":            1711836900
}

JWE Header:
{
  "alg": "A256GCMKW",
  "enc": "A256GCM",
  "kid": "2"          ← key version for rotation
}

Cookie config:
  Name:     session_token
  Path:     /
  HttpOnly: true
  Secure:   true
  SameSite: Strict
  MaxAge:   900
```

### 3.4 Cookie 3 — `csrf_token` (non-HttpOnly)

Not a bearer credential — contains only a derived HMAC token. Must be readable by JS so the portal can include it in the `X-CSRF-Token` header.

```
Format:  HMAC-SHA256(sub + iat, csrfHMACKey)
Storage: Non-HttpOnly cookie (intentionally JS-readable)
Expiry:  15 minutes

Cookie config:
  Name:     csrf_token
  Path:     /
  HttpOnly: false    ← intentional
  Secure:   true
  SameSite: Strict
  MaxAge:   900
```

### 3.5 Cookie Comparison

| Property | auth_token | session_token | csrf_token |
|----------|-----------|---------------|------------|
| Contents | sub, email, roles, exp | refresh_token, exp | HMAC-derived token |
| Sensitivity | Low | High | None |
| Format | JWS (RS256) | JWE (AES-256-GCM) | Plain string |
| Encrypted | No | Yes | No |
| Signed | Yes (RS256) | Yes (implicit in JWE) | Yes (HMAC) |
| HttpOnly | Yes | Yes | **No** |
| SameSite | Strict | Strict | Strict |
| Expiry | 15 min | Matches OpenFGC TTL | 15 min |
| Key versioned | Yes (kid in header) | Yes (kid in header) | No |

Session cookie size guard (required):
- Before setting `session_token`, compute the final serialized cookie length (value + attributes).
- Enforce a hard cap via `SESSION_COOKIE_MAX_BYTES` (recommended: 3500 bytes).
- If size exceeds cap, fail closed (clear auth cookies and return 401) instead of writing an oversized cookie.

### 3.6 Why Not Encrypt Cookie 1?

Cookie 1 carries identity claims that are not confidential; integrity is the requirement. Signing with RS256 provides tamper protection. Cookie 2 still requires encryption because it contains bearer material.

### 3.7 Recommended Cookie Prefix Hardening

For production deployments, prefer the `__Host-` cookie prefix for all BFF cookies (for example: `__Host-auth_token`, `__Host-session_token`, `__Host-csrf_token`).

`__Host-` requires all of the following:
- `Secure=true`
- `Path=/`
- No `Domain` attribute

These constraints prevent accidental subdomain scoping and reduce cookie injection risk from sibling subdomains.

If `__Host-` naming is enabled, `COOKIE_DOMAIN` must remain empty in all environments.

---

## 4. Authentication Flows

### 4.1 OIDC Login

```
1.  User visits http://localhost:3000
2.  Portal calls GET /auth/me
3.  If `/auth/me` returns 401 → GET /auth/login
4.  If `/auth/me` returns 200 → continue normally
5.  BFF generates PKCE + nonce:
      code_verifier  = random(32 bytes, base64url)
      code_challenge = base64url(SHA256(code_verifier))
      oidc_nonce     = random(16 bytes, base64url)
6.  BFF generates encrypted state carrying callback-bound values:
  state = Encrypt(state_payload{timestamp, code_verifier, oidc_nonce}, STATE_ENC_KEY)
7.  BFF redirects to OIDC provider authorize endpoint:
  <oidc_authorization_endpoint>
        ?client_id=bff_client
        &redirect_uri=http://localhost:8080/auth/callback
        &response_type=code
        &scope=openid profile email offline_access
        &nonce=<oidc_nonce>
        &code_challenge=<code_challenge>
        &code_challenge_method=S256
        &state=<encrypted_state>
      8.  User authenticates at IdP
      9.  IdP redirects: /auth/callback?code=...&state=...
      10. BFF validates state:
  - Decrypt and authenticate with STATE_ENC_KEY
  - Extract timestamp, code_verifier, oidc_nonce
  - Reject if decrypt/authentication fails
      - Reject if timestamp older than STATE_MAX_AGE_MINUTES
      11. BFF exchanges code for tokens with PKCE verifier (backend call — never exposed to client)
      12. BFF fully validates ID token (signature via JWKS, `iss`, `aud`, `exp`/`iat`/`nbf`, and `nonce == oidc_nonce`)
      13. BFF issues Cookie 1 (auth_token, JWS) with identity claims
      14. BFF issues Cookie 2 (session_token, JWE) with refresh-token material
      15. BFF issues Cookie 3 (csrf_token, non-HttpOnly) with HMAC token
      16. BFF redirects to Portal
      17. Portal reads csrf_token cookie, stores value for subsequent request headers
```

### 4.2 Authenticated Request

```
Portal → BFF
  Cookie:       auth_token=<jws>; session_token=<jwe>; csrf_token=<hmac>
  X-CSRF-Token: <hmac>

BFF middleware:
  1. Verify auth_token RS256 signature → extract identity claims
  2. Decrypt session_token (kid → select key) → ensure valid BFF session material
  3. Validate CSRF (all non-safe methods: anything except GET/HEAD/OPTIONS/TRACE): cookie/header match + HMAC recomputation
     using trusted auth claims (`sub`, `iat`) in constant time
  4. Derive trusted upstream headers from validated identity/context:
     - org-id
     - TPP-client-id (for create/update routes)
     - X-Correlation-ID (propagate or generate)
  5. Rewrite path /api/... → /api/v1/... before proxying to consent-server

Proxy → OpenFGC (port 9090)
  ↓
Response returned to Portal
```

### 4.3 Token Refresh

```
Portal → POST /auth/refresh
  X-CSRF-Token: <hmac>
  Cookie: auth_token=<jws>; session_token=<jwe>

BFF:
  1. Validate CSRF: cookie/header match + HMAC recomputation
    using signed auth_token claims (`sub`, `iat`)
  2. Decrypt session_token → extract refresh_token
  3. Validate auth_token signature/sub consistency;
    allow expiry only within a small grace window (recommended: 5 minutes)
  4. POST <oidc_token_endpoint>
       grant_type=refresh_token
       refresh_token=<refresh_token>
      5. Receive refreshed provider tokens (+ new refresh_token if rotation enabled)
      6. Re-validate identity source before reissuing cookies:
        - preferred: validate returned `id_token` (signature/JWKS, iss, aud, exp/iat/nbf)
        - fallback: call UserInfo with refreshed access token and verify `sub` consistency
      7. Reissue session_token (JWE) with updated refresh-token material
      8. Reissue auth_token (JWS) on every successful refresh to reset expiry
      9. Reissue csrf_token using new auth claims (`sub`, `iat`) whenever auth_token is reissued
      10. Return 200 — portal continues with updated cookies

On invalid_grant from IdP:
  → Clear all cookies
  → Return 401
  → Portal redirects to /auth/login
```

> **Note:** The race condition that can occur when multiple BFF replicas simultaneously refresh the same token is addressed at deployment time. See [§11.2](#112-refresh-race-condition).

### 4.4 Logout

```
Portal → POST /auth/logout
  Cookie: auth_token/session_token/csrf_token (as available)

BFF:
  1. Validate same-origin request using strict algorithm:
    - If `Origin` is present: require origin to match one configured value in `PORTAL_ALLOWED_ORIGINS`
    - Else if `Referer` is present: parse and require referer origin to match one configured value in `PORTAL_ALLOWED_ORIGINS`
    - Else: reject request (403)
  2. Clear cookies using the same attributes used at issuance (name/path/domain/samesite/secure)
     so browser deletion is reliable across environments.
     Examples:
     - Set-Cookie: auth_token=;    Path=/; Domain=<COOKIE_DOMAIN when set>; MaxAge=-1; HttpOnly; Secure; SameSite=Strict
     - Set-Cookie: session_token=; Path=/; Domain=<COOKIE_DOMAIN when set>; MaxAge=-1; HttpOnly; Secure; SameSite=Strict
     - Set-Cookie: csrf_token=;    Path=/; Domain=<COOKIE_DOMAIN when set>; MaxAge=-1; Secure; SameSite=Strict
  3. If provider exposes `end_session_endpoint`, call RP-initiated logout endpoint:
      GET <oidc_end_session_endpoint>
         ?post_logout_redirect_uri=http://localhost:3000/login
  4. Redirect to http://localhost:3000/login

Note: `auth_token` is not required for logout. CSRF header is not required for this endpoint.
Note: `id_token_hint` is intentionally omitted because `id_token` is not persisted in `session_token` (cookie-size control).
Provider-compatibility note: if an IdP requires `id_token_hint` for complete RP-initiated logout, fall back to local cookie clear + front-channel redirect to portal login.
```

---

## 5. Endpoints & Routes

### 5.1 Authentication Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/auth/login` | GET | Generates encrypted state, redirects to OIDC provider | No |
| `/auth/callback` | GET | Validates state, exchanges code, issues all cookies | No |
| `/auth/me` | GET | Returns authenticated user identity for portal bootstrap (200/401) | `auth_token` |
| `/auth/logout` | POST | Validates same-origin (`Origin`/`Referer`), clears cookies, calls IdP end-session endpoint when available | Same-origin check (auth_token not required) |
| `/auth/refresh` | POST | Decrypts session_token, refreshes via IdP token endpoint, reissues cookies | Session + CSRF + signed auth_token (expiry allowed only in short grace window) |

### 5.2 User Endpoints (BFF-defined, portal-facing)

These endpoints are purpose-built for data owners (end users). They are not generic passthrough routes.

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/me/consents` | GET | Self-consent list endpoint. BFF always enforces user scope. | Yes |
| `/me/consents/{consentId}` | GET | Retrieve one consent only if owned by authenticated user. | Yes |
| `/me/consents/{consentId}/approve` | POST | Approve consent for the authenticated user and persist approval decision. | Yes |
| `/me/consents/{consentId}/revoke` | PUT | Revoke one consent only if owned by authenticated user. | Yes |

User mapping notes:
- `GET /me/consents` maps to upstream `GET /api/v1/consents?userIds=<principal-sub>`
- Any browser-supplied `userIds` is ignored/overwritten for user role requests
- `POST /me/consents/{consentId}/approve` maps to upstream consent approval operations (`PUT /api/v1/consents/{consentId}` and/or `POST /api/v1/consents/{consentId}/authorizations`) with `status=APPROVED`
- `GET /me/consents/{consentId}` and `PUT /me/consents/{consentId}/revoke` require ownership validation before proxying
- `POST /me/consents/{consentId}/approve` requires ownership validation and must ignore any client-supplied `userId` that does not match principal `sub`

### 5.3 Admin Endpoints (BFF-defined + controlled 1:1 passthrough)

These endpoints are for data holder admins. They expose broader search/management capability and may call consent-server routes in a 1:1 pattern after policy checks.

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/admin/consents` | GET, POST | List / create consents with admin scope. | Admin |
| `/admin/consents/attributes` | GET | Search consents by attribute key/value. | Admin |
| `/admin/consents/validate` | POST | Validate consent access payloads. | Admin |
| `/admin/consents/{consentId}` | GET, PUT | Get / update consent by ID. | Admin |
| `/admin/consents/{consentId}/revoke` | PUT | Revoke consent by ID. | Admin |
| `/admin/consents/{consentId}/authorizations` | GET, POST | List / create authorization resources. | Admin |
| `/admin/consents/{consentId}/authorizations/{authorizationId}` | GET, PUT | Get / update authorization resource. | Admin |
| `/admin/consent-elements` | GET, POST | List / create consent elements. | Admin |
| `/admin/consent-elements/{elementId}` | GET, PUT, DELETE | Get / update / delete consent element. | Admin |
| `/admin/consent-elements/validate` | POST | Validate consent element names. | Admin |
| `/admin/consent-purposes` | GET, POST | List / create consent purposes. | Admin |
| `/admin/consent-purposes/{purposeId}` | GET, PUT, DELETE | Get / update / delete consent purpose. | Admin |

Admin mapping notes:
- `/admin/*` routes map to consent-server `/api/v1/*` using explicit method/path allowlist
- BFF can preserve admin filters (including `userIds`) only after admin-role authorization passes
- Catch-all passthrough is not exposed to users; deny-by-default outside allowlisted routes

### 5.4 Authorization and Routing Rules

- Two actor types are enforced in BFF: data owner user and data holder admin
- User endpoints are self-scoped by default and never trust browser-provided scope
- Admin endpoints allow organization-wide access only after explicit role checks
- Object-level ownership checks are mandatory for user routes targeting a specific consent ID
- BFF strips untrusted client headers and re-creates trusted upstream headers (`org-id`, `TPP-client-id`, `X-Correlation-ID`) from validated context
- Path rewrite contract:
  - User route group: `/me/*` -> mapped by BFF to selected `/api/v1/*` upstream routes
  - Admin route group: `/admin/*` -> mapped by BFF to allowlisted `/api/v1/*` upstream routes

---

## 6. Middleware Stack

### 6.1 Router Structure

Auth routes and protected routes are registered on **separate mux instances**. This is structural safety — the middleware stack physically cannot intercept auth routes, regardless of registration order or Go mux precedence rules.

To keep implementation lean, use a single `RegisterRoutes(...)` function and three focused tests:
- `/auth/login` and `/auth/callback` are not wrapped by auth/session/csrf middleware
- `/auth/me` is wrapped only by `Auth` middleware
- `/api/*` is always wrapped by `Auth` + `Session` + `CSRF`

```go
// Base mux — public auth routes (no auth/session/csrf middleware)
mux := http.NewServeMux()
mux.HandleFunc("GET /auth/login",      authHandler.Login)
mux.HandleFunc("GET /auth/callback",   authHandler.Callback)

// /auth/me mux — lightweight auth bootstrap endpoint for portal
meMux := http.NewServeMux()
meMux.HandleFunc("GET /auth/me", authHandler.Me)
me := AuthMiddleware(meMux)
mux.Handle("/auth/me", me)

// Logout mux — same-origin-protected logout without auth-token requirement
logoutMux := http.NewServeMux()
logoutMux.HandleFunc("POST /auth/logout", authHandler.Logout)
logout := OriginCheckMiddleware(logoutMux)
mux.Handle("/auth/logout", logout)

// Refresh mux — dedicated protection for token rotation
refreshMux := http.NewServeMux()
refreshMux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
refresh := SessionMiddleware(
             CSRFMiddleware(refreshMux))
mux.Handle("/auth/refresh", refresh)

// Protected mux — all routes require full middleware stack
protectedMux := http.NewServeMux()
protectedMux.HandleFunc("/api/{path...}", proxyHandler.Handle)

// Wrap protected mux — order matters
protected := AuthMiddleware(
               SessionMiddleware(
                 CSRFMiddleware(protectedMux)))

mux.Handle("/api/", protected)

// Apply global middleware so every route (including /auth/login/callback/logout)
// gets CORS, logs, and security headers.
root := SecurityHeadersMiddleware(
          CORSMiddleware(
            LoggerMiddleware(mux)))
// http.ListenAndServe(":8080", root)
```

### 6.2 Request Flow

```
Incoming Request
  │
  ▼
Security Headers Middleware
  Adds: Content-Security-Policy, X-Content-Type-Options,
        X-Frame-Options, Referrer-Policy
  │
  ▼
CORS Middleware
  Access-Control-Allow-Origin:      <request-origin-if-in-PORTAL_ALLOWED_ORIGINS>
  Access-Control-Allow-Credentials: true              (required for cookies)
  OPTIONS preflight handled here
  │
  ▼
Logger Middleware
  │
  ├─── Auth Routes (/auth/*) ────────────────────────────────────────────────┐
  │    Route-specific middleware only                                         │
  │    GET  /auth/login                                                       │
  │    GET  /auth/callback                                                    │
  │    GET  /auth/me  (Auth middleware only)                                  │
  └───────────────────────────────────────────────────────────────────────────┘
  │
  ├─── Logout Route (/auth/logout) ──────────────────────────────────────────┐
  │    Same-origin Middleware (`Origin`/`Referer`)                            │
  └───────────────────────────────────────────────────────────────────────────┘
  │
  ├─── Refresh Route (/auth/refresh) ────────────────────────────────────────┐
  │    Session Middleware                                                     │
  │    CSRF Middleware                                                        │
  │    Mandatory auth_token sub/signature consistency check                   │
  │    Strict error mapping (invalid_grant => clear cookies + 401)           │
  │    No internal retry loop                                                 │
  └───────────────────────────────────────────────────────────────────────────┘
  │
  ▼
Auth Middleware  (/api/*)
  1. Extract auth_token cookie
  2. Verify RS256 signature
  3. Check expiry → 401 if expired
  4. Extract identity claims → attach to request context
  │
  ▼
Session Middleware
  1. Extract session_token cookie
  2. Read kid from JWE header → select decryption key
  3. AES-256-GCM decrypt → extract refresh/session material
  4. If kid != active version → reissue cookie with current key on response
  │
  ▼
CSRF Middleware  (all non-safe methods: except GET, HEAD, OPTIONS, TRACE)
  1. Extract X-CSRF-Token header
  2. Extract csrf_token cookie value
  3. Require header == cookie value (double-submit)
  4. Recompute expected HMAC from trusted auth claims (`sub`, `iat`)
     and compare in constant time → 403 if mismatch
  │
  ▼
Proxy Handler
  Set: org-id, TPP-client-id (route-dependent), X-Correlation-ID
  Rewrite: /api/* → /api/v1/*
  Forward to OpenFGC
```

---

## 7. Core Functions & Signatures

### 7.1 Cookie 1 — Identity Token (JWS)

```go
// Generate RS256-signed JWT containing identity claims and a kid header
func GenerateAuthToken(claims IdentityClaims, privateKey *rsa.PrivateKey, keyVersion string) (string, error)

// Read kid from JWT header, select verification key from configured key set,
// verify RS256 signature and return claims
func ValidateAuthToken(token string, publicKeys map[string]*rsa.PublicKey) (IdentityClaims, string /*kid*/, error)
```

### 7.2 Cookie 2 — Session Token (JWE)

```go
// Encrypt session claims (refresh/session material) with AES-256-GCM,
// embed kid in JWE header
func GenerateSessionToken(claims SessionClaims, keys map[string][]byte, activeKeyVersion string) (string, error)

// Read kid from JWE header, select key, decrypt
// If kid != activeKeyVersion: caller should reissue cookie
func DecryptSessionToken(token string, keys map[string][]byte) (SessionClaims, string /*kid*/, error)
```

### 7.3 CSRF

```go
// HMAC-SHA256(sub + iat, csrfHMACKey) — tied to user identity
func GenerateCSRFToken(sub string, iat int64, hmacKey []byte) string

// Verify double-submit equality and HMAC authenticity in constant time
func ValidateCSRFToken(cookieValue, headerValue, sub string, iat int64, hmacKey []byte) bool
```

### 7.4 OIDC State

```go
// state = AEAD-encrypted payload {timestamp, code_verifier, oidc_nonce}
// No cookie or server storage required
func GenerateState(codeVerifier, oidcNonce string, encKey []byte) (string, error)

// Decrypt + authenticate state, then check timestamp age
func ValidateState(state string, encKey []byte, maxAge time.Duration) (codeVerifier, oidcNonce string, err error)
```

### 7.5 PKCE / ID Token Validation Helpers

```go
// RFC 7636 S256 PKCE pair
func GeneratePKCE() (codeVerifier, codeChallenge string, err error)

// Full OIDC ID token validation: signature (JWKS), issuer, audience,
// time claims (exp/iat/nbf), and nonce binding
func ValidateIDToken(idToken, expectedNonce, expectedIssuer, expectedAudience string) error
```

### 7.6 OIDC Exchange

```go
// Exchange authorization code for IdP tokens (PKCE enabled)
func ExchangeCodeForToken(code, codeVerifier string) (*OIDCTokenResponse, error)

// Use refresh_token to obtain new access token
// Returns ErrInvalidGrant if IdP rejects (rotation consumed the token)
func RefreshAccessToken(refreshToken string) (*OIDCTokenResponse, error)

// Build provider end_session URL for RP-initiated logout (if supported)
func BuildLogoutURL(postLogoutRedirectURI string) string
```

### 7.7 Proxy

```go
// Clone and forward request to OpenFGC, applying trusted header/path transforms
func ProxyToOpenFGC(r *http.Request, target *url.URL, upstreamHeaders map[string]string) (*http.Response, error)
```

Proxy security contract (required):
- Strip hop-by-hop headers (`Connection`, `Keep-Alive`, `Proxy-Authenticate`, `Proxy-Authorization`, `TE`, `Trailer`, `Transfer-Encoding`, `Upgrade`)
- Ignore untrusted forwarding headers from clients; only add forwarding headers from trusted edge data
- Ignore incoming browser-supplied `org-id` and `TPP-client-id`; set both from trusted auth/context
- For user-scoped routes (for example `GET /me/consents`), inject `userIds=<principal-sub>` and ignore browser-supplied `userIds`
- Propagate or generate `X-Correlation-ID` for every proxied request
- Preserve query string exactly while rewriting only the path prefix `/api/` → `/api/v1/`
- Enforce request body size limits before proxying
- Enforce upstream timeout budget and deterministic timeout mapping
- Do not retry non-idempotent methods automatically
- Enforce mandatory method/path allowlist in proxy handler logic (deny by default with 404/405) even when router uses `/api/{path...}` catch-all

### 7.8 Key Rotation Helpers

```go
// Load versioned keys from config
// e.g. JWE_KEYS={"1":"oldhex...","2":"newhex..."}
func LoadKeyMap(raw string) (map[string][]byte, error)

// Return the active key for encryption
func ActiveKey(keys map[string][]byte, activeVersion string) ([]byte, error)
```

---

## 8. Security

### 8.1 XSS Protection

- `HttpOnly` on `auth_token` and `session_token` — JS cannot read either cookie
- `csrf_token` is non-HttpOnly intentionally, but contains no sensitive credential
- Security headers on all responses (see [§8.7](#87-security-headers))

### 8.2 CSRF Protection

- `SameSite=Strict` on all cookies — primary defence, blocks cross-site submission
- HMAC double-submit — `csrf_token` cookie must match `X-CSRF-Token` header
- CSRF validation is required for all non-safe HTTP methods (anything except `GET`, `HEAD`, `OPTIONS`, `TRACE`)
- CSRF tokens are verified by server-side HMAC recomputation against trusted auth claims (`sub`, `iat`) with constant-time comparison
- Subdomain cookie injection cannot produce a valid token without the CSRF HMAC key
- `POST /auth/logout` is intentionally exempt from CSRF header validation to preserve no-JS logout resilience; it is protected by strict same-origin checks plus `SameSite=Strict` cookies. Matching algorithm: origin must match one configured value in `PORTAL_ALLOWED_ORIGINS` using `Origin`, else `Referer`, else reject.

### 8.3 Token Security

- Cookie 1 (auth_token): RS256 — user can read payload, cannot forge claims
- Cookie 2 (session_token): AES-256-GCM — refresh/session material opaque to the user
- ID token is fully validated during callback: signature (JWKS), issuer, audience, time claims, and nonce
- 15-minute expiry limits window for cookie theft
- Provider refresh token rotation (if enabled) invalidates refresh tokens on use
- RP-initiated logout terminates IdP session — not just local cookies (when end-session is supported)

### 8.4 Key Management

| Key | Used For | Rotation Impact |
|-----|----------|-----------------|
| RSA Private Key | Sign auth_token (Cookie 1) | Existing tokens unverifiable — see rotation procedure |
| RSA Public Keys (versioned set) | Verify auth_token (Cookie 1) via JWT `kid` lookup | Old/new keys can overlap during rotation |
| AES-256 Key (versioned) | Encrypt session_token (Cookie 2) | Old versions retained during transition |
| CSRF HMAC Key | Sign csrf_token | All CSRF tokens invalid — reissued on next auth |
| State Encryption Key | Encrypt/decrypt OIDC state | In-flight logins fail — negligible impact |

- Never hardcode keys. Use environment variables in development, a secrets vault (HashiCorp Vault, AWS Secrets Manager) in production.
- All keys are versioned. See [§11.4](#114-key-rotation-procedure) for the rotation procedure.

### 8.5 Known Tradeoff: Revocation

All stateless token systems share a fundamental constraint: a stolen cookie is valid until expiry regardless of encryption. Encryption on Cookie 2 protects against the **legitimate user** extracting raw token values — it does not protect against physical cookie theft (device compromise). The 15-minute expiry is the primary mitigation.

If **immediate per-token revocation** is required, a minimal server-side denylist must be introduced. This is a deliberate architectural decision and should be made explicitly if the threat model requires it. It is out of scope for the current stateless design.

### 8.6 Provider Token Assumptions (Header-Trust Backend Mode)

In this deployment mode, OpenFGC does not verify bearer tokens and trusts upstream headers from the BFF.

Required provider capabilities:
- refresh token issuance for this client (commonly via `offline_access`)
- ID token at login callback (for identity bootstrap)
- refresh flow that returns either a verifiable `id_token` or an access token usable with UserInfo

JWT access tokens are recommended but not mandatory for proxying in this mode, because the BFF does not forward bearer tokens to OpenFGC.

**Confirm before implementation:** In your IdP admin console, verify refresh token issuance, ID token validation parameters (`iss`/`aud`), and (if required) RP-initiated logout endpoint availability.

### 8.7 Security Headers

Applied by `SecurityHeadersMiddleware` on all responses:

```
Content-Security-Policy: default-src 'none'
X-Content-Type-Options:  nosniff
X-Frame-Options:         DENY
Referrer-Policy:         strict-origin-when-cross-origin
```

Caching policy (required on auth/session-sensitive responses):

```
Cache-Control: no-store
Pragma:        no-cache
```

For CORS responses, include:

```
Vary: Origin
```

`default-src 'none'` is safe for the BFF because it returns only JSON and redirects — no scripts, styles, or media. The portal's CSP (where content policy is more nuanced) is defined separately and is out of scope for this plan.

### 8.8 Session Cookie Size Note

The BFF intentionally does not persist `id_token` in `session_token`. This is a size-safety decision to avoid browser/proxy cookie limit failures. RP-initiated logout is still attempted via `end_session_endpoint` without `id_token_hint`.

---

## 9. Project Structure

```
portal/backend/
├── cmd/
│   └── server/
│       ├── main.go                  # Entry point, router setup
│       └── servicemanager.go        # Service and middleware initialisation
│
├── internal/
│   ├── auth/
│   │   ├── handler.go               # /auth route handlers
│   │   ├── oidc.go                  # OIDC provider client, ExchangeCodeForToken, RefreshAccessToken
│   │   ├── identity_token.go        # Cookie 1: GenerateAuthToken, ValidateAuthToken
│   │   ├── session_token.go         # Cookie 2: GenerateSessionToken, DecryptSessionToken
│   │   ├── csrf.go                  # GenerateCSRFToken, ValidateCSRFToken
│   │   ├── state.go                 # GenerateState, ValidateState
│   │   └── service.go               # Auth business logic, cookie issuance
│   │
│   ├── proxy/
│   │   ├── handler.go               # Proxy route handler
│   │   └── service.go               # ProxyToOpenFGC
│   │
│   ├── middleware/
│   │   ├── auth.go                  # Cookie 1 validation
│   │   ├── session.go               # Cookie 2 decryption, key selection
│   │   ├── csrf.go                  # CSRF double-submit validation
│   │   ├── cors.go                  # CORS — explicit origin, credentials
│   │   ├── security_headers.go      # CSP, X-Frame-Options, etc.
│   │   └── logger.go                # Request / response logging
│   │
│   ├── config/
│   │   └── config.go                # Env config, key loading, validation
│   │
│   └── model/
│       └── types.go                 # IdentityClaims, SessionClaims, OIDCTokenResponse
│
├── tests/
│   ├── unit/
│   │   ├── identity_token_test.go
│   │   ├── session_token_test.go
│   │   ├── csrf_test.go
│   │   └── state_test.go
│   └── integration/
│       └── auth_test.go
│
├── docs/
│   ├── DEPLOYMENT.md                # Rate limiting, race condition, key rotation
│   └── API.md                       # Endpoint reference
│
├── .env.example
├── go.mod
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

---

## 10. Configuration

Configuration implementation note:
- Use `Koanf` as the configuration loader/merger for the BFF.
- Keep environment variables as the primary runtime source.
- Support file-backed configuration for local/dev overlays as needed.

```bash
# ── Server ────────────────────────────────────────────────────────────────
PORT=8080
ENV=development
LOG_LEVEL=info

# ── OIDC Provider ─────────────────────────────────────────────────────────
OIDC_ISSUER_URL=https://localhost:9443
OIDC_CLIENT_ID=bff_client
OIDC_CLIENT_SECRET=your_client_secret
OIDC_REDIRECT_URI=http://localhost:8080/auth/callback
OIDC_SCOPE=openid profile email offline_access
OIDC_RESOURCE_AUDIENCE=openfgc-api          # optional in header-trust backend mode; keep when provider policy requires audience-bound tokens
# If `offline_access` is removed, disable `/auth/refresh` and related portal refresh flow.

# ── Cookie 1: auth_token (JWS, RS256) ────────────────────────────────────
JWT_RSA_PRIVATE_KEY_PATH=./keys/rsa_private.pem
JWT_RSA_PUBLIC_KEYS={"1":"./keys/rsa_public_v1.pem","2":"./keys/rsa_public_v2.pem"}
JWT_KEY_VERSION=2                           # active signing key version (written to JWT kid)
JWT_EXPIRY_MINUTES=15

# ── Cookie 2: session_token (JWE, AES-256-GCM) ───────────────────────────
# JSON map of version → hex-encoded 32-byte key
# Retain old versions during rotation (see DEPLOYMENT.md)
JWE_KEYS={"1":"<old_32byte_hex>","2":"<new_32byte_hex>"}
JWE_ACTIVE_KEY_VERSION=2
SESSION_EXPIRY_MINUTES=15
SESSION_COOKIE_MAX_BYTES=3500              # hard cap for full serialized Set-Cookie length

# ── CSRF ──────────────────────────────────────────────────────────────────
CSRF_HMAC_KEY=<32-byte-hex-encoded-key>

# ── OIDC State ────────────────────────────────────────────────────────────
STATE_ENC_KEY=<32-byte-hex-encoded-key>
STATE_MAX_AGE_MINUTES=10

# ── OpenFGC Backend ───────────────────────────────────────────────────────
OPENFGC_API_URL=http://localhost:9090
OPENFGC_API_TIMEOUT=10s

# ── CORS ──────────────────────────────────────────────────────────────────
PORTAL_ALLOWED_ORIGINS=http://localhost:3000 # comma-separated allowlist; never wildcard

# ── Cookies ───────────────────────────────────────────────────────────────
AUTH_COOKIE_NAME=auth_token                  # prod: __Host-auth_token
SESSION_COOKIE_NAME=session_token            # prod: __Host-session_token
CSRF_COOKIE_NAME=csrf_token                  # prod: __Host-csrf_token
COOKIE_DOMAIN=                               # localhost dev: leave empty (host-only). prod: set real domain
                                             # when using __Host-* cookie names, this MUST stay empty
COOKIE_PATH=/                                # required so cookies set on /auth are also sent to /api
COOKIE_SECURE=true                           # default
# Local HTTP-only dev override:
#   COOKIE_SECURE=false                      # use ONLY with http://localhost in development
#   Never use false in staging/production.
COOKIE_SAME_SITE=Strict

# Refresh accepts slightly expired auth_token only within this grace window
REFRESH_AUTH_GRACE_SECONDS=300

# ── OIDC Refresh Token Rotation ───────────────────────────────────────────
ENABLE_REFRESH_TOKEN_ROTATION=true
```

---

## 11. Deployment Notes

### 11.1 Rate Limiting

Rate limiting on auth endpoints (`/auth/login`, `/auth/callback`, `/auth/refresh`, `/auth/logout`) is an **infrastructure concern** and must be enforced at the gateway or reverse proxy layer before requests reach the BFF. The BFF does not implement rate limiting internally.

Recommended limits as a starting point:

| Endpoint | Limit |
|----------|-------|
| `/auth/login` | 20 req/min per IP |
| `/auth/callback` | 20 req/min per IP |
| `/auth/refresh` | 60 req/min per IP |
| `/auth/logout` | 30 req/min per IP |

In local development, run Nginx or Traefik in docker-compose in front of the BFF from day one to maintain parity with staging and production.

### 11.2 Refresh Race Condition

When provider refresh token rotation is enabled and multiple BFF replicas simultaneously attempt to refresh the same token, the provider will reject all but the first call with `invalid_grant`. The BFF handles this gracefully by clearing cookies and returning 401. The portal must handle 401 by redirecting to `/auth/login`.

v1 policy is fixed: accept `401` on refresh collision and force re-login (no distributed lock, no proactive client timer requirement).

### 11.3 Refresh Endpoint Hardening

`/auth/refresh` is the highest-risk endpoint in this design because it mints fresh bearer material from a long-lived credential.

Hard requirements:

- Keep infrastructure rate limiting enabled for `/auth/refresh` in every environment (including local docker parity)
- Do not implement automatic server-side retry on token refresh failures
- Use deterministic error mapping:
  - `invalid_grant` => clear all auth cookies, return 401
  - transient upstream errors/timeouts => return 503 (or 502) without mutating cookies
  - malformed/missing session cookie => return 401
  - auth_token signature mismatch or expiry beyond grace window => return 401
- Log refresh failures with stable reason codes (no token material in logs)
- Portal must avoid silent infinite refresh loops (single refresh attempt per user action/navigation cycle)
- Add integration tests for concurrent refresh calls and repeated 401 handling

### 11.4 Key Rotation Procedure

The `kid` field in the JWE header allows the BFF to select the correct decryption key from a versioned map. This enables graceful rotation: old cookies remain decryptable during the transition window, and are transparently reissued with the new key.

**Rotation steps (JWE / AES key):**

```
Step 1 — Add new key, keep old key, bump active version
  JWE_KEYS={"1":"<old_key>","2":"<new_key>"}
  JWE_ACTIVE_KEY_VERSION=2
  Deploy. New cookies use key version 2. Old cookies (version 1) still decrypt.
  Session middleware reissues version-1 cookies with version-2 on every response.

Step 2 — Wait one TTL window (15 minutes)
  All active sessions now carry version-2 cookies.

Step 3 — Remove old key
  JWE_KEYS={"2":"<new_key>"}
  Deploy. Any remaining version-1 cookies return ErrUnknownKeyVersion → 401 → re-auth.
```

**Rotation steps (RSA keypair / JWT signing key):**

```
Step 1 — Generate new RSA keypair
Step 2 — Add new public key to JWKS endpoint alongside old public key
          Both old and new tokens verify correctly during transition.
Step 3 — Update JWT_RSA_PRIVATE_KEY_PATH to new private key, bump JWT_KEY_VERSION
          New auth_token cookies signed with new key. Old cookies verify via old public key.
Step 4 — Wait one TTL window (15 minutes)
Step 5 — Remove old public key from JWKS endpoint
```

### 11.5 IdP TLS in Local Development

If the selected IdP uses a self-signed certificate in local development, TLS verification will fail in the BFF container. Resolution options:

```yaml
# docker-compose.yml — mount IdP cert into BFF container
services:
  bff:
    volumes:
      - ./certs/idp.crt:/certs/idp.crt:ro
    environment:
      SSL_CERT_FILE: /certs/idp.crt
```

Do **not** use `InsecureSkipVerify: true` in any environment other than local development, and never commit it without an explicit warning comment.

### 11.6 Header-Trust Backend Boundary (Mandatory)

When OpenFGC trusts upstream headers and does not verify bearer tokens, network boundary controls are mandatory:

- OpenFGC must not be directly reachable from browsers/public networks
- Allow inbound access to OpenFGC only from BFF (private network ACL / security group)
- At ingress, strip any client-supplied `org-id` and `TPP-client-id`; only BFF may set them
- Prefer mTLS between BFF and OpenFGC, or equivalent service-to-service authentication at the edge
- Deny or drop requests that bypass BFF path mapping/policy enforcement

---

## 12. Implementation Checklist

### Setup
- [ ] Initialise Go 1.22+ project, confirm stdlib mux path parameter support
- [ ] Confirm IdP token format is JWT and refresh tokens are enabled (see [§8.6](#86-provider-token-format-assumption))
- [ ] Generate RSA keypair for auth_token (Cookie 1)
- [ ] Generate AES-256 key(s) for session_token (Cookie 2)
- [ ] Generate HMAC key for CSRF tokens
- [ ] Generate encryption key for OIDC state
- [ ] Set cookie policy with `Path=/` for auth_token, session_token, and csrf_token
- [ ] Configure `AUTH_COOKIE_NAME`, `SESSION_COOKIE_NAME`, and `CSRF_COOKIE_NAME` per environment
- [ ] Mount IdP TLS cert in docker-compose (see [§11.5](#115-idp-tls-in-local-development))

### Authentication
- [ ] `GeneratePKCE` + OIDC `nonce` generation on login
- [ ] `GenerateState` / `ValidateState` — encrypted + authenticated, timestamp-verified, carries PKCE verifier + nonce
- [ ] `/auth/login` — generate PKCE + nonce + state, redirect to OIDC provider with `code_challenge_method=S256`
- [ ] `/auth/me` — validate `auth_token` and return identity for portal bootstrap (200/401)
- [ ] `/auth/callback` — validate state, exchange code with `code_verifier`, fully validate ID token (signature/JWKS, `iss`, `aud`, `exp`/`iat`/`nbf`, nonce), issue all three cookies
- [ ] Do not persist `id_token` in `session_token` (size guard). Use `end_session_endpoint` without `id_token_hint`
- [ ] `/auth/logout` — POST + same-origin allowlist (`Origin`/`Referer`) validation, clear all cookies, call provider end-session endpoint when available
- [ ] `/auth/refresh` — require CSRF + session_token + signed auth_token consistency; allow auth_token expiry only within a short grace window

### Token Functions
- [ ] `GenerateAuthToken` / `ValidateAuthToken` (JWS, RS256, JWT `kid` + public-key-set lookup)
- [ ] `GenerateSessionToken` / `DecryptSessionToken` (JWE, AES-256-GCM, kid-versioned, stores refresh/session material only)
- [ ] Enforce `SESSION_COOKIE_MAX_BYTES` before writing `session_token`; fail closed on overflow
- [ ] `GenerateCSRFToken` / `ValidateCSRFToken` (HMAC over `sub`+`iat`, double-submit equality, constant-time verification)
- [ ] `GeneratePKCE` / `ValidateIDToken`
- [ ] On refresh, enforce identity re-validation (validate returned `id_token` or UserInfo `sub` consistency) before reissuing auth cookies
- [ ] `LoadKeyMap` / `ActiveKey` (versioned key management)
- [ ] Reissue CSRF token whenever `auth_token` is reissued (because CSRF HMAC binds to auth `sub` + `iat`)

### Middleware
- [ ] `SecurityHeadersMiddleware` — CSP, X-Content-Type-Options, X-Frame-Options, Referrer-Policy
- [ ] `CORSMiddleware` — origin allowlist matching, credentials: true, OPTIONS preflight
- [ ] `AuthMiddleware` — Cookie 1 RS256 verification
- [ ] `SessionMiddleware` — Cookie 2 AES-GCM decryption, kid-based key selection, transparent reissue
- [ ] `CSRFMiddleware` — HMAC-bound double-submit validation for all non-safe methods (`GET`, `HEAD`, `OPTIONS`, `TRACE` exempt)
- [ ] `OriginCheckMiddleware` — strict same-origin allowlist validation for `POST /auth/logout`
- [ ] `LoggerMiddleware`
- [ ] Logout route middleware chain — `OriginCheck` for `POST /auth/logout`
- [ ] Refresh route middleware chain — `Session` → `CSRF` + mandatory auth_token sub/signature consistency for `POST /auth/refresh`
- [ ] Authorization scoping middleware/policy — enforce user-scoped filters on `/me/*` routes using authenticated principal (`sub`)
- [ ] Top-level router wrapper — apply middleware in canonical order: `SecurityHeaders` → `CORS` → `Logger`
- [ ] Router structure — public auth routes on base mux, refresh on dedicated protected mux, `/api/*` on full protected mux (see [§6.1](#61-router-structure))
- [ ] Keep route wiring lean via one `RegisterRoutes(...)` entry point and middleware-chain tests

### Proxy & Testing
- [ ] `ProxyToOpenFGC` — clone request, map path `/api/*` to `/api/v1/*`, and inject trusted upstream headers
- [ ] Proxy hardening — hop-by-hop header stripping, ignore client-supplied tenant/client headers, correlation-id propagation, request size limits, timeout/retry policy, method/path allowlisting
- [ ] Implement `GET /me/consents` → upstream `/api/v1/consents?userIds=<bff-injected-sub>` mapping
- [ ] Unit tests: identity_token, session_token, csrf, state
- [ ] Unit tests: middleware-chain assertions
- [ ] Integration tests: full auth flow (login → callback → authenticated request → refresh → logout)
- [ ] Integration tests: proxy header transforms (`org-id`, `TPP-client-id`, `X-Correlation-ID`) and client-header override prevention
- [ ] Integration tests: proxy path mapping `/api/*` -> `/api/v1/*` with query-string preservation
- [ ] Integration tests: `/me/consents` always injects authenticated `userIds` and ignores client-supplied user filters
- [ ] Integration tests: refresh failure mapping (`invalid_grant` -> 401 + cookie clear, upstream timeout -> 503 without cookie mutation)
- [ ] Integration tests: refresh collision behavior (`invalid_grant` path -> 401 + re-login)
- [ ] Integration tests: logout same-origin allowlist algorithm (`Origin`/`Referer` in allowlist accepted, absent/mismatch rejected)
- [ ] Integration tests: cache-control and CORS `Vary: Origin` headers on auth endpoints
- [ ] Docker + docker-compose with Nginx/Traefik for rate limiting parity
- [ ] Deployment controls: OpenFGC private-only exposure (BFF-origin traffic only), client header stripping at edge, and optional mTLS between BFF and OpenFGC

### Documentation
- [ ] `docs/DEPLOYMENT.md` — rate limiting requirements, v1 refresh race policy (401 -> re-login), key rotation procedure
- [ ] `docs/API.md` — endpoint reference
- [ ] `.env.example` — all variables with descriptions

---

## 13. Patterns, Practices, and Standards

- HTTP stack: Go stdlib `net/http` with `ServeMux` route registration and middleware composition
- File/module structure: standard Go layout with clear boundaries (`cmd`, `internal`, `tests`, `docs`) and feature-oriented packages for `auth`, `proxy`, `middleware`, and `config`
- Configuration model: centralized configuration loading/validation using `Koanf`, with explicit environment-driven settings for security keys, cookie policy, OIDC, CORS, and upstream targets
- Design approach: contract-first API definitions via OpenAPI, with BFF route and payload contracts maintained from spec
- Dependency approach: stdlib-first implementation style; add third-party libraries only when there is a clear functional need
- Security baseline: AES-256-GCM for encrypted session material, HMAC for CSRF validation, versioned key rotation, and externalized secret management
- Testing model: layered tests with unit coverage for token/middleware helpers and integration coverage for auth, proxying, and security/error paths
- Delivery model: reproducible containerized packaging and local parity environments using `Dockerfile` + `docker-compose`, with scripted build/test workflows
- Operational controls: deployment boundary hardening (header trust boundary, edge stripping rules, private backend exposure) and infrastructure-level rate limiting for auth endpoints
