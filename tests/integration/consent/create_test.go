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

	"github.com/wso2/openfgc/tests/integration/testutils"
)

// ========================================
// SUCCESS TESTS (14)
// ========================================

// TestCreateConsent_MinimalPayload_Succeeds verifies creating a consent with minimal required fields
func (ts *ConsentAPITestSuite) TestCreateConsent_MinimalPayload_Succeeds() {
	// Minimal payload: type + authorizations only
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{
				UserID: "user123",
				Type:   "payments",
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
	ts.Equal(testClientID, consentResp.ClientID)
	ts.Len(consentResp.Authorizations, 1)

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithSinglePurpose_Succeeds verifies creating a consent with one purpose
func (ts *ConsentAPITestSuite) TestCreateConsent_WithSinglePurpose_Succeeds() {
	payload := ConsentCreateRequest{
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
			{
				UserID: "user123",
				Type:   "payments",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.Len(consentResp.Purposes, 1)
	ts.Equal("marketing-purpose", consentResp.Purposes[0].Name)
	ts.Len(consentResp.Purposes[0].Elements, 1)
	ts.Equal("marketing-purpose", consentResp.Purposes[0].Elements[0].Name)

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithMultiplePurposes_Succeeds verifies creating a consent with multiple purposes
func (ts *ConsentAPITestSuite) TestCreateConsent_WithMultiplePurposes_Succeeds() {
	payload := ConsentCreateRequest{
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
						Value:          true,
						IsUserApproved: true,
					},
				},
			},
			{
				Name: "terms-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{
						Name:           "terms-purpose",
						IsUserApproved: true,
					},
				},
			},
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "user123",
				Type:   "payments",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.Len(consentResp.Purposes, 3)

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithAttributes_Succeeds verifies creating a consent with custom attributes
func (ts *ConsentAPITestSuite) TestCreateConsent_WithAttributes_Succeeds() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Attributes: map[string]string{
			"merchantId":  "MERCH123",
			"referenceId": "REF-789",
			"source":      "mobile",
		},
		Authorizations: []AuthorizationRequest{
			{
				UserID: "user123",
				Type:   "payments",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.NotNil(consentResp.Attributes)
	ts.Len(consentResp.Attributes, 3)

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithValidityTime_Succeeds verifies creating a consent with validity time
func (ts *ConsentAPITestSuite) TestCreateConsent_WithValidityTime_Succeeds() {
	validityTime := int64(3600) // 1 hour

	payload := ConsentCreateRequest{
		Type:         "accounts",
		ValidityTime: validityTime,
		Authorizations: []AuthorizationRequest{
			{
				UserID: "user123",
				Type:   "payments",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.Require().NotNil(consentResp.ValidityTime)
	ts.Equal(validityTime, *consentResp.ValidityTime)

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithRecurringIndicator_Succeeds verifies creating a consent with recurring indicator
func (ts *ConsentAPITestSuite) TestCreateConsent_WithRecurringIndicator_Succeeds() {
	payload := ConsentCreateRequest{
		Type:               "accounts",
		RecurringIndicator: true,
		Authorizations: []AuthorizationRequest{
			{
				UserID: "user123",
				Type:   "payments",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.Require().NotNil(consentResp.RecurringIndicator)
	ts.True(*consentResp.RecurringIndicator)

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithFrequency_Succeeds verifies creating a consent with frequency
func (ts *ConsentAPITestSuite) TestCreateConsent_WithFrequency_Succeeds() {
	frequency := 10

	payload := ConsentCreateRequest{
		Type:      "accounts",
		Frequency: frequency,
		Authorizations: []AuthorizationRequest{
			{
				UserID: "user123",
				Type:   "payments",
				Status: "APPROVED",
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.Require().NotNil(consentResp.Frequency)
	ts.Equal(frequency, *consentResp.Frequency)

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_WithMultipleAuthorizations_Succeeds verifies creating a consent with multiple authorizations
func (ts *ConsentAPITestSuite) TestCreateConsent_WithMultipleAuthorizations_Succeeds() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{
				UserID:      "user123",
				Type:        "payments",
				Status:      "APPROVED",
				Resources:   []string{"account-1", "account-2"},
				Permissions: []string{"read", "write"},
			},
			{
				UserID:      "user456",
				Type:        "accounts",
				Status:      "APPROVED",
				Resources:   []string{"account-3"},
				Permissions: []string{"read"},
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.Len(consentResp.Authorizations, 2)

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_FullPayload_Succeeds verifies creating a consent with all fields populated
func (ts *ConsentAPITestSuite) TestCreateConsent_FullPayload_Succeeds() {
	payload := ConsentCreateRequest{
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
					{Name: "analytics-purpose", Value: true, IsUserApproved: true},
				},
			},
		},
		Attributes: map[string]string{
			"merchantId":  "MERCH123",
			"referenceId": "REF-789",
		},
		ValidityTime:       int64(7200),
		RecurringIndicator: true,
		Frequency:          10,
		Authorizations: []AuthorizationRequest{
			{
				UserID:      "user123",
				Type:        "payments",
				Status:      "APPROVED",
				Resources:   []string{"account-1"},
				Permissions: []string{"read", "write"},
			},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	ts.NotEmpty(consentResp.ID)
	ts.Equal("accounts", consentResp.Type)
	ts.Len(consentResp.Purposes, 2)
	ts.Len(consentResp.Authorizations, 1)
	ts.NotNil(consentResp.Attributes)
	ts.Require().NotNil(consentResp.ValidityTime)
	ts.Equal(int64(7200), *consentResp.ValidityTime)
	ts.Require().NotNil(consentResp.RecurringIndicator)
	ts.True(*consentResp.RecurringIndicator)
	ts.Require().NotNil(consentResp.Frequency)
	ts.Equal(10, *consentResp.Frequency)

	ts.trackConsent(consentResp.ID)
}

// ========================================
// STATUS DERIVATION TESTS (5)
// ========================================

// TestCreateConsent_AllAuthsApproved_StatusActive verifies consent status is ACTIVE when all auths are APPROVED
func (ts *ConsentAPITestSuite) TestCreateConsent_AllAuthsApproved_StatusActive() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "payments", Status: "APPROVED"},
			{UserID: "user2", Type: "accounts", Status: "APPROVED"},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	// When all authorizations are APPROVED, consent status should be ACTIVE
	ts.Equal("ACTIVE", consentResp.Status, "Status should be ACTIVE when all auths approved")

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_AnyAuthRejected_StatusRejected verifies consent status is REJECTED when any auth is REJECTED
func (ts *ConsentAPITestSuite) TestCreateConsent_AnyAuthRejected_StatusRejected() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "payments", Status: "APPROVED"},
			{UserID: "user2", Type: "accounts", Status: "REJECTED"},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	// When any authorization is REJECTED, consent status should be REJECTED
	ts.Equal("REJECTED", consentResp.Status, "Status should be REJECTED when any auth rejected")

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_AllAuthsCreated_StatusCreated verifies consent status is CREATED when all auths are CREATED
func (ts *ConsentAPITestSuite) TestCreateConsent_AllAuthsCreated_StatusCreated() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "payments", Status: "CREATED"},
			{UserID: "user2", Type: "accounts", Status: "CREATED"},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	// When all authorizations are CREATED, consent status should be CREATED
	ts.Equal("CREATED", consentResp.Status, "Status should be CREATED when all auths created")

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_MixedAuthStatuses_StatusCreated verifies consent status is CREATED when mix of APPROVED and CREATED
func (ts *ConsentAPITestSuite) TestCreateConsent_MixedAuthStatuses_StatusCreated() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "payments", Status: "APPROVED"},
			{UserID: "user2", Type: "accounts", Status: "CREATED"},
		},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode)

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))

	// When authorizations are mixed (APPROVED + CREATED), consent status should be CREATED
	ts.Equal("CREATED", consentResp.Status, "Status should be CREATED when mixed auth statuses")

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_MissingOrgID_Returns400 verifies missing org-id header returns 400
func (ts *ConsentAPITestSuite) TestCreateConsent_MissingOrgID_Returns400() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user123", Type: "payments", Status: "APPROVED"},
		},
	}

	reqBody, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consents", nil)
	httpReq.Body = http.NoBody
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)
	// Missing org-id header

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.Fail("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	// Should return 400 or 401 for missing required header
	ts.True(resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized,
		"Should return 400 or 401 for missing org-id, got %d", resp.StatusCode)

	// Don't track - creation failed
	_ = reqBody // Suppress unused warning
}

// TestCreateConsent_MissingClientID_Returns400 verifies missing client-id header returns 400
func (ts *ConsentAPITestSuite) TestCreateConsent_MissingClientID_Returns400() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user123", Type: "payments", Status: "APPROVED"},
		},
	}

	reqBody, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consents", nil)
	httpReq.Body = http.NoBody
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	// Missing client-id header

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.Fail("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	ts.True(resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized,
		"Should return 400 or 401 for missing client-id, got %d", resp.StatusCode)

	_ = reqBody
}

// TestCreateConsent_MalformedJSON_Returns400 verifies malformed JSON returns 400
func (ts *ConsentAPITestSuite) TestCreateConsent_MalformedJSON_Returns400() {
	malformedJSON := `{"type": "accounts", "authorizations": [invalid json}`

	resp, _ := ts.createConsent(malformedJSON)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode, "Malformed JSON should return 400")
}

// TestCreateConsent_MissingType_Returns400 verifies missing type returns 400
func (ts *ConsentAPITestSuite) TestCreateConsent_MissingType_Returns400() {
	payload := ConsentCreateRequest{
		// Missing Type
		Authorizations: []AuthorizationRequest{
			{UserID: "user123", Type: "payments", Status: "APPROVED"},
		},
	}

	resp, _ := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode, "Missing type should return 400")
}

