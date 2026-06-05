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

// Package model provides data models for consents.
package model

import (
	"time"

	authmodel "github.com/wso2/openfgc/internal/authresource/model"
)

// =============================================================================
// DB types — store layer only, db tags, no json tags
// =============================================================================

// Consent is one row from the CONSENT table.
type Consent struct {
	ConsentID                  string `db:"CONSENT_ID"`
	CreatedTime                int64  `db:"CREATED_TIME"`
	UpdatedTime                int64  `db:"UPDATED_TIME"`
	GroupID                    string `db:"GROUP_ID"`
	ConsentType                string `db:"CONSENT_TYPE"`
	CurrentStatus              string `db:"CURRENT_STATUS"`
	ConsentFrequency           *int   `db:"CONSENT_FREQUENCY"`
	ExpirationTime             *int64 `db:"EXPIRATION_TIME"` // Unix ms timestamp; nil = no expiry
	RecurringIndicator         *bool  `db:"RECURRING_INDICATOR"`
	DataAccessValidityDuration *int64 `db:"DATA_ACCESS_VALIDITY_DURATION"`
	OrgID                      string `db:"ORG_ID"`
}

// GetCreatedTime returns CreatedTime as a time.Time (stored as Unix ms).
func (c *Consent) GetCreatedTime() time.Time {
	return time.UnixMilli(c.CreatedTime)
}

// GetUpdatedTime returns UpdatedTime as a time.Time (stored as Unix ms).
func (c *Consent) GetUpdatedTime() time.Time {
	return time.UnixMilli(c.UpdatedTime)
}

// ConsentPurposeMapping is one row from the PURPOSE_CONSENT_MAPPING table.
// The link is to a specific PURPOSE_VERSION_ID rather than a PURPOSE_ID.
type ConsentPurposeMapping struct {
	ConsentID        string `db:"CONSENT_ID"`
	PurposeVersionID string `db:"PURPOSE_VERSION_ID"`
	OrgID            string `db:"ORG_ID"`
}

// ConsentElementApproval is one row from the CONSENT_ELEMENT_APPROVAL table.
// Tracks whether a user approved a specific element version within a specific purpose version.
type ConsentElementApproval struct {
	ConsentID        string  `db:"CONSENT_ID"`
	PurposeVersionID string  `db:"PURPOSE_VERSION_ID"`
	ElementVersionID string  `db:"ELEMENT_VERSION_ID"`
	Approved         bool    `db:"APPROVED"`
	Value            *string `db:"VALUE"` // user-provided value stored as-is; plain string for basic/xml elements, JSON string for json elements; nil if not set
	OrgID            string  `db:"ORG_ID"`
}

// =============================================================================
// Store join result types — results of JOIN queries, used only between the
// store and service layers; no tags.
// =============================================================================

// ConsentPurposeRow is returned by GetConsentPurposeMappingsByConsentID.
// Joins PURPOSE_CONSENT_MAPPING with PURPOSE to carry purpose metadata alongside the mapping.
type ConsentPurposeRow struct {
	ConsentID        string
	PurposeVersionID string
	PurposeID        string
	PurposeName      string
	PurposeGroupID   string
	PurposeVersion   int
	DisplayName      *string
	Description      *string
	OrgID            string
}

// ConsentApprovalRow is returned by GetPurposeElementApprovalsByConsentID.
// Joins CONSENT_ELEMENT_APPROVAL with ELEMENT and PURPOSE_ELEMENT_MAPPING to carry
// element metadata alongside the user's approval state.
type ConsentApprovalRow struct {
	ConsentID          string
	PurposeVersionID   string
	ElementVersionID   string
	ElementID          string
	ElementName        string
	ElementNamespace   string
	ElementVersionNum  int
	ElementType        string
	ElementDisplayName *string
	ElementDescription *string
	Mandatory          bool
	Approved           bool
	Value              *string
	OrgID              string
}

// =============================================================================
// Service input types — handler → service, no tags
// =============================================================================

// PurposeRef identifies a consent purpose in a create/update request.
// When Version is nil the service resolves to the latest purpose version.
// PurposeName matches the `name` field in the API request body.
type PurposeRef struct {
	PurposeName string
	Version     *int // nil = use latest; parsed from the "v1" / "v2" string in the API request
}

// ElementApprovalInput captures one element's approval data from the API request.
// Name and Namespace together identify the element within the purpose version.
// Namespace is always set to a non-empty value by the handler (defaults to "default").
type ElementApprovalInput struct {
	Name      string
	Namespace string
	Approved  bool
	Value     interface{} // arbitrary user value; service stores as string for basic/xml, JSON-marshals for json elements
}

