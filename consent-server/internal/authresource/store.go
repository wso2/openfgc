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

package authresource

import (
	"context"
	"fmt"

	"github.com/wso2/consent-management-api/internal/authresource/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/stores/interfaces"
)

// DBQuery objects for all auth resource operations
var (
	QueryCreateAuthResource = dbmodel.DBQuery{
		ID:    "CREATE_AUTH_RESOURCE",
		Query: "INSERT INTO CONSENT_AUTH_RESOURCE (AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
	}

	QueryGetAuthResourceByID = dbmodel.DBQuery{
		ID:    "GET_AUTH_RESOURCE_BY_ID",
		Query: "SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID FROM CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryGetAuthResourcesByConsentID = dbmodel.DBQuery{
		ID:    "GET_AUTH_RESOURCES_BY_CONSENT_ID",
		Query: "SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryUpdateAuthResource = dbmodel.DBQuery{
		ID:    "UPDATE_AUTH_RESOURCE",
		Query: "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = ?, USER_ID = ?, RESOURCES = ?, UPDATED_TIME = ? WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryUpdateAuthResourceStatus = dbmodel.DBQuery{
		ID:    "UPDATE_AUTH_RESOURCE_STATUS",
		Query: "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = ?, UPDATED_TIME = ? WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryDeleteAuthResource = dbmodel.DBQuery{
		ID:    "DELETE_AUTH_RESOURCE",
		Query: "DELETE FROM CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryDeleteAuthResourcesByConsentID = dbmodel.DBQuery{
		ID:    "DELETE_AUTH_RESOURCES_BY_CONSENT_ID",
		Query: "DELETE FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryCheckAuthResourceExists = dbmodel.DBQuery{
		ID:    "CHECK_AUTH_RESOURCE_EXISTS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryGetAuthResourcesByUserID = dbmodel.DBQuery{
		ID:    "GET_AUTH_RESOURCES_BY_USER_ID",
		Query: "SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID FROM CONSENT_AUTH_RESOURCE WHERE USER_ID = ? AND ORG_ID = ?",
	}

	QueryUpdateAllStatusByConsentID = dbmodel.DBQuery{
		ID:    "UPDATE_ALL_STATUS_BY_CONSENT_ID",
		Query: "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = ?, UPDATED_TIME = ? WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryGetAuthResourcesByConsentIDs = dbmodel.DBQuery{
		ID:    "GET_AUTH_RESOURCES_BY_CONSENT_IDS",
		Query: "", // Built dynamically
	}
)

// store implements interfaces.AuthResourceStore
type store struct {
}

// NewAuthResourceStore creates a new auth resource store
func NewAuthResourceStore() interfaces.AuthResourceStore {
	return &store{}
}

// getDBClient retrieves the database client from the provider
func (s *store) getDBClient() (provider.DBClientInterface, error) {
	return provider.GetDBProvider().GetConsentDBClient()
}

// Create creates a new auth resource within a transaction
func (s *store) Create(tx dbmodel.TxInterface, authResource *model.AuthResource) error {
	_, err := tx.Exec(QueryCreateAuthResource,
		authResource.AuthID,
		authResource.ConsentID,
		authResource.AuthType,
		authResource.UserID,
		authResource.AuthStatus,
		authResource.UpdatedTime,
		authResource.Resources,
		authResource.OrgID,
	)
	return err
}

// GetByID retrieves an auth resource by ID
func (s *store) GetByID(ctx context.Context, authID, orgID string) (*model.AuthResource, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	results, err := dbClient.Query(QueryGetAuthResourceByID, authID, orgID)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("auth resource not found")
	}
	return mapToAuthResource(results[0]), nil
}

// GetByConsentID retrieves all auth resources for a consent
func (s *store) GetByConsentID(ctx context.Context, consentID, orgID string) ([]model.AuthResource, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	results, err := dbClient.Query(QueryGetAuthResourcesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	authResources := make([]model.AuthResource, 0, len(results))
	for _, row := range results {
		authResources = append(authResources, *mapToAuthResource(row))
	}
	return authResources, nil
}

// Update updates an auth resource within a transaction
func (s *store) Update(tx dbmodel.TxInterface, authResource *model.AuthResource) error {
	_, err := tx.Exec(QueryUpdateAuthResource,
		authResource.AuthStatus,
		authResource.UserID,
		authResource.Resources,
		authResource.UpdatedTime,
		authResource.AuthID,
		authResource.OrgID,
	)
	return err
}

// UpdateStatus updates only the status of an auth resource within a transaction
func (s *store) UpdateStatus(tx dbmodel.TxInterface, authID, orgID, status string, updatedTime int64) error {
	_, err := tx.Exec(QueryUpdateAuthResourceStatus, status, updatedTime, authID, orgID)
	return err
}

// Delete deletes an auth resource within a transaction
func (s *store) Delete(tx dbmodel.TxInterface, authID, orgID string) error {
	_, err := tx.Exec(QueryDeleteAuthResource, authID, orgID)
	return err
}

// DeleteByConsentID deletes all auth resources for a consent within a transaction
func (s *store) DeleteByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteAuthResourcesByConsentID, consentID, orgID)
	return err
}

