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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/tests/integration/testutils"
)

// ========================================
// GET /consent-elements/{elementId} Tests
// ========================================

// TestGetElementByID_StringType_ReturnsWithValue tests retrieving a string type element with value property
func (ts *ElementAPITestSuite) TestGetElementByID_StringType_ReturnsWithValue() {
	t := ts.T()

	// Create element
	payload := []ConsentElementCreateRequest{
		{
			Name:        "test_license_read_get",
			Description: "License read permission",
			Type:        "basic",
			Properties: map[string]string{
				"value": "license:read",
			},
		},
	}

	resp, body := ts.createElement(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create element: %s", body)

	var createResp ElementCreateResponse
	err := json.Unmarshal([]byte(body), &createResp)
	require.NoError(t, err, "Failed to parse create response")
	require.Len(t, createResp.Data, 1, "Expected 1 element created")

	elementID := createResp.Data[0].ID
	ts.trackElement(elementID)

	// Get the element
	resp, body = ts.getElement(elementID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get element: %s", body)

	var getResp ElementResponse
	err = json.Unmarshal([]byte(body), &getResp)
	require.NoError(t, err, "Failed to parse get response")

	// Verify all fields
	require.Equal(t, elementID, getResp.ID, "ID mismatch")
	require.Equal(t, "test_license_read_get", getResp.Name, "Name mismatch")
	require.NotNil(t, getResp.Description, "Description should not be nil")
	require.Equal(t, "License read permission", *getResp.Description, "Description mismatch")
	require.Equal(t, "basic", getResp.Type, "Type mismatch")
	require.NotNil(t, getResp.Properties, "Properties should not be nil")
	require.Equal(t, "license:read", getResp.Properties["value"], "Value property mismatch")
}

// TestGetElementByID_JsonPayloadType_ReturnsWithValidationSchema tests retrieving json-payload element
func (ts *ElementAPITestSuite) TestGetElementByID_JsonPayloadType_ReturnsWithValidationSchema() {
	t := ts.T()

	validationSchema := `{"type":"object","properties":{"accountNumber":{"type":"string"}}}`

	payload := []ConsentElementCreateRequest{
		{
			Name:        "test_account_schema_get",
			Description: "Account schema validation",
			Type:        "json-payload",
			Properties: map[string]string{
				"validationSchema": validationSchema,
			},
		},
	}

	resp, body := ts.createElement(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create element: %s", body)

	var createResp ElementCreateResponse
	err := json.Unmarshal([]byte(body), &createResp)
	require.NoError(t, err)
	require.Len(t, createResp.Data, 1)

	elementID := createResp.Data[0].ID
	ts.trackElement(elementID)

	// Get and verify
	resp, body = ts.getElement(elementID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get element: %s", body)

	var getResp ElementResponse
	err = json.Unmarshal([]byte(body), &getResp)
	require.NoError(t, err)

	require.Equal(t, elementID, getResp.ID)
	require.Equal(t, "test_account_schema_get", getResp.Name)
	require.Equal(t, "json-payload", getResp.Type)
	require.NotNil(t, getResp.Properties["validationSchema"])
}

// TestGetElementByID_ResourceFieldType_ReturnsWithBothPaths tests retrieving resource-field with both paths
func (ts *ElementAPITestSuite) TestGetElementByID_ResourceFieldType_ReturnsWithBothPaths() {
	t := ts.T()

	payload := []ConsentElementCreateRequest{
		{
			Name:        "test_first_name_get",
			Description: "First name field",
			Type:        "resource-field",
			Properties: map[string]string{
				"resourcePath": "/users",
				"jsonPath":     "$.firstName",
			},
		},
	}

	resp, body := ts.createElement(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create element: %s", body)

	var createResp ElementCreateResponse
	err := json.Unmarshal([]byte(body), &createResp)
	require.NoError(t, err)

	elementID := createResp.Data[0].ID
	ts.trackElement(elementID)

	// Get and verify both paths present
	resp, body = ts.getElement(elementID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get element: %s", body)

	var getResp ElementResponse
	err = json.Unmarshal([]byte(body), &getResp)
	require.NoError(t, err)

	require.Equal(t, "resource-field", getResp.Type)
	require.Equal(t, "/users", getResp.Properties["resourcePath"], "ResourcePath missing")
	require.Equal(t, "$.firstName", getResp.Properties["jsonPath"], "JsonPath missing")
}

// TestGetElementByID_NonExistent_ReturnsNotFound tests getting non-existent element
func (ts *ElementAPITestSuite) TestGetElementByID_NonExistent_ReturnsNotFound() {
	t := ts.T()

	nonExistentID := "00000000-0000-0000-0000-000000000000"
	resp, body := ts.getElement(nonExistentID)

	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404 for non-existent element")

	var errResp ErrorResponse
	err := json.Unmarshal([]byte(body), &errResp)
	require.NoError(t, err)
	require.Equal(t, "CE-1016", errResp.Code)
	require.Contains(t, strings.ToLower(errResp.Description), "not found")
}

// TestGetElementByID_AfterDelete_ReturnsNotFound tests getting deleted element returns 404
func (ts *ElementAPITestSuite) TestGetElementByID_AfterDelete_ReturnsNotFound() {
	t := ts.T()

	// Create element
	payload := []ConsentElementCreateRequest{
		{
			Name:        "test_to_delete",
			Description: "Will be deleted",
			Type:        "basic",
		},
	}

	resp, body := ts.createElement(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	elementID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deleteElementWithCheck(elementID)
	require.True(t, deleted, "Failed to delete element")

	// Try to get - should be 404
	resp, body = ts.getElement(elementID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404 after deletion")

	var errResp ErrorResponse
	json.Unmarshal([]byte(body), &errResp)
	require.Equal(t, "CE-1016", errResp.Code)
}

// TestGetElementByID_ErrorCases tests error scenarios for GET by ID
func (ts *ElementAPITestSuite) TestGetElementByID_ErrorCases() {
	testCases := []struct {
		name            string
		elementID       string
		setOrgHeader    bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name:            "MissingOrgID_ReturnsValidationError",
			elementID:       "00000000-0000-0000-0000-000000000000",
			setOrgHeader:    false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1003",
			messageContains: "organization ID is required",
		},
		{
			name:            "InvalidUUIDFormat_ReturnsNotFound",
			elementID:       "invalid-uuid-format",
			setOrgHeader:    true,
			expectedStatus:  http.StatusNotFound,
			expectedCode:    "CE-1016",
			messageContains: "not found",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-elements/%s", baseURL, tc.elementID), nil)

			if tc.setOrgHeader {
				req.Header.Set(testutils.HeaderOrgID, testOrgID)
				req.Header.Set(testutils.HeaderClientID, testClientID)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch for %s", tc.name)

			var errResp ErrorResponse
			json.NewDecoder(resp.Body).Decode(&errResp)
			require.Equal(t, tc.expectedCode, errResp.Code, "Error code mismatch")
			require.Contains(t, strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains),
				"Error message should contain '%s', got: %s", tc.messageContains, errResp.Description)
		})
	}
}

// ========================================
// GET /consent-elements (LIST) Tests
// ========================================

// TestListElements_DefaultPagination_ReturnsAllElements tests listing with default pagination
func (ts *ElementAPITestSuite) TestListElements_DefaultPagination_ReturnsAllElements() {
	t := ts.T()

	// Create 3 elements
	payload := []ConsentElementCreateRequest{
		{Name: "test_list_1", Description: "First element", Type: "basic"},
		{Name: "test_list_2", Description: "Second element", Type: "basic"},
		{Name: "test_list_3", Description: "Third element", Type: "basic"},
	}

	resp, body := ts.createElement(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create elements: %s", body)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// List all elements
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-elements", baseURL), nil)
	req.Header.Set("org-id", testOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp ElementListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	// Verify metadata exists
	require.NotNil(t, listResp.Metadata, "Metadata should be present")
	require.GreaterOrEqual(t, listResp.Metadata.Total, 3, "Should have at least 3 elements")
	require.Equal(t, 0, listResp.Metadata.Offset, "Default offset should be 0")
	require.Equal(t, 100, listResp.Metadata.Limit, "Default limit should be 100")
	require.GreaterOrEqual(t, listResp.Metadata.Count, 3, "Count should be at least 3")

	// Verify data array
	require.NotEmpty(t, listResp.Data, "Data array should not be empty")
}

// TestListElements_WithLimit_ReturnsPaginatedResults tests pagination with custom limit
func (ts *ElementAPITestSuite) TestListElements_WithLimit_ReturnsPaginatedResults() {
	t := ts.T()

	// Create 5 elements
	elements := []ConsentElementCreateRequest{
		{Name: "test_page_1", Description: "Page test 1", Type: "basic"},
		{Name: "test_page_2", Description: "Page test 2", Type: "basic"},
		{Name: "test_page_3", Description: "Page test 3", Type: "basic"},
		{Name: "test_page_4", Description: "Page test 4", Type: "basic"},
		{Name: "test_page_5", Description: "Page test 5", Type: "basic"},
	}

	resp, body := ts.createElement(elements)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// Request with limit=2
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-elements?limit=2", baseURL), nil)
	req.Header.Set(testutils.HeaderOrgID, testOrgID)
	req.Header.Set(testutils.HeaderClientID, testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp ElementListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	require.Equal(t, 2, listResp.Metadata.Limit, "Limit should be 2")
	require.LessOrEqual(t, listResp.Metadata.Count, 2, "Count should not exceed limit")
	require.GreaterOrEqual(t, listResp.Metadata.Total, 5, "Total should be at least 5")
}

// TestListElements_WithLimitAndOffset_ReturnsCorrectPage tests pagination with offset
func (ts *ElementAPITestSuite) TestListElements_WithLimitAndOffset_ReturnsCorrectPage() {
	t := ts.T()

	// Create 3 elements to ensure we have data
	elements := []ConsentElementCreateRequest{
		{Name: "test_offset_1", Description: "Offset test 1", Type: "basic"},
		{Name: "test_offset_2", Description: "Offset test 2", Type: "basic"},
		{Name: "test_offset_3", Description: "Offset test 3", Type: "basic"},
	}

	resp, body := ts.createElement(elements)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// Request with limit=1&offset=1 (second item)
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-elements?limit=1&offset=1", baseURL), nil)
	req.Header.Set(testutils.HeaderOrgID, testOrgID)
	req.Header.Set(testutils.HeaderClientID, testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp ElementListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	require.Equal(t, 1, listResp.Metadata.Limit, "Limit should be 1")
	require.Equal(t, 1, listResp.Metadata.Offset, "Offset should be 1")
	require.LessOrEqual(t, listResp.Metadata.Count, 1, "Count should not exceed 1")
}

// TestListElements_FilterByName_ReturnsMatchingElement tests name filtering
func (ts *ElementAPITestSuite) TestListElements_FilterByName_ReturnsMatchingElement() {
	t := ts.T()

	// Create elements with distinctive names
	elements := []ConsentElementCreateRequest{
		{Name: "test_filter_exact", Description: "Exact match test", Type: "basic"},
		{Name: "test_filter_other", Description: "Other element", Type: "basic"},
	}

	resp, body := ts.createElement(elements)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// Filter by exact name
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-elements?name=test_filter_exact", baseURL), nil)
	req.Header.Set("org-id", testOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp ElementListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	// Should find at least one match
	require.GreaterOrEqual(t, listResp.Metadata.Total, 1, "Should find at least 1 matching element")

	// Verify the filtered result contains our element
	found := false
	for _, e := range listResp.Data {
		if e.Name == "test_filter_exact" {
			found = true
			break
		}
	}
	require.True(t, found, "Should find element with exact name match")
}

// TestListElements_EmptyOrg_ReturnsEmptyArray tests listing for org with no elements
func (ts *ElementAPITestSuite) TestListElements_EmptyOrg_ReturnsEmptyArray() {
	t := ts.T()

	// Use a different org ID that has no elements
	emptyOrgID := "org-empty-12345678"

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-elements", baseURL), nil)
	req.Header.Set("org-id", emptyOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp ElementListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	require.Equal(t, 0, listResp.Metadata.Total, "Total should be 0 for empty org")
	require.Equal(t, 0, listResp.Metadata.Count, "Count should be 0")
	require.Empty(t, listResp.Data, "Data array should be empty")
}

// TestListElements_VerifyAllTypes_ReturnsMixedTypes tests that all element types are returned
func (ts *ElementAPITestSuite) TestListElements_VerifyAllTypes_ReturnsMixedTypes() {
	t := ts.T()

	// Create one of each type
	elements := []ConsentElementCreateRequest{
		{
			Name:        "test_alltypes_string",
			Description: "String type",
			Type:        "basic",
		},
		{
			Name:        "test_alltypes_jsonpayload",
			Description: "JSON Payload type",
			Type:        "json-payload",
			Properties: map[string]string{
				"validationSchema": `{"type":"object"}`,
			},
		},
		{
			Name:        "test_alltypes_resourcefield",
			Description: "Resource Field type",
			Type:        "resource-field",
			Properties: map[string]string{
				"resourcePath": "/test",
				"jsonPath":     "$.test",
			},
		},
	}

	resp, body := ts.createElement(elements)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// List all elements
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-elements", baseURL), nil)
	req.Header.Set("org-id", testOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp ElementListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	// Verify we have at least one of each type in the results
	typesSeen := make(map[string]bool)
	for _, e := range listResp.Data {
		typesSeen[e.Type] = true
	}

	require.True(t, typesSeen["basic"], "Should have basic element")
	require.True(t, typesSeen["json-payload"], "Should have json-payload element")
	require.True(t, typesSeen["resource-field"], "Should have resource-field element")
}

// TestListElements_ErrorCases tests error scenarios for LIST
func (ts *ElementAPITestSuite) TestListElements_ErrorCases() {
	testCases := []struct {
		name            string
		setOrgHeader    bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name:            "MissingOrgID_ReturnsValidationError",
			setOrgHeader:    false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1003",
			messageContains: "organization ID is required",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-elements", baseURL), nil)

			if tc.setOrgHeader {
				req.Header.Set("org-id", testOrgID)
				req.Header.Set("TPP-client-id", testClientID)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.expectedStatus, resp.StatusCode)

			var errResp ErrorResponse
			json.NewDecoder(resp.Body).Decode(&errResp)
			require.Equal(t, tc.expectedCode, errResp.Code)
			require.Contains(t, strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains))
		})
	}
}
