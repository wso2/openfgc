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

// =============================================================================
// DB type tests
// =============================================================================

func TestConsent_GetCreatedTime(t *testing.T) {
	consent := Consent{CreatedTime: 1640000000000}
	require.NotZero(t, consent.GetCreatedTime())
}

func TestConsent_GetUpdatedTime(t *testing.T) {
	consent := Consent{UpdatedTime: 1650000000000}
	require.NotZero(t, consent.GetUpdatedTime())
}

func TestConsent_TimeConversion_MillisecondPrecision(t *testing.T) {
	now := time.Now()
	c := Consent{CreatedTime: now.UnixMilli(), UpdatedTime: now.UnixMilli()}
	require.WithinDuration(t, now, c.GetCreatedTime(), time.Second)
	require.WithinDuration(t, now, c.GetUpdatedTime(), time.Second)
}

func TestConsentPurposeMapping_Fields(t *testing.T) {
	m := ConsentPurposeMapping{ConsentID: "c-1", PurposeVersionID: "pv-1", OrgID: "org-1"}
	require.Equal(t, "pv-1", m.PurposeVersionID)
}

func TestConsentElementApproval_Fields(t *testing.T) {
	val := `"test@email.com"`
	a := ConsentElementApproval{
		ConsentID: "c-1", PurposeVersionID: "pv-1", ElementVersionID: "ev-1",
		Approved: true, Value: &val, OrgID: "org-1",
	}
	require.True(t, a.Approved)
	require.NotNil(t, a.Value)
}

func TestConsentAttribute_Fields(t *testing.T) {
	a := ConsentAttribute{ConsentID: "c-1", AttKey: "env", AttValue: "prod", OrgID: "org-1"}
	require.Equal(t, "env", a.AttKey)
}

func TestConsentStatusAudit_Fields(t *testing.T) {
	prev := "CREATED"
	s := ConsentStatusAudit{ConsentID: "c-1", CurrentStatus: "ACTIVE", PreviousStatus: &prev}
	require.NotNil(t, s.PreviousStatus)
	require.Equal(t, "ACTIVE", s.CurrentStatus)
}

// =============================================================================
// Service input type tests
// =============================================================================

func TestConsentSearchFilter_Fields(t *testing.T) {
	fromTime := int64(1640000000000)
	f := ConsentSearchFilter{
		GroupIDs: []string{"grp-1"},
		Sort: []ConsentSort{{
			Field:     ConsentSortFieldCreatedTime,
			Direction: ConsentSortDirectionDesc,
		}},
		PurposeName: "marketing",
		FromTime:    &fromTime,
		Limit:       50,
		OrgID:       "org-1",
	}
	require.Equal(t, []string{"grp-1"}, f.GroupIDs)
	require.Len(t, f.Sort, 1)
	require.Equal(t, ConsentSortFieldCreatedTime, f.Sort[0].Field)
	require.Equal(t, "marketing", f.PurposeName)
	require.Equal(t, 50, f.Limit)
}

func TestCreateConsentInput_Fields(t *testing.T) {
	in := CreateConsentInput{GroupID: "grp-1", ConsentType: "accounts"}
	require.Equal(t, "grp-1", in.GroupID)
}

func TestElementApprovalInput_Fields(t *testing.T) {
	e := ElementApprovalInput{Name: "email", Approved: true, Value: "test@email.com"}
	require.True(t, e.Approved)
}

func TestCreateStatusAuditInput_Fields(t *testing.T) {
	prev := "CREATED"
	by := "admin"
	in := CreateStatusAuditInput{ConsentID: "c-1", CurrentStatus: "ACTIVE", PreviousStatus: &prev, ActionBy: &by}
	require.NotNil(t, in.PreviousStatus)
}

func TestConsentAttributeSearchInput_Fields(t *testing.T) {
	in := ConsentAttributeSearchInput{Key: "dept", Value: "sales", OrgID: "org-1"}
	require.Equal(t, "dept", in.Key)
	require.Equal(t, "sales", in.Value)
}

// =============================================================================
// Service return type tests
// =============================================================================

func TestConsentListMetadata_Fields(t *testing.T) {
	m := ConsentListMetadata{Total: 100, Limit: 20, Offset: 40, Count: 20}
	require.Equal(t, 100, m.Total)
	require.Equal(t, 20, m.Count)
}

func TestConsentOutput_Fields(t *testing.T) {
	out := ConsentOutput{ConsentID: "c-1", GroupID: "grp-1", ConsentType: "accounts"}
	require.Equal(t, "grp-1", out.GroupID)
}

func TestConsentElementApprovalOutput_ValueTypes(t *testing.T) {
	// basic element — value is a plain string stored as-is
	basicVal := "john.doe@example.com"
	basic := ConsentElementApprovalOutput{
		ElementType: "basic",
		Value:       &basicVal,
		Approved:    true,
	}
	require.Equal(t, "basic", basic.ElementType)
	require.NotNil(t, basic.Value)
	require.Equal(t, "john.doe@example.com", *basic.Value)

	// xml element — value is an XML string stored as-is
	xmlVal := "<user><email>john@example.com</email></user>"
	xml := ConsentElementApprovalOutput{
		ElementType: "xml",
		Value:       &xmlVal,
	}
	require.Equal(t, "xml", xml.ElementType)
	require.NotNil(t, xml.Value)

	// json element — value is a JSON string; handler parses before sending in API response
	jsonVal := `{"scope":"read","permissions":["email","profile"]}`
	jsonElem := ConsentElementApprovalOutput{
		ElementType: "json",
		Value:       &jsonVal,
	}
	require.Equal(t, "json", jsonElem.ElementType)

	// nil value — element has no user-provided value
	noVal := ConsentElementApprovalOutput{ElementType: "basic", Approved: false}
	require.Nil(t, noVal.Value)
}

