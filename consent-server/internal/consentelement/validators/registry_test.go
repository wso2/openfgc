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

// TestNewElementTypeHandlerRegistry tests creating a new registry
func TestNewElementTypeHandlerRegistry(t *testing.T) {
	registry := NewElementTypeHandlerRegistry()
	require.NotNil(t, registry)
	require.NotNil(t, registry.handlers)
	require.Equal(t, 0, len(registry.handlers))
}

// TestRegister tests registering handlers
func TestRegister(t *testing.T) {
	testCases := []struct {
		name          string
		handlers      []ElementTypeHandler
		expectError   bool
		errorContains string
	}{
		{
			name:        "Register single handler",
			handlers:    []ElementTypeHandler{&BasicElementTypeHandler{}},
			expectError: false,
		},
		{
			name: "Register multiple different handlers",
			handlers: []ElementTypeHandler{
				&BasicElementTypeHandler{},
				&JsonPayloadElementTypeHandler{},
				&ResourceFieldElementTypeHandler{},
			},
			expectError: false,
		},
		{
			name: "Register duplicate handler",
			handlers: []ElementTypeHandler{
				&BasicElementTypeHandler{},
				&BasicElementTypeHandler{},
			},
			expectError:   true,
			errorContains: "already registered",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewElementTypeHandlerRegistry()
			var err error

			for _, handler := range tc.handlers {
				err = registry.Register(handler)
				if err != nil {
					break
				}
			}

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tc.handlers), len(registry.handlers))
			}
		})
	}
}

// TestGet tests retrieving handlers from registry
func TestGet(t *testing.T) {
	testCases := []struct {
		name          string
		setupHandlers []ElementTypeHandler
		getType       string
		expectError   bool
		expectedType  string
	}{
		{
			name:          "Get existing handler",
			setupHandlers: []ElementTypeHandler{&BasicElementTypeHandler{}},
			getType:       "basic",
			expectError:   false,
			expectedType:  "basic",
		},
		{
			name:          "Get non-existent handler",
			setupHandlers: []ElementTypeHandler{&BasicElementTypeHandler{}},
			getType:       "non-existent-type",
			expectError:   true,
		},
		{
			name: "Get from multiple handlers",
			setupHandlers: []ElementTypeHandler{
				&BasicElementTypeHandler{},
				&JsonPayloadElementTypeHandler{},
				&ResourceFieldElementTypeHandler{},
			},
			getType:      "json-payload",
			expectError:  false,
			expectedType: "json-payload",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewElementTypeHandlerRegistry()
			for _, handler := range tc.setupHandlers {
				_ = registry.Register(handler)
			}

			handler, err := registry.Get(tc.getType)

			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, handler)
				require.Contains(t, err.Error(), "no handler registered")
			} else {
				require.NoError(t, err)
				require.NotNil(t, handler)
				require.Equal(t, tc.expectedType, handler.GetType())
			}
		})
	}
}

// TestGetAllTypes tests retrieving all registered types
func TestGetAllTypes(t *testing.T) {
	testCases := []struct {
		name          string
		setupHandlers []ElementTypeHandler
		expectedCount int
		expectedTypes []string
	}{
		{
			name:          "Empty registry",
			setupHandlers: []ElementTypeHandler{},
			expectedCount: 0,
			expectedTypes: []string{},
		},
		{
			name:          "Single handler",
			setupHandlers: []ElementTypeHandler{&BasicElementTypeHandler{}},
			expectedCount: 1,
			expectedTypes: []string{"basic"},
		},
		{
			name: "Multiple handlers",
			setupHandlers: []ElementTypeHandler{
				&BasicElementTypeHandler{},
				&JsonPayloadElementTypeHandler{},
				&ResourceFieldElementTypeHandler{},
			},
			expectedCount: 3,
			expectedTypes: []string{"basic", "json-payload", "resource-field"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewElementTypeHandlerRegistry()
			for _, handler := range tc.setupHandlers {
				_ = registry.Register(handler)
			}

			types := registry.GetAllTypes()
			require.Equal(t, tc.expectedCount, len(types))

			for _, expectedType := range tc.expectedTypes {
				require.Contains(t, types, expectedType)
			}
		})
	}
}

// TestGetHandler tests the global GetHandler function
func TestGetHandler(t *testing.T) {
	// Default registry should have all three handlers registered in init()
	testCases := []struct {
		name        string
		typeStr     string
		expectError bool
	}{
		{
			name:        "Get basic handler",
			typeStr:     "basic",
			expectError: false,
		},
		{
			name:        "Get json-payload handler",
			typeStr:     "json-payload",
			expectError: false,
		},
		{
			name:        "Get resource-field handler",
			typeStr:     "resource-field",
			expectError: false,
		},
		{
			name:        "Get non-existent handler",
			typeStr:     "unknown-type",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := GetHandler(tc.typeStr)

			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, handler)
			} else {
				require.NoError(t, err)
				require.NotNil(t, handler)
				require.Equal(t, tc.typeStr, handler.GetType())
			}
		})
	}
}

// TestGetAllHandlerTypes tests the global GetAllHandlerTypes function
func TestGetAllHandlerTypes(t *testing.T) {
	types := GetAllHandlerTypes()
	
	// Default registry should have 3 handlers registered
	require.Equal(t, 3, len(types))
	require.Contains(t, types, "basic")
	require.Contains(t, types, "json-payload")
	require.Contains(t, types, "resource-field")
}

// TestGetDefaultRegistry tests the global registry getter
func TestGetDefaultRegistry(t *testing.T) {
	registry := GetDefaultRegistry()
	require.NotNil(t, registry)
	
	// Should be the same instance on multiple calls
	registry2 := GetDefaultRegistry()
	require.Equal(t, registry, registry2)
	
	// Should have handlers from init()
	types := registry.GetAllTypes()
	require.Equal(t, 3, len(types))
}