// ConsentPurposeInput is the handler→service representation of one purpose in a create/update.
type ConsentPurposeInput struct {
	PurposeRef PurposeRef
	Elements   []ElementApprovalInput
}

// CreateConsentInput is the input to the CreateConsent service method.
// GroupID is read from the group-id request header.
type CreateConsentInput struct {
	GroupID                    string
	ConsentType                string
	ExpirationTime             *int64
	ConsentFrequency           *int
	RecurringIndicator         *bool
	DataAccessValidityDuration *int64
	Attributes                 map[string]string
	Purposes                   []ConsentPurposeInput
	Authorizations             []authmodel.CreateAuthResourceInput
}

// UpdateConsentInput is the input to the UpdateConsent service method.
// All slice/map fields are non-nil to distinguish "caller sent empty list" from "caller omitted field".
type UpdateConsentInput struct {
	ConsentType                string
	ExpirationTime             *int64
	ConsentFrequency           *int
	RecurringIndicator         *bool
	DataAccessValidityDuration *int64
	Attributes                 map[string]string
	Purposes                   []ConsentPurposeInput
	Authorizations             []authmodel.CreateAuthResourceInput
}

// ConsentSearchFilter holds query parameters for the SearchConsents service method.
type ConsentSearchFilter struct {
	ConsentIDs       []string
	GroupIDs         []string // replaces ClientIDs
	ConsentTypes     []string
	ConsentStatuses  []string
	UserIDs          []string
	PurposeName      string // filter consents that reference this purpose name
	PurposeVersion   *int   // combined with PurposeName to pin a specific version
	ElementName      string // filter consents whose purpose contains this element
	ElementNamespace string // combined with ElementName
	ElementVersion   *int   // combined with ElementName/ElementNamespace
	FromTime         *int64 // Unix ms lower bound on UPDATED_TIME
	ToTime           *int64 // Unix ms upper bound on UPDATED_TIME
	Limit            int
	Offset           int
	OrgID            string
}

// =============================================================================
// Service return types — service → handler, no tags
// =============================================================================

// ConsentElementApprovalOutput is the service-layer representation of one element within a consent purpose.
// Combines the element's definition (from PURPOSE_ELEMENT_MAPPING + ELEMENT tables) with the
// user's approval data (from CONSENT_ELEMENT_APPROVAL).
//
// Value is the raw stored string — plain for basic/xml elements, JSON string for json elements.
// The handler interprets it based on ElementType when building the API response.
//
// DisplayName, Description, and Properties are populated from the ELEMENT table and are used by
// the validate endpoint's enriched response. Regular create/get/update handlers can ignore them.
type ConsentElementApprovalOutput struct {
	ElementVersionID string
	ElementID        string
	Name             string
	Namespace        string
	VersionNum       int
	ElementType      string // "basic", "json", "xml" — determines how Value is interpreted
	Mandatory        bool
	Approved         bool
	Value            *string           // raw stored value; plain string or JSON string depending on ElementType
	DisplayName      *string           // from element definition; used in validate response
	Description      *string           // from element definition; used in validate response
	Properties       map[string]string // from element definition; used in validate response
}

// ConsentPurposeOutput is the service-layer representation of one purpose in a consent.
type ConsentPurposeOutput struct {
	PurposeVersionID string
	PurposeID        string
	Name             string
	GroupID          string
	VersionNum       int
	DisplayName      *string
	Description      *string
	Properties       map[string]string
	Elements         []ConsentElementApprovalOutput
}

// ConsentOutput is the service-layer output for a consent after create/get/update.
type ConsentOutput struct {
	ConsentID                  string
	GroupID                    string
	ConsentType                string
	CurrentStatus              string
	ConsentFrequency           *int
	ExpirationTime             *int64
	RecurringIndicator         *bool
	DataAccessValidityDuration *int64
	CreatedTime                int64
	UpdatedTime                int64
	OrgID                      string
	Attributes                 map[string]string
	Purposes                   []ConsentPurposeOutput
	Authorizations             []authmodel.AuthResourceOutput
}

// ConsentListOutput is the return type from SearchConsents.
type ConsentListOutput struct {
	Data   []ConsentOutput
	Total  int
	Offset int
	Count  int
	Limit  int
}

// ConsentAttributeSearchOutput is the return type from SearchConsentsByAttribute.
type ConsentAttributeSearchOutput struct {
	ConsentIDs []string
	Count      int
}

// ConsentRevokeInput is the input to the RevokeConsent service method.
type ConsentRevokeInput struct {
	ActionBy string
	Reason   string
}

// ConsentRevokeOutput is the return type from RevokeConsent.
type ConsentRevokeOutput struct {
	ActionTime int64
	ActionBy   string
	Reason     string
}

// ResourceParamsInput holds optional resource context for the ValidateConsent service method.
type ResourceParamsInput struct {
	Resource   string
	HTTPMethod string
	Context    string
}

