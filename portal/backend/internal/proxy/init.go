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

package proxy

import (
	"net/http"
	"strings"

	"github.com/wso2/openfgc/portal/backend/internal/system/auth"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
	"github.com/wso2/openfgc/portal/backend/internal/system/middleware"
)

// Initialize sets up the proxy module and registers routes.
func Initialize(mux *http.ServeMux, cfg config.Config, validator *auth.Validator) error {
	handler, err := NewHandler(cfg.Proxy)
	if err != nil {
		return err
	}

	identityOptions := middleware.IdentityOptions{
		PlaceholderModeEnabled: cfg.Proxy.PlaceholderModeEnabled,
		PlaceholderUserID:      cfg.Proxy.PlaceholderUserID,
		PlaceholderOrgID:       cfg.Proxy.PlaceholderOrgID,
	}
	protected := middleware.Authenticate(middleware.RequireScope(http.HandlerFunc(handler.API), apiScope), validator, identityOptions)
	mux.Handle("/api/{path...}", protected)
	return nil
}

func apiScope(r *http.Request) string {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	method := r.Method
	if strings.HasPrefix(path, "consents") {
		if method == http.MethodGet {
			return "portal:consents:read:any"
		}
		return "portal:consents:write:any"
	}
	if strings.HasPrefix(path, "consent-elements") {
		if method == http.MethodGet {
			return "portal:elements:read"
		}
		return "portal:elements:write"
	}
	if strings.HasPrefix(path, "consent-purposes") {
		if method == http.MethodGet {
			return "portal:purposes:read"
		}
		return "portal:purposes:write"
	}
	return ""
}
