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

	"github.com/wso2/consent-management-api/internal/consentelement/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/stores/interfaces"
)

// DBQuery objects for all consent element operations
var (
	QueryCreateElement = dbmodel.DBQuery{
		ID:    "CREATE_CONSENT_ELEMENT",
		Query: "INSERT INTO CONSENT_ELEMENT (ID, NAME, DESCRIPTION, TYPE, ORG_ID) VALUES (?, ?, ?, ?, ?)",
	}

	QueryGetElementByID = dbmodel.DBQuery{
		ID:    "GET_CONSENT_ELEMENT_BY_ID",
		Query: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ID = ? AND ORG_ID = ?",
	}

	QueryGetElementByName = dbmodel.DBQuery{
		ID:    "GET_CONSENT_ELEMENT_BY_NAME",
		Query: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE NAME = ? AND ORG_ID = ?",
	}

	QueryListElements = dbmodel.DBQuery{
		ID:    "LIST_CONSENT_ELEMENTS",
		Query: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ORG_ID = ? ORDER BY NAME LIMIT ? OFFSET ?",
	}

	QueryListElementsWithName = dbmodel.DBQuery{
		ID:    "LIST_CONSENT_ELEMENTS_WITH_NAME",
		Query: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_ELEMENT WHERE ORG_ID = ? AND NAME LIKE ? ORDER BY NAME LIMIT ? OFFSET ?",
	}

	QueryCountElements = dbmodel.DBQuery{
		ID:    "COUNT_CONSENT_ELEMENTS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE ORG_ID = ?",
	}

	QueryCountElementsWithName = dbmodel.DBQuery{
		ID:    "COUNT_CONSENT_ELEMENTS_WITH_NAME",
		Query: "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE ORG_ID = ? AND NAME LIKE ?",
	}

	QueryUpdateElement = dbmodel.DBQuery{
		ID:    "UPDATE_CONSENT_ELEMENT",
		Query: "UPDATE CONSENT_ELEMENT SET NAME = ?, DESCRIPTION = ?, TYPE = ? WHERE ID = ? AND ORG_ID = ?",
	}

	QueryDeleteElement = dbmodel.DBQuery{
		ID:    "DELETE_CONSENT_ELEMENT",
		Query: "DELETE FROM CONSENT_ELEMENT WHERE ID = ? AND ORG_ID = ?",
	}

	QueryCheckElementNameExists = dbmodel.DBQuery{
		ID:    "CHECK_ELEMENT_NAME_EXISTS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_ELEMENT WHERE NAME = ? AND ORG_ID = ?",
	}

	QueryCreateProperty = dbmodel.DBQuery{
		ID:    "CREATE_ELEMENT_PROPERTY",
		Query: "INSERT INTO CONSENT_ELEMENT_PROPERTY (ELEMENT_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES (?, ?, ?, ?)",
	}

	QueryGetPropertiesByElementID = dbmodel.DBQuery{
		ID:    "GET_PROPERTIES_BY_ELEMENT_ID",
		Query: "SELECT ELEMENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ELEMENT_PROPERTY WHERE ELEMENT_ID = ? AND ORG_ID = ?",
	}

	QueryDeletePropertiesByElementID = dbmodel.DBQuery{
		ID:    "DELETE_PROPERTIES_BY_ELEMENT_ID",
		Query: "DELETE FROM CONSENT_ELEMENT_PROPERTY WHERE ELEMENT_ID = ? AND ORG_ID = ?",
	}

	QueryLinkElementToConsent = dbmodel.DBQuery{
		ID:    "LINK_ELEMENT_TO_CONSENT",
		Query: "INSERT INTO CONSENT_ELEMENT_MAPPING (CONSENT_ID, ELEMENT_ID, ORG_ID, VALUE, IS_USER_APPROVED, IS_MANDATORY) VALUES (?, ?, ?, ?, ?, ?)",
	}

	QueryGetMappingsByConsentID = dbmodel.DBQuery{
		ID: "GET_MAPPINGS_BY_CONSENT_ID",
		Query: `SELECT cpm.CONSENT_ID, cpm.ELEMENT_ID, cpm.ORG_ID, cpm.VALUE, cpm.IS_USER_APPROVED, cpm.IS_MANDATORY, cp.NAME
				FROM CONSENT_ELEMENT_MAPPING cpm
				INNER JOIN CONSENT_ELEMENT cp ON cpm.ELEMENT_ID = cp.ID
				WHERE cpm.CONSENT_ID = ? AND cpm.ORG_ID = ?`,
	}

	QueryGetIDsByNames = dbmodel.DBQuery{
		ID:    "GET_IDS_BY_NAMES",
		Query: "SELECT ID, NAME FROM CONSENT_ELEMENT WHERE ORG_ID = ? AND NAME IN (%s)",
	}

	QueryDeleteMappingsByConsentID = dbmodel.DBQuery{
		ID:    "DELETE_MAPPINGS_BY_CONSENT_ID",
		Query: "DELETE FROM CONSENT_ELEMENT_MAPPING WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryGetMappingsByConsentIDs = dbmodel.DBQuery{
		ID:    "GET_MAPPINGS_BY_CONSENT_IDS",
		Query: "", // Built dynamically
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
		// Add wildcards for partial match (case-insensitive search)
		namePattern := "%" + name + "%"

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

// mapToConsentElement maps a database row to ConsentElement model
// Note: DBClient normalizes column names to lowercase
func mapToConsentElement(row map[string]interface{}) *model.ConsentElement {
	if row == nil {
		return nil
	}

	element := &model.ConsentElement{}

	// Handle string columns (may be string or []byte from MySQL)
	if id, ok := row["id"].(string); ok {
		element.ID = id
	} else if id, ok := row["id"].([]byte); ok {
		element.ID = string(id)
	}

	if name, ok := row["name"].(string); ok {
		element.Name = name
	} else if name, ok := row["name"].([]byte); ok {
		element.Name = string(name)
	}

	if desc, ok := row["description"].(string); ok {
		descCopy := desc
		element.Description = &descCopy
	} else if desc, ok := row["description"].([]byte); ok {
		descCopy := string(desc)
		element.Description = &descCopy
	}

	if pType, ok := row["type"].(string); ok {
		element.Type = pType
	} else if pType, ok := row["type"].([]byte); ok {
		element.Type = string(pType)
	}

	if orgID, ok := row["org_id"].(string); ok {
		element.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		element.OrgID = string(orgID)
	}

	// Initialize empty properties map
	element.Properties = make(map[string]string)

	return element
}

// mapToConsentElementProperty maps a database row to ConsentElementProperty model
// Note: DBClient normalizes column names to lowercase
func mapToConsentElementProperty(row map[string]interface{}) *model.ConsentElementProperty {
	if row == nil {
		return nil
	}

	attr := &model.ConsentElementProperty{}

	// Handle string columns (may be string or []byte from MySQL)
	if id, ok := row["id"].(string); ok {
		attr.ID = id
	} else if id, ok := row["id"].([]byte); ok {
		attr.ID = string(id)
	}

	if elementID, ok := row["element_id"].(string); ok {
		attr.ElementID = elementID
	} else if elementID, ok := row["element_id"].([]byte); ok {
		attr.ElementID = string(elementID)
	}

	// Try lowercase first (normalized), then uppercase (raw column names)
	if key, ok := row["attr_key"].(string); ok {
		attr.Key = key
	} else if key, ok := row["attr_key"].([]byte); ok {
		attr.Key = string(key)
	} else if key, ok := row["att_key"].(string); ok {
		attr.Key = key
	} else if key, ok := row["att_key"].([]byte); ok {
		attr.Key = string(key)
	} else if key, ok := row["ATT_KEY"].(string); ok {
		attr.Key = key
	} else if key, ok := row["ATT_KEY"].([]byte); ok {
		attr.Key = string(key)
	}

	if value, ok := row["attr_value"].(string); ok {
		attr.Value = value
	} else if value, ok := row["attr_value"].([]byte); ok {
		attr.Value = string(value)
	} else if value, ok := row["att_value"].(string); ok {
		attr.Value = value
	} else if value, ok := row["att_value"].([]byte); ok {
		attr.Value = string(value)
	} else if value, ok := row["ATT_VALUE"].(string); ok {
		attr.Value = value
	} else if value, ok := row["ATT_VALUE"].([]byte); ok {
		attr.Value = string(value)
	}

	if orgID, ok := row["org_id"].(string); ok {
		attr.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		attr.OrgID = string(orgID)
	}

	return attr
}

// LinkElementToConsent links an element to a consent within a transaction
func (elementStore *store) LinkElementToConsent(tx dbmodel.TxInterface, consentID, elementID, orgID string, value *string, isUserApproved, isMandatory bool) error {
	_, err := tx.Exec(QueryLinkElementToConsent,
		consentID, elementID, orgID, value, isUserApproved, isMandatory)
	return err
}

// GetMappingsByConsentID retrieves all element mappings for a consent with their values
func (elementStore *store) GetMappingsByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentElementMapping, error) {
	dbClient, err := elementStore.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetMappingsByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	mappings := make([]model.ConsentElementMapping, 0, len(rows))
	for _, row := range rows {
		mapping := mapToConsentElementMapping(row)
		if mapping != nil {
			mappings = append(mappings, *mapping)
		}
	}

	return mappings, nil
}

// GetMappingsByConsentIDs retrieves element mappings for multiple consents with their values
func (elementStore *store) GetMappingsByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]model.ConsentElementMapping, error) {
	if len(consentIDs) == 0 {
		return []model.ConsentElementMapping{}, nil
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
		ID: QueryGetMappingsByConsentIDs.ID,
		Query: fmt.Sprintf(`SELECT cpm.CONSENT_ID, cpm.ELEMENT_ID, cpm.ORG_ID, cpm.VALUE, cpm.IS_USER_APPROVED, cpm.IS_MANDATORY, cp.NAME
				FROM CONSENT_ELEMENT_MAPPING cpm
				INNER JOIN CONSENT_ELEMENT cp ON cpm.ELEMENT_ID = cp.ID
				WHERE cpm.CONSENT_ID IN (%s) AND cpm.ORG_ID = ?`, placeholders),
	}

	dbClient, err := elementStore.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(query, args...)
	if err != nil {
		return nil, err
	}

	mappings := make([]model.ConsentElementMapping, 0, len(rows))
	for _, row := range rows {
		mapping := mapToConsentElementMapping(row)
		if mapping != nil {
			mappings = append(mappings, *mapping)
		}
	}

	return mappings, nil
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
	query := fmt.Sprintf(QueryGetIDsByNames.Query, placeholders)

	// Create query object with formatted SQL
	formattedQuery := dbmodel.DBQuery{
		ID:    "GET_IDS_BY_NAMES_DYNAMIC",
		Query: query,
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
		var id, name string

		// Handle both string and []byte types (MySQL returns []byte for strings)
		if idVal, ok := row["id"]; ok {
			if idStr, ok := idVal.(string); ok {
				id = idStr
			} else if idBytes, ok := idVal.([]byte); ok {
				id = string(idBytes)
			}
		}

		if nameVal, ok := row["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
			} else if nameBytes, ok := nameVal.([]byte); ok {
				name = string(nameBytes)
			}
		}

		if id != "" && name != "" {
			result[name] = id
		}
	}
	return result, nil
}

