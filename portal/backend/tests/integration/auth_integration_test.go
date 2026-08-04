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

package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/wso2/openfgc/portal/backend/internal/system/auth"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
)

type authIntegrationProvider struct {
	t      *testing.T
	server *httptest.Server
	key    *rsa.PrivateKey
	keyID  string
	scopes string

	mu           sync.Mutex
	refreshCalls int
}

func newAuthIntegrationProvider(t *testing.T, scopes string) *authIntegrationProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	provider := &authIntegrationProvider{t: t, key: key, keyID: "integration-key", scopes: scopes}
	provider.server = httptest.NewServer(http.HandlerFunc(provider.serveHTTP))
	t.Cleanup(provider.server.Close)
	return provider
}

func (p *authIntegrationProvider) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		writeAuthIntegrationJSON(w, map[string]any{
			"issuer": p.server.URL, "authorization_endpoint": p.server.URL + "/authorize",
			"token_endpoint": p.server.URL + "/token", "jwks_uri": p.server.URL + "/jwks",
			"end_session_endpoint":                  p.server.URL + "/logout",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/jwks":
		writeAuthIntegrationJSON(w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: &p.key.PublicKey, KeyID: p.keyID, Algorithm: "RS256", Use: "sig",
		}}})
	case "/token":
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if clientID, secret, ok := r.BasicAuth(); !ok || clientID != "integration-client" || secret != "integration-secret" {
			http.Error(w, "invalid client", http.StatusUnauthorized)
			return
		}
		if r.Form.Get("grant_type") != "refresh_token" && r.Form.Get("code_verifier") == "" {
			http.Error(w, "missing PKCE verifier", http.StatusBadRequest)
			return
		}
		refreshToken := "refresh-token-initial"
		if r.Form.Get("grant_type") == "refresh_token" {
			p.mu.Lock()
			p.refreshCalls++
			p.mu.Unlock()
			refreshToken = "refresh-token-rotated"
		}
		writeAuthIntegrationJSON(w, p.tokenResponse(refreshToken))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *authIntegrationProvider) tokenResponse(refreshToken string) map[string]any {
	now := time.Now()
	accessToken := signAuthIntegrationToken(p.t, p.key, p.keyID, map[string]any{
		"iss": p.server.URL, "sub": "integration-user", "aud": "integration-api",
		"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "nbf": now.Add(-time.Minute).Unix(),
		"org_id": "integration-org", "scope": p.scopes,
	})
	idToken := signAuthIntegrationToken(p.t, p.key, p.keyID, map[string]any{
		"iss": p.server.URL, "sub": "integration-user", "aud": "integration-client",
		"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(),
	})
	return map[string]any{
		"access_token": accessToken, "refresh_token": refreshToken, "id_token": idToken,
		"token_type": "Bearer", "expires_in": 3600,
	}
}

type authIntegrationUpstreamCapture struct {
	mu      sync.Mutex
	orgID   string
	userIDs string
}

func (c *authIntegrationUpstreamCapture) handler(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	c.orgID = r.Header.Get("org-id")
	c.userIDs = r.URL.Query().Get("userIds")
	c.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	writeAuthIntegrationJSON(w, map[string]any{
		"data": []any{}, "metadata": map[string]int{"total": 0, "offset": 0, "count": 0, "limit": 10},
	})
}

type authIntegrationFixture struct {
	provider *authIntegrationProvider
	upstream *httptest.Server
	bff      *httptest.Server
	client   *http.Client
	capture  *authIntegrationUpstreamCapture
	cfg      config.Config
}

