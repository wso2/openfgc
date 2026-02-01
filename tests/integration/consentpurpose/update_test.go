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
	"fmt"
	"net/http"

	"github.com/stretchr/testify/require"
)

// ================================================
// PUT /consent-purposes/{purposeId} Tests
// ================================================

// TestUpdatePurpose_ChangeName_Success tests updating the purpose name
func (ts *PurposeAPITestSuite) TestUpdatePurpose_ChangeName_Success() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_update_name_original",
		Description: "Original name",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Update name
	updatePayload := PurposeUpdateRequest{
		Name:        "test_update_name_modified",
		Description: "Original name",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update purpose: %s", body)

	var updateResp PurposeResponse
	err := json.Unmarshal(body, &updateResp)
	require.NoError(t, err)

	require.Equal(t, "test_update_name_modified", updateResp.Name)
	require.GreaterOrEqual(t, updateResp.UpdatedTime, createdPurpose.UpdatedTime, "UpdatedTime should be equal or newer")
}

// TestUpdatePurpose_ChangeDescription_Success tests updating description
func (ts *PurposeAPITestSuite) TestUpdatePurpose_ChangeDescription_Success() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_update_description",
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

	// Update description
	updatePayload := PurposeUpdateRequest{
		Name:        "test_update_description",
		Description: "This is the new updated description",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeResponse
	json.Unmarshal(body, &updateResp)

	require.Equal(t, "This is the new updated description", *updateResp.Description)
}

