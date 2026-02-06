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

// TestIsValidJSON tests the isValidJSON helper function
func TestIsValidJSON(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid JSON object",
			input:    `{"key":"value"}`,
			expected: true,
		},
		{
			name:     "Valid JSON array",
			input:    `["item1","item2"]`,
			expected: true,
		},
		{
			name:     "Valid JSON string",
			input:    `"string"`,
			expected: true,
		},
		{
			name:     "Valid JSON number",
			input:    `123`,
			expected: true,
		},
		{
			name:     "Valid JSON boolean",
			input:    `true`,
			expected: true,
		},
		{
			name:     "Valid JSON null",
			input:    `null`,
			expected: true,
		},
		{
			name:     "Valid complex JSON",
			input:    `{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}}}`,
			expected: true,
		},
		{
			name:     "Valid nested JSON",
			input:    `{"outer":{"inner":{"deep":"value"}}}`,
			expected: true,
		},
		{
			name:     "Invalid JSON - missing quotes",
			input:    `{key:value}`,
			expected: false,
		},
		{
			name:     "Invalid JSON - trailing comma",
			input:    `{"key":"value",}`,
			expected: false,
		},
		{
			name:     "Invalid JSON - missing closing brace",
			input:    `{"key":"value"`,
			expected: false,
		},
		{
			name:     "Invalid JSON - single quotes",
			input:    `{'key':'value'}`,
			expected: false,
		},
		{
			name:     "Invalid JSON - plain text",
			input:    `not json at all`,
			expected: false,
		},
		{
			name:     "Empty string",
			input:    ``,
			expected: false,
		},
		{
			name:     "Whitespace only",
			input:    `   `,
			expected: false,
		},
		{
			name:     "Valid JSON with whitespace",
			input:    `  {"key": "value"}  `,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidJSON(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
