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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wso2/openfgc/internal/consent/model"
)

// =============================================================================
// getString
// =============================================================================

func TestGetString(t *testing.T) {
	cases := []struct {
		name     string
		row      map[string]interface{}
		key      string
		expected string
	}{
		{"string value", map[string]interface{}{"k": "hello"}, "k", "hello"},
		{"byte slice value", map[string]interface{}{"k": []byte("world")}, "k", "world"},
		{"missing key returns empty", map[string]interface{}{"other": "v"}, "k", ""},
		{"integer value returns empty", map[string]interface{}{"k": 42}, "k", ""},
		{"nil value returns empty", map[string]interface{}{"k": nil}, "k", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, getString(tc.row, tc.key))
		})
	}
}

// =============================================================================
// getStringPtr
// =============================================================================

func TestGetStringPtr(t *testing.T) {
	cases := []struct {
		name        string
		row         map[string]interface{}
		key         string
		expectNil   bool
		expectValue string
	}{
		{"string value returns pointer", map[string]interface{}{"k": "val"}, "k", false, "val"},
		{"byte slice value returns pointer", map[string]interface{}{"k": []byte("bytes")}, "k", false, "bytes"},
		{"missing key returns nil", map[string]interface{}{}, "k", true, ""},
		{"nil value returns nil", map[string]interface{}{"k": nil}, "k", true, ""},
		{"integer value returns nil", map[string]interface{}{"k": 99}, "k", true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := getStringPtr(tc.row, tc.key)
			if tc.expectNil {
				require.Nil(t, p)
			} else {
				require.NotNil(t, p)
				require.Equal(t, tc.expectValue, *p)
			}
		})
	}
}

// =============================================================================
// getInt64
// =============================================================================

func TestGetInt64(t *testing.T) {
	cases := []struct {
		name     string
		row      map[string]interface{}
		key      string
		expected int64
	}{
		{"int64 value", map[string]interface{}{"k": int64(42)}, "k", 42},
		{"int32 value", map[string]interface{}{"k": int32(10)}, "k", 10},
		{"int value", map[string]interface{}{"k": int(7)}, "k", 7},
		{"float64 value", map[string]interface{}{"k": float64(3.9)}, "k", 3},
		{"uint8 slice parseable", map[string]interface{}{"k": []uint8("1234567890")}, "k", 1234567890},
		{"string parseable", map[string]interface{}{"k": "9876543210"}, "k", 9876543210},
		{"string not parseable returns 0", map[string]interface{}{"k": "not-int"}, "k", 0},
		{"missing key returns 0", map[string]interface{}{}, "k", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, getInt64(tc.row, tc.key))
		})
	}
}

// =============================================================================
// getInt64Ptr
// =============================================================================

func TestGetInt64Ptr(t *testing.T) {
	cases := []struct {
		name        string
		row         map[string]interface{}
		key         string
		expectNil   bool
		expectValue int64
	}{
		{"int64 value returns pointer", map[string]interface{}{"k": int64(100)}, "k", false, 100},
		{"byte slice parseable returns pointer", map[string]interface{}{"k": []byte("200")}, "k", false, 200},
		{"nil value returns nil", map[string]interface{}{"k": nil}, "k", true, 0},
		{"missing key returns nil", map[string]interface{}{}, "k", true, 0},
		{"empty byte slice returns nil", map[string]interface{}{"k": []byte("")}, "k", true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := getInt64Ptr(tc.row, tc.key)
			if tc.expectNil {
				require.Nil(t, p)
			} else {
				require.NotNil(t, p)
				require.Equal(t, tc.expectValue, *p)
			}
		})
	}
}

// =============================================================================
// getInt
// =============================================================================

func TestGetInt(t *testing.T) {
	cases := []struct {
		name     string
		row      map[string]interface{}
		key      string
		expected int
	}{
		{"int64 value", map[string]interface{}{"k": int64(3)}, "k", 3},
		{"uint32 value", map[string]interface{}{"k": uint32(7)}, "k", 7},
		{"int32 value", map[string]interface{}{"k": int32(5)}, "k", 5},
		{"string numeric value", map[string]interface{}{"k": "42"}, "k", 42},
		{"string with spaces", map[string]interface{}{"k": " 10 "}, "k", 10},
		{"string empty returns 0", map[string]interface{}{"k": ""}, "k", 0},
		{"string non-numeric returns 0", map[string]interface{}{"k": "nope"}, "k", 0},
		{"[]byte numeric value", map[string]interface{}{"k": []byte("99")}, "k", 99},
		{"[]byte with spaces", map[string]interface{}{"k": []byte(" 7 ")}, "k", 7},
		{"[]byte empty returns 0", map[string]interface{}{"k": []byte("")}, "k", 0},
		{"[]byte non-numeric returns 0", map[string]interface{}{"k": []byte("bad")}, "k", 0},
		{"nil value returns 0", map[string]interface{}{"k": nil}, "k", 0},
		{"missing key returns 0", map[string]interface{}{}, "k", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, getInt(tc.row, tc.key))
		})
	}
}

