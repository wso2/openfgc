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

package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	authmodel "github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/system/config"
)

// ─── Delegation support ───────────────────────────────────────────────────────
// All delegation data is stored in the existing CONSENT_ATTRIBUTE table as

// Attribute key constants for delegation metadata stored in CONSENT_ATTRIBUTE.
const (
	// AttrDelegationType is the kind of delegation.
	// Examples: "parental_biological", "parental_legal", "guardian", "carer", "power_of_attorney"
	AttrDelegationType = "delegation.type"

	// AttrDelegationPrincipalID is the userId of the actual data subject
	// (the person WHOSE data is being processed, not the person giving consent).
	AttrDelegationPrincipalID = "delegation.principal_id"

	// AttrGuardianValidUntil is the Unix timestamp (seconds) after which the
	// delegation expires. For minors, set this to the child's 18th birthday.
	// After this point only the principal may act on the consent.
	AttrGuardianValidUntil = "guardian.valid_until"

	// AttrGuardianRevocationPolicy controls who may revoke a delegated consent.
	// Use RevocationPolicyAny or RevocationPolicySubjectOnly below.
	AttrGuardianRevocationPolicy = "guardian.revocation_policy"
)

// RevocationPolicy controls who may revoke a delegated consent.
type RevocationPolicy string

const (
	// RevocationPolicyAny allows any registered delegate with canRevoke=true to revoke.
	RevocationPolicyAny RevocationPolicy = "ANY"
	// RevocationPolicySubjectOnly allows only the data principal to revoke.
	RevocationPolicySubjectOnly RevocationPolicy = "SUBJECT_ONLY"
)

// DelegationConfig is a parsed view of the delegation CONSENT_ATTRIBUTE rows.
// Built at runtime from the attribute map — not stored as a separate struct.
type DelegationConfig struct {
	Type             string
	PrincipalID      string
	ValidUntil       int64
	RevocationPolicy RevocationPolicy
}

// IsGuardianConsent returns true when this consent was given on behalf of someone else.
func (d DelegationConfig) IsGuardianConsent() bool {
	return d.Type != ""
}

// IsExpired returns true when the delegation period has ended.
// When true, only the data principal may act on the consent.
func (d DelegationConfig) IsExpired() bool {
	if d.ValidUntil == 0 {
		return false
	}
	return time.Now().Unix() > d.ValidUntil
}

// ─── end delegation support ───────────────────────────────────────────────────

// Consent represents the CONSENT table
type Consent struct {
	ConsentID                  string `db:"CONSENT_ID" json:"consentId"`
	CreatedTime                int64  `db:"CREATED_TIME" json:"createdTime"`
	UpdatedTime                int64  `db:"UPDATED_TIME" json:"updatedTime"`
	ClientID                   string `db:"CLIENT_ID" json:"clientId"`
	ConsentType                string `db:"CONSENT_TYPE" json:"consentType"`
	CurrentStatus              string `db:"CURRENT_STATUS" json:"currentStatus"`
	ConsentFrequency           *int   `db:"CONSENT_FREQUENCY" json:"consentFrequency,omitempty"`
	ValidityTime               *int64 `db:"VALIDITY_TIME" json:"validityTime,omitempty"`
	RecurringIndicator         *bool  `db:"RECURRING_INDICATOR" json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64 `db:"DATA_ACCESS_VALIDITY_DURATION" json:"dataAccessValidityDuration,omitempty"`
	OrgID                      string `db:"ORG_ID" json:"orgId"`
}

// JSON type for handling JSON fields in MySQL
type JSON json.RawMessage

// Scan implements the sql.Scanner interface for JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("unsupported type for JSON: %T", value)
	}

	// Validate that it's valid JSON by attempting to unmarshal and remarshal
	var temp interface{}
	if err := json.Unmarshal(bytes, &temp); err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}

	// Remarshal to ensure clean JSON
	cleanBytes, err := json.Marshal(temp)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	*j = JSON(cleanBytes)
	return nil
}

// Value implements the driver.Valuer interface for JSON
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

// MarshalJSON implements json.Marshaler
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON implements json.Unmarshaler
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return nil
	}
	*j = JSON(data)
	return nil
}

// ConsentElementItem represents a single consent element with name, value, and selection status
type ConsentElementItem struct {
	Name           string                 `json:"name"`
	Value          interface{}            `json:"value,omitempty"`          // Can be string, object, or array - omitted when nil
	IsUserApproved *bool                  `json:"isUserApproved,omitempty"` // Optional: defaults to false if not provided
	IsMandatory    *bool                  `json:"isMandatory,omitempty"`    // Optional: defaults to true if not provided
	Type           *string                `json:"type,omitempty"`           // Enriched from element definition (optional)
	Description    *string                `json:"description,omitempty"`    // Enriched from element definition (optional)
	Properties     map[string]interface{} `json:"properties,omitempty"`     // Enriched from element definition (optional)
}

// ConsentPurposeItem represents a purpose in consent API
type ConsentPurposeItem struct {
	PurposeName string                       `json:"name" binding:"required"`
	Elements    []ConsentElementApprovalItem `json:"elements" binding:"required,min=1"`
}

// ConsentElementApprovalItem represents an element approval within a purpose (for POST, GET, PUT, Search)
type ConsentElementApprovalItem struct {
	ElementName    string      `json:"name" binding:"required"`
	IsUserApproved bool        `json:"isUserApproved"`
	Value          interface{} `json:"value,omitempty"`
	// IsMandatory is tracked internally but excluded from JSON in regular responses
	IsMandatory bool `json:"-"`
}

// ConsentElementApprovalItemValidate represents a purpose approval with enriched details (for Validate endpoint)
type ConsentElementApprovalItemValidate struct {
	ElementName    string                 `json:"name" binding:"required"`
	IsUserApproved bool                   `json:"isUserApproved"`
	Value          interface{}            `json:"value,omitempty"`
	IsMandatory    bool                   `json:"isMandatory"`
	Type           string                 `json:"type,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
}

