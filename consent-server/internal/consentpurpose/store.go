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
	"errors"
	"fmt"
	"strings"

	"github.com/wso2/openfgc/internal/consentpurpose/model"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/database/provider"
	"github.com/wso2/openfgc/internal/system/stores/interfaces"
)

// Sentinel errors
var (
	ErrPurposeNotFound = errors.New("purpose not found")
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

	queryCheckElementInPurposes = dbmodel.DBQuery{
		ID:    "CHECK_ELEMENT_IN_PURPOSES",
		Query: "SELECT COUNT(*) as count FROM PURPOSE_ELEMENT_MAPPING WHERE ELEMENT_ID = ? AND ORG_ID = ?",
	}

	// Base queries for list purposes (used for dynamic query building)
	BaseListPurposesQuery = `SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID 
			  FROM CONSENT_PURPOSE 
			  WHERE ORG_ID = ?`

	BaseCountPurposesQuery = `SELECT COUNT(*) as count FROM CONSENT_PURPOSE WHERE ORG_ID = ?`
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

// mapRowToPurpose maps a database row to a ConsentPurpose model
func (s *store) mapRowToPurpose(row map[string]interface{}) model.ConsentPurpose {
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

	return purpose
}

// buildListPurposesQuery builds dynamic query with filters for listing purposes
func (s *store) buildListPurposesQuery(orgID, name string, clientIDs []string, elementNames []string) (string, string, []interface{}, []interface{}) {
	baseQuery := BaseListPurposesQuery
	countQuery := BaseCountPurposesQuery

	args := []interface{}{orgID}
	countArgs := []interface{}{orgID}

	// Filter by name (partial match using LIKE)
	if name != "" {
		// Escape SQL wildcard characters and backslashes to prevent unintended matches
		escaper := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
		escapedName := escaper.Replace(name)
		// Add wildcards for partial match (collation determines case sensitivity)
		namePattern := "%" + escapedName + "%"

		baseQuery += ` AND NAME LIKE ? ESCAPE '\\'`
		countQuery += ` AND NAME LIKE ? ESCAPE '\\'`
		args = append(args, namePattern)
		countArgs = append(countArgs, namePattern)
	}

	// Filter by clientIDs
	if len(clientIDs) > 0 {
		placeholders := strings.Repeat("?,", len(clientIDs))
		placeholders = placeholders[:len(placeholders)-1]
		baseQuery += ` AND CLIENT_ID IN (` + placeholders + `)`
		countQuery += ` AND CLIENT_ID IN (` + placeholders + `)`
		for _, clientID := range clientIDs {
			args = append(args, clientID)
			countArgs = append(countArgs, clientID)
		}
	}

	// Filter by elementNames - AND logic: purpose must contain ALL specified elements
	if len(elementNames) > 0 {
		for _, elementName := range elementNames {
			subQuery := ` AND EXISTS (
				SELECT 1 FROM PURPOSE_ELEMENT_MAPPING pem
				JOIN CONSENT_ELEMENT ce ON pem.ELEMENT_ID = ce.ID AND pem.ORG_ID = ce.ORG_ID
				WHERE pem.PURPOSE_ID = CONSENT_PURPOSE.ID 
				  AND pem.ORG_ID = CONSENT_PURPOSE.ORG_ID
				  AND ce.NAME = ?
			)`
			baseQuery += subQuery
			countQuery += subQuery
			args = append(args, elementName)
			countArgs = append(countArgs, elementName)
		}
	}

	return baseQuery, countQuery, args, countArgs
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
		return nil, ErrPurposeNotFound
	}

	// Map database row to purpose model
	purpose = s.mapRowToPurpose(rows[0])

	// Load elements for the purpose
	elements, err := s.GetPurposeElements(ctx, purposeID, purpose.OrgID)
	if err != nil {
		return nil, err
	}
	purpose.Elements = elements

	return &purpose, nil
}

// ListPurposes retrieves a list of purposes with filtering
func (s *store) ListPurposes(ctx context.Context, orgID, name string, clientIDs []string, elementNames []string, offset, limit int) ([]model.ConsentPurpose, int, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get database client: %w", err)
	}

	// Build dynamic query with filters
	query, countQuery, args, countArgs := s.buildListPurposesQuery(orgID, name, clientIDs, elementNames)

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
		purpose := s.mapRowToPurpose(row)
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

// IsElementUsedInPurposes checks if a element is used in any purpose
func (s *store) IsElementUsedInPurposes(ctx context.Context, elementID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(queryCheckElementInPurposes, elementID, orgID)
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
