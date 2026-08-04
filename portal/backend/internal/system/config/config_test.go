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

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("BFF_SERVER__PORT", "8082")
	t.Setenv("BFF_LOG__LEVEL", "debug")
	t.Setenv("BFF_CORS__ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	t.Setenv("BFF_PROXY__MAX_RESPONSE_BYTES", "2097152")
	_ = os.Unsetenv("BFF_CONFIG_FILE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if cfg.Server.Port != 8082 {
		t.Fatalf("expected port 8082, got %d", cfg.Server.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("expected log level debug, got %s", cfg.Log.Level)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 cors origins, got %d", len(cfg.CORS.AllowedOrigins))
	}
	if cfg.CORS.AllowedOrigins[0] != "http://localhost:3000" {
		t.Fatalf("unexpected first origin: %s", cfg.CORS.AllowedOrigins[0])
	}
	if cfg.Proxy.MaxResponseBytes != 2097152 {
		t.Fatalf("expected max response bytes 2097152, got %d", cfg.Proxy.MaxResponseBytes)
	}
}

func TestOrgIDOverrideFromEnv(t *testing.T) {
	t.Setenv("BFF_AUTH__ORG_ID_OVERRIDE", "ORG-001")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}
	if cfg.Auth.OrgIDOverride != "ORG-001" {
		t.Fatalf("unexpected organization override: %q", cfg.Auth.OrgIDOverride)
	}
}

func TestOrgIDClaimCanBeEmptyWhenOverrideIsConfigured(t *testing.T) {
	cfg := validAuthValidationConfig("development")
	cfg.Auth.OrgIDClaim = ""
	cfg.Auth.OrgIDOverride = "ORG-001"

	if err := validateAuth(cfg); err != nil {
		t.Fatalf("expected organization override to replace claim requirement: %v", err)
	}
}

func TestInvalidCORSOriginRejected(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		errText string
	}{
		{name: "invalid url", origin: "http://[::1", errText: "invalid URL"},
		{name: "contains path", origin: "http://localhost:3000/some/path", errText: "must not contain a path"},
		{name: "contains query", origin: "http://localhost:3000?debug=true", errText: "must not contain a query string"},
		{name: "contains fragment", origin: "http://localhost:3000#app", errText: "must not contain a fragment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BFF_CORS__ALLOWED_ORIGINS", tt.origin)

			_, err := Load()
			if err == nil {
				t.Fatal("expected config load to fail for invalid CORS origin")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("expected error to contain %q, got %v", tt.errText, err)
			}
		})
	}
}

func TestAllowCredentialsRequiresNonWildcardOrigins(t *testing.T) {
	t.Setenv("BFF_CORS__ALLOW_CREDENTIALS", "true")
	t.Setenv("BFF_CORS__ALLOWED_ORIGINS", "*")

	_, err := Load()
	if err == nil {
		t.Fatal("expected config load to fail for wildcard origins with credentials")
	}
	if !strings.Contains(err.Error(), "cannot contain wildcard") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAllowCredentialsEnvParsing(t *testing.T) {
	t.Setenv("BFF_CORS__ALLOW_CREDENTIALS", "true")
	t.Setenv("BFF_CORS__ALLOWED_ORIGINS", "http://localhost:5173")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load success, got: %v", err)
	}
	if !cfg.CORS.AllowCredentials {
		t.Fatal("expected cors.allow_credentials to be true")
	}
}

func TestPlaceholderModeBlockedInProduction(t *testing.T) {
	t.Setenv("BFF_ENV", "production")
	t.Setenv("BFF_PROXY__PLACEHOLDER_MODE_ENABLED", "true")
	t.Setenv("BFF_PROXY__PLACEHOLDER_USER_ID", "user@example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when placeholder mode is enabled in production")
	}
	if !strings.Contains(err.Error(), "cannot be true in production") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaceholderValuesRejectedWhenModeDisabled(t *testing.T) {
	tests := []struct {
		name    string
		envName string
		errText string
	}{
		{name: "user id", envName: "BFF_PROXY__PLACEHOLDER_USER_ID", errText: "proxy.placeholder_user_id must be empty"},
		{name: "org id", envName: "BFF_PROXY__PLACEHOLDER_ORG_ID", errText: "proxy.placeholder_org_id must be empty"},
		{name: "scopes", envName: "BFF_PROXY__PLACEHOLDER_SCOPES", errText: "proxy.placeholder_scopes must be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BFF_PROXY__PLACEHOLDER_MODE_ENABLED", "false")
			t.Setenv(tt.envName, "placeholder-value")

			_, err := Load()
			if err == nil {
				t.Fatal("expected error when placeholder value is set while mode is disabled")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("expected error to contain %q, got %v", tt.errText, err)
			}
		})
	}
}

