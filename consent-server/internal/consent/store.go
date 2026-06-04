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
	dbconst "github.com/wso2/openfgc/internal/system/database/constants"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/database/provider"
	dbutils "github.com/wso2/openfgc/internal/system/database/utils"
	"github.com/wso2/openfgc/internal/system/stores/interfaces"
)

// consentColumns is the SELECT column list shared across CONSENT table queries.
const consentColumns = "CONSENT_ID, CREATED_TIME, UPDATED_TIME, GROUP_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, EXPIRATION_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID"

// Pre-defined DBQuery objects for simple, single-path operations.
var (
	QueryCreateConsent = dbmodel.DBQuery{
		ID:            "CREATE_CONSENT",
		Query:         "INSERT INTO CONSENT (" + consentColumns + ") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT (" + consentColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
	}

	QueryGetConsentByID = dbmodel.DBQuery{
		ID:            "GET_CONSENT_BY_ID",
		Query:         "SELECT " + consentColumns + " FROM CONSENT WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT " + consentColumns + " FROM CONSENT WHERE CONSENT_ID = $1 AND ORG_ID = $2",
	}

	QueryUpdateConsent = dbmodel.DBQuery{
		ID:            "UPDATE_CONSENT",
		Query:         "UPDATE CONSENT SET UPDATED_TIME = ?, CONSENT_TYPE = ?, CONSENT_FREQUENCY = ?, EXPIRATION_TIME = ?, RECURRING_INDICATOR = ?, DATA_ACCESS_VALIDITY_DURATION = ? WHERE CONSENT_ID = ? AND ORG_ID = ?",
		PostgresQuery: "UPDATE CONSENT SET UPDATED_TIME = $1, CONSENT_TYPE = $2, CONSENT_FREQUENCY = $3, EXPIRATION_TIME = $4, RECURRING_INDICATOR = $5, DATA_ACCESS_VALIDITY_DURATION = $6 WHERE CONSENT_ID = $7 AND ORG_ID = $8",
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

	// Purpose-consent mapping queries
	QueryCreateConsentPurposeMapping = dbmodel.DBQuery{
		ID:            "CREATE_CONSENT_PURPOSE_MAPPING",
		Query:         "INSERT INTO PURPOSE_CONSENT_MAPPING (CONSENT_ID, PURPOSE_VERSION_ID, ORG_ID) VALUES (?, ?, ?)",
		PostgresQuery: "INSERT INTO PURPOSE_CONSENT_MAPPING (CONSENT_ID, PURPOSE_VERSION_ID, ORG_ID) VALUES ($1, $2, $3)",
	}

	QueryGetConsentPurposesByConsentID = dbmodel.DBQuery{
		ID: "GET_PURPOSES_BY_CONSENT_ID",
		Query: `SELECT pcm.CONSENT_ID, pcm.PURPOSE_VERSION_ID, p.ID AS PURPOSE_ID,
				p.NAME AS PURPOSE_NAME, p.GROUP_ID AS PURPOSE_GROUP_ID,
				p.VERSION AS PURPOSE_VERSION, p.DISPLAY_NAME, p.DESCRIPTION, pcm.ORG_ID
			FROM PURPOSE_CONSENT_MAPPING pcm
			JOIN PURPOSE p ON pcm.PURPOSE_VERSION_ID = p.VERSION_ID AND pcm.ORG_ID = p.ORG_ID
			WHERE pcm.CONSENT_ID = ? AND pcm.ORG_ID = ?
			ORDER BY p.NAME`,
		PostgresQuery: `SELECT pcm.CONSENT_ID, pcm.PURPOSE_VERSION_ID, p.ID AS PURPOSE_ID,
				p.NAME AS PURPOSE_NAME, p.GROUP_ID AS PURPOSE_GROUP_ID,
				p.VERSION AS PURPOSE_VERSION, p.DISPLAY_NAME, p.DESCRIPTION, pcm.ORG_ID
			FROM PURPOSE_CONSENT_MAPPING pcm
			JOIN PURPOSE p ON pcm.PURPOSE_VERSION_ID = p.VERSION_ID AND pcm.ORG_ID = p.ORG_ID
			WHERE pcm.CONSENT_ID = $1 AND pcm.ORG_ID = $2
			ORDER BY p.NAME`,
	}

	// CheckPurposeUsedInConsents checks if any version of a logical purpose is used in any consent.
	QueryCheckPurposeUsedInConsents = dbmodel.DBQuery{
		ID: "CHECK_PURPOSE_USED_IN_CONSENTS",
		Query: `SELECT COUNT(*) AS count FROM PURPOSE_CONSENT_MAPPING pcm
			JOIN PURPOSE p ON pcm.PURPOSE_VERSION_ID = p.VERSION_ID
			WHERE p.ID = ? AND pcm.ORG_ID = ?`,
		PostgresQuery: `SELECT COUNT(*) AS count FROM PURPOSE_CONSENT_MAPPING pcm
			JOIN PURPOSE p ON pcm.PURPOSE_VERSION_ID = p.VERSION_ID
			WHERE p.ID = $1 AND pcm.ORG_ID = $2`,
	}

	// Element approval queries
	QueryCreateElementApproval = dbmodel.DBQuery{
		ID:            "CREATE_ELEMENT_APPROVAL",
		Query:         "INSERT INTO CONSENT_ELEMENT_APPROVAL (CONSENT_ID, PURPOSE_VERSION_ID, ELEMENT_VERSION_ID, APPROVED, VALUE, ORG_ID) VALUES (?, ?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO CONSENT_ELEMENT_APPROVAL (CONSENT_ID, PURPOSE_VERSION_ID, ELEMENT_VERSION_ID, APPROVED, VALUE, ORG_ID) VALUES ($1, $2, $3, $4, $5, $6)",
	}

	QueryGetElementApprovalsByConsentID = dbmodel.DBQuery{
		ID: "GET_ELEMENT_APPROVALS_BY_CONSENT_ID",
		Query: `SELECT pa.CONSENT_ID, pa.PURPOSE_VERSION_ID, pa.ELEMENT_VERSION_ID,
				e.ID AS ELEMENT_ID, e.NAME AS ELEMENT_NAME, e.NAMESPACE AS ELEMENT_NAMESPACE,
				e.VERSION AS ELEMENT_VERSION, e.TYPE AS ELEMENT_TYPE,
				e.DISPLAY_NAME AS ELEMENT_DISPLAY_NAME, e.DESCRIPTION AS ELEMENT_DESCRIPTION,
				m.MANDATORY, pa.APPROVED, pa.VALUE, pa.ORG_ID
			FROM CONSENT_ELEMENT_APPROVAL pa
			JOIN ELEMENT e ON pa.ELEMENT_VERSION_ID = e.VERSION_ID
			JOIN PURPOSE_ELEMENT_MAPPING m
				ON pa.PURPOSE_VERSION_ID = m.PURPOSE_VERSION_ID
				AND pa.ELEMENT_VERSION_ID = m.ELEMENT_VERSION_ID
			WHERE pa.CONSENT_ID = ? AND pa.ORG_ID = ?
			ORDER BY pa.PURPOSE_VERSION_ID, e.NAME`,
		PostgresQuery: `SELECT pa.CONSENT_ID, pa.PURPOSE_VERSION_ID, pa.ELEMENT_VERSION_ID,
				e.ID AS ELEMENT_ID, e.NAME AS ELEMENT_NAME, e.NAMESPACE AS ELEMENT_NAMESPACE,
				e.VERSION AS ELEMENT_VERSION, e.TYPE AS ELEMENT_TYPE,
				e.DISPLAY_NAME AS ELEMENT_DISPLAY_NAME, e.DESCRIPTION AS ELEMENT_DESCRIPTION,
				m.MANDATORY, pa.APPROVED, pa.VALUE, pa.ORG_ID
			FROM CONSENT_ELEMENT_APPROVAL pa
			JOIN ELEMENT e ON pa.ELEMENT_VERSION_ID = e.VERSION_ID
			JOIN PURPOSE_ELEMENT_MAPPING m
				ON pa.PURPOSE_VERSION_ID = m.PURPOSE_VERSION_ID
				AND pa.ELEMENT_VERSION_ID = m.ELEMENT_VERSION_ID
			WHERE pa.CONSENT_ID = $1 AND pa.ORG_ID = $2
			ORDER BY pa.PURPOSE_VERSION_ID, e.NAME`,
	}

	QueryGetElementPropertiesByConsentID = dbmodel.DBQuery{
		ID: "GET_ELEMENT_PROPERTIES_BY_CONSENT_ID",
		Query: `SELECT ep.ELEMENT_VERSION_ID, ep.ATT_KEY, ep.ATT_VALUE
			FROM ELEMENT_PROPERTY ep
			JOIN CONSENT_ELEMENT_APPROVAL cea
				ON ep.ELEMENT_VERSION_ID = cea.ELEMENT_VERSION_ID AND ep.ORG_ID = cea.ORG_ID
			WHERE cea.CONSENT_ID = ? AND cea.ORG_ID = ?`,
		PostgresQuery: `SELECT ep.ELEMENT_VERSION_ID, ep.ATT_KEY, ep.ATT_VALUE
			FROM ELEMENT_PROPERTY ep
			JOIN CONSENT_ELEMENT_APPROVAL cea
				ON ep.ELEMENT_VERSION_ID = cea.ELEMENT_VERSION_ID AND ep.ORG_ID = cea.ORG_ID
			WHERE cea.CONSENT_ID = $1 AND cea.ORG_ID = $2`,
	}

	QueryGetPurposePropertiesByConsentID = dbmodel.DBQuery{
		ID: "GET_PURPOSE_PROPERTIES_BY_CONSENT_ID",
		Query: `SELECT pp.PURPOSE_VERSION_ID, pp.ATT_KEY, pp.ATT_VALUE
			FROM PURPOSE_PROPERTY pp
			JOIN PURPOSE_CONSENT_MAPPING pcm
				ON pp.PURPOSE_VERSION_ID = pcm.PURPOSE_VERSION_ID AND pp.ORG_ID = pcm.ORG_ID
			WHERE pcm.CONSENT_ID = ? AND pcm.ORG_ID = ?`,
		PostgresQuery: `SELECT pp.PURPOSE_VERSION_ID, pp.ATT_KEY, pp.ATT_VALUE
			FROM PURPOSE_PROPERTY pp
			JOIN PURPOSE_CONSENT_MAPPING pcm
				ON pp.PURPOSE_VERSION_ID = pcm.PURPOSE_VERSION_ID AND pp.ORG_ID = pcm.ORG_ID
			WHERE pcm.CONSENT_ID = $1 AND pcm.ORG_ID = $2`,
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

	QueryGetExpiredConsents = dbmodel.DBQuery{
		ID:            "GET_EXPIRED_CONSENTS",
		Query:         "SELECT CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID FROM CONSENT WHERE VALIDITY_TIME < ? AND CURRENT_STATUS IN (%s)",
		PostgresQuery: "SELECT CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID FROM CONSENT WHERE VALIDITY_TIME < $1 AND CURRENT_STATUS IN (%s)",
	}

	// QueryGetAttributesByConsentIDs Dynamic query stubs — built at runtime based on filter values.
	QueryGetAttributesByConsentIDs = dbmodel.DBQuery{ID: "GET_ATTRIBUTES_BY_CONSENT_IDS", Query: ""}
	QuerySearchConsentsCount       = dbmodel.DBQuery{ID: "COUNT_CONSENT_SEARCH_RESULTS", Query: ""}
	QuerySearchConsentsData        = dbmodel.DBQuery{ID: "SEARCH_CONSENTS_DATA", Query: ""}
)

// store implements the interfaces.ConsentStore interface.
type store struct{}

// NewConsentStore creates a new consent store.
func NewConsentStore() interfaces.ConsentStore {
	return &store{}
}

func (s *store) getDBClient() (provider.DBClientInterface, error) {
	return provider.GetDBProvider().GetConsentDBClient()
}

// =============================================================================
// Write operations (transactional)
// =============================================================================

// Create inserts a new consent within a transaction.
func (s *store) Create(tx dbmodel.TxInterface, consent *model.Consent) error {
	_, err := tx.Exec(QueryCreateConsent,
		consent.ConsentID, consent.CreatedTime, consent.UpdatedTime, consent.GroupID,
		consent.ConsentType, consent.CurrentStatus, consent.ConsentFrequency,
		consent.ExpirationTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
		consent.OrgID)
	return err
}

// Update updates a consent's mutable fields within a transaction.
func (s *store) Update(tx dbmodel.TxInterface, consent *model.Consent) error {
	_, err := tx.Exec(QueryUpdateConsent,
		consent.UpdatedTime, consent.ConsentType, consent.ConsentFrequency,
		consent.ExpirationTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
		consent.ConsentID, consent.OrgID)
	return err
}

// UpdateStatus updates the consent's current status within a transaction.
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

// CreateAttributes inserts multiple consent attributes within a transaction.
func (s *store) CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentAttribute) error {
	for _, attr := range attributes {
		if _, err := tx.Exec(QueryCreateAttribute, attr.ConsentID, attr.AttKey, attr.AttValue, attr.OrgID); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAttributesByConsentID deletes all attributes for a consent within a transaction.
func (s *store) DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteAttributesByConsentID, consentID, orgID)
	return err
}

// CreateStatusAudit inserts a status audit entry within a transaction.
func (s *store) CreateStatusAudit(tx dbmodel.TxInterface, audit *model.ConsentStatusAudit) error {
	_, err := tx.Exec(QueryCreateStatusAudit,
		audit.StatusAuditID, audit.ConsentID, audit.CurrentStatus, audit.ActionTime,
		audit.Reason, audit.ActionBy, audit.PreviousStatus, audit.OrgID)
	return err
}

// LinkPurposeVersionToConsent records that a consent was created against a specific purpose version.
func (s *store) LinkPurposeVersionToConsent(tx dbmodel.TxInterface, consentID, purposeVersionID, orgID string) error {
	_, err := tx.Exec(QueryCreateConsentPurposeMapping, consentID, purposeVersionID, orgID)
	return err
}

// CreateElementApproval records a user's approval state for one element within a purpose version.
func (s *store) CreateElementApproval(tx dbmodel.TxInterface, approval *model.ConsentElementApproval) error {
	_, err := tx.Exec(QueryCreateElementApproval,
		approval.ConsentID, approval.PurposeVersionID, approval.ElementVersionID,
		approval.Approved, approval.Value, approval.OrgID)
	return err
}

// DeletePurposesByConsentID removes all purpose-version links for a consent within a transaction.
func (s *store) DeletePurposesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteConsentPurposesByConsentID, consentID, orgID)
	return err
}

