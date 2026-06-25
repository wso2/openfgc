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

// TestConsentHistory covers consent history and status-history read surfaces.
//
// Isolation: every sub-test gets a fresh org via freshOrgID() and no DB cleanup is needed.
//
// Layout:
//   - setup creates any required consent/auth-resource state for the scenario.
//   - check performs the request flow and assertions for the scenario.
//   - History assertions intentionally read through the public API surface rather than
//     inspecting storage directly, so these tests verify contract shape as well as data.
func (ts *ConsentAPITestSuite) TestConsentHistory() {
	type testCase struct {
		name  string
		setup func(orgID string) *ConsentResponse
		check func(orgID string, created *ConsentResponse)
	}

	cases := []testCase{
		// -----------------------------------------------------------------------
		// History list basics
		// -----------------------------------------------------------------------
		{
			name: "new consent has empty history",
			setup: func(orgID string) *ConsentResponse {
				return ts.mustCreateConsent(orgID, "grp-hist-empty", ConsentCreateRequest{Type: "accounts"})
			},
			check: func(orgID string, created *ConsentResponse) {
				status, history := ts.doGetConsentHistory(orgID, created.ID, false)

				ts.Equal(http.StatusOK, status)
				ts.Require().NotNil(history)
				ts.Equal(created.ID, history.ID)
				ts.Empty(history.History)
			},
		},

		// -----------------------------------------------------------------------
		// Consent update history
		// -----------------------------------------------------------------------
		{
			name: "consent PUT creates history with pre-update snapshot",
			setup: func(orgID string) *ConsentResponse {
				return ts.mustCreateConsent(orgID, "grp-hist-put", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"region": "EU"},
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Type: "authorisation", Status: "APPROVED"},
					},
				})
			},
			check: func(orgID string, created *ConsentResponse) {
				groupID := "grp-hist-put"
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
				ts.Equal("EU", snapshot["attributes"].(map[string]any)["region"])
				ts.assertHistorySnapshotFieldPresence(orgID, created.ID, true, true)
			},
		},
		{
			name: "revoke creates history",
			setup: func(orgID string) *ConsentResponse {
				return ts.mustCreateConsent(orgID, "grp-hist-revoke", ConsentCreateRequest{Type: "accounts"})
			},
			check: func(orgID string, created *ConsentResponse) {
				status, revokeResp := ts.doRevokeConsent(orgID, created.ID, ConsentRevokeRequest{ActionBy: "tester"})
				ts.Equal(http.StatusOK, status)
				ts.Require().NotNil(revokeResp)

				_, history := ts.doGetConsentHistory(orgID, created.ID, false)
				entry := findHistoryByReason(history.History, "Consent revoked")
				ts.Require().NotNil(entry)
				ts.Require().NotNil(entry.ActionBy)
				ts.Equal("tester", *entry.ActionBy)
			},
		},
		{
			name: "expiry creates history",
			setup: func(orgID string) *ConsentResponse {
				futureExpiration := time.Now().Add(2 * time.Minute).UnixMilli()
				return ts.mustCreateConsent(orgID, "grp-hist-expiry", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &futureExpiration,
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Status: "APPROVED"},
					},
				})
			},
			check: func(orgID string, created *ConsentResponse) {
				pastExpiration := time.Now().Add(-2 * time.Minute).UnixMilli()
				status, updated := ts.doUpdateConsent(orgID, "grp-hist-expiry", created.ID, ConsentUpdateRequest{
					ExpirationTime: &pastExpiration,
				})
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
			},
		},

		// -----------------------------------------------------------------------
		// Authorization-driven history
		// -----------------------------------------------------------------------
		{
			name: "auth-resource POST creates history",
			setup: func(orgID string) *ConsentResponse {
				return ts.mustCreateConsent(orgID, "grp-hist-auth-post", ConsentCreateRequest{Type: "accounts"})
			},
			check: func(orgID string, created *ConsentResponse) {
				status, body := ts.doRequest(http.MethodPost, "/api/v1/consents/"+created.ID+"/authorizations", orgID, "", map[string]any{
					"userId": "user-001",
					"status": "APPROVED",
				})
				ts.Equal(http.StatusOK, status, "unexpected auth-resource POST response: %s", body)

				_, history := ts.doGetConsentHistory(orgID, created.ID, false)
				entry := findHistoryByReason(history.History, "Consent authorizations added")
				ts.Require().NotNil(entry)
				ts.Nil(entry.ActionBy)
			},
		},
		{
			name: "auth-resource PUT creates history",
			setup: func(orgID string) *ConsentResponse {
				created := ts.mustCreateConsent(orgID, "grp-hist-auth-put", ConsentCreateRequest{Type: "accounts"})
				ts.createAuthResourceForHistory(orgID, created.ID)
				return created
			},
			check: func(orgID string, created *ConsentResponse) {
				authID := ts.findSingleAuthorizationID(orgID, created.ID)
				status, body := ts.doRequest(http.MethodPut, "/api/v1/consents/"+created.ID+"/authorizations/"+authID, orgID, "", map[string]any{
					"userId": "user-001",
					"status": "REJECTED",
				})
				ts.Equal(http.StatusOK, status, "unexpected auth-resource PUT response: %s", body)

				_, history := ts.doGetConsentHistory(orgID, created.ID, false)
				entry := findHistoryByReason(history.History, "Consent authorizations updated")
				ts.Require().NotNil(entry)
				ts.Nil(entry.ActionBy)
			},
		},

		// -----------------------------------------------------------------------
		// Status history on consent GET
		// -----------------------------------------------------------------------
		{
			name: "includeStatusHistory controls statusHistory on consent GET",
			setup: func(orgID string) *ConsentResponse {
				return ts.mustCreateConsent(orgID, "grp-hist-status", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Status: "APPROVED"},
					},
				})
			},
			check: func(orgID string, created *ConsentResponse) {
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
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()
			created := tc.setup(orgID)
			ts.Require().NotNil(created)
			tc.check(orgID, created)
		})
	}
}

