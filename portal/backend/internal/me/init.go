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

package me

import (
	"net/http"

	"github.com/wso2/openfgc/portal/backend/internal/system/auth"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
	"github.com/wso2/openfgc/portal/backend/internal/system/middleware"
)

// Initialize sets up the me module and registers routes.
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

	protect := func(next http.Handler, scope string) http.Handler {
		return middleware.Authenticate(middleware.RequireScope(next, func(*http.Request) string { return scope }), validator, identityOptions)
	}
	mux.Handle("GET /me/consents", protect(http.HandlerFunc(handler.Consents), "portal:consents:read:self"))
	mux.Handle("GET /me/consents/{consentId}", protect(http.HandlerFunc(handler.ConsentByID), "portal:consents:read:self"))
	mux.Handle("POST /me/consents/{consentId}/approve", protect(http.HandlerFunc(handler.ConsentApprove), "portal:consents:write:self"))
	mux.Handle("PUT /me/consents/{consentId}/revoke", protect(http.HandlerFunc(handler.ConsentRevoke), "portal:consents:write:self"))

	return nil
}
