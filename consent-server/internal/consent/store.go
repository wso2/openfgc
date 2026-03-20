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

package consent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/wso2/openfgc/internal/consent/model"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/database/provider"
	dbutils "github.com/wso2/openfgc/internal/system/database/utils"
	"github.com/wso2/openfgc/internal/system/stores/interfaces"
)

// DBQuery objects for consent operations
var (
	QueryCreateConsent = dbmodel.DBQuery{
		ID:            "CREATE_CONSENT",
		Query:         "INSERT INTO CONSENT (CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT (CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
	}

	QueryGetConsentByID = dbmodel.DBQuery{
		ID:            "GET_CONSENT_BY_ID",
		Query:         "SELECT CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID FROM CONSENT WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID FROM CONSENT WHERE CONSENT_ID = $1 AND ORG_ID = $2",
	}

	QueryUpdateConsent = dbmodel.DBQuery{
		ID:            "UPDATE_CONSENT",
		Query:         "UPDATE CONSENT SET UPDATED_TIME = ?, CONSENT_TYPE = ?, CONSENT_FREQUENCY = ?, VALIDITY_TIME = ?, RECURRING_INDICATOR = ?, DATA_ACCESS_VALIDITY_DURATION = ? WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "UPDATE CONSENT SET UPDATED_TIME = $1, CONSENT_TYPE = $2, CONSENT_FREQUENCY = $3, VALIDITY_TIME = $4, RECURRING_INDICATOR = $5, DATA_ACCESS_VALIDITY_DURATION = $6 WHERE CONSENT_ID = $7 AND ORG_ID = $8",
	}

	QueryUpdateConsentStatus = dbmodel.DBQuery{
		ID:            "UPDATE_CONSENT_STATUS",
		Query:         "UPDATE CONSENT SET CURRENT_STATUS = ?, UPDATED_TIME = ? WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "UPDATE CONSENT SET CURRENT_STATUS = $1, UPDATED_TIME = $2 WHERE CONSENT_ID = $3 AND ORG_ID = $4",
	}

	// Attribute queries
	QueryCreateAttribute = dbmodel.DBQuery{
		ID:            "CREATE_CONSENT_ATTRIBUTE",
		Query:         "INSERT INTO CONSENT_ATTRIBUTE (CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES (?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT_ATTRIBUTE (CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES ($1, $2, $3, $4)",
	}

	QueryGetAttributesByConsentID = dbmodel.DBQuery{
		ID:            "GET_ATTRIBUTES_BY_CONSENT_ID",
		Query:         "SELECT CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = $1 AND ORG_ID = $2",
	}

	QueryDeleteAttributesByConsentID = dbmodel.DBQuery{
		ID:            "DELETE_ATTRIBUTES_BY_CONSENT_ID",
		Query:         "DELETE FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = $1 AND ORG_ID = $2",
	}

	QueryFindConsentIDsByAttributeKey = dbmodel.DBQuery{
		ID:            "FIND_CONSENT_IDS_BY_ATTRIBUTE_KEY",
		Query:         "SELECT DISTINCT CONSENT_ID FROM CONSENT_ATTRIBUTE WHERE ATT_KEY = ? AND ORG_ID = ? ORDER BY CONSENT_ID",
		PostgresQuery: "SELECT DISTINCT CONSENT_ID FROM CONSENT_ATTRIBUTE WHERE ATT_KEY = $1 AND ORG_ID = $2 ORDER BY CONSENT_ID",
	}

	QueryFindConsentIDsByAttribute = dbmodel.DBQuery{
		ID:            "FIND_CONSENT_IDS_BY_ATTRIBUTE",
		Query:         "SELECT DISTINCT CONSENT_ID FROM CONSENT_ATTRIBUTE WHERE ATT_KEY = ? AND ATT_VALUE = ? AND ORG_ID = ? ORDER BY CONSENT_ID",
		PostgresQuery: "SELECT DISTINCT CONSENT_ID FROM CONSENT_ATTRIBUTE WHERE ATT_KEY = $1 AND ATT_VALUE = $2 AND ORG_ID = $3 ORDER BY CONSENT_ID",
	}

	// Status audit queries
	QueryCreateStatusAudit = dbmodel.DBQuery{
		ID:            "CREATE_STATUS_AUDIT",
		Query:         "INSERT INTO CONSENT_STATUS_AUDIT (STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME, REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT_STATUS_AUDIT (STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME, REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
	}

	QueryGetAttributesByConsentIDs = dbmodel.DBQuery{
		ID:    "GET_ATTRIBUTES_BY_CONSENT_IDS",
		Query: "", // Built dynamically
	}

	// Purpose Consent queries
	QueryCreateConsentPurposeMapping = dbmodel.DBQuery{
		ID:            "CREATE_CONSENT_PURPOSE_MAPPING",
		Query:         "INSERT INTO PURPOSE_CONSENT_MAPPING (CONSENT_ID, PURPOSE_ID, ORG_ID) VALUES (?, ?, ?)",
		PostgresQuery: "INSERT INTO PURPOSE_CONSENT_MAPPING (CONSENT_ID, PURPOSE_ID, ORG_ID) VALUES ($1, $2, $3)",
	}

	QueryGetConsentPurposesByConsentID = dbmodel.DBQuery{
		ID: "GET_PURPOSES_BY_CONSENT_ID",
		Query: `
			SELECT 
				pgc.CONSENT_ID,
				pgc.PURPOSE_ID,
				pg.NAME as PURPOSE_NAME
			FROM PURPOSE_CONSENT_MAPPING pgc
			JOIN CONSENT_PURPOSE pg ON pgc.PURPOSE_ID = pg.ID AND pgc.ORG_ID = pg.ORG_ID
			WHERE pgc.CONSENT_ID = ? AND pgc.ORG_ID = ?
			ORDER BY pg.NAME
		`,
		PostgresQuery: `
			SELECT 
				pgc.CONSENT_ID,
				pgc.PURPOSE_ID,
				pg.NAME as PURPOSE_NAME
			FROM PURPOSE_CONSENT_MAPPING pgc
			JOIN CONSENT_PURPOSE pg ON pgc.PURPOSE_ID = pg.ID AND pgc.ORG_ID = pg.ORG_ID
			WHERE pgc.CONSENT_ID = $1 AND pgc.ORG_ID = $2
			ORDER BY pg.NAME
		`,
	}

	QueryCheckPurposeUsedInConsents = dbmodel.DBQuery{
		ID:            "CHECK_PURPOSE_USED_IN_CONSENTS",
		Query:         "SELECT COUNT(*) as count FROM PURPOSE_CONSENT_MAPPING WHERE PURPOSE_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT COUNT(*) as count FROM PURPOSE_CONSENT_MAPPING WHERE PURPOSE_ID = $1 AND ORG_ID = $2",
	}

	QueryCreateElementApproval = dbmodel.DBQuery{
		ID:            "CREATE_ELEMENT_APPROVAL",
		Query:         "INSERT INTO CONSENT_ELEMENT_APPROVAL (CONSENT_ID, PURPOSE_ID, ELEMENT_ID, IS_USER_APPROVED, VALUE, ORG_ID) VALUES (?, ?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT_ELEMENT_APPROVAL (CONSENT_ID, PURPOSE_ID, ELEMENT_ID, IS_USER_APPROVED, VALUE, ORG_ID) VALUES ($1, $2, $3, $4, $5, $6)",
	}

	QueryGetElementApprovalsByConsentID = dbmodel.DBQuery{
		ID: "GET_ELEMENT_APPROVALS_BY_CONSENT_ID",
		Query: `
			SELECT 
				pa.CONSENT_ID,
				pa.PURPOSE_ID,
				pg.NAME as PURPOSE_NAME,
				pa.ELEMENT_ID,
				p.NAME as ELEMENT_NAME,
				pa.IS_USER_APPROVED,
				pa.VALUE,
				gm.IS_MANDATORY
			FROM CONSENT_ELEMENT_APPROVAL pa
		JOIN CONSENT_ELEMENT p ON pa.ELEMENT_ID = p.ID AND pa.ORG_ID = p.ORG_ID
		JOIN CONSENT_PURPOSE pg ON pa.PURPOSE_ID = pg.ID AND pa.ORG_ID = pg.ORG_ID
		JOIN PURPOSE_ELEMENT_MAPPING gm ON pa.PURPOSE_ID = gm.PURPOSE_ID 
			AND pa.ELEMENT_ID = gm.ELEMENT_ID AND pa.ORG_ID = gm.ORG_ID
			WHERE pa.CONSENT_ID = ? AND pa.ORG_ID = ?
			ORDER BY pg.NAME, p.NAME
		`,
		PostgresQuery: `
			SELECT 
				pa.CONSENT_ID,
				pa.PURPOSE_ID,
				pg.NAME as PURPOSE_NAME,
				pa.ELEMENT_ID,
				p.NAME as ELEMENT_NAME,
				pa.IS_USER_APPROVED,
				pa.VALUE,
				gm.IS_MANDATORY
			FROM CONSENT_ELEMENT_APPROVAL pa
		JOIN CONSENT_ELEMENT p ON pa.ELEMENT_ID = p.ID AND pa.ORG_ID = p.ORG_ID
		JOIN CONSENT_PURPOSE pg ON pa.PURPOSE_ID = pg.ID AND pa.ORG_ID = pg.ORG_ID
		JOIN PURPOSE_ELEMENT_MAPPING gm ON pa.PURPOSE_ID = gm.PURPOSE_ID 
			AND pa.ELEMENT_ID = gm.ELEMENT_ID AND pa.ORG_ID = gm.ORG_ID
			WHERE pa.CONSENT_ID = $1 AND pa.ORG_ID = $2
			ORDER BY pg.NAME, p.NAME
		`,
	}

	QueryDeleteConsentPurposesByConsentID = dbmodel.DBQuery{
		ID:            "DELETE_PURPOSES_BY_CONSENT_ID",
		Query:         "DELETE FROM PURPOSE_CONSENT_MAPPING WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM PURPOSE_CONSENT_MAPPING WHERE CONSENT_ID = $1 AND ORG_ID = $2",
	}

	QueryDeleteElementApprovalsByConsentID = dbmodel.DBQuery{
		ID:            "DELETE_ELEMENT_APPROVALS_BY_CONSENT_ID",
		Query:         "DELETE FROM CONSENT_ELEMENT_APPROVAL WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM CONSENT_ELEMENT_APPROVAL WHERE CONSENT_ID = $1 AND ORG_ID = $2",
	}
)

