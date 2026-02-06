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

// ResourceFieldElementTypeHandler handles "resource-field" consent elements
// Resource field type requires resourcePath and jsonPath to be present
type ResourceFieldElementTypeHandler struct{}

// GetType returns the type identifier
func (handler *ResourceFieldElementTypeHandler) GetType() string {
	return "resource-field"
}

// validates properties for resource-field
// Mandatory: resourcePath and jsonPath must be present
func (handler *ResourceFieldElementTypeHandler) ValidateProperties(properties map[string]string) []ValidationError {
	var errors []ValidationError

	// resourcePath is MANDATORY
	if path, exists := properties["resourcePath"]; !exists || path == "" {
		errors = append(errors, ValidationError{
			Field:   "resourcePath",
			Message: "resourcePath is required for resource-field",
		})
	}

	// jsonPath is MANDATORY
	if path, exists := properties["jsonPath"]; !exists || path == "" {
		errors = append(errors, ValidationError{
			Field:   "jsonPath",
			Message: "jsonPath is required for resource-field",
		})
	}

	return errors
}

// ProcessProperties processes properties for resource-field
// Basic processing, could add defaults or validation
func (handler *ResourceFieldElementTypeHandler) ProcessProperties(properties map[string]string) map[string]string {
	// Return as-is
	return properties
}

// GetPropertySpec returns the property specification for resource-field
func (handler *ResourceFieldElementTypeHandler) GetPropertySpec() []ElementPropertySpec {
	return []ElementPropertySpec{
		{
			Name:        "resourcePath",
			Required:    true,
			Type:        "string",
			Description: "Resource path (required)",
			Example:     "/accounts",
		},
		{
			Name:        "jsonPath",
			Required:    true,
			Type:        "string",
			Description: "JSON path for extraction (required)",
			Example:     "Data.amount",
		},
		{
			Name:        "validationSchema",
			Required:    false,
			Type:        "json",
			Description: "Optional validation schema",
			Example:     `{"type":"number"}`,
		},
	}
}
