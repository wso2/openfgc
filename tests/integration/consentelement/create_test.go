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
	"io"
	"net/http"
	"strings"

	"github.com/wso2/openfgc/tests/integration/testutils"
)

// TestCreateElement_StringType_WithValue creates a string type element with value property
func (ts *ElementAPITestSuite) TestCreateElement_StringType_WithValue() {
	element := ConsentElementCreateRequest{
		Name:        "test_license_read",
		Description: "Allows accessing driving license API",
		Type:        "basic",
		Properties: map[string]string{
			"value": "license:read",
		},
	}

	// Create element
	resp, body := ts.createElement([]ConsentElementCreateRequest{element})
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)

	created := createResp.Data[0]
	ts.Require().Equal(element.Name, created.Name)
	ts.Require().Equal(element.Type, created.Type)
	ts.Require().Equal(element.Properties["value"], created.Properties["value"])
	ts.Require().NotEmpty(created.ID)

	// Track for suite cleanup
	ts.trackElement(created.ID)
}

// TestCreateElement_StringType_NoProperties creates a string type element with no properties
func (ts *ElementAPITestSuite) TestCreateElement_StringType_NoProperties() {
	element := ConsentElementCreateRequest{
		Name:        "test_basic_string",
		Description: "String type with no properties",
		Type:        "basic",
		Properties:  map[string]string{},
	}

	resp, body := ts.createElement([]ConsentElementCreateRequest{element})
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)
	ts.Require().Equal(element.Name, createResp.Data[0].Name)
	ts.trackElement(createResp.Data[0].ID) // Track for suite cleanup
}

// TestCreateElement_JsonPayloadType_WithValidationSchema creates a json-payload element
func (ts *ElementAPITestSuite) TestCreateElement_JsonPayloadType_WithValidationSchema() {
	element := ConsentElementCreateRequest{
		Name:        "test_account_schema",
		Description: "Account access schema validation",
		Type:        "json-payload",
		Properties: map[string]string{
			"validationSchema": "{}",
		},
	}

	resp, body := ts.createElement([]ConsentElementCreateRequest{element})
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)

	created := createResp.Data[0]
	ts.Require().Equal(element.Name, created.Name)
	ts.Require().Equal("json-payload", created.Type)
	ts.Require().NotEmpty(created.Properties["validationSchema"])
	ts.trackElement(created.ID) // Track for suite cleanup
}

// TestCreateElement_ResourceFieldType_FirstName creates a resource-field element with jsonPath and resourcePath
func (ts *ElementAPITestSuite) TestCreateElement_ResourceFieldType_FirstName() {
	element := ConsentElementCreateRequest{
		Name:        "test_first_name",
		Description: "Allows access to the user's first name",
		Type:        "resource-field",
		Properties: map[string]string{
			"jsonPath":     "$.personal.firstName",
			"resourcePath": "/user/{nic}",
		},
	}

	resp, body := ts.createElement([]ConsentElementCreateRequest{element})
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)

	created := createResp.Data[0]
	ts.Require().Equal(element.Name, created.Name)
	ts.Require().Equal("resource-field", created.Type)
	ts.Require().Equal("$.personal.firstName", created.Properties["jsonPath"])
	ts.Require().Equal("/user/{nic}", created.Properties["resourcePath"])
	ts.trackElement(created.ID) // Track for suite cleanup
}

// TestCreateElement_Batch_ThreeResourceFieldTypeElements creates 3 resource-field elements in one request
func (ts *ElementAPITestSuite) TestCreateElement_Batch_ThreeResourceFieldTypeElements() {
	elements := []ConsentElementCreateRequest{
		{
			Name:        "test_batch_first_name",
			Description: "Allows access to the user's first name",
			Type:        "resource-field",
			Properties: map[string]string{
				"jsonPath":     "$.personal.firstName",
				"resourcePath": "/user/{nic}",
			},
		},
		{
			Name:        "test_batch_last_name",
			Description: "Allows access to the user's last name",
			Type:        "resource-field",
			Properties: map[string]string{
				"jsonPath":     "$.personal.lastName",
				"resourcePath": "/user/{nic}",
			},
		},
		{
			Name:        "test_batch_full_name",
			Description: "Allows access to the user's full name",
			Type:        "resource-field",
			Properties: map[string]string{
				"jsonPath":     "$.personal.fullName",
				"resourcePath": "/user/{nic}",
			},
		},
	}

	resp, body := ts.createElement(elements)
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 3)

	// Verify all three were created
	for i, element := range elements {
		ts.Require().Equal(element.Name, createResp.Data[i].Name)
		ts.Require().Equal("resource-field", createResp.Data[i].Type)
		ts.Require().NotEmpty(createResp.Data[i].ID)
	}

	// Track all for cleanup
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}
}