func newAuthIntegrationFixture(t *testing.T, tokenScopes string) *authIntegrationFixture {
	t.Helper()
	provider := newAuthIntegrationProvider(t, tokenScopes)
	capture := &authIntegrationUpstreamCapture{}
	upstream := httptest.NewServer(http.HandlerFunc(capture.handler))
	t.Cleanup(upstream.Close)

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg := *loaded
	cfg.Auth = config.AuthConfig{
		Enabled: true, IssuerURL: provider.server.URL,
		ClientID: "integration-client", ClientSecret: "integration-secret",
		PortalURL: "http://portal.example/", RedirectURI: "http://bff.example/auth/callback",
		PostLogoutRedirectURI: "http://portal.example/signed-out",
		Scopes:                []string{"openid", "profile"}, ResourceAudience: "integration-api",
		AllowedSigningAlgorithms: []string{"RS256"}, HTTPTimeout: 5 * time.Second,
		RefreshTimeout: 5 * time.Second, ClockSkew: 30 * time.Second,
		ScopeClaim: "scope", OrgIDClaim: "org_id",
		AccessTokenPart1Cookie: "integration-at-p1", AccessTokenPart2Cookie: "integration-at-p2",
		RefreshTokenPart1Cookie: "integration-rt-p1", RefreshTokenPart2Cookie: "integration-rt-p2",
		IDTokenPart1Cookie: "integration-id-p1", IDTokenPart2Cookie: "integration-id-p2",
		OAuthStateCookie: "integration-oauth-state", PKCEVerifierCookie: "integration-pkce-verifier",
		CookieSecure: false, CookieSameSite: "Lax", LoginTransactionMaxAgeSeconds: 600,
		RefreshCookieMaxAgeSeconds: 3600,
		MaxTokenPartBytes:          3800, MaxReconstructedTokenBytes: 7600,
	}
	cfg.Proxy.OpenFGCAPIURL = upstream.URL
	cfg.Proxy.PlaceholderModeEnabled = false
	cfg.Proxy.PlaceholderUserID = ""
	cfg.Proxy.PlaceholderOrgID = ""
	cfg.CORS = config.CORSConfig{
		AllowedOrigins:   []string{"http://frontend.example"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Correlation-ID"},
		AllowCredentials: true,
	}
	handler, err := newIntegrationHandler(cfg)
	if err != nil {
		t.Fatalf("assemble BFF: %v", err)
	}
	bff := httptest.NewServer(handler)
	t.Cleanup(bff.Close)
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	return &authIntegrationFixture{provider: provider, upstream: upstream, bff: bff, client: client, capture: capture, cfg: cfg}
}

func (f *authIntegrationFixture) login(t *testing.T) map[string]*http.Cookie {
	t.Helper()
	loginResponse, err := f.client.Get(f.bff.URL + "/auth/login")
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	_ = loginResponse.Body.Close()
	if loginResponse.StatusCode != http.StatusFound {
		t.Fatalf("login returned %d", loginResponse.StatusCode)
	}
	authorizationURL, err := url.Parse(loginResponse.Header.Get("Location"))
	if err != nil || authorizationURL.Host != strings.TrimPrefix(f.provider.server.URL, "http://") {
		t.Fatalf("unexpected authorization redirect: %q", loginResponse.Header.Get("Location"))
	}
	if authorizationURL.Query().Get("client_id") != f.cfg.Auth.ClientID || authorizationURL.Query().Get("redirect_uri") != f.cfg.Auth.RedirectURI {
		t.Fatalf("authorization redirect has incorrect client parameters: %s", authorizationURL.RawQuery)
	}
	state := authorizationURL.Query().Get("state")
	if state == "" || authorizationURL.Query().Get("code_challenge") == "" || authorizationURL.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("authorization redirect is missing state or S256 PKCE: %s", authorizationURL.RawQuery)
	}
	loginCookies := authIntegrationCookiesByName(loginResponse.Cookies())
	if loginCookies[f.cfg.Auth.OAuthStateCookie] == nil || loginCookies[f.cfg.Auth.PKCEVerifierCookie] == nil {
		t.Fatal("login did not issue transaction cookies")
	}

	callbackRequest, _ := http.NewRequest(http.MethodGet, f.bff.URL+"/auth/callback?code=integration-code&state="+url.QueryEscape(state), nil)
	callbackRequest.AddCookie(loginCookies[f.cfg.Auth.OAuthStateCookie])
	callbackRequest.AddCookie(loginCookies[f.cfg.Auth.PKCEVerifierCookie])
	callbackResponse, err := f.client.Do(callbackRequest)
	if err != nil {
		t.Fatalf("callback request: %v", err)
	}
	_ = callbackResponse.Body.Close()
	if callbackResponse.StatusCode != http.StatusFound || callbackResponse.Header.Get("Location") != f.cfg.Auth.PortalURL {
		t.Fatalf("unexpected callback response: %d %q", callbackResponse.StatusCode, callbackResponse.Header.Get("Location"))
	}
	replayResponse, err := f.client.Get(f.bff.URL + "/auth/callback?code=integration-code&state=" + url.QueryEscape(state))
	if err != nil {
		t.Fatalf("replayed callback request: %v", err)
	}
	_ = replayResponse.Body.Close()
	replayLocation, parseErr := url.Parse(replayResponse.Header.Get("Location"))
	if replayResponse.StatusCode != http.StatusFound || parseErr != nil || replayLocation.Query().Get("auth_error") != "login_failed" {
		t.Fatalf("replayed callback was not rejected: %d %q", replayResponse.StatusCode, replayResponse.Header.Get("Location"))
	}
	cookies := authIntegrationCookiesByName(callbackResponse.Cookies())
	for _, name := range []string{
		f.cfg.Auth.AccessTokenPart1Cookie, f.cfg.Auth.AccessTokenPart2Cookie,
		f.cfg.Auth.RefreshTokenPart1Cookie, f.cfg.Auth.RefreshTokenPart2Cookie,
		f.cfg.Auth.IDTokenPart1Cookie, f.cfg.Auth.IDTokenPart2Cookie,
	} {
		if cookies[name] == nil {
			t.Fatalf("callback did not issue %s", name)
		}
	}
	return cookies
}

