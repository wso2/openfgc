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

// Package config provides configuration loading and validation for the BFF service.
package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config is the root configuration model for the BFF service.
type Config struct {
	Env    string       `koanf:"env"`
	Server ServerConfig `koanf:"server"`
	Log    LogConfig    `koanf:"log"`
	CORS   CORSConfig   `koanf:"cors"`
	Auth   AuthConfig   `koanf:"auth"`
	Proxy  ProxyConfig  `koanf:"proxy"`
}

// AuthConfig contains OIDC confidential-client, JWT validation, and split-cookie settings.
type AuthConfig struct {
	Enabled                       bool          `koanf:"enabled"`
	IssuerURL                     string        `koanf:"issuer_url"`
	ClientID                      string        `koanf:"client_id"`
	ClientSecret                  string        `koanf:"-"`
	PortalURL                     string        `koanf:"portal_url"`
	RedirectURI                   string        `koanf:"redirect_uri"`
	PostLogoutRedirectURI         string        `koanf:"post_logout_redirect_uri"`
	Scopes                        []string      `koanf:"scopes"`
	ResourceAudience              string        `koanf:"resource_audience"`
	AllowedSigningAlgorithms      []string      `koanf:"allowed_signing_algorithms"`
	HTTPTimeout                   time.Duration `koanf:"http_timeout"`
	RefreshTimeout                time.Duration `koanf:"refresh_timeout"`
	ScopeClaim                    string        `koanf:"scope_claim"`
	OrgIDClaim                    string        `koanf:"org_id_claim"`
	OrgIDOverride                 string        `koanf:"org_id_override"`
	RequireAccessTokenType        bool          `koanf:"require_access_token_type"`
	AccessTokenTypeClaim          string        `koanf:"access_token_type_claim"`
	AccessTokenTypeValue          string        `koanf:"access_token_type_value"`
	AccessTokenPart1Cookie        string        `koanf:"access_token_part1_cookie"`
	AccessTokenPart2Cookie        string        `koanf:"access_token_part2_cookie"`
	RefreshTokenPart1Cookie       string        `koanf:"refresh_token_part1_cookie"`
	RefreshTokenPart2Cookie       string        `koanf:"refresh_token_part2_cookie"`
	IDTokenPart1Cookie            string        `koanf:"id_token_part1_cookie"`
	IDTokenPart2Cookie            string        `koanf:"id_token_part2_cookie"`
	OAuthStateCookie              string        `koanf:"oauth_state_cookie"`
	PKCEVerifierCookie            string        `koanf:"pkce_verifier_cookie"`
	CookieSecure                  bool          `koanf:"cookie_secure"`
	CookieSameSite                string        `koanf:"cookie_same_site"`
	ClockSkew                     time.Duration `koanf:"clock_skew"`
	LoginTransactionMaxAgeSeconds int           `koanf:"login_transaction_max_age_seconds"`
	RefreshCookieMaxAgeSeconds    int           `koanf:"refresh_cookie_max_age_seconds"`
	MaxTokenPartBytes             int           `koanf:"max_token_part_bytes"`
	MaxReconstructedTokenBytes    int           `koanf:"max_reconstructed_token_bytes"`
}

// CORSConfig contains browser cross-origin policy settings for local/frontend integration.
type CORSConfig struct {
	AllowedOrigins   []string `koanf:"allowed_origins"`
	AllowedMethods   []string `koanf:"allowed_methods"`
	AllowedHeaders   []string `koanf:"allowed_headers"`
	AllowCredentials bool     `koanf:"allow_credentials"`
}

