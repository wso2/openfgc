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

package unit

import (
	"strings"
	"testing"

	"github.com/wso2/openfgc/portal/backend/internal/config"
)

func TestPlaceholderModeBlockedInProduction(t *testing.T) {
	t.Setenv("BFF_ENV", "production")
	t.Setenv("BFF_PROXY__PLACEHOLDER_MODE_ENABLED", "true")
	t.Setenv("BFF_PROXY__PLACEHOLDER_USER_ID", "user@example.com")

	_, err := config.Load()
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
		{name: "client id", envName: "BFF_PROXY__PLACEHOLDER_CLIENT_ID", errText: "proxy.placeholder_client_id must be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BFF_PROXY__PLACEHOLDER_MODE_ENABLED", "false")
			t.Setenv(tt.envName, "placeholder-value")

			_, err := config.Load()
			if err == nil {
				t.Fatal("expected error when placeholder value is set while mode is disabled")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("expected error to contain %q, got %v", tt.errText, err)
			}
		})
	}
}

func TestAllowedPassthroughMethodsEnvJSON(t *testing.T) {
	t.Setenv("BFF_PROXY__ALLOWED_PASSTHROUGH_METHODS", `["get", "put"]`)

	cfg, err := config.Load()
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

			_, err := config.Load()
			if err == nil {
				t.Fatal("expected config load error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("expected error to contain %q, got %v", tt.errText, err)
			}
		})
	}
}
