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

package consentpurpose

import (
	"context"
	"fmt"
	"strings"

	"github.com/wso2/consent-management-api/internal/consentpurpose/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/stores/interfaces"
)

// DBQuery objects for all purpose operations
var (
	QueryCreatePurpose = dbmodel.DBQuery{
		ID:    "CREATE_PURPOSE",
		Query: "INSERT INTO CONSENT_PURPOSE (ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?)",
	}

	QueryGetPurposeByID = dbmodel.DBQuery{
		ID:    "GET_PURPOSE_BY_ID",
		Query: "SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID FROM CONSENT_PURPOSE WHERE ID = ? AND ORG_ID = ?",
	}

	QueryGetPurposeByName = dbmodel.DBQuery{
		ID:    "GET_PURPOSE_BY_NAME",
		Query: "SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID FROM CONSENT_PURPOSE WHERE NAME = ? AND ORG_ID = ?",
	}

	QueryListPurposes = dbmodel.DBQuery{
		ID:    "LIST_PURPOSES",
		Query: "SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID FROM CONSENT_PURPOSE WHERE ORG_ID = ? ORDER BY CREATED_TIME DESC LIMIT ? OFFSET ?",
	}

	QueryCountPurposes = dbmodel.DBQuery{
		ID:    "COUNT_PURPOSES",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE WHERE ORG_ID = ?",
	}

	QueryUpdatePurpose = dbmodel.DBQuery{
		ID:    "UPDATE_PURPOSE",
		Query: "UPDATE CONSENT_PURPOSE SET NAME = ?, DESCRIPTION = ?, UPDATED_TIME = ? WHERE ID = ? AND ORG_ID = ?",
	}

	QueryDeletePurpose = dbmodel.DBQuery{
		ID:    "DELETE_PURPOSE",
		Query: "DELETE FROM CONSENT_PURPOSE WHERE ID = ? AND ORG_ID = ?",
	}

	QueryCheckPurposeNameExists = dbmodel.DBQuery{
		ID:    "CHECK_PURPOSE_NAME_EXISTS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE WHERE NAME = ? AND CLIENT_ID = ? AND ORG_ID = ?",
	}

	QueryCheckPurposeNameExistsExcluding = dbmodel.DBQuery{
		ID:    "CHECK_PURPOSE_NAME_EXISTS_EXCLUDING",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE WHERE NAME = ? AND CLIENT_ID = ? AND ORG_ID = ? AND ID != ?",
	}

	QueryLinkElementToPurpose = dbmodel.DBQuery{
		ID:    "LINK_ELEMENT_TO_PURPOSE",
		Query: "INSERT INTO PURPOSE_ELEMENT_MAPPING (PURPOSE_ID, ELEMENT_ID, IS_MANDATORY, ORG_ID) VALUES (?, ?, ?, ?)",
	}

	QueryGetPurposeElements = dbmodel.DBQuery{
		ID: "GET_PURPOSE_ELEMENTS",
		Query: `SELECT m.ELEMENT_ID, e.NAME as ELEMENT_NAME, m.IS_MANDATORY 
				FROM PURPOSE_ELEMENT_MAPPING m 
				JOIN CONSENT_ELEMENT e ON m.ELEMENT_ID = e.ID AND m.ORG_ID = e.ORG_ID 
				WHERE m.PURPOSE_ID = ? AND m.ORG_ID = ?`,
	}

	QueryDeletePurposeElements = dbmodel.DBQuery{
		ID:    "DELETE_PURPOSE_ELEMENTS",
		Query: "DELETE FROM PURPOSE_ELEMENT_MAPPING WHERE PURPOSE_ID = ? AND ORG_ID = ?",
	}

	QueryGetElementIDByName = dbmodel.DBQuery{
		ID:    "GET_ELEMENT_ID_BY_NAME",
		Query: "SELECT ID FROM CONSENT_ELEMENT WHERE NAME = ? AND ORG_ID = ?",
	}

	queryCheckElementInPurposes = dbmodel.DBQuery{
		ID:    "CHECK_ELEMENT_IN_PURPOSES",
		Query: "SELECT COUNT(*) as count FROM PURPOSE_ELEMENT_MAPPING WHERE ELEMENT_ID = ? AND ORG_ID = ?",
	}
)