func (ts *ConsentAPITestSuite) createAuthResourceForHistory(orgID, consentID string) string {
	status, body := ts.doRequest(http.MethodPost, "/api/v1/consents/"+consentID+"/authorizations", orgID, "", map[string]any{
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

// findSingleAuthorizationID fetches the consent and returns the only authorization id.
// History auth-resource tests use a single authorization, so this keeps setup and check code simple.
func (ts *ConsentAPITestSuite) findSingleAuthorizationID(orgID, consentID string) string {
	status, consent := ts.doGetConsent(orgID, consentID)
	ts.Require().Equal(http.StatusOK, status)
	ts.Require().NotNil(consent)
	ts.Require().Len(consent.Authorizations, 1)
	ts.Require().NotEmpty(consent.Authorizations[0].ID)
	return consent.Authorizations[0].ID
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

func (ts *ConsentAPITestSuite) decodeHistorySnapshot(entry ConsentHistoryResponse) map[string]any {
	ts.Require().NotEmpty(entry.Snapshot)
	var snapshot map[string]any
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
		History []map[string]any `json:"history"`
	}
	ts.Require().NoError(json.Unmarshal(body, &raw))
	ts.Require().NotEmpty(raw.History)
	_, exists := raw.History[0]["snapshot"]
	ts.Equal(wantPresent, exists)
}

func (ts *ConsentAPITestSuite) assertRawFieldPresence(body []byte, field string, wantPresent bool) {
	var raw map[string]any
	ts.Require().NoError(json.Unmarshal(body, &raw))
	_, exists := raw[field]
	ts.Equal(wantPresent, exists)
}

// assertInitialStatusHistoryOmitsPreviousStatus verifies the initial status-audit entry
// does not serialize previousStatus at all, instead of emitting it with a null value.
func (ts *ConsentAPITestSuite) assertInitialStatusHistoryOmitsPreviousStatus(body []byte, currentStatus string) {
	var raw struct {
		StatusHistory []map[string]any `json:"statusHistory"`
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
