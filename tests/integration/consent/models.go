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

package consent

import "encoding/json"

// =============================================================================
// Request types — what we send to the server.
// These mirror the server's internal/consent/model/consent.go API types exactly.
// GroupID is NOT in the body; it is sent as the "group-id" request header.
// =============================================================================

// ConsentCreateRequest is the body for POST /consents.
type ConsentCreateRequest struct {
	Type                       string                 `json:"type"`
	ExpirationTime             *int64                 `json:"expirationTime,omitempty"`
	Frequency                  *int                   `json:"frequency,omitempty"`
	RecurringIndicator         *bool                  `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                 `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string      `json:"attributes,omitempty"`
	Purposes                   []PurposeRefRequest    `json:"purposes,omitempty"`
	Authorizations             []AuthorizationRequest `json:"authorizations,omitempty"`
}

// ConsentUpdateRequest is the body for PUT /consents/{consentId}.
// Authorizations, Purposes, and Attributes intentionally omit omitempty — sending
// an explicit empty value removes all existing entries of that type.
type ConsentUpdateRequest struct {
	Type                       string                 `json:"type,omitempty"`
	ExpirationTime             *int64                 `json:"expirationTime,omitempty"`
	Frequency                  *int                   `json:"frequency,omitempty"`
	RecurringIndicator         *bool                  `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                 `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string      `json:"attributes"`
	Purposes                   []PurposeRefRequest    `json:"purposes"`
	Authorizations             []AuthorizationRequest `json:"authorizations"`
}

// ConsentRevokeRequest is the body for PUT /consents/{consentId}/revoke.
type ConsentRevokeRequest struct {
	ActionBy         string `json:"actionBy"`
	RevocationReason string `json:"revocationReason,omitempty"`
}

// PurposeRefRequest references a purpose by name in a consent body.
// Version follows "v1", "v2", … format; omit to use the latest version.
type PurposeRefRequest struct {
	Name     string                   `json:"name"`
	Version  *string                  `json:"version,omitempty"`
	Elements []ElementApprovalRequest `json:"elements"`
}

// ElementApprovalRequest is one element approval within a purpose reference.
// Namespace defaults to "default" on the server when absent.
type ElementApprovalRequest struct {
	Name      string      `json:"name"`
	Namespace string      `json:"namespace,omitempty"`
	Approved  bool        `json:"approved"`
	Value     interface{} `json:"value,omitempty"`
}

// AuthorizationRequest is one authorization entry in a consent body.
// Type defaults to "default" and Status defaults to "APPROVED" when absent.
type AuthorizationRequest struct {
	UserID    string      `json:"userId,omitempty"`
	Type      string      `json:"type,omitempty"`
	Status    string      `json:"status,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// =============================================================================
// Response types — what we receive from the server.
// Field names mirror the server's JSON tags exactly.
// =============================================================================

// AuthorizationResponse is one authorization entry in a consent response.
type AuthorizationResponse struct {
	ID          string      `json:"id"`
	UserID      *string     `json:"userId,omitempty"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources,omitempty"`
}

