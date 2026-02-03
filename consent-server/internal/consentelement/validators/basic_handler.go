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

// BasicElementTypeHandler handles "basic" consent elements
// Basic type has no mandatory properties - all properties are optional
type BasicElementTypeHandler struct{}

// GetType returns the type identifier
func (handler *BasicElementTypeHandler) GetType() string {
	return "basic"
}

// ValidateProperties validates properties for basic type
// Basic type has no mandatory properties, so validation always passes
func (handler *BasicElementTypeHandler) ValidateProperties(properties map[string]string) []ValidationError {
	// Basic type: no mandatory properties
	// All properties are optional
	return nil
}

// ProcessProperties processes properties for basic type
// No special processing needed for basic type
func (handler *BasicElementTypeHandler) ProcessProperties(properties map[string]string) map[string]string {
	// Return as-is, no transformation needed
	return properties
}

// GetPropertySpec returns the property specification for basic type
func (handler *BasicElementTypeHandler) GetPropertySpec() []ElementPropertySpec {
	return []ElementPropertySpec{
		{
			Name:        "validationSchema",
			Required:    false,
			Type:        "json",
			Description: "JSON schema for validation",
			Example:     `{"type":"string","minLength":1}`,
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
