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
		{"POST", constants.APIBasePath + "/consent-purposes"},
		{"GET", constants.APIBasePath + "/consent-purposes"},
		{"GET", constants.APIBasePath + "/consent-purposes/test-id"},
		{"PUT", constants.APIBasePath + "/consent-purposes/test-id"},
		{"DELETE", constants.APIBasePath + "/consent-purposes/test-id"},
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
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	registerRoutes(mux, handler)

	// Test that routes are registered
	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/consent-purposes"},
		{"GET", "/api/v1/consent-purposes"},
		{"GET", "/api/v1/consent-purposes/purpose-123"},
		{"PUT", "/api/v1/consent-purposes/purpose-123"},
		{"DELETE", "/api/v1/consent-purposes/purpose-123"},
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

// TestNewConsentPurposeService tests service creation
func TestNewConsentPurposeService(t *testing.T) {
	registry := &stores.StoreRegistry{}

	service := NewConsentPurposeService(registry)

	require.NotNil(t, service)

	// Verify it implements the interface
	var _ ConsentPurposeService = service
}

// TestNewConsentPurposeHandler tests handler creation
func TestNewConsentPurposeHandler(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	handler := newConsentPurposeHandler(mockService)

	require.NotNil(t, handler)
	require.Equal(t, mockService, handler.service)
}
