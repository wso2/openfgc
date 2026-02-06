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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wso2/openfgc/internal/consentpurpose/model"
)

// TestNewPurposeStore tests store creation
func TestNewPurposeStore(t *testing.T) {
	store := NewPurposeStore()
	require.NotNil(t, store)
}

// TestMapRowToPurpose tests mapping database row to ConsentPurpose
func TestMapRowToPurpose(t *testing.T) {
	testCases := []struct {
		name     string
		row      map[string]interface{}
		expected model.ConsentPurpose
	}{
		{
			name: "Complete purpose with description",
			row: map[string]interface{}{
				"id":           []uint8("purpose-123"),
				"name":         []uint8("Test Purpose"),
				"description":  []uint8("Test Description"),
				"client_id":    []uint8("client-456"),
				"created_time": int64(1234567890),
				"updated_time": int64(1234567900),
				"org_id":       []uint8("org-789"),
			},
			expected: func() model.ConsentPurpose {
				desc := "Test Description"
				return model.ConsentPurpose{
					ID:          "purpose-123",
					Name:        "Test Purpose",
					Description: &desc,
					ClientID:    "client-456",
					CreatedTime: 1234567890,
					UpdatedTime: 1234567900,
					OrgID:       "org-789",
				}
			}(),
		},
		{
			name: "Purpose without description",
			row: map[string]interface{}{
				"id":           []uint8("purpose-123"),
				"name":         []uint8("Test Purpose"),
				"description":  nil,
				"client_id":    []uint8("client-456"),
				"created_time": int64(1234567890),
				"updated_time": int64(1234567900),
				"org_id":       []uint8("org-789"),
			},
			expected: model.ConsentPurpose{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: nil,
				ClientID:    "client-456",
				CreatedTime: 1234567890,
				UpdatedTime: 1234567900,
				OrgID:       "org-789",
			},
		},
		{
			name: "Empty row",
			row:  map[string]interface{}{},
			expected: model.ConsentPurpose{
				ID:          "",
				Name:        "",
				Description: nil,
				ClientID:    "",
				CreatedTime: 0,
				UpdatedTime: 0,
				OrgID:       "",
			},
		},
		{
			name: "Partial purpose",
			row: map[string]interface{}{
				"id":        []uint8("purpose-123"),
				"name":      []uint8("Test Purpose"),
				"client_id": []uint8("client-456"),
				"org_id":    []uint8("org-789"),
			},
			expected: model.ConsentPurpose{
				ID:          "purpose-123",
				Name:        "Test Purpose",
				Description: nil,
				ClientID:    "client-456",
				CreatedTime: 0,
				UpdatedTime: 0,
				OrgID:       "org-789",
			},
		},
		{
			name: "Purpose with wrong type fields",
			row: map[string]interface{}{
				"id":           123,
				"name":         456,
				"description":  true,
				"client_id":    []uint8("client-456"),
				"created_time": "not-a-number",
				"updated_time": int64(1234567900),
				"org_id":       []uint8("org-789"),
			},
			expected: model.ConsentPurpose{
				ID:          "",
				Name:        "",
				Description: nil,
				ClientID:    "client-456",
				CreatedTime: 0,
				UpdatedTime: 1234567900,
				OrgID:       "org-789",
			},
		},
	}

	s := &store{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := s.mapRowToPurpose(tc.row)

			require.Equal(t, tc.expected.ID, result.ID)
			require.Equal(t, tc.expected.Name, result.Name)
			require.Equal(t, tc.expected.ClientID, result.ClientID)
			require.Equal(t, tc.expected.CreatedTime, result.CreatedTime)
			require.Equal(t, tc.expected.UpdatedTime, result.UpdatedTime)
			require.Equal(t, tc.expected.OrgID, result.OrgID)

			if tc.expected.Description == nil {
				require.Nil(t, result.Description)
			} else {
				require.NotNil(t, result.Description)
				require.Equal(t, *tc.expected.Description, *result.Description)
			}
		})
	}
}

