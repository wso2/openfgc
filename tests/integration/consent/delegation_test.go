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

// ========================================
// DELEGATION TESTS
// ========================================

// TestCreateConsent_WithDelegation_Succeeds verifies creating a delegated consent
func (ts *ConsentAPITestSuite) TestCreateConsent_WithDelegation_Succeeds() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Delegation: &DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-456",
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "mother-111",
				Status: "APPROVED",
			},
			{
				UserID: "father-222",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode, "Status should be 201")

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.NotEmpty(consentResp.ID, "Consent ID should be generated")
	ts.Equal("accounts", consentResp.Type)

	// Verify delegation in response
	ts.NotNil(consentResp.Delegation, "Delegation should be present in response")
	ts.Equal("parental_biological", consentResp.Delegation.Type)
	ts.Equal("ANY", consentResp.Delegation.RevocationPolicy)
	ts.Equal("child-456", consentResp.Delegation.OnBehalfOf)

	// Verify auth type inferred as delegate
	for _, auth := range consentResp.Authorizations {
		ts.Equal("delegate", auth.Type, "Auth type should be inferred as delegate")
	}

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithoutDelegation_NoDelegationInResponse verifies normal consent has no delegation
func (ts *ConsentAPITestSuite) TestCreateConsent_WithoutDelegation_NoDelegationInResponse() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{
				UserID: "user123",
				Type:   "authorisation",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode, "Status should be 201")

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.Nil(consentResp.Delegation, "Delegation should be nil for non-delegated consent")
	ts.Equal("authorisation", consentResp.Authorizations[0].Type, "Auth type should remain as provided")

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithDelegation_AuthTypeOverridden verifies auth type is overridden to delegate
func (ts *ConsentAPITestSuite) TestCreateConsent_WithDelegation_AuthTypeOverridden() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Delegation: &DelegationRequest{
			Type:             "registered_carer",
			RevocationPolicy: "BOTH",
			OnBehalfOf:       "patient-321",
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "carer-100",
				Type:   "authorisation",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode, "Status should be 201")

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	// Even though "authorisation" was sent, it should be overridden to "delegate"
	ts.Equal("delegate", consentResp.Authorizations[0].Type, "Auth type should be overridden to delegate")

	ts.trackConsent(consentResp.ID)
}

// TestGetConsent_WithDelegation_ReturnsDelegation verifies GET returns delegation data
func (ts *ConsentAPITestSuite) TestGetConsent_WithDelegation_ReturnsDelegation() {
	// Create delegated consent first
	payload := ConsentCreateRequest{
		Type: "accounts",
		Delegation: &DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-789",
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "parent-111",
				Status: "APPROVED",
			},
		},
	}

	createResp, createBody := ts.createConsent(payload)
	defer createResp.Body.Close()
	ts.Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// GET the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()
	ts.Equal(http.StatusOK, getResp.StatusCode)

	var fetched ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &fetched))

	// Verify delegation in GET response
	ts.NotNil(fetched.Delegation, "Delegation should be present in GET response")
	ts.Equal("parental_biological", fetched.Delegation.Type)
	ts.Equal("ANY", fetched.Delegation.RevocationPolicy)
	ts.Equal("child-789", fetched.Delegation.OnBehalfOf)
}

// TestUpdateConsent_WithDelegation_DelegationUnchanged verifies update does not affect delegation
func (ts *ConsentAPITestSuite) TestUpdateConsent_WithDelegation_DelegationUnchanged() {
	// Create delegated consent
	payload := ConsentCreateRequest{
		Type: "accounts",
		Delegation: &DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-update-test",
		},
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{Name: "marketing-purpose", IsUserApproved: true},
				},
			},
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "parent-update",
				Status: "APPROVED",
			},
		},
	}

	createResp, createBody := ts.createConsent(payload)
	defer createResp.Body.Close()
	ts.Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Update the consent
	updatePayload := ConsentUpdateRequest{
		Purposes: []ConsentPurposeItem{
			{
				Name: "marketing-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{Name: "marketing-purpose", IsUserApproved: false},
				},
			},
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "parent-update",
				Type:   "delegate",
				Status: "APPROVED",
			},
		},
		Attributes: map[string]string{},
	}

	updateResp, updateBody := ts.updateConsent(created.ID, updatePayload)
	defer updateResp.Body.Close()
	ts.Equal(http.StatusOK, updateResp.StatusCode)

	var updated ConsentResponse
	ts.NoError(json.Unmarshal(updateBody, &updated))

	// Delegation should still be present and unchanged
	ts.NotNil(updated.Delegation, "Delegation should still be present after update")
	ts.Equal("parental_biological", updated.Delegation.Type)
	ts.Equal("child-update-test", updated.Delegation.OnBehalfOf)
}

