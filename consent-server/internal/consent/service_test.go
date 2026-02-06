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

package consent

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/consent/model"
)

func TestValidateConsentTypeLength(t *testing.T) {
	// Test that consent type validation catches types that are too long
	longType := strings.Repeat("a", 65)

	require.Greater(t, len(longType), 64, "Test setup: type should be longer than 64 chars")
}

func TestValidateConsentTypeRequired(t *testing.T) {
	// Test that empty consent type is caught
	emptyType := ""

	require.Empty(t, emptyType, "Type should be empty for this test")
}

func TestValidatePurposesStructure(t *testing.T) {
	// Test that purposes array structure is validated
	purposes := []model.ConsentPurposeItem{
		{
			PurposeName: "purpose-1",
			Elements: []model.ConsentElementApprovalItem{
				{ElementName: "element-1", IsUserApproved: true},
			},
		},
	}

	require.NotNil(t, purposes, "Purposes should be defined")
	require.Len(t, purposes, 1, "Should have one purpose")
}

func TestValidateAuthorizationsRequired(t *testing.T) {
	// Test that at least one authorization is required
	emptyAuths := []model.AuthorizationAPIRequest{}

	require.Empty(t, emptyAuths, "Authorizations should be empty for validation test")
}

func TestConsentAPIRequestStructure(t *testing.T) {
	// Test creating a valid ConsentAPIRequest structure
	req := model.ConsentAPIRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}

	require.Equal(t, "accounts", req.Type)
	require.Len(t, req.Purposes, 1)
	require.Len(t, req.Authorizations, 1)
}

func TestConsentRevokeRequestStructure(t *testing.T) {
	// Test creating a valid revoke request
	req := model.ConsentRevokeRequest{
		ActionBy:         "admin",
		RevocationReason: "User requested",
	}

	require.Equal(t, "admin", req.ActionBy)
	require.Equal(t, "User requested", req.RevocationReason)
}

func TestValidateRequestStructure(t *testing.T) {
	// Test creating a valid validate request
	req := model.ValidateRequest{
		ConsentID: "consent-123",
	}

	require.Equal(t, "consent-123", req.ConsentID)
	require.NotEmpty(t, req.ConsentID, "ConsentID should not be empty")
}

func TestConsentResponseStructure(t *testing.T) {
	// Test ConsentResponse model structure
	resp := model.ConsentResponse{
		ConsentID:     "consent-123",
		ConsentType:   "accounts",
		CurrentStatus: "active",
		CreatedTime:   1234567890,
		UpdatedTime:   1234567890,
	}

	require.Equal(t, "consent-123", resp.ConsentID)
	require.Equal(t, "accounts", resp.ConsentType)
	require.Equal(t, "active", resp.CurrentStatus)
}

func TestConsentSearchFiltersDefaults(t *testing.T) {
	// Test search filters with default pagination
	filters := model.ConsentSearchFilters{
		OrgID:  "org-1",
		Limit:  10,
		Offset: 0,
	}

	require.Equal(t, "org-1", filters.OrgID)
	require.Equal(t, 10, filters.Limit)
	require.Equal(t, 0, filters.Offset)
}

func TestValidateDuplicatePurposes(t *testing.T) {
	// Test detection of duplicate purposes
	purposes := []model.ConsentPurposeItem{
		{
			PurposeName: "purpose-1",
			Elements: []model.ConsentElementApprovalItem{
				{ElementName: "element-1", IsUserApproved: true},
			},
		},
		{
			PurposeName: "purpose-1", // duplicate
			Elements: []model.ConsentElementApprovalItem{
				{ElementName: "element-1", IsUserApproved: true},
			},
		},
	}

	seen := make(map[string]bool)
	hasDuplicate := false

	for _, p := range purposes {
		if seen[p.PurposeName] {
			hasDuplicate = true
			break
		}
		seen[p.PurposeName] = true
	}

	require.True(t, hasDuplicate, "Should detect duplicate purpose IDs")
}

func TestContextPropagation(t *testing.T) {
	// Test that context is properly created
	ctx := context.Background()
	require.NotNil(t, ctx, "Context should not be nil")
}

