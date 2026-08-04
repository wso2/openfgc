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
)

// Initialize sets up the me module and registers routes.
func Initialize(mux *http.ServeMux, cfg config.Config, authManager *auth.Manager) error {
	handler, err := NewHandler(cfg.Proxy)
	if err != nil {
		return err
	}

	mux.Handle("GET /me", authManager.Require(http.HandlerFunc(handler.CurrentUser)))
	mux.Handle("GET /me/consents", authManager.Require(http.HandlerFunc(handler.Consents), auth.ScopeConsentsReadSelf))
	mux.Handle("GET /me/consents/{consentId}", authManager.Require(http.HandlerFunc(handler.ConsentByID), auth.ScopeConsentsReadSelf))
	mux.Handle("POST /me/consents/{consentId}/approve", authManager.Require(http.HandlerFunc(handler.ConsentApprove), auth.ScopeConsentsWriteSelf))
	mux.Handle("POST /me/consents/{consentId}/reject", authManager.Require(http.HandlerFunc(handler.ConsentReject), auth.ScopeConsentsWriteSelf))
	mux.Handle("POST /me/consents/{consentId}/revoke", authManager.Require(http.HandlerFunc(handler.ConsentRevoke), auth.ScopeConsentsWriteSelf))

	return nil
}
