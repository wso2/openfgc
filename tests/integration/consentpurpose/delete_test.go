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

// ===================================================
// DELETE /consent-purposes/{purposeId} Tests
// ===================================================

// TestDeletePurpose_ExistingPurpose_Success tests deleting an existing purpose
func (ts *PurposeAPITestSuite) TestDeletePurpose_ExistingPurpose_Success() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_delete_existing",
		Description: "To be deleted",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)

	// Delete the purpose
	resp, _ := ts.deletePurpose(createdPurpose.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "Delete should return 204")

	// Verify it's gone
	resp, _ = ts.getPurpose(createdPurpose.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Purpose should not exist after deletion")
}

// TestDeletePurpose_WithMultiplePurposes_Success tests deleting purpose with multiple purposes
func (ts *PurposeAPITestSuite) TestDeletePurpose_WithMultiplePurposes_Success() {
	t := ts.T()

	// Create purpose with multiple purposes
	createPayload := PurposeCreateRequest{
		Name:        "test_delete_multi_purposes",
		Description: "Multiple purposes to delete",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: true},
			{Name: "test_address", IsMandatory: false},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)

	// Delete the purpose
	resp, _ := ts.deletePurpose(createdPurpose.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify deletion
	resp, _ = ts.getPurpose(createdPurpose.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestDeletePurpose_NonExistentID_ReturnsNotFound tests deleting non-existent purpose
func (ts *PurposeAPITestSuite) TestDeletePurpose_NonExistentID_ReturnsNotFound() {
	t := ts.T()

	resp, body := ts.deletePurpose("PURPOSE-non-existent-id-xyz")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	// Message could be "not found" or "Resource Not Found"
	msg := strings.ToLower(errResp.Message)
	require.True(t, strings.Contains(msg, "not found"), "Expected 'not found' in message, got: %s", errResp.Message)
}

// TestDeletePurpose_InvalidUUIDFormat_ReturnsBadRequestOrNotFound tests invalid ID
func (ts *PurposeAPITestSuite) TestDeletePurpose_InvalidUUIDFormat_ReturnsBadRequestOrNotFound() {
	t := ts.T()

	resp, _ := ts.deletePurpose("invalid-id-format")
	// Could be 400 or 404 depending on validation
	require.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode)
}

// TestDeletePurpose_Twice_ReturnsNotFound tests double deletion
func (ts *PurposeAPITestSuite) TestDeletePurpose_Twice_ReturnsNotFound() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_delete_twice",
		Description: "Delete twice test",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)

	// First deletion
	resp, _ := ts.deletePurpose(createdPurpose.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Second deletion should fail
	resp, _ = ts.deletePurpose(createdPurpose.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Second delete should return 404")
}

// TestDeletePurpose_MissingOrgIDHeader_Fails tests missing org-id header
func (ts *PurposeAPITestSuite) TestDeletePurpose_MissingOrgIDHeader_Fails() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_delete_missing_orgid",
		Description: "Missing header test",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Try to delete without org-id header
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, createdPurpose.ID),
		nil)
	// Deliberately omit org-id header

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing org-id")
}

// TestDeletePurpose_WrongOrgID_ReturnsNotFound tests deleting with wrong org
func (ts *PurposeAPITestSuite) TestDeletePurpose_WrongOrgID_ReturnsNotFound() {
	t := ts.T()

	// Create purpose with our test org
	createPayload := PurposeCreateRequest{
		Name:        "test_delete_wrong_org",
		Description: "Wrong org test",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Try to delete with different org-id
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, createdPurpose.ID),
		nil)
	httpReq.Header.Set("org-id", "different-org-id")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should not find purpose in different org")
}

// TestDeletePurpose_PurposesRemainIntact_Success tests that purposes are not deleted
func (ts *PurposeAPITestSuite) TestDeletePurpose_PurposesRemainIntact_Success() {
	t := ts.T()

	// Create purpose using existing purposes
	createPayload := PurposeCreateRequest{
		Name:        "test_delete_purposes_intact",
		Description: "Purposes should remain",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: false},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)

	// Delete the purpose
	resp, _ := ts.deletePurpose(createdPurpose.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify purpose is gone
	resp, _ = ts.getPurpose(createdPurpose.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Verify purposes still exist by creating another purpose with same purposes
	createPayload2 := PurposeCreateRequest{
		Name:        "test_purposes_still_exist",
		Description: "Using same purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: true},
		},
	}

	createResp, body = ts.createPurpose(createPayload2)
	require.Equal(t, http.StatusCreated, createResp.StatusCode, "Should be able to reuse purposes: %s", body)
	var newPurpose PurposeResponse
	json.Unmarshal(body, &newPurpose)
	ts.trackPurpose(newPurpose.ID)
}

// TestDeletePurpose_CanRecreateWithSameName_Success tests recreation after deletion
func (ts *PurposeAPITestSuite) TestDeletePurpose_CanRecreateWithSameName_Success() {
	t := ts.T()

	purposeName := "test_delete_and_recreate"

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        purposeName,
		Description: "First version",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var firstPurpose PurposeResponse
	json.Unmarshal(body, &firstPurpose)

	// Delete it
	resp, _ := ts.deletePurpose(firstPurpose.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Recreate with same name
	createPayload.Description = "Second version"
	createResp, body = ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode, "Should allow recreation: %s", body)

	var secondPurpose PurposeResponse
	json.Unmarshal(body, &secondPurpose)
	ts.trackPurpose(secondPurpose.ID)

	require.Equal(t, purposeName, secondPurpose.Name)
	require.NotEqual(t, firstPurpose.ID, secondPurpose.ID, "Should have different ID")
}

// TestDeletePurpose_MultiplePurposes_OnlyDeletesOne tests selective deletion
func (ts *PurposeAPITestSuite) TestDeletePurpose_MultiplePurposes_OnlyDeletesOne() {
	t := ts.T()

	// Create first purpose
	createPayload1 := PurposeCreateRequest{
		Name:        "test_delete_selective_1",
		Description: "First purpose",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload1)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var purpose1 PurposeResponse
	json.Unmarshal(body, &purpose1)
	ts.trackPurpose(purpose1.ID)

	// Create second purpose
	createPayload2 := PurposeCreateRequest{
		Name:        "test_delete_selective_2",
		Description: "Second purpose",
		Elements: []PurposeElement{
			{Name: "test_phone", IsMandatory: true},
		},
	}

	createResp, body = ts.createPurpose(createPayload2)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var purpose2 PurposeResponse
	json.Unmarshal(body, &purpose2)
	ts.trackPurpose(purpose2.ID)

	// Delete first purpose only
	resp, _ := ts.deletePurpose(purpose1.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify first is gone
	resp, _ = ts.getPurpose(purpose1.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Verify second still exists
	resp, body = ts.getPurpose(purpose2.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Second purpose should still exist: %s", body)

	var stillExists PurposeResponse
	json.Unmarshal(body, &stillExists)
	require.Equal(t, purpose2.ID, stillExists.ID)
	require.Equal(t, "test_delete_selective_2", stillExists.Name)
}

// TestDeletePurpose_EmptyID_ReturnsBadRequest tests deletion with empty ID
func (ts *PurposeAPITestSuite) TestDeletePurpose_EmptyID_ReturnsBadRequest() {
	t := ts.T()

	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purposes/", testServerURL),
		nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 404 or 405 (Method Not Allowed) since the route doesn't match
	require.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, resp.StatusCode)
}