// ConsentValidateInput is the input to the ValidateConsent service method.
type ConsentValidateInput struct {
	ConsentID       string
	GroupID         string
	UserID          string
	Headers         map[string]interface{}
	Payload         map[string]interface{}
	ElectedResource string
	ResourceParams  *ResourceParamsInput
}

// ConsentValidateOutput is the return type from ValidateConsent.
// ConsentInfo reuses ConsentOutput so the handler can format the enriched validate response.
type ConsentValidateOutput struct {
	IsValid          bool
	ErrorCode        int
	ErrorMessage     string
	ErrorDescription string
	ConsentInfo      *ConsentOutput
}

// =============================================================================
// API request types — HTTP boundary, handler only, json tags, no db tags
// =============================================================================

// ConsentPurposeElementApprovalRequest is one element approval within a consent purpose request body.
// Namespace defaults to "default" when absent.
type ConsentPurposeElementApprovalRequest struct {
	Name      string      `json:"name"`
	Namespace string      `json:"namespace,omitempty"`
	Approved  bool        `json:"approved"`
	Value     interface{} `json:"value,omitempty"`
}

// ConsentPurposeRefRequest references a purpose version in a consent create/update body.
// Purposes are identified by name (not purposeId) in request bodies.
// Version follows the "v1", "v2", … format; when absent the service uses the latest version.
// purposeId is response-only and is not accepted in requests.
type ConsentPurposeRefRequest struct {
	Name     string                                 `json:"name"`
	Version  *string                                `json:"version,omitempty"`
	Elements []ConsentPurposeElementApprovalRequest `json:"elements"`
}

// AuthorizationRequest is one authorization entry in a consent create/update body.
// UserID is required — it identifies the user who performed the authorization.
// Type is optional and defaults to "default" when absent.
// Status is optional and defaults to "APPROVED" when absent.
type AuthorizationRequest struct {
	UserID    string      `json:"userId"`
	Type      string      `json:"type,omitempty"`
	Status    string      `json:"status,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// ConsentCreateRequest is the body for POST /consents.
// GroupID is not in the body — it is read from the group-id request header.
type ConsentCreateRequest struct {
	Type                       string                      `json:"type"`
	ExpirationTime             *int64                      `json:"expirationTime,omitempty"` // Unix milliseconds
	Frequency                  *int                        `json:"frequency,omitempty"`
	RecurringIndicator         *bool                       `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                      `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string           `json:"attributes,omitempty"`
	Purposes                   []ConsentPurposeRefRequest  `json:"purposes,omitempty"`
	Authorizations             []AuthorizationRequest      `json:"authorizations,omitempty"`
}

// ConsentUpdateRequest is the body for PUT /consents/{consentId}.
// Purposes, Authorizations, and Attributes intentionally omit `omitempty` so that sending an
// explicit empty array/map removes all existing entries.
type ConsentUpdateRequest struct {
	Type                       string                      `json:"type,omitempty"`
	ExpirationTime             *int64                      `json:"expirationTime,omitempty"` // Unix milliseconds
	Frequency                  *int                        `json:"frequency,omitempty"`
	RecurringIndicator         *bool                       `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                      `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string           `json:"attributes"`
	Purposes                   []ConsentPurposeRefRequest  `json:"purposes"`
	Authorizations             []AuthorizationRequest      `json:"authorizations"`
}

// ConsentRevokeRequest is the body for POST /consents/{consentId}/revoke.
type ConsentRevokeRequest struct {
	ActionBy         string `json:"actionBy"`
	RevocationReason string `json:"revocationReason,omitempty"`
}

// ResourceParams holds optional resource context for consent validation.
type ResourceParams struct {
	Resource   string `json:"resource,omitempty"`
	HTTPMethod string `json:"httpMethod,omitempty"`
	Context    string `json:"context,omitempty"`
}

// ConsentValidateRequest is the body for POST /consents/validate.
type ConsentValidateRequest struct {
	ConsentID       string                 `json:"consentId"`
	GroupID         string                 `json:"groupId,omitempty"`
	UserID          string                 `json:"userId,omitempty"`
	Headers         map[string]interface{} `json:"headers,omitempty"`
	Payload         map[string]interface{} `json:"payload,omitempty"`
	ElectedResource string                 `json:"electedResource,omitempty"`
	ResourceParams  *ResourceParams        `json:"resourceParams,omitempty"`
}

// =============================================================================
// API response types — HTTP boundary, handler only, json tags, no db tags
// =============================================================================

// ConsentPurposeElementApprovalResponse is one element in a consent purpose response.
// Combines the element's definition with the user's approval state for this consent.
type ConsentPurposeElementApprovalResponse struct {
	ElementID string      `json:"elementId"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Version   string      `json:"version"` // "v1", "v2", ...
	Mandatory bool        `json:"mandatory"`
	Approved  bool        `json:"approved"`
	Value     interface{} `json:"value,omitempty"`
}

