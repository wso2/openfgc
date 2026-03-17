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

// Package consentelement provides consent element management functionality.
package consentelement

import (
	"context"
	"fmt"
	"strings"

	"github.com/wso2/openfgc/internal/consentelement/model"
	dbconst "github.com/wso2/openfgc/internal/system/database/constants"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/database/provider"
	dbutils "github.com/wso2/openfgc/internal/system/database/utils"
	"github.com/wso2/openfgc/internal/system/stores/interfaces"
)

// DBQuery objects for all consent element operations
var (
	QueryCreateElement = dbmodel.DBQuery{
		ID:            "CREATE_CONSENT_ELEMENT",
		Query:         "INSERT INTO CONSENT_ELEMENT (ID, NAME, DESCRIPTION, TYPE, ORG_ID) VALUES (?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT_ELEMENT (ID, NAME, DESCRIPTION, TYPE, ORG_ID) VALUES ($1, $2, $3, $4, $5)",
	}

	QueryGetElementByID = dbmodel.DBQuery{
		ID:            "GET_CONSENT_ELEMENT_BY_ID",
		Query:         "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ID = $1 AND ORG_ID = $2",
	}

	QueryGetElementByName = dbmodel.DBQuery{
		ID:            "GET_CONSENT_ELEMENT_BY_NAME",
		Query:         "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE NAME = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE NAME = $1 AND ORG_ID = $2",
	}

	QueryListElements = dbmodel.DBQuery{
		ID:            "LIST_CONSENT_ELEMENTS",
		Query:         "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ORG_ID = ? ORDER BY NAME LIMIT ? OFFSET ?",
		PostgresQuery: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ORG_ID = $1 ORDER BY NAME LIMIT $2 OFFSET $3",
	}

	QueryListElementsWithName = dbmodel.DBQuery{
		ID:            "LIST_CONSENT_ELEMENTS_WITH_NAME",
		Query:         "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ORG_ID = ? AND NAME LIKE ? ORDER BY NAME LIMIT ? OFFSET ?",
		PostgresQuery: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ORG_ID = $1 AND NAME LIKE $2 ESCAPE '|' ORDER BY NAME LIMIT $3 OFFSET $4",
		SQLiteQuery:   "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ORG_ID = ? AND NAME LIKE ? ESCAPE '|' ORDER BY NAME LIMIT ? OFFSET ?",
	}

	QueryCountElements = dbmodel.DBQuery{
		ID:            "COUNT_CONSENT_ELEMENTS",
		Query:         "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE ORG_ID = ?",
		PostgresQuery: "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE ORG_ID = $1",
	}

	QueryCountElementsWithName = dbmodel.DBQuery{
		ID:            "COUNT_CONSENT_ELEMENTS_WITH_NAME",
		Query:         "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE ORG_ID = ? AND NAME LIKE ?",
		PostgresQuery: "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE ORG_ID = $1 AND NAME LIKE $2 ESCAPE '|'",
		SQLiteQuery:   "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE ORG_ID = ? AND NAME LIKE ? ESCAPE '|'",
	}

	QueryUpdateElement = dbmodel.DBQuery{
		ID:            "UPDATE_CONSENT_ELEMENT",
		Query:         "UPDATE CONSENT_ELEMENT SET NAME = ?, DESCRIPTION = ?, TYPE = ? WHERE ID = ? AND ORG_ID = ?",
		PostgresQuery: "UPDATE CONSENT_ELEMENT SET NAME = $1, DESCRIPTION = $2, TYPE = $3 WHERE ID = $4 AND ORG_ID = $5",
	}

	QueryDeleteElement = dbmodel.DBQuery{
		ID:            "DELETE_CONSENT_ELEMENT",
		Query:         "DELETE FROM CONSENT_ELEMENT WHERE ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM CONSENT_ELEMENT WHERE ID = $1 AND ORG_ID = $2",
	}

	QueryCheckElementNameExists = dbmodel.DBQuery{
		ID:            "CHECK_ELEMENT_NAME_EXISTS",
		Query:         "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE NAME = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE NAME = $1 AND ORG_ID = $2",
	}

	QueryCreateProperty = dbmodel.DBQuery{
		ID:            "CREATE_ELEMENT_PROPERTY",
		Query:         "INSERT INTO CONSENT_ELEMENT_PROPERTY (ELEMENT_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES (?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT_ELEMENT_PROPERTY (ELEMENT_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES ($1, $2, $3, $4)",
	}

	QueryGetPropertiesByElementID = dbmodel.DBQuery{
		ID:            "GET_PROPERTIES_BY_ELEMENT_ID",
		Query:         "SELECT ELEMENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ELEMENT_PROPERTY WHERE ELEMENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT ELEMENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ELEMENT_PROPERTY WHERE ELEMENT_ID = $1 AND ORG_ID = $2",
	}

	QueryDeletePropertiesByElementID = dbmodel.DBQuery{
		ID:            "DELETE_PROPERTIES_BY_ELEMENT_ID",
		Query:         "DELETE FROM CONSENT_ELEMENT_PROPERTY WHERE ELEMENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM CONSENT_ELEMENT_PROPERTY WHERE ELEMENT_ID = $1 AND ORG_ID = $2",
	}

	QueryGetIDsByNames = dbmodel.DBQuery{
		ID:    "GET_IDS_BY_NAMES",
		Query: "SELECT ID, NAME FROM CONSENT_ELEMENT WHERE ORG_ID = ? AND NAME IN (%s)",
	}
)

