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

package consent

import (
	"net/http"

	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/stores"
)

// Initialize sets up the consent module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentService {
	// Create service and handler using the registry
	service := newConsentService(registry)
	handler := newConsentHandler(service)

	// Register routes
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all consent routes
func registerRoutes(mux *http.ServeMux, handler *consentHandler) {
	// POST /api/v1/consents - Create consent
	mux.HandleFunc("POST "+constants.APIBasePath+"/consents", handler.createConsent)

	// GET /api/v1/consents/{consentId} - Get consent by ID
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}", handler.getConsent)

	// GET /api/v1/consents - List/search consents
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents", handler.listConsents)

	// PUT /api/v1/consents/{consentId} - Update consent
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consents/{consentId}", handler.updateConsent)

	// PUT /api/v1/consents/{consentId}/revoke - Revoke consent
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consents/{consentId}/revoke", handler.revokeConsent)

	// POST /api/v1/consents/validate - Validate consent
	mux.HandleFunc("POST "+constants.APIBasePath+"/consents/validate", handler.validateConsent)

	// GET /api/v1/consents/attributes - Search consents by attribute
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/attributes", handler.searchConsentsByAttribute)
}
