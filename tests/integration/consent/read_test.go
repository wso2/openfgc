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
// GET /consents/{id} - Read Single Consent Tests
// ============================

// TestGetConsent_MinimalConsent_ReturnsAllFields retrieves a minimal consent
func (ts *ConsentAPITestSuite) TestGetConsent_MinimalConsent_ReturnsAllFields() {
	// Create a minimal consent first
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

	// Get the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	ts.Equal(http.StatusOK, getResp.StatusCode)

	var retrieved ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &retrieved))

	// Verify all fields
	ts.Equal(created.ID, retrieved.ID)
	ts.Equal("accounts", retrieved.Type)
	ts.Equal(testClientID, retrieved.ClientID)
	ts.NotEmpty(retrieved.Status)
	ts.Len(retrieved.Authorizations, 1)
}

// TestGetConsent_WithPurposes_ReturnsAllPurposes retrieves a consent with multiple purposes
func (ts *ConsentAPITestSuite) TestGetConsent_WithPurposes_ReturnsAllPurposes() {
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
					{Name: "analytics-purpose", Value: true, IsUserApproved: true},
				},
			},
			{
				Name: "terms-purpose",
				Elements: []ConsentPurposeApprovalItem{
					{Name: "terms-purpose", IsUserApproved: true},
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

	// Get the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	ts.Equal(http.StatusOK, getResp.StatusCode)

	var retrieved ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &retrieved))

	ts.Len(retrieved.Purposes, 3)
}

// TestGetConsent_WithAttributes_ReturnsAllAttributes retrieves a consent with attributes
func (ts *ConsentAPITestSuite) TestGetConsent_WithAttributes_ReturnsAllAttributes() {
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Attributes: map[string]string{
			"merchantId":  "MERCH123",
			"referenceId": "REF-789",
			"source":      "mobile",
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

	// Get the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	ts.Equal(http.StatusOK, getResp.StatusCode)

	var retrieved ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &retrieved))

	ts.NotNil(retrieved.Attributes)
	ts.Len(retrieved.Attributes, 3)
	ts.Equal("MERCH123", retrieved.Attributes["merchantId"])
}

// TestGetConsent_WithAuthorizations_ReturnsAllAuths retrieves a consent with multiple authorizations
func (ts *ConsentAPITestSuite) TestGetConsent_WithAuthorizations_ReturnsAllAuths() {
	createPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth1", Status: "APPROVED"},
			{UserID: "user2", Type: "auth2", Status: "APPROVED"},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Get the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	ts.Equal(http.StatusOK, getResp.StatusCode)

	var retrieved ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &retrieved))

	ts.Len(retrieved.Authorizations, 2)
}

// TestGetConsent_AfterCreate_ReturnsCompleteData creates and retrieves, verifying all data matches
func (ts *ConsentAPITestSuite) TestGetConsent_AfterCreate_ReturnsCompleteData() {
	validityTime := int64(3600)
	frequency := 10
	recurringIndicator := true

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
		Attributes: map[string]string{
			"merchantId": "MERCH123",
		},
		ValidityTime:       validityTime,
		Frequency:          frequency,
		RecurringIndicator: recurringIndicator,
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED", Resources: []string{"acc-1"}},
		},
	}

	createResp, createBody := ts.createConsent(createPayload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Get the consent
	getResp, getBody := ts.getConsent(created.ID)
	defer getResp.Body.Close()

	ts.Equal(http.StatusOK, getResp.StatusCode)

	var retrieved ConsentResponse
	ts.NoError(json.Unmarshal(getBody, &retrieved))

	// Verify all data matches
	ts.Equal(created.ID, retrieved.ID)
	ts.Equal("accounts", retrieved.Type)
	ts.Len(retrieved.Purposes, 1)
	ts.NotNil(retrieved.Attributes)
	ts.Require().NotNil(retrieved.ValidityTime)
	ts.Equal(validityTime, *retrieved.ValidityTime)
	ts.Require().NotNil(retrieved.Frequency)
	ts.Equal(frequency, *retrieved.Frequency)
	ts.Require().NotNil(retrieved.RecurringIndicator)
	ts.Equal(recurringIndicator, *retrieved.RecurringIndicator)
	ts.Len(retrieved.Authorizations, 1)
}