// store implements the interfaces.ConsentElementStore interface
type store struct {
}

// NewConsentElementStore creates a new consent element store
func NewConsentElementStore() interfaces.ConsentElementStore {
	return &store{}
}

// getDBClient retrieves the database client from the provider
func (s *store) getDBClient() (provider.DBClientInterface, error) {
	return provider.GetDBProvider().GetConsentDBClient()
}

// Create creates a new consent element within a transaction
func (elementStore *store) Create(tx dbmodel.TxInterface, element *model.ConsentElement) error {
	_, err := tx.Exec(QueryCreateElement,
		element.ID, element.Name, element.Description, element.Type, element.OrgID)
	return err
}

// GetByID retrieves a consent element by ID
func (elementStore *store) GetByID(ctx context.Context, elementID, orgID string) (*model.ConsentElement, error) {
	dbClient, err := elementStore.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetElementByID, elementID, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return mapToConsentElement(rows[0]), nil
}

// GetByName retrieves a consent element by name
func (elementStore *store) GetByName(ctx context.Context, name, orgID string) (*model.ConsentElement, error) {
	dbClient, err := elementStore.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetElementByName, name, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return mapToConsentElement(rows[0]), nil
}

// List retrieves a paginated list of consent elements
func (elementStore *store) List(ctx context.Context, orgID string, limit, offset int, name string) ([]model.ConsentElement, int, error) {
	dbClient, err := elementStore.getDBClient()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get database client: %w", err)
	}

	var countRows []map[string]interface{}
	var rows []map[string]interface{}

	// Use different queries based on whether name filter is provided
	if name != "" {
		// Escape SQL wildcard characters to prevent unintended matches
		var escaper *strings.Replacer
		if dbClient.GetDBType() == dbconst.DatabaseTypeSQLite || dbClient.GetDBType() == dbconst.DatabaseTypePostgres {
			// SQLite/PostgreSQL: use '|' as escape char (single char, no quoting issues)
			escaper = strings.NewReplacer("|", "||", "%", "|%", "_", "|_")
		} else {
			// MySQL: use '\' as the escape character (MySQL default)
			escaper = strings.NewReplacer("%", "\\%", "_", "\\_")
		}
		escapedName := escaper.Replace(name)
		// Add wildcards for partial match (case-insensitive search)
		namePattern := "%" + escapedName + "%"

		// Get total count with name filter
		countRows, err = dbClient.Query(QueryCountElementsWithName, orgID, namePattern)
		if err != nil {
			return nil, 0, err
		}

		// Get paginated results with name filter
		rows, err = dbClient.Query(QueryListElementsWithName, orgID, namePattern, limit, offset)
		if err != nil {
			return nil, 0, err
		}
	} else {
		// Get total count without name filter
		countRows, err = dbClient.Query(QueryCountElements, orgID)
		if err != nil {
			return nil, 0, err
		}

		// Get paginated results without name filter
		rows, err = dbClient.Query(QueryListElements, orgID, limit, offset)
		if err != nil {
			return nil, 0, err
		}
	}

	totalCount := 0
	if len(countRows) > 0 {
		if count, ok := countRows[0]["count"].(int64); ok {
			totalCount = int(count)
		}
	}

	elements := make([]model.ConsentElement, 0, len(rows))
	for _, row := range rows {
		element := mapToConsentElement(row)
		if element != nil {
			elements = append(elements, *element)
		}
	}

	return elements, totalCount, nil
}

