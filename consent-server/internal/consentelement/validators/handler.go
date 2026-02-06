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

import "encoding/json"

// ValidationError represents a single validation error for a property
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ElementPropertySpec defines metadata about a property for a element type
type ElementPropertySpec struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Type        string `json:"type"` // "string", "json", etc.
	Description string `json:"description"`
	Example     string `json:"example"`
}

// ElementTypeHandler defines behavior for a specific consent element type
type ElementTypeHandler interface {
	// GetType returns the type string this handler manages (e.g., "basic", "json-payload", "resource-field")
	GetType() string

	// ValidateProperties checks if required properties are present and valid
	// Returns ValidationErrors if validation fails, empty slice if valid
	ValidateProperties(properties map[string]string) []ValidationError

	// ProcessProperties transforms/normalizes properties before storage
	// Useful for sanitization, defaults, or derived values
	ProcessProperties(properties map[string]string) map[string]string

	// GetPropertySpec returns the schema/spec for this handler's properties
	// Useful for documentation and dynamic UI generation
	GetPropertySpec() []ElementPropertySpec
}

// Helper function to validate JSON string
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
