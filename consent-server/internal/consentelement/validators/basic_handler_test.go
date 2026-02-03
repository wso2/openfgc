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

// TestBasicElementTypeHandler_GetType tests the GetType method
func TestBasicElementTypeHandler_GetType(t *testing.T) {
	handler := &BasicElementTypeHandler{}
	require.Equal(t, "basic", handler.GetType())
}

// TestBasicElementTypeHandler_ValidateProperties tests property validation
func TestBasicElementTypeHandler_ValidateProperties(t *testing.T) {
	handler := &BasicElementTypeHandler{}

	testCases := []struct {
		name       string
		properties map[string]string
		expectNil  bool
	}{
		{
			name:       "Nil properties",
			properties: nil,
			expectNil:  true,
		},
		{
			name:       "Empty properties",
			properties: map[string]string{},
			expectNil:  true,
		},
		{
			name: "With optional properties",
			properties: map[string]string{
				"validationSchema": `{"type":"string"}`,
				"resourcePath":     "/accounts",
				"jsonPath":         "Data.amount",
			},
			expectNil: true,
		},
		{
			name: "With partial properties",
			properties: map[string]string{
				"resourcePath": "/accounts",
			},
			expectNil: true,
		},
		{
			name: "With unknown properties",
			properties: map[string]string{
				"unknownProp": "value",
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errors := handler.ValidateProperties(tc.properties)
			if tc.expectNil {
				require.Nil(t, errors)
			}
		})
	}
}

// TestBasicElementTypeHandler_ProcessProperties tests property processing
func TestBasicElementTypeHandler_ProcessProperties(t *testing.T) {
	handler := &BasicElementTypeHandler{}

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

// TestBasicElementTypeHandler_GetPropertySpec tests property specification
func TestBasicElementTypeHandler_GetPropertySpec(t *testing.T) {
	handler := &BasicElementTypeHandler{}
	spec := handler.GetPropertySpec()

	require.NotNil(t, spec)
	require.Equal(t, 3, len(spec))

	// Check property names
	propertyNames := make([]string, len(spec))
	for i, prop := range spec {
		propertyNames[i] = prop.Name
	}
	require.Contains(t, propertyNames, "validationSchema")
	require.Contains(t, propertyNames, "resourcePath")
	require.Contains(t, propertyNames, "jsonPath")

	// Verify all properties are optional
	for _, prop := range spec {
		require.False(t, prop.Required, "Property %s should be optional", prop.Name)
	}
}
