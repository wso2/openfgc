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

	"github.com/wso2/openfgc/consent-server/internal/consentelement/model"
	dbconst "github.com/wso2/openfgc/consent-server/internal/system/database/constants"
	dbmodel "github.com/wso2/openfgc/consent-server/internal/system/database/model"
	"github.com/wso2/openfgc/consent-server/internal/system/database/provider"
	dbutils "github.com/wso2/openfgc/consent-server/internal/system/database/utils"
	"github.com/wso2/openfgc/consent-server/internal/system/stores/interfaces"
)

// elementColumns is the SELECT column list shared across all ELEMENT queries.
const elementColumns = "VERSION_ID, ID, NAME, NAMESPACE, TYPE, VERSION, DISPLAY_NAME, DESCRIPTION, ELEMENT_SCHEMA, CREATED_TIME, ORG_ID"

// Pre-defined DBQuery objects for simple, single-path operations.
var (
	QueryInsertElementVersion = dbmodel.DBQuery{
		ID:            "INSERT_ELEMENT_VERSION",
		Query:         "INSERT INTO ELEMENT (VERSION_ID, ID, NAME, NAMESPACE, TYPE, VERSION, DISPLAY_NAME, DESCRIPTION, ELEMENT_SCHEMA, CREATED_TIME, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO ELEMENT (VERSION_ID, ID, NAME, NAMESPACE, TYPE, VERSION, DISPLAY_NAME, DESCRIPTION, ELEMENT_SCHEMA, CREATED_TIME, ORG_ID) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
	}

	QueryInsertElementProperty = dbmodel.DBQuery{
		ID:            "INSERT_ELEMENT_PROPERTY",
		Query:         "INSERT INTO ELEMENT_PROPERTY (ELEMENT_VERSION_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES (?, ?, ?, ?)",
		PostgresQuery: "INSERT INTO ELEMENT_PROPERTY (ELEMENT_VERSION_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES ($1, $2, $3, $4)",
	}

	QueryGetLatestElementVersion = dbmodel.DBQuery{
		ID:            "GET_LATEST_ELEMENT_VERSION",
		Query:         "SELECT " + elementColumns + " FROM ELEMENT WHERE ID = ? AND ORG_ID = ? ORDER BY VERSION DESC LIMIT 1",
		PostgresQuery: "SELECT " + elementColumns + " FROM ELEMENT WHERE ID = $1 AND ORG_ID = $2 ORDER BY VERSION DESC LIMIT 1",
	}

	QueryGetElementVersion = dbmodel.DBQuery{
		ID:            "GET_ELEMENT_VERSION",
		Query:         "SELECT " + elementColumns + " FROM ELEMENT WHERE ID = ? AND VERSION = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT " + elementColumns + " FROM ELEMENT WHERE ID = $1 AND VERSION = $2 AND ORG_ID = $3",
	}

	QueryListElementVersions = dbmodel.DBQuery{
		ID:            "LIST_ELEMENT_VERSIONS",
		Query:         "SELECT " + elementColumns + " FROM ELEMENT WHERE ID = ? AND ORG_ID = ? ORDER BY VERSION ASC",
		PostgresQuery: "SELECT " + elementColumns + " FROM ELEMENT WHERE ID = $1 AND ORG_ID = $2 ORDER BY VERSION ASC",
	}

	QueryGetElementByNameAndNamespace = dbmodel.DBQuery{
		ID:            "GET_ELEMENT_BY_NAME_AND_NAMESPACE",
		Query:         "SELECT " + elementColumns + " FROM ELEMENT WHERE NAME = ? AND NAMESPACE = ? AND ORG_ID = ? ORDER BY VERSION DESC LIMIT 1",
		PostgresQuery: "SELECT " + elementColumns + " FROM ELEMENT WHERE NAME = $1 AND NAMESPACE = $2 AND ORG_ID = $3 ORDER BY VERSION DESC LIMIT 1",
	}

	QueryElementExists = dbmodel.DBQuery{
		ID:            "ELEMENT_EXISTS",
		Query:         "SELECT COUNT(*) AS cnt FROM ELEMENT WHERE ID = ? AND ORG_ID = ? LIMIT 1",
		PostgresQuery: "SELECT COUNT(*) AS cnt FROM ELEMENT WHERE ID = $1 AND ORG_ID = $2 LIMIT 1",
	}

	QueryDeleteElementVersion = dbmodel.DBQuery{
		ID:            "DELETE_ELEMENT_VERSION",
		Query:         "DELETE FROM ELEMENT WHERE VERSION_ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM ELEMENT WHERE VERSION_ID = $1 AND ORG_ID = $2",
	}

	QueryDeleteElement = dbmodel.DBQuery{
		ID:            "DELETE_ELEMENT",
		Query:         "DELETE FROM ELEMENT WHERE ID = ? AND ORG_ID = ?",
		PostgresQuery: "DELETE FROM ELEMENT WHERE ID = $1 AND ORG_ID = $2",
	}

	QueryGetPropertiesByVersionID = dbmodel.DBQuery{
		ID:            "GET_ELEMENT_PROPERTIES_BY_VERSION_ID",
		Query:         "SELECT ELEMENT_VERSION_ID, ATT_KEY, ATT_VALUE FROM ELEMENT_PROPERTY WHERE ELEMENT_VERSION_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT ELEMENT_VERSION_ID, ATT_KEY, ATT_VALUE FROM ELEMENT_PROPERTY WHERE ELEMENT_VERSION_ID = $1 AND ORG_ID = $2",
	}

	QueryIsVersionReferencedByPurpose = dbmodel.DBQuery{
		ID:            "IS_ELEMENT_VERSION_REFERENCED_BY_PURPOSE",
		Query:         "SELECT COUNT(*) AS cnt FROM PURPOSE_ELEMENT_MAPPING WHERE ELEMENT_VERSION_ID = ? AND ORG_ID = ?",
		PostgresQuery: "SELECT COUNT(*) AS cnt FROM PURPOSE_ELEMENT_MAPPING WHERE ELEMENT_VERSION_ID = $1 AND ORG_ID = $2",
	}

	// Stubs for queries built dynamically at runtime.
	QueryBatchGetElementProperties = dbmodel.DBQuery{
		ID:    "BATCH_GET_ELEMENT_PROPERTIES",
		Query: "", // Built dynamically with IN clause based on batch size
	}
	QueryListElementsCount = dbmodel.DBQuery{
		ID:    "LIST_ELEMENTS_COUNT_DYNAMIC",
		Query: "", // Built dynamically based on list filters
	}
	QueryListElementsData = dbmodel.DBQuery{
		ID:    "LIST_ELEMENTS_DATA_DYNAMIC",
		Query: "", // Built dynamically based on list filters
	}
)