// =============================================================================
// getIntPtr
// =============================================================================

func TestGetIntPtr(t *testing.T) {
	cases := []struct {
		name        string
		row         map[string]interface{}
		key         string
		expectNil   bool
		expectValue int
	}{
		{"int64 value returns pointer", map[string]interface{}{"k": int64(42)}, "k", false, 42},
		{"byte slice parseable returns pointer", map[string]interface{}{"k": []byte("99")}, "k", false, 99},
		{"nil value returns nil", map[string]interface{}{"k": nil}, "k", true, 0},
		{"missing key returns nil", map[string]interface{}{}, "k", true, 0},
		{"empty byte slice returns nil", map[string]interface{}{"k": []byte("")}, "k", true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := getIntPtr(tc.row, tc.key)
			if tc.expectNil {
				require.Nil(t, p)
			} else {
				require.NotNil(t, p)
				require.Equal(t, tc.expectValue, *p)
			}
		})
	}
}

// =============================================================================
// getBool
// =============================================================================

func TestGetBool(t *testing.T) {
	cases := []struct {
		name     string
		row      map[string]interface{}
		key      string
		expected bool
	}{
		{"bool true", map[string]interface{}{"k": true}, "k", true},
		{"bool false", map[string]interface{}{"k": false}, "k", false},
		{"int64 non-zero is true", map[string]interface{}{"k": int64(1)}, "k", true},
		{"int64 zero is false", map[string]interface{}{"k": int64(0)}, "k", false},
		{"uint8 non-zero is true", map[string]interface{}{"k": uint8(1)}, "k", true},
		{"uint8 zero is false", map[string]interface{}{"k": uint8(0)}, "k", false},
		{"int32 non-zero is true", map[string]interface{}{"k": int32(1)}, "k", true},
		{"int32 zero is false", map[string]interface{}{"k": int32(0)}, "k", false},
		{"missing key returns false", map[string]interface{}{}, "k", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, getBool(tc.row, tc.key))
		})
	}
}

// =============================================================================
// getBoolPtr
// =============================================================================

func TestGetBoolPtr(t *testing.T) {
	cases := []struct {
		name        string
		row         map[string]interface{}
		key         string
		expectNil   bool
		expectValue bool
	}{
		{"bool true returns pointer", map[string]interface{}{"k": true}, "k", false, true},
		{"bool false returns pointer", map[string]interface{}{"k": false}, "k", false, false},
		{"int64 non-zero returns true pointer", map[string]interface{}{"k": int64(5)}, "k", false, true},
		{"int64 zero returns false pointer", map[string]interface{}{"k": int64(0)}, "k", false, false},
		{"uint8 non-zero returns true pointer", map[string]interface{}{"k": uint8(1)}, "k", false, true},
		{"uint8 zero returns false pointer", map[string]interface{}{"k": uint8(0)}, "k", false, false},
		{"nil value returns nil", map[string]interface{}{"k": nil}, "k", true, false},
		{"missing key returns nil", map[string]interface{}{}, "k", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := getBoolPtr(tc.row, tc.key)
			if tc.expectNil {
				require.Nil(t, p)
			} else {
				require.NotNil(t, p)
				require.Equal(t, tc.expectValue, *p)
			}
		})
	}
}

// =============================================================================
// mapToConsent
// =============================================================================

func TestMapToConsent_NilRow(t *testing.T) {
	require.Nil(t, mapToConsent(nil))
}

func TestMapToConsent_FullRow(t *testing.T) {
	freq := int64(5)
	exp := int64(1700000000000)
	dur := int64(86400000)

	row := map[string]interface{}{
		"consent_id":                    "cid-1",
		"created_time":                  int64(1000000000000),
		"updated_time":                  int64(1000000001000),
		"group_id":                      "grp-abc",
		"consent_type":                  "regulatory",
		"current_status":                "ACTIVE",
		"consent_frequency":             freq,
		"expiration_time":               exp,
		"recurring_indicator":           int64(1),
		"data_access_validity_duration": dur,
		"org_id":                        "org-xyz",
	}

	c := mapToConsent(row)
	require.NotNil(t, c)
	require.Equal(t, "cid-1", c.ConsentID)
	require.Equal(t, int64(1000000000000), c.CreatedTime)
	require.Equal(t, int64(1000000001000), c.UpdatedTime)
	require.Equal(t, "grp-abc", c.GroupID)
	require.Equal(t, "regulatory", c.ConsentType)
	require.Equal(t, "ACTIVE", c.CurrentStatus)

	require.NotNil(t, c.ConsentFrequency)
	require.Equal(t, 5, *c.ConsentFrequency)

	require.NotNil(t, c.ExpirationTime)
	require.Equal(t, int64(1700000000000), *c.ExpirationTime)

	require.NotNil(t, c.RecurringIndicator)
	require.True(t, *c.RecurringIndicator)

	require.NotNil(t, c.DataAccessValidityDuration)
	require.Equal(t, int64(86400000), *c.DataAccessValidityDuration)

	require.Equal(t, "org-xyz", c.OrgID)
}