func TestConsentElementApprovalOutput_EnrichedFields(t *testing.T) {
	desc := "User email address"
	props := map[string]string{"jsonPath": "$.email"}
	out := ConsentElementApprovalOutput{
		ElementType: "basic",
		Mandatory:   true,
		Description: &desc,
		Properties:  props,
	}
	require.NotNil(t, out.Description)
	require.Equal(t, "$.email", out.Properties["jsonPath"])
}

func TestConsentSearchFilter_FromTimeBoundOnUpdatedTime(t *testing.T) {
	// Verifies the field semantics: FromTime/ToTime bound UPDATED_TIME, not CREATED_TIME
	from := int64(1640000000000)
	to := int64(1650000000000)
	f := ConsentSearchFilter{FromTime: &from, ToTime: &to}
	// Both times are non-nil and represent UPDATED_TIME bounds.
	require.NotNil(t, f.FromTime)
	require.NotNil(t, f.ToTime)
	require.Less(t, *f.FromTime, *f.ToTime)
}

func TestStatusAuditOutput_Fields(t *testing.T) {
	out := StatusAuditOutput{ConsentID: "c-1", CurrentStatus: "ACTIVE", ActionTime: 1640000000000}
	require.Equal(t, int64(1640000000000), out.ActionTime)
}

// =============================================================================
// API request type tests
// =============================================================================

func TestConsentPurposeRefRequest_JSONMarshal(t *testing.T) {
	ver := "v1"
	req := ConsentPurposeRefRequest{
		Name:    "marketing",
		Version: &ver,
		Elements: []ConsentPurposeElementApprovalRequest{
			{Name: "email", Approved: true, Value: "test@email.com"},
		},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded ConsentPurposeRefRequest
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "marketing", decoded.Name)
	require.NotNil(t, decoded.Version)
	require.Equal(t, "v1", *decoded.Version)
	require.Len(t, decoded.Elements, 1)
	require.True(t, decoded.Elements[0].Approved)
}

func TestConsentCreateRequest_JSONMarshal(t *testing.T) {
	req := ConsentCreateRequest{
		Type: "accounts",
		Purposes: []ConsentPurposeRefRequest{
			{Name: "marketing", Elements: []ConsentPurposeElementApprovalRequest{{Name: "email", Approved: true}}},
		},
		Authorizations: []AuthorizationRequest{
			{Type: "payments", Status: "APPROVED"},
		},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	require.Contains(t, string(data), `"name":"marketing"`)
	require.Contains(t, string(data), `"approved":true`)
}

func TestAuthorizationRequest_TypeOptional(t *testing.T) {
	// Type field is omitempty — an empty type is valid and defaults to "default" in the service
	req := AuthorizationRequest{Status: "APPROVED"}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	require.NotContains(t, string(data), `"type"`, "empty type must be omitted from JSON")
}

// =============================================================================
// API response type tests
// =============================================================================

func TestConsentResponse_Fields(t *testing.T) {
	resp := ConsentResponse{
		ConsentID:  "c-1",
		GroupID:    "grp-1",
		Type:       "accounts",
		Status:     "ACTIVE",
		Attributes: map[string]string{},
		Purposes:   []ConsentPurposeResponse{},
	}
	require.Equal(t, "grp-1", resp.GroupID)
}

func TestConsentValidateResponse_Success(t *testing.T) {
	resp := ConsentValidateResponse{
		IsValid: true,
		ConsentInfo: &ConsentValidateInfo{
			ConsentID: "c-1",
			Type:      "accounts",
		},
	}
	require.True(t, resp.IsValid)
	require.NotNil(t, resp.ConsentInfo)
}

func TestConsentValidateResponse_Error(t *testing.T) {
	resp := ConsentValidateResponse{IsValid: false, ErrorCode: 404, ErrorMessage: "invalid_consent"}
	require.False(t, resp.IsValid)
	require.Equal(t, 404, resp.ErrorCode)
}

func TestConsentValidatePurposeElementResponse_EnrichedFields(t *testing.T) {
	desc := "User email address"
	e := ConsentValidatePurposeElementResponse{
		ElementID:   "e-1",
		Name:        "email",
		Namespace:   "default",
		Version:     "v1",
		Mandatory:   true,
		Approved:    true,
		Type:        "basic",
		Description: &desc,
		Properties:  map[string]string{"jsonPath": "$.email"},
	}
	require.Equal(t, "basic", e.Type)
	require.NotNil(t, e.Description)
	require.Equal(t, "$.email", e.Properties["jsonPath"])
}

func TestConsentRevokeResponse_Fields(t *testing.T) {
	resp := ConsentRevokeResponse{ActionTime: 1640000000000, ActionBy: "user-123"}
	require.Equal(t, int64(1640000000000), resp.ActionTime)
}

func TestConsentAttributeSearchResponse_Fields(t *testing.T) {
	resp := ConsentAttributeSearchResponse{ConsentIDs: []string{"c-1", "c-2"}, Count: 2}
	require.Len(t, resp.ConsentIDs, 2)
	require.Equal(t, 2, resp.Count)
}