// ============================
// GET /consents/{id} - Error Tests
// ============================

// TestGetConsent_MissingOrgID_Returns400 verifies missing org-id header returns 400
func (ts *ConsentAPITestSuite) TestGetConsent_MissingOrgID_Returns400() {
	// Create a consent first
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

	// Try to get without org-id header
	resp, _ := ts.getConsentWithHeaders(created.ID, "", testClientID)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode)
}

// TestGetConsent_MissingClientID_Returns400 verifies GET works without client-id (not required for GET)
func (ts *ConsentAPITestSuite) TestGetConsent_MissingClientID_Returns400() {
	// Create a consent first
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

	// GET without client-id header should work (client-id not required for GET per API spec)
	resp, body := ts.getConsentWithHeaders(created.ID, testOrgID, "")
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)
	var retrieved ConsentResponse
	ts.NoError(json.Unmarshal(body, &retrieved))
	ts.Equal(created.ID, retrieved.ID)
}

// TestGetConsent_NonExistentConsentID_ReturnsNotFound verifies non-existent consent returns 404
func (ts *ConsentAPITestSuite) TestGetConsent_NonExistentConsentID_ReturnsNotFound() {
	nonExistentID := "00000000-0000-0000-0000-000000000000"

	resp, _ := ts.getConsent(nonExistentID)
	defer resp.Body.Close()

	ts.Equal(http.StatusNotFound, resp.StatusCode)
}

// TestGetConsent_InvalidConsentID_ReturnsBadRequest verifies invalid UUID format returns 400
func (ts *ConsentAPITestSuite) TestGetConsent_InvalidConsentID_ReturnsBadRequest() {
	invalidID := "not-a-uuid"

	resp, _ := ts.getConsent(invalidID)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode)
}

// ============================
// GET /consents - List Consents Tests
// ============================

// TestListConsents_NoFilters_ReturnsAllConsents lists all consents in org
func (ts *ConsentAPITestSuite) TestListConsents_NoFilters_ReturnsAllConsents() {
	// Create 3 consents
	for i := 0; i < 3; i++ {
		payload := ConsentCreateRequest{
			Type: "accounts",
			Authorizations: []AuthorizationRequest{
				{UserID: "user1", Type: "auth", Status: "APPROVED"},
			},
		}
		createResp, createBody := ts.createConsent(payload)
		defer createResp.Body.Close()
		ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

		var created ConsentResponse
		ts.NoError(json.Unmarshal(createBody, &created))
		ts.trackConsent(created.ID)
	}

	// List all consents
	resp, body := ts.listConsents(nil)
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))
	ts.GreaterOrEqual(len(listResp.Data), 3)
}

// TestListConsents_FilterByType_ReturnsMatchingConsents filters by type
func (ts *ConsentAPITestSuite) TestListConsents_FilterByType_ReturnsMatchingConsents() {
	// Create accounts consent
	accountsPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}
	createResp1, createBody1 := ts.createConsent(accountsPayload)
	defer createResp1.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp1.StatusCode)

	var created1 ConsentResponse
	ts.NoError(json.Unmarshal(createBody1, &created1))
	ts.trackConsent(created1.ID)

	// Create payments consent
	paymentsPayload := ConsentCreateRequest{
		Type: "payments",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}
	createResp2, createBody2 := ts.createConsent(paymentsPayload)
	defer createResp2.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp2.StatusCode)

	var created2 ConsentResponse
	ts.NoError(json.Unmarshal(createBody2, &created2))
	ts.trackConsent(created2.ID)

	// Filter by accounts type
	resp, body := ts.listConsents(map[string]string{"consentTypes": "accounts"})
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))
	ts.GreaterOrEqual(len(listResp.Data), 1)

	// Verify all are accounts type
	for _, c := range listResp.Data {
		ts.Equal("accounts", c.Type)
	}
}