// ConsentPurposeItemValidate represents a purpose with enriched details (for Validate endpoint)
type ConsentPurposeItemValidate struct {
	PurposeName string                               `json:"name" binding:"required"`
	Elements    []ConsentElementApprovalItemValidate `json:"elements" binding:"required,min=1"`
}

// ConsentPurposeCreateRequest - internal format for purpose processing
type ConsentPurposeCreateRequest struct {
	PurposeName string
	PurposeID   string
	Elements    []ConsentElementApprovalCreateRequest
}

// ConsentElementApprovalCreateRequest - internal format for element approval
type ConsentElementApprovalCreateRequest struct {
	ElementID      string
	ElementName    string
	IsUserApproved bool
	Value          *string // JSON string
	IsMandatory    bool    // from purpose definition
}

// ConsentElementApprovalRecord represents the DB record for element approvals
type ConsentElementApprovalRecord struct {
	ConsentID      string
	PurposeID      string
	PurposeName    string
	ElementID      string
	ElementName    string
	IsUserApproved bool
	IsMandatory    bool
	Value          *string // JSON string
	OrgID          string
}

// ConsentPurposeMapping represents the mapping between consent and purposes
// from PURPOSE_CONSENT_MAPPING table
type ConsentPurposeMapping struct {
	ConsentID   string
	PurposeID   string
	PurposeName string
}

// ConsentAPIRequest represents the API payload for creating a consent (external format)
// Note: Status is not included in the request - it will be derived from authorization states
type ConsentAPIRequest struct {
	Type                       string                    `json:"type" binding:"required"`
	ValidityTime               *int64                    `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                     `json:"recurringIndicator,omitempty"`
	Frequency                  *int                      `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                    `json:"dataAccessValidityDuration,omitempty"`
	Purposes                   []ConsentPurposeItem      `json:"purposes" binding:"required,min=1"`
	Attributes                 map[string]string         `json:"attributes,omitempty"`
	Authorizations             []AuthorizationAPIRequest `json:"authorizations"` // Remove omitempty to allow explicit empty array in updates
	// CallerID is set by the handler from the X-User-ID request header.
	// It is NOT read from the JSON body (json:"-").
	// Used by delegation validation to ensure the consent initiator is not
	// also the data principal — cannot self-delegate
	CallerID string `json:"-"`
	// PrincipalID is the userId of the data subject when this consent is being
	// given on behalf of another person (e.g., parent creating consent for child).
	// It mirrors the delegation.principal_id attribute stored in CONSENT_ATTRIBUTE.
	// Provided as a convenience field on the request for validation.
	PrincipalID string `json:"principalId,omitempty"`
}

