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

	"github.com/wso2/openfgc/consent-server/internal/system/constants"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
)

// Initialize sets up the consent element module and registers routes.
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentElementService {
	service := NewConsentElementService(registry)
	handler := newConsentElementHandler(service)
	registerRoutes(mux, handler)
	return service
}

func registerRoutes(mux *http.ServeMux, handler *consentElementHandler) {
	base := constants.APIBasePath + "/consent-elements"

	mux.HandleFunc("POST "+base, handler.createElements)
	mux.HandleFunc("GET "+base, handler.listElements)
	mux.HandleFunc("GET "+base+"/{elementId}", handler.getElement)
	mux.HandleFunc("GET "+base+"/{elementId}/versions", handler.listElementVersions)
	mux.HandleFunc("POST "+base+"/{elementId}/versions", handler.createElementVersion)
	mux.HandleFunc("GET "+base+"/{elementId}/versions/{version}", handler.getElementVersion)
	mux.HandleFunc("DELETE "+base+"/{elementId}/versions/{version}", handler.deleteElementVersion)
}
