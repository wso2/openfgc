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

package utils

import (
	"fmt"
	"net/http"

	"github.com/wso2/openfgc/internal/system/constants"
)

// Validate orgID and clientID in the request headers.
func ValidateOrgIdAndClientIdIsPresent(r *http.Request) error {
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get(constants.HeaderTPPClientID)

	if err := ValidateOrgID(orgID); err != nil {
		return err
	}
	if err := ValidateClientID(clientID); err != nil {
		return err
	}
	return nil
}

// ValidateOrgID validates organization ID
func ValidateOrgID(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}
	if len(orgID) > 255 {
		return fmt.Errorf("organization ID too long (max 255 chars)")
	}
	return nil
}

// ValidateClientID validates client ID
func ValidateClientID(clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if len(clientID) > 255 {
		return fmt.Errorf("client ID too long (max 255 chars)")
	}
	return nil
}

// ValidateRequired validates a field is not empty
func ValidateRequired(fieldName, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidatePagination validates limit and offset
func ValidatePagination(limit, offset int) error {
	if limit < 1 || limit > 100 {
		return fmt.Errorf("limit must be between 1 and 100")
	}
	if offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	return nil
}

// ValidateUUID validates UUID format using existing IsValidUUID
func ValidateUUID(id string) error {
	if !IsValidUUID(id) {
		return fmt.Errorf("invalid UUID format: %s", id)
	}
	return nil
}

// ValidateConsentID validates consent ID format
func ValidateConsentID(consentID string) error {
	if err := ValidateRequired("consentID", consentID); err != nil {
		return err
	}
	if len(consentID) > 100 {
		return fmt.Errorf("consent ID too long (max 100 chars)")
	}
	return ValidateUUID(consentID)
}
