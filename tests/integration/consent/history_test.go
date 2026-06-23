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
	"time"
)

func (ts *ConsentAPITestSuite) TestConsentHistory() {
	ts.Run("new consent has empty history", func() {
		orgID := freshOrgID()
		created := ts.mustCreateConsent(orgID, "grp-hist-empty", ConsentCreateRequest{Type: "accounts"})

		status, history := ts.doGetConsentHistory(orgID, created.ID, false)

		ts.Equal(http.StatusOK, status)
		ts.Require().NotNil(history)
		ts.Equal(created.ID, history.ID)
		ts.Empty(history.History)
	})

	ts.Run("consent PUT creates history with pre-update snapshot", func() {
		orgID := freshOrgID()
		groupID := "grp-hist-put"
		created := ts.mustCreateConsent(orgID, groupID, ConsentCreateRequest{
			Type:       "accounts",
			Attributes: map[string]string{"region": "EU"},
			Authorizations: []AuthorizationRequest{
				{UserID: "user-001", Type: "authorisation", Status: "APPROVED"},
			},
		})

		status, updated := ts.doUpdateConsent(orgID, groupID, created.ID, ConsentUpdateRequest{
			Type:       "payments",
			Attributes: map[string]string{"region": "LK"},
		})
		ts.Equal(http.StatusOK, status)
		ts.Require().NotNil(updated)

		_, historyWithoutSnapshots := ts.doGetConsentHistory(orgID, created.ID, false)
		ts.Require().Len(historyWithoutSnapshots.History, 1)
		entry := historyWithoutSnapshots.History[0]
		ts.Require().NotNil(entry.Reason)
		ts.Equal("Consent updated", *entry.Reason)
		ts.Require().NotNil(entry.ActionBy)
		ts.Equal(groupID, *entry.ActionBy)
		ts.Empty(entry.Snapshot)
		ts.assertHistorySnapshotFieldPresence(orgID, created.ID, false, false)

		_, historyWithSnapshots := ts.doGetConsentHistory(orgID, created.ID, true)
		ts.Require().Len(historyWithSnapshots.History, 1)
		snapshot := ts.decodeHistorySnapshot(historyWithSnapshots.History[0])
		ts.Equal("accounts", snapshot["type"])
		ts.Equal("EU", snapshot["attributes"].(map[string]interface{})["region"])
		ts.assertHistorySnapshotFieldPresence(orgID, created.ID, true, true)
	})

	ts.Run("revoke creates history", func() {
		orgID := freshOrgID()
		created := ts.mustCreateConsent(orgID, "grp-hist-revoke", ConsentCreateRequest{Type: "accounts"})

		status, revokeResp := ts.doRevokeConsent(orgID, created.ID, ConsentRevokeRequest{ActionBy: "tester"})
		ts.Equal(http.StatusOK, status)
		ts.Require().NotNil(revokeResp)

		_, history := ts.doGetConsentHistory(orgID, created.ID, false)
		entry := findHistoryByReason(history.History, "Consent revoked")
		ts.Require().NotNil(entry)
		ts.Require().NotNil(entry.ActionBy)
		ts.Equal("tester", *entry.ActionBy)
	})

	ts.Run("expiry creates history", func() {
		orgID := freshOrgID()
		futureExpiration := time.Now().Add(2 * time.Minute).UnixMilli()
		pastExpiration := time.Now().Add(-2 * time.Minute).UnixMilli()
		created := ts.mustCreateConsent(orgID, "grp-hist-expiry", ConsentCreateRequest{
			Type:           "accounts",
			ExpirationTime: &futureExpiration,
			Authorizations: []AuthorizationRequest{
				{UserID: "user-001", Status: "APPROVED"},
			},
		})

		status, updated := ts.doUpdateConsent(orgID, "grp-hist-expiry", created.ID, ConsentUpdateRequest{ExpirationTime: &pastExpiration})
		ts.Equal(http.StatusOK, status)
		ts.Require().NotNil(updated)
		ts.Equal("EXPIRED", updated.Status)

		_, history := ts.doGetConsentHistory(orgID, created.ID, false)
		updateEntry := findHistoryByReason(history.History, "Consent updated")
		expireEntry := findHistoryByReason(history.History, "Consent expired")
		ts.Require().NotNil(updateEntry)
		ts.Require().NotNil(expireEntry)
		ts.Require().NotNil(expireEntry.ActionBy)
		ts.Equal("SYSTEM", *expireEntry.ActionBy)
	})

	ts.Run("auth-resource POST creates history", func() {
		orgID := freshOrgID()
		created := ts.mustCreateConsent(orgID, "grp-hist-auth-post", ConsentCreateRequest{Type: "accounts"})

		status, body := ts.doRequest(http.MethodPost, "/api/v1/consents/"+created.ID+"/authorizations", orgID, "", map[string]interface{}{
			"userId": "user-001",
			"status": "APPROVED",
		})
		ts.Equal(http.StatusOK, status, "unexpected auth-resource POST response: %s", body)

		_, history := ts.doGetConsentHistory(orgID, created.ID, false)
		entry := findHistoryByReason(history.History, "Consent authorizations added")
		ts.Require().NotNil(entry)
		ts.Nil(entry.ActionBy)
	})

	ts.Run("auth-resource PUT creates history", func() {
		orgID := freshOrgID()
		created := ts.mustCreateConsent(orgID, "grp-hist-auth-put", ConsentCreateRequest{Type: "accounts"})
		authID := ts.createAuthResourceForHistory(orgID, created.ID)

		status, body := ts.doRequest(http.MethodPut, "/api/v1/consents/"+created.ID+"/authorizations/"+authID, orgID, "", map[string]interface{}{
			"userId": "user-001",
			"status": "REJECTED",
		})
		ts.Equal(http.StatusOK, status, "unexpected auth-resource PUT response: %s", body)

		_, history := ts.doGetConsentHistory(orgID, created.ID, false)
		entry := findHistoryByReason(history.History, "Consent authorizations updated")
		ts.Require().NotNil(entry)
		ts.Nil(entry.ActionBy)
	})

	ts.Run("includeStatusHistory controls statusHistory on consent GET", func() {
		orgID := freshOrgID()
		created := ts.mustCreateConsent(orgID, "grp-hist-status", ConsentCreateRequest{
			Type: "accounts",
			Authorizations: []AuthorizationRequest{
				{UserID: "user-001", Status: "APPROVED"},
			},
		})

		status, rawBody := ts.doRequest(http.MethodGet, "/api/v1/consents/"+created.ID, orgID, "", nil)
		ts.Equal(http.StatusOK, status)
		ts.assertRawFieldPresence(rawBody, "statusHistory", false)

		status, withHistory := ts.doGetConsentWithStatusHistory(orgID, created.ID)
		ts.Equal(http.StatusOK, status)
		ts.Require().NotNil(withHistory)
		ts.NotEmpty(withHistory.StatusHistory)
		initial := findStatusHistoryByCurrentStatus(withHistory.StatusHistory, "ACTIVE")
		ts.Require().NotNil(initial)
		ts.Nil(initial.PreviousStatus)

		status, rawWithHistory := ts.doRequest(http.MethodGet, "/api/v1/consents/"+created.ID+"?includeStatusHistory=true", orgID, "", nil)
		ts.Equal(http.StatusOK, status)
		ts.assertInitialStatusHistoryOmitsPreviousStatus(rawWithHistory, "ACTIVE")
	})
}

