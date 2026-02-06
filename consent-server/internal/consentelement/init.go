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

// Package consentelement provides consent element management functionality.
package consentelement

import (
	"net/http"

	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/stores"
)

// Initialize sets up the consent element module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentElementService {
	// Create service and handler using the registry
	service := newConsentElementService(registry)
	handler := newConsentElementHandler(service)

	// Register routes
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all consent element routes
func registerRoutes(mux *http.ServeMux, handler *consentElementHandler) {
	// POST /api/v1/consent-elements - Create element
	mux.HandleFunc("POST "+constants.APIBasePath+"/consent-elements", handler.createElement)

	// GET /api/v1/consent-elements/{elementId} - Get element by ID
	mux.HandleFunc("GET "+constants.APIBasePath+"/consent-elements/{elementId}", handler.getElement)

	// GET /api/v1/consent-elements - List elements
	mux.HandleFunc("GET "+constants.APIBasePath+"/consent-elements", handler.listElements)

	// POST /api/v1/consent-elements/validate - Validate element names
	mux.HandleFunc("POST "+constants.APIBasePath+"/consent-elements/validate", handler.validateElements)

	// PUT /api/v1/consent-elements/{elementId} - Update element
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consent-elements/{elementId}", handler.updateElement)

	// DELETE /api/v1/consent-elements/{elementId} - Delete element
	mux.HandleFunc("DELETE "+constants.APIBasePath+"/consent-elements/{elementId}", handler.deleteElement)
}
