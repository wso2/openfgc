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
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/system/config"
)

func TestValidateAuthResourceCreateRequest_Success(t *testing.T) {
	t.Skip("Skipping - requires config initialization")
	req := model.ConsentAuthResourceCreateRequest{
		AuthType:   "accounts",
		AuthStatus: "authorized",
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "org-123")
	require.NoError(t, err)
}

func TestValidateAuthResourceCreateRequest_MissingConsentID(t *testing.T) {
	req := model.ConsentAuthResourceCreateRequest{
		AuthType:   "accounts",
		AuthStatus: "authorized",
	}

	err := ValidateAuthResourceCreateRequest(req, "", "org-123")
	require.Error(t, err)
	require.Contains(t, err.Error(), "consentID is required")
}

func TestValidateAuthResourceCreateRequest_MissingOrgID(t *testing.T) {
	req := model.ConsentAuthResourceCreateRequest{
		AuthType:   "accounts",
		AuthStatus: "authorized",
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "orgID is required")
}

func TestValidateAuthResourceCreateRequest_MissingAuthType(t *testing.T) {
	req := model.ConsentAuthResourceCreateRequest{
		AuthStatus: "authorized",
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "org-123")
	require.Error(t, err)
	require.Contains(t, err.Error(), "authType is required")
}

func TestValidateAuthResourceCreateRequest_MissingAuthStatus(t *testing.T) {
	req := model.ConsentAuthResourceCreateRequest{
		AuthType: "accounts",
	}

	err := ValidateAuthResourceCreateRequest(req, "consent-123", "org-123")
	require.Error(t, err)
	require.Contains(t, err.Error(), "authStatus is required")
}

func TestValidateAuthStatus_ValidStatus(t *testing.T) {
	// Create mock mappings
	mappings := config.AuthStatusMappings{
		SystemExpiredState: "system_expired",
		SystemRevokedState: "system_revoked",
	}

	err := ValidateAuthStatus("authorized", mappings)
	require.NoError(t, err)
}

func TestValidateAuthResourceUpdateRequest_Success(t *testing.T) {
	t.Skip("Skipping - requires config initialization")
	status := "revoked"
	req := model.ConsentAuthResourceUpdateRequest{
		AuthStatus: status,
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateAuthResourceUpdateRequest_WithUserID(t *testing.T) {
	userID := "user-123"
	req := model.ConsentAuthResourceUpdateRequest{
		UserID: &userID,
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateAuthResourceUpdateRequest_WithResources(t *testing.T) {
	resources := map[string]interface{}{"key": "value"}
	req := model.ConsentAuthResourceUpdateRequest{
		Resources: resources,
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateAuthResourceUpdateRequest_EmptyRequest(t *testing.T) {
	req := model.ConsentAuthResourceUpdateRequest{}

	err := ValidateAuthResourceUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one field must be provided")
}

func TestValidateAuthResourceUpdateRequest_MultipleFields(t *testing.T) {
	t.Skip("Skipping - requires config initialization")
	status := "authorized"
	userID := "user-123"
	req := model.ConsentAuthResourceUpdateRequest{
		AuthStatus: status,
		UserID:     &userID,
		Resources:  map[string]interface{}{"key": "value"},
	}

	err := ValidateAuthResourceUpdateRequest(req)
	require.NoError(t, err)
}