// DeleteElementApprovalsByConsentID removes all element approvals for a consent within a transaction.
func (s *store) DeleteElementApprovalsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteElementApprovalsByConsentID, consentID, orgID)
	return err
}

// =============================================================================
// Read operations
// =============================================================================

// GetByID retrieves a consent by ID.
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

// GetAttributesByConsentID retrieves all attributes for a single consent.
func (s *store) GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentAttribute, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetAttributesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	attrs := make([]model.ConsentAttribute, 0, len(rows))
	for _, row := range rows {
		if attr := mapToConsentAttribute(row); attr != nil {
			attrs = append(attrs, *attr)
		}
	}
	return attrs, nil
}

// GetAttributesByConsentIDs retrieves attributes for multiple consents, grouped by consent ID.
// One DB round-trip for all consents.
func (s *store) GetAttributesByConsentIDs(ctx context.Context, consentIDs []string, orgID string) (map[string]map[string]string, error) {
	if len(consentIDs) == 0 {
		return make(map[string]map[string]string), nil
	}
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	ph := strings.Repeat("?,", len(consentIDs))
	args := make([]interface{}, 0, len(consentIDs)+1)
	for _, id := range consentIDs {
		args = append(args, id)
	}
	args = append(args, orgID)

	rawSQL := fmt.Sprintf("SELECT CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID IN (%s) AND ORG_ID = ?", ph[:len(ph)-1])
	q := QueryGetAttributesByConsentIDs
	q.Query = rawSQL
	q.PostgresQuery = dbutils.ConvertToPostgresParams(rawSQL)

	rows, err := dbClient.Query(q, args...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]string)
	for _, row := range rows {
		attr := mapToConsentAttribute(row)
		if attr == nil {
			continue
		}
		if result[attr.ConsentID] == nil {
			result[attr.ConsentID] = make(map[string]string)
		}
		result[attr.ConsentID][attr.AttKey] = attr.AttValue
	}
	return result, nil
}