// store implements interfaces.ConsentElementStore.
type store struct{}

// NewConsentElementStore creates a new consent element store.
func NewConsentElementStore() interfaces.ConsentElementStore {
	return &store{}
}

func (s *store) getDBClient() (provider.DBClientInterface, error) {
	return provider.GetDBProvider().GetConsentDBClient()
}

// CreateVersion inserts a new element version row and its properties within a transaction.
func (s *store) CreateVersion(tx dbmodel.TxInterface, elementVersion *model.ElementVersion) error {
	_, err := tx.Exec(QueryInsertElementVersion,
		elementVersion.VersionID, elementVersion.ID, elementVersion.Name, elementVersion.Namespace, elementVersion.Type, elementVersion.VersionNum,
		elementVersion.DisplayName, elementVersion.Description, elementVersion.Schema, elementVersion.CreatedTime, elementVersion.OrgID,
	)
	if err != nil {
		return err
	}
	for k, val := range elementVersion.Properties {
		if _, err := tx.Exec(QueryInsertElementProperty, elementVersion.VersionID, k, val, elementVersion.OrgID); err != nil {
			return err
		}
	}
	return nil
}

// GetLatestVersion returns the highest-numbered version of an element, with properties.
func (s *store) GetLatestVersion(ctx context.Context, elementID, orgID string) (*model.ElementVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetLatestElementVersion, elementID, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	elementVersion := mapToElementVersion(rows[0])
	elementVersion.Properties, err = s.fetchProperties(dbClient, elementVersion.VersionID, orgID)
	return elementVersion, err
}

// GetVersion returns a specific version by version number, with properties.
func (s *store) GetVersion(ctx context.Context, elementID string, version int, orgID string) (*model.ElementVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetElementVersion, elementID, version, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	elementVersion := mapToElementVersion(rows[0])
	elementVersion.Properties, err = s.fetchProperties(dbClient, elementVersion.VersionID, orgID)
	return elementVersion, err
}

// ListVersions returns all versions of one element ordered ascending, with properties for each.
func (s *store) ListVersions(ctx context.Context, elementID, orgID string) ([]model.ElementVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryListElementVersions, elementID, orgID)
	if err != nil {
		return nil, err
	}
	versions := mapToElementVersionSlice(rows)
	if err := s.populateProperties(dbClient, versions, orgID); err != nil {
		return nil, err
	}
	return versions, nil
}

// GetByNameAndNamespace returns the latest version of an element with the given name and namespace.
// Returns nil if not found. Properties are not populated (used for existence checks only).
func (s *store) GetByNameAndNamespace(ctx context.Context, name, namespace, orgID string) (*model.ElementVersion, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryGetElementByNameAndNamespace, name, namespace, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return mapToElementVersion(rows[0]), nil
}

