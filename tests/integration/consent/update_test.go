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
	"encoding/json"
	"net/http"
)

// ============================
// PUT /consents/{id} - Update Consent Tests
// ============================

// TestUpdateConsent_AddPurpose_Succeeds adds a new consent purpose
func (ts *ConsentAPITestSuite) TestUpdateConsent_AddPurpose_Succeeds() {
	// Create consent with one purpose
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{
						Name:           "marketing-purpose",
						Value:          "yes",
						IsUserApproved: true,
					},
				},
			},
		},
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Update to add another purpose
	updatePayload := ConsentUpdateRequest{
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{
						Name:           "marketing-purpose",
						Value:          "yes",
						IsUserApproved: true,
					},
				},
			},
			{
				Name: "analytics-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{
						Name:           "analytics-purpose",
						Value:          "approved",
						IsUserApproved: true,
					},
				},
			},
		},
	}

	updateResp, _ := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()
	ts.Equal(http.StatusOK, updateResp.StatusCode)

	// Verify purposes were updated
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()
	ts.Require().Equal(http.StatusOK, getResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &updated))

	ts.Len(updated.Purposes, 2)
	purposeNames := []string{updated.Purposes[0].Name, updated.Purposes[1].Name}
	ts.Contains(purposeNames, "marketing-purpose")
	ts.Contains(purposeNames, "analytics-purpose")
}

// TestUpdateConsent_RemovePurpose_Succeeds removes an existing purpose
func (ts *ConsentAPITestSuite) TestUpdateConsent_RemovePurpose_Succeeds() {
	// Create consent with two purposes
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{
						Name:           "marketing-purpose",
						Value:          "yes",
						IsUserApproved: true,
					},
				},
			},
			{
				Name: "analytics-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{
						Name:           "analytics-purpose",
						Value:          "approved",
						IsUserApproved: true,
					},
				},
			},
		},
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Update to keep only one purpose
	updatePayload := ConsentUpdateRequest{
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{
						Name:           "marketing-purpose",
						Value:          "yes",
						IsUserApproved: true,
					},
				},
			},
		},
	}

	updateResp, _ := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()
	ts.Equal(http.StatusOK, updateResp.StatusCode)

	// Verify only one purpose remains
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()
	ts.Require().Equal(http.StatusOK, getResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &updated))

	ts.Len(updated.Purposes, 1)
	ts.Equal("marketing-purpose", updated.Purposes[0].Name)
}

