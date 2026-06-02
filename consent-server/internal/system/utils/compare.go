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

package utils

import (
	"encoding/json"
	"fmt"
)

// PointersEqual compares pointer values by dereferencing non-nil pointers.
func PointersEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// CanonicalJSONValue marshals a value and normalizes valid JSON for semantic comparisons.
func CanonicalJSONValue(value interface{}) string {
	if value == nil {
		return ""
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return CanonicalJSONString(string(bytes))
}

// CanonicalJSONString normalizes valid JSON strings while preserving invalid JSON as-is.
func CanonicalJSONString(value string) string {
	if value == "" {
		return ""
	}
	var normalized interface{}
	if err := json.Unmarshal([]byte(value), &normalized); err != nil {
		return value
	}
	bytes, err := json.Marshal(normalized)
	if err != nil {
		return value
	}
	return string(bytes)
}