// GetConsentIDsByAttributeKey returns all consent IDs that have a specific attribute key.
func (s *store) GetConsentIDsByAttributeKey(ctx context.Context, key, orgID string) ([]string, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryFindConsentIDsByAttributeKey, key, orgID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if id := getString(row, "consent_id"); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// GetConsentIDsByAttribute returns all consent IDs that have a specific attribute key-value pair.
func (s *store) GetConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryFindConsentIDsByAttribute, key, value, orgID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if id := getString(row, "consent_id"); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// GetPurposesByConsentID returns all purpose rows joined with PURPOSE metadata for a consent.
func (s *store) GetPurposesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentPurposeRow, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetConsentPurposesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	result := make([]model.ConsentPurposeRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapToConsentPurposeRow(row))
	}
	return result, nil
}

// GetElementApprovalsByConsentID returns all approval rows joined with ELEMENT metadata for a consent.
func (s *store) GetElementApprovalsByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentApprovalRow, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetElementApprovalsByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	result := make([]model.ConsentApprovalRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapToConsentApprovalRow(row))
	}
	return result, nil
}

// GetElementPropertiesByConsentID returns element properties for all elements in the consent,
// keyed by element version ID then attribute key.
func (s *store) GetElementPropertiesByConsentID(ctx context.Context, consentID, orgID string) (map[string]map[string]string, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetElementPropertiesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]string)
	for _, row := range rows {
		versionID := getString(row, "element_version_id")
		key := getString(row, "att_key")
		value := getString(row, "att_value")
		if versionID == "" || key == "" {
			continue
		}
		if result[versionID] == nil {
			result[versionID] = make(map[string]string)
		}
		result[versionID][key] = value
	}
	return result, nil
}