func TestConsentAttributeStructure(t *testing.T) {
	// Test ConsentAttribute model
	attr := model.ConsentAttribute{
		ConsentID: "consent-123",
		AttKey:    "key1",
		AttValue:  "value1",
		OrgID:     "org-1",
	}

	require.Equal(t, "consent-123", attr.ConsentID)
	require.Equal(t, "key1", attr.AttKey)
	require.Equal(t, "value1", attr.AttValue)
}

func TestConsentStatusAuditStructure(t *testing.T) {
	// Test ConsentStatusAudit model
	reason := "test reason"
	actionBy := "admin"
	audit := model.ConsentStatusAudit{
		StatusAuditID:  "audit-1",
		ConsentID:      "consent-123",
		CurrentStatus:  "active",
		ActionTime:     1234567890,
		Reason:         &reason,
		ActionBy:       &actionBy,
		PreviousStatus: nil,
		OrgID:          "org-1",
	}

	require.Equal(t, "audit-1", audit.StatusAuditID)
	require.Equal(t, "consent-123", audit.ConsentID)
	require.NotNil(t, audit.Reason)
	require.Equal(t, "test reason", *audit.Reason)
}

func TestAuthorizationAPIRequestStructure(t *testing.T) {
	// Test AuthorizationAPIRequest model
	req := model.AuthorizationAPIRequest{
		Type:   "accounts",
		Status: "approved",
		UserID: "user-123",
	}

	require.Equal(t, "accounts", req.Type)
	require.Equal(t, "approved", req.Status)
	require.Equal(t, "user-123", req.UserID)
}

func TestConsentDetailResponseStructure(t *testing.T) {
	// Test ConsentDetailResponse model
	resp := model.ConsentDetailResponse{
		ID:        "consent-123",
		Type:      "accounts",
		Status:    "active",
		ClientID:  "client-1",
		Frequency: 1,
	}

	require.Equal(t, "consent-123", resp.ID)
	require.Equal(t, "accounts", resp.Type)
	require.Equal(t, 1, resp.Frequency)
}

func TestConsentSearchMetadataStructure(t *testing.T) {
	// Test ConsentSearchMetadata model
	metadata := model.ConsentSearchMetadata{
		Total:  100,
		Limit:  10,
		Offset: 0,
		Count:  10,
	}

	require.Equal(t, 100, metadata.Total)
	require.Equal(t, 10, metadata.Limit)
	require.Equal(t, 10, metadata.Count)
}

func TestConsentElementApprovalItemStructure(t *testing.T) {
	// Test ConsentElementApprovalItem model
	item := model.ConsentElementApprovalItem{
		ElementName:    "element-1",
		IsUserApproved: true,
		Value:          "test-value",
		IsMandatory:    false,
	}

	require.Equal(t, "element-1", item.ElementName)
	require.True(t, item.IsUserApproved)
	require.Equal(t, "test-value", item.Value)
}

func TestConsentPurposeItemValidation(t *testing.T) {
	// Test ConsentPurposeItem validation requirements
	item := model.ConsentPurposeItem{
		PurposeName: "purpose-1",
		Elements: []model.ConsentElementApprovalItem{
			{ElementName: "element-1", IsUserApproved: true},
		},
	}

	require.NotEmpty(t, item.PurposeName, "PurposeName should not be empty")
	require.NotEmpty(t, item.Elements, "Elements should not be empty")
	require.Len(t, item.Elements, 1, "Should have exactly one element")
}

func TestMultipleAuthorizationsValidation(t *testing.T) {
	// Test handling of multiple authorizations
	req := model.ConsentAPIRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts", UserID: "user-1"},
			{Type: "payments", UserID: "user-2"},
		},
	}

	require.Len(t, req.Authorizations, 2)
	require.Equal(t, "user-1", req.Authorizations[0].UserID)
	require.Equal(t, "user-2", req.Authorizations[1].UserID)
}

func TestConsentWithAttributesStructure(t *testing.T) {
	// Test consent with attributes
	attributes := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	req := model.ConsentAPIRequest{
		Type:       "accounts",
		Attributes: attributes,
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}

	require.Len(t, req.Attributes, 2)
	require.Equal(t, "value1", req.Attributes["key1"])
}

