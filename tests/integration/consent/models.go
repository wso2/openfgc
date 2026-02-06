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

// ConsentPurposeItem represents a consent purpose (logical grouping of elements) in the request/response
type ConsentPurposeItem struct {
	Name     string                       `json:"name"`
	Elements []ConsentPurposeApprovalItem `json:"elements"`
}

// ConsentPurposeApprovalItem represents an element approval within a purpose
type ConsentPurposeApprovalItem struct {
	Name           string      `json:"name"`
	Value          interface{} `json:"value,omitempty"`
	IsUserApproved bool        `json:"isUserApproved"`
}

// AuthorizationRequest represents authorization data in consent creation/update
type AuthorizationRequest struct {
	UserID         string   `json:"userId"`
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	Resources      []string `json:"resources,omitempty"`
	Permissions    []string `json:"permissions,omitempty"`
	ExpirationDate string   `json:"expirationDate,omitempty"`
}

// ConsentCreateRequest represents the payload for creating a consent
type ConsentCreateRequest struct {
	Type               string                 `json:"type"`
	Purposes           []ConsentPurposeItem   `json:"purposes,omitempty"`
	Authorizations     []AuthorizationRequest `json:"authorizations"`
	Attributes         map[string]string      `json:"attributes,omitempty"`
	ValidityTime       int64                  `json:"validityTime,omitempty"`
	RecurringIndicator bool                   `json:"recurringIndicator,omitempty"`
	Frequency          int                    `json:"frequency,omitempty"`
}

// ConsentUpdateRequest represents the payload for updating a consent
type ConsentUpdateRequest struct {
	Type               string                 `json:"type,omitempty"`
	Purposes           []ConsentPurposeItem   `json:"purposes"`       // Remove omitempty to allow empty arrays for removal
	Authorizations     []AuthorizationRequest `json:"authorizations"` // Remove omitempty to allow empty arrays for removal
	Attributes         map[string]string      `json:"attributes"`     // Remove omitempty to allow empty maps for removal
	ValidityTime       *int64                 `json:"validityTime,omitempty"`
	RecurringIndicator *bool                  `json:"recurringIndicator,omitempty"`
	Frequency          *int                   `json:"frequency,omitempty"`
}

// ConsentRevokeRequest represents the payload for revoking a consent
type ConsentRevokeRequest struct {
	Reason   string `json:"reason,omitempty"`
	ActionBy string `json:"actionBy"`
}

// AuthorizationResponse represents authorization data in consent response
type AuthorizationResponse struct {
	ID          string      `json:"id"`
	UserID      *string     `json:"userId,omitempty"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources,omitempty"`
}

// ConsentResponse represents the API response for a consent
type ConsentResponse struct {
	ID                         string                  `json:"id"`
	ClientID                   string                  `json:"clientId"`
	Type                       string                  `json:"type"`
	Status                     string                  `json:"status"`
	Purposes                   []ConsentPurposeItem    `json:"purposes"`
	Authorizations             []AuthorizationResponse `json:"authorizations"`
	Attributes                 map[string]string       `json:"attributes"`
	ValidityTime               *int64                  `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                   `json:"recurringIndicator,omitempty"`
	Frequency                  *int                    `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                  `json:"dataAccessValidityDuration,omitempty"`
	CreatedTime                int64                   `json:"createdTime"`
	UpdatedTime                int64                   `json:"updatedTime"`
}

// ConsentListResponse represents the API response for listing consents
type ConsentListResponse struct {
	Data []ConsentResponse `json:"data"`
	Meta struct {
		Total  int `json:"total"`
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
		Count  int `json:"count"`
	} `json:"meta"`
}

// ErrorResponse represents error responses from the API
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ConsentValidateRequest represents the payload for validating a consent
type ConsentValidateRequest struct {
	Headers         map[string]interface{} `json:"headers,omitempty"`
	Payload         map[string]interface{} `json:"payload,omitempty"`
	ElectedResource string                 `json:"electedResource,omitempty"`
	ConsentID       string                 `json:"consentId"`
	UserID          string                 `json:"userId,omitempty"`
	ClientID        string                 `json:"clientId,omitempty"`
	ResourceParams  *struct {
		Resource   string `json:"resource,omitempty"`
		HTTPMethod string `json:"httpMethod,omitempty"`
		Context    string `json:"context,omitempty"`
	} `json:"resourceParams,omitempty"`
}

// ConsentValidateResponse represents the API response for consent validation
type ConsentValidateResponse struct {
	IsValid            bool                   `json:"isValid"`
	ModifiedPayload    interface{}            `json:"modifiedPayload,omitempty"`
	ErrorCode          int                    `json:"errorCode,omitempty"`
	ErrorMessage       string                 `json:"errorMessage,omitempty"`
	ErrorDescription   string                 `json:"errorDescription,omitempty"`
	ConsentInformation *ConsentValidateDetail `json:"consentInformation,omitempty"`
}

// ConsentValidateDetail represents consent information in validate response
type ConsentValidateDetail struct {
	ID                         string                  `json:"id"`
	Type                       string                  `json:"type"`
	ClientID                   string                  `json:"clientId"`
	Status                     string                  `json:"status"`
	CreatedTime                int64                   `json:"createdTime"`
	UpdatedTime                int64                   `json:"updatedTime"`
	Purposes                   []ConsentPurposeItem    `json:"purposes"`
	Authorizations             []AuthorizationResponse `json:"authorizations"`
	Attributes                 map[string]string       `json:"attributes"`
	ValidityTime               *int64                  `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                   `json:"recurringIndicator,omitempty"`
	Frequency                  *int                    `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                  `json:"dataAccessValidityDuration,omitempty"`
}
