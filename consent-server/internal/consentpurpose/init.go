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

package consentpurpose

import (
	"net/http"

	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/stores"
)

// Initialize sets up the consent purpose module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentPurposeService {
	// Create service and handler using the registry
	service := NewConsentPurposeService(registry)
	handler := newConsentPurposeHandler(service)

	// Register routes
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all consent purpose routes
func registerRoutes(mux *http.ServeMux, handler *consentPurposeHandler) {
	// POST /api/v1/consent-purposes - Create consent purpose
	mux.HandleFunc("POST "+constants.APIBasePath+"/consent-purposes", handler.createPurpose)

	// GET /api/v1/consent-purposes/{purposeId} - Get consent purpose by ID
	mux.HandleFunc("GET "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.getPurpose)

	// GET /api/v1/consent-purposes - List consent purposes
	mux.HandleFunc("GET "+constants.APIBasePath+"/consent-purposes", handler.listPurposes)

	// PUT /api/v1/consent-purposes/{purposeId} - Update consent purpose
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.updatePurpose)

	// DELETE /api/v1/consent-purposes/{purposeId} - Delete consent purpose
	mux.HandleFunc("DELETE "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.deletePurpose)
}