// TestUpdatePurpose_AddPurpose_Success tests adding a purpose to the purpose
func (ts *PurposeAPITestSuite) TestUpdatePurpose_AddPurpose_Success() {
	t := ts.T()

	// Create purpose with single purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_add_purpose",
		Description: "Will add more purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Update to add more purposes
	updatePayload := PurposeUpdateRequest{
		Name:        "test_add_purpose",
		Description: "Will add more purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: false},
			{Name: "test_address", IsMandatory: false},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeResponse
	json.Unmarshal(body, &updateResp)

	require.Len(t, updateResp.Elements, 3)
}

// TestUpdatePurpose_RemovePurpose_Success tests removing a purpose from the purpose
func (ts *PurposeAPITestSuite) TestUpdatePurpose_RemovePurpose_Success() {
	t := ts.T()

	// Create purpose with multiple purposes
	createPayload := PurposeCreateRequest{
		Name:        "test_remove_purpose",
		Description: "Will remove purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: false},
			{Name: "test_address", IsMandatory: false},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Update to remove purposes
	updatePayload := PurposeUpdateRequest{
		Name:        "test_remove_purpose",
		Description: "Will remove purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeResponse
	json.Unmarshal(body, &updateResp)

	require.Len(t, updateResp.Elements, 1)
	require.Equal(t, "test_email", updateResp.Elements[0].Name)
}

// TestUpdatePurpose_ChangeMandatoryFlag_Success tests changing isMandatory flag
func (ts *PurposeAPITestSuite) TestUpdatePurpose_ChangeMandatoryFlag_Success() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_change_mandatory",
		Description: "Change mandatory flag",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Update to make it optional
	updatePayload := PurposeUpdateRequest{
		Name:        "test_change_mandatory",
		Description: "Change mandatory flag",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: false},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeResponse
	json.Unmarshal(body, &updateResp)

	require.False(t, updateResp.Elements[0].IsMandatory)
}

// TestUpdatePurpose_ReplaceAllPurposes_Success tests completely replacing purposes
func (ts *PurposeAPITestSuite) TestUpdatePurpose_ReplaceAllPurposes_Success() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_replace_purposes",
		Description: "Replace all purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_phone", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Update with completely different purposes
	updatePayload := PurposeUpdateRequest{
		Name:        "test_replace_purposes",
		Description: "Replace all purposes",
		Elements: []PurposeElement{
			{Name: "test_marketing", IsMandatory: false},
			{Name: "test_analytics", IsMandatory: false},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeResponse
	json.Unmarshal(body, &updateResp)

	require.Len(t, updateResp.Elements, 2)

	// Verify old purposes are gone, new ones present
	names := make([]string, 0)
	for _, p := range updateResp.Elements {
		names = append(names, p.Name)
	}
	require.Contains(t, names, "test_marketing")
	require.Contains(t, names, "test_analytics")
	require.NotContains(t, names, "test_email")
	require.NotContains(t, names, "test_phone")
}

// TestUpdatePurpose_UpdateAll_Success tests updating all fields at once
func (ts *PurposeAPITestSuite) TestUpdatePurpose_UpdateAll_Success() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_update_all_original",
		Description: "Original",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Update everything
	updatePayload := PurposeUpdateRequest{
		Name:        "test_update_all_modified",
		Description: "Completely changed",
		Elements: []PurposeElement{
			{Name: "test_phone", IsMandatory: false},
			{Name: "test_address", IsMandatory: true},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeResponse
	json.Unmarshal(body, &updateResp)

	require.Equal(t, "test_update_all_modified", updateResp.Name)
	require.Equal(t, "Completely changed", *updateResp.Description)
	require.Len(t, updateResp.Elements, 2)
}

// TestUpdatePurpose_NonExistentID_ReturnsNotFound tests updating non-existent purpose
func (ts *PurposeAPITestSuite) TestUpdatePurpose_NonExistentID_ReturnsNotFound() {
	t := ts.T()

	updatePayload := PurposeUpdateRequest{
		Name:        "test_nonexistent",
		Description: "Does not exist",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updatePurpose("PURPOSE-non-existent-id", updatePayload)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404: %s", body)
}

// TestUpdatePurpose_DuplicateName_DifferentPurpose_Fails tests duplicate name with another purpose
func (ts *PurposeAPITestSuite) TestUpdatePurpose_DuplicateName_DifferentPurpose_Fails() {
	t := ts.T()

	// Create first purpose
	createPayload1 := PurposeCreateRequest{
		Name:        "test_update_dup_purpose1",
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
		Name:        "test_update_dup_purpose2",
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

	// Try to update purpose2 with purpose1's name
	updatePayload := PurposeUpdateRequest{
		Name:        "test_update_dup_purpose1", // Same as purpose1
		Description: "Trying to duplicate",
		Elements: []PurposeElement{
			{Name: "test_phone", IsMandatory: true},
		},
	}

	resp, body := ts.updatePurpose(purpose2.ID, updatePayload)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "Should reject duplicate name: %s", body)
}

// TestUpdatePurpose_SameName_Success tests updating with same name (idempotent update)
func (ts *PurposeAPITestSuite) TestUpdatePurpose_SameName_Success() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_update_same_name",
		Description: "Original",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Update with same name but different description
	updatePayload := PurposeUpdateRequest{
		Name:        "test_update_same_name", // Same name
		Description: "Updated description",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Should allow update with same name: %s", body)
}

// TestUpdatePurpose_InvalidPurposeName_Fails tests updating with non-existent purpose
func (ts *PurposeAPITestSuite) TestUpdatePurpose_InvalidPurposeName_Fails() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_update_invalid_purpose",
		Description: "Valid purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Try to update with invalid purpose
	updatePayload := PurposeUpdateRequest{
		Name:        "test_update_invalid_purpose",
		Description: "Invalid purposes",
		Elements: []PurposeElement{
			{Name: "invalid_purpose_xyz", IsMandatory: true},
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject invalid purpose: %s", body)
}

// TestUpdatePurpose_EmptyPurposes_Fails tests updating with no purposes
func (ts *PurposeAPITestSuite) TestUpdatePurpose_EmptyPurposes_Fails() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_update_empty_purposes",
		Description: "Has purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Try to update with empty purposes
	updatePayload := PurposeUpdateRequest{
		Name:        "test_update_empty_purposes",
		Description: "No purposes",
		Elements:    []PurposeElement{},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject empty purposes: %s", body)
}

// TestUpdatePurpose_DuplicateElementInPurpose_Fails tests duplicate purpose in the update payload
func (ts *PurposeAPITestSuite) TestUpdatePurpose_DuplicateElementInPurpose_Fails() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name:        "test_update_dup_purpose",
		Description: "Valid",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	// Try to update with duplicate purposes
	updatePayload := PurposeUpdateRequest{
		Name:        "test_update_dup_purpose",
		Description: "Duplicate purposes",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
			{Name: "test_email", IsMandatory: false}, // Duplicate
		},
	}

	resp, body := ts.updatePurpose(createdPurpose.ID, updatePayload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject duplicate purposes: %s", body)
}

// TestUpdatePurpose_MissingRequiredHeaders_Fails tests missing headers
func (ts *PurposeAPITestSuite) TestUpdatePurpose_MissingRequiredHeaders_Fails() {
	t := ts.T()

	// Create purpose
	createPayload := PurposeCreateRequest{
		Name: "test_update_missing_headers",
		Elements: []PurposeElement{
			{Name: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdPurpose PurposeResponse
	json.Unmarshal(body, &createdPurpose)
	ts.trackPurpose(createdPurpose.ID)

	updatePayload := PurposeUpdateRequest{
		Name: "test_update_missing_headers_mod",
		Elements: []PurposeElement{
			{Name: "test_phone", IsMandatory: true},
		},
	}

	reqBody, _ := json.Marshal(updatePayload)

	// Try without org-id
	httpReq, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, createdPurpose.ID),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("TPP-client-id", testClientID)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(httpReq)
	resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing org-id")

	// Try without TPP-client-id
	httpReq, _ = http.NewRequest("PUT",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, createdPurpose.ID),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("org-id", testOrgID)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, _ = client.Do(httpReq)
	resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing TPP-client-id")
}