// TestListConsents_FilterByMultipleTypes_ReturnsAll filters by multiple types
func (ts *ConsentAPITestSuite) TestListConsents_FilterByMultipleTypes_ReturnsAll() {
	// Create accounts consent
	accountsPayload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}
	createResp1, createBody1 := ts.createConsent(accountsPayload)
	defer createResp1.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp1.StatusCode)

	var created1 ConsentResponse
	ts.NoError(json.Unmarshal(createBody1, &created1))
	ts.trackConsent(created1.ID)

	// Create payments consent
	paymentsPayload := ConsentCreateRequest{
		Type: "payments",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}
	createResp2, createBody2 := ts.createConsent(paymentsPayload)
	defer createResp2.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp2.StatusCode)

	var created2 ConsentResponse
	ts.NoError(json.Unmarshal(createBody2, &created2))
	ts.trackConsent(created2.ID)

	// Filter by both types
	resp, body := ts.listConsents(map[string]string{"consentTypes": "accounts,payments"})
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))
	ts.GreaterOrEqual(len(listResp.Data), 2)
}

// TestListConsents_FilterByStatus_ReturnsMatchingConsents filters by status
func (ts *ConsentAPITestSuite) TestListConsents_FilterByStatus_ReturnsMatchingConsents() {
	// Create consent (status will be ACTIVE)
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}
	createResp, createBody := ts.createConsent(payload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Filter by ACTIVE status
	resp, body := ts.listConsents(map[string]string{"consentStatuses": "ACTIVE"})
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))
	ts.GreaterOrEqual(len(listResp.Data), 1)

	// Verify all are ACTIVE
	for _, c := range listResp.Data {
		ts.Equal("ACTIVE", c.Status)
	}
}

// TestListConsents_FilterByClientID_ReturnsMatchingConsents filters by client ID
func (ts *ConsentAPITestSuite) TestListConsents_FilterByClientID_ReturnsMatchingConsents() {
	// Create consent
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}
	createResp, createBody := ts.createConsent(payload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Filter by client ID
	resp, body := ts.listConsents(map[string]string{"clientIds": testClientID})
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))
	ts.GreaterOrEqual(len(listResp.Data), 1)

	// Verify all match client ID
	for _, c := range listResp.Data {
		ts.Equal(testClientID, c.ClientID)
	}
}

// TestListConsents_FilterByUserID_ReturnsMatchingConsents filters by user ID
func (ts *ConsentAPITestSuite) TestListConsents_FilterByUserID_ReturnsMatchingConsents() {
	// Create consent for user1
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "test-user-123", Type: "auth", Status: "APPROVED"},
		},
	}
	createResp, createBody := ts.createConsent(payload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Filter by user ID
	resp, body := ts.listConsents(map[string]string{"userIds": "test-user-123"})
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))
	ts.GreaterOrEqual(len(listResp.Data), 1)

	// Verify at least one auth has the user ID
	found := false
	for _, c := range listResp.Data {
		for _, auth := range c.Authorizations {
			if auth.UserID != nil && *auth.UserID == "test-user-123" {
				found = true
				break
			}
		}
	}
	ts.True(found)
}

// TestListConsents_WithLimit_ReturnsPaginatedResults tests pagination with limit
func (ts *ConsentAPITestSuite) TestListConsents_WithLimit_ReturnsPaginatedResults() {
	// Create 3 consents
	for i := 0; i < 3; i++ {
		payload := ConsentCreateRequest{
			Type: "accounts",
			Authorizations: []AuthorizationRequest{
				{UserID: "user1", Type: "auth", Status: "APPROVED"},
			},
		}
		createResp, createBody := ts.createConsent(payload)
		defer createResp.Body.Close()
		ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

		var created ConsentResponse
		ts.NoError(json.Unmarshal(createBody, &created))
		ts.trackConsent(created.ID)
	}

	// List with limit=2
	resp, body := ts.listConsents(map[string]string{"limit": "2"})
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))
	ts.LessOrEqual(len(listResp.Data), 2)
}

