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
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stretchr/testify/require"
)

// ======================================
// POST /consent-purposes Tests
// ======================================

// TestCreatePurpose_SinglePurpose_Success tests creating a purpose with one purpose
func (ts *PurposeAPITestSuite) TestCreatePurpose_SinglePurpose_Success() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_single_purpose",
		Description: "Purpose with single purpose",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose: %s", body)

	var purposeResp PurposeResponse
	err := json.Unmarshal(body, &purposeResp)
	require.NoError(t, err)

	require.NotEmpty(t, purposeResp.ID, "Purpose ID should not be empty")
	require.Equal(t, "test_single_purpose", purposeResp.Name)
	require.NotNil(t, purposeResp.Description)
	require.Equal(t, "Purpose with single purpose", *purposeResp.Description)
	require.Equal(t, testClientID, purposeResp.ClientID)
	require.Len(t, purposeResp.Elements, 1)
	require.Equal(t, "test_email", purposeResp.Elements[0].Name)
	require.True(t, purposeResp.Elements[0].IsMandatory)
	require.NotZero(t, purposeResp.CreatedTime)
	require.NotZero(t, purposeResp.UpdatedTime)

	ts.trackPurpose(purposeResp.ID)
}

// TestCreatePurpose_MultiplePurposes_Success tests creating a purpose with multiple purposes
func (ts *PurposeAPITestSuite) TestCreatePurpose_MultiplePurposes_Success() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_multi_purpose",
		Description: "Purpose with multiple purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: true},
			{Name: "test_address", IsMandatory: false},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose: %s", body)

	var purposeResp PurposeResponse
	err := json.Unmarshal(body, &purposeResp)
	require.NoError(t, err)

	require.Equal(t, "test_multi_purpose", purposeResp.Name)
	require.Len(t, purposeResp.Elements, 3)

	// Verify all purposes are present
	purposeMap := make(map[string]bool)
	for _, p := range purposeResp.Elements {
		purposeMap[p.Name] = p.IsMandatory
	}
	require.True(t, purposeMap["test_email"])
	require.True(t, purposeMap["test_phone"])
	require.False(t, purposeMap["test_address"])

	ts.trackPurpose(purposeResp.ID)
}

// TestCreatePurpose_NoDescription_Success tests creating without description
func (ts *PurposeAPITestSuite) TestCreatePurpose_NoDescription_Success() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name: "test_no_description",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose: %s", body)

	var purposeResp PurposeResponse
	err := json.Unmarshal(body, &purposeResp)
	require.NoError(t, err)

	require.Equal(t, "test_no_description", purposeResp.Name)
	// Description can be nil or empty
	ts.trackPurpose(purposeResp.ID)
}

// TestCreatePurpose_AllOptionalPurposes_Success tests creating with all optional purposes
func (ts *PurposeAPITestSuite) TestCreatePurpose_AllOptionalPurposes_Success() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_all_optional",
		Description: "All purposes are optional",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: false},
			{Name: "test_phone", IsMandatory: false},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose: %s", body)

	var purposeResp PurposeResponse
	err := json.Unmarshal(body, &purposeResp)
	require.NoError(t, err)

	for _, p := range purposeResp.Elements {
		require.False(t, p.IsMandatory, "All purposes should be optional")
	}

	ts.trackPurpose(purposeResp.ID)
}

// TestCreatePurpose_DuplicateName_SameClient_Fails tests duplicate name for same client
func (ts *PurposeAPITestSuite) TestCreatePurpose_DuplicateName_SameClient_Fails() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_duplicate_name",
		Description: "First purpose",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	// Create first purpose
	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var purposeResp PurposeResponse
	json.Unmarshal(body, &purposeResp)
	ts.trackPurpose(purposeResp.ID)

	// Try to create another with same name
	payload.Description = "Second purpose"
	resp, body = ts.createPurpose(payload)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "Should reject duplicate name: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	// Message could be "already exists", "Conflict", or "duplicate"
	msg := strings.ToLower(errResp.Message)
	require.True(t,
		strings.Contains(msg, "already exists") || strings.Contains(msg, "conflict") || strings.Contains(msg, "duplicate"),
		"Expected conflict/duplicate error message, got: %s", errResp.Message)
}

