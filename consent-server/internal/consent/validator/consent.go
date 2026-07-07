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

	authvalidator "github.com/wso2/openfgc/consent-server/internal/authresource/validator"
	"github.com/wso2/openfgc/consent-server/internal/consent/model"
	"github.com/wso2/openfgc/consent-server/internal/system/config"
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
// Priority: rejected > created > active.
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

	// Evaluate ALL auth statuses with priority logic
	hasRejected := false
	hasCreated := false
	allApproved := true

	for _, authStatus := range authStatuses {
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
