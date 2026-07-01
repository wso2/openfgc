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
	"strconv"
	"strings"

	"github.com/wso2/openfgc/consent-server/internal/authresource/model"
	dbmodel "github.com/wso2/openfgc/consent-server/internal/system/database/model"
	"github.com/wso2/openfgc/consent-server/internal/system/database/provider"
	dbutils "github.com/wso2/openfgc/consent-server/internal/system/database/utils"
	"github.com/wso2/openfgc/consent-server/internal/system/stores/interfaces"
)

// authResourceColumns is the SELECT column list shared across CONSENT_AUTH_RESOURCE table queries.
const authResourceColumns = "AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID"

// Pre-defined DBQuery objects for simple, single-path operations.
var (
	QueryCreateAuthResource = dbmodel.DBQuery{
		ID:            "CREATE_AUTH_RESOURCE",
		Query:         "INSERT INTO CONSENT_AUTH_RESOURCE (" + authResourceColumns + ") VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT_AUTH_RESOURCE (" + authResourceColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
	}

	QueryGetAuthResourceByID = dbmodel.DBQuery{
		ID:            "GET_AUTH_RESOURCE_BY_ID",
		Query:         "SELECT " + authResourceColumns + " FROM CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT " + authResourceColumns + " FROM CONSENT_AUTH_RESOURCE WHERE AUTH_ID = $1 AND ORG_ID = $2",
	}

	QueryGetAuthResourcesByConsentID = dbmodel.DBQuery{
		ID:            "GET_AUTH_RESOURCES_BY_CONSENT_ID",
		Query:         "SELECT " + authResourceColumns + " FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT " + authResourceColumns + " FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = $1 AND ORG_ID = $2",
	}

	QueryUpdateAuthResource = dbmodel.DBQuery{
		ID:            "UPDATE_AUTH_RESOURCE",
		Query:         "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = ?, USER_ID = ?, RESOURCES = ?, UPDATED_TIME = ? WHERE AUTH_ID = ? AND ORG_ID = ?",
		PostgresQuery: "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = $1, USER_ID = $2, RESOURCES = $3, UPDATED_TIME = $4 WHERE AUTH_ID = $5 AND ORG_ID = $6",
	}

	QueryDeleteAuthResourcesByConsentID = dbmodel.DBQuery{
		ID:            "DELETE_AUTH_RESOURCES_BY_CONSENT_ID",
		Query:         "DELETE FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = $1 AND ORG_ID = $2",
	}

	QueryUpdateAllStatusByConsentID = dbmodel.DBQuery{
		ID:            "UPDATE_ALL_STATUS_BY_CONSENT_ID",
		Query:         "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = ?, UPDATED_TIME = ? WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = $1, UPDATED_TIME = $2 WHERE CONSENT_ID = $3 AND ORG_ID = $4",
	}

	// Dynamic query stub — built at runtime based on the number of consent IDs.
	QueryGetAuthResourcesByConsentIDs = dbmodel.DBQuery{ID: "GET_AUTH_RESOURCES_BY_CONSENT_IDS", Query: ""}
)

// store implements the interfaces.AuthResourceStore interface.
type store struct{}

// NewAuthResourceStore creates a new auth resource store.
func NewAuthResourceStore() interfaces.AuthResourceStore {
	return &store{}
}

func (s *store) getDBClient() (provider.DBClientInterface, error) {
	return provider.GetDBProvider().GetConsentDBClient()
}

// =============================================================================
// Write operations (transactional)
// =============================================================================

// Create inserts a new CONSENT_AUTH_RESOURCE row within a transaction.
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

// Update overwrites AUTH_STATUS, USER_ID, RESOURCES, and UPDATED_TIME for an auth resource
// within a transaction. AUTH_ID and ORG_ID are used as the lookup key.
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

// DeleteByConsentID removes all CONSENT_AUTH_RESOURCE rows for a consent within a transaction.
func (s *store) DeleteByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteAuthResourcesByConsentID, consentID, orgID)
	return err
}

// UpdateAllStatusByConsentID sets AUTH_STATUS and UPDATED_TIME for every auth resource
// belonging to a consent within a transaction.
func (s *store) UpdateAllStatusByConsentID(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error {
	_, err := tx.Exec(QueryUpdateAllStatusByConsentID, status, updatedTime, consentID, orgID)
	return err
}

// =============================================================================
// Read operations
// =============================================================================

func queryRowsInTx(tx dbmodel.TxInterface, query dbmodel.DBQuery, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{}, len(columns))
		for i, column := range columns {
			row[strings.ToLower(column)] = values[i]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetByConsentIDTx returns all auth resource rows for a consent within a transaction.
func (s *store) GetByConsentIDTx(tx dbmodel.TxInterface, consentID, orgID string) ([]model.AuthResource, error) {
	rows, err := queryRowsInTx(tx, QueryGetAuthResourcesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	authResources := make([]model.AuthResource, 0, len(rows))
	for _, row := range rows {
		authResources = append(authResources, *mapToAuthResource(row))
	}
	return authResources, nil
}

// GetByID returns the CONSENT_AUTH_RESOURCE row for the given AUTH_ID, or nil if not found.
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

// GetByConsentID returns all auth resource rows for a consent.
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

// GetByConsentIDs returns auth resource rows for multiple consents in one query.
// Returns an empty slice (not an error) when consentIDs is empty.
func (s *store) GetByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]model.AuthResource, error) {
	if len(consentIDs) == 0 {
		return []model.AuthResource{}, nil
	}

	// Build the IN-clause placeholders and argument list dynamically.
	placeholders := make([]byte, 0, len(consentIDs)*3)
	args := make([]interface{}, 0, len(consentIDs)+1)
	for i, id := range consentIDs {
		if i > 0 {
			placeholders = append(placeholders, ',', ' ')
		}
		placeholders = append(placeholders, '?')
		args = append(args, id)
	}
	args = append(args, orgID)

	mysqlQuery := fmt.Sprintf(
		"SELECT "+authResourceColumns+" FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID IN (%s) AND ORG_ID = ?",
		placeholders,
	)
	query := dbmodel.DBQuery{
		ID:            QueryGetAuthResourcesByConsentIDs.ID,
		Query:         mysqlQuery,
		PostgresQuery: dbutils.ConvertToPostgresParams(mysqlQuery),
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

// =============================================================================
// Mappers — DBClient normalizes column names to lowercase.
// =============================================================================

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

// =============================================================================
// DB row helpers
// =============================================================================

func getString(row map[string]interface{}, key string) string {
	switch v := row[key].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	return ""
}

func getStringPtr(row map[string]interface{}, key string) *string {
	switch v := row[key].(type) {
	case string:
		return &v
	case []byte:
		s := string(v)
		return &s
	}
	return nil
}

func getInt64(row map[string]interface{}, key string) int64 {
	switch v := row[key].(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case []uint8:
		if parsed, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}