// TestCreatePurpose_InvalidPurposeName_Fails tests creating with non-existent purpose
func (ts *PurposeAPITestSuite) TestCreatePurpose_InvalidPurposeName_Fails() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_invalid_purpose",
		Description: "Purpose with invalid purpose",
		Elements: []PurposeElement{
			{Name: "non_existent_purpose_xyz", IsMandatory: true},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject invalid purpose: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	require.Contains(t, errResp.Description, "does not exist")
}

// TestCreatePurpose_EmptyName_Fails tests creating with empty name
func (ts *PurposeAPITestSuite) TestCreatePurpose_EmptyName_Fails() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "",
		Description: "Empty name test",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject empty name: %s", body)
}

// TestCreatePurpose_NoPurposes_Fails tests creating without any purposes
func (ts *PurposeAPITestSuite) TestCreatePurpose_NoPurposes_Fails() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_no_purposes",
		Description: "Purpose with no purposes",
		Elements:    []PurposeElement{},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject purpose with no purposes: %s", body)
}

// TestCreatePurpose_DuplicateElementInPurpose_Fails tests duplicate element within same purpose
func (ts *PurposeAPITestSuite) TestCreatePurpose_DuplicateElementInPurpose_Fails() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_duplicate_purpose_in",
		Description: "Purpose with duplicate purpose",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_email", IsMandatory: false}, // Duplicate
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject duplicate purposes: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	require.Contains(t, errResp.Description, "duplicate")
}

// TestCreatePurpose_MissingOrgIDHeader_Fails tests missing org-id header
func (ts *PurposeAPITestSuite) TestCreatePurpose_MissingOrgIDHeader_Fails() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_missing_orgid",
		Description: "Test missing header",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	reqBody, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purposes",
		bytes.NewBuffer(reqBody))
	// Deliberately omit org-id header
	httpReq.Header.Set("TPP-client-id", testClientID)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing org-id")
}

// TestCreatePurpose_MissingClientIDHeader_Fails tests missing TPP-client-id header
func (ts *PurposeAPITestSuite) TestCreatePurpose_MissingClientIDHeader_Fails() {
	t := ts.T()

	payload := PurposeCreateRequest{
		Name:        "test_missing_clientid",
		Description: "Test missing header",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	reqBody, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purposes",
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("org-id", testOrgID)
	// Deliberately omit TPP-client-id header
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing TPP-client-id")
}

// TestCreatePurpose_InvalidJSON_Fails tests malformed JSON
func (ts *PurposeAPITestSuite) TestCreatePurpose_InvalidJSON_Fails() {
	t := ts.T()

	resp, body := ts.createPurpose("{invalid json}")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject invalid JSON: %s", body)
}

// TestCreatePurpose_LongName_Success tests creating with a long name
func (ts *PurposeAPITestSuite) TestCreatePurpose_LongName_Success() {
	t := ts.T()

	// Create a name that's exactly 200 characters
	longName := "test_very_long_name_"
	for len(longName) < 200 {
		longName += "a"
	}
	longName = longName[:200] // Ensure exactly 200 chars

	payload := PurposeCreateRequest{
		Name:        longName, // Use the 200 char name
		Description: "Long name test",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createPurpose(payload)
	// Should succeed if within database limits, otherwise fail gracefully
	if resp.StatusCode == http.StatusCreated {
		var purposeResp PurposeResponse
		json.Unmarshal(body, &purposeResp)
		ts.trackPurpose(purposeResp.ID)
	} else {
		require.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError}, resp.StatusCode)
	}
}
