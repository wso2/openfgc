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
	Proxy  ProxyConfig  `koanf:"proxy"`
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

	PlaceholderModeEnabled bool   `koanf:"placeholder_mode_enabled"`
	PlaceholderUserID      string `koanf:"placeholder_user_id"`
	PlaceholderOrgID       string `koanf:"placeholder_org_id"`
	PlaceholderClientID    string `koanf:"placeholder_client_id"`

	AllowedPassthrough []string `koanf:"allowed_passthrough_methods"`
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
	if err := k.Set("cors.allowed_headers", []string{"Content-Type", "X-Correlation-ID"}); err != nil {
		return err
	}
	if err := k.Set("cors.allow_credentials", false); err != nil {
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
	if err := k.Set("proxy.placeholder_mode_enabled", false); err != nil {
		return err
	}
	if err := k.Set("proxy.placeholder_user_id", ""); err != nil {
		return err
	}
	if err := k.Set("proxy.placeholder_org_id", ""); err != nil {
		return err
	}
	if err := k.Set("proxy.placeholder_client_id", ""); err != nil {
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
	for _, raw := range cfg.CORS.AllowedOrigins {
		origin := strings.TrimSpace(raw)
		if origin == "" {
			continue
		}
		if cfg.CORS.AllowCredentials && origin == "*" {
			return fmt.Errorf("cors.allowed_origins cannot contain wildcard when cors.allow_credentials is true")
		}
		u, err := url.ParseRequestURI(origin)
		if err != nil {
			return fmt.Errorf("cors.allowed_origins contains invalid URL %q: %w", origin, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("cors.allowed_origins contains unsupported scheme for %q", origin)
		}
		if u.Host == "" {
			return fmt.Errorf("cors.allowed_origins contains missing host for %q", origin)
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
	if !cfg.Proxy.PlaceholderModeEnabled && cfg.Proxy.PlaceholderUserID != "" {
		return fmt.Errorf("proxy.placeholder_user_id must be empty when placeholder mode is disabled")
	}
	if !cfg.Proxy.PlaceholderModeEnabled && cfg.Proxy.PlaceholderOrgID != "" {
		return fmt.Errorf("proxy.placeholder_org_id must be empty when placeholder mode is disabled")
	}
	if !cfg.Proxy.PlaceholderModeEnabled && cfg.Proxy.PlaceholderClientID != "" {
		return fmt.Errorf("proxy.placeholder_client_id must be empty when placeholder mode is disabled")
	}
	if len(cfg.Proxy.AllowedPassthrough) == 0 {
		return fmt.Errorf("proxy.allowed_passthrough_methods must not be empty")
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
