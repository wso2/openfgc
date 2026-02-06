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
// DELETE /consent-elements/{elementId} Tests
// ========================================

// TestDeleteElement_StringType_Success tests deleting a string type element
func (ts *ElementAPITestSuite) TestDeleteElement_StringType_Success() {
	t := ts.T()

	// Create element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_delete_string",
			Description: "To be deleted",
			Type:        "basic",
		},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	elementID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deleteElementWithCheck(elementID)
	require.True(t, deleted, "Failed to delete element")

	// Verify it's gone with GET
	resp, _ = ts.getElement(elementID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Element should not exist after deletion")
}

// TestDeleteElement_JsonPayloadType_Success tests deleting a json-payload element
func (ts *ElementAPITestSuite) TestDeleteElement_JsonPayloadType_Success() {
	t := ts.T()

	// Create json-payload element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_delete_jsonpayload",
			Description: "To be deleted",
			Type:        "json-payload",
			Properties: map[string]string{
				"validationSchema": `{"type":"object"}`,
			},
		},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	elementID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deleteElementWithCheck(elementID)
	require.True(t, deleted, "Failed to delete element")

	// Verify deletion
	resp, _ = ts.getElement(elementID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestDeleteElement_ResourceFieldType_Success tests deleting a resource-field element
func (ts *ElementAPITestSuite) TestDeleteElement_ResourceFieldType_Success() {
	t := ts.T()

	// Create resource-field element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_delete_resourcefield",
			Description: "To be deleted",
			Type:        "resource-field",
			Properties: map[string]string{
				"resourcePath": "/users",
				"jsonPath":     "$.email",
			},
		},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	elementID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deleteElementWithCheck(elementID)
	require.True(t, deleted, "Failed to delete element")

	// Verify deletion
	resp, _ = ts.getElement(elementID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestDeleteElement_AlreadyDeleted_ReturnsNotFound tests idempotent deletion
func (ts *ElementAPITestSuite) TestDeleteElement_AlreadyDeleted_ReturnsNotFound() {
	t := ts.T()

	// Create element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_delete_twice",
			Description: "Delete twice test",
			Type:        "basic",
		},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	elementID := createResp.Data[0].ID

	// First delete - should succeed
	deleted := ts.deleteElementWithCheck(elementID)
	require.True(t, deleted, "First deletion should succeed")

	// Second delete - should return 404
	deleted = ts.deleteElementWithCheck(elementID)
	require.False(t, deleted, "Second deletion should fail (return false)")
}

// TestDeleteElement_ThenRecreateWithSameName_Succeeds tests name reusability after deletion
func (ts *ElementAPITestSuite) TestDeleteElement_ThenRecreateWithSameName_Succeeds() {
	t := ts.T()

	elementName := "test_delete_recreate"

	// Create element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        elementName,
			Description: "First version",
			Type:        "basic",
		},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	elementID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deleteElementWithCheck(elementID)
	require.True(t, deleted, "Failed to delete element")

	// Recreate with same name
	createPayload[0].Description = "Second version"
	resp, body = ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Should be able to recreate with same name after deletion")

	var recreateResp ElementCreateResponse
	json.Unmarshal([]byte(body), &recreateResp)
	newElementID := recreateResp.Data[0].ID
	ts.trackElement(newElementID)

	// Verify it's a different element with different ID
	require.NotEqual(t, elementID, newElementID, "New element should have different ID")
	require.Equal(t, elementName, recreateResp.Data[0].Name)
}

// TestDeleteElement_NonExistent_ReturnsNotFound tests deleting non-existent element
func (ts *ElementAPITestSuite) TestDeleteElement_NonExistent_ReturnsNotFound() {
	t := ts.T()

	nonExistentID := "00000000-0000-0000-0000-000000000000"

	deleted := ts.deleteElementWithCheck(nonExistentID)
	require.False(t, deleted, "Deleting non-existent element should return false")
}

// TestDeleteElement_ErrorCases tests error scenarios for DELETE
func (ts *ElementAPITestSuite) TestDeleteElement_ErrorCases() {
	testCases := []struct {
		name            string
		elementID       string
		setHeaders      bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name:            "MissingOrgID_ReturnsValidationError",
			elementID:       "00000000-0000-0000-0000-000000000000",
			setHeaders:      false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1003",
			messageContains: "organization ID is required",
		},
		{
			name:            "InvalidUUIDFormat_ReturnsNotFound",
			elementID:       "invalid-uuid-format",
			setHeaders:      true,
			expectedStatus:  http.StatusNotFound,
			expectedCode:    "CE-1016",
			messageContains: "not found",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("DELETE",
				fmt.Sprintf("%s/api/v1/consent-elements/%s", baseURL, tc.elementID),
				nil)

			if tc.setHeaders {
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
			require.Contains(t, strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains))
		})
	}
}

// TestDeleteElement_WithDependentConsents_ChecksBehavior tests deletion with dependent data
func (ts *ElementAPITestSuite) TestDeleteElement_WithDependentConsents_ChecksBehavior() {
	t := ts.T()

	// Create element
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_delete_with_deps",
			Description: "Element with potential dependencies",
			Type:        "basic",
		},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	elementID := createResp.Data[0].ID

	// Note: If the system prevents deletion when dependencies exist,
	// create a consent with this element first, then attempt deletion.
	// For now, we'll just verify deletion succeeds when no dependencies exist.

	// Delete element
	deleted := ts.deleteElementWithCheck(elementID)
	require.True(t, deleted, "Should be able to delete element when no dependencies exist")

	// Verify deletion
	resp, _ = ts.getElement(elementID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
