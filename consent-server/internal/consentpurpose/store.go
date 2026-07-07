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

	"github.com/wso2/openfgc/consent-server/internal/consentpurpose/model"
	dbconst "github.com/wso2/openfgc/consent-server/internal/system/database/constants"
	dbmodel "github.com/wso2/openfgc/consent-server/internal/system/database/model"
	"github.com/wso2/openfgc/consent-server/internal/system/database/provider"
	dbutils "github.com/wso2/openfgc/consent-server/internal/system/database/utils"
	"github.com/wso2/openfgc/consent-server/internal/system/stores/interfaces"
)

// purposeColumns is the SELECT column list shared across all PURPOSE queries.
const purposeColumns = "VERSION_ID, ID, NAME, GROUP_ID, VERSION, DISPLAY_NAME, DESCRIPTION, CREATED_TIME, ORG_ID"

// Pre-defined DBQuery objects for simple, single-path operations.
var (
	QueryInsertPurposeVersion = dbmodel.DBQuery{
		ID:            "INSERT_PURPOSE_VERSION",
		Query:         "INSERT INTO PURPOSE (VERSION_ID, ID, NAME, GROUP_ID, VERSION, DISPLAY_NAME, DESCRIPTION, CREATED_TIME, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO PURPOSE (VERSION_ID, ID, NAME, GROUP_ID, VERSION, DISPLAY_NAME, DESCRIPTION, CREATED_TIME, ORG_ID) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
	}

	QueryInsertPurposeProperty = dbmodel.DBQuery{
		ID:            "INSERT_PURPOSE_PROPERTY",
		Query:         "INSERT INTO PURPOSE_PROPERTY (PURPOSE_VERSION_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES (?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO PURPOSE_PROPERTY (PURPOSE_VERSION_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES ($1, $2, $3, $4)",
	}

	QueryGetLatestPurposeVersion = dbmodel.DBQuery{
		ID:            "GET_LATEST_PURPOSE_VERSION",
		Query:         "SELECT " + purposeColumns + " FROM PURPOSE WHERE ID = ? AND ORG_ID = ? ORDER BY VERSION DESC LIMIT 1",
		PostgresQuery: "SELECT " + purposeColumns + " FROM PURPOSE WHERE ID = $1 AND ORG_ID = $2 ORDER BY VERSION DESC LIMIT 1",
	}

	QueryGetPurposeVersion = dbmodel.DBQuery{
		ID:            "GET_PURPOSE_VERSION",
		Query:         "SELECT " + purposeColumns + " FROM PURPOSE WHERE ID = ? AND VERSION = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT " + purposeColumns + " FROM PURPOSE WHERE ID = $1 AND VERSION = $2 AND ORG_ID = $3",
	}

	QueryGetPurposeVersionByID = dbmodel.DBQuery{
		ID:            "GET_PURPOSE_VERSION_BY_ID",
		Query:         "SELECT " + purposeColumns + " FROM PURPOSE WHERE VERSION_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT " + purposeColumns + " FROM PURPOSE WHERE VERSION_ID = $1 AND ORG_ID = $2",
	}

	QueryListPurposeVersions = dbmodel.DBQuery{
		ID:            "LIST_PURPOSE_VERSIONS",
		Query:         "SELECT " + purposeColumns + " FROM PURPOSE WHERE ID = ? AND ORG_ID = ? ORDER BY VERSION ASC",
		PostgresQuery: "SELECT " + purposeColumns + " FROM PURPOSE WHERE ID = $1 AND ORG_ID = $2 ORDER BY VERSION ASC",
	}

	QueryPurposeExists = dbmodel.DBQuery{
		ID:            "PURPOSE_EXISTS",
		Query:         "SELECT COUNT(*) AS cnt FROM PURPOSE WHERE ID = ? AND ORG_ID = ? LIMIT 1",
		PostgresQuery: "SELECT COUNT(*) AS cnt FROM PURPOSE WHERE ID = $1 AND ORG_ID = $2 LIMIT 1",
	}

	QueryGetPurposeByNameAndGroupID = dbmodel.DBQuery{
		ID:            "GET_PURPOSE_BY_NAME_AND_GROUP_ID",
		Query:         "SELECT " + purposeColumns + " FROM PURPOSE WHERE NAME = ? AND GROUP_ID = ? AND ORG_ID = ? ORDER BY VERSION DESC LIMIT 1",
		PostgresQuery: "SELECT " + purposeColumns + " FROM PURPOSE WHERE NAME = $1 AND GROUP_ID = $2 AND ORG_ID = $3 ORDER BY VERSION DESC LIMIT 1",
	}

	QueryExistsPurposeByNameInOrg = dbmodel.DBQuery{
		ID:            "EXISTS_PURPOSE_BY_NAME_IN_ORG",
		Query:         "SELECT COUNT(*) AS cnt FROM PURPOSE WHERE NAME = ? AND ORG_ID = ? LIMIT 1",
		PostgresQuery: "SELECT COUNT(*) AS cnt FROM PURPOSE WHERE NAME = $1 AND ORG_ID = $2 LIMIT 1",
	}

	QueryDeletePurposeVersion = dbmodel.DBQuery{
		ID:            "DELETE_PURPOSE_VERSION",
		Query:         "DELETE FROM PURPOSE WHERE VERSION_ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM PURPOSE WHERE VERSION_ID = $1 AND ORG_ID = $2",
	}

	QueryDeletePurpose = dbmodel.DBQuery{
		ID:            "DELETE_PURPOSE",
		Query:         "DELETE FROM PURPOSE WHERE ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM PURPOSE WHERE ID = $1 AND ORG_ID = $2",
	}

	QueryGetPropertiesByPurposeVersionID = dbmodel.DBQuery{
		ID:            "GET_PURPOSE_PROPERTIES_BY_VERSION_ID",
		Query:         "SELECT PURPOSE_VERSION_ID, ATT_KEY, ATT_VALUE FROM PURPOSE_PROPERTY WHERE PURPOSE_VERSION_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT PURPOSE_VERSION_ID, ATT_KEY, ATT_VALUE FROM PURPOSE_PROPERTY WHERE PURPOSE_VERSION_ID = $1 AND ORG_ID = $2",
	}

	QueryLinkElementVersionToPurposeVersion = dbmodel.DBQuery{
		ID:            "LINK_ELEMENT_VERSION_TO_PURPOSE_VERSION",
		Query:         "INSERT INTO PURPOSE_ELEMENT_MAPPING (PURPOSE_VERSION_ID, ELEMENT_VERSION_ID, MANDATORY, ORG_ID) VALUES (?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO PURPOSE_ELEMENT_MAPPING (PURPOSE_VERSION_ID, ELEMENT_VERSION_ID, MANDATORY, ORG_ID) VALUES ($1, $2, $3, $4)",
	}

	QueryGetPurposeVersionElements = dbmodel.DBQuery{
		ID: "GET_PURPOSE_VERSION_ELEMENTS",
		Query: `SELECT m.ELEMENT_VERSION_ID, e.ID AS ELEMENT_ID, e.NAME, e.NAMESPACE, e.VERSION, m.MANDATORY, e.TYPE, e.ELEMENT_SCHEMA
				FROM PURPOSE_ELEMENT_MAPPING m
				JOIN ELEMENT e ON m.ELEMENT_VERSION_ID = e.VERSION_ID
				WHERE m.PURPOSE_VERSION_ID = ? AND m.ORG_ID = ?`,
		PostgresQuery: `SELECT m.ELEMENT_VERSION_ID, e.ID AS ELEMENT_ID, e.NAME, e.NAMESPACE, e.VERSION, m.MANDATORY, e.TYPE, e.ELEMENT_SCHEMA
				FROM PURPOSE_ELEMENT_MAPPING m
				JOIN ELEMENT e ON m.ELEMENT_VERSION_ID = e.VERSION_ID
				WHERE m.PURPOSE_VERSION_ID = $1 AND m.ORG_ID = $2`,
	}

	QueryIsElementVersionUsedInPurposes = dbmodel.DBQuery{
		ID:            "IS_ELEMENT_VERSION_USED_IN_PURPOSES",
		Query:         "SELECT COUNT(*) AS cnt FROM PURPOSE_ELEMENT_MAPPING WHERE ELEMENT_VERSION_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT COUNT(*) AS cnt FROM PURPOSE_ELEMENT_MAPPING WHERE ELEMENT_VERSION_ID = $1 AND ORG_ID = $2",
	}

	QueryIsPurposeVersionUsedInConsents = dbmodel.DBQuery{
		ID:            "IS_PURPOSE_VERSION_USED_IN_CONSENTS",
		Query:         "SELECT COUNT(*) AS cnt FROM PURPOSE_CONSENT_MAPPING WHERE PURPOSE_VERSION_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT COUNT(*) AS cnt FROM PURPOSE_CONSENT_MAPPING WHERE PURPOSE_VERSION_ID = $1 AND ORG_ID = $2",
	}

	// Stubs for queries built dynamically at runtime.
	QueryListPurposesCount       = dbmodel.DBQuery{ID: "LIST_PURPOSES_COUNT_DYNAMIC", Query: ""}
	QueryListPurposesData        = dbmodel.DBQuery{ID: "LIST_PURPOSES_DATA_DYNAMIC", Query: ""}
	QueryBatchGetPurposeProps    = dbmodel.DBQuery{ID: "BATCH_GET_PURPOSE_PROPERTIES", Query: ""}
	QueryBatchGetPurposeElements = dbmodel.DBQuery{ID: "BATCH_GET_PURPOSE_ELEMENTS", Query: ""}
)