// ServerConfig contains HTTP server runtime settings.
type ServerConfig struct {
	Host            string        `koanf:"host"`
	Port            int           `koanf:"port"`
	ReadTimeout     time.Duration `koanf:"read_timeout"`
	WriteTimeout    time.Duration `koanf:"write_timeout"`
	IdleTimeout     time.Duration `koanf:"idle_timeout"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
}

// LogConfig contains logging configuration for the BFF.
type LogConfig struct {
	Level string `koanf:"level"`
}

// ProxyConfig contains upstream proxy behavior and placeholder identity settings.
type ProxyConfig struct {
	OpenFGCAPIURL     string        `koanf:"openfgc_api_url"`
	OpenFGCAPITimeout time.Duration `koanf:"openfgc_api_timeout"`
	MaxRequestBytes   int64         `koanf:"max_request_bytes"`
	MaxResponseBytes  int64         `koanf:"max_response_bytes"`

	PlaceholderModeEnabled bool     `koanf:"placeholder_mode_enabled"`
	PlaceholderUserID      string   `koanf:"placeholder_user_id"`
	PlaceholderOrgID       string   `koanf:"placeholder_org_id"`
	PlaceholderScopes      []string `koanf:"placeholder_scopes"`
	AllowedPassthrough     []string `koanf:"allowed_passthrough_methods"`
}

// Load initializes configuration from defaults, optional file, and environment variables.
func Load() (*Config, error) {
	k := koanf.New(".")

	if err := setDefaults(k); err != nil {
		return nil, fmt.Errorf("set defaults: %w", err)
	}

	configPath := os.Getenv("BFF_CONFIG_FILE")
	if configPath != "" {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load config file: %w", err)
		}
	}

	if err := k.Load(env.Provider("BFF_", ".", func(s string) string {
		s = strings.TrimPrefix(s, "BFF_")
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "__", ".")
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("load env config: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if rawMethods := os.Getenv("BFF_PROXY__ALLOWED_PASSTHROUGH_METHODS"); rawMethods != "" {
		methods, err := ParseMethods(rawMethods)
		if err != nil {
			return nil, fmt.Errorf("parse proxy.allowed_passthrough_methods: %w", err)
		}
		if len(methods) > 0 {
			cfg.Proxy.AllowedPassthrough = methods
		}
	}

	if rawOrigins := os.Getenv("BFF_CORS__ALLOWED_ORIGINS"); rawOrigins != "" {
		cfg.CORS.AllowedOrigins = ParseCSV(rawOrigins)
	}
	if rawMethods := os.Getenv("BFF_CORS__ALLOWED_METHODS"); rawMethods != "" {
		cfg.CORS.AllowedMethods = ParseCSV(rawMethods)
	}
	if rawHeaders := os.Getenv("BFF_CORS__ALLOWED_HEADERS"); rawHeaders != "" {
		cfg.CORS.AllowedHeaders = ParseCSV(rawHeaders)
	}
	if rawScopes := os.Getenv("BFF_AUTH__SCOPES"); rawScopes != "" {
		cfg.Auth.Scopes = strings.Fields(rawScopes)
	}
	if rawScopes := os.Getenv("BFF_PROXY__PLACEHOLDER_SCOPES"); rawScopes != "" {
		cfg.Proxy.PlaceholderScopes = strings.Fields(rawScopes)
	}
	if rawAlgorithms := os.Getenv("BFF_AUTH__ALLOWED_SIGNING_ALGORITHMS"); rawAlgorithms != "" {
		cfg.Auth.AllowedSigningAlgorithms = ParseCSV(rawAlgorithms)
	}
	// Client secrets are intentionally sourced only from the process environment.
	// They are never loaded from the optional YAML configuration file.
	cfg.Auth.ClientSecret = os.Getenv("BFF_AUTH__CLIENT_SECRET")

	return &cfg, validate(cfg)
}

func setDefaults(k *koanf.Koanf) error {
	if err := k.Set("server.host", "0.0.0.0"); err != nil {
		return err
	}
	if err := k.Set("server.port", 8080); err != nil {
		return err
	}
	if err := k.Set("server.read_timeout", "15s"); err != nil {
		return err
	}
	if err := k.Set("server.write_timeout", "15s"); err != nil {
		return err
	}
	if err := k.Set("server.idle_timeout", "60s"); err != nil {
		return err
	}
	if err := k.Set("server.shutdown_timeout", "10s"); err != nil {
		return err
	}
	if err := k.Set("env", "development"); err != nil {
		return err
	}
	if err := k.Set("log.level", "info"); err != nil {
		return err
	}
	if err := k.Set("cors.allowed_origins", []string{}); err != nil {
		return err
	}
	if err := k.Set("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}); err != nil {
		return err
	}
	if err := k.Set("cors.allowed_headers", []string{"Content-Type", "Authorization", "X-Correlation-ID"}); err != nil {
		return err
	}
	if err := k.Set("cors.allow_credentials", false); err != nil {
		return err
	}
	if err := k.Set("auth.enabled", false); err != nil {
		return err
	}
	if err := k.Set("auth.scopes", []string{"openid", "profile"}); err != nil {
		return err
	}
	if err := k.Set("auth.allowed_signing_algorithms", []string{"RS256"}); err != nil {
		return err
	}
	if err := k.Set("auth.http_timeout", "5s"); err != nil {
		return err
	}
	if err := k.Set("auth.refresh_timeout", "10s"); err != nil {
		return err
	}
	if err := k.Set("auth.clock_skew", "30s"); err != nil {
		return err
	}
	if err := k.Set("auth.scope_claim", "scope"); err != nil {
		return err
	}
	if err := k.Set("auth.org_id_claim", "org_id"); err != nil {
		return err
	}
	if err := k.Set("auth.org_id_override", ""); err != nil {
		return err
	}
	if err := k.Set("auth.require_access_token_type", false); err != nil {
		return err
	}
	if err := k.Set("auth.access_token_type_claim", "token_type"); err != nil {
		return err
	}
	if err := k.Set("auth.access_token_type_value", "access_token"); err != nil {
		return err
	}
	if err := k.Set("auth.access_token_part1_cookie", "portal-at-p1"); err != nil {
		return err
	}
	if err := k.Set("auth.access_token_part2_cookie", "portal-at-p2"); err != nil {
		return err
	}
	if err := k.Set("auth.refresh_token_part1_cookie", "portal-rt-p1"); err != nil {
		return err
	}
	if err := k.Set("auth.refresh_token_part2_cookie", "portal-rt-p2"); err != nil {
		return err
	}
	if err := k.Set("auth.id_token_part1_cookie", "portal-id-p1"); err != nil {
		return err
	}
	if err := k.Set("auth.id_token_part2_cookie", "portal-id-p2"); err != nil {
		return err
	}
	if err := k.Set("auth.oauth_state_cookie", "portal-oauth-state"); err != nil {
		return err
	}
	if err := k.Set("auth.pkce_verifier_cookie", "portal-pkce-verifier"); err != nil {
		return err
	}
	if err := k.Set("auth.cookie_secure", false); err != nil {
		return err
	}
	if err := k.Set("auth.cookie_same_site", "Lax"); err != nil {
		return err
	}
	if err := k.Set("auth.refresh_cookie_max_age_seconds", 86400); err != nil {
		return err
	}
	if err := k.Set("auth.login_transaction_max_age_seconds", 600); err != nil {
		return err
	}
	if err := k.Set("auth.max_token_part_bytes", 3800); err != nil {
		return err
	}
	if err := k.Set("auth.max_reconstructed_token_bytes", 7600); err != nil {
		return err
	}
	if err := k.Set("proxy.openfgc_api_url", "http://localhost:9090"); err != nil {
		return err
	}
	if err := k.Set("proxy.openfgc_api_timeout", "10s"); err != nil {
		return err
	}
	if err := k.Set("proxy.max_request_bytes", int64(1048576)); err != nil {
		return err
	}
	if err := k.Set("proxy.max_response_bytes", int64(10485760)); err != nil {
		return err
	}
	if err := k.Set("proxy.placeholder_mode_enabled", false); err != nil {
		return err
	}
	if err := k.Set("proxy.placeholder_user_id", ""); err != nil {
		return err
	}
	if err := k.Set("proxy.placeholder_org_id", ""); err != nil {
		return err
	}
	if err := k.Set("proxy.placeholder_scopes", []string{}); err != nil {
		return err
	}
	if err := k.Set("proxy.allowed_passthrough_methods", []string{"GET", "POST", "PUT", "DELETE"}); err != nil {
		return err
	}

	return nil
}

func validate(cfg Config) error {
	if cfg.Server.Port <= 0 {
		return fmt.Errorf("server.port must be a positive value")
	}
	if cfg.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("server.shutdown_timeout must be > 0")
	}
	if _, err := ValidateOpenFGCAPIURL(cfg.Proxy.OpenFGCAPIURL); err != nil {
		return err
	}
	if cfg.Proxy.OpenFGCAPITimeout <= 0 {
		return fmt.Errorf("proxy.openfgc_api_timeout must be > 0")
	}
	if cfg.Proxy.MaxRequestBytes <= 0 {
		return fmt.Errorf("proxy.max_request_bytes must be > 0")
	}
	if cfg.Proxy.MaxResponseBytes <= 0 {
		return fmt.Errorf("proxy.max_response_bytes must be > 0")
	}
	for _, raw := range cfg.CORS.AllowedOrigins {
		origin := strings.TrimSpace(raw)
		if origin == "" {
			continue
		}
		if cfg.CORS.AllowCredentials && origin == "*" {
			return fmt.Errorf("cors.allowed_origins cannot contain wildcard when cors.allow_credentials is true")
		}
		u, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("cors.allowed_origins contains invalid URL %q: %w", origin, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("cors.allowed_origins contains unsupported scheme for %q", origin)
		}
		if u.Host == "" {
			return fmt.Errorf("cors.allowed_origins contains missing host for %q", origin)
		}
		if u.Path != "" && u.Path != "/" {
			return fmt.Errorf("cors.allowed_origins must not contain a path for %q", origin)
		}
		if u.RawQuery != "" {
			return fmt.Errorf("cors.allowed_origins must not contain a query string for %q", origin)
		}
		if u.Fragment != "" {
			return fmt.Errorf("cors.allowed_origins must not contain a fragment for %q", origin)
		}
	}
	if len(cfg.CORS.AllowedMethods) == 0 {
		return fmt.Errorf("cors.allowed_methods must not be empty")
	}
	if len(cfg.CORS.AllowedHeaders) == 0 {
		return fmt.Errorf("cors.allowed_headers must not be empty")
	}
	if cfg.CORS.AllowCredentials {
		if len(cfg.CORS.AllowedOrigins) == 0 {
			return fmt.Errorf("cors.allowed_origins must not be empty when cors.allow_credentials is true")
		}
	}
	if cfg.Proxy.PlaceholderModeEnabled && strings.EqualFold(cfg.Env, "production") {
		return fmt.Errorf("proxy.placeholder_mode_enabled cannot be true in production")
	}
	if err := validateAuth(cfg); err != nil {
		return err
	}
	if !cfg.Proxy.PlaceholderModeEnabled && cfg.Proxy.PlaceholderUserID != "" {
		return fmt.Errorf("proxy.placeholder_user_id must be empty when placeholder mode is disabled")
	}
	if !cfg.Proxy.PlaceholderModeEnabled && cfg.Proxy.PlaceholderOrgID != "" {
		return fmt.Errorf("proxy.placeholder_org_id must be empty when placeholder mode is disabled")
	}
	if !cfg.Proxy.PlaceholderModeEnabled && len(cfg.Proxy.PlaceholderScopes) != 0 {
		return fmt.Errorf("proxy.placeholder_scopes must be empty when placeholder mode is disabled")
	}
	if len(cfg.Proxy.AllowedPassthrough) == 0 {
		return fmt.Errorf("proxy.allowed_passthrough_methods must not be empty")
	}
	return nil
}

func validateAuth(cfg Config) error {
	if !cfg.Auth.Enabled {
		if strings.EqualFold(cfg.Env, "production") {
			return fmt.Errorf("auth.enabled must be true in production")
		}
		return nil
	}
	required := map[string]string{
		"auth.issuer_url": cfg.Auth.IssuerURL,
		"auth.client_id":  cfg.Auth.ClientID,
		"auth.client_secret (BFF_AUTH__CLIENT_SECRET)": cfg.Auth.ClientSecret,
		"auth.portal_url":               cfg.Auth.PortalURL,
		"auth.redirect_uri":             cfg.Auth.RedirectURI,
		"auth.post_logout_redirect_uri": cfg.Auth.PostLogoutRedirectURI,
		"auth.resource_audience":        cfg.Auth.ResourceAudience,
		"auth.scope_claim":              cfg.Auth.ScopeClaim,
	}
	if strings.TrimSpace(cfg.Auth.OrgIDOverride) == "" {
		required["auth.org_id_claim"] = cfg.Auth.OrgIDClaim
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty when auth is enabled", name)
		}
	}
	for name, value := range map[string]string{
		"auth.issuer_url":               cfg.Auth.IssuerURL,
		"auth.portal_url":               cfg.Auth.PortalURL,
		"auth.redirect_uri":             cfg.Auth.RedirectURI,
		"auth.post_logout_redirect_uri": cfg.Auth.PostLogoutRedirectURI,
	} {
		if err := validateAbsoluteHTTPURL(value, strings.EqualFold(cfg.Env, "production")); err != nil {
			return fmt.Errorf("%s %w", name, err)
		}
	}
	if len(cfg.Auth.Scopes) == 0 || len(cfg.Auth.AllowedSigningAlgorithms) == 0 {
		return fmt.Errorf("auth scopes and allowed signing algorithms must not be empty")
	}
	if cfg.Auth.HTTPTimeout <= 0 || cfg.Auth.RefreshTimeout <= 0 {
		return fmt.Errorf("auth HTTP timeouts must be > 0")
	}
	if cfg.Auth.ClockSkew < 0 {
		return fmt.Errorf("auth.clock_skew must be >= 0")
	}
	if cfg.Auth.MaxTokenPartBytes <= 0 || cfg.Auth.MaxReconstructedTokenBytes < cfg.Auth.MaxTokenPartBytes*2 {
		return fmt.Errorf("auth token size limits are invalid")
	}
	if cfg.Auth.RefreshCookieMaxAgeSeconds <= 0 {
		return fmt.Errorf("auth.refresh_cookie_max_age_seconds must be > 0")
	}
	if cfg.Auth.LoginTransactionMaxAgeSeconds <= 0 || cfg.Auth.LoginTransactionMaxAgeSeconds > 600 {
		return fmt.Errorf("auth.login_transaction_max_age_seconds must be between 1 and 600")
	}
	switch strings.ToLower(cfg.Auth.CookieSameSite) {
	case "strict", "lax", "none":
	default:
		return fmt.Errorf("auth.cookie_same_site must be Strict, Lax, or None")
	}
	if strings.EqualFold(cfg.Auth.CookieSameSite, "none") && !cfg.Auth.CookieSecure {
		return fmt.Errorf("auth.cookie_secure must be true when SameSite=None")
	}
	cookieNames := []string{
		cfg.Auth.AccessTokenPart1Cookie, cfg.Auth.AccessTokenPart2Cookie,
		cfg.Auth.RefreshTokenPart1Cookie, cfg.Auth.RefreshTokenPart2Cookie,
		cfg.Auth.IDTokenPart1Cookie, cfg.Auth.IDTokenPart2Cookie,
		cfg.Auth.OAuthStateCookie, cfg.Auth.PKCEVerifierCookie,
	}
	seenCookieNames := make(map[string]struct{}, len(cookieNames))
	for _, name := range cookieNames {
		if strings.TrimSpace(name) == "" || (&http.Cookie{Name: name, Value: "value"}).Valid() != nil {
			return fmt.Errorf("auth cookie names must be valid and non-empty")
		}
		if _, duplicate := seenCookieNames[name]; duplicate {
			return fmt.Errorf("auth cookie names must be unique")
		}
		seenCookieNames[name] = struct{}{}
	}
	if strings.EqualFold(cfg.Env, "production") && !cfg.Auth.CookieSecure {
		return fmt.Errorf("auth.cookie_secure must be true in production")
	}
	if cfg.Proxy.PlaceholderModeEnabled {
		return fmt.Errorf("proxy placeholder mode and auth cannot both be enabled")
	}
	return nil
}

func validateAbsoluteHTTPURL(raw string, requireHTTPS bool) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("must be an absolute HTTP(S) URL")
	}
	if requireHTTPS && parsed.Scheme != "https" {
		return fmt.Errorf("must use HTTPS in production")
	}
	return nil
}

// ValidateOpenFGCAPIURL validates and parses the configured upstream OpenFGC API URL.
func ValidateOpenFGCAPIURL(rawURL string) (*url.URL, error) {
	upstream := strings.TrimSpace(rawURL)
	if upstream == "" {
		return nil, fmt.Errorf("proxy.openfgc_api_url must not be empty")
	}

	parsed, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("proxy.openfgc_api_url must be a valid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("proxy.openfgc_api_url must use http or https scheme")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("proxy.openfgc_api_url must include a host")
	}

	return parsed, nil
}

// ParseMethods parses a JSON array of HTTP methods from BFF_PROXY__ALLOWED_PASSTHROUGH_METHODS.
func ParseMethods(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var methods []string
	if err := json.Unmarshal([]byte(raw), &methods); err != nil {
		return nil, err
	}
	for i := range methods {
		methods[i] = strings.ToUpper(strings.TrimSpace(methods[i]))
	}
	return methods, nil
}

// ParseCSV parses comma-separated values and removes empty entries.
func ParseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}
