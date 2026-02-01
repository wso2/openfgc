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

package consentpurpose

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stretchr/testify/require"
)

// =========================================
// GET /consent-purposes Tests (List)
// =========================================

// TestListPurposes_NoFilters_ReturnsAllPurposes tests listing all consent purposes without filters
func (ts *PurposeAPITestSuite) TestListPurposes_NoFilters_ReturnsAllPurposes() {
	t := ts.T()

	// Create multiple purposes
	purpose1Payload := PurposeCreateRequest{
		Name:        "test_marketing_purpose",
		Description: "Marketing communications purpose",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: false},
		},
	}

	purpose2Payload := PurposeCreateRequest{
		Name:        "test_analytics_purpose",
		Description: "Analytics purpose",
		Elements: []PurposeElement{
			{Name: "test_analytics", IsMandatory: true},
		},
	}

	// Create first purpose
	resp, body := ts.createPurpose(purpose1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose1: %s", body)
	var purpose1Resp PurposeResponse
	err := json.Unmarshal(body, &purpose1Resp)
	require.NoError(t, err)
	ts.trackPurpose(purpose1Resp.ID)

	// Create second purpose
	resp, body = ts.createPurpose(purpose2Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose2: %s", body)
	var purpose2Resp PurposeResponse
	err = json.Unmarshal(body, &purpose2Resp)
	require.NoError(t, err)
	ts.trackPurpose(purpose2Resp.ID)

	// List all purposes
	resp, body = ts.listPurposes("", nil, nil, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list purposes: %s", body)

	var listResp PurposeListResponse
	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should have at least our 2 purposes
	require.GreaterOrEqual(t, len(listResp.Data), 2, "Expected at least 2 purposes")
	require.GreaterOrEqual(t, listResp.Metadata.Total, 2, "Total should be at least 2")

	// Find our purposes in the list
	foundPurpose1 := false
	foundPurpose2 := false
	for _, purpose := range listResp.Data {
		if purpose.ID == purpose1Resp.ID {
			foundPurpose1 = true
			require.Equal(t, "test_marketing_purpose", purpose.Name)
			require.Len(t, purpose.Elements, 2)
		}
		if purpose.ID == purpose2Resp.ID {
			foundPurpose2 = true
			require.Equal(t, "test_analytics_purpose", purpose.Name)
			require.Len(t, purpose.Elements, 1)
		}
	}
	require.True(t, foundPurpose1, "Purpose 1 not found in list")
	require.True(t, foundPurpose2, "Purpose 2 not found in list")
}

// TestListPurposes_FilterByName_ReturnsMatchingPurpose tests filtering by exact name
func (ts *PurposeAPITestSuite) TestListPurposes_FilterByName_ReturnsMatchingPurpose() {
	t := ts.T()

	// Create purposes with unique names
	purpose1Payload := PurposeCreateRequest{
		Name:        "test_filter_by_name_1",
		Description: "First purpose",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	purpose2Payload := PurposeCreateRequest{
		Name:        "test_filter_by_name_2",
		Description: "Second purpose",
		Elements: []PurposeElement{
			{Name: "test_phone", IsMandatory: true},
		},
	}

	// Create purposes
	resp, body := ts.createPurpose(purpose1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose1Resp PurposeResponse
	json.Unmarshal(body, &purpose1Resp)
	ts.trackPurpose(purpose1Resp.ID)

	resp, body = ts.createPurpose(purpose2Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose2Resp PurposeResponse
	json.Unmarshal(body, &purpose2Resp)
	ts.trackPurpose(purpose2Resp.ID)

	// Filter by first purpose name
	resp, body = ts.listPurposes("test_filter_by_name_1", nil, nil, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list purposes: %s", body)

	var listResp PurposeListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should only return the matching purpose
	require.GreaterOrEqual(t, len(listResp.Data), 1, "Expected at least 1 purpose")

	// Find our purpose in the results
	found := false
	for _, g := range listResp.Data {
		if g.ID == purpose1Resp.ID && g.Name == "test_filter_by_name_1" {
			found = true
			break
		}
	}
	require.True(t, found, "Should find the created purpose with matching name")
}

// TestListPurposes_FilterByClientID_ReturnsMatchingPurposes tests filtering by clientID
func (ts *PurposeAPITestSuite) TestListPurposes_FilterByClientID_ReturnsMatchingPurposes() {
	t := ts.T()

	// Create purpose with default client
	purpose1Payload := PurposeCreateRequest{
		Name:        "test_client_filter_default",
		Description: "Purpose for default client",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createPurpose(purpose1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose1Resp PurposeResponse
	json.Unmarshal(body, &purpose1Resp)
	ts.trackPurpose(purpose1Resp.ID)

	// List purposes filtered by our test client ID
	resp, body = ts.listPurposes("", []string{testClientID}, nil, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list purposes: %s", body)

	var listResp PurposeListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should have at least our purpose
	require.GreaterOrEqual(t, len(listResp.Data), 1, "Expected at least 1 purpose")

	// All returned purposes should have our client ID
	foundOurPurpose := false
	for _, purpose := range listResp.Data {
		require.Equal(t, testClientID, purpose.ClientID, "All purposes should have the filtered client ID")
		if purpose.ID == purpose1Resp.ID {
			foundOurPurpose = true
		}
	}
	require.True(t, foundOurPurpose, "Our purpose not found in filtered results")
}

// TestListPurposes_FilterBySingleElementName_ReturnsPurposesContainingElement tests filtering by single purpose
func (ts *PurposeAPITestSuite) TestListPurposes_FilterBySingleElementName_ReturnsPurposesContainingElement() {
	t := ts.T()

	// Create purposes with different purposes
	purpose1Payload := PurposeCreateRequest{
		Name:        "test_single_purpose_filter_1",
		Description: "Purpose with email",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: false},
		},
	}

	purpose2Payload := PurposeCreateRequest{
		Name:        "test_single_purpose_filter_2",
		Description: "Purpose with email and address",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_address", IsMandatory: false},
		},
	}

	purpose3Payload := PurposeCreateRequest{
		Name:        "test_single_purpose_filter_3",
		Description: "Purpose without email",
		Elements: []PurposeElement{
			{Name: "test_analytics", IsMandatory: true},
		},
	}

	// Create purposes
	resp, body := ts.createPurpose(purpose1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose1Resp PurposeResponse
	json.Unmarshal(body, &purpose1Resp)
	ts.trackPurpose(purpose1Resp.ID)

	resp, body = ts.createPurpose(purpose2Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose2Resp PurposeResponse
	json.Unmarshal(body, &purpose2Resp)
	ts.trackPurpose(purpose2Resp.ID)

	resp, body = ts.createPurpose(purpose3Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose3Resp PurposeResponse
	json.Unmarshal(body, &purpose3Resp)
	ts.trackPurpose(purpose3Resp.ID)

	// Filter by "test_email" - should return purpose1 and purpose2 only
	resp, body = ts.listPurposes("", nil, []string{"test_email"}, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list purposes: %s", body)

	var listResp PurposeListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should have at least 2 purposes (purpose1 and purpose2)
	require.GreaterOrEqual(t, len(listResp.Data), 2, "Expected at least 2 purposes with test_email")

	// Verify our purposes are in the list and purpose3 is not
	foundPurpose1 := false
	foundPurpose2 := false
	foundPurpose3 := false
	for _, purpose := range listResp.Data {
		if purpose.ID == purpose1Resp.ID {
			foundPurpose1 = true
		}
		if purpose.ID == purpose2Resp.ID {
			foundPurpose2 = true
		}
		if purpose.ID == purpose3Resp.ID {
			foundPurpose3 = true
		}
	}
	require.True(t, foundPurpose1, "Purpose 1 should be in results")
	require.True(t, foundPurpose2, "Purpose 2 should be in results")
	require.False(t, foundPurpose3, "Purpose 3 should NOT be in results")
}

// TestListPurposes_FilterByMultipleElementNames_ANDLogic_ReturnsOnlyPurposesWithAllElements tests AND logic
func (ts *PurposeAPITestSuite) TestListPurposes_FilterByMultipleElementNames_ANDLogic_ReturnsOnlyPurposesWithAllElements() {
	t := ts.T()

	// Create purposes with different purpose combinations
	purpose1Payload := PurposeCreateRequest{
		Name:        "test_and_logic_both",
		Description: "Purpose with BOTH email AND phone",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: true},
			{Name: "test_address", IsMandatory: false},
		},
	}

	purpose2Payload := PurposeCreateRequest{
		Name:        "test_and_logic_email_only",
		Description: "Purpose with ONLY email",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_marketing", IsMandatory: false},
		},
	}

	purpose3Payload := PurposeCreateRequest{
		Name:        "test_and_logic_phone_only",
		Description: "Purpose with ONLY phone",
		Elements: []PurposeElement{
			{Name: "test_phone", IsMandatory: true},
		},
	}

	purpose4Payload := PurposeCreateRequest{
		Name:        "test_and_logic_neither",
		Description: "Purpose with NEITHER email NOR phone",
		Elements: []PurposeElement{
			{Name: "test_analytics", IsMandatory: true},
		},
	}

	// Create purposes
	resp, body := ts.createPurpose(purpose1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose1Resp PurposeResponse
	json.Unmarshal(body, &purpose1Resp)
	ts.trackPurpose(purpose1Resp.ID)

	resp, body = ts.createPurpose(purpose2Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose2Resp PurposeResponse
	json.Unmarshal(body, &purpose2Resp)
	ts.trackPurpose(purpose2Resp.ID)

	resp, body = ts.createPurpose(purpose3Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose3Resp PurposeResponse
	json.Unmarshal(body, &purpose3Resp)
	ts.trackPurpose(purpose3Resp.ID)

	resp, body = ts.createPurpose(purpose4Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purpose4Resp PurposeResponse
	json.Unmarshal(body, &purpose4Resp)
	ts.trackPurpose(purpose4Resp.ID)

	// Filter by BOTH "test_email" AND "test_phone" - should return ONLY purpose1
	resp, body = ts.listPurposes("", nil, []string{"test_email", "test_phone"}, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list purposes: %s", body)

	var listResp PurposeListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should have at least 1 purpose (purpose1)
	require.GreaterOrEqual(t, len(listResp.Data), 1, "Expected at least 1 purpose with both elements")

	// Verify only purpose1 is in the list (has both purposes)
	foundPurpose1 := false
	foundPurpose2 := false
	foundPurpose3 := false
	foundPurpose4 := false
	for _, purpose := range listResp.Data {
		if purpose.ID == purpose1Resp.ID {
			foundPurpose1 = true
			// Verify it has both purposes
			hasEmail := false
			hasPhone := false
			for _, p := range purpose.Elements {
				if p.Name == "test_email" {
					hasEmail = true
				}
				if p.Name == "test_phone" {
					hasPhone = true
				}
			}
			require.True(t, hasEmail, "Purpose should have test_email")
			require.True(t, hasPhone, "Purpose should have test_phone")
		}
		if purpose.ID == purpose2Resp.ID {
			foundPurpose2 = true
		}
		if purpose.ID == purpose3Resp.ID {
			foundPurpose3 = true
		}
		if purpose.ID == purpose4Resp.ID {
			foundPurpose4 = true
		}
	}
	require.True(t, foundPurpose1, "Purpose 1 (has both) should be in results")
	require.False(t, foundPurpose2, "Purpose 2 (only email) should NOT be in results")
	require.False(t, foundPurpose3, "Purpose 3 (only phone) should NOT be in results")
	require.False(t, foundPurpose4, "Purpose 4 (neither) should NOT be in results")
}

// TestListPurposes_CombineAllFilters_ReturnsCorrectlyFilteredPurposes tests all 3 filters together
func (ts *PurposeAPITestSuite) TestListPurposes_CombineAllFilters_ReturnsCorrectlyFilteredPurposes() {
	t := ts.T()

	// Create a purpose that matches all filters
	matchingPurposePayload := PurposeCreateRequest{
		Name:        "test_all_filters_match",
		Description: "Purpose that matches all filters",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: true},
		},
	}

	// Create a purpose with same purposes but different name
	differentNamePayload := PurposeCreateRequest{
		Name:        "test_all_filters_different_name",
		Description: "Different name",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: true},
		},
	}

	// Create matching purpose
	resp, body := ts.createPurpose(matchingPurposePayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var matchingResp PurposeResponse
	json.Unmarshal(body, &matchingResp)
	ts.trackPurpose(matchingResp.ID)

	// Create non-matching purpose
	resp, body = ts.createPurpose(differentNamePayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var differentNameResp PurposeResponse
	json.Unmarshal(body, &differentNameResp)
	ts.trackPurpose(differentNameResp.ID)

	// Filter by name + clientID + names (all 3 filters)
	resp, body = ts.listPurposes(
		"test_all_filters_match",
		[]string{testClientID},
		[]string{"test_email", "test_phone"},
		100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list purposes: %s", body)

	var listResp PurposeListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should return exactly 1 purpose - the matching one
	require.Equal(t, 1, listResp.Metadata.Total, "Expected exactly 1 matching purpose")
	require.Len(t, listResp.Data, 1, "Expected exactly 1 purpose in data")
	require.Equal(t, matchingResp.ID, listResp.Data[0].ID)
	require.Equal(t, "test_all_filters_match", listResp.Data[0].Name)
	require.Equal(t, testClientID, listResp.Data[0].ClientID)
}

// TestListPurposes_Pagination_ReturnsCorrectSubset tests pagination parameters
func (ts *PurposeAPITestSuite) TestListPurposes_Pagination_ReturnsCorrectSubset() {
	t := ts.T()

	// Create multiple purposes for pagination testing
	for i := 1; i <= 5; i++ {
		payload := PurposeCreateRequest{
			Name:        fmt.Sprintf("test_pagination_purpose_%d", i),
			Description: fmt.Sprintf("Pagination test purpose %d", i),
			Elements: []PurposeElement{
				{Name: "test_email", IsMandatory: true},
			},
		}

		resp, body := ts.createPurpose(payload)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var purposeResp PurposeResponse
		json.Unmarshal(body, &purposeResp)
		ts.trackPurpose(purposeResp.ID)
	}

	// Test first page (limit 2, offset 0)
	resp, body := ts.listPurposes("", nil, nil, 2, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp PurposeListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	require.Len(t, listResp.Data, 2, "First page should have 2 purposes")
	require.Equal(t, 0, listResp.Metadata.Offset)
	require.Equal(t, 2, listResp.Metadata.Limit)

	// Test second page (limit 2, offset 2)
	resp, body = ts.listPurposes("", nil, nil, 2, 2)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	require.LessOrEqual(t, len(listResp.Data), 2, "Second page should have at most 2 purposes")
	require.Equal(t, 2, listResp.Metadata.Offset)
	require.Equal(t, 2, listResp.Metadata.Limit)
}

// TestListPurposes_EmptyResult_ReturnsEmptyList tests when no purposes match filters
func (ts *PurposeAPITestSuite) TestListPurposes_EmptyResult_ReturnsEmptyList() {
	t := ts.T()

	// Filter by non-existent name
	resp, body := ts.listPurposes("non_existent_purpose_name_xyz", nil, nil, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return OK even with no results: %s", body)

	var listResp PurposeListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	require.Equal(t, 0, listResp.Metadata.Total, "Total should be 0")
	require.Len(t, listResp.Data, 0, "Data should be empty")
}