// store implements interfaces.ConsentPurposeStore.
type store struct{}

// NewPurposeStore creates a new purpose store.
func NewPurposeStore() interfaces.ConsentPurposeStore {
	return &store{}
}

func (s *store) getDBClient() (provider.DBClientInterface, error) {
	return provider.GetDBProvider().GetConsentDBClient()
}

// =============================================================================
// Write operations (transactional)
// =============================================================================

// CreatePurposeVersion inserts a new purpose version row and its properties within a transaction.
// Element mappings are linked separately via LinkElementVersionToPurposeVersion.
func (s *store) CreateVersion(tx dbmodel.TxInterface, pv *model.PurposeVersion) error {
	_, err := tx.Exec(QueryInsertPurposeVersion,
		pv.VersionID, pv.ID, pv.Name, pv.GroupID, pv.VersionNum,
		pv.DisplayName, pv.Description, pv.CreatedTime, pv.OrgID,
	)
	if err != nil {
		return err
	}
	for k, val := range pv.Properties {
		if _, err := tx.Exec(QueryInsertPurposeProperty, pv.VersionID, k, val, pv.OrgID); err != nil {
			return err
		}
	}
	return nil
}

// DeletePurposeVersion deletes a specific version row.
// PURPOSE_PROPERTY and PURPOSE_ELEMENT_MAPPING rows cascade automatically.
func (s *store) DeleteVersion(tx dbmodel.TxInterface, purposeVersionID, orgID string) error {
	_, err := tx.Exec(QueryDeletePurposeVersion, purposeVersionID, orgID)
	return err
}

