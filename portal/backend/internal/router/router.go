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

// Package router wires HTTP routes and middleware composition for the BFF.
package router

import (
	"log/slog"
	"net/http"

	"github.com/wso2/openfgc/portal/backend/internal/config"
	"github.com/wso2/openfgc/portal/backend/internal/health"
	"github.com/wso2/openfgc/portal/backend/internal/middleware"
	"github.com/wso2/openfgc/portal/backend/internal/proxy"
)

// New builds the root HTTP handler with health, proxy, and /me routes.
func New(log *slog.Logger, cfg config.Config) (http.Handler, error) {
	mux := http.NewServeMux()

	healthHandler := health.NewHandler()
	mux.HandleFunc("GET /health/liveness", healthHandler.Liveness)
	mux.HandleFunc("GET /health/readiness", healthHandler.Readiness)
	mux.HandleFunc("GET /health", healthHandler.Liveness)

	proxyHandler, err := proxy.NewHandler(cfg.Proxy)
	if err != nil {
		return nil, err
	}
	userIDOptions := middleware.UserIDOptions{
		PlaceholderModeEnabled: cfg.Proxy.PlaceholderModeEnabled,
		PlaceholderUserID:      cfg.Proxy.PlaceholderUserID,
		Environment:            cfg.Env,
	}

	mux.Handle("GET /me/consents", middleware.UserID(http.HandlerFunc(proxyHandler.MeConsents), userIDOptions))
	mux.Handle("GET /me/consents/{consentId}", middleware.UserID(http.HandlerFunc(proxyHandler.MeConsentByID), userIDOptions))
	mux.Handle("POST /me/consents/{consentId}/approve", middleware.UserID(http.HandlerFunc(proxyHandler.MeConsentApprove), userIDOptions))
	mux.Handle("PUT /me/consents/{consentId}/revoke", middleware.UserID(http.HandlerFunc(proxyHandler.MeConsentRevoke), userIDOptions))
	mux.HandleFunc("/api/{path...}", proxyHandler.API)

	withCORS := middleware.CORS(mux, middleware.CORSOptions{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
	})

	return middleware.CorrelationID(log, withCORS), nil
}
