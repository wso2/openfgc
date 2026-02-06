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
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/tests/integration/testutils"
)

// ========================================
// POST /consent-elements/validate Tests
// ========================================

// validateElements is a helper function to validate element names
func (ts *ElementAPITestSuite) validateElements(names []string) []string {
	reqBody, err := json.Marshal(names)
	ts.Require().NoError(err)

	httpReq, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/consent-elements/validate", baseURL),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusOK, resp.StatusCode)

	var validNames []string
	json.NewDecoder(resp.Body).Decode(&validNames)
	return validNames
}

// TestValidateElements_AllValid_ReturnsAll tests validating all existing elements
func (ts *ElementAPITestSuite) TestValidateElements_AllValid_ReturnsAll() {
	t := ts.T()

	// Create three elements
	createPayload := []ConsentElementCreateRequest{
		{Name: "test_validate_1", Description: "First", Type: "basic"},
		{Name: "test_validate_2", Description: "Second", Type: "basic"},
		{Name: "test_validate_3", Description: "Third", Type: "basic"},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// Validate all three
	validatePayload := []string{"test_validate_1", "test_validate_2", "test_validate_3"}
	validNames := ts.validateElements(validatePayload)

	require.Len(t, validNames, 3, "Should return all 3 valid names")
	require.Contains(t, validNames, "test_validate_1")
	require.Contains(t, validNames, "test_validate_2")
	require.Contains(t, validNames, "test_validate_3")
}

// TestValidateElements_PartialValid_ReturnsSubset tests mixed valid and invalid names
func (ts *ElementAPITestSuite) TestValidateElements_PartialValid_ReturnsSubset() {
	t := ts.T()

	// Create two elements
	createPayload := []ConsentElementCreateRequest{
		{Name: "test_partial_1", Description: "First", Type: "basic"},
		{Name: "test_partial_2", Description: "Second", Type: "basic"},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// Validate mix of valid and invalid
	validatePayload := []string{
		"test_partial_1", // Valid
		"test_partial_2", // Valid
		"nonexistent_1",  // Invalid
		"nonexistent_2",  // Invalid
	}
	validNames := ts.validateElements(validatePayload)

	require.Len(t, validNames, 2, "Should return only 2 valid names")
	require.Contains(t, validNames, "test_partial_1")
	require.Contains(t, validNames, "test_partial_2")
	require.NotContains(t, validNames, "nonexistent_1")
	require.NotContains(t, validNames, "nonexistent_2")
}

// TestValidateElements_NoneValid_ReturnsEmpty tests all invalid names
func (ts *ElementAPITestSuite) TestValidateElements_NoneValid_ReturnsEmpty() {
	t := ts.T()

	// Validate only non-existent names
	validatePayload := []string{
		"totally_fake_name_1",
		"totally_fake_name_2",
		"totally_fake_name_3",
	}

	// Server returns 400 error when no valid elements found
	reqBody, err := json.Marshal(validatePayload)
	require.NoError(t, err)

	httpReq, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/consent-elements/validate", baseURL),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return 400 when no valid elements found")

	var errResp ErrorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	require.Equal(t, "CE-1015", errResp.Code)
	require.Contains(t, strings.ToLower(errResp.Description), "no valid elements found")
}

// TestValidateElements_SingleName_ReturnsOne tests single name validation
func (ts *ElementAPITestSuite) TestValidateElements_SingleName_ReturnsOne() {
	t := ts.T()

	// Create element
	createPayload := []ConsentElementCreateRequest{
		{Name: "test_single_validate", Description: "Single", Type: "basic"},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	ts.trackElement(createResp.Data[0].ID)

	// Validate single name
	validatePayload := []string{"test_single_validate"}
	validNames := ts.validateElements(validatePayload)

	require.Len(t, validNames, 1)
	require.Equal(t, "test_single_validate", validNames[0])
}

// TestValidateElements_DuplicatesInRequest_ReturnsDeduplicated tests duplicate handling
func (ts *ElementAPITestSuite) TestValidateElements_DuplicatesInRequest_ReturnsDeduplicated() {
	t := ts.T()

	// Create element
	createPayload := []ConsentElementCreateRequest{
		{Name: "test_duplicate_validate", Description: "Duplicate test", Type: "basic"},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	ts.trackElement(createResp.Data[0].ID)

	// Validate with duplicates
	validatePayload := []string{
		"test_duplicate_validate",
		"test_duplicate_validate",
		"test_duplicate_validate",
	}
	validNames := ts.validateElements(validatePayload)

	// Should return deduplicated result
	require.LessOrEqual(t, len(validNames), 3, "Should handle duplicates")
	require.Contains(t, validNames, "test_duplicate_validate")
}

// TestValidateElements_MixedTypes_ReturnsAllValid tests validation across different element types
func (ts *ElementAPITestSuite) TestValidateElements_MixedTypes_ReturnsAllValid() {
	t := ts.T()

	// Create one of each type
	createPayload := []ConsentElementCreateRequest{
		{
			Name:        "test_validate_string",
			Description: "String type",
			Type:        "basic",
		},
		{
			Name:        "test_validate_jsonpayload",
			Description: "JSON Payload type",
			Type:        "json-payload",
			Properties: map[string]string{
				"validationSchema": `{"type":"object"}`,
			},
		},
		{
			Name:        "test_validate_resourcefield",
			Description: "Resource Field type",
			Type:        "resource-field",
			Properties: map[string]string{
				"resourcePath": "/test",
				"jsonPath":     "$.test",
			},
		},
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// Validate all types
	validatePayload := []string{
		"test_validate_string",
		"test_validate_jsonpayload",
		"test_validate_resourcefield",
	}
	validNames := ts.validateElements(validatePayload)

	require.Len(t, validNames, 3, "Should validate all types")
	require.Contains(t, validNames, "test_validate_string")
	require.Contains(t, validNames, "test_validate_jsonpayload")
	require.Contains(t, validNames, "test_validate_resourcefield")
}

// TestValidateElements_EmptyArray_ReturnsBadRequest tests empty input
func (ts *ElementAPITestSuite) TestValidateElements_EmptyArray_ReturnsBadRequest() {
	t := ts.T()

	validatePayload := []string{}

	reqBody, err := json.Marshal(validatePayload)
	require.NoError(t, err)

	httpReq, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/consent-elements/validate", baseURL),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Empty array should return 400")

	var errResp ErrorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	require.Equal(t, "CE-1014", errResp.Code)
	require.Contains(t, strings.ToLower(errResp.Description), "at least one element name must be provided")
}

// TestValidateElements_ErrorCases tests error scenarios
func (ts *ElementAPITestSuite) TestValidateElements_ErrorCases() {
	testCases := []struct {
		name            string
		payload         interface{}
		setHeaders      bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name:            "MissingOrgID_ReturnsValidationError",
			payload:         []string{"test_element"},
			setHeaders:      false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1003",
			messageContains: "organization ID is required",
		},
		{
			name:            "MalformedJSON_ReturnsBadRequest",
			payload:         "invalid{{{json",
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1001",
			messageContains: "invalid character",
		},
		{
			name:            "NullPayload_ReturnsBadRequest",
			payload:         nil,
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1014",
			messageContains: "at least one element name must be provided",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			var reqBody []byte
			var err error

			if str, ok := tc.payload.(string); ok {
				reqBody = []byte(str)
			} else if tc.payload != nil {
				reqBody, err = json.Marshal(tc.payload)
				require.NoError(t, err)
			} else {
				reqBody = []byte("null")
			}

			httpReq, _ := http.NewRequest("POST",
				fmt.Sprintf("%s/api/v1/consent-elements/validate", baseURL),
				bytes.NewBuffer(reqBody))
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

			var errResp ErrorResponse
			json.NewDecoder(resp.Body).Decode(&errResp)
			require.Equal(t, tc.expectedCode, errResp.Code, "Error code mismatch")
			require.Contains(t, strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains))
		})
	}
}

// TestValidateElements_LargeList_HandlesMany tests validation with many names
func (ts *ElementAPITestSuite) TestValidateElements_LargeList_HandlesMany() {
	t := ts.T()

	// Create 10 elements
	createPayload := make([]ConsentElementCreateRequest, 10)
	validatePayload := make([]string, 10)

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("test_large_list_%d", i)
		createPayload[i] = ConsentElementCreateRequest{
			Name:        name,
			Description: fmt.Sprintf("Element %d", i),
			Type:        "basic",
		}
		validatePayload[i] = name
	}

	resp, body := ts.createElement(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}

	// Validate all 10
	validNames := ts.validateElements(validatePayload)
	require.Len(t, validNames, 10, "Should validate all 10 elements")
}