// DeletePurpose deletes all versions of a purpose. Called when the last version is removed.
func (s *store) DeletePurpose(tx dbmodel.TxInterface, purposeID, orgID string) error {
	_, err := tx.Exec(QueryDeletePurpose, purposeID, orgID)
	return err
}

// LinkElementVersionToPurposeVersion inserts a row into PURPOSE_ELEMENT_MAPPING.
func (s *store) LinkElementVersion(tx dbmodel.TxInterface, purposeVersionID, elementVersionID, orgID string, mandatory bool) error {
	_, err := tx.Exec(QueryLinkElementVersionToPurposeVersion, purposeVersionID, elementVersionID, mandatory, orgID)
	return err
}

// =============================================================================
// Read operations
// =============================================================================

// GetLatestPurposeVersion returns the highest-numbered version of a purpose, with properties and elements.
// Returns nil if not found.
func (s *store) GetLatestVersion(ctx context.Context, purposeID, orgID string) (*model.PurposeVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetLatestPurposeVersion, purposeID, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	pv := mapToPurposeVersion(rows[0])
	if err := s.populateVersion(dbClient, pv, orgID); err != nil {
		return nil, err
	}
	return pv, nil
}

// GetPurposeVersion returns a specific version by version number, with properties and elements.
// Returns nil if not found.
func (s *store) GetVersion(ctx context.Context, purposeID string, version int, orgID string) (*model.PurposeVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetPurposeVersion, purposeID, version, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	pv := mapToPurposeVersion(rows[0])
	if err := s.populateVersion(dbClient, pv, orgID); err != nil {
		return nil, err
	}
	return pv, nil
}