// ─── Delegation response types ────────────────────────────────────────────────
// These types support the GET /consents/{consentId}/delegates endpoint and
// the delegate-related test assertions.

// DelegationAPIRequest carries optional delegation metadata on an authorization
// resource at consent creation time. It is used when the consent is being given
// on behalf of another person (e.g., parent for child). Stored in the RESOURCES
// JSON blob of CONSENT_AUTH_RESOURCE — no schema change needed.
type DelegationAPIRequest struct {
	// Type is the delegation relationship, e.g. "parental_biological", "carer".
	Type string `json:"type,omitempty"`
	// PrincipalID is the userId of the actual data subject (the person whose data
	// is being processed). Must match delegation.principal_id in CONSENT_ATTRIBUTE.
	PrincipalID string `json:"principalId,omitempty"`
	// CanRevoke indicates whether this delegate may revoke the consent.
	CanRevoke bool `json:"canRevoke,omitempty"`
	// CanModify indicates whether this delegate may modify the consent.
	CanModify bool `json:"canModify,omitempty"`
}

// mergeDelegationIntoResources merges the Delegation fields into the Resources map
// so they are persisted in the RESOURCES JSON blob of CONSENT_AUTH_RESOURCE.
// When Delegation is nil or Resources already contains onBehalfOf, the input is
// returned unchanged. This is the single place where delegation metadata is
// serialized — no DB schema change required.
func mergeDelegationIntoResources(resources interface{}, delegation *DelegationAPIRequest) interface{} {
	if delegation == nil {
		return resources
	}

	// Start with an empty map or a copy of the existing resources map.
	merged := make(map[string]interface{})
	if resources != nil {
		// resources may already be a map (from JSON decode) or a raw JSON []byte.
		switch v := resources.(type) {
		case map[string]interface{}:
			for k, val := range v {
				merged[k] = val
			}
		default:
			// Fallback: round-trip through JSON to get a generic map.
			b, err := json.Marshal(resources)
			if err == nil {
				_ = json.Unmarshal(b, &merged)
			}
		}
	}

	// Write delegation fields into the map.
	// onBehalfOf is the key the service layer reads to identify delegate rows.
	if delegation.PrincipalID != "" {
		merged["onBehalfOf"] = delegation.PrincipalID
	}
	if delegation.Type != "" {
		merged["delegationType"] = delegation.Type
	}
	// Always write bool flags so false is explicit (not missing).
	merged["canRevoke"] = delegation.CanRevoke
	merged["canModify"] = delegation.CanModify

	return merged
}

// DelegateInfo represents a single registered delegate on a consent.
// Assembled at read time from a CONSENT_AUTH_RESOURCE row and its RESOURCES blob.
type DelegateInfo struct {
	// AuthID is the CONSENT_AUTH_RESOURCE.AUTH_ID for this delegate row.
	AuthID string `json:"authId"`
	// UserID is the delegate's userId (e.g., the parent's account ID).
	UserID string `json:"userId"`
	// DelegationType mirrors the delegation.type value (e.g., "parental_biological").
	DelegationType string `json:"delegationType,omitempty"`
	// Status is the CONSENT_AUTH_RESOURCE.AUTH_STATUS (e.g., "approved").
	Status string `json:"status"`
	// CanRevoke is true when this delegate may revoke the consent.
	CanRevoke bool `json:"canRevoke"`
	// CanModify is true when this delegate may modify the consent.
	CanModify bool `json:"canModify"`
	// OnBehalfOf is the principalID this delegate is acting for.
	OnBehalfOf string `json:"onBehalfOf,omitempty"`
	// UpdatedTime is the CONSENT_AUTH_RESOURCE.UPDATED_TIME in milliseconds.
	UpdatedTime int64 `json:"updatedTime"`
}

// DelegateListResponse is the response body for GET /consents/{consentId}/delegates.
// It aggregates all delegate auth resources for a single consent.
type DelegateListResponse struct {
	// ConsentID is the consent this response belongs to.
	ConsentID string `json:"consentId"`
	// DataPrincipalID is the userId of the data subject (from delegation.principal_id).
	DataPrincipalID string `json:"dataPrincipalId,omitempty"`
	// RevocationPolicy mirrors guardian.revocation_policy (e.g., "ANY").
	RevocationPolicy string `json:"revocationPolicy,omitempty"`
	// ValidUntil is the delegation expiry Unix timestamp (from guardian.valid_until).
	ValidUntil int64 `json:"validUntil,omitempty"`
	// IsDelegationExpired is true when the current time is past ValidUntil.
	IsDelegationExpired bool `json:"isDelegationExpired"`
	// DelegateCount is the total number of delegates registered on this consent.
	DelegateCount int `json:"delegateCount"`
	// Delegates is the list of individual delegate records.
	Delegates []DelegateInfo `json:"delegates"`
}