// store implements the ConsentPurposeStore interface
type store struct {
}

// NewPurposeStore creates a new purpose store
func NewPurposeStore() interfaces.ConsentPurposeStore {
	return &store{}
}

// getDBClient retrieves the database client from the provider
func (s *store) getDBClient() (provider.DBClientInterface, error) {
	return provider.GetDBProvider().GetConsentDBClient()
}

// CreatePurpose creates a new purpose
func (s *store) CreatePurpose(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error {
	_, err := tx.Exec(QueryCreatePurpose,
		purpose.ID,
		purpose.Name,
		purpose.Description,
		purpose.ClientID,
		purpose.CreatedTime,
		purpose.UpdatedTime,
		purpose.OrgID,
	)
	return err
}

// GetPurposeByID retrieves a purpose by ID with its purposes
func (s *store) GetPurposeByID(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	var purpose model.ConsentPurpose
	rows, err := dbClient.Query(QueryGetPurposeByID, purposeID, orgID)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("purpose not found")
	}

	// Extract values from the first row
	row := rows[0]
	if id, ok := row["id"].([]uint8); ok {
		purpose.ID = string(id)
	}
	if name, ok := row["name"].([]uint8); ok {
		purpose.Name = string(name)
	}
	if desc, ok := row["description"]; ok && desc != nil {
		if descBytes, ok := desc.([]uint8); ok {
			descStr := string(descBytes)
			purpose.Description = &descStr
		}
	}
	if clientID, ok := row["client_id"].([]uint8); ok {
		purpose.ClientID = string(clientID)
	}
	if createdTime, ok := row["created_time"].(int64); ok {
		purpose.CreatedTime = createdTime
	}
	if updatedTime, ok := row["updated_time"].(int64); ok {
		purpose.UpdatedTime = updatedTime
	}
	if orgID, ok := row["org_id"].([]uint8); ok {
		purpose.OrgID = string(orgID)
	}

	// Load elements for the purpose
	elements, err := s.GetPurposeElements(ctx, purposeID, orgID)
	if err != nil {
		return nil, err
	}
	purpose.Elements = elements

	return &purpose, nil
}

// GetPurposeByName retrieves a purpose by name
func (s *store) GetPurposeByName(ctx context.Context, name, orgID string) (*model.ConsentPurpose, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	var purpose model.ConsentPurpose
	rows, err := dbClient.Query(QueryGetPurposeByName, name, orgID)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("purpose not found")
	}

	// Extract values from the first row
	row := rows[0]
	if id, ok := row["id"].([]uint8); ok {
		purpose.ID = string(id)
	}
	if name, ok := row["name"].([]uint8); ok {
		purpose.Name = string(name)
	}
	if desc, ok := row["description"]; ok && desc != nil {
		if descBytes, ok := desc.([]uint8); ok {
			descStr := string(descBytes)
			purpose.Description = &descStr
		}
	}
	if clientID, ok := row["client_id"].([]uint8); ok {
		purpose.ClientID = string(clientID)
	}
	if createdTime, ok := row["created_time"].(int64); ok {
		purpose.CreatedTime = createdTime
	}
	if updatedTime, ok := row["updated_time"].(int64); ok {
		purpose.UpdatedTime = updatedTime
	}
	if orgIDVal, ok := row["org_id"].([]uint8); ok {
		purpose.OrgID = string(orgIDVal)
	}

	// Load elements for the purpose
	elements, err := s.GetPurposeElements(ctx, purpose.ID, orgID)
	if err != nil {
		return nil, err
	}
	purpose.Elements = elements

	return &purpose, nil
}

