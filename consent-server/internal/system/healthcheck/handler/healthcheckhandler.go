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

// Package handler provides HTTP handlers for managing health check related API requests.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/wso2/openfgc/internal/system/healthcheck/model"
	"github.com/wso2/openfgc/internal/system/healthcheck/service"
	"github.com/wso2/openfgc/internal/system/log"
)

// HealthCheckHandler defines the handler for managing health check API requests.
type HealthCheckHandler struct {
	service service.HealthCheckServiceInterface
}

// NewHealthCheckHandler creates a new instance of HealthCheckHandler.
func NewHealthCheckHandler() *HealthCheckHandler {
	return &HealthCheckHandler{
		service: service.GetHealthCheckService(),
	}
}

// HandleLivenessRequest handles the health check liveness request.
func (hch *HealthCheckHandler) HandleLivenessRequest(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "HealthCheckHandler"))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"UP"}`))
	logger.Debug("Health Check Liveness response sent")
}

// HandleReadinessRequest handles the health check readiness request.
func (hch *HealthCheckHandler) HandleReadinessRequest(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "HealthCheckHandler"))

	serverstatus := hch.service.CheckReadiness()

	if serverstatus.Status != model.StatusUp {
		logger.Error("Readiness check failed", log.String("serverstatus", string(serverstatus.Status)))
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		logger.Debug("Readiness check passed", log.String("serverstatus", string(serverstatus.Status)))
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(serverstatus)
	if err != nil {
		logger.Error("Error while encoding readiness response", log.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logger.Debug("Health Check Readiness response sent")
}