// ─── end delegation response types ───────────────────────────────────────────

// AuthorizationAPIRequest represents the API payload for authorization resource (external format)
// Status field represents the authorization status/state (created, approved, rejected, or custom)
type AuthorizationAPIRequest struct {
	UserID    string      `json:"userId,omitempty"`
	Type      string      `json:"type" binding:"required"`
	Status    string      `json:"status,omitempty"` // Optional: defaults to "approved" if not provided
	Resources interface{} `json:"resources,omitempty"`
	// Delegation carries optional delegation metadata when this authorization represents
	// a delegate acting on behalf of a data principal. Stored inside the RESOURCES JSON blob.
	// Not a separate DB column — no schema change required.
	Delegation *DelegationAPIRequest `json:"delegation,omitempty"`
}

// ToAuthResourceCreateRequest converts API request format to internal format
func (req *AuthorizationAPIRequest) ToAuthResourceCreateRequest() *authmodel.ConsentAuthResourceCreateRequest {

	AuthStatusMappings := config.Get().Consent.AuthStatusMappings
	var userID *string
	if req.UserID != "" {
		userID = &req.UserID
	}

	// Default status to "created" if not provided
	// Note: This defaults to CreatedState (unlike consent-embedded authorizations which default to ApprovedState)
	// because direct authorization creation typically requires explicit approval workflow
	status := req.Status
	if status == "" {
		status = string(AuthStatusMappings.CreatedState)
	}

	return &authmodel.ConsentAuthResourceCreateRequest{
		AuthType:   req.Type,
		UserID:     userID,
		AuthStatus: status, // Store the status value in AuthStatus field
		Resources:  req.Resources,
	}
}

// AuthorizationAPIUpdateRequest represents the API payload for updating authorization resource (external format)
type AuthorizationAPIUpdateRequest struct {
	UserID    string      `json:"userId,omitempty"`
	Type      string      `json:"type,omitempty"`
	Status    string      `json:"status,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// ToAuthResourceUpdateRequest converts API update request format to internal format
func (req *AuthorizationAPIUpdateRequest) ToAuthResourceUpdateRequest() *authmodel.ConsentAuthResourceUpdateRequest {
	var userID *string
	if req.UserID != "" {
		userID = &req.UserID
	}

	return &authmodel.ConsentAuthResourceUpdateRequest{
		AuthStatus: req.Status,
		UserID:     userID,
		Resources:  req.Resources,
	}
}

// ConsentAPIUpdateRequest represents the API payload for updating a consent (external format)
// Note: Status is not included in the request - it will be derived from authorization states
// Note: Purposes, Attributes, and Authorizations don't have omitempty to allow empty arrays/maps for removal
type ConsentAPIUpdateRequest struct {
	Type                       string                    `json:"type,omitempty"`
	ValidityTime               *int64                    `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                     `json:"recurringIndicator,omitempty"`
	Frequency                  *int                      `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                    `json:"dataAccessValidityDuration,omitempty"`
	Purposes                   []ConsentPurposeItem      `json:"purposes"`
	Attributes                 map[string]string         `json:"attributes"`
	Authorizations             []AuthorizationAPIRequest `json:"authorizations"`
}

// ConsentCreateRequest represents the internal request payload for creating a consent
type ConsentCreateRequest struct {
	Purposes                   []ConsentPurposeCreateRequest                `json:"purposes" binding:"required,min=1"`
	ConsentType                string                                       `json:"consentType" binding:"required"`
	CurrentStatus              string                                       `json:"currentStatus" binding:"required"`
	ConsentFrequency           *int                                         `json:"consentFrequency,omitempty"`
	ValidityTime               *int64                                       `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                                        `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                                       `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string                            `json:"attributes,omitempty"`
	AuthResources              []authmodel.ConsentAuthResourceCreateRequest `json:"authResources,omitempty"`
}

