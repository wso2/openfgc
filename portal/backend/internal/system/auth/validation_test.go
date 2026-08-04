/*
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
 */

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/wso2/openfgc/portal/backend/internal/system/config"
	systemcontext "github.com/wso2/openfgc/portal/backend/internal/system/context"
)

type oidcValidationProvider struct {
	t      *testing.T
	server *httptest.Server
	key1   *rsa.PrivateKey
	key2   *rsa.PrivateKey
	key3   *rsa.PrivateKey

	mu         sync.Mutex
	keys       []jose.JSONWebKey
	jwksCalls  int
	failNextJW bool
	token      http.HandlerFunc
	endSession bool
}

func newOIDCValidationProvider(t *testing.T) *oidcValidationProvider {
	t.Helper()
	p := &oidcValidationProvider{t: t}
	p.key1 = generateRSAKey(t)
	p.key2 = generateRSAKey(t)
	p.key3 = generateRSAKey(t)
	p.keys = []jose.JSONWebKey{publicJWK(p.key1, "key-1")}
	p.server = httptest.NewServer(http.HandlerFunc(p.serveHTTP))
	t.Cleanup(p.server.Close)
	return p
}

func (p *oidcValidationProvider) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		metadata := map[string]any{
			"issuer": p.server.URL, "authorization_endpoint": p.server.URL + "/authorize",
			"token_endpoint": p.server.URL + "/token", "jwks_uri": p.server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		if p.endSession {
			metadata["end_session_endpoint"] = p.server.URL + "/logout"
		}
		writeTestJSON(w, metadata)
	case "/jwks":
		p.mu.Lock()
		p.jwksCalls++
		fail := p.failNextJW
		p.failNextJW = false
		keys := append([]jose.JSONWebKey(nil), p.keys...)
		p.mu.Unlock()
		if fail {
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		writeTestJSON(w, jose.JSONWebKeySet{Keys: keys})
	case "/token":
		if p.token == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p.token(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *oidcValidationProvider) manager(t *testing.T, mutate func(*config.AuthConfig)) *Manager {
	t.Helper()
	cfg := testAuthConfig(p.server.URL)
	if mutate != nil {
		mutate(&cfg)
	}
	manager, err := NewManager(context.Background(), cfg, config.ProxyConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return manager
}

func TestAccessTokenValidationFailures(t *testing.T) {
	p := newOIDCValidationProvider(t)
	now := time.Now()
	valid := map[string]any{
		"iss": p.server.URL, "sub": " user-1 ", "aud": "portal-api",
		"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "nbf": now.Add(-time.Minute).Unix(),
		"org_id": " org-1 ", "scope": ScopeConsentsReadSelf + " " + ScopeConsentsReadSelf + " " + ScopeElementsRead,
	}

	tests := []struct {
		name       string
		claims     map[string]any
		key        *rsa.PrivateKey
		keyID      string
		algorithm  jose.SignatureAlgorithm
		mutateCfg  func(*config.AuthConfig)
		wantValid  bool
		wantScopes []string
	}{
		{name: "valid and normalized", claims: cloneClaims(valid), key: p.key1, keyID: "key-1", algorithm: jose.RS256, wantValid: true, wantScopes: []string{ScopeConsentsReadSelf, ScopeElementsRead}},
		{name: "invalid signature", claims: cloneClaims(valid), key: p.key2, keyID: "key-1", algorithm: jose.RS256},
		{name: "unsupported algorithm", claims: cloneClaims(valid), key: p.key1, keyID: "key-1", algorithm: jose.PS256},
		{name: "expired", claims: withClaim(valid, "exp", now.Add(-time.Hour).Unix()), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "future nbf", claims: withClaim(valid, "nbf", now.Add(10*time.Minute).Unix()), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "wrong issuer", claims: withClaim(valid, "iss", "https://wrong.example"), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "wrong audience", claims: withClaim(valid, "aud", "wrong-api"), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "missing subject", claims: withoutClaim(valid, "sub"), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "blank organization", claims: withClaim(valid, "org_id", " "), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "malformed organization", claims: withClaim(valid, "org_id", 123), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "malformed scopes", claims: withClaim(valid, "scope", 123), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{
			name: "wrong token type", claims: withClaim(valid, "typ", "refresh"), key: p.key1, keyID: "key-1", algorithm: jose.RS256,
			mutateCfg: func(cfg *config.AuthConfig) {
				cfg.RequireAccessTokenType = true
				cfg.AccessTokenTypeClaim = "typ"
				cfg.AccessTokenTypeValue = "access"
			},
		},
		{
			name: "valid token type", claims: withClaim(valid, "typ", "access"), key: p.key1, keyID: "key-1", algorithm: jose.RS256, wantValid: true,
			mutateCfg: func(cfg *config.AuthConfig) {
				cfg.RequireAccessTokenType = true
				cfg.AccessTokenTypeClaim = "typ"
				cfg.AccessTokenTypeValue = "access"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := p.manager(t, test.mutateCfg)
			raw := signTestTokenWithAlgorithm(t, test.key, test.keyID, test.algorithm, test.claims)
			principal, _, err := manager.validateAccessToken(context.Background(), raw)
			if (err == nil) != test.wantValid {
				t.Fatalf("valid=%v, error=%v", test.wantValid, err)
			}
			if !test.wantValid {
				return
			}
			if principal.UserID != "user-1" || principal.OrgID != "org-1" {
				t.Fatalf("claims were not normalized: %#v", principal)
			}
			for _, scope := range test.wantScopes {
				if _, ok := principal.Scopes[scope]; !ok {
					t.Errorf("missing normalized scope %q", scope)
				}
			}
		})
	}
}

func TestAccessTokenArrayScopeClaim(t *testing.T) {
	p := newOIDCValidationProvider(t)
	manager := p.manager(t, nil)
	now := time.Now()
	raw := signTestToken(t, p.key1, "key-1", map[string]any{
		"iss": p.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
		"org_id": "org", "scope": []string{ScopeElementsRead, "", ScopeElementsRead},
	})
	principal, _, err := manager.validateAccessToken(context.Background(), raw)
	if err != nil || len(principal.Scopes) != 1 {
		t.Fatalf("unexpected array-scope result: %#v, %v", principal, err)
	}
}

func TestAccessTokenOrganizationOverride(t *testing.T) {
	p := newOIDCValidationProvider(t)
	manager := p.manager(t, func(cfg *config.AuthConfig) {
		cfg.OrgIDOverride = " ORG-001 "
	})
	now := time.Now()
	raw := signTestToken(t, p.key1, "key-1", map[string]any{
		"iss": p.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
		"scope": ScopeConsentsReadSelf,
	})

	principal, _, err := manager.validateAccessToken(context.Background(), raw)
	if err != nil {
		t.Fatalf("expected token without organization claim to use override: %v", err)
	}
	if principal.OrgID != "ORG-001" {
		t.Fatalf("unexpected organization ID: %q", principal.OrgID)
	}
}

func TestIDTokenValidationFailures(t *testing.T) {
	p := newOIDCValidationProvider(t)
	manager := p.manager(t, nil)
	now := time.Now()
	valid := map[string]any{
		"iss": p.server.URL, "sub": "user-1", "aud": "portal-client",
		"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "nbf": now.Add(-time.Minute).Unix(),
	}
	tests := []struct {
		name      string
		claims    map[string]any
		key       *rsa.PrivateKey
		keyID     string
		algorithm jose.SignatureAlgorithm
		wantValid bool
	}{
		{name: "valid", claims: cloneClaims(valid), key: p.key1, keyID: "key-1", algorithm: jose.RS256, wantValid: true},
		{name: "invalid signature", claims: cloneClaims(valid), key: p.key2, keyID: "key-1", algorithm: jose.RS256},
		{name: "unsupported algorithm", claims: cloneClaims(valid), key: p.key1, keyID: "key-1", algorithm: jose.PS256},
		{name: "expired", claims: withClaim(valid, "exp", now.Add(-time.Hour).Unix()), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "future nbf", claims: withClaim(valid, "nbf", now.Add(10*time.Minute).Unix()), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "wrong issuer", claims: withClaim(valid, "iss", "https://wrong.example"), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "wrong audience", claims: withClaim(valid, "aud", "wrong-client"), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
		{name: "missing subject", claims: withoutClaim(valid, "sub"), key: p.key1, keyID: "key-1", algorithm: jose.RS256},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := signTestTokenWithAlgorithm(t, test.key, test.keyID, test.algorithm, test.claims)
			_, err := manager.validateIDToken(context.Background(), raw)
			if (err == nil) != test.wantValid {
				t.Fatalf("valid=%v, error=%v", test.wantValid, err)
			}
		})
	}
}

func TestJWKSCacheUnknownKeyRotationAndTransientFailure(t *testing.T) {
	p := newOIDCValidationProvider(t)
	manager := p.manager(t, nil)
	now := time.Now()
	claims := map[string]any{
		"iss": p.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
		"org_id": "org", "scope": ScopeElementsRead,
	}

	validate := func(key *rsa.PrivateKey, keyID string, wantValid bool) {
		t.Helper()
		_, _, err := manager.validateAccessToken(context.Background(), signTestToken(t, key, keyID, claims))
		if (err == nil) != wantValid {
			t.Fatalf("key %s valid=%v, error=%v", keyID, wantValid, err)
		}
	}
	validate(p.key1, "key-1", true)
	validate(p.key1, "key-1", true)
	p.mu.Lock()
	if p.jwksCalls != 1 {
		t.Fatalf("expected cached JWKS after one fetch, got %d", p.jwksCalls)
	}
	p.keys = append(p.keys, publicJWK(p.key2, "key-2"))
	p.mu.Unlock()
	validate(p.key2, "key-2", true)
	p.mu.Lock()
	if p.jwksCalls != 2 {
		t.Fatalf("expected unknown kid to refresh JWKS, got %d calls", p.jwksCalls)
	}
	p.keys = append(p.keys, publicJWK(p.key3, "key-3"))
	p.failNextJW = true
	p.mu.Unlock()
	validate(p.key3, "key-3", false)
	validate(p.key3, "key-3", true)
}

func TestAuthenticationFailureAndAuthorizationFailureSemantics(t *testing.T) {
	manager := &Manager{cfg: config.AuthConfig{}, proxyCfg: config.ProxyConfig{}}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	unauthorized := httptest.NewRecorder()
	manager.Require(next).ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/", nil))
	if unauthorized.Code != http.StatusUnauthorized || unauthorized.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("unexpected authentication failure: %d, %q", unauthorized.Code, unauthorized.Header().Get("WWW-Authenticate"))
	}

	manager.proxyCfg = config.ProxyConfig{PlaceholderModeEnabled: true, PlaceholderUserID: "user", PlaceholderOrgID: "org"}
	forbidden := httptest.NewRecorder()
	manager.Require(next, "missing-scope").ServeHTTP(forbidden, httptest.NewRequest(http.MethodGet, "/", nil))
	if forbidden.Code != http.StatusForbidden || forbidden.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("unexpected authorization failure: %d, %q", forbidden.Code, forbidden.Header().Get("WWW-Authenticate"))
	}
}

func TestSplitBearerAuthenticationRejectionCases(t *testing.T) {
	provider := newOIDCValidationProvider(t)
	manager := provider.manager(t, nil)
	now := time.Now()
	raw := signTestToken(t, provider.key1, "key-1", map[string]any{
		"iss": provider.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
		"org_id": "org", "scope": ScopeElementsRead,
	})
	part1, part2, err := splitToken(raw, manager.cfg.MaxTokenPartBytes)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name          string
		authorization []string
		cookie        string
	}{
		{name: "missing part 1", cookie: manager.cfg.AccessTokenPart2Cookie + "=" + part2},
		{name: "missing part 2", authorization: []string{"Bearer " + part1}},
		{name: "complete token injection", authorization: []string{"Bearer " + raw}, cookie: manager.cfg.AccessTokenPart2Cookie + "=" + part2},
		{name: "wrong half order", authorization: []string{"Bearer " + part2}, cookie: manager.cfg.AccessTokenPart2Cookie + "=" + part1},
		{name: "tampered part 1", authorization: []string{"Bearer x" + part1[1:]}, cookie: manager.cfg.AccessTokenPart2Cookie + "=" + part2},
		{name: "tampered part 2", authorization: []string{"Bearer " + part1}, cookie: manager.cfg.AccessTokenPart2Cookie + "=x" + part2[1:]},
		{name: "duplicate part 1", authorization: []string{"Bearer " + part1, "Bearer " + part1}, cookie: manager.cfg.AccessTokenPart2Cookie + "=" + part2},
		{name: "duplicate part 2", authorization: []string{"Bearer " + part1}, cookie: manager.cfg.AccessTokenPart2Cookie + "=" + part2 + "; " + manager.cfg.AccessTokenPart2Cookie + "=" + part2},
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			for _, value := range test.authorization {
				request.Header.Add("Authorization", value)
			}
			if test.cookie != "" {
				request.Header.Set("Cookie", test.cookie)
			}
			recorder := httptest.NewRecorder()
			manager.Require(next).ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized || recorder.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("got status %d and challenge %q", recorder.Code, recorder.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestEveryProtectedPolicyAcceptsReconstructedTokenWithRequiredScope(t *testing.T) {
	provider := newOIDCValidationProvider(t)
	manager := provider.manager(t, nil)
	now := time.Now()
	raw := signTestToken(t, provider.key1, "key-1", map[string]any{
		"iss": provider.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
		"org_id": "org", "scope": strings.Join(AllPortalScopes, " "),
	})
	part1, part2, err := splitToken(raw, manager.cfg.MaxTokenPartBytes)
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := systemcontext.PrincipalFromContext(r.Context())
		if !ok || principal.UserID != "user" || principal.OrgID != "org" {
			t.Errorf("missing validated principal: %#v", principal)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	tests := []struct {
		method, path, scope string
		api                 bool
	}{
		{"GET", "/me/consents", ScopeConsentsReadSelf, false},
		{"GET", "/me/consents/c1", ScopeConsentsReadSelf, false},
		{"POST", "/me/consents/c1/approve", ScopeConsentsWriteSelf, false},
		{"POST", "/me/consents/c1/reject", ScopeConsentsWriteSelf, false},
		{"POST", "/me/consents/c1/revoke", ScopeConsentsWriteSelf, false},
		{"GET", "/api/consents", ScopeConsentsReadAny, true},
		{"POST", "/api/consents", ScopeConsentsWriteAny, true},
		{"GET", "/api/consents/attributes", ScopeConsentsReadAny, true},
		{"POST", "/api/consents/validate", ScopeConsentsReadAny, true},
		{"GET", "/api/consents/c1", ScopeConsentsReadAny, true},
		{"PUT", "/api/consents/c1", ScopeConsentsWriteAny, true},
		{"GET", "/api/consents/c1/history", ScopeConsentsReadAny, true},
		{"POST", "/api/consents/c1/revoke", ScopeConsentsWriteAny, true},
		{"GET", "/api/consents/c1/authorizations", ScopeConsentsReadAny, true},
		{"POST", "/api/consents/c1/authorizations", ScopeConsentsWriteAny, true},
		{"GET", "/api/consents/c1/authorizations/a1", ScopeConsentsReadAny, true},
		{"PUT", "/api/consents/c1/authorizations/a1", ScopeConsentsWriteAny, true},
		{"GET", "/api/consent-elements", ScopeElementsRead, true},
		{"POST", "/api/consent-elements", ScopeElementsWrite, true},
		{"GET", "/api/consent-elements/e1", ScopeElementsRead, true},
		{"GET", "/api/consent-elements/e1/versions", ScopeElementsRead, true},
		{"POST", "/api/consent-elements/e1/versions", ScopeElementsWrite, true},
		{"GET", "/api/consent-elements/e1/versions/v1", ScopeElementsRead, true},
		{"DELETE", "/api/consent-elements/e1/versions/v1", ScopeElementsWrite, true},
		{"GET", "/api/consent-purposes", ScopePurposesRead, true},
		{"POST", "/api/consent-purposes", ScopePurposesWrite, true},
		{"GET", "/api/consent-purposes/p1", ScopePurposesRead, true},
		{"GET", "/api/consent-purposes/p1/versions", ScopePurposesRead, true},
		{"POST", "/api/consent-purposes/p1/versions", ScopePurposesWrite, true},
		{"GET", "/api/consent-purposes/p1/versions/v1", ScopePurposesRead, true},
		{"DELETE", "/api/consent-purposes/p1/versions/v1", ScopePurposesWrite, true},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+part1)
			request.AddCookie(&http.Cookie{Name: manager.cfg.AccessTokenPart2Cookie, Value: part2})
			recorder := httptest.NewRecorder()
			if test.api {
				manager.RequireAPI(next).ServeHTTP(recorder, request)
			} else {
				manager.Require(next, test.scope).ServeHTTP(recorder, request)
			}
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("got %d: %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestAPIRequiresAnyScopeAndCarriesOrganizationPrincipal(t *testing.T) {
	provider := newOIDCValidationProvider(t)
	manager := provider.manager(t, nil)
	now := time.Now()
	requestWithScopes := func(scopes string) *http.Request {
		t.Helper()
		raw := signTestToken(t, provider.key1, "key-1", map[string]any{
			"iss": provider.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
			"org_id": "trusted-org", "scope": scopes,
		})
		part1, part2, err := splitToken(raw, manager.cfg.MaxTokenPartBytes)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodGet, "/api/consents?org-id=spoofed", nil)
		request.Header.Set("Authorization", "Bearer "+part1)
		request.AddCookie(&http.Cookie{Name: manager.cfg.AccessTokenPart2Cookie, Value: part2})
		return request
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := systemcontext.PrincipalFromContext(r.Context())
		if !ok || principal.OrgID != "trusted-org" {
			t.Fatalf("unexpected principal: %#v", principal)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	forbidden := httptest.NewRecorder()
	manager.RequireAPI(next).ServeHTTP(forbidden, requestWithScopes(ScopeConsentsReadSelf))
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("self scope must not authorize /api route: %d", forbidden.Code)
	}
	authorized := httptest.NewRecorder()
	manager.RequireAPI(next).ServeHTTP(authorized, requestWithScopes(ScopeConsentsReadAny))
	if authorized.Code != http.StatusNoContent {
		t.Fatalf("any scope should authorize /api route: %d", authorized.Code)
	}
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func publicJWK(key *rsa.PrivateKey, keyID string) jose.JSONWebKey {
	return jose.JSONWebKey{Key: &key.PublicKey, KeyID: keyID, Algorithm: "RS256", Use: "sig"}
}

func signTestTokenWithAlgorithm(t *testing.T, key *rsa.PrivateKey, keyID string, algorithm jose.SignatureAlgorithm, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: algorithm, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return serialized
}

func cloneClaims(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func withClaim(source map[string]any, name string, value any) map[string]any {
	result := cloneClaims(source)
	result[name] = value
	return result
}

func withoutClaim(source map[string]any, name string) map[string]any {
	result := cloneClaims(source)
	delete(result, name)
	return result
}
