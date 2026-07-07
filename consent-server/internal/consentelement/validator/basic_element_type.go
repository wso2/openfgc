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

import "github.com/wso2/openfgc/consent-server/internal/consentelement/model"

// BasicElementType handles "basic" consent elements.
// Basic type has no mandatory properties — all properties are optional.
type BasicElementType struct{}

func (t *BasicElementType) GetType() string {
	return model.ElementTypeBasic
}

// ValidateSchema accepts any schema value — schema is optional for basic elements.
func (t *BasicElementType) ValidateSchema(schema *string) *ValidationError {
	return nil
}

// ValidateProperties is reserved for future property-level constraints.
func (t *BasicElementType) ValidateProperties(properties map[string]string) []ValidationError {
	return nil
}
