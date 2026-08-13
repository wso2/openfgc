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
// Authorization type is optional — the service defaults it to "primary" when absent.
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

// ValidateAuthTypeConstraints validates authorization types and participation across a full
// set of authorizations.
//
// Delegation is modelled by two authorization types: "delegate", the person consenting on
// behalf of another, and "delegate_subject", the person the consent is about who cannot
// consent themselves. A consent using either type is a delegated consent, and must contain
// both and nothing besides — admitting "primary" or a custom type alongside them would leave
// it ambiguous whose decision the consent records.
//
// Consents that use neither delegation type are unconstrained: "primary" and custom types
// such as "agent" or "account_owner" may appear in any combination.
//
// Independently of type, at least one authorization must carry a status other than the
// configured recorded status, since a set of exclusively passive participants records no
// decision by anyone.
func ValidateAuthTypeConstraints(authorizations []model.AuthorizationRequest) error {
	auths := make([]authTypeStatus, 0, len(authorizations))
	for _, auth := range authorizations {
		auths = append(auths, authTypeStatus{authType: effectiveAuthType(auth.Type), status: auth.Status})
	}

	return validateAuthTypeConstraints(auths)
}

// ValidateAuthResourceTypeConstraints applies the same rules as ValidateAuthTypeConstraints to
// the authorizations a consent holds once a write completes. Callers that add or modify a single
// authorization pass the resulting set rather than the incoming change, since the rules describe
// a consent as a whole and cannot be evaluated from one authorization in isolation.
func ValidateAuthResourceTypeConstraints(resources []authmodel.AuthResource) error {
	auths := make([]authTypeStatus, 0, len(resources))
	for _, resource := range resources {
		auths = append(auths, authTypeStatus{authType: effectiveAuthType(resource.AuthType), status: resource.AuthStatus})
	}

	return validateAuthTypeConstraints(auths)
}

// authTypeStatus is the projection of an authorization the type and participation rules act on.
// It lets an incoming request and a stored authorization resource be checked by the same code.
type authTypeStatus struct {
	authType string
	status   string
}

func validateAuthTypeConstraints(auths []authTypeStatus) error {
	if len(auths) == 0 {
		return nil
	}

	if err := validateParticipation(auths); err != nil {
		return err
	}

	return validateDelegationTypes(auths)
}

// validateParticipation requires at least one authorization whose status is not the configured
// recorded status. The check is skipped when no recorded status is configured, leaving the
// status validation applied elsewhere as the only constraint.
func validateParticipation(auths []authTypeStatus) error {
	cfg := config.Get()
	if cfg == nil {
		return nil
	}

	recordedStatus := string(cfg.Consent.GetRecordedAuthStatus())
	if recordedStatus == "" {
		return nil
	}

	for _, auth := range auths {
		// An omitted status is defaulted to the approved state by the service layer. The
		// comparison ignores case to match how derivation recognises the recorded status.
		if auth.status == "" || !strings.EqualFold(auth.status, recordedStatus) {
			return nil
		}
	}

	return fmt.Errorf("at least one authorization must have an active status; %s alone does not constitute consent",
		recordedStatus)
}

// validateDelegationTypes enforces the shape of a delegated consent. Consents that use no
// delegation type are left unvalidated.
func validateDelegationTypes(auths []authTypeStatus) error {
	delegated := false
	for _, auth := range auths {
		if authmodel.IsDelegationAuthType(auth.authType) {
			delegated = true
			break
		}
	}
	if !delegated {
		return nil
	}

	hasDelegate := false
	hasDelegateSubject := false

	for _, auth := range auths {
		switch auth.authType {
		case authmodel.AuthTypeDelegate:
			hasDelegate = true
		case authmodel.AuthTypeDelegateSubject:
			hasDelegateSubject = true
		default:
			return fmt.Errorf("a delegated consent may only contain '%s' and '%s' authorizations; found '%s'",
				authmodel.AuthTypeDelegate, authmodel.AuthTypeDelegateSubject, auth.authType)
		}
	}

	if !hasDelegate {
		return fmt.Errorf("authorization type '%s' requires at least one '%s' in the same consent",
			authmodel.AuthTypeDelegateSubject, authmodel.AuthTypeDelegate)
	}
	if !hasDelegateSubject {
		return fmt.Errorf("authorization type '%s' requires at least one '%s' in the same consent",
			authmodel.AuthTypeDelegate, authmodel.AuthTypeDelegateSubject)
	}

	return nil
}

// effectiveAuthType returns the authorization type the service layer persists, which is
// AuthTypePrimary when the caller omits one. Reserved types are normalized so a differently
// cased value (e.g. "Delegate") is still recognised and validated as a delegation type.
func effectiveAuthType(authType string) string {
	if authType == "" {
		return authmodel.AuthTypePrimary
	}
	return authmodel.NormalizeAuthType(authType)
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
// Statuses that record no decision are excluded first. RECORDED marks a participant present on
// the consent who takes no decision, such as the subject of a delegated consent. SYS_EXPIRED and
// SYS_REVOKED are set by the server across every authorization of a consent when it expires or is
// revoked, and describe the consent's lifecycle rather than any participant's choice.
//
// The remaining statuses are then evaluated by priority:
//
//	Any REJECTED → consent REJECTED
//	Any CREATED  → consent CREATED
//	All APPROVED → consent ACTIVE
//
// Comparison ignores case throughout.
func EvaluateConsentStatusFromAuthStatuses(authStatuses []string) string {
	cfg := config.Get()
	if cfg == nil {
		return "created"
	}
	consentConfig := cfg.Consent
	if len(authStatuses) == 0 {
		// No auth resources - default to created status
		return string(consentConfig.GetCreatedConsentStatus())
	}

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

	// Every authorization is recorded or system-set, leaving no decision to evaluate.
	if len(participatingStatuses) == 0 {
		return string(consentConfig.GetCreatedConsentStatus())
	}

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