// Exists checks if an auth resource exists
func (s *store) Exists(ctx context.Context, authID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}

	results, err := dbClient.Query(QueryCheckAuthResourceExists, authID, orgID)
	if err != nil {
		return false, err
	}
	if len(results) == 0 {
		return false, nil
	}
	count, ok := results[0]["count"].(int64)
	if !ok {
		return false, nil
	}
	return count > 0, nil
}

// GetByUserID retrieves all auth resources for a user
func (s *store) GetByUserID(ctx context.Context, userID, orgID string) ([]model.AuthResource, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	results, err := dbClient.Query(QueryGetAuthResourcesByUserID, userID, orgID)
	if err != nil {
		return nil, err
	}

	authResources := make([]model.AuthResource, 0, len(results))
	for _, row := range results {
		authResources = append(authResources, *mapToAuthResource(row))
	}
	return authResources, nil
}

// UpdateAllStatusByConsentID updates status for all auth resources of a consent within a transaction
func (s *store) UpdateAllStatusByConsentID(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error {
	_, err := tx.Exec(QueryUpdateAllStatusByConsentID, status, updatedTime, consentID, orgID)
	return err
}

// GetByConsentIDs retrieves auth resources for multiple consents
func (s *store) GetByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]model.AuthResource, error) {
	if len(consentIDs) == 0 {
		return []model.AuthResource{}, nil
	}

	// Build placeholders for IN clause
	placeholders := ""
	args := make([]interface{}, 0, len(consentIDs)+1)
	for i, id := range consentIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args = append(args, id)
	}
	args = append(args, orgID)

	// Build dynamic query
	query := dbmodel.DBQuery{
		ID:    QueryGetAuthResourcesByConsentIDs.ID,
		Query: fmt.Sprintf("SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID IN (%s) AND ORG_ID = ?", placeholders),
	}

	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	results, err := dbClient.Query(query, args...)
	if err != nil {
		return nil, err
	}

	authResources := make([]model.AuthResource, 0, len(results))
	for _, row := range results {
		authResources = append(authResources, *mapToAuthResource(row))
	}
	return authResources, nil
}

// mapToAuthResource converts a database row map to AuthResource
// Note: DBClient normalizes column names to lowercase
func mapToAuthResource(row map[string]interface{}) *model.AuthResource {
	authResource := &model.AuthResource{}

	// Handle string columns (may be string or []byte from MySQL)
	if v, ok := row["auth_id"].(string); ok {
		authResource.AuthID = v
	} else if v, ok := row["auth_id"].([]byte); ok {
		authResource.AuthID = string(v)
	}

	if v, ok := row["consent_id"].(string); ok {
		authResource.ConsentID = v
	} else if v, ok := row["consent_id"].([]byte); ok {
		authResource.ConsentID = string(v)
	}

	if v, ok := row["auth_type"].(string); ok {
		authResource.AuthType = v
	} else if v, ok := row["auth_type"].([]byte); ok {
		authResource.AuthType = string(v)
	}

	if v, ok := row["user_id"].(string); ok {
		authResource.UserID = &v
	} else if v, ok := row["user_id"].([]byte); ok {
		str := string(v)
		authResource.UserID = &str
	}

	if v, ok := row["auth_status"].(string); ok {
		authResource.AuthStatus = v
	} else if v, ok := row["auth_status"].([]byte); ok {
		authResource.AuthStatus = string(v)
	}

	if v, ok := row["updated_time"].(int64); ok {
		authResource.UpdatedTime = v
	}

	if v, ok := row["resources"].(string); ok {
		authResource.Resources = &v
	} else if v, ok := row["resources"].([]byte); ok {
		str := string(v)
		authResource.Resources = &str
	}

	if v, ok := row["org_id"].(string); ok {
		authResource.OrgID = v
	} else if v, ok := row["org_id"].([]byte); ok {
		authResource.OrgID = string(v)
	}

	return authResource
}