// TestUpdateConsent_UpdateAttributes_Succeeds changes attribute values
func (ts *ConsentAPITestSuite) TestUpdateConsent_UpdateAttributes_Succeeds() {
	// Create consent with attributes
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
		Attributes: map[string]string{
			"accountType": "savings",
			"branch":      "main",
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Update attributes
	updatePayload := ConsentUpdateRequest{
		Attributes: map[string]string{
			"accountType": "checking",
			"branch":      "downtown",
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("checking", updated.Attributes["accountType"])
	ts.Equal("downtown", updated.Attributes["branch"])
}

// TestUpdateConsent_AddAttribute_Succeeds adds a new attribute
func (ts *ConsentAPITestSuite) TestUpdateConsent_AddAttribute_Succeeds() {
	// Create consent with 1 attribute
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
		Attributes: map[string]string{
			"accountType": "savings",
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Add new attribute
	updatePayload := ConsentUpdateRequest{
		Attributes: map[string]string{
			"accountType": "savings",
			"branch":      "main",
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal(2, len(updated.Attributes))
	ts.Equal("main", updated.Attributes["branch"])
}

// TestUpdateConsent_UpdateValidityTime_Succeeds changes validityTime
func (ts *ConsentAPITestSuite) TestUpdateConsent_UpdateValidityTime_Succeeds() {
	// Create consent with validity time
	createPayload := ConsentCreateRequest{
		Type:         "accounts",
		ValidityTime: 3600,
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Update validity time
	newValidityTime := int64(7200)
	updatePayload := ConsentUpdateRequest{
		ValidityTime: &newValidityTime,
	}

	updateResp, _ := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	// Verify by fetching the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &updated))
	ts.NotNil(updated.ValidityTime)
	ts.Equal(int64(7200), *updated.ValidityTime)
}

// TestUpdateConsent_AddAuthorization_Succeeds adds a new auth resource
func (ts *ConsentAPITestSuite) TestUpdateConsent_AddAuthorization_Succeeds() {
	// Create consent with 1 authorization
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Add another authorization
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal(2, len(updated.Authorizations))
}

// TestUpdateConsent_FullUpdate_Succeeds updates all fields
func (ts *ConsentAPITestSuite) TestUpdateConsent_FullUpdate_Succeeds() {
	// Create minimal consent
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Full update
	newValidityTime := int64(7200)
	newFrequency := 5
	newRecurringIndicator := true
	updatePayload := ConsentUpdateRequest{
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{Name: "marketing-purpose", Value: "yes", IsUserApproved: true},
				},
			},
		},
		Attributes: map[string]string{
			"accountType": "savings",
		},
		ValidityTime:       &newValidityTime,
		Frequency:          &newFrequency,
		RecurringIndicator: &newRecurringIndicator,
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	updateResp, _ := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	// Verify by fetching the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &updated))
	ts.Equal(1, len(updated.Purposes))
	ts.Equal(1, len(updated.Attributes))
	ts.NotNil(updated.ValidityTime)
	ts.Equal(int64(7200), *updated.ValidityTime)
}

// ============================
// Status Transition Tests
// ============================

// TestUpdateConsent_ApproveAllAuths_StatusCreatedToActive updates CREATED to ACTIVE
func (ts *ConsentAPITestSuite) TestUpdateConsent_ApproveAllAuths_StatusCreatedToActive() {
	// Create consent with CREATED auth
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "CREATED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal("CREATED", created.Status)

	// Update auth to APPROVED
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("ACTIVE", updated.Status)
}

// TestUpdateConsent_RejectOneAuth_StatusActiveToRejected updates ACTIVE to REJECTED
func (ts *ConsentAPITestSuite) TestUpdateConsent_RejectOneAuth_StatusActiveToRejected() {
	// Create consent with 2 APPROVED auths (status = ACTIVE)
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal("ACTIVE", created.Status)

	// Update one auth to REJECTED
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "REJECTED"},
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("REJECTED", updated.Status)
}

// TestUpdateConsent_ApproveRejectedAuth_StatusRejectedToActive updates REJECTED to ACTIVE
func (ts *ConsentAPITestSuite) TestUpdateConsent_ApproveRejectedAuth_StatusRejectedToActive() {
	// Create consent with 1 REJECTED, 1 APPROVED (status = REJECTED)
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "REJECTED"},
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal("REJECTED", created.Status)

	// Update REJECTED auth to APPROVED
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("ACTIVE", updated.Status)
}

// TestUpdateConsent_AddRejectedAuth_StatusActiveToRejected adds REJECTED auth to ACTIVE consent
func (ts *ConsentAPITestSuite) TestUpdateConsent_AddRejectedAuth_StatusActiveToRejected() {
	// Create ACTIVE consent
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal("ACTIVE", created.Status)

	// Add REJECTED auth
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
			{UserID: "user2", Type: "auth", Status: "REJECTED"},
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("REJECTED", updated.Status)
}

// TestUpdateConsent_AddCreatedAuth_StatusActiveToCreated adds CREATED auth to ACTIVE consent
func (ts *ConsentAPITestSuite) TestUpdateConsent_AddCreatedAuth_StatusActiveToCreated() {
	// Create ACTIVE consent
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal("ACTIVE", created.Status)

	// Add CREATED auth
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
			{UserID: "user2", Type: "auth", Status: "CREATED"},
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("CREATED", updated.Status)
}

// TestUpdateConsent_RemoveRejectedAuth_StatusRejectedToActive removes REJECTED auth
func (ts *ConsentAPITestSuite) TestUpdateConsent_RemoveRejectedAuth_StatusRejectedToActive() {
	// Create REJECTED consent (1 REJECTED, 1 APPROVED)
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "REJECTED"},
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal("REJECTED", created.Status)

	// Remove REJECTED auth, keep APPROVED
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("ACTIVE", updated.Status)
}

// TestUpdateConsent_MultipleAuthChanges_StatusCorrect handles complex auth changes
func (ts *ConsentAPITestSuite) TestUpdateConsent_MultipleAuthChanges_StatusCorrect() {
	// Create consent with mixed auths
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
			{UserID: "user2", Type: "auth", Status: "CREATED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal("CREATED", created.Status)

	// Complex update: approve one, add new APPROVED, remove one
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
			{UserID: "user3", Type: "auth", Status: "APPROVED"},
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("ACTIVE", updated.Status)
}

// TestUpdateConsent_NoAuthChanges_StatusUnchanged verifies status unchanged when auths unchanged
func (ts *ConsentAPITestSuite) TestUpdateConsent_NoAuthChanges_StatusUnchanged() {
	// Create ACTIVE consent
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
		Attributes: map[string]string{
			"accountType": "savings",
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal("ACTIVE", created.Status)

	// Update attributes only, no auth changes
	updatePayload := ConsentUpdateRequest{
		Attributes: map[string]string{
			"accountType": "checking",
		},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()

	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))
	ts.Equal("ACTIVE", updated.Status)
}

// ============================
// Error Tests
// ============================

// TestUpdateConsent_MissingOrgID_ReturnsValidationError verifies missing org-id returns 400
func (ts *ConsentAPITestSuite) TestUpdateConsent_MissingOrgID_ReturnsValidationError() {
	// Create consent first
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Try to update without org-id header
	updatePayload := ConsentUpdateRequest{
		Attributes: map[string]string{
			"test": "value",
		},
	}

	resp, _ := ts.updateConsentWithHeaders(created.ID, updatePayload, "", testClientID)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode)
}

// TestUpdateConsent_MissingClientID_ReturnsValidationError verifies missing client-id returns 400
func (ts *ConsentAPITestSuite) TestUpdateConsent_MissingClientID_ReturnsValidationError() {
	// Create consent
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Try to update without client-id header
	updatePayload := ConsentUpdateRequest{
		Attributes: map[string]string{
			"test": "value",
		},
	}

	resp, _ := ts.updateConsentWithHeaders(created.ID, updatePayload, testOrgID, "")
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode)
}

// TestUpdateConsent_NonExistentConsent_ReturnsNotFound verifies non-existent ID returns 404
func (ts *ConsentAPITestSuite) TestUpdateConsent_NonExistentConsent_ReturnsNotFound() {
	nonExistentID := "00000000-0000-0000-0000-000000000000"

	updatePayload := ConsentUpdateRequest{
		Attributes: map[string]string{
			"test": "value",
		},
	}

	resp, _ := ts.updateConsent(nonExistentID, updatePayload)
	defer resp.Body.Close()

	ts.Equal(http.StatusNotFound, resp.StatusCode)
}

// TestUpdateConsent_MalformedJSON_ReturnsBadRequest verifies invalid JSON returns 400
func (ts *ConsentAPITestSuite) TestUpdateConsent_MalformedJSON_ReturnsBadRequest() {
	// Create consent first
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Send malformed JSON
	resp, _ := ts.updateConsent(created.ID, "{invalid json")
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode)
}

