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

package consentelement

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	dbconst "github.com/wso2/openfgc/consent-server/internal/system/database/constants"
	"github.com/wso2/openfgc/consent-server/internal/consentelement/model"
	providermock "github.com/wso2/openfgc/consent-server/tests/mocks/database/providermock"
)

func TestNewConsentElementStore(t *testing.T) {
	s := NewConsentElementStore()
	require.NotNil(t, s)
}

func TestGetString(t *testing.T) {
	cases := []struct {
		name     string
		row      map[string]interface{}
		key      string
		expected string
	}{
		{"string value", map[string]interface{}{"k": "v"}, "k", "v"},
		{"byte slice value", map[string]interface{}{"k": []byte("v")}, "k", "v"},
		{"missing key", map[string]interface{}{"other": "v"}, "k", ""},
		{"nil row", nil, "k", ""},
		{"integer value", map[string]interface{}{"k": 42}, "k", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, getString(tc.row, tc.key))
		})
	}
}

func TestGetInt64(t *testing.T) {
	require.Equal(t, int64(42), getInt64(map[string]interface{}{"k": int64(42)}, "k"))
	require.Equal(t, int64(0), getInt64(map[string]interface{}{"k": "not-int"}, "k"))
	require.Equal(t, int64(0), getInt64(nil, "k"))
}

func TestGetInt(t *testing.T) {
	require.Equal(t, 3, getInt(map[string]interface{}{"k": int64(3)}, "k"))
	require.Equal(t, 5, getInt(map[string]interface{}{"k": int32(5)}, "k"))
	require.Equal(t, 7, getInt(map[string]interface{}{"k": uint32(7)}, "k"))
	require.Equal(t, 0, getInt(map[string]interface{}{"k": "nope"}, "k"))
}

func TestMapToElementVersion(t *testing.T) {
	t.Run("nil row returns nil", func(t *testing.T) {
		require.Nil(t, mapToElementVersion(nil))
	})

	t.Run("complete row", func(t *testing.T) {
		row := map[string]interface{}{
			"version_id":     "vid-1",
			"id":             "elem-1",
			"name":           "email",
			"namespace":      "default",
			"type":           "basic",
			"version":        int64(2),
			"display_name":   "Email Address",
			"description":    "User email",
			"element_schema": nil,
			"created_time":   int64(1700000000),
			"org_id":         "org-1",
		}
		v := mapToElementVersion(row)
		require.NotNil(t, v)
		require.Equal(t, "vid-1", v.VersionID)
		require.Equal(t, "elem-1", v.ID)
		require.Equal(t, "email", v.Name)
		require.Equal(t, "default", v.Namespace)
		require.Equal(t, "basic", v.Type)
		require.Equal(t, 2, v.VersionNum)
		require.NotNil(t, v.DisplayName)
		require.Equal(t, "Email Address", *v.DisplayName)
		require.NotNil(t, v.Description)
		require.Equal(t, "User email", *v.Description)
		require.Nil(t, v.Schema)
		require.Equal(t, int64(1700000000), v.CreatedTime)
		require.Equal(t, "org-1", v.OrgID)
	})

	t.Run("byte slice values", func(t *testing.T) {
		row := map[string]interface{}{
			"version_id": []byte("vid-2"),
			"id":         []byte("elem-2"),
			"name":       []byte("phone"),
			"namespace":  []byte("ns1"),
			"type":       []byte("xml"),
			"version":    int64(1),
			"org_id":     []byte("org-2"),
		}
		v := mapToElementVersion(row)
		require.Equal(t, "vid-2", v.VersionID)
		require.Equal(t, "phone", v.Name)
		require.Equal(t, "ns1", v.Namespace)
	})

	t.Run("empty row returns zero values", func(t *testing.T) {
		v := mapToElementVersion(map[string]interface{}{})
		require.NotNil(t, v)
		require.Equal(t, "", v.ID)
		require.Equal(t, 0, v.VersionNum)
	})
}

// DB query/transaction tests are covered by integration tests.
func TestStoreDBOperations(t *testing.T) {
	t.Skip("Database operation tests covered by integration tests")
}

// --- getStringPtr ---

func TestGetStringPtr(t *testing.T) {
	s := "hello"
	got := getStringPtr(map[string]interface{}{"k": s}, "k")
	require.NotNil(t, got)
	require.Equal(t, "hello", *got)

	b := []byte("world")
	got = getStringPtr(map[string]interface{}{"k": b}, "k")
	require.NotNil(t, got)
	require.Equal(t, "world", *got)

	got = getStringPtr(map[string]interface{}{"k": nil}, "k")
	require.Nil(t, got)

	got = getStringPtr(map[string]interface{}{}, "k")
	require.Nil(t, got)

	got = getStringPtr(map[string]interface{}{"k": 42}, "k")
	require.Nil(t, got)
}

// --- mapToElementVersionSlice ---

func TestMapToElementVersionSlice(t *testing.T) {
	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := mapToElementVersionSlice([]map[string]interface{}{})
		require.NotNil(t, result)
		require.Len(t, result, 0)
	})

	t.Run("valid rows are mapped", func(t *testing.T) {
		rows := []map[string]interface{}{
			{"version_id": "v1", "id": "e1", "name": "email", "namespace": "default", "type": "basic", "version": int64(1), "org_id": "org-1"},
			{"version_id": "v2", "id": "e2", "name": "phone", "namespace": "default", "type": "basic", "version": int64(1), "org_id": "org-1"},
		}
		result := mapToElementVersionSlice(rows)
		require.Len(t, result, 2)
		require.Equal(t, "v1", result[0].VersionID)
		require.Equal(t, "v2", result[1].VersionID)
	})

	t.Run("nil rows are skipped", func(t *testing.T) {
		rows := []map[string]interface{}{
			{"version_id": "v1", "id": "e1", "name": "email", "namespace": "default", "type": "basic", "version": int64(1), "org_id": "org-1"},
			nil,
		}
		result := mapToElementVersionSlice(rows)
		require.Len(t, result, 1)
		require.Equal(t, "v1", result[0].VersionID)
	})
}

