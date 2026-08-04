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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/wso2/openfgc/portal/backend/internal/system/config"
)

func TestOIDCLoginCallbackRefreshAndLogout(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyID := "test-key"
	var issuer *httptest.Server
	var refreshCalls int
	var expectedPKCEChallenge string
	issuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeTestJSON(w, map[string]any{
				"issuer": issuer.URL, "authorization_endpoint": issuer.URL + "/authorize",
				"token_endpoint": issuer.URL + "/token", "jwks_uri": issuer.URL + "/jwks",
				"end_session_endpoint":                  issuer.URL + "/logout",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/jwks":
			writeTestJSON(w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
				Key: &privateKey.PublicKey, KeyID: keyID, Algorithm: "RS256", Use: "sig",
			}}})
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse token form: %v", err)
			}
			if clientID, clientSecret, ok := r.BasicAuth(); !ok || clientID != "portal-client" || clientSecret != "secret" {
				t.Errorf("unexpected confidential-client authentication")
			}
			if r.Form.Get("grant_type") != "refresh_token" {
				verifier := r.Form.Get("code_verifier")
				if !validPKCEVerifier(verifier) || pkceChallenge(verifier) != expectedPKCEChallenge {
					t.Errorf("authorization-code exchange did not contain the matching PKCE verifier")
				}
			}
			now := time.Now()
			access := signTestToken(t, privateKey, keyID, map[string]any{
				"iss": issuer.URL, "sub": "user-1", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
				"iat": now.Unix(), "nbf": now.Add(-time.Minute).Unix(), "org_id": "org-1",
				"scope": ScopeConsentsReadSelf + " " + ScopeConsentsWriteSelf,
			})
			idToken := signTestToken(t, privateKey, keyID, map[string]any{
				"iss": issuer.URL, "sub": "user-1", "aud": "portal-client", "exp": now.Add(time.Hour).Unix(), "iat": now.Unix(),
			})
			refresh := "refresh-token-initial"
			if r.Form.Get("grant_type") == "refresh_token" {
				refreshCalls++
				refresh = "refresh-token-rotated"
			}
			writeTestJSON(w, map[string]any{
				"access_token": access, "refresh_token": refresh, "id_token": idToken,
				"token_type": "Bearer", "expires_in": 3600,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer issuer.Close()

	cfg := testAuthConfig(issuer.URL)
	manager, err := NewManager(context.Background(), cfg, config.ProxyConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	mux := http.NewServeMux()
	manager.RegisterRoutes(mux)

	login := httptest.NewRecorder()
	mux.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	authorizationURL, parseErr := url.Parse(login.Header().Get("Location"))
	if login.Code != http.StatusFound || parseErr != nil {
		t.Fatalf("unexpected login redirect: %d %s", login.Code, login.Header().Get("Location"))
	}
	state := authorizationURL.Query().Get("state")
	expectedPKCEChallenge = authorizationURL.Query().Get("code_challenge")
	if audience := authorizationURL.Query().Get("audience"); audience != cfg.ResourceAudience {
		t.Fatalf("unexpected authorization audience: %q", audience)
	}
	if state == "" || expectedPKCEChallenge == "" || authorizationURL.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("login redirect missing state or S256 PKCE parameters: %s", authorizationURL.RawQuery)
	}
	transactionCookies := cookiesByName(login.Result().Cookies())
	for _, name := range []string{cfg.OAuthStateCookie, cfg.PKCEVerifierCookie} {
		cookie := transactionCookies[name]
		if cookie == nil || !cookie.HttpOnly || cookie.Path != "/" || cookie.MaxAge != cfg.LoginTransactionMaxAgeSeconds {
			t.Fatalf("invalid login transaction cookie %s: %#v", name, cookie)
		}
	}

	callback := httptest.NewRecorder()
	callbackRequest := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code&state="+url.QueryEscape(state), nil)
	callbackRequest.AddCookie(transactionCookies[cfg.OAuthStateCookie])
	callbackRequest.AddCookie(transactionCookies[cfg.PKCEVerifierCookie])
	mux.ServeHTTP(callback, callbackRequest)
	if callback.Code != http.StatusFound || callback.Header().Get("Location") != cfg.PortalURL {
		t.Fatalf("unexpected callback: %d %s", callback.Code, callback.Header().Get("Location"))
	}
	cookies := cookiesByName(callback.Result().Cookies())
	for _, name := range []string{cfg.OAuthStateCookie, cfg.PKCEVerifierCookie} {
		if cookies[name] == nil || cookies[name].MaxAge != -1 {
			t.Fatalf("callback did not consume login transaction cookie %s", name)
		}
	}
	for _, name := range []string{cfg.AccessTokenPart1Cookie, cfg.AccessTokenPart2Cookie, cfg.RefreshTokenPart1Cookie, cfg.RefreshTokenPart2Cookie, cfg.IDTokenPart1Cookie, cfg.IDTokenPart2Cookie} {
		if cookies[name] == nil {
			t.Fatalf("missing cookie %s", name)
		}
	}
	if cookies[cfg.AccessTokenPart1Cookie].HttpOnly || !cookies[cfg.AccessTokenPart2Cookie].HttpOnly || cookies[cfg.IDTokenPart2Cookie].HttpOnly {
		t.Fatal("unexpected split-cookie visibility")
	}
	protectedRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	protectedRequest.Header.Set("Authorization", "Bearer "+cookies[cfg.AccessTokenPart1Cookie].Value)
	protectedRequest.AddCookie(cookies[cfg.AccessTokenPart2Cookie])
	forbidden := httptest.NewRecorder()
	manager.Require(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), ScopeElementsWrite).ServeHTTP(forbidden, protectedRequest)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("expected missing scope to return 403, got %d", forbidden.Code)
	}

	refreshPart := cookies[cfg.RefreshTokenPart1Cookie].Value
	refreshRequest := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(url.Values{"refresh_token": {refreshPart}}.Encode()))
	refreshRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	refreshRequest.AddCookie(cookies[cfg.RefreshTokenPart2Cookie])
	refresh := httptest.NewRecorder()
	mux.ServeHTTP(refresh, refreshRequest)
	if refresh.Code != http.StatusNoContent || refreshCalls != 1 {
		t.Fatalf("unexpected refresh: status=%d calls=%d", refresh.Code, refreshCalls)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutRequest.Header.Set("Authorization", "Bearer "+cookies[cfg.AccessTokenPart1Cookie].Value)
	for _, cookie := range cookies {
		logoutRequest.AddCookie(cookie)
	}
	logout := httptest.NewRecorder()
	mux.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusOK {
		t.Fatalf("unexpected logout status: %d", logout.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(logout.Body.Bytes(), &payload); err != nil || !strings.HasPrefix(payload["logoutUrl"], issuer.URL+"/logout?") {
		t.Fatalf("unexpected logout payload: %s", logout.Body.String())
	}
}

func TestCallbackFailureMappingAndRedaction(t *testing.T) {
	tests := []struct {
		name       string
		requestURL string
		token      http.HandlerFunc
		maxPart    int
	}{
		{name: "missing code", requestURL: "/auth/callback"},
		{name: "duplicate code", requestURL: "/auth/callback?code=secret-code&code=second"},
		{
			name: "token endpoint rejection", requestURL: "/auth/callback?code=secret-code",
			token: func(w http.ResponseWriter, _ *http.Request) {
				writeTestJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "sensitive detail"})
			},
		},
		{
			name: "incomplete token response", requestURL: "/auth/callback?code=secret-code",
			token: func(w http.ResponseWriter, _ *http.Request) {
				writeTestJSON(w, map[string]any{"access_token": "access", "token_type": "Bearer"})
			},
		},
		{
			name: "cookie limit failure", requestURL: "/auth/callback?code=secret-code", maxPart: 8,
			token: func(w http.ResponseWriter, r *http.Request) {
				p := providerFromRequestContext(t, r)
				writeValidTokenResponse(t, w, p, "refresh-token", true)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := newOIDCValidationProvider(t)
			if test.token != nil {
				provider.token = func(w http.ResponseWriter, r *http.Request) {
					r = r.WithContext(context.WithValue(r.Context(), validationProviderContextKey{}, provider))
					test.token(w, r)
				}
			}
			var logs bytes.Buffer
			cfg := testAuthConfig(provider.server.URL)
			if test.maxPart > 0 {
				cfg.MaxTokenPartBytes = test.maxPart
			}
			manager, err := NewManager(context.Background(), cfg, config.ProxyConfig{}, slog.New(slog.NewTextHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			state, transactionCookies := startLoginTransaction(t, manager)
			request := httptest.NewRequest(http.MethodGet, test.requestURL, nil)
			query := request.URL.Query()
			query.Set("state", state)
			request.URL.RawQuery = query.Encode()
			request.AddCookie(transactionCookies[cfg.OAuthStateCookie])
			request.AddCookie(transactionCookies[cfg.PKCEVerifierCookie])
			manager.Callback(recorder, request)
			if recorder.Code != http.StatusFound {
				t.Fatalf("expected generic redirect, got %d", recorder.Code)
			}
			location, err := url.Parse(recorder.Header().Get("Location"))
			if err != nil || location.Query().Get("auth_error") != "login_failed" {
				t.Fatalf("unexpected failure location: %q", recorder.Header().Get("Location"))
			}
			if strings.Contains(recorder.Body.String(), "secret-code") || strings.Contains(logs.String(), "secret-code") || strings.Contains(logs.String(), "sensitive detail") {
				t.Fatalf("callback leaked sensitive details: body=%q logs=%q", recorder.Body.String(), logs.String())
			}
		})
	}
}

func TestCallbackRejectsMissingMismatchedAndReplayedState(t *testing.T) {
	provider := newOIDCValidationProvider(t)
	var tokenCalls atomic.Int32
	provider.token = func(w http.ResponseWriter, _ *http.Request) {
		tokenCalls.Add(1)
		writeValidTokenResponse(t, w, provider, "refresh-token", true)
	}
	manager := provider.manager(t, nil)

	for _, test := range []struct {
		name  string
		state func(string) []string
	}{
		{name: "missing state", state: func(string) []string { return nil }},
		{name: "mismatched state", state: func(string) []string { return []string{"different-state"} }},
		{name: "duplicate state", state: func(value string) []string { return []string{value, value} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, cookies := startLoginTransaction(t, manager)
			request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code", nil)
			query := request.URL.Query()
			query["state"] = test.state(state)
			request.URL.RawQuery = query.Encode()
			request.AddCookie(cookies[manager.cfg.OAuthStateCookie])
			request.AddCookie(cookies[manager.cfg.PKCEVerifierCookie])

			recorder := httptest.NewRecorder()
			manager.Callback(recorder, request)

			assertCallbackFailure(t, recorder)
			assertTransactionCookiesCleared(t, recorder, manager.cfg)
		})
	}

	state, cookies := startLoginTransaction(t, manager)
	newRequest := func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code&state="+url.QueryEscape(state), nil)
		request.AddCookie(cookies[manager.cfg.OAuthStateCookie])
		request.AddCookie(cookies[manager.cfg.PKCEVerifierCookie])
		return request
	}
	success := httptest.NewRecorder()
	manager.Callback(success, newRequest())
	if success.Code != http.StatusFound || success.Header().Get("Location") != manager.cfg.PortalURL {
		t.Fatalf("valid callback failed: %d %q", success.Code, success.Header().Get("Location"))
	}

	replay := httptest.NewRecorder()
	replayRequest := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code&state="+url.QueryEscape(state), nil)
	manager.Callback(replay, replayRequest)
	assertCallbackFailure(t, replay)
	if tokenCalls.Load() != 1 {
		t.Fatalf("replayed callback reached token endpoint; calls=%d", tokenCalls.Load())
	}
}

func assertCallbackFailure(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusFound {
		t.Fatalf("expected callback failure redirect, got %d", recorder.Code)
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil || location.Query().Get("auth_error") != "login_failed" {
		t.Fatalf("unexpected callback failure location: %q", recorder.Header().Get("Location"))
	}
}

func assertTransactionCookiesCleared(t *testing.T, recorder *httptest.ResponseRecorder, cfg config.AuthConfig) {
	t.Helper()
	cookies := cookiesByName(recorder.Result().Cookies())
	for _, name := range []string{cfg.OAuthStateCookie, cfg.PKCEVerifierCookie} {
		if cookies[name] == nil || cookies[name].MaxAge != -1 || !cookies[name].HttpOnly {
			t.Fatalf("transaction cookie %s was not securely consumed: %#v", name, cookies[name])
		}
	}
}

func TestRefreshRequestValidation(t *testing.T) {
	manager := &Manager{cfg: config.AuthConfig{
		Enabled: true, RefreshTokenPart2Cookie: "rt2", MaxTokenPartBytes: 32, MaxReconstructedTokenBytes: 64,
	}}
	tests := []struct {
		name        string
		contentType string
		body        string
		cookie      string
		wantStatus  int
	}{
		{name: "wrong content type", contentType: "application/json", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "missing readable part", contentType: "application/x-www-form-urlencoded", body: "", cookie: "rt2=part2", wantStatus: http.StatusUnauthorized},
		{name: "duplicate readable part", contentType: "application/x-www-form-urlencoded", body: "refresh_token=a&refresh_token=b", cookie: "rt2=part2", wantStatus: http.StatusUnauthorized},
		{name: "extra form field", contentType: "application/x-www-form-urlencoded", body: "refresh_token=a&extra=b", cookie: "rt2=part2", wantStatus: http.StatusUnauthorized},
		{name: "missing secure part", contentType: "application/x-www-form-urlencoded", body: "refresh_token=part1", wantStatus: http.StatusUnauthorized},
		{name: "duplicate secure part", contentType: "application/x-www-form-urlencoded", body: "refresh_token=part1", cookie: "rt2=a; rt2=b", wantStatus: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(test.body))
			r.Header.Set("Content-Type", test.contentType)
			if test.cookie != "" {
				r.Header.Set("Cookie", test.cookie)
			}
			recorder := httptest.NewRecorder()
			manager.Refresh(recorder, r)
			if recorder.Code != test.wantStatus {
				t.Fatalf("got %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestRefreshRotationRetentionFailureAndAtomicCookies(t *testing.T) {
	tests := []struct {
		name              string
		refreshReturned   string
		invalidGrant      bool
		invalidIDToken    bool
		wantStatus        int
		wantRefreshCookie bool
		wantClears        bool
	}{
		{name: "rotated refresh token", refreshReturned: "refresh-rotated", wantStatus: http.StatusNoContent, wantRefreshCookie: true},
		{name: "retain existing refresh token", wantStatus: http.StatusNoContent},
		{name: "invalid or expired refresh token", invalidGrant: true, wantStatus: http.StatusUnauthorized, wantClears: true},
		{name: "invalid new ID token clears atomically", refreshReturned: "refresh-rotated", invalidIDToken: true, wantStatus: http.StatusUnauthorized, wantClears: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := newOIDCValidationProvider(t)
			provider.token = func(w http.ResponseWriter, _ *http.Request) {
				if test.invalidGrant {
					writeTestJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
					return
				}
				writeRefreshResponse(t, w, provider, test.refreshReturned, test.invalidIDToken)
			}
			manager := provider.manager(t, nil)
			recorder := performRefresh(t, manager, "refresh-original")
			if recorder.Code != test.wantStatus {
				t.Fatalf("got %d, want %d: %s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			cookies := recorder.Result().Cookies()
			refreshCookieFound := false
			clearCount := 0
			for _, cookie := range cookies {
				if cookie.Name == manager.cfg.RefreshTokenPart1Cookie && cookie.MaxAge > 0 {
					refreshCookieFound = true
				}
				if cookie.MaxAge == -1 {
					clearCount++
				}
			}
			if refreshCookieFound != test.wantRefreshCookie {
				t.Fatalf("refresh cookie emitted=%v, want %v", refreshCookieFound, test.wantRefreshCookie)
			}
			if test.wantClears && clearCount != 6 {
				t.Fatalf("expected all six cookies to be cleared, got %d", clearCount)
			}
			if test.wantClears {
				assertLastCookieForEachNameClears(t, cookies)
			}
		})
	}
}

func TestConcurrentRefreshWithNonRotatingToken(t *testing.T) {
	provider := newOIDCValidationProvider(t)
	var calls atomic.Int32
	provider.token = func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writeRefreshResponse(t, w, provider, "", false)
	}
	manager := provider.manager(t, nil)

	const requestCount = 4
	statuses := make(chan int, requestCount)
	var wait sync.WaitGroup
	for range requestCount {
		wait.Add(1)
		go func() {
			defer wait.Done()
			statuses <- performRefresh(t, manager, "refresh-original").Code
		}()
	}
	wait.Wait()
	close(statuses)
	for status := range statuses {
		if status != http.StatusNoContent {
			t.Errorf("concurrent refresh returned %d", status)
		}
	}
	if calls.Load() != requestCount {
		t.Fatalf("expected %d independent refresh exchanges, got %d", requestCount, calls.Load())
	}
}

func TestCookieOnlyLogoutRejectedAndSuccessfulLogoutClearsCookies(t *testing.T) {
	provider := newOIDCValidationProvider(t)
	manager := provider.manager(t, nil)
	mux := http.NewServeMux()
	manager.RegisterRoutes(mux)

	cookieOnly := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	cookieOnly.AddCookie(&http.Cookie{Name: manager.cfg.AccessTokenPart2Cookie, Value: "part2"})
	rejected := httptest.NewRecorder()
	mux.ServeHTTP(rejected, cookieOnly)
	if rejected.Code != http.StatusUnauthorized || rejected.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("cookie-only logout was not rejected correctly: %d", rejected.Code)
	}

	now := time.Now()
	access := signTestToken(t, provider.key1, "key-1", map[string]any{
		"iss": provider.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
		"org_id": "org", "scope": ScopeConsentsReadSelf,
	})
	part1, part2, err := splitToken(access, manager.cfg.MaxTokenPartBytes)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer "+part1)
	request.AddCookie(&http.Cookie{Name: manager.cfg.AccessTokenPart2Cookie, Value: part2})
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("logout returned %d: %s", recorder.Code, recorder.Body.String())
	}
	assertLastCookieForEachNameClears(t, recorder.Result().Cookies())
	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil || payload["logoutUrl"] != manager.cfg.PostLogoutRedirectURI {
		t.Fatalf("expected post-logout fallback URL, got %s", recorder.Body.String())
	}
}

type validationProviderContextKey struct{}

func providerFromRequestContext(t *testing.T, r *http.Request) *oidcValidationProvider {
	t.Helper()
	provider, ok := r.Context().Value(validationProviderContextKey{}).(*oidcValidationProvider)
	if !ok {
		t.Fatal("validation provider missing from request context")
	}
	return provider
}

func writeValidTokenResponse(t *testing.T, w http.ResponseWriter, provider *oidcValidationProvider, refresh string, includeID bool) {
	t.Helper()
	now := time.Now()
	response := map[string]any{
		"access_token": signTestToken(t, provider.key1, "key-1", map[string]any{
			"iss": provider.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
			"org_id": "org", "scope": ScopeConsentsReadSelf,
		}),
		"refresh_token": refresh, "token_type": "Bearer", "expires_in": 3600,
	}
	if includeID {
		response["id_token"] = signTestToken(t, provider.key1, "key-1", map[string]any{
			"iss": provider.server.URL, "sub": "user", "aud": "portal-client", "exp": now.Add(time.Hour).Unix(),
		})
	}
	writeTestJSON(w, response)
}

func writeRefreshResponse(t *testing.T, w http.ResponseWriter, provider *oidcValidationProvider, refresh string, invalidID bool) {
	t.Helper()
	now := time.Now()
	response := map[string]any{
		"access_token": signTestToken(t, provider.key1, "key-1", map[string]any{
			"iss": provider.server.URL, "sub": "user", "aud": "portal-api", "exp": now.Add(time.Hour).Unix(),
			"org_id": "org", "scope": ScopeConsentsReadSelf,
		}),
		"token_type": "Bearer", "expires_in": 3600,
	}
	if refresh != "" {
		response["refresh_token"] = refresh
	}
	if invalidID {
		response["id_token"] = "not-a-jwt"
	} else {
		response["id_token"] = signTestToken(t, provider.key1, "key-1", map[string]any{
			"iss": provider.server.URL, "sub": "user", "aud": "portal-client", "exp": now.Add(time.Hour).Unix(),
		})
	}
	writeTestJSON(w, response)
}

func performRefresh(t *testing.T, manager *Manager, refreshToken string) *httptest.ResponseRecorder {
	t.Helper()
	part1, part2, err := splitToken(refreshToken, manager.cfg.MaxTokenPartBytes)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(url.Values{"refresh_token": {part1}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: manager.cfg.RefreshTokenPart2Cookie, Value: part2})
	recorder := httptest.NewRecorder()
	manager.Refresh(recorder, request)
	return recorder
}

func assertLastCookieForEachNameClears(t *testing.T, cookies []*http.Cookie) {
	t.Helper()
	last := make(map[string]*http.Cookie)
	for _, cookie := range cookies {
		last[cookie.Name] = cookie
	}
	if len(last) != 6 {
		t.Fatalf("expected all six token cookie names, got %d", len(last))
	}
	for name, cookie := range last {
		if cookie.MaxAge != -1 || cookie.Value != "" || cookie.Path != "/" {
			t.Fatalf("cookie %s was not finally cleared: %#v", name, cookie)
		}
	}
}

func writeTestJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