// TestRevokeConsent_WithDelegation_Succeeds verifies revoking a delegated consent works
func (ts *ConsentAPITestSuite) TestRevokeConsent_WithDelegation_Succeeds() {
	// Create delegated consent
	payload := ConsentCreateRequest{
		Type: "accounts",
		Delegation: &DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-revoke-test",
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "parent-revoke",
				Status: "APPROVED",
			},
		},
	}

	createResp, createBody := ts.createConsent(payload)
	defer createResp.Body.Close()
	ts.Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Revoke the consent
	revokeResp, _ := ts.revokeConsent(created.ID, "Parent withdrew consent")
	defer revokeResp.Body.Close()
	ts.Equal(http.StatusOK, revokeResp.StatusCode)

	// Verify consent is revoked
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	var fetched ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &fetched))
	ts.Equal("REVOKED", fetched.Status, "Consent should be revoked")
	ts.NotNil(fetched.Delegation, "Delegation should still be present after revoke")
}

// TestListConsents_WithDelegationFilter_ReturnsOnlyDelegated verifies delegation=true filter
func (ts *ConsentAPITestSuite) TestListConsents_WithDelegationFilter_ReturnsOnlyDelegated() {
	// Create a delegated consent
	delegatedPayload := ConsentCreateRequest{
		Type: "accounts",
		Delegation: &DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-filter-test",
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "parent-filter",
				Status: "APPROVED",
			},
		},
	}

	createResp, createBody := ts.createConsent(delegatedPayload)
	defer createResp.Body.Close()
	ts.Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Search with delegation=true
	listResp, listBody := ts.listConsents(map[string]string{
		"delegation": "true",
	})
	defer listResp.Body.Close()
	ts.Equal(http.StatusOK, listResp.StatusCode)

	var listResult ConsentListResponse
	ts.NoError(json.Unmarshal(listBody, &listResult))

	// All returned consents should have delegation
	for _, consent := range listResult.Data {
		ts.NotNil(consent.Delegation, "All consents with delegation=true filter should have delegation")
	}
}

// TestListConsents_DelegationFilterWithUserIds_ReturnsFiltered verifies delegation=true&userIds filter
func (ts *ConsentAPITestSuite) TestListConsents_DelegationFilterWithUserIds_ReturnsFiltered() {
	uniqueChild := "child-userid-filter-test"

	// Create delegated consent for specific child
	payload := ConsentCreateRequest{
		Type: "accounts",
		Delegation: &DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       uniqueChild,
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "parent-userid-filter",
				Status: "APPROVED",
			},
		},
	}

	createResp, createBody := ts.createConsent(payload)
	defer createResp.Body.Close()
	ts.Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Search with delegation=true&userIds=child
	listResp, listBody := ts.listConsents(map[string]string{
		"delegation": "true",
		"userIds":    uniqueChild,
	})
	defer listResp.Body.Close()
	ts.Equal(http.StatusOK, listResp.StatusCode)

	var listResult ConsentListResponse
	ts.NoError(json.Unmarshal(listBody, &listResult))

	// All returned consents should be for the specific child
	for _, consent := range listResult.Data {
		ts.NotNil(consent.Delegation)
		ts.Equal(uniqueChild, consent.Delegation.OnBehalfOf, "Should only return consents for the specified child")
	}
}

// TestCreateConsent_WithDelegation_DifferentTypes verifies different delegation types work
func (ts *ConsentAPITestSuite) TestCreateConsent_WithDelegation_DifferentTypes() {
	types := []struct {
		delegationType   string
		revocationPolicy string
		onBehalfOf       string
	}{
		{"registered_carer", "BOTH", "patient-001"},
		{"power_of_attorney", "BOTH", "client-001"},
		{"court_appointed", "SUBJECT_ONLY", "ward-001"},
		{"emergency_proxy", "BOTH", "patient-002"},
		{"spousal", "BOTH", "patient-003"},
	}

	for _, tt := range types {
		payload := ConsentCreateRequest{
			Type: "accounts",
			Delegation: &DelegationRequest{
				Type:             tt.delegationType,
				RevocationPolicy: tt.revocationPolicy,
				OnBehalfOf:       tt.onBehalfOf,
			},
			Authorizations: []AuthorizationRequest{
				{
					UserID: "delegate-user",
					Status: "APPROVED",
				},
			},
		}

		resp, body := ts.createConsent(payload)
		defer resp.Body.Close()

		ts.Equal(http.StatusCreated, resp.StatusCode, "Should create consent with delegation type: "+tt.delegationType)

		var consentResp ConsentResponse
		ts.NoError(json.Unmarshal(body, &consentResp))

		ts.Equal(tt.delegationType, consentResp.Delegation.Type)
		ts.Equal(tt.revocationPolicy, consentResp.Delegation.RevocationPolicy)
		ts.Equal(tt.onBehalfOf, consentResp.Delegation.OnBehalfOf)

		ts.trackConsent(consentResp.ID)
	}
}
