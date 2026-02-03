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

// TestResourceFieldElementTypeHandler_GetType tests the GetType method
func TestResourceFieldElementTypeHandler_GetType(t *testing.T) {
	handler := &ResourceFieldElementTypeHandler{}
	require.Equal(t, "resource-field", handler.GetType())
}

// TestResourceFieldElementTypeHandler_ValidateProperties tests property validation
func TestResourceFieldElementTypeHandler_ValidateProperties(t *testing.T) {
	handler := &ResourceFieldElementTypeHandler{}

	testCases := []struct {
		name          string
		properties    map[string]string
		expectErrors  bool
		expectedCount int
		errorFields   []string
	}{
		{
			name: "Valid properties",
			properties: map[string]string{
				"resourcePath": "/accounts",
				"jsonPath":     "Data.amount",
			},
			expectErrors: false,
		},
		{
			name: "Valid with optional validationSchema",
			properties: map[string]string{
				"resourcePath":     "/accounts",
				"jsonPath":         "Data.amount",
				"validationSchema": `{"type":"number"}`,
			},
			expectErrors: false,
		},
		{
			name:          "Missing both required properties",
			properties:    map[string]string{},
			expectErrors:  true,
			expectedCount: 2,
			errorFields:   []string{"resourcePath", "jsonPath"},
		},
		{
			name: "Missing resourcePath",
			properties: map[string]string{
				"jsonPath": "Data.amount",
			},
			expectErrors:  true,
			expectedCount: 1,
			errorFields:   []string{"resourcePath"},
		},
		{
			name: "Missing jsonPath",
			properties: map[string]string{
				"resourcePath": "/accounts",
			},
			expectErrors:  true,
			expectedCount: 1,
			errorFields:   []string{"jsonPath"},
		},
		{
			name: "Empty resourcePath",
			properties: map[string]string{
				"resourcePath": "",
				"jsonPath":     "Data.amount",
			},
			expectErrors:  true,
			expectedCount: 1,
			errorFields:   []string{"resourcePath"},
		},
		{
			name: "Empty jsonPath",
			properties: map[string]string{
				"resourcePath": "/accounts",
				"jsonPath":     "",
			},
			expectErrors:  true,
			expectedCount: 1,
			errorFields:   []string{"jsonPath"},
		},
		{
			name:          "Nil properties",
			properties:    nil,
			expectErrors:  true,
			expectedCount: 2,
			errorFields:   []string{"resourcePath", "jsonPath"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errors := handler.ValidateProperties(tc.properties)

			if tc.expectErrors {
				require.NotNil(t, errors)
				require.Equal(t, tc.expectedCount, len(errors))

				for _, field := range tc.errorFields {
					found := false
					for _, err := range errors {
						if err.Field == field {
							found = true
							require.NotEmpty(t, err.Message)
							break
						}
					}
					require.True(t, found, "Expected error for field: %s", field)
				}
			} else {
				require.Nil(t, errors)
			}
		})
	}
}

// TestResourceFieldElementTypeHandler_ProcessProperties tests property processing
func TestResourceFieldElementTypeHandler_ProcessProperties(t *testing.T) {
	handler := &ResourceFieldElementTypeHandler{}

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
				"resourcePath": "/accounts",
				"jsonPath":     "Data.amount",
			},
			expected: map[string]string{
				"resourcePath": "/accounts",
				"jsonPath":     "Data.amount",
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

// TestResourceFieldElementTypeHandler_GetPropertySpec tests property specification
func TestResourceFieldElementTypeHandler_GetPropertySpec(t *testing.T) {
	handler := &ResourceFieldElementTypeHandler{}
	spec := handler.GetPropertySpec()

	require.NotNil(t, spec)
	require.Equal(t, 3, len(spec))

	// Check required properties
	requiredCount := 0
	optionalCount := 0
	for _, prop := range spec {
		if prop.Required {
			requiredCount++
			require.Contains(t, []string{"resourcePath", "jsonPath"}, prop.Name)
		} else {
			optionalCount++
		}
	}

	require.Equal(t, 2, requiredCount, "Should have 2 required properties")
	require.Equal(t, 1, optionalCount, "Should have 1 optional property")
}