func TestMapToConsent_NullableFieldsAreNil(t *testing.T) {
	row := map[string]interface{}{
		"consent_id":                    "cid-2",
		"created_time":                  int64(0),
		"updated_time":                  int64(0),
		"group_id":                      "grp-1",
		"consent_type":                  "informational",
		"current_status":                "REVOKED",
		"consent_frequency":             nil,
		"expiration_time":               nil,
		"recurring_indicator":           nil,
		"data_access_validity_duration": nil,
		"org_id":                        "org-1",
	}
	c := mapToConsent(row)
	require.NotNil(t, c)
	require.Nil(t, c.ConsentFrequency)
	require.Nil(t, c.ExpirationTime)
	require.Nil(t, c.RecurringIndicator)
	require.Nil(t, c.DataAccessValidityDuration)
}

// =============================================================================
// mapToConsentAttribute
// =============================================================================

func TestMapToConsentAttribute_NilRow(t *testing.T) {
	require.Nil(t, mapToConsentAttribute(nil))
}

func TestMapToConsentAttribute_FullRow(t *testing.T) {
	row := map[string]interface{}{
		"consent_id": "cid-1",
		"att_key":    "customer_type",
		"att_value":  "premium",
		"org_id":     "org-1",
	}
	attr := mapToConsentAttribute(row)
	require.NotNil(t, attr)
	require.Equal(t, "cid-1", attr.ConsentID)
	require.Equal(t, "customer_type", attr.AttKey)
	require.Equal(t, "premium", attr.AttValue)
	require.Equal(t, "org-1", attr.OrgID)
}

// =============================================================================
// mapToConsentPurposeRow
// =============================================================================

func TestMapToConsentPurposeRow_FullRow(t *testing.T) {
	dispName := "Marketing Consent"
	desc := "For marketing communications"

	row := map[string]interface{}{
		"consent_id":         "cid-1",
		"purpose_version_id": "pvid-1",
		"purpose_id":         "pid-1",
		"purpose_name":       "Marketing",
		"purpose_group_id":   "pgrp-1",
		"purpose_version":    int64(3),
		"display_name":       dispName,
		"description":        desc,
		"org_id":             "org-1",
	}

	pr := mapToConsentPurposeRow(row)
	require.Equal(t, "cid-1", pr.ConsentID)
	require.Equal(t, "pvid-1", pr.PurposeVersionID)
	require.Equal(t, "pid-1", pr.PurposeID)
	require.Equal(t, "Marketing", pr.PurposeName)
	require.Equal(t, "pgrp-1", pr.PurposeGroupID)
	require.Equal(t, 3, pr.PurposeVersion)
	require.NotNil(t, pr.DisplayName)
	require.Equal(t, dispName, *pr.DisplayName)
	require.NotNil(t, pr.Description)
	require.Equal(t, desc, *pr.Description)
	require.Equal(t, "org-1", pr.OrgID)
}

func TestMapToConsentPurposeRow_NilOptionalFields(t *testing.T) {
	row := map[string]interface{}{
		"consent_id":         "cid-2",
		"purpose_version_id": "pvid-2",
		"purpose_id":         "pid-2",
		"purpose_name":       "Analytics",
		"purpose_group_id":   "pgrp-2",
		"purpose_version":    int64(1),
		"display_name":       nil,
		"description":        nil,
		"org_id":             "org-2",
	}

	pr := mapToConsentPurposeRow(row)
	require.Nil(t, pr.DisplayName)
	require.Nil(t, pr.Description)
}

// =============================================================================
// mapToConsentApprovalRow
// =============================================================================