// ConsentUpdateRequest represents the request payload for updating a consent
type ConsentUpdateRequest struct {
	Purposes                   []ConsentPurposeCreateRequest                `json:"purposes,omitempty"`
	ConsentType                string                                       `json:"consentType,omitempty"`
	CurrentStatus              string                                       `json:"currentStatus,omitempty"`
	ConsentFrequency           *int                                         `json:"consentFrequency,omitempty"`
	ValidityTime               *int64                                       `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                                        `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                                       `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string                            `json:"attributes,omitempty"`
	AuthResources              []authmodel.ConsentAuthResourceCreateRequest `json:"authResources,omitempty"`
}

// ConsentResponse represents the response after consent creation/retrieval
type ConsentResponse struct {
	ConsentID                  string                          `json:"consentId"`
	Purposes                   []ConsentPurposeItem            `json:"purposes"`
	CreatedTime                int64                           `json:"createdTime"`
	UpdatedTime                int64                           `json:"updatedTime"`
	ClientID                   string                          `json:"clientId"`
	ConsentType                string                          `json:"consentType"`
	CurrentStatus              string                          `json:"currentStatus"`
	ConsentFrequency           *int                            `json:"consentFrequency,omitempty"`
	ValidityTime               *int64                          `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                           `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                          `json:"dataAccessValidityDuration,omitempty"`
	OrgID                      string                          `json:"orgId"`
	Attributes                 map[string]string               `json:"attributes,omitempty"`
	AuthResources              []authmodel.ConsentAuthResource `json:"authResources,omitempty"`
}

// ConsentSearchParams represents search parameters for consent queries
type ConsentSearchParams struct {
	ConsentIDs      []string `form:"consentIds"`
	ClientIDs       []string `form:"clientIds"`
	ConsentTypes    []string `form:"consentTypes"`
	ConsentStatuses []string `form:"consentStatuses"`
	UserIDs         []string `form:"userIds"`
	FromTime        *int64   `form:"fromTime"`
	ToTime          *int64   `form:"toTime"`
	Limit           int      `form:"limit"`
	Offset          int      `form:"offset"`
	OrgID           string   `form:"-"` // Extracted from header
}

// ConsentSearchResponse represents the response for consent search
type ConsentSearchResponse struct {
	Data     []ConsentResponse     `json:"data"`
	Metadata ConsentSearchMetadata `json:"metadata"`
}

// ConsentSearchMetadata represents pagination metadata
type ConsentSearchMetadata struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Count  int `json:"count"` // Number of results in current page
}

// ConsentSearchFilters represents search criteria for consents
type ConsentSearchFilters struct {
	ConsentTypes    []string // e.g., ["accounts", "payments"]
	ConsentStatuses []string // e.g., ["active", "revoked"]
	ClientIDs       []string // TPP client IDs
	UserIDs         []string // End-user IDs
	PurposeNames    []string // Purpose names - returns consents containing ANY of these purposes
	FromTime        *int64   // Unix timestamp - start of time window
	ToTime          *int64   // Unix timestamp - end of time window
	Limit           int
	Offset          int
	OrgID           string
	// DataPrincipalID filters consents by the data subject stored in
	// CONSENT_ATTRIBUTE (key = delegation.principal_id).
	// Use this when a parent/guardian wants to see consents for their child.
	// The service will verify CallerID is an authorised delegate
	DataPrincipalID string
	// CallerID is the authenticated user making the list request.
	// Required when DataPrincipalID is set so the service can verify
	// the caller is an authorised delegate before returning results
	CallerID string
}

// ConsentDetailResponse represents a detailed consent with related data
type ConsentDetailResponse struct {
	ID                         string                `json:"id"`
	Purposes                   []ConsentPurposeItem  `json:"purposes"`
	CreatedTime                int64                 `json:"createdTime"`
	UpdatedTime                int64                 `json:"updatedTime"`
	ClientID                   string                `json:"clientId"`
	Type                       string                `json:"type"`
	Status                     string                `json:"status"`
	Frequency                  int                   `json:"frequency"`
	ValidityTime               int64                 `json:"validityTime"`
	RecurringIndicator         bool                  `json:"recurringIndicator"`
	DataAccessValidityDuration int64                 `json:"dataAccessValidityDuration"`
	Attributes                 map[string]string     `json:"attributes"`
	Authorizations             []AuthorizationDetail `json:"authorizations"`
	// DelegationExpired is true when guardian.valid_until has passed.
	// The principal is now an adult or has regained capacity. The portal should
	// prompt them to review and re-confirm or revoke inherited consents.
	// PendingAdultReview was removed — it was identical to DelegationExpired and
	// added no distinct information. One clear flag is better than two identical ones.
	DelegationExpired bool `json:"delegationExpired,omitempty"`
}

// AuthorizationDetail represents authorization resource details
type AuthorizationDetail struct {
	ID          string      `json:"id"`
	UserID      string      `json:"userId"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources,omitempty"`
}