// TestBuildListPurposesQuery tests dynamic query building for listing purposes
func TestBuildListPurposesQuery(t *testing.T) {
	testCases := []struct {
		name              string
		orgID             string
		purposeName       string
		clientIDs         []string
		elementNames      []string
		expectedQueryPart string
		expectedArgsCount int
	}{
		{
			name:              "No filters",
			orgID:             "org-123",
			purposeName:       "",
			clientIDs:         nil,
			elementNames:      nil,
			expectedQueryPart: "WHERE ORG_ID = ?",
			expectedArgsCount: 1,
		},
		{
			name:              "Filter by name only",
			orgID:             "org-123",
			purposeName:       "Test Purpose",
			clientIDs:         nil,
			elementNames:      nil,
			expectedQueryPart: "AND NAME = ?",
			expectedArgsCount: 2,
		},
		{
			name:              "Filter by single client ID",
			orgID:             "org-123",
			purposeName:       "",
			clientIDs:         []string{"client-1"},
			elementNames:      nil,
			expectedQueryPart: "AND CLIENT_ID IN (?)",
			expectedArgsCount: 2,
		},
		{
			name:              "Filter by multiple client IDs",
			orgID:             "org-123",
			purposeName:       "",
			clientIDs:         []string{"client-1", "client-2", "client-3"},
			elementNames:      nil,
			expectedQueryPart: "AND CLIENT_ID IN (?,?,?)",
			expectedArgsCount: 4,
		},
		{
			name:              "Filter by single element name",
			orgID:             "org-123",
			purposeName:       "",
			clientIDs:         nil,
			elementNames:      []string{"element-1"},
			expectedQueryPart: "AND EXISTS",
			expectedArgsCount: 2,
		},
		{
			name:              "Filter by multiple element names",
			orgID:             "org-123",
			purposeName:       "",
			clientIDs:         nil,
			elementNames:      []string{"element-1", "element-2"},
			expectedQueryPart: "AND EXISTS",
			expectedArgsCount: 3,
		},
		{
			name:              "All filters combined",
			orgID:             "org-123",
			purposeName:       "Test Purpose",
			clientIDs:         []string{"client-1", "client-2"},
			elementNames:      []string{"element-1"},
			expectedQueryPart: "AND NAME = ?",
			expectedArgsCount: 5,
		},
	}

	s := &store{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query, countQuery, args, countArgs := s.buildListPurposesQuery(tc.orgID, tc.purposeName, tc.clientIDs, tc.elementNames)

			// Verify query contains expected parts
			require.Contains(t, query, "WHERE ORG_ID = ?")
			require.Contains(t, countQuery, "WHERE ORG_ID = ?")

			if tc.expectedQueryPart != "" {
				require.Contains(t, query, tc.expectedQueryPart)
				require.Contains(t, countQuery, tc.expectedQueryPart)
			}

			// Verify argument count
			require.Len(t, args, tc.expectedArgsCount)
			require.Len(t, countArgs, tc.expectedArgsCount)

			// Verify first argument is always orgID
			require.Equal(t, tc.orgID, args[0])
			require.Equal(t, tc.orgID, countArgs[0])

			// Verify name filter if provided
			if tc.purposeName != "" {
				require.Contains(t, args, tc.purposeName)
				require.Contains(t, countArgs, tc.purposeName)
			}

			// Verify client IDs if provided
			for _, clientID := range tc.clientIDs {
				require.Contains(t, args, clientID)
				require.Contains(t, countArgs, clientID)
			}

			// Verify element names if provided
			for _, elementName := range tc.elementNames {
				require.Contains(t, args, elementName)
				require.Contains(t, countArgs, elementName)
			}
		})
	}
}

// TestBuildListPurposesQuery_BaseQueries tests that base queries are used correctly
func TestBuildListPurposesQuery_BaseQueries(t *testing.T) {
	s := &store{}

	query, countQuery, _, _ := s.buildListPurposesQuery("org-123", "", nil, nil)

	// Verify base queries are included
	require.Contains(t, query, "SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID")
	require.Contains(t, query, "FROM CONSENT_PURPOSE")
	require.Contains(t, countQuery, "SELECT COUNT(*) as count FROM CONSENT_PURPOSE")
}

// TestCreatePurpose tests CreatePurpose function
func TestCreatePurpose(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestGetPurposeByID tests GetPurposeByID function
func TestGetPurposeByID(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}

// TestListPurposes tests ListPurposes function
func TestListPurposes(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}

// TestUpdatePurpose tests UpdatePurpose function
func TestUpdatePurpose(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestDeletePurpose tests DeletePurpose function
func TestDeletePurpose(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestCheckPurposeNameExists tests CheckPurposeNameExists function
func TestCheckPurposeNameExists(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}

// TestLinkElementToPurpose tests LinkElementToPurpose function
func TestLinkElementToPurpose(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestGetPurposeElements tests GetPurposeElements function
func TestGetPurposeElements(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}

// TestDeletePurposeElements tests DeletePurposeElements function
func TestDeletePurposeElements(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestIsElementUsedInPurposes tests IsElementUsedInPurposes function
func TestIsElementUsedInPurposes(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}