// ListPurposes retrieves a list of purposes with filtering
func (s *store) ListPurposes(ctx context.Context, orgID, name string, clientIDs []string, purposeNames []string, offset, limit int) ([]model.ConsentPurpose, int, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get database client: %w", err)
	}

	// Build dynamic query based on filters
	query := `SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID 
			  FROM CONSENT_PURPOSE 
			  WHERE ORG_ID = ?`
	countQuery := `SELECT COUNT(*) as count FROM CONSENT_PURPOSE WHERE ORG_ID = ?`

	args := []interface{}{orgID}
	countArgs := []interface{}{orgID}

	// Filter by name (exact match or partial match with LIKE)
	if name != "" {
		query += ` AND NAME = ?`
		countQuery += ` AND NAME = ?`
		args = append(args, name)
		countArgs = append(countArgs, name)
	}

	// Filter by clientIDs
	if len(clientIDs) > 0 {
		placeholders := strings.Repeat("?,", len(clientIDs))
		placeholders = placeholders[:len(placeholders)-1]
		query += ` AND CLIENT_ID IN (` + placeholders + `)`
		countQuery += ` AND CLIENT_ID IN (` + placeholders + `)`
		for _, clientID := range clientIDs {
			args = append(args, clientID)
			countArgs = append(countArgs, clientID)
		}
	}

	// Filter by purposeNames - AND logic: purpose must contain ALL specified purposes
	if len(purposeNames) > 0 {
		// For each purpose name, ensure the purpose contains it
		for _, purposeName := range purposeNames {
			subQuery := ` AND EXISTS (
				SELECT 1 FROM PURPOSE_ELEMENT_MAPPING m
				JOIN CONSENT_ELEMENT p ON m.ELEMENT_ID = p.ID AND m.ORG_ID = p.ORG_ID
				WHERE m.PURPOSE_ID = CONSENT_PURPOSE.ID 
				  AND m.ORG_ID = CONSENT_PURPOSE.ORG_ID
				  AND p.NAME = ?
			)`
			query += subQuery
			countQuery += subQuery
			args = append(args, purposeName)
			countArgs = append(countArgs, purposeName)
		}
	}

	// Get total count
	var total int
	countQueryDB := dbmodel.DBQuery{
		ID:    "COUNT_FILTERED_PURPOSES",
		Query: countQuery,
	}
	rows, err := dbClient.Query(countQueryDB, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	if len(rows) > 0 {
		if count, ok := rows[0]["count"].(int64); ok {
			total = int(count)
		}
	}

	// Add sorting and pagination
	query += ` ORDER BY CREATED_TIME DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	// Execute query
	var purposes []model.ConsentPurpose
	listQueryDB := dbmodel.DBQuery{
		ID:    "LIST_FILTERED_PURPOSES",
		Query: query,
	}
	rows, err = dbClient.Query(listQueryDB, args...)
	if err != nil {
		return nil, 0, err
	}

	for _, row := range rows {
		var purpose model.ConsentPurpose
		if id, ok := row["id"].([]uint8); ok {
			purpose.ID = string(id)
		}
		if name, ok := row["name"].([]uint8); ok {
			purpose.Name = string(name)
		}
		if desc, ok := row["description"]; ok && desc != nil {
			if descBytes, ok := desc.([]uint8); ok {
				descStr := string(descBytes)
				purpose.Description = &descStr
			}
		}
		if clientID, ok := row["client_id"].([]uint8); ok {
			purpose.ClientID = string(clientID)
		}
		if createdTime, ok := row["created_time"].(int64); ok {
			purpose.CreatedTime = createdTime
		}
		if updatedTime, ok := row["updated_time"].(int64); ok {
			purpose.UpdatedTime = updatedTime
		}
		if orgID, ok := row["org_id"].([]uint8); ok {
			purpose.OrgID = string(orgID)
		}
		purposes = append(purposes, purpose)
	}

	// Load elements for each purpose
	for i := range purposes {
		elements, err := s.GetPurposeElements(ctx, purposes[i].ID, orgID)
		if err != nil {
			return nil, 0, err
		}
		purposes[i].Elements = elements
	}

	return purposes, total, nil
}

// UpdatePurpose updates an existing purpose
func (s *store) UpdatePurpose(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error {
	_, err := tx.Exec(QueryUpdatePurpose,
		purpose.Name,
		purpose.Description,
		purpose.UpdatedTime,
		purpose.ID,
		purpose.OrgID,
	)
	return err
}

// DeletePurpose deletes a purpose
func (s *store) DeletePurpose(tx dbmodel.TxInterface, purposeID, orgID string) error {
	_, err := tx.Exec(QueryDeletePurpose, purposeID, orgID)
	return err
}

// CheckPurposeNameExists checks if a purpose name exists for a client
func (s *store) CheckPurposeNameExists(ctx context.Context, name, clientID, orgID string, excludePurposeID *string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}

	var count int
	var rows []map[string]interface{}

	if excludePurposeID != nil {
		rows, err = dbClient.Query(QueryCheckPurposeNameExistsExcluding, name, clientID, orgID, *excludePurposeID)
	} else {
		rows, err = dbClient.Query(QueryCheckPurposeNameExists, name, clientID, orgID)
	}

	if err != nil {
		return false, err
	}

	if len(rows) > 0 {
		if countVal, ok := rows[0]["count"].(int64); ok {
			count = int(countVal)
		}
	}

	return count > 0, nil
}

// LinkElementToPurpose links an element to a purpose
func (s *store) LinkElementToPurpose(tx dbmodel.TxInterface, purposeID, elementID, orgID string, isMandatory bool) error {
	_, err := tx.Exec(QueryLinkElementToPurpose,
		purposeID,
		elementID,
		isMandatory,
		orgID,
	)
	return err
}

// GetPurposeElements retrieves all elements for a purpose
func (s *store) GetPurposeElements(ctx context.Context, purposeID, orgID string) ([]model.PurposeElement, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetPurposeElements, purposeID, orgID)
	if err != nil {
		return nil, err
	}

	var purposes []model.PurposeElement
	for _, row := range rows {
		var p model.PurposeElement
		if elementID, ok := row["element_id"].([]uint8); ok {
			p.ElementID = string(elementID)
		}
		if elementName, ok := row["element_name"].([]uint8); ok {
			p.ElementName = string(elementName)
		}
		if isMandatory, ok := row["is_mandatory"].(int64); ok {
			p.IsMandatory = isMandatory != 0
		}
		purposes = append(purposes, p)
	}
	return purposes, nil
}

// DeletePurposeElements deletes all element mappings for a purpose
func (s *store) DeletePurposeElements(tx dbmodel.TxInterface, purposeID, orgID string) error {
	_, err := tx.Exec(QueryDeletePurposeElements, purposeID, orgID)
	return err
}

// GetPurposeIDByName retrieves a purpose ID by name
func (s *store) GetPurposeIDByName(ctx context.Context, purposeName, orgID string) (string, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return "", fmt.Errorf("failed to get database client: %w", err)
	}

	var purposeID string
	rows, err := dbClient.Query(QueryGetElementIDByName, purposeName, orgID)
	if err != nil {
		return "", err
	}

	if len(rows) == 0 {
		return "", fmt.Errorf("purpose '%s' not found", purposeName)
	}

	if id, ok := rows[0]["id"].([]uint8); ok {
		purposeID = string(id)
	}
	return purposeID, nil
}

// ValidatePurposeNames validates that all purpose names exist and returns a map of name -> ID
func (s *store) ValidatePurposeNames(ctx context.Context, purposeNames []string, orgID string) (map[string]string, error) {
	if len(purposeNames) == 0 {
		return map[string]string{}, nil
	}

	placeholders := strings.Repeat("?,", len(purposeNames))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf("SELECT NAME, ID FROM CONSENT_PURPOSE WHERE ORG_ID = ? AND NAME IN (%s)", placeholders)

	args := []interface{}{orgID}
	for _, name := range purposeNames {
		args = append(args, name)
	}

	validateQueryDB := dbmodel.DBQuery{
		ID:    "VALIDATE_PURPOSE_NAMES",
		Query: query,
	}

	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(validateQueryDB, args...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, row := range rows {
		var name, id string
		if nameBytes, ok := row["name"].([]uint8); ok {
			name = string(nameBytes)
		}
		if idBytes, ok := row["id"].([]uint8); ok {
			id = string(idBytes)
		}
		result[name] = id
	}

	return result, nil
}

// IsElementUsedInPurposes checks if a purpose is used in any purpose
func (s *store) IsElementUsedInPurposes(ctx context.Context, purposeID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(queryCheckElementInPurposes, purposeID, orgID)
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