// ConsentPurposeResponse is one purpose in a consent response (create/get/update).
type ConsentPurposeResponse struct {
	PurposeID string                                  `json:"purposeId"`
	Name      string                                  `json:"name"`
	Version   string                                  `json:"version"` // "v1", "v2", ...
	Elements  []ConsentPurposeElementApprovalResponse `json:"elements"`
}

// AuthorizationResponse is one authorization in a consent response.
type AuthorizationResponse struct {
	ID          string      `json:"id"`
	UserID      *string     `json:"userId,omitempty"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources,omitempty"`
}

// ConsentResponse is the response body for POST, GET, and PUT /consents.
type ConsentResponse struct {
	ConsentID                  string                   `json:"id"`
	GroupID                    string                   `json:"groupId"`
	Type                       string                   `json:"type"`
	Status                     string                   `json:"status"`
	CreatedTime                int64                    `json:"createdTime"`
	UpdatedTime                int64                    `json:"updatedTime"`
	ExpirationTime             *int64                   `json:"expirationTime,omitempty"`
	Frequency                  *int                     `json:"frequency,omitempty"`
	RecurringIndicator         *bool                    `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                   `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string        `json:"attributes"`
	Purposes                   []ConsentPurposeResponse `json:"purposes"`
	Authorizations             []AuthorizationResponse  `json:"authorizations"`
}

// ConsentListMetadata holds pagination metadata for the list response.
type ConsentListMetadata struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
	Limit  int `json:"limit"`
}

// ConsentListResponse is the response body for GET /consents.
type ConsentListResponse struct {
	Data     []ConsentResponse   `json:"data"`
	Metadata ConsentListMetadata `json:"metadata"`
}

// ConsentRevokeResponse is the response body for POST /consents/{consentId}/revoke.
type ConsentRevokeResponse struct {
	ActionTime       int64  `json:"actionTime"`
	ActionBy         string `json:"actionBy"`
	RevocationReason string `json:"revocationReason,omitempty"`
}

// ConsentAttributeSearchResponse is the response body for GET /consents/attributes.
type ConsentAttributeSearchResponse struct {
	ConsentIDs []string `json:"consentIds"`
	Count      int      `json:"count"`
}

// =============================================================================
// Validate-specific API response types
//
// The validate endpoint returns enriched element details (type, description,
// properties from the element definition) that regular responses omit.
// =============================================================================

// ConsentValidatePurposeElementResponse is one element in a validate consent response.
// Extends ConsentPurposeElementApprovalResponse with enriched definition fields.
type ConsentValidatePurposeElementResponse struct {
	ElementID   string            `json:"elementId"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Version     string            `json:"version"`
	Mandatory   bool              `json:"mandatory"`
	Approved    bool              `json:"approved"`
	Value       interface{}       `json:"value,omitempty"`
	DisplayName *string           `json:"displayName,omitempty"`
	Type        string            `json:"type,omitempty"`        // element type: "basic", "json", "xml"
	Description *string           `json:"description,omitempty"` // element description from definition
	Properties  map[string]string `json:"properties,omitempty"`  // element properties from definition
}

// ConsentValidatePurposeResponse is one purpose in a validate consent response.
// Extends ConsentPurposeResponse with enriched definition fields.
type ConsentValidatePurposeResponse struct {
	PurposeID   string                                  `json:"purposeId"`
	Name        string                                  `json:"name"`
	Version     string                                  `json:"version"`
	DisplayName *string                                 `json:"displayName,omitempty"`
	Description *string                                 `json:"description,omitempty"`
	Properties  map[string]string                       `json:"properties,omitempty"`
	Elements    []ConsentValidatePurposeElementResponse `json:"elements"`
}

// ConsentValidateInfo is the consent information returned inside the validate response.
// Identical top-level fields to ConsentResponse but uses the enriched purpose/element types.
type ConsentValidateInfo struct {
	ConsentID                  string                           `json:"id"`
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

// ConsentValidateResponse is the response body for POST /consents/validate.
// ModifiedPayload is removed — the validate endpoint no longer mutates responses.
type ConsentValidateResponse struct {
	IsValid          bool                 `json:"isValid"`
	ErrorCode        int                  `json:"errorCode,omitempty"`
	ErrorMessage     string               `json:"errorMessage,omitempty"`
	ErrorDescription string               `json:"errorDescription,omitempty"`
	ConsentInfo      *ConsentValidateInfo `json:"consentInformation,omitempty"`
}
