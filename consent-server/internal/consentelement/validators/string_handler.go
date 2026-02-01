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

// StringElementTypeHandler handles "string-type" consent elements
// String type has no mandatory properties - all properties are optional
type StringElementTypeHandler struct{}

// GetType returns the type identifier
func (handler *StringElementTypeHandler) GetType() string {
	return "string-type"
}

// ValidateProperties validates properties for string type
// String type has no mandatory properties, so validation always passes
func (handler *StringElementTypeHandler) ValidateProperties(properties map[string]string) []ValidationError {
	// String type: no mandatory properties
	// All properties are optional
	return nil
}

// ProcessProperties processes properties for string type
// No special processing needed for string type
func (handler *StringElementTypeHandler) ProcessProperties(properties map[string]string) map[string]string {
	// Return as-is, no transformation needed
	return properties
}

// GetPropertySpec returns the property specification for string type
func (handler *StringElementTypeHandler) GetPropertySpec() []ElementPropertySpec {
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