func TestPlaceholderScopesEnv(t *testing.T) {
	t.Setenv("BFF_PROXY__PLACEHOLDER_MODE_ENABLED", "true")
	t.Setenv("BFF_PROXY__PLACEHOLDER_USER_ID", "local-user")
	t.Setenv("BFF_PROXY__PLACEHOLDER_ORG_ID", "local-org")
	t.Setenv("BFF_PROXY__PLACEHOLDER_SCOPES", "portal:consents:read:self portal:elements:read")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}
	want := []string{"portal:consents:read:self", "portal:elements:read"}
	if !reflect.DeepEqual(cfg.Proxy.PlaceholderScopes, want) {
		t.Fatalf("unexpected placeholder scopes: %#v", cfg.Proxy.PlaceholderScopes)
	}
}

func TestAllowedPassthroughMethodsEnvJSON(t *testing.T) {
	t.Setenv("BFF_PROXY__ALLOWED_PASSTHROUGH_METHODS", `["get", "put"]`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if len(cfg.Proxy.AllowedPassthrough) != 2 {
		t.Fatalf("expected 2 allowed methods, got %d", len(cfg.Proxy.AllowedPassthrough))
	}
	if cfg.Proxy.AllowedPassthrough[0] != "GET" || cfg.Proxy.AllowedPassthrough[1] != "PUT" {
		t.Fatalf("unexpected methods: %#v", cfg.Proxy.AllowedPassthrough)
	}
}

func TestOpenFGCAPIURLRequiresHTTPSchemeAndHost(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		errText string
	}{
		{name: "empty url", url: "", errText: "must not be empty"},
		{name: "relative url", url: "/consent-server", errText: "must use http or https scheme"},
		{name: "missing host", url: "http:///api", errText: "must include a host"},
		{name: "unsupported scheme", url: "ftp://localhost:9090", errText: "must use http or https scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BFF_PROXY__OPENFGC_API_URL", tt.url)

			_, err := Load()
			if err == nil {
				t.Fatal("expected config load error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("expected error to contain %q, got %v", tt.errText, err)
			}
		})
	}
}

