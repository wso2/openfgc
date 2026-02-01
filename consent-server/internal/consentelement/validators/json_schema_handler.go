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

// JsonSchemaElementTypeHandler handles "json-payload-type" consent elements
// JSON payload type requires validationSchema property to be present and valid JSON
type JsonSchemaElementTypeHandler struct{}

// GetType returns the type identifier
func (handler *JsonSchemaElementTypeHandler) GetType() string {
	return "json-payload-type"
}

// ValidateProperties validates properties for json-payload-type
// Mandatory: validationSchema must be present and valid JSON
func (handler *JsonSchemaElementTypeHandler) ValidateProperties(properties map[string]string) []ValidationError {
	var errors []ValidationError

	// validationSchema is MANDATORY
	schema, exists := properties["validationSchema"]
	if !exists || schema == "" {
		errors = append(errors, ValidationError{
			Field:   "validationSchema",
			Message: "validationSchema is required for json-payload-type",
		})
		return errors
	}

	// Validate that validationSchema is valid JSON
	if !isValidJSON(schema) {
		errors = append(errors, ValidationError{
			Field:   "validationSchema",
			Message: "validationSchema must be valid JSON",
		})
	}

	return errors
}

// ProcessProperties processes properties for json-payload-type
// Could normalize JSON, add defaults, etc.
func (handler *JsonSchemaElementTypeHandler) ProcessProperties(properties map[string]string) map[string]string {
	// Return as-is, basic processing
	return properties
}

// GetPropertySpec returns the property specification for json-payload-type
func (handler *JsonSchemaElementTypeHandler) GetPropertySpec() []ElementPropertySpec {
	return []ElementPropertySpec{
		{
			Name:        "validationSchema",
			Required:    true,
			Type:        "json",
			Description: "JSON schema for validation (required)",
			Example:     `{"type":"object","properties":{"name":{"type":"string"}}}`,
		},
		{
			Name:        "resourcePath",
			Required:    false,
			Type:        "string",
			Description: "Resource path for this element",
			Example:     "/accounts",
		},
		{
			Name:        "jsonPath",
			Required:    false,
			Type:        "string",
			Description: "JSON path for data extraction",
			Example:     "Data.amount",
		},
	}
}