// GetPurposeVersionByID returns a purpose version by its VERSION_ID, with properties and elements.
// Returns nil if not found.
func (s *store) GetVersionByID(ctx context.Context, purposeVersionID, orgID string) (*model.PurposeVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetPurposeVersionByID, purposeVersionID, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	pv := mapToPurposeVersion(rows[0])
	if err := s.populateVersion(dbClient, pv, orgID); err != nil {
		return nil, err
	}
	return pv, nil
}

// ListPurposeVersions returns all versions of one purpose ordered ascending, with properties and elements.
func (s *store) ListVersions(ctx context.Context, purposeID, orgID string) ([]model.PurposeVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryListPurposeVersions, purposeID, orgID)
	if err != nil {
		return nil, err
	}
	versions := mapToPurposeVersionSlice(rows)
	if err := s.populateVersionSlice(dbClient, versions, orgID); err != nil {
		return nil, err
	}
	return versions, nil
}

// PurposeExists reports whether any version of the purpose exists.
func (s *store) PurposeExists(ctx context.Context, purposeID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryPurposeExists, purposeID, orgID)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return getInt64(rows[0], "cnt") > 0, nil
}

// GetByNameAndGroupID returns the latest version of a purpose with the given name and groupID,
// or nil if not found. Used to check for name uniqueness before creating a new purpose.
func (s *store) GetByNameAndGroupID(ctx context.Context, name, groupID, orgID string) (*model.PurposeVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetPurposeByNameAndGroupID, name, groupID, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return mapToPurposeVersion(rows[0]), nil
}

// ExistsByNameInOrg reports whether any purpose with the given name exists in the org,
// across all groups. Used to block name reuse regardless of group scope.
func (s *store) ExistsByNameInOrg(ctx context.Context, name, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryExistsPurposeByNameInOrg, name, orgID)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	cnt, ok := rows[0]["cnt"]
	if !ok {
		return false, nil
	}
	switch v := cnt.(type) {
	case int64:
		return v > 0, nil
	case []byte:
		return string(v) != "0", nil
	default:
		return false, nil
	}
}

// GetPurposeVersionElements returns all element refs for a specific purpose version.
func (s *store) GetVersionElements(ctx context.Context, purposeVersionID, orgID string) ([]model.PurposeMappedElement, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	return s.fetchElements(dbClient, purposeVersionID, orgID)
}

// IsElementVersionUsedInPurposes reports whether any purpose version references this element version.
func (s *store) IsElementVersionUsed(ctx context.Context, elementVersionID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryIsElementVersionUsedInPurposes, elementVersionID, orgID)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return getInt64(rows[0], "cnt") > 0, nil
}

// IsPurposeVersionUsedInConsents reports whether any consent references this purpose version.
func (s *store) IsVersionUsedInConsents(ctx context.Context, purposeVersionID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryIsPurposeVersionUsedInConsents, purposeVersionID, orgID)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return getInt64(rows[0], "cnt") > 0, nil
}

// ListPurposes returns the latest version of each purpose matching the filters, with total count.
// When filters.Details is false, Properties and Elements are not populated.
func (s *store) List(ctx context.Context, orgID string, filters model.PurposeListFilter) ([]model.PurposeVersion, int, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get database client: %w", err)
	}

	countQ, dataQ, dataArgs, countArgs := s.buildListQuery(dbClient, orgID, filters)

	countRows, err := dbClient.Query(countQ, countArgs...)
	if err != nil {
		return nil, 0, err
	}
	total := 0
	if len(countRows) > 0 {
		total = int(getInt64(countRows[0], "cnt"))
	}

	rows, err := dbClient.Query(dataQ, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	versions := mapToPurposeVersionSlice(rows)

	if filters.Details && len(versions) > 0 {
		if err := s.populateVersionSlice(dbClient, versions, orgID); err != nil {
			return nil, 0, err
		}
	}

	return versions, total, nil
}