// --- likePattern ---

func mockClientWithType(t *testing.T, dbType string) *providermock.DBClientInterface {
	t.Helper()
	m := providermock.NewDBClientInterface(t)
	m.On("GetDBType").Maybe().Return(dbType)
	return m
}

func TestLikePattern(t *testing.T) {
	t.Run("mysql wraps with percent, no escape clause", func(t *testing.T) {
		pattern, escapeClause := likePattern(mockClientWithType(t, dbconst.DatabaseTypeMySQL), "email")
		require.Equal(t, "%email%", pattern)
		require.Equal(t, "", escapeClause)
	})

	t.Run("mysql escapes percent and underscore", func(t *testing.T) {
		pattern, escapeClause := likePattern(mockClientWithType(t, dbconst.DatabaseTypeMySQL), "100%_done")
		require.Equal(t, `%100\%\_done%`, pattern)
		require.Equal(t, "", escapeClause)
	})

	t.Run("sqlite wraps with percent and adds escape clause", func(t *testing.T) {
		pattern, escapeClause := likePattern(mockClientWithType(t, dbconst.DatabaseTypeSQLite), "email")
		require.Equal(t, "%email%", pattern)
		require.Equal(t, " ESCAPE '|'", escapeClause)
	})

	t.Run("sqlite escapes percent, underscore, and pipe", func(t *testing.T) {
		pattern, escapeClause := likePattern(mockClientWithType(t, dbconst.DatabaseTypeSQLite), "a|b%c_d")
		require.Equal(t, "%a||b|%c|_d%", pattern)
		require.Equal(t, " ESCAPE '|'", escapeClause)
	})

	t.Run("postgres behaves same as sqlite", func(t *testing.T) {
		pattern, escapeClause := likePattern(mockClientWithType(t, dbconst.DatabaseTypePostgres), "a%b")
		require.Equal(t, "%a|%b%", pattern)
		require.Equal(t, " ESCAPE '|'", escapeClause)
	})
}

// --- buildListQuery ---

func TestBuildListQuery(t *testing.T) {
	s := &store{}
	orgID := "org-1"

	t.Run("no filters produces latest-version join", func(t *testing.T) {
		dbClient := mockClientWithType(t, dbconst.DatabaseTypeMySQL)
		_, dataQ, dataArgs, _ := s.buildListQuery(dbClient, orgID, model.ElementListFilter{Limit: 10, Offset: 0})
		require.Contains(t, dataQ.Query, "INNER JOIN")
		require.Contains(t, dataQ.Query, "MAX_VERSION")
		require.Contains(t, dataQ.Query, "LIMIT ? OFFSET ?")
		require.Equal(t, orgID, dataArgs[0])
		require.Equal(t, orgID, dataArgs[1])
		require.Equal(t, 10, dataArgs[len(dataArgs)-2])
		require.Equal(t, 0, dataArgs[len(dataArgs)-1])
	})

	t.Run("version filter queries directly without join", func(t *testing.T) {
		dbClient := mockClientWithType(t, dbconst.DatabaseTypeMySQL)
		version := 2
		_, dataQ, dataArgs, _ := s.buildListQuery(dbClient, orgID, model.ElementListFilter{Limit: 5, Offset: 0, Version: &version})
		require.NotContains(t, dataQ.Query, "INNER JOIN")
		require.Contains(t, dataQ.Query, "e.VERSION = ?")
		require.Equal(t, 2, dataArgs[1])
	})

	t.Run("name filter adds LIKE clause", func(t *testing.T) {
		dbClient := mockClientWithType(t, dbconst.DatabaseTypeMySQL)
		_, dataQ, _, _ := s.buildListQuery(dbClient, orgID, model.ElementListFilter{Limit: 10, Name: "email"})
		require.Contains(t, dataQ.Query, "e.NAME LIKE ?")
	})

	t.Run("namespace filter adds equality clause", func(t *testing.T) {
		dbClient := mockClientWithType(t, dbconst.DatabaseTypeMySQL)
		_, dataQ, _, _ := s.buildListQuery(dbClient, orgID, model.ElementListFilter{Limit: 10, Namespace: "default"})
		require.Contains(t, dataQ.Query, "e.NAMESPACE = ?")
	})

	t.Run("type filter adds equality clause", func(t *testing.T) {
		dbClient := mockClientWithType(t, dbconst.DatabaseTypeMySQL)
		_, dataQ, _, _ := s.buildListQuery(dbClient, orgID, model.ElementListFilter{Limit: 10, Type: "basic"})
		require.Contains(t, dataQ.Query, "e.TYPE = ?")
	})

	t.Run("count query wraps data query", func(t *testing.T) {
		dbClient := mockClientWithType(t, dbconst.DatabaseTypeMySQL)
		countQ, _, _, _ := s.buildListQuery(dbClient, orgID, model.ElementListFilter{Limit: 10})
		require.True(t, strings.HasPrefix(countQ.Query, "SELECT COUNT(*) AS cnt FROM ("))
	})

	t.Run("postgres query uses dollar placeholders", func(t *testing.T) {
		dbClient := mockClientWithType(t, dbconst.DatabaseTypePostgres)
		_, dataQ, _, _ := s.buildListQuery(dbClient, orgID, model.ElementListFilter{Limit: 10})
		require.Contains(t, dataQ.PostgresQuery, "$1")
	})
}
