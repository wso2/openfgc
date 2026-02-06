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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wso2/openfgc/internal/consentelement/model"
)

// TestNewConsentElementStore tests store creation
func TestNewConsentElementStore(t *testing.T) {
	store := NewConsentElementStore()
	require.NotNil(t, store)
}

// TestGetString tests the getString helper function
func TestGetString(t *testing.T) {
	testCases := []struct {
		name     string
		row      map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "String value",
			row:      map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "Byte slice value",
			row:      map[string]interface{}{"key": []byte("value")},
			key:      "key",
			expected: "value",
		},
		{
			name:     "Missing key",
			row:      map[string]interface{}{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "Nil row",
			row:      nil,
			key:      "key",
			expected: "",
		},
		{
			name:     "Empty row",
			row:      map[string]interface{}{},
			key:      "key",
			expected: "",
		},
		{
			name:     "Integer value",
			row:      map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "Nil value",
			row:      map[string]interface{}{"key": nil},
			key:      "key",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getString(tc.row, tc.key)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestMapToConsentElement tests mapping database row to ConsentElement
func TestMapToConsentElement(t *testing.T) {
	testCases := []struct {
		name     string
		row      map[string]interface{}
		expected *model.ConsentElement
	}{
		{
			name: "Complete element with description",
			row: map[string]interface{}{
				"id":          "elem-123",
				"name":        "test_element",
				"type":        "basic",
				"org_id":      "org-456",
				"description": "Test description",
			},
			expected: &model.ConsentElement{
				ID:          "elem-123",
				Name:        "test_element",
				Type:        "basic",
				OrgID:       "org-456",
				Description: stringPtr("Test description"),
				Properties:  make(map[string]string),
			},
		},
		{
			name: "Element without description",
			row: map[string]interface{}{
				"id":     "elem-123",
				"name":   "test_element",
				"type":   "basic",
				"org_id": "org-456",
			},
			expected: &model.ConsentElement{
				ID:          "elem-123",
				Name:        "test_element",
				Type:        "basic",
				OrgID:       "org-456",
				Description: nil,
				Properties:  make(map[string]string),
			},
		},
		{
			name: "Element with byte slice values",
			row: map[string]interface{}{
				"id":          []byte("elem-123"),
				"name":        []byte("test_element"),
				"type":        []byte("basic"),
				"org_id":      []byte("org-456"),
				"description": []byte("Test description"),
			},
			expected: &model.ConsentElement{
				ID:          "elem-123",
				Name:        "test_element",
				Type:        "basic",
				OrgID:       "org-456",
				Description: stringPtr("Test description"),
				Properties:  make(map[string]string),
			},
		},
		{
			name: "Element with empty description",
			row: map[string]interface{}{
				"id":          "elem-123",
				"name":        "test_element",
				"type":        "basic",
				"org_id":      "org-456",
				"description": "",
			},
			expected: &model.ConsentElement{
				ID:          "elem-123",
				Name:        "test_element",
				Type:        "basic",
				OrgID:       "org-456",
				Description: nil,
				Properties:  make(map[string]string),
			},
		},
		{
			name:     "Nil row",
			row:      nil,
			expected: nil,
		},
		{
			name: "Empty row",
			row:  map[string]interface{}{},
			expected: &model.ConsentElement{
				ID:          "",
				Name:        "",
				Type:        "",
				OrgID:       "",
				Description: nil,
				Properties:  make(map[string]string),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapToConsentElement(tc.row)

			if tc.expected == nil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, tc.expected.ID, result.ID)
				require.Equal(t, tc.expected.Name, result.Name)
				require.Equal(t, tc.expected.Type, result.Type)
				require.Equal(t, tc.expected.OrgID, result.OrgID)
				require.NotNil(t, result.Properties)

				if tc.expected.Description == nil {
					require.Nil(t, result.Description)
				} else {
					require.NotNil(t, result.Description)
					require.Equal(t, *tc.expected.Description, *result.Description)
				}
			}
		})
	}
}

// TestMapToConsentElementProperty tests mapping database row to ConsentElementProperty
func TestMapToConsentElementProperty(t *testing.T) {
	testCases := []struct {
		name     string
		row      map[string]interface{}
		expected *model.ConsentElementProperty
	}{
		{
			name: "Complete property",
			row: map[string]interface{}{
				"element_id": "elem-456",
				"att_key":    "key1",
				"att_value":  "value1",
				"org_id":     "org-789",
			},
			expected: &model.ConsentElementProperty{
				ElementID: "elem-456",
				Key:       "key1",
				Value:     "value1",
				OrgID:     "org-789",
			},
		},
		{
			name: "Property with byte slice values",
			row: map[string]interface{}{
				"element_id": []byte("elem-456"),
				"att_key":    []byte("key1"),
				"att_value":  []byte("value1"),
				"org_id":     []byte("org-789"),
			},
			expected: &model.ConsentElementProperty{
				ElementID: "elem-456",
				Key:       "key1",
				Value:     "value1",
				OrgID:     "org-789",
			},
		},
		{
			name:     "Nil row",
			row:      nil,
			expected: nil,
		},
		{
			name: "Empty row",
			row:  map[string]interface{}{},
			expected: &model.ConsentElementProperty{

				ElementID: "",
				Key:       "",
				Value:     "",
				OrgID:     "",
			},
		},
		{
			name: "Partial property",
			row: map[string]interface{}{
				"element_id": "elem-456",
				"att_key":    "key1",
			},
			expected: &model.ConsentElementProperty{
				ElementID: "elem-456",
				Key:       "key1",
				Value:     "",
				OrgID:     "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapToConsentElementProperty(tc.row)

			if tc.expected == nil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, tc.expected.ElementID, result.ElementID)
				require.Equal(t, tc.expected.Key, result.Key)
				require.Equal(t, tc.expected.Value, result.Value)
				require.Equal(t, tc.expected.OrgID, result.OrgID)
			}
		})
	}
}

// TestGetIDsByNames_EmptyInput tests GetIDsByNames with empty input
func TestGetIDsByNames_EmptyInput(t *testing.T) {
	// Note: This test doesn't require database mocking as it returns early
	// Database layer tests are covered by integration tests
	t.Skip("Database layer tests covered by integration tests")
}

// TestCreate tests Create function
func TestCreate(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestGetByID tests GetByID function
func TestGetByID(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}

// TestList tests List function
func TestList(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}

// TestUpdate tests Update function
func TestUpdate(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestDelete tests Delete function
func TestDelete(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestCheckNameExists tests CheckNameExists function
func TestCheckNameExists(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}

// TestCreateProperties tests CreateProperties function
func TestCreateProperties(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}

// TestGetPropertiesByElementID tests GetPropertiesByElementID function
func TestGetPropertiesByElementID(t *testing.T) {
	// Database query tests are covered by integration tests
	t.Skip("Database query tests covered by integration tests")
}

// TestDeleteProperties tests DeleteProperties function
func TestDeleteProperties(t *testing.T) {
	// Database transaction tests are covered by integration tests
	t.Skip("Database transaction tests covered by integration tests")
}