// ElementExists reports whether any version of the element exists.
func (s *store) ElementExists(ctx context.Context, elementID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryElementExists, elementID, orgID)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return getInt64(rows[0], "cnt") > 0, nil
}

// DeleteVersion deletes a specific version row. ELEMENT_PROPERTY rows cascade automatically.
func (s *store) DeleteVersion(tx dbmodel.TxInterface, versionID, orgID string) error {
	_, err := tx.Exec(QueryDeleteElementVersion, versionID, orgID)
	return err
}

// DeleteElement deletes all versions of an element. Called when the last version is removed.
func (s *store) DeleteElement(tx dbmodel.TxInterface, elementID, orgID string) error {
	_, err := tx.Exec(QueryDeleteElement, elementID, orgID)
	return err
}

// IsVersionReferencedByPurpose reports whether any purpose version references this element version.
func (s *store) IsVersionReferencedByPurpose(ctx context.Context, versionID, orgID string) (bool, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return false, fmt.Errorf("failed to get database client: %w", err)
	}
	rows, err := dbClient.Query(QueryIsVersionReferencedByPurpose, versionID, orgID)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return getInt64(rows[0], "cnt") > 0, nil
}

// List returns the latest version of each element matching the filters, with total count.
// When filters.Details is false, Schema and Properties are not populated.
func (s *store) List(ctx context.Context, orgID string, filters model.ElementListFilter) ([]model.ElementVersion, int, error) {
	dbClient, err := s.getDBClient()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get database client: %w", err)
	}

	countQuery, dataQuery, args, countArgs := s.buildListQuery(dbClient, orgID, filters)

	countRows, err := dbClient.Query(countQuery, countArgs...)
	if err != nil {
		return nil, 0, err
	}
	total := 0
	if len(countRows) > 0 {
		total = int(getInt64(countRows[0], "cnt"))
	}

	rows, err := dbClient.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	versions := mapToElementVersionSlice(rows)

	if filters.Details && len(versions) > 0 {
		if err := s.populateProperties(dbClient, versions, orgID); err != nil {
			return nil, 0, err
		}
	} else {
		for i := range versions {
			versions[i].Schema = nil
		}
	}

	return versions, total, nil
}

// buildListQuery constructs the count and data queries for List based on the provided filters.
// When filters.Version is nil, results are limited to the latest version per element.
// When filters.Version is set, that exact version is queried directly.
func (s *store) buildListQuery(dbClient provider.DBClientInterface, orgID string, filters model.ElementListFilter) (countQ, dataQ dbmodel.DBQuery, dataArgs, countArgs []interface{}) {
	isVersionFiltered := filters.Version != nil

	var sb strings.Builder
	var whereClauses []string
	var baseArgs []interface{}

	if isVersionFiltered {
		// Direct version filter: query ELEMENT rows matching the given version.
		sb.WriteString("SELECT " + elementColumns + " FROM ELEMENT e WHERE e.ORG_ID = ?")
		baseArgs = append(baseArgs, orgID)
		whereClauses = append(whereClauses, "e.VERSION = ?")
		baseArgs = append(baseArgs, *filters.Version)
	} else {
		// No version filter: join with subquery to get latest version per element.
		sb.WriteString("SELECT e." + strings.ReplaceAll(elementColumns, ", ", ", e.") +
			" FROM ELEMENT e" +
			" INNER JOIN (SELECT ID, MAX(VERSION) AS MAX_VERSION FROM ELEMENT WHERE ORG_ID = ? GROUP BY ID) AS latest" +
			" ON e.ID = latest.ID AND e.VERSION = latest.MAX_VERSION" +
			" WHERE e.ORG_ID = ?")
		baseArgs = append(baseArgs, orgID, orgID)
	}

	if filters.Name != "" {
		namePattern, escapeClause := likePattern(dbClient, filters.Name)
		whereClauses = append(whereClauses, "e.NAME LIKE ?"+escapeClause)
		baseArgs = append(baseArgs, namePattern)
	}
	if filters.Namespace != "" {
		whereClauses = append(whereClauses, "e.NAMESPACE = ?")
		baseArgs = append(baseArgs, filters.Namespace)
	}
	if filters.Type != "" {
		whereClauses = append(whereClauses, "e.TYPE = ?")
		baseArgs = append(baseArgs, filters.Type)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " AND " + strings.Join(whereClauses, " AND ")
	}

	baseSQL := sb.String() + whereSQL

	countSQL := "SELECT COUNT(*) AS cnt FROM (" + baseSQL + ") AS filtered"
	dataSQL := baseSQL + " ORDER BY e.NAME ASC, e.NAMESPACE ASC, e.ID ASC LIMIT ? OFFSET ?"

	dataArgs = append(baseArgs, filters.Limit, filters.Offset)
	countArgs = baseArgs

	countQ = QueryListElementsCount
	countQ.Query = countSQL
	countQ.PostgresQuery = dbutils.ConvertToPostgresParams(countSQL)
	dataQ = QueryListElementsData
	dataQ.Query = dataSQL
	dataQ.PostgresQuery = dbutils.ConvertToPostgresParams(dataSQL)
	return countQ, dataQ, dataArgs, countArgs
}

