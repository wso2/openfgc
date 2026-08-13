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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// =============================================================================
// DB type tests
// =============================================================================

func TestAuthResource_Fields(t *testing.T) {
	userID := "user-123"
	resources := `{"accounts": ["acc1", "acc2"]}`
	r := AuthResource{
		AuthID:      "auth-123",
		ConsentID:   "consent-456",
		AuthType:    "accounts",
		UserID:      &userID,
		AuthStatus:  "APPROVED",
		UpdatedTime: 1234567890,
		Resources:   &resources,
		OrgID:       "org-123",
	}
	require.Equal(t, "auth-123", r.AuthID)
	require.NotNil(t, r.UserID)
	require.Equal(t, "user-123", *r.UserID)
}

func TestAuthResource_NilFields(t *testing.T) {
	r := AuthResource{AuthID: "auth-1", AuthType: "default", AuthStatus: "APPROVED"}
	require.Nil(t, r.UserID)
	require.Nil(t, r.Resources)
}

// =============================================================================
// Service input type tests
// =============================================================================

func TestCreateAuthResourceInput_TypeOptional(t *testing.T) {
	// Type is optional — empty string signals "use DefaultAuthType"
	in := CreateAuthResourceInput{AuthStatus: "APPROVED"}
	require.Empty(t, in.AuthType)
}

func TestCreateAuthResourceInput_Fields(t *testing.T) {
	userID := "user-123"
	in := CreateAuthResourceInput{
		AuthType:   "authorisation",
		UserID:     &userID,
		AuthStatus: "APPROVED",
		Resources:  map[string]interface{}{"accounts": []string{"acc1"}},
	}
	require.Equal(t, "authorisation", in.AuthType)
	require.NotNil(t, in.UserID)
}

func TestUpdateAuthResourceInput_Fields(t *testing.T) {
	in := UpdateAuthResourceInput{AuthStatus: "REVOKED"}
	require.Equal(t, "REVOKED", in.AuthStatus)
	require.Empty(t, in.AuthType)
}

// =============================================================================
// Service return type tests
// =============================================================================

func TestAuthResourceOutput_Fields(t *testing.T) {
	userID := "user-1"
	out := AuthResourceOutput{
		AuthID:     "auth-1",
		AuthType:   "default",
		UserID:     &userID,
		AuthStatus: "APPROVED",
		Resources:  map[string]interface{}{"accountIds": []string{"acc-1"}},
	}
	require.Equal(t, "default", out.AuthType)
	require.NotNil(t, out.Resources)
}

func TestAuthResourceListOutput_Fields(t *testing.T) {
	out := AuthResourceListOutput{
		Data: []AuthResourceOutput{
			{AuthID: "auth-1", AuthType: "default"},
			{AuthID: "auth-2", AuthType: "authorisation"},
		},
	}
	require.Len(t, out.Data, 2)
}

// =============================================================================
// API request type tests
// =============================================================================

func TestAuthResourceCreateRequest_TypeOmitted(t *testing.T) {
	// Omitting type must produce no "type" key in JSON
	req := AuthResourceCreateRequest{Status: "APPROVED"}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	require.NotContains(t, string(data), `"type"`, "omitted type must not appear in JSON")
}

func TestAuthResourceCreateRequest_JSONMarshal(t *testing.T) {
	userID := "user-1"
	req := AuthResourceCreateRequest{
		UserID:    &userID,
		Type:      "authorisation",
		Status:    "APPROVED",
		Resources: map[string]interface{}{"accounts": []string{"acc1"}},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded AuthResourceCreateRequest
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "authorisation", decoded.Type)
	require.Equal(t, "APPROVED", decoded.Status)
}

func TestAuthResourceUpdateRequest_JSONMarshal(t *testing.T) {
	req := AuthResourceUpdateRequest{Status: "REVOKED", Resources: map[string]interface{}{"reason": "user request"}}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded AuthResourceUpdateRequest
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "REVOKED", decoded.Status)
}

// =============================================================================
// API response type tests
// =============================================================================

func TestAuthResourceResponse_JSONMarshal(t *testing.T) {
	userID := "user-1"
	resp := AuthResourceResponse{
		ID:          "auth-1",
		UserID:      &userID,
		Type:        "authorisation",
		Status:      "APPROVED",
		UpdatedTime: 1702800000,
		Resources:   map[string]interface{}{"accounts": []string{"acc1"}},
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded AuthResourceResponse
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "auth-1", decoded.ID)
	require.Equal(t, "authorisation", decoded.Type)
}

func TestAuthTypeConstants(t *testing.T) {
	require.Equal(t, "primary", AuthTypePrimary)
	require.Equal(t, "delegate", AuthTypeDelegate)
	require.Equal(t, "delegate_subject", AuthTypeDelegateSubject)
}

func TestIsDelegationAuthType(t *testing.T) {
	require.True(t, IsDelegationAuthType(AuthTypeDelegate))
	require.True(t, IsDelegationAuthType(AuthTypeDelegateSubject))
	require.False(t, IsDelegationAuthType(AuthTypePrimary))
	require.False(t, IsDelegationAuthType("agent"))
	require.False(t, IsDelegationAuthType(""))
}
