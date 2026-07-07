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

package validator

import (
	"strings"

	"github.com/wso2/openfgc/consent-server/internal/consentelement/model"
)

// XMLElementType handles "xml" consent elements.
// Schema is required and must be a non-empty string.
type XMLElementType struct{}

func (t *XMLElementType) GetType() string {
	return model.ElementTypeXML
}

// ValidateSchema requires a non-nil, non-empty (and non-whitespace) schema for xml elements.
func (t *XMLElementType) ValidateSchema(schema *string) *ValidationError {
	if schema == nil || strings.TrimSpace(*schema) == "" {
		return &ValidationError{Field: "schema", Message: "schema is required for xml elements"}
	}
	return nil
}

// ValidateProperties is reserved for future property-level constraints.
func (t *XMLElementType) ValidateProperties(properties map[string]string) []ValidationError {
	return nil
}
