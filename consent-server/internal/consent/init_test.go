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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/system/stores"
)

func TestInitialize(t *testing.T) {
	// Test that Initialize creates service and registers routes
	mux := http.NewServeMux()
	registry := &stores.StoreRegistry{}

	service := Initialize(mux, registry)

	require.NotNil(t, service, "Service should be initialized")
}

func TestRegisterRoutes(t *testing.T) {
	// Test that routes are properly registered
	mux := http.NewServeMux()
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	registerRoutes(mux, handler)

	// Test that routes respond (even if with errors due to mock)
	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/consents"},
		{"GET", "/api/v1/consents"},
		{"POST", "/api/v1/consents/validate"},
		{"GET", "/api/v1/consents/attributes"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			// Just verify the route exists (will return error due to missing headers but that's OK)
			require.NotEqual(t, http.StatusNotFound, rr.Code, "Route should be registered")
		})
	}
}

func TestNewConsentService(t *testing.T) {
	// Test service creation
	registry := &stores.StoreRegistry{}
	service := newConsentService(registry)

	require.NotNil(t, service, "Service should not be nil")
}

func TestNewConsentHandler(t *testing.T) {
	// Test handler creation
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	require.NotNil(t, handler, "Handler should not be nil")
	require.NotNil(t, handler.service, "Handler service should not be nil")
}