// =============================================================================
// Query builder
// =============================================================================

// buildListQuery constructs the count and data queries for ListPurposes based on filters.
// When filters.PurposeVersion is nil, results are limited to the latest version per purpose.
func (s *store) buildListQuery(dbClient provider.DBClientInterface, orgID string, filters model.PurposeListFilter) (countQ, dataQ dbmodel.DBQuery, dataArgs, countArgs []interface{}) {
	isVersionFiltered := filters.PurposeVersion != nil

	var sb strings.Builder
	var whereClauses []string
	var baseArgs []interface{}

	if isVersionFiltered {
		sb.WriteString("SELECT " + purposeColumns + " FROM PURPOSE p WHERE p.ORG_ID = ?")
		baseArgs = append(baseArgs, orgID)
		whereClauses = append(whereClauses, "p.VERSION = ?")
		baseArgs = append(baseArgs, *filters.PurposeVersion)
	} else {
		// Join with subquery to get the latest version per purpose.
		cols := "p." + strings.ReplaceAll(purposeColumns, ", ", ", p.")
		sb.WriteString("SELECT " + cols +
			" FROM PURPOSE p" +
			" INNER JOIN (SELECT ID, MAX(VERSION) AS MAX_VERSION FROM PURPOSE WHERE ORG_ID = ? GROUP BY ID) AS latest" +
			" ON p.ID = latest.ID AND p.VERSION = latest.MAX_VERSION" +
			" WHERE p.ORG_ID = ?")
		baseArgs = append(baseArgs, orgID, orgID)
	}

	if len(filters.GroupIDs) > 0 {
		placeholders := strings.Repeat("?,", len(filters.GroupIDs))
		placeholders = placeholders[:len(placeholders)-1]
		whereClauses = append(whereClauses, "p.GROUP_ID IN ("+placeholders+")")
		for _, gid := range filters.GroupIDs {
			baseArgs = append(baseArgs, gid)
		}
	}

	if filters.PurposeName != "" {
		pattern, escapeClause := purposeLikePattern(dbClient, filters.PurposeName)
		whereClauses = append(whereClauses, "p.NAME LIKE ?"+escapeClause)
		baseArgs = append(baseArgs, pattern)
	}

	// Element filter — single EXISTS subquery combining all provided element conditions.
	if filters.ElementName != "" || filters.ElementNamespace != "" || filters.ElementVersion != nil {
		var elemClauses []string
		elemClauses = append(elemClauses, "m.PURPOSE_VERSION_ID = p.VERSION_ID", "m.ORG_ID = p.ORG_ID")
		if filters.ElementName != "" {
			elemClauses = append(elemClauses, "e.NAME = ?")
			baseArgs = append(baseArgs, filters.ElementName)
		}
		if filters.ElementNamespace != "" {
			elemClauses = append(elemClauses, "e.NAMESPACE = ?")
			baseArgs = append(baseArgs, filters.ElementNamespace)
		}
		if filters.ElementVersion != nil {
			elemClauses = append(elemClauses, "e.VERSION = ?")
			baseArgs = append(baseArgs, *filters.ElementVersion)
		}
		existsSQL := " EXISTS (SELECT 1 FROM PURPOSE_ELEMENT_MAPPING m JOIN ELEMENT e ON m.ELEMENT_VERSION_ID = e.VERSION_ID WHERE " +
			strings.Join(elemClauses, " AND ") + ")"
		whereClauses = append(whereClauses, existsSQL)
	}

	if len(whereClauses) > 0 {
		sb.WriteString(" AND " + strings.Join(whereClauses, " AND "))
	}

	baseSQL := sb.String()
	countSQL := "SELECT COUNT(*) AS cnt FROM (" + baseSQL + ") AS filtered"
	dataSQL := baseSQL + " ORDER BY p.NAME ASC LIMIT ? OFFSET ?"

	dataArgs = append(baseArgs, filters.Limit, filters.Offset)
	countArgs = baseArgs

	countQ = QueryListPurposesCount
	countQ.Query = countSQL
	countQ.PostgresQuery = dbutils.ConvertToPostgresParams(countSQL)
	dataQ = QueryListPurposesData
	dataQ.Query = dataSQL
	dataQ.PostgresQuery = dbutils.ConvertToPostgresParams(dataSQL)
	return countQ, dataQ, dataArgs, countArgs
}