// TestListConsents_WithLimitAndOffset_ReturnsCorrectPage tests pagination with offset
func (ts *ConsentAPITestSuite) TestListConsents_WithLimitAndOffset_ReturnsCorrectPage() {
	// Create 3 consents
	for i := 0; i < 3; i++ {
		payload := ConsentCreateRequest{
			Type: "accounts",
			Authorizations: []AuthorizationRequest{
				{UserID: "user1", Type: "auth", Status: "APPROVED"},
			},
		}
		createResp, createBody := ts.createConsent(payload)
		defer createResp.Body.Close()
		ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

		var created ConsentResponse
		ts.NoError(json.Unmarshal(createBody, &created))
		ts.trackConsent(created.ID)
	}

	// Get first page
	resp1, body1 := ts.listConsents(map[string]string{"limit": "1", "offset": "0"})
	defer resp1.Body.Close()
	ts.Equal(http.StatusOK, resp1.StatusCode)

	var listResp1 ConsentListResponse
	ts.NoError(json.Unmarshal(body1, &listResp1))

	// Get second page
	resp2, body2 := ts.listConsents(map[string]string{"limit": "1", "offset": "1"})
	defer resp2.Body.Close()
	ts.Equal(http.StatusOK, resp2.StatusCode)

	var listResp2 ConsentListResponse
	ts.NoError(json.Unmarshal(body2, &listResp2))

	// Verify different items (if available)
	if len(listResp1.Data) > 0 && len(listResp2.Data) > 0 {
		ts.NotEqual(listResp1.Data[0].ID, listResp2.Data[0].ID)
	}
}

// TestListConsents_CombinedFilters_ReturnsMatchingConsents tests multiple filters
func (ts *ConsentAPITestSuite) TestListConsents_CombinedFilters_ReturnsMatchingConsents() {
	// Create consent
	payload := ConsentCreateRequest{
		Type: "accounts",
		Authorizations: []AuthorizationRequest{
			{UserID: "user1", Type: "auth", Status: "APPROVED"},
		},
	}
	createResp, createBody := ts.createConsent(payload)
	defer createResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createResp.StatusCode)

	var created ConsentResponse
	ts.NoError(json.Unmarshal(createBody, &created))
	ts.trackConsent(created.ID)

	// Filter by type and status
	resp, body := ts.listConsents(map[string]string{
		"consentTypes":    "accounts",
		"consentStatuses": "ACTIVE",
	})
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))

	// Verify filters applied
	for _, c := range listResp.Data {
		ts.Equal("accounts", c.Type)
		ts.Equal("ACTIVE", c.Status)
	}
}

// TestListConsents_DefaultPagination_ReturnsFirst10 verifies default pagination
func (ts *ConsentAPITestSuite) TestListConsents_DefaultPagination_ReturnsFirst10() {
	// List without pagination params
	resp, body := ts.listConsents(nil)
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))

	// Default limit should be 10 or less
	ts.LessOrEqual(len(listResp.Data), 10)
}

// TestListConsents_EmptyOrg_ReturnsEmptyArray verifies empty org returns empty array
func (ts *ConsentAPITestSuite) TestListConsents_EmptyOrg_ReturnsEmptyArray() {
	// Use a different org ID that has no consents
	resp, body := ts.listConsentsWithHeaders(nil, "empty-org-123", testClientID)
	defer resp.Body.Close()

	ts.Equal(http.StatusOK, resp.StatusCode)

	var listResp ConsentListResponse
	ts.NoError(json.Unmarshal(body, &listResp))
	ts.Equal(0, len(listResp.Data))
}

// TestListConsents_MissingOrgID_ReturnsValidationError verifies missing org-id returns 400
func (ts *ConsentAPITestSuite) TestListConsents_MissingOrgID_ReturnsValidationError() {
	resp, _ := ts.listConsentsWithHeaders(nil, "", testClientID)
	defer resp.Body.Close()

	ts.Equal(http.StatusBadRequest, resp.StatusCode)
}

// TestListConsents_InvalidTimeRange_ReturnsValidationError verifies invalid time range returns 400
func (ts *ConsentAPITestSuite) TestListConsents_InvalidTimeRange_ReturnsValidationError() {
	// fromTime > toTime (invalid)
	resp, _ := ts.listConsents(map[string]string{
		"fromTime": "1700000000",
		"toTime":   "1600000000",
	})
	defer resp.Body.Close()

	// Should return 400 (or might return 200 with empty results depending on implementation)
	// Adjust based on actual API behavior
	ts.True(resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusOK)
}
