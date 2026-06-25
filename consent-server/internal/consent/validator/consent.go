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
	"fmt"
	"strings"
	"time"

	authmodel "github.com/wso2/openfgc/internal/authresource/model"
	authvalidator "github.com/wso2/openfgc/internal/authresource/validator"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/system/config"
)

// minExpirationTimestamp is the smallest raw value accepted for expirationTime.
// It equals 10^9 (September 9, 2001 in Unix seconds).
// Values below this threshold are not valid Unix timestamps in either the seconds
// or milliseconds format that the server accepts, so they are rejected outright.
const minExpirationTimestamp = int64(1_000_000_000)

// ValidateConsentCreateRequest validates a consent creation request.
// groupID is read from the group-id request header, not the body.
// Authorization type is optional — the service defaults it to "default" when absent.
func ValidateConsentCreateRequest(req model.ConsentCreateRequest, groupID, orgID string) error {
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if len(req.Type) > 64 {
		return fmt.Errorf("type must be at most 64 characters")
	}
	if groupID == "" {
		return fmt.Errorf("group-id header is required")
	}
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}

	if req.ExpirationTime != nil && *req.ExpirationTime < 0 {
		return fmt.Errorf("expirationTime must be non-negative")
	}
	if req.ExpirationTime != nil && *req.ExpirationTime > 0 && *req.ExpirationTime < minExpirationTimestamp {
		return fmt.Errorf("expirationTime is not a valid Unix timestamp; provide seconds (10 digits) or milliseconds (13 digits)")
	}
	if req.Frequency != nil && *req.Frequency < 0 {
		return fmt.Errorf("frequency must be non-negative")
	}

	for key, value := range req.Attributes {
		if len(key) > 255 {
			return fmt.Errorf("attribute key %q exceeds maximum length of 255 characters", key)
		}
		if len(value) > 1024 {
			return fmt.Errorf("attribute value for key %q exceeds maximum length of 1024 characters", key)
		}
	}

	for i, authReq := range req.Authorizations {
		if authReq.UserID == "" {
			return fmt.Errorf("authorizations[%d]: userId is required", i)
		}
		if authReq.Status != "" {
			cfg := config.Get()
			if cfg == nil {
				return fmt.Errorf("configuration not initialized")
			}
			if err := authvalidator.ValidateAuthStatus(authReq.Status, cfg.Consent.AuthStatusMappings); err != nil {
				return fmt.Errorf("authorizations[%d]: %w", i, err)
			}
		}
	}

	// Validate auth type constraints across the full authorization set
	if err := ValidateAuthTypeConstraints(req.Authorizations); err != nil {
		return err
	}

	return nil
}

// ValidateConsentUpdateRequest validates a consent update request.
func ValidateConsentUpdateRequest(req model.ConsentUpdateRequest) error {
	if req.Type == "" && req.Frequency == nil &&
		req.ExpirationTime == nil && req.RecurringIndicator == nil &&
		req.DataAccessValidityDuration == nil &&
		req.Attributes == nil && req.Authorizations == nil && req.Purposes == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}

	if req.Type != "" && len(req.Type) > 64 {
		return fmt.Errorf("type must be at most 64 characters")
	}
	if req.ExpirationTime != nil && *req.ExpirationTime < 0 {
		return fmt.Errorf("expirationTime must be non-negative")
	}
	if req.ExpirationTime != nil && *req.ExpirationTime > 0 && *req.ExpirationTime < minExpirationTimestamp {
		return fmt.Errorf("expirationTime is not a valid Unix timestamp; provide seconds (10 digits) or milliseconds (13 digits)")
	}
	if req.Frequency != nil && *req.Frequency < 0 {
		return fmt.Errorf("frequency must be non-negative")
	}

	for key, value := range req.Attributes {
		if len(key) > 255 {
			return fmt.Errorf("attribute key %q exceeds maximum length of 255 characters", key)
		}
		if len(value) > 1024 {
			return fmt.Errorf("attribute value for key %q exceeds maximum length of 1024 characters", key)
		}
	}

	for i, authReq := range req.Authorizations {
		if authReq.UserID == "" {
			return fmt.Errorf("authorizations[%d]: userId is required", i)
		}
		if authReq.Status != "" {
			cfg := config.Get()
			if cfg == nil {
				return fmt.Errorf("configuration not initialized")
			}
			if err := authvalidator.ValidateAuthStatus(authReq.Status, cfg.Consent.AuthStatusMappings); err != nil {
				return fmt.Errorf("authorizations[%d]: %w", i, err)
			}
		}
	}

	// Validate auth type constraints if authorizations are being replaced
	if req.Authorizations != nil {
		if err := ValidateAuthTypeConstraints(req.Authorizations); err != nil {
			return err
		}
	}

	return nil
}

