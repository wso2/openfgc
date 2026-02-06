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
	"time"

	"github.com/stretchr/testify/require"
)

func TestConsent_GetCreatedTime(t *testing.T) {
	consent := Consent{
		CreatedTime: 1640000000000,
	}
	createdTime := consent.GetCreatedTime()
	require.NotZero(t, createdTime)
}

func TestConsent_GetUpdatedTime(t *testing.T) {
	consent := Consent{
		UpdatedTime: 1650000000000,
	}
	updatedTime := consent.GetUpdatedTime()
	require.NotZero(t, updatedTime)
}

func TestConsentResponse_TimeConversion(t *testing.T) {
	now := time.Now()
	nowMillis := now.UnixMilli()
	consent := Consent{
		ConsentID:     "test",
		CreatedTime:   nowMillis,
		UpdatedTime:   nowMillis,
		ConsentType:   "accounts",
		CurrentStatus: "active",
	}
	createdTime := consent.GetCreatedTime()
	updatedTime := consent.GetUpdatedTime()
	require.WithinDuration(t, now, createdTime, time.Second)
	require.WithinDuration(t, now, updatedTime, time.Second)
}

func TestConsentSearchFilters_Structure(t *testing.T) {
	fromTime := int64(1640000000000)
	toTime := int64(1650000000000)
	filters := ConsentSearchFilters{
		ConsentTypes:    []string{"accounts"},
		ConsentStatuses: []string{"active"},
		FromTime:        &fromTime,
		ToTime:          &toTime,
		Limit:           50,
		OrgID:           "org-123",
	}
	require.NotNil(t, filters.FromTime)
	require.Equal(t, 50, filters.Limit)
}

func TestConsentSearchMetadata_Structure(t *testing.T) {
	metadata := ConsentSearchMetadata{
		Total:  100,
		Limit:  20,
		Offset: 40,
		Count:  20,
	}
	require.Equal(t, 100, metadata.Total)
	require.Equal(t, 20, metadata.Count)
}

func TestValidateResponse_Success(t *testing.T) {
	resp := ValidateResponse{
		IsValid: true,
		ConsentInformation: &ValidateConsentAPIResponse{
			ID:   "consent-123",
			Type: "accounts",
		},
	}
	require.True(t, resp.IsValid)
	require.NotNil(t, resp.ConsentInformation)
}

func TestValidateResponse_Error(t *testing.T) {
	resp := ValidateResponse{
		IsValid:      false,
		ErrorCode:    404,
		ErrorMessage: "invalid_consent",
	}
	require.False(t, resp.IsValid)
	require.Equal(t, 404, resp.ErrorCode)
}

func TestConsentRevokeResponse_Structure(t *testing.T) {
	resp := ConsentRevokeResponse{
		ActionTime: 1640000000000,
		ActionBy:   "user-123",
	}
	require.Equal(t, int64(1640000000000), resp.ActionTime)
	require.Equal(t, "user-123", resp.ActionBy)
}

func TestConsentAttributeSearchResponse_Structure(t *testing.T) {
	resp := ConsentAttributeSearchResponse{
		ConsentIDs: []string{"consent-1", "consent-2"},
	}
	require.Len(t, resp.ConsentIDs, 2)
}

func TestConsentDetailResponse_Structure(t *testing.T) {
	resp := ConsentDetailResponse{
		ID:         "consent-123",
		Type:       "accounts",
		Status:     "active",
		Frequency:  5,
		Attributes: map[string]string{"key1": "value1"},
	}
	require.Equal(t, "consent-123", resp.ID)
	require.Equal(t, 5, resp.Frequency)
}

func TestConsentPurposeItem_Structure(t *testing.T) {
	item := ConsentPurposeItem{
		PurposeName: "purpose-1",
		Elements: []ConsentElementApprovalItem{
			{ElementName: "element-1", IsUserApproved: true, Value: "test-value"},
		},
	}
	require.Equal(t, "purpose-1", item.PurposeName)
	require.Len(t, item.Elements, 1)
}

func TestConsentPurposeItem_JSONMarshaling(t *testing.T) {
	item := ConsentPurposeItem{
		PurposeName: "purpose-1",
		Elements: []ConsentElementApprovalItem{
			{ElementName: "element-1", IsUserApproved: true, Value: "test-value"},
		},
	}
	data, err := json.Marshal(item)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var unmarshaled ConsentPurposeItem
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, "purpose-1", unmarshaled.PurposeName)
}

func TestAuthorizationDetail_Structure(t *testing.T) {
	auth := AuthorizationDetail{
		ID:          "auth-123",
		UserID:      "user-123",
		Type:        "accounts",
		Status:      "approved",
		UpdatedTime: 1640000000000,
	}
	require.Equal(t, "auth-123", auth.ID)
	require.Equal(t, "user-123", auth.UserID)
}

func TestConsentElementApprovalItem_Structure(t *testing.T) {
	element := ConsentElementApprovalItem{
		ElementName:    "element-1",
		IsUserApproved: true,
		Value:          "test-value",
		IsMandatory:    false,
	}
	require.Equal(t, "element-1", element.ElementName)
	require.True(t, element.IsUserApproved)
}
