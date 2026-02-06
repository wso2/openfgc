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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/tests/integration/testutils"
)

// ========================================
// PUT /consent-elements/{elementId} Tests
// ========================================

// TestUpdateElement_DescriptionChange_Succeeds tests updating only the description
func (ts *ElementAPITestSuite) TestUpdateElement_DescriptionChange_Succeeds() {
	t := ts.T()

	// Create an element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_update_desc",
			Description: "Original description",
			Type:        "basic",
		},
	}

	resp, bodyBytes := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	elementID := createResp.Data[0].ID
	ts.trackElement(elementID)

	// Update description
	updatePayload := ConsentElementUpdateRequest{
		Name:        "test_update_desc",
		Description: "Updated description",
		Type:        "basic",
	}

	resp, bodyBytes = ts.updateElement(elementID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update element: %s", bodyBytes)

	var updateResp ElementResponse
	json.Unmarshal(bodyBytes, &updateResp)
	require.Equal(t, "Updated description", *updateResp.Description)
}

// TestUpdateElement_PropertyChange_JsonPath_Succeeds tests updating jsonPath property
func (ts *ElementAPITestSuite) TestUpdateElement_PropertyChange_JsonPath_Succeeds() {
	t := ts.T()

	// Create resource-field element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_update_jsonpath",
			Description: "Resource field element",
			Type:        "resource-field",
			Properties: map[string]string{
				"resourcePath": "/users",
				"jsonPath":     "$.firstName",
			},
		},
	}

	resp, bodyBytes := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	elementID := createResp.Data[0].ID
	ts.trackElement(elementID)

	// Update jsonPath
	updatePayload := ConsentElementUpdateRequest{
		Name:        "test_update_jsonpath",
		Description: "Resource field element",
		Type:        "resource-field",
		Properties: map[string]string{
			"resourcePath": "/users",
			"jsonPath":     "$.profile.firstName", // Changed
		},
	}

	resp, bodyBytes = ts.updateElement(elementID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update element: %s", bodyBytes)

	var updateResp ElementResponse
	json.Unmarshal(bodyBytes, &updateResp)
	require.Equal(t, "$.profile.firstName", updateResp.Properties["jsonPath"])
}

// TestUpdateElement_TypeChange_StringToJsonPayload_Succeeds tests changing element type
func (ts *ElementAPITestSuite) TestUpdateElement_TypeChange_StringToJsonPayload_Succeeds() {
	t := ts.T()

	// Create basic element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_type_change",
			Description: "Type change test",
			Type:        "basic",
		},
	}

	resp, bodyBytes := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	elementID := createResp.Data[0].ID
	ts.trackElement(elementID)

	// Change to json-payload
	updatePayload := ConsentElementUpdateRequest{
		Name:        "test_type_change",
		Description: "Type change test",
		Type:        "json-payload",
		Properties: map[string]string{
			"validationSchema": `{"type":"object","properties":{"name":{"type":"string"}}}`,
		},
	}

	resp, bodyBytes = ts.updateElement(elementID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update element: %s", bodyBytes)

	var updateResp ElementResponse
	json.Unmarshal(bodyBytes, &updateResp)
	require.Equal(t, "json-payload", updateResp.Type)
	require.NotEmpty(t, updateResp.Properties["validationSchema"])
}

// TestUpdateElement_AllFieldsAtOnce_Succeeds tests updating all fields simultaneously
func (ts *ElementAPITestSuite) TestUpdateElement_AllFieldsAtOnce_Succeeds() {
	t := ts.T()

	// Create element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_full_update",
			Description: "Original",
			Type:        "basic",
			Properties: map[string]string{
				"value": "old:value",
			},
		},
	}

	resp, bodyBytes := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	elementID := createResp.Data[0].ID
	ts.trackElement(elementID)

	// Update all fields
	updatePayload := ConsentElementUpdateRequest{
		Name:        "test_full_update_new",
		Description: "Completely new description",
		Type:        "resource-field",
		Properties: map[string]string{
			"resourcePath": "/new/path",
			"jsonPath":     "$.newField",
		},
	}

	resp, bodyBytes = ts.updateElement(elementID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update element: %s", bodyBytes)

	var updateResp ElementResponse
	json.Unmarshal(bodyBytes, &updateResp)
	require.Equal(t, "test_full_update_new", updateResp.Name)
	require.Equal(t, "Completely new description", *updateResp.Description)
	require.Equal(t, "resource-field", updateResp.Type)
	require.Equal(t, "/new/path", updateResp.Properties["resourcePath"])
	require.Equal(t, "$.newField", updateResp.Properties["jsonPath"])
}