// ConsentDetailSearchResponse wraps detailed consent search results
type ConsentDetailSearchResponse struct {
	Data     []ConsentDetailResponse `json:"data"`
	Metadata ConsentSearchMetadata   `json:"metadata"`
}

// ConsentRevokeRequest represents the request to revoke a consent
type ConsentRevokeRequest struct {
	ActionBy         string `json:"actionBy" binding:"required"`
	RevocationReason string `json:"revocationReason,omitempty"`
}

// GetCreatedTime returns the created time as a time.Time
func (c *Consent) GetCreatedTime() time.Time {
	return time.Unix(0, c.CreatedTime*int64(time.Millisecond))
}

// GetUpdatedTime returns the updated time as a time.Time
func (c *Consent) GetUpdatedTime() time.Time {
	return time.Unix(0, c.UpdatedTime*int64(time.Millisecond))
}

// ToConsentCreateRequest converts API request format to internal format
// Note: CurrentStatus will be set by the handler based on authorization states
func (req *ConsentAPIRequest) ToConsentCreateRequest() (*ConsentCreateRequest, error) {

	AuthStatusMappings := config.Get().Consent.AuthStatusMappings

	createReq := &ConsentCreateRequest{
		ConsentType:                req.Type,
		CurrentStatus:              "", // Will be set by handler based on auth states
		Attributes:                 req.Attributes,
		ValidityTime:               req.ValidityTime,
		ConsentFrequency:           req.Frequency,
		RecurringIndicator:         req.RecurringIndicator,
		DataAccessValidityDuration: req.DataAccessValidityDuration,
	}

	// Structure purposes data (validation happens in service layer)
	purposes := make([]ConsentPurposeCreateRequest, len(req.Purposes))
	for i, purposeInput := range req.Purposes {
		elements := make([]ConsentElementApprovalCreateRequest, len(purposeInput.Elements))
		for j, elementInput := range purposeInput.Elements {
			var valueJSON *string
			if elementInput.Value != nil {
				valueBytes, err := json.Marshal(elementInput.Value)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal purpose value: %v", err)
				}
				valueStr := string(valueBytes)
				valueJSON = &valueStr
			}

			elements[j] = ConsentElementApprovalCreateRequest{
				ElementName:    elementInput.ElementName,
				IsUserApproved: elementInput.IsUserApproved,
				Value:          valueJSON,
			}
		}

		purposes[i] = ConsentPurposeCreateRequest{
			PurposeName: purposeInput.PurposeName,
			Elements:    elements,
		}
	}
	createReq.Purposes = purposes

	// Map authorizations to auth resources
	if len(req.Authorizations) > 0 {
		createReq.AuthResources = make([]authmodel.ConsentAuthResourceCreateRequest, len(req.Authorizations))
		for i, auth := range req.Authorizations {
			var userID *string
			if auth.UserID != "" {
				userID = &auth.UserID
			}

			// Default status to "approved" if not provided
			// Note: Consent-embedded authorizations default to ApprovedState (unlike direct auth resource
			// creation which defaults to CreatedState) because they're created as part of consent flow
			status := auth.Status
			if status == "" {
				status = string(AuthStatusMappings.ApprovedState)
			}

			// persisting, so onBehalfOf/canRevoke/canModify/delegationType are
			// stored in the RESOURCES blob and readable by the delegates endpoint.
			resources := mergeDelegationIntoResources(auth.Resources, auth.Delegation)
			createReq.AuthResources[i] = authmodel.ConsentAuthResourceCreateRequest{
				AuthType:   auth.Type,
				UserID:     userID,
				AuthStatus: status,
				Resources:  resources,
			}
		}
	}

	return createReq, nil
}

