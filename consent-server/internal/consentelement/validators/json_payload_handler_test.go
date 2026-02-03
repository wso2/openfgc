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

package validators

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestJsonPayloadElementTypeHandler_GetType tests the GetType method
func TestJsonPayloadElementTypeHandler_GetType(t *testing.T) {
	handler := &JsonPayloadElementTypeHandler{}
	require.Equal(t, "json-payload", handler.GetType())
}

// TestJsonPayloadElementTypeHandler_ValidateProperties tests property validation
func TestJsonPayloadElementTypeHandler_ValidateProperties(t *testing.T) {
	handler := &JsonPayloadElementTypeHandler{}

	testCases := []struct {
		name          string
		properties    map[string]string
		expectErrors  bool
		expectedCount int
		errorField    string
		errorContains string
	}{
		{
			name: "Valid with simple JSON",
			properties: map[string]string{
				"validationSchema": `{"type":"string"}`,
			},
			expectErrors: false,
		},
		{
			name: "Valid with complex JSON schema",
			properties: map[string]string{
				"validationSchema": `{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}}}`,
			},
			expectErrors: false,
		},
		{
			name: "Valid with optional properties",
			properties: map[string]string{
				"validationSchema": `{"type":"string"}`,
				"resourcePath":     "/accounts",
				"jsonPath":         "Data.amount",
			},
			expectErrors: false,
		},
		{
			name:          "Missing validationSchema",
			properties:    map[string]string{},
			expectErrors:  true,
			expectedCount: 1,
			errorField:    "validationSchema",
			errorContains: "required",
		},
		{
			name: "Empty validationSchema",
			properties: map[string]string{
				"validationSchema": "",
			},
			expectErrors:  true,
			expectedCount: 1,
			errorField:    "validationSchema",
			errorContains: "required",
		},
		{
			name: "Invalid JSON in validationSchema",
			properties: map[string]string{
				"validationSchema": `{invalid json}`,
			},
			expectErrors:  true,
			expectedCount: 1,
			errorField:    "validationSchema",
			errorContains: "valid JSON",
		},
		{
			name: "Malformed JSON - missing closing brace",
			properties: map[string]string{
				"validationSchema": `{"type":"string"`,
			},
			expectErrors:  true,
			expectedCount: 1,
			errorField:    "validationSchema",
			errorContains: "valid JSON",
		},
		{
			name: "Non-JSON string",
			properties: map[string]string{
				"validationSchema": "not a json string",
			},
			expectErrors:  true,
			expectedCount: 1,
			errorField:    "validationSchema",
			errorContains: "valid JSON",
		},
		{
			name:          "Nil properties",
			properties:    nil,
			expectErrors:  true,
			expectedCount: 1,
			errorField:    "validationSchema",
			errorContains: "required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errors := handler.ValidateProperties(tc.properties)

			if tc.expectErrors {
				require.NotNil(t, errors)
				require.Equal(t, tc.expectedCount, len(errors))
				require.Equal(t, tc.errorField, errors[0].Field)
				require.Contains(t, errors[0].Message, tc.errorContains)
			} else {
				require.Nil(t, errors)
			}
		})
	}
}

// TestJsonPayloadElementTypeHandler_ProcessProperties tests property processing
func TestJsonPayloadElementTypeHandler_ProcessProperties(t *testing.T) {
	handler := &JsonPayloadElementTypeHandler{}

	testCases := []struct {
		name       string
		properties map[string]string
		expected   map[string]string
	}{
		{
			name:       "Nil properties",
			properties: nil,
			expected:   nil,
		},
		{
			name:       "Empty properties",
			properties: map[string]string{},
			expected:   map[string]string{},
		},
		{
			name: "With properties",
			properties: map[string]string{
				"validationSchema": `{"type":"string"}`,
				"resourcePath":     "/accounts",
			},
			expected: map[string]string{
				"validationSchema": `{"type":"string"}`,
				"resourcePath":     "/accounts",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handler.ProcessProperties(tc.properties)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestJsonPayloadElementTypeHandler_GetPropertySpec tests property specification
func TestJsonPayloadElementTypeHandler_GetPropertySpec(t *testing.T) {
	handler := &JsonPayloadElementTypeHandler{}
	spec := handler.GetPropertySpec()

	require.NotNil(t, spec)
	require.Equal(t, 3, len(spec))

	// Find validationSchema property
	var validationSchemaProp *ElementPropertySpec
	for i := range spec {
		if spec[i].Name == "validationSchema" {
			validationSchemaProp = &spec[i]
			break
		}
	}

	require.NotNil(t, validationSchemaProp)
	require.True(t, validationSchemaProp.Required)
	require.Equal(t, "json", validationSchemaProp.Type)
}