// ValidateAuthTypeConstraints validates the auth type rules across a full set of authorizations.
//
// Rules for first-class types:
//   - "delegate" requires at least one "delegate_subject" in the same consent
//   - "delegate_subject" requires at least one "delegate" in the same consent
//   - "primary" cannot mix with "delegate" or "delegate_subject"
//
// Custom types (anything not primary/delegate/delegate_subject) skip these rules entirely.
//
// Universal rule (applies to all types):
//   - At least one authorization must have a non-RECORDED status.
//     This prevents consents where nobody actively consented.
func ValidateAuthTypeConstraints(authorizations []model.AuthorizationRequest) error {
	if len(authorizations) == 0 {
		return nil
	}

	hasPrimary := false
	hasDelegate := false
	hasDelegateSubject := false
	hasFirstClass := false

	cfg := config.Get()

	// Determine the recorded status string for the universal participation check
	var recordedStatus string
	if cfg != nil {
		recordedStatus = string(cfg.Consent.GetRecordedAuthStatus())
	}

	hasNonRecorded := false

	for _, auth := range authorizations {
		// Resolve the effective auth type (empty defaults to "default" which is treated like primary)
		authType := auth.Type
		if authType == "" {
			authType = authmodel.DefaultAuthType
		}

		switch authType {
		case authmodel.AuthTypePrimary:
			hasPrimary = true
			hasFirstClass = true
		case authmodel.AuthTypeDelegate:
			hasDelegate = true
			hasFirstClass = true
		case authmodel.AuthTypeDelegateSubject:
			hasDelegateSubject = true
			hasFirstClass = true
		case authmodel.DefaultAuthType:
			// "default" is treated as self-consent (like primary) for validation purposes
			hasPrimary = true
		}

		// Universal participation check: at least one auth must not be RECORDED
		effectiveStatus := auth.Status
		if effectiveStatus == "" {
			// Empty status defaults to APPROVED — that's a participating status
			hasNonRecorded = true
		} else if recordedStatus == "" || effectiveStatus != recordedStatus {
			hasNonRecorded = true
		}
	}

	// Universal rule: at least one authorization must actively participate
	if !hasNonRecorded {
		return fmt.Errorf("at least one authorization must have an active status; RECORDED alone does not constitute consent")
	}

	// First-class type pairing rules (only enforced when first-class types are present)
	if hasFirstClass {
		// primary cannot mix with delegate or delegate_subject
		if hasPrimary && (hasDelegate || hasDelegateSubject) {
			return fmt.Errorf("authorization type 'primary' cannot be mixed with 'delegate' or 'delegate_subject' in the same consent")
		}

		// delegate requires at least one delegate_subject
		if hasDelegate && !hasDelegateSubject {
			return fmt.Errorf("authorization type 'delegate' requires at least one 'delegate_subject' in the same consent")
		}

		// delegate_subject requires at least one delegate
		if hasDelegateSubject && !hasDelegate {
			return fmt.Errorf("authorization type 'delegate_subject' requires at least one 'delegate' in the same consent")
		}
	}

	return nil
}

// ValidateConsentGetRequest validates consent retrieval request parameters.
func ValidateConsentGetRequest(consentID, orgID string) error {
	if consentID == "" {
		return fmt.Errorf("consent ID cannot be empty")
	}
	if len(consentID) > 255 {
		return fmt.Errorf("consent ID too long (max 255 characters)")
	}
	if orgID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if len(orgID) > 255 {
		return fmt.Errorf("organization ID too long (max 255 characters)")
	}
	return nil
}