// store implements the interfaces.ConsentStore interface
type store struct {
}

// NewConsentStore creates a new consent store
func NewConsentStore() interfaces.ConsentStore {
	return &store{}
}

// getDBClient retrieves the database client from the provider
func (s *store) getDBClient() (provider.DBClientInterface, error) {
	return provider.GetDBProvider().GetConsentDBClient()
}

// Create a new consent within a transaction
func (s *store) Create(tx dbmodel.TxInterface, consent *model.Consent) error {
	_, err := tx.Exec(QueryCreateConsent,
		consent.ConsentID, consent.CreatedTime, consent.UpdatedTime, consent.ClientID,
		consent.ConsentType, consent.CurrentStatus, consent.ConsentFrequency,
		consent.ValidityTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
		consent.OrgID)
	return err
}

// GetByID retrieves a consent by ID
func (s *store) GetByID(ctx context.Context, consentID, orgID string) (*model.Consent, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetConsentByID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return mapToConsent(rows[0]), nil
}

// Search retrieves consents based on filters with pagination
func (s *store) Search(ctx context.Context, filters model.ConsentSearchFilters) ([]model.Consent, int, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get database client: %w", err)
	}

	// Build WHERE clause dynamically
	whereConditions := []string{"CONSENT.ORG_ID = ?"}
	args := []interface{}{filters.OrgID}
	countArgs := []interface{}{filters.OrgID}

	// Add consentTypes filter (IN clause)
	if len(filters.ConsentTypes) > 0 {
		placeholders := make([]string, len(filters.ConsentTypes))
		for i, ct := range filters.ConsentTypes {
			placeholders[i] = "?"
			args = append(args, ct)
			countArgs = append(countArgs, ct)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CONSENT_TYPE IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add consentStatuses filter (IN clause) - convert to uppercase
	if len(filters.ConsentStatuses) > 0 {
		placeholders := make([]string, len(filters.ConsentStatuses))
		for i, status := range filters.ConsentStatuses {
			placeholders[i] = "?"
			// Convert to uppercase to match DB values (ACTIVE, REJECTED, etc.)
			args = append(args, strings.ToUpper(status))
			countArgs = append(countArgs, strings.ToUpper(status))
		}
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CURRENT_STATUS IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add clientIds filter (IN clause)
	if len(filters.ClientIDs) > 0 {
		placeholders := make([]string, len(filters.ClientIDs))
		for i, clientID := range filters.ClientIDs {
			placeholders[i] = "?"
			args = append(args, clientID)
			countArgs = append(countArgs, clientID)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CLIENT_ID IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add userIds filter (via JOIN with CONSENT_AUTH_RESOURCE)
	joinClause := ""
	if len(filters.UserIDs) > 0 {
		placeholders := make([]string, len(filters.UserIDs))
		for i, userID := range filters.UserIDs {
			placeholders[i] = "?"
			args = append(args, userID)
			countArgs = append(countArgs, userID)
		}
		joinClause = " INNER JOIN CONSENT_AUTH_RESOURCE car ON CONSENT.CONSENT_ID = car.CONSENT_ID AND CONSENT.ORG_ID = car.ORG_ID"
		whereConditions = append(whereConditions, fmt.Sprintf("car.USER_ID IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add purposeNames filter (via JOIN with PURPOSE_CONSENT_MAPPING and CONSENT_PURPOSE)
	if len(filters.PurposeNames) > 0 {
		placeholders := make([]string, len(filters.PurposeNames))
		for i, purposeName := range filters.PurposeNames {
			placeholders[i] = "?"
			args = append(args, purposeName)
			countArgs = append(countArgs, purposeName)
		}
		joinClause += " INNER JOIN PURPOSE_CONSENT_MAPPING pcm ON CONSENT.CONSENT_ID = pcm.CONSENT_ID AND CONSENT.ORG_ID = pcm.ORG_ID"
		joinClause += " INNER JOIN CONSENT_PURPOSE cp ON pcm.PURPOSE_ID = cp.ID AND pcm.ORG_ID = cp.ORG_ID"
		whereConditions = append(whereConditions, fmt.Sprintf("cp.NAME IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add time range filters (timestamps in milliseconds) - filter by UPDATED_TIME
	if filters.FromTime != nil {
		whereConditions = append(whereConditions, "CONSENT.UPDATED_TIME >= ?")
		args = append(args, *filters.FromTime)
		countArgs = append(countArgs, *filters.FromTime)
	}

	if filters.ToTime != nil {
		whereConditions = append(whereConditions, "CONSENT.UPDATED_TIME <= ?")
		args = append(args, *filters.ToTime)
		countArgs = append(countArgs, *filters.ToTime)
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Build COUNT query
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT CONSENT.CONSENT_ID) as count FROM CONSENT%s WHERE %s",
		joinClause, whereClause)

	// Execute count query
	countRows, err := dbClient.Query(dbmodel.DBQuery{
		ID:            "COUNT_SEARCH_RESULTS",
		Query:         countQuery,
		PostgresQuery: dbutils.ConvertToPostgresParams(countQuery),
	}, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	totalCount := 0
	if len(countRows) > 0 {
		if count, ok := countRows[0]["count"].(int64); ok {
			totalCount = int(count)
		} else if countVal, ok := countRows[0]["count"].([]uint8); ok {
			// MySQL may return count as []uint8
			if parsedCount, parseErr := strconv.ParseInt(string(countVal), 10, 64); parseErr == nil {
				totalCount = int(parsedCount)
			}
		}
	}

	// Build SELECT query with DISTINCT to handle JOIN duplicates
	selectQuery := fmt.Sprintf(
		"SELECT DISTINCT CONSENT.CONSENT_ID, CONSENT.CREATED_TIME, CONSENT.UPDATED_TIME, CONSENT.CLIENT_ID, CONSENT.CONSENT_TYPE, CONSENT.CURRENT_STATUS, CONSENT.CONSENT_FREQUENCY, CONSENT.VALIDITY_TIME, CONSENT.RECURRING_INDICATOR, CONSENT.DATA_ACCESS_VALIDITY_DURATION, CONSENT.ORG_ID FROM CONSENT%s WHERE %s ORDER BY CONSENT.CREATED_TIME DESC LIMIT ? OFFSET ?",
		joinClause,
		whereClause,
	)

	// Add pagination parameters
	args = append(args, filters.Limit, filters.Offset)

	// Execute search query
	rows, err := dbClient.Query(dbmodel.DBQuery{
		ID:            "SEARCH_CONSENTS",
		Query:         selectQuery,
		PostgresQuery: dbutils.ConvertToPostgresParams(selectQuery),
	}, args...)
	if err != nil {
		return nil, 0, err
	}

	consents := make([]model.Consent, 0, len(rows))
	for _, row := range rows {
		consent := mapToConsent(row)
		if consent != nil {
			consents = append(consents, *consent)
		}
	}

	return consents, totalCount, nil
}

// Update updates a consent within a transaction
func (s *store) Update(tx dbmodel.TxInterface, consent *model.Consent) error {
	_, err := tx.Exec(QueryUpdateConsent,
		consent.UpdatedTime, consent.ConsentType, consent.ConsentFrequency,
		consent.ValidityTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
		consent.ConsentID, consent.OrgID)
	return err
}

// UpdateStatus updates consent status within a transaction
func (s *store) UpdateStatus(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error {
	result, err := tx.Exec(QueryUpdateConsentStatus, status, updatedTime, consentID, orgID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no consent found with CONSENT_ID=%s and ORG_ID=%s", consentID, orgID)
	}

	return nil
}

// CreateAttributes creates multiple consent attributes within a transaction
func (s *store) CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentAttribute) error {
	for _, attribute := range attributes {
		_, err := tx.Exec(QueryCreateAttribute,
			attribute.ConsentID, attribute.AttKey, attribute.AttValue, attribute.OrgID)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAttributesByConsentID retrieves attributes for a consent
func (s *store) GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentAttribute, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetAttributesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	attributes := make([]model.ConsentAttribute, 0, len(rows))
	for _, row := range rows {
		attribute := mapToConsentAttribute(row)
		if attribute != nil {
			attributes = append(attributes, *attribute)
		}
	}

	return attributes, nil
}

// GetAttributesByConsentIDs retrieves attributes for multiple consents, grouped by consent ID
func (s *store) GetAttributesByConsentIDs(ctx context.Context, consentIDs []string, orgID string) (map[string]map[string]string, error) {
	if len(consentIDs) == 0 {
		return make(map[string]map[string]string), nil
	}

	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
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
	mysqlQuery := fmt.Sprintf("SELECT CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID IN (%s) AND ORG_ID = ?", placeholders)
	query := dbmodel.DBQuery{
		ID:            QueryGetAttributesByConsentIDs.ID,
		Query:         mysqlQuery,
		PostgresQuery: dbutils.ConvertToPostgresParams(mysqlQuery),
	}

	rows, err := dbClient.Query(query, args...)
	if err != nil {
		return nil, err
	}

	// Group attributes by consent ID
	result := make(map[string]map[string]string)
	for _, row := range rows {
		attribute := mapToConsentAttribute(row)
		if attribute != nil {
			if result[attribute.ConsentID] == nil {
				result[attribute.ConsentID] = make(map[string]string)
			}
			result[attribute.ConsentID][attribute.AttKey] = attribute.AttValue
		}
	}

	return result, nil
}

// DeleteAttributesByConsentID deletes all attributes for a consent within a transaction
func (s *store) DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteAttributesByConsentID, consentID, orgID)
	return err
}

// FindConsentIDsByAttributeKey finds all consent IDs that have a specific attribute key
func (s *store) FindConsentIDsByAttributeKey(ctx context.Context, key, orgID string) ([]string, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryFindConsentIDsByAttributeKey, key, orgID)
	if err != nil {
		return nil, err
	}

	consentIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if consentID := getString(row, "consent_id"); consentID != "" {
			consentIDs = append(consentIDs, consentID)
		}
	}

	return consentIDs, nil
}

// FindConsentIDsByAttribute finds all consent IDs that have a specific attribute key-value pair
func (s *store) FindConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryFindConsentIDsByAttribute, key, value, orgID)
	if err != nil {
		return nil, err
	}

	consentIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if consentID := getString(row, "consent_id"); consentID != "" {
			consentIDs = append(consentIDs, consentID)
		}
	}

	return consentIDs, nil
}

// CreateStatusAudit creates a status audit entry within a transaction
func (s *store) CreateStatusAudit(tx dbmodel.TxInterface, audit *model.ConsentStatusAudit) error {
	_, err := tx.Exec(QueryCreateStatusAudit,
		audit.StatusAuditID, audit.ConsentID, audit.CurrentStatus, audit.ActionTime,
		audit.Reason, audit.ActionBy, audit.PreviousStatus, audit.OrgID)
	return err
}

// Mapper functions

// mapToConsent converts a database row map to Consent
// Note: DBClient normalizes column names to lowercase
func mapToConsent(row map[string]interface{}) *model.Consent {
	if row == nil {
		return nil
	}

	return &model.Consent{
		ConsentID:                  getString(row, "consent_id"),
		CreatedTime:                getInt64(row, "created_time"),
		UpdatedTime:                getInt64(row, "updated_time"),
		ClientID:                   getString(row, "client_id"),
		ConsentType:                getString(row, "consent_type"),
		CurrentStatus:              getString(row, "current_status"),
		ConsentFrequency:           getIntPointer(row, "consent_frequency"),
		ValidityTime:               getInt64Pointer(row, "validity_time"),
		RecurringIndicator:         getBoolPointer(row, "recurring_indicator"),
		DataAccessValidityDuration: getInt64Pointer(row, "data_access_validity_duration"),
		OrgID:                      getString(row, "org_id"),
	}
}

// mapToConsentAttribute converts a database row map to ConsentAttribute
// Note: DBClient normalizes column names to lowercase
func mapToConsentAttribute(row map[string]interface{}) *model.ConsentAttribute {
	if row == nil {
		return nil
	}

	return &model.ConsentAttribute{
		ConsentID: getString(row, "consent_id"),
		AttKey:    getString(row, "att_key"),
		AttValue:  getString(row, "att_value"),
		OrgID:     getString(row, "org_id"),
	}
}

// CreateConsentPurposeMapping links a consent to a purpose
func (s *store) CreateConsentPurposeMapping(tx dbmodel.TxInterface, consentID, purposeID, orgID string) error {
	_, err := tx.Exec(QueryCreateConsentPurposeMapping, consentID, purposeID, orgID)
	return err
}

// CheckPurposeUsedInConsents checks if a purpose is used in any consents
func (s *store) CheckPurposeUsedInConsents(ctx context.Context, purposeID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryCheckPurposeUsedInConsents, purposeID, orgID)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}

	count := int64(0)
	if countVal, ok := rows[0]["count"].(int64); ok {
		count = countVal
	} else if countVal, ok := rows[0]["count"].([]uint8); ok {
		// MySQL may return count as []uint8
		if parsedCount, parseErr := strconv.ParseInt(string(countVal), 10, 64); parseErr == nil {
			count = parsedCount
		}
	}

	return count > 0, nil
}

// GetConsentPurposeMappingsByConsentID retrieves all purpose mappings for a consent
func (s *store) GetConsentPurposeMappingsByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentPurposeMapping, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetConsentPurposesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	mappings := make([]model.ConsentPurposeMapping, 0)
	for _, row := range rows {
		mapping := model.ConsentPurposeMapping{
			ConsentID:   getString(row, "consent_id"),
			PurposeID:   getString(row, "purpose_id"),
			PurposeName: getString(row, "purpose_name"),
		}
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// CreatePurposeElementApproval creates a purpose approval record
func (s *store) CreatePurposeElementApproval(tx dbmodel.TxInterface, approval *model.ConsentElementApprovalRecord) error {
	_, err := tx.Exec(QueryCreateElementApproval,
		approval.ConsentID,
		approval.PurposeID,
		approval.ElementID,
		approval.IsUserApproved,
		approval.Value, // JSON string or nil
		approval.OrgID,
	)
	return err
}

// GetPurposeElementApprovalsByConsentID retrieves all purpose approvals for a consent, grouped by purpose
func (s *store) GetPurposeElementApprovalsByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentElementApprovalRecord, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	rows, err := dbClient.Query(QueryGetElementApprovalsByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	approvals := make([]model.ConsentElementApprovalRecord, 0)
	for _, row := range rows {
		approval := model.ConsentElementApprovalRecord{
			ConsentID:      getString(row, "consent_id"),
			PurposeID:      getString(row, "purpose_id"),
			PurposeName:    getString(row, "purpose_name"),
			ElementID:      getString(row, "element_id"),
			ElementName:    getString(row, "element_name"),
			IsUserApproved: getBool(row, "is_user_approved"),
			IsMandatory:    getBool(row, "is_mandatory"),
			Value:          getStringPointer(row, "value"),
			OrgID:          orgID,
		}
		approvals = append(approvals, approval)
	}

	return approvals, nil
}

// DeleteConsentPurposeMappingsByConsentID deletes all purpose mappings for a consent
func (s *store) DeleteConsentPurposeMappingsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteConsentPurposesByConsentID, consentID, orgID)
	return err
}

// DeletePurposeElementApprovalsByConsentID deletes all purpose approval records for a consent
func (s *store) DeletePurposeElementApprovalsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteElementApprovalsByConsentID, consentID, orgID)
	return err
}

