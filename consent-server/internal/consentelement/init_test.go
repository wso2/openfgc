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

package consentelement

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/stores"
)

// TestInitialize tests the Initialize function
func TestInitialize(t *testing.T) {
	mux := http.NewServeMux()
	registry := &stores.StoreRegistry{}

	service := Initialize(mux, registry)

	require.NotNil(t, service)

	// Verify routes are registered by making test requests
	testCases := []struct {
		method string
		path   string
	}{
		{"POST", constants.APIBasePath + "/consent-elements"},
		{"GET", constants.APIBasePath + "/consent-elements"},
		{"GET", constants.APIBasePath + "/consent-elements/test-id"},
		{"PUT", constants.APIBasePath + "/consent-elements/test-id"},
		{"DELETE", constants.APIBasePath + "/consent-elements/test-id"},
		{"POST", constants.APIBasePath + "/consent-elements/validate"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			// Should not return 404 - route should be registered
			// It will return 400 (missing org-id) but that means route exists
			require.NotEqual(t, http.StatusNotFound, rr.Code,
				"Route should be registered: %s %s", tc.method, tc.path)
		})
	}
}

// TestRegisterRoutes tests route registration
func TestRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	mockService := NewMockConsentElementService(t)
	handler := newConsentElementHandler(mockService)

	registerRoutes(mux, handler)

	// Test that routes are registered
	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/consent-elements"},
		{"GET", "/api/v1/consent-elements"},
		{"GET", "/api/v1/consent-elements/elem-123"},
		{"PUT", "/api/v1/consent-elements/elem-123"},
		{"DELETE", "/api/v1/consent-elements/elem-123"},
		{"POST", "/api/v1/consent-elements/validate"},
	}

	for _, route := range routes {
		t.Run(route.method+"_"+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader(""))
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			// Route exists if we don't get 404
			require.NotEqual(t, http.StatusNotFound, rr.Code,
				"Route not registered: %s %s", route.method, route.path)
		})
	}
}

// TestNewConsentElementService tests service creation
func TestNewConsentElementService(t *testing.T) {
	registry := &stores.StoreRegistry{}

	service := newConsentElementService(registry)

	require.NotNil(t, service)

	// Verify it implements the interface
	var _ ConsentElementService = service
}

// TestNewConsentElementHandler tests handler creation
func TestNewConsentElementHandler(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	handler := newConsentElementHandler(mockService)

	require.NotNil(t, handler)
	require.Equal(t, mockService, handler.service)
}