// mapToConsentElementMapping maps a database row to ConsentElementMapping model
// Note: DBClient normalizes column names to lowercase
func mapToConsentElementMapping(row map[string]interface{}) *model.ConsentElementMapping {
	if row == nil {
		return nil
	}

	mapping := &model.ConsentElementMapping{}

	// Handle string columns (may be string or []byte from MySQL)
	if consentID, ok := row["consent_id"].(string); ok {
		mapping.ConsentID = consentID
	} else if consentID, ok := row["consent_id"].([]byte); ok {
		mapping.ConsentID = string(consentID)
	}

	if elementID, ok := row["element_id"].(string); ok {
		mapping.ElementID = elementID
	} else if elementID, ok := row["element_id"].([]byte); ok {
		mapping.ElementID = string(elementID)
	}

	if orgID, ok := row["org_id"].(string); ok {
		mapping.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		mapping.OrgID = string(orgID)
	}

	if value, ok := row["value"].(string); ok {
		mapping.Value = value
	} else if value, ok := row["value"].([]byte); ok {
		mapping.Value = string(value)
	}

	// Handle boolean columns (may be bool or int64 from MySQL)
	if isUserApproved, ok := row["is_user_approved"].(bool); ok {
		mapping.IsUserApproved = isUserApproved
	} else if isUserApproved, ok := row["is_user_approved"].(int64); ok {
		mapping.IsUserApproved = isUserApproved != 0
	}

	if isMandatory, ok := row["is_mandatory"].(bool); ok {
		mapping.IsMandatory = isMandatory
	} else if isMandatory, ok := row["is_mandatory"].(int64); ok {
		mapping.IsMandatory = isMandatory != 0
	}

	if name, ok := row["name"].(string); ok {
		mapping.Name = name
	} else if name, ok := row["name"].([]byte); ok {
		mapping.Name = string(name)
	}

	return mapping
}

// DeleteMappingsByConsentID deletes all consent element mappings for a consent within a transaction
func (elementStore *store) DeleteMappingsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteMappingsByConsentID, consentID, orgID)
	return err
}