// TestCreateElement_Batch_MixedTypes creates elements with different types in one batch
func (ts *ElementAPITestSuite) TestCreateElement_Batch_MixedTypes() {
	elements := []ConsentElementCreateRequest{
		{
			Name:        "test_mixed_string",
			Description: "String type",
			Type:        "basic",
			Properties: map[string]string{
				"value": "api:resource:read",
			},
		},
		{
			Name:        "test_mixed_json_payload",
			Description: "JSON Payload type",
			Type:        "json-payload",
			Properties: map[string]string{
				"validationSchema": "{}",
			},
		},
		{
			Name:        "test_mixed_resource_field",
			Description: "Resource Field type",
			Type:        "resource-field",
			Properties: map[string]string{
				"jsonPath":     "$.data.field",
				"resourcePath": "/resource/{id}",
			},
		},
	}

	resp, body := ts.createElement(elements)
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp ElementCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 3)

	// Verify types
	ts.Require().Equal("basic", createResp.Data[0].Type)
	ts.Require().Equal("json-payload", createResp.Data[1].Type)
	ts.Require().Equal("resource-field", createResp.Data[2].Type)

	// Track all for cleanup
	for _, e := range createResp.Data {
		ts.trackElement(e.ID)
	}
}

// TestCreateElement_RetrieveAndVerifyAllFields creates an element and verifies all fields via GET
func (ts *ElementAPITestSuite) TestCreateElement_RetrieveAndVerifyAllFields() {
	element := ConsentElementCreateRequest{
		Name:        "test_verify_all_fields",
		Description: "Test element for field verification",
		Type:        "resource-field",
		Properties: map[string]string{
			"jsonPath":     "$.user.email",
			"resourcePath": "/user/{id}",
		},
	}

	// Create
	createHttpResp, createBody := ts.createElement([]ConsentElementCreateRequest{element})
	defer createHttpResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createHttpResp.StatusCode)

	var createResp ElementCreateResponse
	err := json.Unmarshal(createBody, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)

	elementID := createResp.Data[0].ID
	ts.trackElement(elementID) // Track for suite cleanup

	// Retrieve
	getResp, getBody := ts.getElement(elementID)
	defer getResp.Body.Close()
	ts.Require().Equal(http.StatusOK, getResp.StatusCode)

	var retrieved ElementResponse
	err = json.Unmarshal(getBody, &retrieved)
	ts.Require().NoError(err)

	// Verify all fields match
	ts.Require().Equal(elementID, retrieved.ID)
	ts.Require().Equal(element.Name, retrieved.Name)
	ts.Require().NotNil(retrieved.Description)
	ts.Require().Equal(element.Description, *retrieved.Description)
	ts.Require().Equal(element.Type, retrieved.Type)
	ts.Require().Equal(element.Properties["jsonPath"], retrieved.Properties["jsonPath"])
	ts.Require().Equal(element.Properties["resourcePath"], retrieved.Properties["resourcePath"])
}