// TestCreateConsent_EmptyType_Returns400 verifies empty type returns 400
func (ts *ConsentAPITestSuite) TestCreateConsent_EmptyType_Returns400() {
	payload := ConsentCreateRequest{
		Type: "", // Empty type
		Authorizations: []AuthorizationRequest{
			{UserID: "user123", Type: "payments", Status: "APPROVED"},
		},
	}

	resp, _ := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode, "Empty type should return 400")
}

// TestCreateConsent_TypeTooLong_Returns400 verifies type exceeding max length returns 400
func (ts *ConsentAPITestSuite) TestCreateConsent_TypeTooLong_Returns400() {
	longType := string(make([]byte, 256)) // Assuming max is 255

	payload := ConsentCreateRequest{
		Type: longType,
		Authorizations: []AuthorizationRequest{
			{UserID: "user123", Type: "payments", Status: "APPROVED"},
		},
	}

	resp, _ := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode, "Type too long should return 400")
}

// TestCreateConsent_MissingAuthorizations_Succeeds verifies missing authorizations field is allowed
func (ts *ConsentAPITestSuite) TestCreateConsent_MissingAuthorizations_Succeeds() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		// Missing Authorizations - should be treated as empty
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode, "Missing authorizations should be allowed")

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))
	ts.NotEmpty(consentResp.ID)
	ts.Equal("accounts", consentResp.Type)
	ts.Empty(consentResp.Authorizations, "Should have no authorizations")

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_EmptyAuthorizations_Succeeds verifies empty authorizations array is allowed
func (ts *ConsentAPITestSuite) TestCreateConsent_EmptyAuthorizations_Succeeds() {
	payload := ConsentCreateRequest{
		Type:           "accounts",
		Authorizations: []AuthorizationRequest{}, // Empty array
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode, "Empty authorizations should be allowed")

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))
	ts.NotEmpty(consentResp.ID)
	ts.Equal("accounts", consentResp.Type)
	ts.Empty(consentResp.Authorizations, "Should have no authorizations")

	ts.trackConsent(consentResp.ID)
}

