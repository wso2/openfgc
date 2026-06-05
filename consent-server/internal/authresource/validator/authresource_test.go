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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/system/config"
)

// TestMain initializes a minimal configuration required by the validator
func TestMain(m *testing.M) {
	cfg := &config.Config{
		Consent: config.ConsentConfig{
			AuthStatusMappings: config.AuthStatusMappings{
				ApprovedState:      "authorized",
				RejectedState:      "rejected",
				CreatedState:       "created",
				SystemExpiredState: "system_expired",
				SystemRevokedState: "system_revoked",
			},
		},
	}
	config.SetGlobal(cfg)
	os.Exit(m.Run())
}

// =============================================================================
// ValidateAuthResourceCreateRequest
// =============================================================================

func TestValidateAuthResourceCreateRequest_Success(t *testing.T) {
	// Type and Status are both optional; UserID is required.
	userID := "user-001"
	req := model.AuthResourceCreateRequest{
		UserID: &userID,
		Type:   "accounts",
		Status: "authorized",
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "org-123")
	require.NoError(t, err)
}

func TestValidateAuthResourceCreateRequest_MissingUserID(t *testing.T) {
	req := model.AuthResourceCreateRequest{
		Type:   "accounts",
		Status: "authorized",
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "org-123")
	require.Error(t, err)
	require.Contains(t, err.Error(), "userId is required")
}

func TestValidateAuthResourceCreateRequest_NoTypeOrStatus(t *testing.T) {
	// Both Type and Status are optional — only UserID is required.
	userID := "user-001"
	req := model.AuthResourceCreateRequest{
		UserID: &userID,
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "org-123")
	require.NoError(t, err)
}

func TestValidateAuthResourceCreateRequest_MissingConsentID(t *testing.T) {
	userID := "user-001"
	req := model.AuthResourceCreateRequest{
		UserID: &userID,
		Type:   "accounts",
		Status: "authorized",
	}

	err := ValidateAuthResourceCreateRequest(req, "", "org-123")
	require.Error(t, err)
	require.Contains(t, err.Error(), "consentID is required")
}

func TestValidateAuthResourceCreateRequest_MissingOrgID(t *testing.T) {
	userID := "user-001"
	req := model.AuthResourceCreateRequest{
		UserID: &userID,
		Type:   "accounts",
		Status: "authorized",
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "orgID is required")
}

func TestValidateAuthResourceCreateRequest_SystemReservedStatus(t *testing.T) {
	userID := "user-001"
	req := model.AuthResourceCreateRequest{
		UserID: &userID,
		Status: "system_expired",
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "org-123")
	require.Error(t, err)
	require.Contains(t, err.Error(), "system-reserved")
}

// =============================================================================
// ValidateAuthStatus
// =============================================================================

func TestValidateAuthStatus_ValidStatus(t *testing.T) {
	mappings := config.AuthStatusMappings{
		SystemExpiredState: "system_expired",
		SystemRevokedState: "system_revoked",
	}

	err := ValidateAuthStatus("authorized", mappings)
	require.NoError(t, err)
}

func TestValidateAuthStatus_SystemExpiredRejected(t *testing.T) {
	mappings := config.AuthStatusMappings{
		SystemExpiredState: "system_expired",
		SystemRevokedState: "system_revoked",
	}

	require.Error(t, ValidateAuthStatus("system_expired", mappings))
	require.Error(t, ValidateAuthStatus("system_revoked", mappings))
}

// =============================================================================
// ValidateAuthResourceUpdateRequest
// =============================================================================

func TestValidateAuthResourceUpdateRequest_Success(t *testing.T) {
	userID := "user-001"
	req := model.AuthResourceUpdateRequest{
		UserID: &userID,
		Status: "revoked",
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateAuthResourceUpdateRequest_MissingUserID(t *testing.T) {
	// userId is required — omitting it must return an error.
	req := model.AuthResourceUpdateRequest{
		Status: "revoked",
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "userId is required")
}

func TestValidateAuthResourceUpdateRequest_WithUserID(t *testing.T) {
	// UserID alone is not sufficient — at least one other field must also be present.
	userID := "user-123"
	req := model.AuthResourceUpdateRequest{
		UserID: &userID,
		Status: "authorized",
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateAuthResourceUpdateRequest_WithResources(t *testing.T) {
	userID := "user-001"
	req := model.AuthResourceUpdateRequest{
		UserID:    &userID,
		Resources: map[string]interface{}{"key": "value"},
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateAuthResourceUpdateRequest_EmptyRequest(t *testing.T) {
	// Neither UserID nor any other field provided — userId is checked first.
	req := model.AuthResourceUpdateRequest{}

	err := ValidateAuthResourceUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "userId is required")
}

func TestValidateAuthResourceUpdateRequest_UserIDButNoOtherField(t *testing.T) {
	// UserID present but no status/type/resources — must require at least one more field.
	userID := "user-001"
	req := model.AuthResourceUpdateRequest{
		UserID: &userID,
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one field must be provided")
}

func TestValidateAuthResourceUpdateRequest_MultipleFields(t *testing.T) {
	userID := "user-123"
	req := model.AuthResourceUpdateRequest{
		Status:    "authorized",
		UserID:    &userID,
		Resources: map[string]interface{}{"key": "value"},
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.NoError(t, err)
}
