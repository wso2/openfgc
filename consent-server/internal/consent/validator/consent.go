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

	authvalidator "github.com/wso2/openfgc/internal/authresource/validator"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/system/config"
)

// ValidateConsentCreateRequest validates consent creation request
func ValidateConsentCreateRequest(req model.ConsentAPIRequest, clientID, orgID string) error {
	// Required fields
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if len(req.Type) > 64 {
		return fmt.Errorf("type must be at most 64 characters")
	}
	if clientID == "" {
		return fmt.Errorf("clientID is required")
	}
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}

	// Validate auth resources (Authorizations field)
	for i, authReq := range req.Authorizations {
		if authReq.Type == "" {
			return fmt.Errorf("authorizations[%d].type is required", i)
		}
		// Status is optional and defaults to "created" in ToAuthResourceCreateRequest (or "approved" in consent-embedded flows)
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

	// Validate validity time if provided
	if req.ValidityTime != nil && *req.ValidityTime < 0 {
		return fmt.Errorf("validityTime must be non-negative")
	}

	// Validate frequency if provided
	if req.Frequency != nil && *req.Frequency < 0 {
		return fmt.Errorf("frequency must be non-negative")
	}

	return nil
}

// ValidateConsentUpdateRequest validates consent update request (keeping for future use)
func ValidateConsentUpdateRequest(req model.ConsentAPIUpdateRequest) error {
	// At least one field must be provided (check if nil, not if empty)
	// Empty arrays are valid - they indicate removal of all items
	if req.Type == "" && req.Frequency == nil &&
		req.ValidityTime == nil && req.RecurringIndicator == nil &&
		req.Attributes == nil && req.Authorizations == nil && req.Purposes == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}

	// Validate Type length if provided (match create constraint)
	if req.Type != "" && len(req.Type) > 64 {
		return fmt.Errorf("type must be at most 64 characters")
	}

	// Validate validity time if provided
	if req.ValidityTime != nil && *req.ValidityTime < 0 {
		return fmt.Errorf("validityTime must be non-negative")
	}

	// Validate frequency if provided
	if req.Frequency != nil && *req.Frequency < 0 {
		return fmt.Errorf("frequency must be non-negative")
	}

	// Validate auth resources if provided
	if req.Authorizations != nil {
		for i, authReq := range req.Authorizations {
			if authReq.Type == "" {
				return fmt.Errorf("authorizations[%d].type is required", i)
			}
			// Validate auth status if provided
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
	}

	return nil
}

// ValidateConsentGetRequest validates consent retrieval request parameters
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
// This is a helper function for authresource package to avoid import cycles.
// Uses the same priority logic as EvaluateConsentStatus.
func EvaluateConsentStatusFromAuthStatuses(authStatuses []string) string {
	consentConfig := config.Get().Consent

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
	} else {
		return string(consentConfig.GetCreatedConsentStatus())
	}
}

// IsExpired checks if a given validity time has expired
func IsConsentExpired(validityTime int64) bool {
	if validityTime == 0 {
		return false // No expiry set
	}

	// Detect if timestamp is in seconds or milliseconds
	// A reasonable cutoff: timestamps > 10^11 are likely in milliseconds
	// This works until year 5138 in seconds (safely covers our use case)
	const timestampCutoff = 100000000000 // 10^11

	var validityTimeMillis int64
	if validityTime < timestampCutoff {
		// Timestamp is in seconds, convert to milliseconds
		validityTimeMillis = validityTime * 1000
	} else {
		// Timestamp is already in milliseconds
		validityTimeMillis = validityTime
	}

	currentTimeMillis := time.Now().UnixNano() / int64(time.Millisecond)
	return currentTimeMillis > validityTimeMillis
}
