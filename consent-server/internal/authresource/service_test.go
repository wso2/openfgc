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

package authresource

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/authresource/model"
)

func TestAuthResourceStructure(t *testing.T) {
	userID := "user-123"
	resources := `{"accounts": ["acc1", "acc2"]}`

	authResource := model.AuthResource{
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
	require.Equal(t, int64(1234567890), authResource.UpdatedTime)
	require.NotNil(t, authResource.Resources)
	require.Equal(t, "org-123", authResource.OrgID)
}

func TestCreateRequestStructure(t *testing.T) {
	userID := "user-123"
	createReq := model.CreateRequest{
		AuthType:   "accounts",
		UserID:     &userID,
		AuthStatus: "authorized",
		Resources:  map[string]interface{}{"accounts": []string{"acc1"}},
	}

	require.Equal(t, "accounts", createReq.AuthType)
	require.NotNil(t, createReq.UserID)
	require.Equal(t, "user-123", *createReq.UserID)
	require.Equal(t, "authorized", createReq.AuthStatus)
	require.NotNil(t, createReq.Resources)
}

func TestUpdateRequestStructure(t *testing.T) {
	newStatus := "revoked"
	updateReq := model.UpdateRequest{
		AuthStatus: newStatus,
		Resources:  map[string]interface{}{"reason": "user revoked"},
	}

	require.NotNil(t, updateReq.AuthStatus)
	require.Equal(t, "revoked", updateReq.AuthStatus)
	require.NotNil(t, updateReq.Resources)
}

func TestResponseStructure(t *testing.T) {
	userID := "user-123"
	resp := model.Response{
		AuthID:      "auth-123",
		AuthType:    "accounts",
		UserID:      &userID,
		AuthStatus:  "authorized",
		UpdatedTime: 1234567890,
		Resources:   map[string]interface{}{"accounts": []string{"acc1"}},
	}

	require.Equal(t, "auth-123", resp.AuthID)
	require.Equal(t, "accounts", resp.AuthType)
	require.NotNil(t, resp.UserID)
	require.Equal(t, "authorized", resp.AuthStatus)
}

func TestListResponseStructure(t *testing.T) {
	listResp := model.ListResponse{
		Data: []model.Response{
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

func TestAuthResourceNilFields(t *testing.T) {
	authResource := model.AuthResource{
		AuthID:      "auth-123",
		ConsentID:   "consent-456",
		AuthType:    "accounts",
		AuthStatus:  "authorized",
		UpdatedTime: 1234567890,
		OrgID:       "org-123",
	}

	require.Nil(t, authResource.UserID)
	require.Nil(t, authResource.Resources)
}

func TestCreateRequestOptionalFields(t *testing.T) {
	createReq := model.CreateRequest{
		AuthType:   "accounts",
		AuthStatus: "authorized",
	}

	require.Nil(t, createReq.UserID)
	require.Nil(t, createReq.Resources)
}

func TestUpdateRequestOptionalFields(t *testing.T) {
	updateReq := model.UpdateRequest{}

	require.Empty(t, updateReq.AuthStatus)
	require.Nil(t, updateReq.UserID)
	require.Nil(t, updateReq.Resources)
}