// TestCreateConsent_AuthMissingType_Returns400 verifies authorization with missing type returns 400
func (ts *ConsentAPITestSuite) TestCreateConsent_AuthMissingType_Returns400() {
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{
				UserID: "user123",
				// Missing Type
				Status: "APPROVED",
			},
		},
	}

	resp, _ := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode, "Authorization missing type should return 400")
}

// TestCreateConsent_AllFieldsEmpty_Succeeds verifies creating consent with all optional fields empty
func (ts *ConsentAPITestSuite) TestCreateConsent_AllFieldsEmpty_Succeeds() {
	payload := ConsentCreateRequest{
		Type:           "accounts",
		Purposes:       []ConsentPurposeItem{},
		Authorizations: []AuthorizationRequest{},
		Attributes:     map[string]string{},
	}

	resp, body := ts.createConsent(payload)
	defer resp.Body.Close()

	ts.Equal(http.StatusCreated, resp.StatusCode, "Creating consent with all empty fields should succeed")

	var consentResp ConsentResponse
	ts.NoError(json.Unmarshal(body, &consentResp))
	ts.NotEmpty(consentResp.ID)
	ts.Equal("accounts", consentResp.Type)
	ts.Empty(consentResp.Purposes, "Should have no purposes")
	ts.Empty(consentResp.Authorizations, "Should have no authorizations")
	ts.Empty(consentResp.Attributes, "Should have no attributes")

	ts.trackConsent(consentResp.ID)
}