// Helper functions for type conversion
func getString(row map[string]interface{}, key string) string {
	if val, ok := row[key].(string); ok {
		return val
	}
	if val, ok := row[key].([]byte); ok {
		return string(val)
	}
	return ""
}

// getInt64 safely extracts an int64 value from a database row map
// Handles various types returned by different DB drivers
func getInt64(row map[string]interface{}, key string) int64 {
	val := row[key]
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case []uint8: // byte slice
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

func getBool(row map[string]interface{}, key string) bool {
	if val, ok := row[key].(bool); ok {
		return val
	}
	if val, ok := row[key].(int64); ok {
		return val != 0
	}
	if val, ok := row[key].(uint8); ok {
		return val != 0
	}
	return false
}

func getBoolPointer(row map[string]interface{}, key string) *bool {
	if val, ok := row[key].(bool); ok {
		return &val
	}
	if val, ok := row[key].(int64); ok {
		result := val != 0
		return &result
	}
	if val, ok := row[key].(uint8); ok {
		result := val != 0
		return &result
	}
	return nil
}

func getInt64Pointer(row map[string]interface{}, key string) *int64 {
	val, exists := row[key]
	if !exists || val == nil {
		return nil
	}

	switch v := val.(type) {
	case int64:
		return &v
	case []byte: // Also handles []uint8 since they're the same type
		// Handle MySQL driver []byte/[]uint8 results
		if len(v) == 0 {
			return nil
		}
		str := string(v)
		if parsed, err := strconv.ParseInt(str, 10, 64); err == nil {
			return &parsed
		}
		return nil
	}
	return nil
}

func getIntPointer(row map[string]interface{}, key string) *int {
	val, exists := row[key]
	if !exists || val == nil {
		return nil
	}

	switch v := val.(type) {
	case int64:
		result := int(v)
		return &result
	case []byte: // Also handles []uint8 since they're the same type
		// Handle MySQL driver []byte/[]uint8 results
		if len(v) == 0 {
			return nil
		}
		str := string(v)
		if parsed, err := strconv.ParseInt(str, 10, 64); err == nil {
			result := int(parsed)
			return &result
		}
		return nil
	}
	return nil
}

func getStringPointer(row map[string]interface{}, key string) *string {
	if val, ok := row[key].(string); ok {
		return &val
	}
	if val, ok := row[key].([]byte); ok {
		str := string(val)
		return &str
	}
	return nil
}