// ToConsentUpdateRequest converts API update request format to internal format
// Note: CurrentStatus will be set by the handler based on authorization states
func (req *ConsentAPIUpdateRequest) ToConsentUpdateRequest() (*ConsentUpdateRequest, error) {

	AuthStatusMappings := config.Get().Consent.AuthStatusMappings

	// Convert purposes from API format to internal format
	var purposes []ConsentPurposeCreateRequest
	if req.Purposes != nil {
		purposes = make([]ConsentPurposeCreateRequest, len(req.Purposes))
		for i, purposeInput := range req.Purposes {
			// Convert elements within the purpose
			elements := make([]ConsentElementApprovalCreateRequest, len(purposeInput.Elements))
			for j, elementInput := range purposeInput.Elements {
				// Marshal value to JSON if present
				var valueJSON *string
				if elementInput.Value != nil {
					valueBytes, err := json.Marshal(elementInput.Value)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal purpose value for '%s': %w", elementInput.ElementName, err)
					}
					valueStr := string(valueBytes)
					valueJSON = &valueStr
				}

				elements[j] = ConsentElementApprovalCreateRequest{
					ElementName:    elementInput.ElementName,
					IsUserApproved: elementInput.IsUserApproved,
					Value:          valueJSON,
					// PurposeID and IsMandatory will be resolved during validation
				}
			}

			purposes[i] = ConsentPurposeCreateRequest{
				PurposeName: purposeInput.PurposeName,
				Elements:    elements,
				// PurposeID will be resolved during validation
			}
		}
	}

	updateReq := &ConsentUpdateRequest{
		Purposes:                   purposes,
		ConsentType:                req.Type,
		CurrentStatus:              "", // Will be set by handler based on auth states
		Attributes:                 req.Attributes,
		ValidityTime:               req.ValidityTime,
		ConsentFrequency:           req.Frequency,
		RecurringIndicator:         req.RecurringIndicator,
		DataAccessValidityDuration: req.DataAccessValidityDuration,
	}

	// Map authorizations to auth resources
	// If Authorizations is not nil (even if empty), set AuthResources to indicate intent to update
	if req.Authorizations != nil {
		updateReq.AuthResources = make([]authmodel.ConsentAuthResourceCreateRequest, len(req.Authorizations))
		for i, auth := range req.Authorizations {
			var userID *string
			if auth.UserID != "" {
				userID = &auth.UserID
			}

			// Default status to "approved" if not provided
			// Note: Consent-embedded authorizations default to ApprovedState (unlike direct auth resource
			// creation which defaults to CreatedState) because they're created as part of consent flow
			status := auth.Status
			if status == "" {
				status = string(AuthStatusMappings.ApprovedState)
			}

			// fields must be written into the RESOURCES blob on update too.
			resources := mergeDelegationIntoResources(auth.Resources, auth.Delegation)
			updateReq.AuthResources[i] = authmodel.ConsentAuthResourceCreateRequest{
				AuthType:   auth.Type,
				UserID:     userID,
				AuthStatus: status, // Store the status value
				Resources:  resources,
			}
		}
	}

	return updateReq, nil
}

// ConsentAPIResponse represents the API response format for consent (external format)
type ConsentAPIResponse struct {
	ID                         string                     `json:"id"`
	Purposes                   []ConsentPurposeItem       `json:"purposes"`
	CreatedTime                int64                      `json:"createdTime"`
	UpdatedTime                int64                      `json:"updatedTime"`
	ClientID                   string                     `json:"clientId"`
	Type                       string                     `json:"type"`
	Status                     string                     `json:"status"`
	Frequency                  *int                       `json:"frequency,omitempty"`
	ValidityTime               *int64                     `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                      `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                     `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string          `json:"attributes"`
	Authorizations             []AuthorizationAPIResponse `json:"authorizations"`
	ModifiedResponse           interface{}                `json:"modifiedResponse,omitempty"` // Present in GET/POST/PUT, excluded in validate
}

// AuthorizationAPIResponse represents the API response format for authorization resource (external format)
type AuthorizationAPIResponse struct {
	ID          string      `json:"id"`
	UserID      *string     `json:"userId,omitempty"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources"`
}

