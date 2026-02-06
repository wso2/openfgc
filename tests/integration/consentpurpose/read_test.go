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
	"strings"

	"github.com/stretchr/testify/require"
)

// ================================================
// GET /consent-purposes/{purposeId} Tests
// ================================================

// TestGetPurpose_ExistingPurpose_Success tests retrieving an existing purpose
func (ts *PurposeAPITestSuite) TestGetPurpose_ExistingPurpose_Success() {
	t := ts.T()

	// Create a purpose first
	createPayload := PurposeCreateRequest{
		Name:        "test_get_existing",
		Description: "Purpose to retrieve",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: false},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Get the purpose
	resp, body := ts.getPurpose(createdPurpose.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get purpose: %s", body)

	var getResp PurposeResponse
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)

	// Verify all fields match
	require.Equal(t, createdPurpose.ID, getResp.ID)
	require.Equal(t, "test_get_existing", getResp.Name)
	require.NotNil(t, getResp.Description)
	require.Equal(t, "Purpose to retrieve", *getResp.Description)
	require.Equal(t, testClientID, getResp.ClientID)
	require.Len(t, getResp.Elements, 2)
	require.Equal(t, createdPurpose.CreatedTime, getResp.CreatedTime)
	require.Equal(t, createdPurpose.UpdatedTime, getResp.UpdatedTime)

	// Verify purposes
	purposeMap := make(map[string]bool)
	for _, p := range getResp.Elements {
		purposeMap[p.Name] = p.IsMandatory
	}
	require.True(t, purposeMap["test_email"])
	require.False(t, purposeMap["test_phone"])
}

// TestGetPurpose_SinglePurpose_Success tests retrieving purpose with single purpose
func (ts *PurposeAPITestSuite) TestGetPurpose_SinglePurpose_Success() {
	t := ts.T()

	createPayload := PurposeCreateRequest{
		Name:        "test_get_single_purpose",
		Description: "Single element purpose",
		Elements: []PurposeElement{
			{Name: "test_analytics", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Get and verify
	resp, body := ts.getPurpose(createdPurpose.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp PurposeResponse
	json.Unmarshal(body, &getResp)

	require.Len(t, getResp.Elements, 1)
	require.Equal(t, "test_analytics", getResp.Elements[0].Name)
	require.True(t, getResp.Elements[0].IsMandatory)
}

// TestGetPurpose_MultiplePurposes_Success tests retrieving purpose with multiple purposes
func (ts *PurposeAPITestSuite) TestGetPurpose_MultiplePurposes_Success() {
	t := ts.T()

	createPayload := PurposeCreateRequest{
		Name:        "test_get_multi_purposes",
		Description: "Multiple elements purpose",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: true},
			{Name: "test_address", IsMandatory: false},
			{Name: "test_marketing", IsMandatory: false},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Get and verify all purposes
	resp, body := ts.getPurpose(createdPurpose.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp PurposeResponse
	json.Unmarshal(body, &getResp)

	require.Len(t, getResp.Elements, 4)

	// Count mandatory vs optional
	mandatoryCount := 0
	optionalCount := 0
	for _, p := range getResp.Elements {
		if p.IsMandatory {
			mandatoryCount++
		} else {
			optionalCount++
		}
	}
	require.Equal(t, 2, mandatoryCount)
	require.Equal(t, 2, optionalCount)
}

// TestGetPurpose_NoDescription_Success tests retrieving purpose without description
func (ts *PurposeAPITestSuite) TestGetPurpose_NoDescription_Success() {
	t := ts.T()

	createPayload := PurposeCreateRequest{
		Name: "test_get_no_desc",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Get and verify
	resp, body := ts.getPurpose(createdPurpose.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp PurposeResponse
	json.Unmarshal(body, &getResp)

	require.Equal(t, "test_get_no_desc", getResp.Name)
	// Description can be nil or empty string
}

// TestGetPurpose_NonExistentID_ReturnsNotFound tests getting non-existent purpose
func (ts *PurposeAPITestSuite) TestGetPurpose_NonExistentID_ReturnsNotFound() {
	t := ts.T()

	resp, body := ts.getPurpose("PURPOSE-non-existent-id-xyz")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	// Message could be "not found" or "Resource Not Found"
	msg := strings.ToLower(errResp.Message)
	require.True(t, strings.Contains(msg, "not found"), "Expected 'not found' in message, got: %s", errResp.Message)
}

// TestGetPurpose_InvalidUUIDFormat_ReturnsBadRequest tests invalid ID format
func (ts *PurposeAPITestSuite) TestGetPurpose_InvalidUUIDFormat_ReturnsBadRequest() {
	t := ts.T()

	resp, body := ts.getPurpose("invalid-id-format")
	// Could be 400 or 404 depending on validation
	require.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode,
		"Should reject invalid ID format: %s", body)
}

// TestGetPurpose_MissingOrgIDHeader_Fails tests missing org-id header
func (ts *PurposeAPITestSuite) TestGetPurpose_MissingOrgIDHeader_Fails() {
	t := ts.T()

	// Create a purpose first
	createPayload := PurposeCreateRequest{
		Name: "test_get_missing_orgid",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Try to get without org-id header
	httpReq, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, createdPurpose.ID),
		nil)
	// Deliberately omit org-id header

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing org-id")
}

// TestGetPurpose_WrongOrgID_ReturnsNotFound tests accessing with wrong org
func (ts *PurposeAPITestSuite) TestGetPurpose_WrongOrgID_ReturnsNotFound() {
	t := ts.T()

	// Create a purpose with our test org
	createPayload := PurposeCreateRequest{
		Name: "test_get_wrong_org",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Try to get with different org-id
	httpReq, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, createdPurpose.ID),
		nil)
	httpReq.Header.Set("org-id", "different-org-id")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should not find purpose in different org")
}

// TestGetPurpose_AfterUpdate_ReturnsUpdatedData tests getting after update
func (ts *PurposeAPITestSuite) TestGetPurpose_AfterUpdate_ReturnsUpdatedData() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_get_after_update",
		Description: "Original description",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Update the purpose
	updatePayload := PurposeUpdateRequest{
		Name:        "test_get_after_update_modified",
		Description: "Updated description",
		Elements: []PurposeElement{
			{Name: "test_phone", IsMandatory: false},
		},
	}

	updateResp, _ := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	// Get and verify updated data
	resp, body := ts.getPurpose(createdPurpose.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp PurposeResponse
	json.Unmarshal(body, &getResp)

	require.Equal(t, "test_get_after_update_modified", getResp.Name)
	require.Equal(t, "Updated description", *getResp.Description)
	require.Len(t, getResp.Elements, 1)
	require.Equal(t, "test_phone", getResp.Elements[0].Name)
	require.False(t, getResp.Elements[0].IsMandatory)
	require.GreaterOrEqual(t, getResp.UpdatedTime, createdPurpose.UpdatedTime, "UpdatedTime should be equal or newer")
}
