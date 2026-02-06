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

package log

// ContextKey is a custom type for context keys to avoid collisions.
// Using an unexported struct makes it truly unique and prevents collisions.
type ContextKey struct{ name string }

var (
	// ContextKeyTraceID is the context key for storing the trace ID.
	ContextKeyTraceID = ContextKey{"trace_id"}
)

const (
	// LoggerKeyComponentName is the key used to identify the component name in the logger.
	LoggerKeyComponentName = "component"
	// LoggerKeyTraceID is the key used to identify the trace ID (correlation ID) in the logger.
	LoggerKeyTraceID = "correlation-id"
)