// GetPurposePropertiesByConsentID returns purpose properties for all purposes in the consent,
// keyed by purpose version ID then attribute key.
func (s *store) GetPurposePropertiesByConsentID(ctx context.Context, consentID, orgID string) (map[string]map[string]string, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetPurposePropertiesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]string)
	for _, row := range rows {
		versionID := getString(row, "purpose_version_id")
		key := getString(row, "att_key")
		value := getString(row, "att_value")
		if versionID == "" || key == "" {
			continue
		}
		if result[versionID] == nil {
			result[versionID] = make(map[string]string)
		}
		result[versionID][key] = value
	}
	return result, nil
}

// IsPurposeUsedInConsents reports whether any version of a logical purpose is referenced by any consent.
func (s *store) IsPurposeUsedInConsents(ctx context.Context, purposeID, orgID string) (bool, error) {
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
	return extractCount(rows[0]) > 0, nil
}

// Search retrieves consents matching the filters with pagination.
// Uses EXISTS subqueries for purpose/element filters to avoid duplicate rows from JOINs.
func (s *store) Search(ctx context.Context, filters model.ConsentSearchFilter) ([]model.Consent, int, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get database client: %w", err)
	}

	whereConditions := []string{"CONSENT.ORG_ID = ?"}
	args := []interface{}{filters.OrgID}
	countArgs := []interface{}{filters.OrgID}
	joinClause := ""

	if len(filters.ConsentIDs) > 0 {
		ph := strings.Repeat("?,", len(filters.ConsentIDs))
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CONSENT_ID IN (%s)", ph[:len(ph)-1]))
		for _, id := range filters.ConsentIDs {
			args = append(args, id)
			countArgs = append(countArgs, id)
		}
	}

	if len(filters.ConsentTypes) > 0 {
		ph := strings.Repeat("?,", len(filters.ConsentTypes))
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CONSENT_TYPE IN (%s)", ph[:len(ph)-1]))
		for _, ct := range filters.ConsentTypes {
			args = append(args, ct)
			countArgs = append(countArgs, ct)
		}
	}

	if len(filters.ConsentStatuses) > 0 {
		ph := strings.Repeat("?,", len(filters.ConsentStatuses))
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CURRENT_STATUS IN (%s)", ph[:len(ph)-1]))
		for _, st := range filters.ConsentStatuses {
			args = append(args, strings.ToUpper(st))
			countArgs = append(countArgs, strings.ToUpper(st))
		}
	}

	if len(filters.GroupIDs) > 0 {
		ph := strings.Repeat("?,", len(filters.GroupIDs))
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.GROUP_ID IN (%s)", ph[:len(ph)-1]))
		for _, gid := range filters.GroupIDs {
			args = append(args, gid)
			countArgs = append(countArgs, gid)
		}
	}

	// UserIDs filter via JOIN — kept as JOIN so COUNT(DISTINCT) handles any duplicates.
	if len(filters.UserIDs) > 0 {
		ph := strings.Repeat("?,", len(filters.UserIDs))
		joinClause += " INNER JOIN CONSENT_AUTH_RESOURCE car ON CONSENT.CONSENT_ID = car.CONSENT_ID AND CONSENT.ORG_ID = car.ORG_ID"
		whereConditions = append(whereConditions, fmt.Sprintf("car.USER_ID IN (%s)", ph[:len(ph)-1]))
		for _, uid := range filters.UserIDs {
			args = append(args, uid)
			countArgs = append(countArgs, uid)
		}
	}

	// PurposeName filter via EXISTS subquery to avoid duplicate rows.
	if filters.PurposeName != "" {
		pattern, escapeClause := consentLikePattern(dbClient, filters.PurposeName)
		existsSQL := "EXISTS (SELECT 1 FROM PURPOSE_CONSENT_MAPPING pcm" +
			" JOIN PURPOSE p ON pcm.PURPOSE_VERSION_ID = p.VERSION_ID" +
			" WHERE pcm.CONSENT_ID = CONSENT.CONSENT_ID AND pcm.ORG_ID = CONSENT.ORG_ID" +
			" AND p.NAME LIKE ?" + escapeClause
		purposeArgs := []interface{}{pattern}
		if filters.PurposeVersion != nil {
			existsSQL += " AND p.VERSION = ?"
			purposeArgs = append(purposeArgs, *filters.PurposeVersion)
		}
		existsSQL += ")"
		whereConditions = append(whereConditions, existsSQL)
		args = append(args, purposeArgs...)
		countArgs = append(countArgs, purposeArgs...)
	}

	// Element filters via EXISTS subquery.
	if filters.ElementName != "" || filters.ElementNamespace != "" || filters.ElementVersion != nil {
		var elemClauses []string
		var elemArgs []interface{}
		elemClauses = append(elemClauses,
			"pcm2.CONSENT_ID = CONSENT.CONSENT_ID",
			"pcm2.ORG_ID = CONSENT.ORG_ID",
		)
		if filters.ElementName != "" {
			elemClauses = append(elemClauses, "e.NAME = ?")
			elemArgs = append(elemArgs, filters.ElementName)
		}
		if filters.ElementNamespace != "" {
			elemClauses = append(elemClauses, "e.NAMESPACE = ?")
			elemArgs = append(elemArgs, filters.ElementNamespace)
		}
		if filters.ElementVersion != nil {
			elemClauses = append(elemClauses, "e.VERSION = ?")
			elemArgs = append(elemArgs, *filters.ElementVersion)
		}
		existsSQL := "EXISTS (SELECT 1 FROM PURPOSE_CONSENT_MAPPING pcm2" +
			" JOIN PURPOSE_ELEMENT_MAPPING pem ON pcm2.PURPOSE_VERSION_ID = pem.PURPOSE_VERSION_ID" +
			" JOIN ELEMENT e ON pem.ELEMENT_VERSION_ID = e.VERSION_ID" +
			" WHERE " + strings.Join(elemClauses, " AND ") + ")"
		whereConditions = append(whereConditions, existsSQL)
		args = append(args, elemArgs...)
		countArgs = append(countArgs, elemArgs...)
	}

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

	// Count query
	countSQL := fmt.Sprintf("SELECT COUNT(DISTINCT CONSENT.CONSENT_ID) AS count FROM CONSENT%s WHERE %s",
		joinClause, whereClause)
	countQ := QuerySearchConsentsCount
	countQ.Query = countSQL
	countQ.PostgresQuery = dbutils.ConvertToPostgresParams(countSQL)

	countRows, err := dbClient.Query(countQ, countArgs...)
	if err != nil {
		return nil, 0, err
	}
	totalCount := 0
	if len(countRows) > 0 {
		totalCount = int(extractCount(countRows[0]))
	}

	// Data query
	dataSQL := fmt.Sprintf(
		"SELECT DISTINCT CONSENT.CONSENT_ID, CONSENT.CREATED_TIME, CONSENT.UPDATED_TIME,"+
			" CONSENT.GROUP_ID, CONSENT.CONSENT_TYPE, CONSENT.CURRENT_STATUS,"+
			" CONSENT.CONSENT_FREQUENCY, CONSENT.EXPIRATION_TIME, CONSENT.RECURRING_INDICATOR,"+
			" CONSENT.DATA_ACCESS_VALIDITY_DURATION, CONSENT.ORG_ID"+
			" FROM CONSENT%s WHERE %s ORDER BY CONSENT.CREATED_TIME DESC LIMIT ? OFFSET ?",
		joinClause, whereClause,
	)
	dataArgs := append(args, filters.Limit, filters.Offset)
	dataQ := QuerySearchConsentsData
	dataQ.Query = dataSQL
	dataQ.PostgresQuery = dbutils.ConvertToPostgresParams(dataSQL)

	rows, err := dbClient.Query(dataQ, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	consents := make([]model.Consent, 0, len(rows))
	for _, row := range rows {
		if c := mapToConsent(row); c != nil {
			consents = append(consents, *c)
		}
	}
	return consents, totalCount, nil
}

