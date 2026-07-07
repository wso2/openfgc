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

	"github.com/wso2/openfgc/consent-server/internal/system/constants"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
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

// registerRoutes registers all consent purpose routes.
func registerRoutes(mux *http.ServeMux, handler *consentPurposeHandler) {
	base := constants.APIBasePath + "/consent-purposes"

	mux.HandleFunc("POST "+base, handler.createPurpose)
	mux.HandleFunc("GET "+base, handler.listPurposes)
	mux.HandleFunc("GET "+base+"/{purposeId}", handler.getPurpose)
	mux.HandleFunc("DELETE "+base+"/{purposeId}", handler.deletePurpose)

	mux.HandleFunc("GET "+base+"/{purposeId}/versions", handler.listPurposeVersions)
	mux.HandleFunc("POST "+base+"/{purposeId}/versions", handler.createPurposeVersion)
	mux.HandleFunc("GET "+base+"/{purposeId}/versions/{version}", handler.getPurposeVersion)
	mux.HandleFunc("DELETE "+base+"/{purposeId}/versions/{version}", handler.deletePurposeVersion)
}
