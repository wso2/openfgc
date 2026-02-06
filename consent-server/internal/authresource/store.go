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

	"github.com/wso2/openfgc/internal/authresource/model"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/database/provider"
	"github.com/wso2/openfgc/internal/system/stores/interfaces"
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

	QueryDeleteAuthResourcesByConsentID = dbmodel.DBQuery{
		ID:    "DELETE_AUTH_RESOURCES_BY_CONSENT_ID",
		Query: "DELETE FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?",
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
		return nil, nil
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

// DeleteByConsentID deletes all auth resources for a consent within a transaction
func (s *store) DeleteByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteAuthResourcesByConsentID, consentID, orgID)
	return err
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

// getString extracts a string value from a row, handling both string and []byte types
func getString(row map[string]interface{}, key string) string {
	if v, ok := row[key].(string); ok {
		return v
	}
	if v, ok := row[key].([]byte); ok {
		return string(v)
	}
	return ""
}

// getStringPtr extracts a string pointer from a row, handling both string and []byte types
func getStringPtr(row map[string]interface{}, key string) *string {
	if v, ok := row[key].(string); ok {
		return &v
	}
	if v, ok := row[key].([]byte); ok {
		str := string(v)
		return &str
	}
	return nil
}

// getInt64 extracts an int64 value from a row
func getInt64(row map[string]interface{}, key string) int64 {
	if v, ok := row[key].(int64); ok {
		return v
	}
	return 0
}

// mapToAuthResource converts a database row map to AuthResource
// Note: DBClient normalizes column names to lowercase
func mapToAuthResource(row map[string]interface{}) *model.AuthResource {
	return &model.AuthResource{
		AuthID:      getString(row, "auth_id"),
		ConsentID:   getString(row, "consent_id"),
		AuthType:    getString(row, "auth_type"),
		UserID:      getStringPtr(row, "user_id"),
		AuthStatus:  getString(row, "auth_status"),
		UpdatedTime: getInt64(row, "updated_time"),
		Resources:   getStringPtr(row, "resources"),
		OrgID:       getString(row, "org_id"),
	}
}