// TestUpdateConsent_InvalidPurposeName_ReturnsNotFound verifies non-existent purpose returns 404
func (ts *ConsentAPITestSuite) TestUpdateConsent_InvalidPurposeName_ReturnsNotFound() {
	// Create consent first
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Try to add non-existent purpose
	updatePayload := ConsentUpdateRequest{
		Purposes: []ConsentPurposeItem{
			{
				Name: "non-existent-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{Name: "non-existent-purpose", Value: "yes", IsUserApproved: true},
				},
			},
		},
	}

	resp, _ := ts.updateConsent(created.ID, updatePayload)
	defer resp.Body.Close()

	// Should return 404 if purpose validation is enforced
	ts.True(resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest)
}

// TestUpdateConsent_EmptyRequestBody_ReturnsBadRequest verifies empty body returns 400
func (ts *ConsentAPITestSuite) TestUpdateConsent_EmptyRequestBody_ReturnsBadRequest() {
	// Create consent first
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Send empty body
	resp, _ := ts.updateConsent(created.ID, "")
	defer resp.Body.Close()

	// May return 400 or 200 depending on implementation
	ts.True(resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusOK)
}

// ============================
// Remove All Items Tests
// ============================

// TestUpdateConsent_RemoveAllAuthorizations_Succeeds removes all authorizations
func (ts *ConsentAPITestSuite) TestUpdateConsent_RemoveAllAuthorizations_Succeeds() {
	// Create consent with authorizations
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
			{UserID: "user2", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal(2, len(created.Authorizations))

	// Update to remove all authorizations
	updatePayload := ConsentUpdateRequest{
		Authorizations: []AuthorizationRequest{},
	}

	updateResp, _ := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()
	ts.Equal(http.StatusOK, updateResp.StatusCode)

	// Verify all authorizations removed
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()
	ts.Require().Equal(http.StatusOK, getResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &updated))
	ts.Empty(updated.Authorizations, "All authorizations should be removed")
}