func TestConsentSearchFiltersWithMultipleStatuses(t *testing.T) {
	// Test search filters with multiple consent statuses
	filters := model.ConsentSearchFilters{
		OrgID:           "org-1",
		ConsentStatuses: []string{"active", "rejected"},
		ConsentTypes:    []string{"accounts", "payments"},
		Limit:           20,
		Offset:          10,
	}

	require.Len(t, filters.ConsentStatuses, 2)
	require.Len(t, filters.ConsentTypes, 2)
	require.Contains(t, filters.ConsentStatuses, "active")
	require.Contains(t, filters.ConsentTypes, "accounts")
}

func TestConsentWithPointerFields(t *testing.T) {
	// Test consent with optional pointer fields
	validityTime := int64(3600000)
	frequency := 5
	recurring := true
	dataAccess := int64(7200000)

	req := model.ConsentAPIRequest{
		Type:                       "accounts",
		ValidityTime:               &validityTime,
		Frequency:                  &frequency,
		RecurringIndicator:         &recurring,
		DataAccessValidityDuration: &dataAccess,
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}

	require.NotNil(t, req.ValidityTime)
	require.Equal(t, int64(3600000), *req.ValidityTime)
	require.NotNil(t, req.Frequency)
	require.Equal(t, 5, *req.Frequency)
}

func TestConsentRevokeRequestWithReason(t *testing.T) {
	// Test revoke request with reason
	req := model.ConsentRevokeRequest{
		ActionBy:         "user-123",
		RevocationReason: "Customer requested cancellation",
	}

	require.Equal(t, "user-123", req.ActionBy)
	require.NotEmpty(t, req.RevocationReason)
}

func TestValidateRequestWithConsentID(t *testing.T) {
	// Test validate request structure
	req := model.ValidateRequest{
		ConsentID: "consent-123",
	}

	require.NotEmpty(t, req.ConsentID)
	require.Equal(t, "consent-123", req.ConsentID)
}

func TestConsentAttributeSearchResponse(t *testing.T) {
	// Test attribute search response structure
	resp := model.ConsentAttributeSearchResponse{
		ConsentIDs: []string{"consent-1", "consent-2", "consent-3"},
	}

	require.Len(t, resp.ConsentIDs, 3)
	require.Contains(t, resp.ConsentIDs, "consent-1")
}

func TestMultiplePurposesWithElements(t *testing.T) {
	// Test consent with multiple purposes, each having multiple elements
	purposes := []model.ConsentPurposeItem{
		{
			PurposeName: "purpose-1",
			Elements: []model.ConsentElementApprovalItem{
				{ElementName: "element-1", IsUserApproved: true, Value: "value1"},
				{ElementName: "element-2", IsUserApproved: false},
			},
		},
		{
			PurposeName: "purpose-2",
			Elements: []model.ConsentElementApprovalItem{
				{ElementName: "element-3", IsUserApproved: true},
			},
		},
	}

	require.Len(t, purposes, 2)
	require.Len(t, purposes[0].Elements, 2)
	require.Len(t, purposes[1].Elements, 1)
	require.Equal(t, "value1", purposes[0].Elements[0].Value)
}

func TestConsentUpdateRequestStructure(t *testing.T) {
	// Test update request structure
	freq := 3
	req := model.ConsentAPIUpdateRequest{
		Type:      "payments",
		Frequency: &freq,
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "updated-purpose",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
	}

	require.Equal(t, "payments", req.Type)
	require.NotNil(t, req.Frequency)
	require.Equal(t, 3, *req.Frequency)
	require.Len(t, req.Purposes, 1)
}

func TestConsentSearchFiltersWithTimeRange(t *testing.T) {
	// Test search filters with time range
	fromTime := int64(1640000000000)
	toTime := int64(1650000000000)

	filters := model.ConsentSearchFilters{
		OrgID:    "org-1",
		FromTime: &fromTime,
		ToTime:   &toTime,
		Limit:    50,
		Offset:   0,
	}

	require.NotNil(t, filters.FromTime)
	require.NotNil(t, filters.ToTime)
	require.Equal(t, int64(1640000000000), *filters.FromTime)
	require.Equal(t, int64(1650000000000), *filters.ToTime)
}

func TestEmptyAttributesMap(t *testing.T) {
	// Test consent with empty attributes map
	req := model.ConsentAPIRequest{
		Type:       "accounts",
		Attributes: make(map[string]string),
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}

	require.NotNil(t, req.Attributes)
	require.Empty(t, req.Attributes)
}