// GetExpiredConsents retrieves consents that have expired based on the current time and specified expirable statuses.
func (s *store) GetExpiredConsents(ctx context.Context, currentTimeMs int64, expirableStatuses []string) ([]model.Consent, error) {

	if len(expirableStatuses) == 0 {
		return []model.Consent{}, nil
	}

	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	placeholders := make([]string, len(expirableStatuses))
	args := []interface{}{currentTimeMs}
	for i, status := range expirableStatuses {
		placeholders[i] = "?"
		args = append(args, status)
	}

	query := fmt.Sprintf(QueryGetExpiredConsents.Query, strings.Join(placeholders, ","))
	postgresQuery := fmt.Sprintf(QueryGetExpiredConsents.PostgresQuery, strings.Join(placeholders, ","))

	rows, err := dbClient.Query(dbmodel.DBQuery{
		ID:            QueryGetExpiredConsents.ID,
		Query:         query,
		PostgresQuery: postgresQuery,
	}, args...)
	if err != nil {
		return nil, err
	}

	consents := make([]model.Consent, 0, len(rows))
	for _, row := range rows {
		consent := mapToConsent(row)
		if consent != nil {
			consents = append(consents, *consent)
		}
	}

	return consents, nil
}

// =============================================================================
// Mappers — DBClient normalizes column names to lowercase.
// =============================================================================