// TestUpdateElement_NonExistent_ReturnsNotFound tests updating non-existent element
func (ts *ElementAPITestSuite) TestUpdateElement_NonExistent_ReturnsNotFound() {
	t := ts.T()

	nonExistentID := "00000000-0000-0000-0000-000000000000"
	updatePayload := ConsentElementUpdateRequest{
		Name:        "test_nonexistent",
		Description: "Should fail",
		Type:        "basic",
	}

	resp, bodyBytes := ts.updateElement(nonExistentID, updatePayload)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404 for non-existent element")

	var errResp ErrorResponse
	json.Unmarshal(bodyBytes, &errResp)
	require.Equal(t, "CE-1016", errResp.Code)
	require.Contains(t, strings.ToLower(errResp.Description), "not found")
}

// TestUpdateElement_NameConflict_ReturnsBadRequest tests updating to existing name
func (ts *ElementAPITestSuite) TestUpdateElement_NameConflict_ReturnsBadRequest() {
	t := ts.T()

	// Create two elements
	createPayload := []ConsentElementCreateRequest{
		{Name: "test_conflict_1", Description: "First", Type: "basic"},
		{Name: "test_conflict_2", Description: "Second", Type: "basic"},
	}

	resp, bodyBytes := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	elementID1 := createResp.Data[0].ID
	ts.trackElement(elementID1)
	ts.trackElement(createResp.Data[1].ID)

	// Try to update first to have same name as second
	updatePayload := ConsentElementUpdateRequest{
		Name:        "test_conflict_2", // Conflict!
		Description: "Try to rename",
		Type:        "basic",
	}

	resp, bodyBytes = ts.updateElement(elementID1, updatePayload)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "Should return 409 for name conflict")

	var errResp ErrorResponse
	json.Unmarshal(bodyBytes, &errResp)
	require.Equal(t, "CE-1011", errResp.Code)
	require.Contains(t, strings.ToLower(errResp.Description), "already exists")
}

// TestUpdateElement_ErrorCases tests various error scenarios
func (ts *ElementAPITestSuite) TestUpdateElement_ErrorCases() {
	// Create an element for testing
	createPayload := []ConsentElementCreateRequest{
		{Name: "test_update_errors", Description: "For error testing", Type: "basic"},
	}

	resp, bodyBytes := ts.createElement(createPayload)
	require.Equal(ts.T(), http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	elementID := createResp.Data[0].ID
	ts.trackElement(elementID)

	testCases := []struct {
		name            string
		payload         interface{}
		setHeaders      bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name: "MissingName_ReturnsValidationError",
			payload: map[string]interface{}{
				"description": "Missing name",
				"type":        "basic",
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1004",
			messageContains: "element name is required",
		},
		{
			name: "MissingType_ReturnsValidationError",
			payload: map[string]interface{}{
				"name":        "test_no_type",
				"description": "Missing type",
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1005",
			messageContains: "element type is required",
		},
		{
			name: "InvalidType_ReturnsValidationError",
			payload: ConsentElementUpdateRequest{
				Name:        "test_invalid_type",
				Description: "Invalid type",
				Type:        "invalid-type",
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1010",
			messageContains: "invalid element type",
		},
		{
			name: "JsonPayloadType_MissingValidationSchema_ReturnsValidationError",
			payload: ConsentElementUpdateRequest{
				Name:        "test_json_no_schema",
				Description: "Missing validation schema",
				Type:        "json-payload",
				Properties:  map[string]string{},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusInternalServerError,
			expectedCode:    "CE-5010",
			messageContains: "property validation failed",
		},
		{
			name: "ResourceFieldType_MissingResourcePath_ReturnsValidationError",
			payload: ConsentElementUpdateRequest{
				Name:        "test_missing_resource",
				Description: "Missing resourcePath",
				Type:        "resource-field",
				Properties: map[string]string{
					"jsonPath": "$.test",
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusInternalServerError,
			expectedCode:    "CE-5010",
			messageContains: "property validation failed",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			var reqBody []byte
			var err error

			if str, ok := tc.payload.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tc.payload)
				require.NoError(t, err)
			}

			url := fmt.Sprintf("%s/api/v1/consent-elements/%s", testServerURL, elementID)
			httpReq, _ := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
			httpReq.Header.Set(testutils.HeaderContentType, "application/json")

			if tc.setHeaders {
				httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
				httpReq.Header.Set(testutils.HeaderClientID, testClientID)
			}

			client := &http.Client{}
			resp, err := client.Do(httpReq)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch for %s", tc.name)

			body, _ := io.ReadAll(resp.Body)
			var errResp ErrorResponse
			json.Unmarshal(body, &errResp)
			require.Equal(t, tc.expectedCode, errResp.Code, "Error code mismatch")
			require.Contains(t, strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains))
		})
	}
}
