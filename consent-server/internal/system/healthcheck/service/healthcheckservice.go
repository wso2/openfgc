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

// Package service provides health check-related business logic and operations.
package service

import (
	"sync"

	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/database/provider"
	"github.com/wso2/openfgc/internal/system/healthcheck/model"
	"github.com/wso2/openfgc/internal/system/log"
)

var (
	instance *HealthCheckService
	once     sync.Once
)

// Simple health check query
var healthCheckQuery = dbmodel.DBQuery{
	ID:    "health-check",
	Query: "SELECT 1",
}

// HealthCheckServiceInterface defines the interface for the health check service.
type HealthCheckServiceInterface interface {
	CheckReadiness() model.ServerStatus
}

// HealthCheckService is the default implementation of the HealthCheckServiceInterface.
type HealthCheckService struct {
	DBProvider provider.DBProviderInterface
}

// GetHealthCheckService returns a singleton instance of HealthCheckService.
func GetHealthCheckService() HealthCheckServiceInterface {
	once.Do(func() {
		instance = &HealthCheckService{
			DBProvider: provider.GetDBProvider(),
		}
	})
	return instance
}

// CheckReadiness checks the readiness of the server and its dependencies.
func (hcs *HealthCheckService) CheckReadiness() model.ServerStatus {
	consentDBStatus := model.ServiceStatus{
		ServiceName: "ConsentDB",
		Status:      hcs.checkConsentDatabaseStatus(),
	}

	status := model.StatusUp
	if consentDBStatus.Status == model.StatusDown {
		status = model.StatusDown
	}

	return model.ServerStatus{
		Status: status,
		ServiceStatus: []model.ServiceStatus{
			consentDBStatus,
		},
	}
}

// checkConsentDatabaseStatus checks the status of the consent database.
func (hcs *HealthCheckService) checkConsentDatabaseStatus() model.Status {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "HealthCheckService"))

	dbClient, err := hcs.DBProvider.GetConsentDBClient()
	if err != nil {
		logger.Error("Failed to get consent database client", log.Error(err))
		return model.StatusDown
	}

	// Simple ping query to check database connectivity
	_, err = dbClient.Query(healthCheckQuery)
	if err != nil {
		logger.Error("Failed to ping consent database", log.Error(err))
		return model.StatusDown
	}

	return model.StatusUp
}