func TestMapToConsentApprovalRow_FullRow(t *testing.T) {
	dispName := "User Email Address"
	elemDesc := "The primary email of the user"
	value := "user@example.com"

	row := map[string]interface{}{
		"consent_id":           "cid-1",
		"purpose_version_id":   "pvid-1",
		"element_version_id":   "evid-1",
		"element_id":           "eid-1",
		"element_name":         "user_email",
		"element_namespace":    "identity",
		"element_version":      int64(2),
		"element_type":         "basic",
		"element_display_name": dispName,
		"element_description":  elemDesc,
		"mandatory":            true,
		"approved":             true,
		"value":                value,
		"org_id":               "org-1",
	}

	ar := mapToConsentApprovalRow(row)
	require.Equal(t, "cid-1", ar.ConsentID)
	require.Equal(t, "pvid-1", ar.PurposeVersionID)
	require.Equal(t, "evid-1", ar.ElementVersionID)
	require.Equal(t, "eid-1", ar.ElementID)
	require.Equal(t, "user_email", ar.ElementName)
	require.Equal(t, "identity", ar.ElementNamespace)
	require.Equal(t, 2, ar.ElementVersionNum)
	require.Equal(t, "basic", ar.ElementType)
	require.NotNil(t, ar.ElementDisplayName)
	require.Equal(t, dispName, *ar.ElementDisplayName)
	require.NotNil(t, ar.ElementDescription)
	require.Equal(t, elemDesc, *ar.ElementDescription)
	require.True(t, ar.Mandatory)
	require.True(t, ar.Approved)
	require.NotNil(t, ar.Value)
	require.Equal(t, value, *ar.Value)
	require.Equal(t, "org-1", ar.OrgID)
}

func TestMapToConsentApprovalRow_NullableFieldsAreNil(t *testing.T) {
	row := map[string]interface{}{
		"consent_id":           "cid-2",
		"purpose_version_id":   "pvid-2",
		"element_version_id":   "evid-2",
		"element_id":           "eid-2",
		"element_name":         "user_phone",
		"element_namespace":    "identity",
		"element_version":      int64(1),
		"element_type":         "basic",
		"element_display_name": nil,
		"element_description":  nil,
		"mandatory":            false,
		"approved":             false,
		"value":                nil,
		"org_id":               "org-2",
	}

	ar := mapToConsentApprovalRow(row)
	require.Nil(t, ar.ElementDisplayName)
	require.Nil(t, ar.ElementDescription)
	require.False(t, ar.Mandatory)
	require.False(t, ar.Approved)
	require.Nil(t, ar.Value)
}

func TestBuildConsentSearchOrderBy_Default(t *testing.T) {
	orderBy := buildConsentSearchOrderBy(nil)
	require.Equal(t, "CONSENT.CREATED_TIME DESC, CONSENT.CONSENT_ID ASC", orderBy)
}

func TestBuildConsentSearchOrderBy_CustomMultiField(t *testing.T) {
	orderBy := buildConsentSearchOrderBy([]model.ConsentSort{
		{
			Field:     model.ConsentSortFieldStatus,
			Direction: model.ConsentSortDirectionAsc,
		},
		{
			Field:     model.ConsentSortFieldValidityTime,
			Direction: model.ConsentSortDirectionAsc,
		},
		{
			Field:     model.ConsentSortFieldCreatedTime,
			Direction: model.ConsentSortDirectionDesc,
		},
	})

	require.Equal(t,
		"CONSENT.CURRENT_STATUS ASC, CASE WHEN CONSENT.EXPIRATION_TIME IS NULL THEN 1 ELSE 0 END ASC, CONSENT.EXPIRATION_TIME ASC, CONSENT.CREATED_TIME DESC, CONSENT.CONSENT_ID ASC",
		orderBy)
}

func TestBuildConsentSearchOrderBy_ValidityTimeDescTreatsNoExpiryAsLatest(t *testing.T) {
	orderBy := buildConsentSearchOrderBy([]model.ConsentSort{
		{
			Field:     model.ConsentSortFieldValidityTime,
			Direction: model.ConsentSortDirectionDesc,
		},
	})

	require.Equal(t,
		"CASE WHEN CONSENT.EXPIRATION_TIME IS NULL THEN 0 ELSE 1 END ASC, CONSENT.EXPIRATION_TIME DESC, CONSENT.CONSENT_ID ASC",
		orderBy)
}

func TestBuildConsentSearchOrderBy_InvalidValuesFallbackSafely(t *testing.T) {
	orderBy := buildConsentSearchOrderBy([]model.ConsentSort{
		{
			Field:     model.ConsentSortField("unsupported"),
			Direction: model.ConsentSortDirectionAsc,
		},
		{
			Field:     model.ConsentSortFieldCreatedTime,
			Direction: model.ConsentSortDirection("SIDEWAYS"),
		},
	})

	require.Equal(t, "CONSENT.CREATED_TIME DESC, CONSENT.CONSENT_ID ASC", orderBy)
}