func mapToConsent(row map[string]interface{}) *model.Consent {
	if row == nil {
		return nil
	}
	return &model.Consent{
		ConsentID:                  getString(row, "consent_id"),
		CreatedTime:                getInt64(row, "created_time"),
		UpdatedTime:                getInt64(row, "updated_time"),
		GroupID:                    getString(row, "group_id"),
		ConsentType:                getString(row, "consent_type"),
		CurrentStatus:              getString(row, "current_status"),
		ConsentFrequency:           getIntPtr(row, "consent_frequency"),
		ExpirationTime:             getInt64Ptr(row, "expiration_time"),
		RecurringIndicator:         getBoolPtr(row, "recurring_indicator"),
		DataAccessValidityDuration: getInt64Ptr(row, "data_access_validity_duration"),
		OrgID:                      getString(row, "org_id"),
	}
}

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

func mapToConsentPurposeRow(row map[string]interface{}) model.ConsentPurposeRow {
	return model.ConsentPurposeRow{
		ConsentID:        getString(row, "consent_id"),
		PurposeVersionID: getString(row, "purpose_version_id"),
		PurposeID:        getString(row, "purpose_id"),
		PurposeName:      getString(row, "purpose_name"),
		PurposeGroupID:   getString(row, "purpose_group_id"),
		PurposeVersion:   getInt(row, "purpose_version"),
		DisplayName:      getStringPtr(row, "display_name"),
		Description:      getStringPtr(row, "description"),
		OrgID:            getString(row, "org_id"),
	}
}

