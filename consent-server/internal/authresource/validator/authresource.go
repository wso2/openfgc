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

	"github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/system/config"
)

// ValidateAuthResourceCreateRequest validates auth resource creation request
func ValidateAuthResourceCreateRequest(req model.ConsentAuthResourceCreateRequest, consentID, orgID string) error {
	if consentID == "" {
		return fmt.Errorf("consentID is required")
	}
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}
	if req.AuthType == "" {
		return fmt.Errorf("authType is required")
	}
	if req.AuthStatus == "" {
		return fmt.Errorf("authStatus is required")
	}

	// Validate auth status
	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("configuration not initialized")
	}
	if err := ValidateAuthStatus(req.AuthStatus, cfg.Consent.AuthStatusMappings); err != nil {
		return err
	}

	return nil
}

// ValidateAuthStatus validates authorization status and rejects system-reserved statuses
func ValidateAuthStatus(status string, mappings config.AuthStatusMappings) error {
	if mappings.SystemExpiredState == status ||
		mappings.SystemRevokedState == status {
		return fmt.Errorf("authorization status '%s' is system-reserved and cannot be set by users", status)
	}
	return nil
}

// ValidateAuthResourceUpdateRequest validates auth resource update request
func ValidateAuthResourceUpdateRequest(req model.ConsentAuthResourceUpdateRequest) error {
	// At least one field must be provided
	if req.AuthStatus == "" && req.UserID == nil && req.Resources == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}

	// Validate status if provided
	if req.AuthStatus != "" {
		cfg := config.Get()
		if cfg == nil {
			return fmt.Errorf("configuration not initialized")
		}
		if err := ValidateAuthStatus(req.AuthStatus, cfg.Consent.AuthStatusMappings); err != nil {
			return err
		}
	}

	return nil
}
