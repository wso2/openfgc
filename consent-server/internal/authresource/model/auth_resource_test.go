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

func TestConsentAuthResource_Structure(t *testing.T) {
	userID := "user-123"
	resources := `{"accounts": ["acc1", "acc2"]}`

	authResource := ConsentAuthResource{
		AuthID:      "auth-123",
		ConsentID:   "consent-456",
		AuthType:    "accounts",
		UserID:      &userID,
		AuthStatus:  "authorized",
		UpdatedTime: 1234567890,
		Resources:   &resources,
		OrgID:       "org-123",
	}

	require.Equal(t, "auth-123", authResource.AuthID)
	require.Equal(t, "consent-456", authResource.ConsentID)
	require.Equal(t, "accounts", authResource.AuthType)
	require.NotNil(t, authResource.UserID)
	require.Equal(t, "user-123", *authResource.UserID)
	require.Equal(t, "authorized", authResource.AuthStatus)
}

func TestConsentAuthResourceCreateRequest_JSONMarshaling(t *testing.T) {
	userID := "user-123"
	req := ConsentAuthResourceCreateRequest{
		AuthType:   "accounts",
		UserID:     &userID,
		AuthStatus: "authorized",
		Resources:  map[string]interface{}{"accounts": []string{"acc1"}},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var unmarshaled ConsentAuthResourceCreateRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, "accounts", unmarshaled.AuthType)
	require.Equal(t, "authorized", unmarshaled.AuthStatus)
}

func TestConsentAuthResourceUpdateRequest_JSONMarshaling(t *testing.T) {
	status := "revoked"
	userID := "user-456"
	req := ConsentAuthResourceUpdateRequest{
		AuthStatus: status,
		UserID:     &userID,
		Resources:  map[string]interface{}{"reason": "user requested"},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var unmarshaled ConsentAuthResourceUpdateRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, "revoked", unmarshaled.AuthStatus)
}

func TestConsentAuthResourceResponse_JSONMarshaling(t *testing.T) {
	userID := "user-123"
	resp := ConsentAuthResourceResponse{
		AuthID:      "auth-123",
		AuthType:    "accounts",
		UserID:      &userID,
		AuthStatus:  "authorized",
		UpdatedTime: 1234567890,
		Resources:   map[string]interface{}{"accounts": []string{"acc1"}},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var unmarshaled ConsentAuthResourceResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, "auth-123", unmarshaled.AuthID)
	require.Equal(t, "accounts", unmarshaled.AuthType)
}

func TestConsentAuthResourceListResponse_Structure(t *testing.T) {
	listResp := ConsentAuthResourceListResponse{
		Data: []ConsentAuthResourceResponse{
			{
				AuthID:     "auth-1",
				AuthType:   "accounts",
				AuthStatus: "authorized",
			},
			{
				AuthID:     "auth-2",
				AuthType:   "payments",
				AuthStatus: "pending",
			},
		},
	}

	require.Len(t, listResp.Data, 2)
	require.Equal(t, "auth-1", listResp.Data[0].AuthID)
	require.Equal(t, "auth-2", listResp.Data[1].AuthID)
}

func TestAuthResourceTypeAlias(t *testing.T) {
	var authResource AuthResource
	authResource.AuthID = "test-id"
	require.Equal(t, "test-id", authResource.AuthID)
}

func TestCreateRequestTypeAlias(t *testing.T) {
	var req CreateRequest
	req.AuthType = "accounts"
	require.Equal(t, "accounts", req.AuthType)
}

func TestUpdateRequestTypeAlias(t *testing.T) {
	var req UpdateRequest
	status := "revoked"
	req.AuthStatus = status
	require.Equal(t, "revoked", req.AuthStatus)
}

func TestResponseTypeAlias(t *testing.T) {
	var resp Response
	resp.AuthID = "test-id"
	require.Equal(t, "test-id", resp.AuthID)
}

func TestListResponseTypeAlias(t *testing.T) {
	var listResp ListResponse
	listResp.Data = []Response{{AuthID: "auth-1"}}
	require.Len(t, listResp.Data, 1)
}

func TestConsentAuthResource_NilFields(t *testing.T) {
	authResource := ConsentAuthResource{
		AuthID:      "auth-123",
		ConsentID:   "consent-456",
		AuthType:    "accounts",
		AuthStatus:  "authorized",
		UpdatedTime: 1234567890,
		OrgID:       "org-123",
	}

	require.Nil(t, authResource.UserID)
	require.Nil(t, authResource.Resources)
	require.Nil(t, authResource.ResourceObj)
}

func TestConsentAuthResourceCreateRequest_OptionalFields(t *testing.T) {
	req := ConsentAuthResourceCreateRequest{
		AuthType:   "accounts",
		AuthStatus: "authorized",
	}

	require.Nil(t, req.UserID)
	require.Nil(t, req.Resources)
}

func TestConsentAuthResourceUpdateRequest_EmptyFields(t *testing.T) {
	req := ConsentAuthResourceUpdateRequest{}

	require.Empty(t, req.AuthStatus)
	require.Nil(t, req.UserID)
	require.Nil(t, req.Resources)
}