// EvaluateConsentStatusFromAuthStatuses determines consent status from a list of auth status strings.
//
// The derivation follows two steps:
//
//  1. Filter out non-participating statuses: RECORDED, SYS_EXPIRED, and SYS_REVOKED are
//     excluded because they represent passive participants or system-managed states that
//     should not influence the consent's overall status.
//
//  2. From the remaining participating statuses, apply priority logic:
//     Any REJECTED → consent REJECTED
//     Any CREATED  → consent CREATED
//     All APPROVED → consent ACTIVE
func EvaluateConsentStatusFromAuthStatuses(authStatuses []string) string {
	cfg := config.Get()
	if cfg == nil {
		return "created" // safe fallback
	}
	consentConfig := cfg.Consent
	if len(authStatuses) == 0 {
		// No auth resources - default to created status
		return string(consentConfig.GetCreatedConsentStatus())
	}

	// Step 1: Filter out non-participating statuses.
	// RECORDED = passive participant (child, agent) — no action needed.
	// SYS_EXPIRED / SYS_REVOKED = system-managed states — should not corrupt derivation.
	recordedStatus := strings.ToUpper(string(consentConfig.GetRecordedAuthStatus()))
	sysExpiredStatus := strings.ToUpper(string(consentConfig.GetSystemExpiredAuthStatus()))
	sysRevokedStatus := strings.ToUpper(string(consentConfig.GetSystemRevokedAuthStatus()))

	participatingStatuses := make([]string, 0, len(authStatuses))
	for _, status := range authStatuses {
		upper := strings.ToUpper(status)
		if upper == recordedStatus || upper == sysExpiredStatus || upper == sysRevokedStatus {
			continue
		}
		participatingStatuses = append(participatingStatuses, status)
	}

	// If all statuses were filtered out but there were auth resources,
	// this shouldn't happen in practice because validation ensures at least one
	// non-RECORDED auth exists. Fall back to created as a safety net.
	if len(participatingStatuses) == 0 {
		return string(consentConfig.GetCreatedConsentStatus())
	}

	// Step 2: Evaluate participating statuses with priority logic
	hasRejected := false
	hasCreated := false
	allApproved := true

	for _, authStatus := range participatingStatuses {
		// Map auth status to consent status first (case-insensitive comparison)
		authStatusUpper := strings.ToUpper(authStatus)
		var mappedConsentStatus string

		// Check if auth status matches known auth states
		if authStatusUpper == strings.ToUpper(string(consentConfig.GetApprovedAuthStatus())) || authStatus == "" {
			// Approved or empty/missing status → active consent
			mappedConsentStatus = string(consentConfig.GetActiveConsentStatus())
		} else if authStatusUpper == strings.ToUpper(string(consentConfig.GetRejectedAuthStatus())) {
			// Rejected auth → rejected consent
			mappedConsentStatus = string(consentConfig.GetRejectedConsentStatus())
		} else if authStatusUpper == strings.ToUpper(string(consentConfig.GetCreatedAuthStatus())) {
			// Created auth → created consent
			mappedConsentStatus = string(consentConfig.GetCreatedConsentStatus())
		} else {
			// Unknown status - treat as created
			mappedConsentStatus = string(consentConfig.GetCreatedConsentStatus())
		}

		// Now check the mapped consent status
		if mappedConsentStatus == string(consentConfig.GetRejectedConsentStatus()) {
			hasRejected = true
			allApproved = false
		} else if mappedConsentStatus == string(consentConfig.GetCreatedConsentStatus()) {
			hasCreated = true
			allApproved = false
		} else if mappedConsentStatus != string(consentConfig.GetActiveConsentStatus()) {
			allApproved = false
		}
	}

	// Priority: rejected > created > approved (active)
	if hasRejected {
		return string(consentConfig.GetRejectedConsentStatus())
	} else if hasCreated {
		return string(consentConfig.GetCreatedConsentStatus())
	} else if allApproved {
		return string(consentConfig.GetActiveConsentStatus())
	}
	return string(consentConfig.GetCreatedConsentStatus())
}

// IsConsentExpired reports whether the given expiration timestamp (Unix milliseconds) has passed.
// Returns false if expirationTime is 0 (no expiry set).
func IsConsentExpired(expirationTime int64) bool {
	if expirationTime == 0 {
		return false
	}
	return time.Now().UnixMilli() > expirationTime
}
