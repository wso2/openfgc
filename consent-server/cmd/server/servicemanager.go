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
	"context"
	"net/http"

	"github.com/wso2/openfgc/consent-server/internal/authresource"
	"github.com/wso2/openfgc/consent-server/internal/consent"
	"github.com/wso2/openfgc/consent-server/internal/consentelement"
	"github.com/wso2/openfgc/consent-server/internal/consentpurpose"
	"github.com/wso2/openfgc/consent-server/internal/system/config"
	"github.com/wso2/openfgc/consent-server/internal/system/healthcheck/handler"
	"github.com/wso2/openfgc/consent-server/internal/system/log"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
)

// cancelScheduler holds the cancel function for the consent expiration scheduler goroutine.
// It is set by startConsentExpirationScheduler and called by unregisterServices on shutdown.
var cancelScheduler context.CancelFunc

// registerServices registers all consent management services with the provided HTTP multiplexer.
func registerServices(mux *http.ServeMux) {
	logger := log.GetLogger()

	// Create Store Registry with all stores
	storeRegistry := stores.NewStoreRegistry(
		consent.NewConsentStore(),
		authresource.NewAuthResourceStore(),
		consentelement.NewConsentElementStore(),
		consentpurpose.NewPurposeStore(),
	)
	logger.Debug("Store Registry initialized with all stores")

	// Initialize all services with the registry
	authresource.Initialize(mux, storeRegistry)
	logger.Debug("AuthResource module initialized")

	consentelement.Initialize(mux, storeRegistry)
	logger.Debug("ConsentElement module initialized")

	consentpurpose.Initialize(mux, storeRegistry)
	logger.Debug("ConsentPurpose module initialized")

	svc := consent.Initialize(mux, storeRegistry)
	logger.Debug("Consent module initialized")

	startConsentExpirationScheduler(svc)

	registerHealthCheckEndpoints(mux)
	logger.Debug("Health check endpoints registered")
}

// startConsentExpirationScheduler starts the background scheduler for expiring eligible consents.
// If periodical expiration is disabled in config, the scheduler is not started.
func startConsentExpirationScheduler(svc consent.ConsentService) {
	logger := log.GetLogger()

	cfg := config.Get()

	if !cfg.Consent.PeriodicalExpiration.Enabled {
		logger.Info("Consent periodical expiration is disabled — skipping scheduler startup")
		return
	}

	interval := cfg.Consent.GetExpirationFrequency()
	statuses := consent.ExpirationStatuses{
		ExpirableConsentStatuses: cfg.Consent.GetEligibleConsentStatuses(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelScheduler = cancel
	go consent.StartScheduler(ctx, svc, interval, statuses)
	logger.Info("Consent expiration scheduler started", log.String("interval", interval.String()))
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
func unregisterServices() {
	if cancelScheduler != nil {
		cancelScheduler()
	}
}