// ElementApprovalResponse is one element within a purpose in a consent response.
type ElementApprovalResponse struct {
	ElementID string      `json:"elementId"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Version   string      `json:"version"`
	Mandatory bool        `json:"mandatory"`
	Approved  bool        `json:"approved"`
	Value     interface{} `json:"value,omitempty"`
}

// PurposeResponse is one purpose entry in a consent response.
type PurposeResponse struct {
	PurposeID string                    `json:"purposeId"`
	Name      string                    `json:"name"`
	Version   string                    `json:"version"`
	Elements  []ElementApprovalResponse `json:"elements"`
}

// ConsentResponse is returned by POST, GET, and PUT /consents.
type ConsentResponse struct {
	ID                         string                       `json:"id"`
	GroupID                    string                       `json:"groupId"`
	Type                       string                       `json:"type"`
	Status                     string                       `json:"status"`
	CreatedTime                int64                        `json:"createdTime"`
	UpdatedTime                int64                        `json:"updatedTime"`
	ExpirationTime             *int64                       `json:"expirationTime,omitempty"`
	Frequency                  *int                         `json:"frequency,omitempty"`
	RecurringIndicator         *bool                        `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                       `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string            `json:"attributes"`
	Purposes                   []PurposeResponse            `json:"purposes"`
	Authorizations             []AuthorizationResponse      `json:"authorizations"`
	StatusHistory              []ConsentStatusAuditResponse `json:"statusHistory,omitempty"`
}

// ConsentStatusAuditResponse is one status audit entry returned when includeStatusHistory=true.
type ConsentStatusAuditResponse struct {
	StatusAuditID  string  `json:"statusAuditId"`
	PreviousStatus *string `json:"previousStatus,omitempty"`
	CurrentStatus  string  `json:"currentStatus"`
	ActionTime     int64   `json:"actionTime"`
	ActionBy       *string `json:"actionBy,omitempty"`
	Reason         *string `json:"reason,omitempty"`
}

// ConsentHistoryResponse is one consent history entry.
type ConsentHistoryResponse struct {
	HistoryID  string          `json:"historyId"`
	ActionTime int64           `json:"actionTime"`
	ActionBy   *string         `json:"actionBy,omitempty"`
	Reason     *string         `json:"reason,omitempty"`
	Snapshot   json.RawMessage `json:"snapshot,omitempty"`
}

// ConsentHistoryListResponse is returned by GET /consents/{consentId}/history.
type ConsentHistoryListResponse struct {
	ID      string                   `json:"id"`
	History []ConsentHistoryResponse `json:"history"`
}

// ConsentListResponse is returned by GET /consents.
type ConsentListResponse struct {
	Data     []ConsentResponse `json:"data"`
	Metadata PageMetadata      `json:"metadata"`
}

// PageMetadata carries pagination state in all list responses.
type PageMetadata struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
	Limit  int `json:"limit"`
}

// ConsentRevokeResponse is returned by PUT /consents/{consentId}/revoke.
type ConsentRevokeResponse struct {
	ActionTime       int64  `json:"actionTime"`
	ActionBy         string `json:"actionBy"`
	RevocationReason string `json:"revocationReason,omitempty"`
}

// ConsentAttributeSearchResponse is returned by GET /consents/attributes.
type ConsentAttributeSearchResponse struct {
	ConsentIDs []string `json:"consentIds"`
	Count      int      `json:"count"`
}

// =============================================================================
// Validate request / response types — POST /consents/validate
// =============================================================================

// ConsentValidateRequest is the body for POST /consents/validate.
type ConsentValidateRequest struct {
	ConsentID string `json:"consentId"`
	GroupID   string `json:"groupId,omitempty"`
	UserID    string `json:"userId,omitempty"`
}

// ConsentValidatePurposeElementResponse is one element inside the validate response's
// consentInformation. It extends the regular element with enriched definition fields.
type ConsentValidatePurposeElementResponse struct {
	ElementID   string            `json:"elementId"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Version     string            `json:"version"`
	Mandatory   bool              `json:"mandatory"`
	Approved    bool              `json:"approved"`
	Value       interface{}       `json:"value,omitempty"`
	Type        string            `json:"type,omitempty"`
	DisplayName *string           `json:"displayName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// ConsentValidatePurposeResponse is one purpose inside the validate response's
// consentInformation. It extends the regular purpose with enriched definition fields.
type ConsentValidatePurposeResponse struct {
	PurposeID   string                                  `json:"purposeId"`
	Name        string                                  `json:"name"`
	Version     string                                  `json:"version"`
	DisplayName *string                                 `json:"displayName,omitempty"`
	Description *string                                 `json:"description,omitempty"`
	Properties  map[string]string                       `json:"properties,omitempty"`
	Elements    []ConsentValidatePurposeElementResponse `json:"elements"`
}

// ConsentValidateInfo is the consent payload embedded inside ConsentValidateResponse.
type ConsentValidateInfo struct {
	ID                         string                           `json:"id"`
	GroupID                    string                           `json:"groupId"`
	Type                       string                           `json:"type"`
	Status                     string                           `json:"status"`
	CreatedTime                int64                            `json:"createdTime"`
	UpdatedTime                int64                            `json:"updatedTime"`
	ExpirationTime             *int64                           `json:"expirationTime,omitempty"`
	Frequency                  *int                             `json:"frequency,omitempty"`
	RecurringIndicator         *bool                            `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                           `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string                `json:"attributes"`
	Purposes                   []ConsentValidatePurposeResponse `json:"purposes"`
	Authorizations             []AuthorizationResponse          `json:"authorizations"`
}

// ConsentValidateResponse is returned by POST /consents/validate.
// The HTTP status is always 200 when the consent exists — check isValid.
type ConsentValidateResponse struct {
	IsValid          bool                 `json:"isValid"`
	ErrorCode        int                  `json:"errorCode,omitempty"`
	ErrorMessage     string               `json:"errorMessage,omitempty"`
	ErrorDescription string               `json:"errorDescription,omitempty"`
	ConsentInfo      *ConsentValidateInfo `json:"consentInformation,omitempty"`
}

// ErrorResponse is the structured error body the server returns on HTTP 4xx/5xx.
type ErrorResponse struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
	TraceID     string `json:"traceId"`
}
