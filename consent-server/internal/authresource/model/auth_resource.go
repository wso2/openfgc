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

// Package model provides data models for authorization resources.
package model

const (
	// DefaultAuthType is used when the caller does not specify an authorization type.
	DefaultAuthType = "default"

	// --- First-class authorization types for consent delegation ---

	// AuthTypePrimary indicates self-consent: the person consenting for themselves.
	AuthTypePrimary = "primary"

	// AuthTypeDelegate indicates a person giving consent on behalf of another
	// (e.g., a parent consenting for a child).
	AuthTypeDelegate = "delegate"

	// AuthTypeDelegateSubject indicates a person who is incapable of providing
	// consent by themselves (e.g., a minor child).
	AuthTypeDelegateSubject = "delegate_subject"
)

// FirstClassAuthTypes is the set of auth types that OpenFGC validates.
// Custom types (anything not in this set) are stored and filterable but not validated.
var FirstClassAuthTypes = map[string]bool{
	AuthTypePrimary:         true,
	AuthTypeDelegate:        true,
	AuthTypeDelegateSubject: true,
}

// IsFirstClassAuthType reports whether the given type is a recognized first-class auth type.
func IsFirstClassAuthType(authType string) bool {
	return FirstClassAuthTypes[authType]
}

// =============================================================================
// DB types — store layer only, db tags, no json tags
// =============================================================================

// AuthResource is one row from the CONSENT_AUTH_RESOURCE table.
// Resources is stored as a JSON blob; use AuthResourceOutput for the parsed form.
type AuthResource struct {
	AuthID      string  `db:"AUTH_ID"`
	ConsentID   string  `db:"CONSENT_ID"`
	AuthType    string  `db:"AUTH_TYPE"`
	UserID      *string `db:"USER_ID"`
	AuthStatus  string  `db:"AUTH_STATUS"`
	UpdatedTime int64   `db:"UPDATED_TIME"`
	Resources   *string `db:"RESOURCES"` // JSON-encoded BLOB; nil when not set
	OrgID       string  `db:"ORG_ID"`
}

// =============================================================================
// Service input types — handler → service, no tags
// =============================================================================

// CreateAuthResourceInput is the input to the CreateAuthResource service method.
// AuthType defaults to DefaultAuthType ("default") when empty.
// AuthStatus defaults to the configured approved state when empty.
type CreateAuthResourceInput struct {
	AuthType   string // optional; defaults to DefaultAuthType
	UserID     *string
	AuthStatus string      // optional; defaults to configured approved state
	Resources  interface{} // arbitrary value; service JSON-marshals before storing
}

// UpdateAuthResourceInput is the input to the UpdateAuthResource service method.
// Only non-zero fields are applied; an empty string leaves the existing value unchanged.
type UpdateAuthResourceInput struct {
	AuthType   string
	UserID     *string
	AuthStatus string
	Resources  interface{}
}

// =============================================================================
// Service return types — service → handler, no tags
// =============================================================================

// AuthResourceOutput is the service-layer representation of one authorization resource.
// Resources is already parsed from JSON into an interface{}.
type AuthResourceOutput struct {
	AuthID      string
	ConsentID   string
	AuthType    string
	UserID      *string
	AuthStatus  string
	UpdatedTime int64
	Resources   interface{} // parsed from JSON; nil when not set
	OrgID       string
}

// AuthResourceListOutput is the return type from ListAuthResources.
type AuthResourceListOutput struct {
	Data []AuthResourceOutput
}

// =============================================================================
// API request types — HTTP boundary, handler only, json tags, no db tags
// =============================================================================

// AuthResourceCreateRequest is the body for POST /consents/{consentId}/authorizations.
// Type is optional — when absent the server uses DefaultAuthType ("default").
type AuthResourceCreateRequest struct {
	UserID    *string     `json:"userId,omitempty"`
	Type      string      `json:"type,omitempty"`   // optional; defaults to "default"
	Status    string      `json:"status,omitempty"` // optional; defaults to "APPROVED"
	Resources interface{} `json:"resources,omitempty"`
}

// AuthResourceUpdateRequest is the body for PUT /consents/{consentId}/authorizations/{authId}.
// Type is optional — when absent the existing type is preserved.
type AuthResourceUpdateRequest struct {
	UserID    *string     `json:"userId,omitempty"`
	Type      string      `json:"type,omitempty"`
	Status    string      `json:"status,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// =============================================================================
// API response types — HTTP boundary, handler only, json tags, no db tags
// =============================================================================

// AuthResourceResponse is the response body for authorization resource endpoints.
type AuthResourceResponse struct {
	ID          string      `json:"id"`
	UserID      *string     `json:"userId,omitempty"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources,omitempty"`
}
