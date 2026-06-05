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

// TestRevokeConsent covers PUT /consents/{consentId}/revoke.
//
// Key semantics:
//   - actionBy is required; revocationReason is optional.
//   - After revoke, consent status → SYS_REVOKED; all auth resources → SYS_REVOKED.
//   - Revoking an already-revoked consent → 409 CS-4041.
//   - actionTime in the response is a Unix millisecond timestamp.
func (ts *ConsentAPITestSuite) TestRevokeConsent() {
	type testCase struct {
		name string

		// setup creates the consent to revoke and returns the consentID.
		setup func(orgID string) string

		req       ConsentRevokeRequest
		rawBody   string // for parse errors
		consentID string // override for static error cases
		omitOrgID bool

		wantStatus  int
		wantError   string
		checkResult func(orgID, consentID string, resp *ConsentRevokeResponse)
	}

	cases := []testCase{
		// -----------------------------------------------------------------------
		// Happy paths
		// -----------------------------------------------------------------------
		{
			name: "revoke with actionBy only — response contains actionBy and actionTime",
			setup: func(orgID string) string {
				c := ts.mustCreateConsent(orgID, "grp-rev-basic", ConsentCreateRequest{Type: "accounts"})
				return c.ID
			},
			req:        ConsentRevokeRequest{ActionBy: "admin-user"},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *ConsentRevokeResponse) {
				ts.Equal("admin-user", resp.ActionBy)
				ts.Greater(resp.ActionTime, int64(946684800000),
					"actionTime must be a Unix millisecond timestamp")
				ts.Empty(resp.RevocationReason)
			},
		},
		{
			name: "revoke with actionBy and revocationReason — reason is returned",
			setup: func(orgID string) string {
				c := ts.mustCreateConsent(orgID, "grp-rev-reason", ConsentCreateRequest{Type: "accounts"})
				return c.ID
			},
			req: ConsentRevokeRequest{
				ActionBy:         "consent-owner",
				RevocationReason: "user-requested",
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *ConsentRevokeResponse) {
				ts.Equal("consent-owner", resp.ActionBy)
				ts.Equal("user-requested", resp.RevocationReason)
				ts.Greater(resp.ActionTime, int64(946684800000))
			},
		},
		{
			name: "after revoke, GET consent shows status REVOKED",
			setup: func(orgID string) string {
				c := ts.mustCreateConsent(orgID, "grp-rev-status", ConsentCreateRequest{Type: "accounts"})
				return c.ID
			},
			req:        ConsentRevokeRequest{ActionBy: "admin"},
			wantStatus: http.StatusOK,
			checkResult: func(orgID, consentID string, _ *ConsentRevokeResponse) {
				_, got := ts.doGetConsent(orgID, consentID)
				ts.Require().NotNil(got)
				// Consent status uses the configured revoked_status ("REVOKED").
				// Auth resources use the configured system_revoked_state ("SYS_REVOKED") — see next test.
				ts.Equal("REVOKED", got.Status)
			},
		},
		{
			name: "after revoke, all auth resources get SYS_REVOKED status",
			setup: func(orgID string) string {
				c := ts.mustCreateConsent(orgID, "grp-rev-auth-status", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Type: "accounts", Status: "APPROVED"},
						{UserID: "user-002", Type: "savings", Status: "CREATED"},
					},
				})
				return c.ID
			},
			req:        ConsentRevokeRequest{ActionBy: "admin"},
			wantStatus: http.StatusOK,
			checkResult: func(orgID, consentID string, _ *ConsentRevokeResponse) {
				_, got := ts.doGetConsent(orgID, consentID)
				ts.Require().NotNil(got)
				for _, a := range got.Authorizations {
					ts.Equal("SYS_REVOKED", a.Status,
						"all auth resources must be cascaded to SYS_REVOKED on revoke")
				}
			},
		},

		// -----------------------------------------------------------------------
		// Already revoked
		// -----------------------------------------------------------------------
		{
			name: "revoking an already-revoked consent → 409 CS-4041",
			setup: func(orgID string) string {
				c := ts.mustCreateConsent(orgID, "grp-rev-twice", ConsentCreateRequest{Type: "accounts"})
				// First revoke
				status, _ := ts.doRequest(http.MethodPost, "/api/v1/consents/"+c.ID+"/revoke",
					orgID, "", ConsentRevokeRequest{ActionBy: "admin"})
				ts.Require().Equal(http.StatusOK, status, "first revoke should succeed")
				return c.ID
			},
			req:        ConsentRevokeRequest{ActionBy: "admin"},
			wantStatus: http.StatusConflict,
			wantError:  "CS-4041",
		},

		// -----------------------------------------------------------------------
		// Not-found / header / body errors
		// -----------------------------------------------------------------------
		{
			name:       "non-existent consent ID → 404 CS-4040",
			consentID:  "00000000-0000-0000-0000-000000000000",
			req:        ConsentRevokeRequest{ActionBy: "admin"},
			wantStatus: http.StatusNotFound,
			wantError:  "CS-4040",
		},
		{
			name:       "missing org-id header → 400 CS-4002",
			consentID:  "00000000-0000-0000-0000-000000000001",
			omitOrgID:  true,
			req:        ConsentRevokeRequest{ActionBy: "admin"},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "malformed consent ID → 400 CS-4002",
			consentID:  "not-a-uuid",
			req:        ConsentRevokeRequest{ActionBy: "admin"},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "malformed JSON body → 400 CS-4001",
			consentID:  "00000000-0000-0000-0000-000000000000",
			rawBody:    `{bad json`,
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4001",
		},
		{
			name: "missing actionBy → 400 CS-4002",
			setup: func(orgID string) string {
				c := ts.mustCreateConsent(orgID, "grp-rev-no-actionby", ConsentCreateRequest{Type: "accounts"})
				return c.ID
			},
			req:        ConsentRevokeRequest{}, // actionBy is empty
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()

			consentID := tc.consentID
			if tc.setup != nil {
				consentID = tc.setup(orgID)
			}

			revokeOrgID := orgID
			if tc.omitOrgID {
				revokeOrgID = ""
			}

			var body any = tc.req
			if tc.rawBody != "" {
				body = tc.rawBody
			}

			status, respBody := ts.doRequest(http.MethodPost, "/api/v1/consents/"+consentID+"/revoke",
				revokeOrgID, "", body)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", respBody)

			if tc.wantError != "" {
				ts.assertAPIError(respBody, tc.wantError)
				return
			}

			var resp ConsentRevokeResponse
			ts.Require().NoError(json.Unmarshal(respBody, &resp), "unmarshal ConsentRevokeResponse: %s", respBody)
			if tc.checkResult != nil {
				tc.checkResult(orgID, consentID, &resp)
			}
		})
	}
}