func TestOIDCAuthLifecycleThroughAssembledBFF(t *testing.T) {
	fixture := newAuthIntegrationFixture(t, strings.Join(auth.AllPortalScopes, " "))
	cookies := fixture.login(t)

	currentUserRequest, _ := http.NewRequest(http.MethodGet, fixture.bff.URL+"/me", nil)
	addAuthIntegrationAccess(currentUserRequest, fixture.cfg.Auth, cookies)
	currentUserResponse, err := fixture.client.Do(currentUserRequest)
	if err != nil {
		t.Fatalf("current user request: %v", err)
	}
	if currentUserResponse.StatusCode != http.StatusOK {
		_ = currentUserResponse.Body.Close()
		t.Fatalf("current user returned %d", currentUserResponse.StatusCode)
	}
	var currentUserPayload struct {
		UserID         string   `json:"userId"`
		OrganizationID string   `json:"organizationId"`
		Scopes         []string `json:"scopes"`
	}
	if err := json.NewDecoder(currentUserResponse.Body).Decode(&currentUserPayload); err != nil {
		_ = currentUserResponse.Body.Close()
		t.Fatalf("decode current user: %v", err)
	}
	_ = currentUserResponse.Body.Close()
	if currentUserPayload.UserID != "integration-user" || currentUserPayload.OrganizationID != "integration-org" {
		t.Fatalf("unexpected current user: %+v", currentUserPayload)
	}
	if len(currentUserPayload.Scopes) != len(auth.AllPortalScopes) {
		t.Fatalf("current user scopes = %v", currentUserPayload.Scopes)
	}

	apiRequest, _ := http.NewRequest(http.MethodGet, fixture.bff.URL+"/api/consents?limit=10", nil)
	addAuthIntegrationAccess(apiRequest, fixture.cfg.Auth, cookies)
	apiResponse, err := fixture.client.Do(apiRequest)
	if err != nil {
		t.Fatalf("protected API request: %v", err)
	}
	_ = apiResponse.Body.Close()
	if apiResponse.StatusCode != http.StatusOK {
		t.Fatalf("protected API returned %d", apiResponse.StatusCode)
	}
	fixture.capture.mu.Lock()
	if fixture.capture.orgID != "integration-org" {
		t.Errorf("upstream received org-id %q", fixture.capture.orgID)
	}
	fixture.capture.mu.Unlock()

	meRequest, _ := http.NewRequest(http.MethodGet, fixture.bff.URL+"/me/consents?userIds=attacker", nil)
	addAuthIntegrationAccess(meRequest, fixture.cfg.Auth, cookies)
	meResponse, err := fixture.client.Do(meRequest)
	if err != nil {
		t.Fatalf("protected me request: %v", err)
	}
	_ = meResponse.Body.Close()
	if meResponse.StatusCode != http.StatusOK {
		t.Fatalf("protected me route returned %d", meResponse.StatusCode)
	}
	fixture.capture.mu.Lock()
	if fixture.capture.userIDs != "integration-user" {
		t.Errorf("upstream received userIds %q", fixture.capture.userIDs)
	}
	fixture.capture.mu.Unlock()

	refreshBody := url.Values{"refresh_token": {cookies[fixture.cfg.Auth.RefreshTokenPart1Cookie].Value}}.Encode()
	refreshRequest, _ := http.NewRequest(http.MethodPost, fixture.bff.URL+"/auth/refresh", strings.NewReader(refreshBody))
	refreshRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	refreshRequest.AddCookie(cookies[fixture.cfg.Auth.RefreshTokenPart2Cookie])
	refreshResponse, err := fixture.client.Do(refreshRequest)
	if err != nil {
		t.Fatalf("refresh request: %v", err)
	}
	_ = refreshResponse.Body.Close()
	if refreshResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("refresh returned %d", refreshResponse.StatusCode)
	}
	mergeAuthIntegrationCookies(cookies, refreshResponse.Cookies())
	fixture.provider.mu.Lock()
	if fixture.provider.refreshCalls != 1 {
		t.Errorf("expected one IdP refresh, got %d", fixture.provider.refreshCalls)
	}
	fixture.provider.mu.Unlock()

	logoutRequest, _ := http.NewRequest(http.MethodPost, fixture.bff.URL+"/auth/logout", nil)
	addAuthIntegrationAccess(logoutRequest, fixture.cfg.Auth, cookies)
	logoutRequest.AddCookie(cookies[fixture.cfg.Auth.IDTokenPart1Cookie])
	logoutRequest.AddCookie(cookies[fixture.cfg.Auth.IDTokenPart2Cookie])
	logoutResponse, err := fixture.client.Do(logoutRequest)
	if err != nil {
		t.Fatalf("logout request: %v", err)
	}
	defer func() { _ = logoutResponse.Body.Close() }()
	if logoutResponse.StatusCode != http.StatusOK {
		t.Fatalf("logout returned %d", logoutResponse.StatusCode)
	}
	var logoutPayload map[string]string
	if err := json.NewDecoder(logoutResponse.Body).Decode(&logoutPayload); err != nil {
		t.Fatalf("decode logout response: %v", err)
	}
	logoutURL, err := url.Parse(logoutPayload["logoutUrl"])
	if err != nil || logoutURL.Query().Get("id_token_hint") == "" || logoutURL.Query().Get("post_logout_redirect_uri") != fixture.cfg.Auth.PostLogoutRedirectURI {
		t.Fatalf("unexpected logout URL: %q", logoutPayload["logoutUrl"])
	}
	cleared := 0
	for _, cookie := range logoutResponse.Cookies() {
		if cookie.MaxAge == -1 {
			cleared++
		}
	}
	if cleared != 6 {
		t.Fatalf("logout cleared %d cookies, want 6", cleared)
	}
}