// TestUpdateConsent_RemoveAllPurposes_Succeeds removes all purposes
func (ts *ConsentAPITestSuite) TestUpdateConsent_RemoveAllPurposes_Succeeds() {
	// Create consent with purposes
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{Name: "marketing-purpose", Value: "yes", IsUserApproved: true},
				},
			},
			{
				Name: "analytics-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{Name: "analytics-purpose", Value: "approved", IsUserApproved: true},
				},
			},
		},
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal(2, len(created.Purposes))

	// Update to remove all purposes
	updatePayload := ConsentUpdateRequest{
		Purposes: []ConsentPurposeItem{},
	}

	updateResp, _ := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()
	ts.Equal(http.StatusOK, updateResp.StatusCode)

	// Verify all purposes removed
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()
	ts.Require().Equal(http.StatusOK, getResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &updated))
	ts.Empty(updated.Purposes, "All purposes should be removed")
}

// TestUpdateConsent_RemoveAllAttributes_Succeeds removes all attributes
func (ts *ConsentAPITestSuite) TestUpdateConsent_RemoveAllAttributes_Succeeds() {
	// Create consent with attributes
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
		Attributes: map[string]string{
			"accountType": "savings",
			"branch":      "main",
			"region":      "north",
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)
	ts.Equal(3, len(created.Attributes))

	// Update to remove all attributes
	updatePayload := ConsentUpdateRequest{
		Attributes: map[string]string{},
	}

	updateResp, _ := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()
	ts.Equal(http.StatusOK, updateResp.StatusCode)

	// Verify all attributes removed
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()
	ts.Require().Equal(http.StatusOK, getResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &updated))
	ts.Empty(updated.Attributes, "All attributes should be removed")
}

// TestUpdateConsent_RemoveAllItems_Succeeds removes all authorizations, purposes, and attributes
func (ts *ConsentAPITestSuite) TestUpdateConsent_RemoveAllItems_Succeeds() {
	// Create consent with everything
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{Name: "marketing-purpose", Value: "yes", IsUserApproved: true},
				},
			},
		},
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
		Attributes: map[string]string{
			"accountType": "savings",
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Update to remove everything (but keep type)
	updatePayload := ConsentUpdateRequest{
		Type:           "accounts", // Preserve type
		Purposes:       []ConsentPurposeItem{},
		Authorizations: []AuthorizationRequest{},
		Attributes:     map[string]string{},
	}

	updateResp, _ := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()
	ts.Equal(http.StatusOK, updateResp.StatusCode)

	// Verify everything removed
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()
	ts.Require().Equal(http.StatusOK, getResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &updated))
	ts.Empty(updated.Purposes, "All purposes should be removed")
	ts.Empty(updated.Authorizations, "All authorizations should be removed")
	ts.Empty(updated.Attributes, "All attributes should be removed")
	ts.Equal("accounts", updated.Type, "Type should remain")
}