// likePattern escapes a name string for LIKE search and returns the pattern and any ESCAPE clause.
func likePattern(dbClient provider.DBClientInterface, name string) (pattern, escapeClause string) {
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

// fetchProperties loads properties for a single element version.
func (s *store) fetchProperties(dbClient provider.DBClientInterface, versionID, orgID string) (map[string]string, error) {
	rows, err := dbClient.Query(QueryGetPropertiesByVersionID, versionID, orgID)
	if err != nil {
		return nil, err
	}
	props := make(map[string]string, len(rows))
	for _, row := range rows {
		props[getString(row, "att_key")] = getString(row, "att_value")
	}
	return props, nil
}

// populateProperties batch-fetches properties for a slice of element versions and fills in each
// version's Properties map. One DB round-trip for all versions.
func (s *store) populateProperties(dbClient provider.DBClientInterface, versions []model.ElementVersion, orgID string) error {
	if len(versions) == 0 {
		return nil
	}

	versionIDs := make([]string, len(versions))
	for i, v := range versions {
		versionIDs[i] = v.VersionID
	}

	placeholders := strings.Repeat("?,", len(versionIDs))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]interface{}, 0, len(versionIDs)+1)
	for _, id := range versionIDs {
		args = append(args, id)
	}
	args = append(args, orgID)

	rawSQL := fmt.Sprintf(
		"SELECT ELEMENT_VERSION_ID, ATT_KEY, ATT_VALUE FROM ELEMENT_PROPERTY WHERE ELEMENT_VERSION_ID IN (%s) AND ORG_ID = ?",
		placeholders,
	)
	q := QueryBatchGetElementProperties
	q.Query = rawSQL
	q.PostgresQuery = dbutils.ConvertToPostgresParams(rawSQL)

	rows, err := dbClient.Query(q, args...)
	if err != nil {
		return err
	}

	// Index versions by VersionID for O(1) property assignment.
	byVersionID := make(map[string]int, len(versions))
	for i, v := range versions {
		byVersionID[v.VersionID] = i
		versions[i].Properties = make(map[string]string)
	}
	for _, row := range rows {
		vid := getString(row, "element_version_id")
		if idx, ok := byVersionID[vid]; ok {
			versions[idx].Properties[getString(row, "att_key")] = getString(row, "att_value")
		}
	}
	return nil
}

// mapToElementVersion converts a DB row map to an ElementVersion.
// DBClient normalizes column names to lowercase.
func mapToElementVersion(row map[string]interface{}) *model.ElementVersion {
	if row == nil {
		return nil
	}
	return &model.ElementVersion{
		VersionID:   getString(row, "version_id"),
		ID:          getString(row, "id"),
		Name:        getString(row, "name"),
		Namespace:   getString(row, "namespace"),
		Type:        getString(row, "type"),
		VersionNum:  getInt(row, "version"),
		DisplayName: getStringPtr(row, "display_name"),
		Description: getStringPtr(row, "description"),
		Schema:      getStringPtr(row, "element_schema"),
		CreatedTime: getInt64(row, "created_time"),
		OrgID:       getString(row, "org_id"),
	}
}

// mapToElementVersionSlice converts a slice of DB rows to ElementVersion values.
func mapToElementVersionSlice(rows []map[string]interface{}) []model.ElementVersion {
	versions := make([]model.ElementVersion, 0, len(rows))
	for _, row := range rows {
		if elementVersion := mapToElementVersion(row); elementVersion != nil {
			versions = append(versions, *elementVersion)
		}
	}
	return versions
}

// getString extracts a string from a DB row, handling both string and []byte driver values.
func getString(row map[string]interface{}, key string) string {
	switch v := row[key].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	return ""
}

// getStringPtr returns a pointer to a string extracted from a DB row, or nil if absent/NULL.
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

// getInt64 extracts an int64 from a DB row.
func getInt64(row map[string]interface{}, key string) int64 {
	if v, ok := row[key].(int64); ok {
		return v
	}
	return 0
}

// getInt extracts an int from a DB row, handling int64 and uint32 driver types.
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