func mapToConsentApprovalRow(row map[string]interface{}) model.ConsentApprovalRow {
	return model.ConsentApprovalRow{
		ConsentID:          getString(row, "consent_id"),
		PurposeVersionID:   getString(row, "purpose_version_id"),
		ElementVersionID:   getString(row, "element_version_id"),
		ElementID:          getString(row, "element_id"),
		ElementName:        getString(row, "element_name"),
		ElementNamespace:   getString(row, "element_namespace"),
		ElementVersionNum:  getInt(row, "element_version"),
		ElementType:        getString(row, "element_type"),
		ElementDisplayName: getStringPtr(row, "element_display_name"),
		ElementDescription: getStringPtr(row, "element_description"),
		Mandatory:          getBool(row, "mandatory"),
		Approved:           getBool(row, "approved"),
		Value:              getStringPtr(row, "value"),
		OrgID:              getString(row, "org_id"),
	}
}

// =============================================================================
// DB row helpers
// =============================================================================

// consentLikePattern escapes a string for LIKE search and returns the pattern and ESCAPE clause.
func consentLikePattern(dbClient provider.DBClientInterface, name string) (pattern, escapeClause string) {
	var escaped string
	switch dbClient.GetDBType() {
	case dbconst.DatabaseTypeSQLite, dbconst.DatabaseTypePostgres:
		r := strings.NewReplacer("|", "||", "%", "|%", "_", "|_")
		escaped = r.Replace(name)
		escapeClause = " ESCAPE '|'"
	default: // MySQL
		r := strings.NewReplacer("%", "\\%", "_", "\\_")
		escaped = r.Replace(name)
	}
	return "%" + escaped + "%", escapeClause
}