func TestSplitAuthenticationFailuresThroughAssembledBFF(t *testing.T) {
	fixture := newAuthIntegrationFixture(t, strings.Join(auth.AllPortalScopes, " "))
	cookies := fixture.login(t)
	part1 := cookies[fixture.cfg.Auth.AccessTokenPart1Cookie].Value
	part2 := cookies[fixture.cfg.Auth.AccessTokenPart2Cookie].Value
	fullToken := part1 + part2

	tests := []struct {
		name          string
		authorization []string
		cookieHeader  string
	}{
		{name: "missing readable half", cookieHeader: fixture.cfg.Auth.AccessTokenPart2Cookie + "=" + part2},
		{name: "missing secure half", authorization: []string{"Bearer " + part1}},
		{name: "complete token injection", authorization: []string{"Bearer " + fullToken}, cookieHeader: fixture.cfg.Auth.AccessTokenPart2Cookie + "=" + part2},
		{name: "duplicate bearer", authorization: []string{"Bearer " + part1, "Bearer " + part1}, cookieHeader: fixture.cfg.Auth.AccessTokenPart2Cookie + "=" + part2},
		{name: "duplicate secure cookie", authorization: []string{"Bearer " + part1}, cookieHeader: fixture.cfg.Auth.AccessTokenPart2Cookie + "=" + part2 + "; " + fixture.cfg.Auth.AccessTokenPart2Cookie + "=" + part2},
		{name: "tampered readable half", authorization: []string{"Bearer x" + part1[1:]}, cookieHeader: fixture.cfg.Auth.AccessTokenPart2Cookie + "=" + part2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodGet, fixture.bff.URL+"/api/consents", nil)
			for _, value := range test.authorization {
				request.Header.Add("Authorization", value)
			}
			if test.cookieHeader != "" {
				request.Header.Set("Cookie", test.cookieHeader)
			}
			response, err := fixture.client.Do(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			_ = response.Body.Close()
			if response.StatusCode != http.StatusUnauthorized || response.Header.Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("got status %d and challenge %q", response.StatusCode, response.Header.Get("WWW-Authenticate"))
			}
		})
	}
}