// ToAPIResponse converts internal response format to API response format
func (resp *ConsentResponse) ToAPIResponse() *ConsentAPIResponse {
	// Initialize Attributes with empty object if nil
	attributes := resp.Attributes
	if attributes == nil {
		attributes = make(map[string]string)
	}

	// Initialize Purposes with empty array if nil
	purposes := resp.Purposes
	if purposes == nil {
		purposes = make([]ConsentPurposeItem, 0)
	}

	apiResp := &ConsentAPIResponse{
		ID:                         resp.ConsentID,
		Purposes:                   purposes,
		CreatedTime:                resp.CreatedTime,
		UpdatedTime:                resp.UpdatedTime,
		ClientID:                   resp.ClientID,
		Type:                       resp.ConsentType,
		Status:                     resp.CurrentStatus,
		Frequency:                  resp.ConsentFrequency,
		ValidityTime:               resp.ValidityTime,
		RecurringIndicator:         resp.RecurringIndicator,
		DataAccessValidityDuration: resp.DataAccessValidityDuration,
		Attributes:                 attributes,
		ModifiedResponse:           make(map[string]interface{}),
		Authorizations:             make([]AuthorizationAPIResponse, 0),
	}

	// Map auth resources to authorizations
	if len(resp.AuthResources) > 0 {
		apiResp.Authorizations = make([]AuthorizationAPIResponse, len(resp.AuthResources))
		for i, auth := range resp.AuthResources {
			// Parse resources JSON string to interface
			var resources interface{}
			if auth.Resources != nil && *auth.Resources != "" {
				if err := json.Unmarshal([]byte(*auth.Resources), &resources); err != nil {
					// If parsing fails, set to empty object
					resources = make(map[string]interface{})
				}
			} else {
				// If resources is nil or empty, set to empty object
				resources = make(map[string]interface{})
			}

			apiResp.Authorizations[i] = AuthorizationAPIResponse{
				ID:          auth.AuthID,
				UserID:      auth.UserID,
				Type:        auth.AuthType,
				Status:      auth.AuthStatus,
				UpdatedTime: auth.UpdatedTime,
				Resources:   resources,
			}
		}
	}

	return apiResp
}

// ValidateRequest represents the payload for validation API
type ValidateRequest struct {
	Headers         map[string]interface{} `json:"headers"`
	Payload         map[string]interface{} `json:"payload"`
	ElectedResource string                 `json:"electedResource"`
	ConsentID       string                 `json:"consentId"`
	UserID          string                 `json:"userId"`
	ClientID        string                 `json:"clientId"`
	ResourceParams  struct {
		Resource   string `json:"resource"`
		HTTPMethod string `json:"httpMethod"`
		Context    string `json:"context"`
	} `json:"resourceParams"`
}

// ValidateResponse represents the response for validation API
type ValidateResponse struct {
	IsValid            bool                        `json:"isValid"`
	ModifiedPayload    interface{}                 `json:"modifiedPayload,omitempty"`
	ErrorCode          int                         `json:"errorCode,omitempty"`
	ErrorMessage       string                      `json:"errorMessage,omitempty"`
	ErrorDescription   string                      `json:"errorDescription,omitempty"`
	ConsentInformation *ValidateConsentAPIResponse `json:"consentInformation,omitempty"`
}

// ValidateConsentAPIResponse represents consent information in validate response (excludes modifiedResponse)
type ValidateConsentAPIResponse struct {
	ID                         string                       `json:"id"`
	Type                       string                       `json:"type"`
	ClientID                   string                       `json:"clientId"`
	Status                     string                       `json:"status"`
	CreatedTime                int64                        `json:"createdTime"`
	UpdatedTime                int64                        `json:"updatedTime"`
	ValidityTime               *int64                       `json:"validityTime"`
	RecurringIndicator         *bool                        `json:"recurringIndicator"`
	Frequency                  *int                         `json:"frequency"`
	DataAccessValidityDuration *int64                       `json:"dataAccessValidityDuration"`
	Purposes                   []ConsentPurposeItemValidate `json:"purposes"`
	Attributes                 map[string]string            `json:"attributes,omitempty"`
	Authorizations             []AuthorizationAPIResponse   `json:"authorizations,omitempty"`
}

// ConsentRevokeResponse represents the response after revoking a consent
type ConsentRevokeResponse struct {
	ActionTime       int64  `json:"actionTime"`
	ActionBy         string `json:"actionBy"`
	RevocationReason string `json:"revocationReason,omitempty"`
}
