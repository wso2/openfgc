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

package authresource

// =============================================================================
// Request types — what we send to the server.
// =============================================================================

// AuthResourceCreateRequest is the body for POST /consents/{consentId}/authorizations.
// Type defaults to "primary" and Status defaults to "APPROVED" when absent.
type AuthResourceCreateRequest struct {
	UserID    *string     `json:"userId,omitempty"`
	Type      string      `json:"type,omitempty"`
	Status    string      `json:"status,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// AuthResourceUpdateRequest is the body for PUT /consents/{consentId}/authorizations/{authorizationId}.
// At least one field must be non-zero; server returns AR-4002 otherwise.
type AuthResourceUpdateRequest struct {
	UserID    *string     `json:"userId,omitempty"`
	Type      string      `json:"type,omitempty"`
	Status    string      `json:"status,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// =============================================================================
// Response types — what we receive from the server.
// =============================================================================

// AuthResourceResponse is returned by POST, GET, and PUT /consents/{consentId}/authorizations.
type AuthResourceResponse struct {
	ID          string      `json:"id"`
	UserID      *string     `json:"userId,omitempty"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources,omitempty"`
}

// ConsentStatusResponse is a minimal subset of the consent response used to verify
// that consent status transitions are triggered correctly by auth resource operations.
type ConsentStatusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ErrorResponse is the structured error body the server returns on HTTP 4xx/5xx.
type ErrorResponse struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
	TraceID     string `json:"traceId"`
}