func (ts *ConsentAPITestSuite) createAuthResourceForHistory(orgID, consentID string) string {
	status, body := ts.doRequest(http.MethodPost, "/api/v1/consents/"+consentID+"/authorizations", orgID, "", map[string]interface{}{
		"userId": "user-001",
		"status": "APPROVED",
	})
	ts.Require().Equal(http.StatusOK, status, "unexpected auth-resource POST response: %s", body)

	var resp struct {
		ID string `json:"id"`
	}
	ts.Require().NoError(json.Unmarshal(body, &resp))
	ts.Require().NotEmpty(resp.ID)
	return resp.ID
}

func findHistoryByReason(history []ConsentHistoryResponse, reason string) *ConsentHistoryResponse {
	for i := range history {
		if history[i].Reason != nil && *history[i].Reason == reason {
			return &history[i]
		}
	}
	return nil
}

func findStatusHistoryByCurrentStatus(history []ConsentStatusAuditResponse, status string) *ConsentStatusAuditResponse {
	for i := range history {
		if history[i].CurrentStatus == status {
			return &history[i]
		}
	}
	return nil
}

func (ts *ConsentAPITestSuite) decodeHistorySnapshot(entry ConsentHistoryResponse) map[string]interface{} {
	ts.Require().NotEmpty(entry.Snapshot)
	var snapshot map[string]interface{}
	ts.Require().NoError(json.Unmarshal(entry.Snapshot, &snapshot))
	return snapshot
}

func (ts *ConsentAPITestSuite) assertHistorySnapshotFieldPresence(orgID, consentID string, includeSnapshots, wantPresent bool) {
	path := "/api/v1/consents/" + consentID + "/history"
	if includeSnapshots {
		path += "?includeSnapshots=true"
	}
	status, body := ts.doRequest(http.MethodGet, path, orgID, "", nil)
	ts.Equal(http.StatusOK, status)

	var raw struct {
		History []map[string]interface{} `json:"history"`
	}
	ts.Require().NoError(json.Unmarshal(body, &raw))
	ts.Require().NotEmpty(raw.History)
	_, exists := raw.History[0]["snapshot"]
	ts.Equal(wantPresent, exists)
}

func (ts *ConsentAPITestSuite) assertRawFieldPresence(body []byte, field string, wantPresent bool) {
	var raw map[string]interface{}
	ts.Require().NoError(json.Unmarshal(body, &raw))
	_, exists := raw[field]
	ts.Equal(wantPresent, exists)
}

func (ts *ConsentAPITestSuite) assertInitialStatusHistoryOmitsPreviousStatus(body []byte, currentStatus string) {
	var raw struct {
		StatusHistory []map[string]interface{} `json:"statusHistory"`
	}
	ts.Require().NoError(json.Unmarshal(body, &raw))
	ts.Require().NotEmpty(raw.StatusHistory)
	for _, item := range raw.StatusHistory {
		if item["currentStatus"] == currentStatus {
			_, exists := item["previousStatus"]
			ts.False(exists)
			return
		}
	}
	ts.FailNow("status history entry not found", "currentStatus=%s", currentStatus)
}