func TestMaxResponseBytesMustBePositive(t *testing.T) {
	t.Setenv("BFF_PROXY__MAX_RESPONSE_BYTES", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected config load error")
	}
	if !strings.Contains(err.Error(), "proxy.max_response_bytes must be > 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthClientSecretIsEnvironmentOnly(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	contents := `
auth:
  enabled: true
  issuer_url: https://idp.example.com
  client_id: portal-client
  client_secret: file-secret-must-be-ignored
  portal_url: https://portal.example.com/consents
  redirect_uri: https://portal.example.com/auth/callback
  post_logout_redirect_uri: https://portal.example.com/
  resource_audience: portal-api
`
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BFF_CONFIG_FILE", configPath)

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "BFF_AUTH__CLIENT_SECRET") {
		t.Fatalf("expected file secret to be ignored, got %v", err)
	}

	t.Setenv("BFF_AUTH__CLIENT_SECRET", "environment-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected environment secret to load: %v", err)
	}
	if cfg.Auth.ClientSecret != "environment-secret" {
		t.Fatalf("unexpected client secret source")
	}
}

func TestProductionAuthURLsRequireHTTPS(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		update func(*AuthConfig)
	}{
		{name: "issuer URL", field: "auth.issuer_url", update: func(auth *AuthConfig) { auth.IssuerURL = "http://idp.example.com" }},
		{name: "portal URL", field: "auth.portal_url", update: func(auth *AuthConfig) { auth.PortalURL = "http://portal.example.com/consents" }},
		{name: "redirect URI", field: "auth.redirect_uri", update: func(auth *AuthConfig) { auth.RedirectURI = "http://portal.example.com/auth/callback" }},
		{name: "post-logout redirect URI", field: "auth.post_logout_redirect_uri", update: func(auth *AuthConfig) { auth.PostLogoutRedirectURI = "http://portal.example.com/" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validAuthValidationConfig("production")
			test.update(&cfg.Auth)

			err := validateAuth(cfg)
			if err == nil {
				t.Fatal("expected production HTTP URL to be rejected")
			}
			if !strings.Contains(err.Error(), test.field) || !strings.Contains(err.Error(), "must use HTTPS in production") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestProductionAuthURLsAcceptHTTPS(t *testing.T) {
	if err := validateAuth(validAuthValidationConfig("PrOdUcTiOn")); err != nil {
		t.Fatalf("expected production HTTPS URLs to be accepted: %v", err)
	}
}

func TestDevelopmentAuthURLsAllowHTTP(t *testing.T) {
	cfg := validAuthValidationConfig("development")
	cfg.Auth.IssuerURL = "http://localhost:9443/oauth2/token"
	cfg.Auth.PortalURL = "http://localhost:5173/consents"
	cfg.Auth.RedirectURI = "http://localhost:8080/auth/callback"
	cfg.Auth.PostLogoutRedirectURI = "http://localhost:5173/"
	cfg.Auth.CookieSecure = false

	if err := validateAuth(cfg); err != nil {
		t.Fatalf("expected development HTTP URLs to be accepted: %v", err)
	}
}

func TestLoginTransactionConfigurationValidation(t *testing.T) {
	t.Run("maximum age", func(t *testing.T) {
		cfg := validAuthValidationConfig("development")
		cfg.Auth.LoginTransactionMaxAgeSeconds = 601
		if err := validateAuth(cfg); err == nil || !strings.Contains(err.Error(), "login_transaction_max_age_seconds") {
			t.Fatalf("expected invalid transaction maximum age, got %v", err)
		}
	})

	t.Run("unique cookie names", func(t *testing.T) {
		cfg := validAuthValidationConfig("development")
		cfg.Auth.PKCEVerifierCookie = cfg.Auth.OAuthStateCookie
		if err := validateAuth(cfg); err == nil || !strings.Contains(err.Error(), "cookie names must be unique") {
			t.Fatalf("expected duplicate transaction cookie name to fail, got %v", err)
		}
	})
}

func TestAuthURLsRemainAbsoluteHTTPURLsInEveryEnvironment(t *testing.T) {
	for _, value := range []string{"/relative", "ftp://idp.example.com", "https:///missing-host"} {
		if err := validateAbsoluteHTTPURL(value, false); err == nil || !strings.Contains(err.Error(), "absolute HTTP(S) URL") {
			t.Fatalf("expected %q to be rejected, got %v", value, err)
		}
	}
}

func validAuthValidationConfig(environment string) Config {
	return Config{
		Env: environment,
		Auth: AuthConfig{
			Enabled: true, IssuerURL: "https://idp.example.com", ClientID: "portal-client", ClientSecret: "secret",
			PortalURL: "https://portal.example.com/consents", RedirectURI: "https://bff.example.com/auth/callback",
			PostLogoutRedirectURI: "https://portal.example.com/", Scopes: []string{"openid"}, ResourceAudience: "portal-api",
			AllowedSigningAlgorithms: []string{"RS256"}, HTTPTimeout: 5 * time.Second, RefreshTimeout: 5 * time.Second,
			ScopeClaim: "scope", OrgIDClaim: "org_id", CookieSecure: true, CookieSameSite: "Lax",
			RefreshCookieMaxAgeSeconds: 3600, MaxTokenPartBytes: 3800, MaxReconstructedTokenBytes: 7600,
			AccessTokenPart1Cookie: "at-p1", AccessTokenPart2Cookie: "at-p2",
			RefreshTokenPart1Cookie: "rt-p1", RefreshTokenPart2Cookie: "rt-p2",
			IDTokenPart1Cookie: "id-p1", IDTokenPart2Cookie: "id-p2",
			OAuthStateCookie: "oauth-state", PKCEVerifierCookie: "pkce-verifier",
			LoginTransactionMaxAgeSeconds: 600,
		},
	}
}