// TestCreateElement_ErrorCases tests various error scenarios
func (ts *ElementAPITestSuite) TestCreateElement_ErrorCases() {
	validElement := ConsentElementCreateRequest{
		Name:        "test_error_valid",
		Description: "Valid element",
		Type:        "basic",
		Properties:  map[string]string{},
	}

	testCases := []struct {
		name            string
		payload         interface{}
		setHeaders      bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		// Header validation
		{
			name:            "MissingOrgID_ReturnsValidationError",
			payload:         []ConsentElementCreateRequest{validElement},
			setHeaders:      false,
			expectedStatus:  http.StatusNotFound, // Missing route/org returns 404
			expectedCode:    "",                  // No structured error for 404
			messageContains: "",
		},

		// Request body validation
		{
			name:            "MalformedJSON_ReturnsBadRequest",
			payload:         "invalid{{{json",
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1001",
			messageContains: "invalid character",
		},
		{
			name:            "EmptyArray_RequiresAtLeastOneElement",
			payload:         []ConsentElementCreateRequest{},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1002",
			messageContains: "at least one element must be provided",
		},

		// Name validation
		{
			name: "MissingNameField_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Description: "Missing name",
					Type:        "basic",
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1004",
			messageContains: "element name is required",
		},
		{
			name: "NameExceeds255Chars_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        strings.Repeat("a", 256),
					Description: "Name too long",
					Type:        "basic",
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1006",
			messageContains: "name must not exceed 255 characters",
		},
		{
			name: "DuplicateNameInBatch_ReturnsConflict",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_duplicate",
					Description: "First",
					Type:        "basic",
					Properties:  map[string]string{},
				},
				{
					Name:        "test_error_duplicate",
					Description: "Duplicate",
					Type:        "basic",
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusConflict,
			expectedCode:    "CE-1012",
			messageContains: "duplicate element name 'test_error_duplicate' in request batch",
		},

		// Type validation
		{
			name: "MissingTypeField_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_no_type",
					Description: "Missing type",
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1005",
			messageContains: "element type is required",
		},
		{
			name: "InvalidTypeValue_STANDARD_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_invalid_type",
					Description: "Invalid type",
					Type:        "STANDARD",
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1010",
			messageContains: "invalid element type",
		},
		{
			name: "InvalidTypeValue_OldString_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_old_string_type",
					Description: "Old string type value",
					Type:        "string", // Old type value
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1010",
			messageContains: "invalid element type",
		},
		{
			name: "InvalidTypeValue_OldAttribute_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_old_attribute_type",
					Description: "Old attribute type value",
					Type:        "attribute", // Old type value
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1010",
			messageContains: "invalid element type",
		},

		// Type-specific property validation
		{
			name: "JsonPayloadType_MissingValidationSchema_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_json_no_schema",
					Description: "JSON payload without validationSchema",
					Type:        "json-payload",
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusInternalServerError,
			expectedCode:    "CE-5010",
			messageContains: "property validation failed",
		},
		{
			name: "ResourceFieldType_MissingResourcePath_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_resource_no_path",
					Description: "Resource field without resourcePath",
					Type:        "resource-field",
					Properties: map[string]string{
						"jsonPath": "$.data",
					},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusInternalServerError,
			expectedCode:    "CE-5010",
			messageContains: "resourcePath is required for resource-field",
		},
		{
			name: "ResourceFieldType_MissingJsonPath_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_resource_no_json",
					Description: "Resource field without jsonPath",
					Type:        "resource-field",
					Properties: map[string]string{
						"resourcePath": "/resource/{id}",
					},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusInternalServerError,
			expectedCode:    "CE-5010",
			messageContains: "property validation failed",
		},
		{
			name: "ResourceFieldType_MissingBothPaths_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_resource_no_paths",
					Description: "Resource field without any paths",
					Type:        "resource-field",
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusInternalServerError,
			expectedCode:    "CE-5010",
			messageContains: "resourcePath is required for resource-field",
		},

		// Description validation
		{
			name: "DescriptionExceeds1024Chars_ReturnsValidationError",
			payload: []ConsentElementCreateRequest{
				{
					Name:        "test_error_desc_1025",
					Description: strings.Repeat("x", 1025), // Exceeds 1024 char limit
					Type:        "basic",
					Properties:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CE-1008",
			messageContains: "description must not exceed 1024 characters",
		},
	}

	for _, tc := range testCases {
		ts.Run(tc.name, func() {
			var resp *http.Response
			var body []byte

			if tc.setHeaders {
				resp, body = ts.createElement(tc.payload)
			} else {
				// Create request without headers
				var jsonData []byte
				var err error

				if str, ok := tc.payload.(string); ok {
					jsonData = []byte(str)
				} else {
					jsonData, err = json.Marshal(tc.payload)
					ts.Require().NoError(err)
				}

				req, err := http.NewRequest(http.MethodPost, testServerURL+"/consent-elements", bytes.NewBuffer(jsonData))
				ts.Require().NoError(err)
				req.Header.Set(testutils.HeaderContentType, "application/json")
				// Deliberately not setting org-id and client-id headers

				client := &http.Client{}
				resp, err = client.Do(req)
				ts.Require().NoError(err)
				body, err = io.ReadAll(resp.Body)
				ts.Require().NoError(err)
			}
			defer resp.Body.Close()

			// Verify status code
			ts.Require().Equal(tc.expectedStatus, resp.StatusCode, "Test case: %s", tc.name)

			// Skip error response validation for 404 (no structured error)
			if tc.expectedStatus == http.StatusNotFound {
				return
			}

			// Parse error response
			var errResp ErrorResponse
			err := json.Unmarshal(body, &errResp)
			ts.Require().NoError(err, "Test case: %s", tc.name)

			// Verify error code and message
			ts.Require().Equal(tc.expectedCode, errResp.Code, "Test case: %s", tc.name)
			if tc.messageContains != "" {
				ts.Require().Contains(strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains), "Test case: %s - Description: %s", tc.name, errResp.Description)
			}
			ts.Require().NotEmpty(errResp.TraceID, "Test case: %s", tc.name)
		})
	}
}