// =============================================================================
// Population helpers
// =============================================================================

// populateVersion loads properties and elements for a single purpose version in-place.
func (s *store) populateVersion(dbClient provider.DBClientInterface, pv *model.PurposeVersion, orgID string) error {
	props, err := s.fetchProperties(dbClient, pv.VersionID, orgID)
	if err != nil {
		return err
	}
	pv.Properties = props

	elems, err := s.fetchElements(dbClient, pv.VersionID, orgID)
	if err != nil {
		return err
	}
	pv.Elements = elems
	return nil
}

// populateVersionSlice batch-loads properties and elements for a slice of purpose versions.
// Two DB round-trips total — one for all properties, one for all elements.
func (s *store) populateVersionSlice(dbClient provider.DBClientInterface, versions []model.PurposeVersion, orgID string) error {
	if len(versions) == 0 {
		return nil
	}

	versionIDs := make([]string, len(versions))
	for i, v := range versions {
		versionIDs[i] = v.VersionID
	}

	// Batch-fetch properties.
	if err := s.batchPopulateProperties(dbClient, versions, versionIDs, orgID); err != nil {
		return err
	}

	// Batch-fetch elements.
	return s.batchPopulateElements(dbClient, versions, versionIDs, orgID)
}

// fetchProperties loads properties for a single purpose version.
func (s *store) fetchProperties(dbClient provider.DBClientInterface, versionID, orgID string) (map[string]string, error) {
	rows, err := dbClient.Query(QueryGetPropertiesByPurposeVersionID, versionID, orgID)
	if err != nil {
		return nil, err
	}
	props := make(map[string]string, len(rows))
	for _, row := range rows {
		props[getString(row, "att_key")] = getString(row, "att_value")
	}
	return props, nil
}

// fetchElements loads all element refs for a single purpose version.
func (s *store) fetchElements(dbClient provider.DBClientInterface, purposeVersionID, orgID string) ([]model.PurposeMappedElement, error) {
	rows, err := dbClient.Query(QueryGetPurposeVersionElements, purposeVersionID, orgID)
	if err != nil {
		return nil, err
	}
	return mapToPurposeMappedElementSlice(rows), nil
}

// batchPopulateProperties fetches all properties for a set of version IDs in one query
// and fills each version's Properties map.
func (s *store) batchPopulateProperties(dbClient provider.DBClientInterface, versions []model.PurposeVersion, versionIDs []string, orgID string) error {
	placeholders := strings.Repeat("?,", len(versionIDs))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]interface{}, 0, len(versionIDs)+1)
	for _, id := range versionIDs {
		args = append(args, id)
	}
	args = append(args, orgID)

	rawSQL := fmt.Sprintf(
		"SELECT PURPOSE_VERSION_ID, ATT_KEY, ATT_VALUE FROM PURPOSE_PROPERTY WHERE PURPOSE_VERSION_ID IN (%s) AND ORG_ID = ?",
		placeholders,
	)
	q := QueryBatchGetPurposeProps
	q.Query = rawSQL
	q.PostgresQuery = dbutils.ConvertToPostgresParams(rawSQL)

	rows, err := dbClient.Query(q, args...)
	if err != nil {
		return err
	}

	byVersionID := make(map[string]int, len(versions))
	for i, v := range versions {
		byVersionID[v.VersionID] = i
		versions[i].Properties = make(map[string]string)
	}
	for _, row := range rows {
		vid := getString(row, "purpose_version_id")
		if idx, ok := byVersionID[vid]; ok {
			versions[idx].Properties[getString(row, "att_key")] = getString(row, "att_value")
		}
	}
	return nil
}

