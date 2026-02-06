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

// Package managers provides functionality for managing and registering system services.
package main

import (
	"net/http"

	"github.com/wso2/openfgc/internal/authresource"
	"github.com/wso2/openfgc/internal/consent"
	"github.com/wso2/openfgc/internal/consentelement"
	"github.com/wso2/openfgc/internal/consentpurpose"
	"github.com/wso2/openfgc/internal/system/healthcheck/handler"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/stores"
)

// registerServices registers all consent management services with the provided HTTP multiplexer.
func registerServices(
	mux *http.ServeMux,
) {
	logger := log.GetLogger()

	// Create Store Registry with all stores
	storeRegistry := stores.NewStoreRegistry(
		consent.NewConsentStore(),
		authresource.NewAuthResourceStore(),
		consentelement.NewConsentElementStore(),
		consentpurpose.NewPurposeStore(),
	)
	logger.Info("Store Registry initialized with all stores")

	// Initialize all services with the registry
	authresource.Initialize(mux, storeRegistry)
	logger.Info("AuthResource module initialized")

	consentelement.Initialize(mux, storeRegistry)
	logger.Info("ConsentElement module initialized")

	consentpurpose.Initialize(mux, storeRegistry)
	logger.Info("ConsentPurpose module initialized")

	consent.Initialize(mux, storeRegistry)
	logger.Info("Consent module initialized")

	// Register health check endpoints
	registerHealthCheckEndpoints(mux)
	logger.Info("Health check endpoints registered")
}

// registerHealthCheckEndpoints registers the health check endpoints.
func registerHealthCheckEndpoints(mux *http.ServeMux) {
	healthCheckHandler := handler.NewHealthCheckHandler()

	// Liveness endpoint - simple check if server is running
	mux.HandleFunc("GET /health/liveness", healthCheckHandler.HandleLivenessRequest)

	// Readiness endpoint - checks if server and dependencies are ready
	mux.HandleFunc("GET /health/readiness", healthCheckHandler.HandleReadinessRequest)

	// Legacy health endpoint (for backward compatibility)
	mux.HandleFunc("GET /health", healthCheckHandler.HandleLivenessRequest)
}

// unregisterServices performs cleanup of all services during shutdown.
// Currently a placeholder for future service cleanup needs.
func unregisterServices() {
	// Future: Add any service-specific cleanup logic here
	// e.g., closing connections, flushing caches, etc.
}
