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
// PUT /consents/{id}/revoke - Revoke Consent Tests
// ============================

// TestRevokeConsent_ActiveConsent_Succeeds revokes an ACTIVE consent
func (ts *ConsentAPITestSuite) TestRevokeConsent_ActiveConsent_Succeeds() {
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

	// Revoke the consent
	revokeResp, _ := ts.revokeConsent(created.ID, "Customer requested revocation")
	defer revokeResp.Body.Close()

	ts.Equal(http.StatusOK, revokeResp.StatusCode)

	// Verify by fetching the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	var revoked ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &revoked))
	ts.Equal("REVOKED", revoked.Status)
}

// TestRevokeConsent_WithReason_Succeeds revokes with a reason
func (ts *ConsentAPITestSuite) TestRevokeConsent_WithReason_Succeeds() {
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

	// Revoke with reason
	reason := "User requested account closure"
	revokeResp, _ := ts.revokeConsent(created.ID, reason)
	defer revokeResp.Body.Close()

	ts.Equal(http.StatusOK, revokeResp.StatusCode)

	// Verify by fetching the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	var revoked ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &revoked))
	ts.Equal("REVOKED", revoked.Status)
}

// TestRevokeConsent_WithoutReason_Succeeds revokes without a reason
func (ts *ConsentAPITestSuite) TestRevokeConsent_WithoutReason_Succeeds() {
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

	// Revoke without reason (empty string)
	revokeResp, _ := ts.revokeConsent(created.ID, "")
	defer revokeResp.Body.Close()

	ts.Equal(http.StatusOK, revokeResp.StatusCode)

	// Verify by fetching the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	var revoked ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &revoked))
	ts.Equal("REVOKED", revoked.Status)
}

// TestRevokeConsent_VerifyStatusChange_Succeeds verifies status changes to REVOKED
func (ts *ConsentAPITestSuite) TestRevokeConsent_VerifyStatusChange_Succeeds() {
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
	ts.Equal("ACTIVE", created.Status)

	// Revoke
	revokeResp, _ := ts.revokeConsent(created.ID, "Test revocation")
	defer revokeResp.Body.Close()
	ts.Equal(http.StatusOK, revokeResp.StatusCode)

	// Verify by getting the consent again
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()
	ts.Equal(http.StatusOK, getResp.StatusCode)

	var fetched ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &fetched))
	ts.Equal("REVOKED", fetched.Status)
}

// ============================
// Error Tests
// ============================

// TestRevokeConsent_MissingOrgID_ReturnsValidationError verifies missing org-id returns 400
func (ts *ConsentAPITestSuite) TestRevokeConsent_MissingOrgID_ReturnsValidationError() {
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

	// Try to revoke without org-id header
	resp, _ := ts.revokeConsentWithHeaders(created.ID, "Test revocation", "", testClientID)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode)
}

// TestRevokeConsent_NonExistentConsent_ReturnsNotFound verifies non-existent ID returns 404
func (ts *ConsentAPITestSuite) TestRevokeConsent_NonExistentConsent_ReturnsNotFound() {
	nonExistentID := "00000000-0000-0000-0000-000000000000"

	resp, _ := ts.revokeConsent(nonExistentID, "Test revocation")
	defer resp.Body.Close()

	ts.Equal(http.StatusNotFound, resp.StatusCode)
}

// TestRevokeConsent_AlreadyRevoked_ReturnsConflict verifies already revoked returns 409
func (ts *ConsentAPITestSuite) TestRevokeConsent_AlreadyRevoked_ReturnsConflict() {
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

	// Revoke once
	revokeResp1, _ := ts.revokeConsent(created.ID, "First revocation")
	defer revokeResp1.Body.Close()
	ts.Equal(http.StatusOK, revokeResp1.StatusCode)

	// Try to revoke again - should return 409 Conflict
	revokeResp2, _ := ts.revokeConsent(created.ID, "Second revocation")
	defer revokeResp2.Body.Close()

	ts.Equal(http.StatusConflict, revokeResp2.StatusCode)
}

// TestRevokeConsent_InvalidConsentID_ReturnsBadRequest verifies invalid UUID returns 400
func (ts *ConsentAPITestSuite) TestRevokeConsent_InvalidConsentID_ReturnsBadRequest() {
	invalidID := "not-a-uuid"

	resp, _ := ts.revokeConsent(invalidID, "Test revocation")
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode)
}