// Update updates an existing consent element within a transaction
func (elementStore *store) Update(tx dbmodel.TxInterface, element *model.ConsentElement) error {
	_, err := tx.Exec(QueryUpdateElement,
		element.Name, element.Description, element.Type, element.ID, element.OrgID)
	return err
}

// Delete deletes a consent element within a transaction
func (elementStore *store) Delete(tx dbmodel.TxInterface, elementID, orgID string) error {
	_, err := tx.Exec(QueryDeleteElement, elementID, orgID)
	return err
}

// CheckNameExists checks if a element name already exists
func (elementStore *store) CheckNameExists(ctx context.Context, name, orgID string) (bool, error) {
	dbClient, err := elementStore.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryCheckElementNameExists, name, orgID)
	if err != nil {
		return false, err
	}

	if len(rows) > 0 {
		if count, ok := rows[0]["count"].(int64); ok {
			return count > 0, nil
		}
	}
	return false, nil
}

// CreateProperties creates multiple element properties within a transaction
func (elementStore *store) CreateProperties(tx dbmodel.TxInterface, properties []model.ConsentElementProperty) error {
	for _, prop := range properties {
		_, err := tx.Exec(QueryCreateProperty,
			prop.ElementID, prop.Key, prop.Value, prop.OrgID)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPropertiesByElementID retrieves all properties for an element
func (elementStore *store) GetPropertiesByElementID(ctx context.Context, elementID, orgID string) ([]model.ConsentElementProperty, error) {
	dbClient, err := elementStore.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetPropertiesByElementID, elementID, orgID)
	if err != nil {
		return nil, err
	}

	properties := make([]model.ConsentElementProperty, 0, len(rows))
	for _, row := range rows {
		prop := mapToConsentElementProperty(row)
		if prop != nil {
			properties = append(properties, *prop)
		}
	}

	return properties, nil
}

// DeletePropertiesByElementID deletes all properties for an element within a transaction
func (elementStore *store) DeletePropertiesByElementID(tx dbmodel.TxInterface, elementID, orgID string) error {
	_, err := tx.Exec(QueryDeletePropertiesByElementID, elementID, orgID)
	return err
}

// getString extracts a string value from a database row column that may be string or []byte
func getString(row map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := row[key]; ok {
			switch v := val.(type) {
			case string:
				return v
			case []byte:
				return string(v)
			}
		}
	}
	return ""
}

// mapToConsentElement maps a database row to ConsentElement model
func mapToConsentElement(row map[string]interface{}) *model.ConsentElement {
	if row == nil {
		return nil
	}

	element := &model.ConsentElement{
		ID:         getString(row, "id"),
		Name:       getString(row, "name"),
		Type:       getString(row, "type"),
		OrgID:      getString(row, "org_id"),
		Properties: make(map[string]string),
	}

	if desc := getString(row, "description"); desc != "" {
		element.Description = &desc
	}

	return element
}

// mapToConsentElementProperty maps a database row to ConsentElementProperty model
func mapToConsentElementProperty(row map[string]interface{}) *model.ConsentElementProperty {
	if row == nil {
		return nil
	}

	return &model.ConsentElementProperty{
		ElementID: getString(row, "element_id"),
		Key:       getString(row, "att_key"),
		Value:     getString(row, "att_value"),
		OrgID:     getString(row, "org_id"),
	}
}

// GetIDsByNames retrieves element IDs by their names (batch lookup)
func (elementStore *store) GetIDsByNames(ctx context.Context, names []string, orgID string) (map[string]string, error) {
	if len(names) == 0 {
		return make(map[string]string), nil
	}

	// Build placeholders for IN clause
	placeholders := ""
	args := make([]interface{}, 0, len(names)+1)
	args = append(args, orgID)

	for i, name := range names {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args = append(args, name)
	}

	// Format query with placeholders
	queryStr := fmt.Sprintf(QueryGetIDsByNames.Query, placeholders)

	// Create query object with formatted SQL
	formattedQuery := dbmodel.DBQuery{
		ID:            "GET_IDS_BY_NAMES_DYNAMIC",
		Query:         queryStr,
		PostgresQuery: dbutils.ConvertToPostgresParams(queryStr),
	}

	dbClient, err := elementStore.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(formattedQuery, args...)
	if err != nil {
		return nil, err
	}

	// Build name -> ID map
	// Note: DBClient normalizes column names to lowercase
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		id := getString(row, "id")
		name := getString(row, "name")
		if id != "" && name != "" {
			result[name] = id
		}
	}
	return result, nil
}