func TestScopeAndCORSAuthorizationThroughAssembledBFF(t *testing.T) {
	fixture := newAuthIntegrationFixture(t, auth.ScopeConsentsReadSelf)
	cookies := fixture.login(t)

	apiRequest, _ := http.NewRequest(http.MethodGet, fixture.bff.URL+"/api/consents", nil)
	addAuthIntegrationAccess(apiRequest, fixture.cfg.Auth, cookies)
	apiResponse, err := fixture.client.Do(apiRequest)
	if err != nil {
		t.Fatal(err)
	}
	_ = apiResponse.Body.Close()
	if apiResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("self scope authorized /api route with status %d", apiResponse.StatusCode)
	}

	meRequest, _ := http.NewRequest(http.MethodGet, fixture.bff.URL+"/me/consents", nil)
	addAuthIntegrationAccess(meRequest, fixture.cfg.Auth, cookies)
	meResponse, err := fixture.client.Do(meRequest)
	if err != nil {
		t.Fatal(err)
	}
	_ = meResponse.Body.Close()
	if meResponse.StatusCode != http.StatusOK {
		t.Fatalf("self scope failed /me route with status %d", meResponse.StatusCode)
	}

	preflight, _ := http.NewRequest(http.MethodOptions, fixture.bff.URL+"/api/consents", nil)
	preflight.Header.Set("Origin", "http://frontend.example")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight.Header.Set("Access-Control-Request-Headers", "Authorization")
	preflightResponse, err := fixture.client.Do(preflight)
	if err != nil {
		t.Fatal(err)
	}
	_ = preflightResponse.Body.Close()
	if preflightResponse.StatusCode != http.StatusNoContent ||
		preflightResponse.Header.Get("Access-Control-Allow-Origin") != "http://frontend.example" ||
		preflightResponse.Header.Get("Access-Control-Allow-Credentials") != "true" ||
		!strings.Contains(preflightResponse.Header.Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("unexpected credentialed preflight response: %d %#v", preflightResponse.StatusCode, preflightResponse.Header)
	}
}

func addAuthIntegrationAccess(request *http.Request, cfg config.AuthConfig, cookies map[string]*http.Cookie) {
	request.Header.Set("Authorization", "Bearer "+cookies[cfg.AccessTokenPart1Cookie].Value)
	request.AddCookie(cookies[cfg.AccessTokenPart2Cookie])
}

func authIntegrationCookiesByName(cookies []*http.Cookie) map[string]*http.Cookie {
	result := make(map[string]*http.Cookie, len(cookies))
	for _, cookie := range cookies {
		result[cookie.Name] = cookie
	}
	return result
}

func mergeAuthIntegrationCookies(destination map[string]*http.Cookie, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie.MaxAge >= 0 {
			destination[cookie.Name] = cookie
		}
	}
}

func signAuthIntegrationToken(t *testing.T, key *rsa.PrivateKey, keyID string, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID))
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return serialized
}

func writeAuthIntegrationJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
