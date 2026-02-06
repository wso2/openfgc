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

package authresource

import (
	"net/http"

	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/stores"
)

// Initialize sets up the auth resource module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) AuthResourceServiceInterface {
	// Create service and handler using the registry
	service := newAuthResourceService(registry)
	handler := newAuthResourceHandler(service)

	// Register routes
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all auth resource HTTP routes
func registerRoutes(mux *http.ServeMux, handler *authResourceHandler) {
	// Create authorization (POST /api/v1/consents/{consentId}/authorizations)
	mux.HandleFunc(
		"POST "+constants.APIBasePath+"/consents/{consentId}/authorizations",
		handler.handleCreate,
	)

	// List authorizations by consent (GET /api/v1/consents/{consentId}/authorizations)
	mux.HandleFunc(
		"GET "+constants.APIBasePath+"/consents/{consentId}/authorizations",
		handler.handleListByConsent,
	)

	// Get single authorization (GET /api/v1/consents/{consentId}/authorizations/{authorizationId})
	mux.HandleFunc(
		"GET "+constants.APIBasePath+"/consents/{consentId}/authorizations/{authorizationId}",
		handler.handleGet,
	)

	// Update authorization (PUT /api/v1/consents/{consentId}/authorizations/{authorizationId})
	mux.HandleFunc(
		"PUT "+constants.APIBasePath+"/consents/{consentId}/authorizations/{authorizationId}",
		handler.handleUpdate,
	)
}