// batchPopulateElements fetches all PURPOSE_ELEMENT_MAPPING rows (joined with ELEMENT) for
// a set of version IDs in one query and fills each version's Elements slice.
func (s *store) batchPopulateElements(dbClient provider.DBClientInterface, versions []model.PurposeVersion, versionIDs []string, orgID string) error {
	placeholders := strings.Repeat("?,", len(versionIDs))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]interface{}, 0, len(versionIDs)+1)
	for _, id := range versionIDs {
		args = append(args, id)
	}
	args = append(args, orgID)

	rawSQL := fmt.Sprintf(
		`SELECT m.PURPOSE_VERSION_ID, m.ELEMENT_VERSION_ID, e.ID AS ELEMENT_ID, e.NAME, e.NAMESPACE, e.VERSION, m.MANDATORY, e.TYPE, e.ELEMENT_SCHEMA
		 FROM PURPOSE_ELEMENT_MAPPING m
		 JOIN ELEMENT e ON m.ELEMENT_VERSION_ID = e.VERSION_ID
		 WHERE m.PURPOSE_VERSION_ID IN (%s) AND m.ORG_ID = ?`,
		placeholders,
	)
	q := QueryBatchGetPurposeElements
	q.Query = rawSQL
	q.PostgresQuery = dbutils.ConvertToPostgresParams(rawSQL)

	rows, err := dbClient.Query(q, args...)
	if err != nil {
		return err
	}

	byVersionID := make(map[string]int, len(versions))
	for i, v := range versions {
		byVersionID[v.VersionID] = i
		versions[i].Elements = []model.PurposeMappedElement{}
	}
	for _, row := range rows {
		vid := getString(row, "purpose_version_id")
		if idx, ok := byVersionID[vid]; ok {
			versions[idx].Elements = append(versions[idx].Elements, mapToPurposeMappedElement(row))
		}
	}
	return nil
}

// =============================================================================
// Mappers
// =============================================================================

func mapToPurposeVersion(row map[string]interface{}) *model.PurposeVersion {
	if row == nil {
		return nil
	}
	return &model.PurposeVersion{
		VersionID:   getString(row, "version_id"),
		ID:          getString(row, "id"),
		Name:        getString(row, "name"),
		GroupID:     getString(row, "group_id"),
		VersionNum:  getInt(row, "version"),
		DisplayName: getStringPtr(row, "display_name"),
		Description: getStringPtr(row, "description"),
		CreatedTime: getInt64(row, "created_time"),
		OrgID:       getString(row, "org_id"),
	}
}

func mapToPurposeVersionSlice(rows []map[string]interface{}) []model.PurposeVersion {
	versions := make([]model.PurposeVersion, 0, len(rows))
	for _, row := range rows {
		if pv := mapToPurposeVersion(row); pv != nil {
			versions = append(versions, *pv)
		}
	}
	return versions
}

func mapToPurposeMappedElement(row map[string]interface{}) model.PurposeMappedElement {
	return model.PurposeMappedElement{
		ElementVersionID: getString(row, "element_version_id"),
		ElementID:        getString(row, "element_id"),
		Name:             getString(row, "name"),
		Namespace:        getString(row, "namespace"),
		VersionNum:       getInt(row, "version"),
		Mandatory:        getBool(row, "mandatory"),
		ElementType:      getString(row, "type"),
		Schema:           getStringPtr(row, "element_schema"),
	}
}

func mapToPurposeMappedElementSlice(rows []map[string]interface{}) []model.PurposeMappedElement {
	elems := make([]model.PurposeMappedElement, 0, len(rows))
	for _, row := range rows {
		elems = append(elems, mapToPurposeMappedElement(row))
	}
	return elems
}

// =============================================================================
// DB row helpers
// =============================================================================

// purposeLikePattern escapes a string for LIKE search and returns the pattern and ESCAPE clause.
func purposeLikePattern(dbClient provider.DBClientInterface, name string) (pattern, escapeClause string) {
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
	if v, ok := row[key].(int64); ok {
		return v
	}
	return 0
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