// extractCount reads the "count" column from a row, handling int64 and []uint8 (MySQL).
func extractCount(row map[string]interface{}) int64 {
	if v, ok := row["count"].(int64); ok {
		return v
	}
	if v, ok := row["count"].([]uint8); ok {
		if parsed, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

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

func getInt64Ptr(row map[string]interface{}, key string) *int64 {
	val, exists := row[key]
	if !exists || val == nil {
		return nil
	}
	switch v := val.(type) {
	case int64:
		return &v
	case []byte:
		if len(v) == 0 {
			return nil
		}
		if parsed, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return &parsed
		}
	}
	return nil
}

func getInt(row map[string]interface{}, key string) int {
	switch v := row[key].(type) {
	case int64:
		return int(v)
	case uint32:
		return int(v)
	case int32:
		return int(v)
	}
	return 0
}

func getIntPtr(row map[string]interface{}, key string) *int {
	val, exists := row[key]
	if !exists || val == nil {
		return nil
	}
	switch v := val.(type) {
	case int64:
		result := int(v)
		return &result
	case []byte:
		if len(v) == 0 {
			return nil
		}
		if parsed, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			result := int(parsed)
			return &result
		}
	}
	return nil
}

func getBool(row map[string]interface{}, key string) bool {
	switch v := row[key].(type) {
	case bool:
		return v
	case int64:
		return v != 0
	case uint8:
		return v != 0
	case int32:
		return v != 0
	}
	return false
}

func getBoolPtr(row map[string]interface{}, key string) *bool {
	val, exists := row[key]
	if !exists || val == nil {
		return nil
	}
	switch v := val.(type) {
	case bool:
		return &v
	case int64:
		result := v != 0
		return &result
	case uint8:
		result := v != 0
		return &result
	}
	return nil
}
