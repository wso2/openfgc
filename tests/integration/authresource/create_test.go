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

package authresource

import (
	"encoding/json"
	"net/http"
)

// TestCreateAuthResource covers POST /consents/{consentId}/authorizations.
//
// Isolation: every sub-test gets a fresh org via freshOrgID() and a new consent.
//
// Layout:
//   - buildBody: prepares any pre-conditions and returns the request body
//     (AuthResourceCreateRequest struct, or raw string for parse-error cases).
//   - omitOrgID: drop the org-id header entirely (use for AR-4007 cases).
//   - wantStatus: expected HTTP status (200 on success).
//   - wantErrorCode: expected AR-XXXX code on non-200 responses.
//   - checkResult: optional assertions on the successful response.
func (ts *AuthResourceAPITestSuite) TestCreateAuthResource() {
	type testCase struct {
		name string

		// buildBody prepares pre-conditions and returns the request body.
		// Receives fresh orgID and the pre-created consentID.
		buildBody func(orgID, consentID string) any

		omitOrgID     bool
		wantStatus    int
		wantErrorCode string
		checkResult   func(orgID, consentID string, resp *AuthResourceResponse)
	}

	cases := []testCase{

		// -----------------------------------------------------------------------
		// Defaults
		// -----------------------------------------------------------------------
		{
			name: "type and status default — type becomes 'default', status becomes 'APPROVED'",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001")}
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *AuthResourceResponse) {
				ts.assertAuthResourceResponse(resp)
				ts.Equal("default", resp.Type)
				ts.Equal("APPROVED", resp.Status)
				ts.Require().NotNil(resp.UserID)
				ts.Equal("user-001", *resp.UserID)
				ts.Nil(resp.Resources, "resources must be absent when not provided")
			},
		},
		{
			name: "missing userId → 400 AR-4002",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{Type: "accounts", Status: "APPROVED"}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "AR-4002",
		},

		// -----------------------------------------------------------------------
		// Optional fields
		// -----------------------------------------------------------------------
		{
			name: "with userId — returned in response",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-alice")}
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *AuthResourceResponse) {
				ts.Require().NotNil(resp.UserID)
				ts.Equal("user-alice", *resp.UserID)
			},
		},
		{
			name: "with explicit type — returned in response",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Type: "payments"}
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *AuthResourceResponse) {
				ts.Equal("payments", resp.Type)
			},
		},
		{
			name: "status CREATED — stored and returned",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "CREATED"}
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *AuthResourceResponse) {
				ts.Equal("CREATED", resp.Status)
			},
		},
		{
			name: "status REJECTED — stored and returned",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "REJECTED"}
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *AuthResourceResponse) {
				ts.Equal("REJECTED", resp.Status)
			},
		},

		// -----------------------------------------------------------------------
		// Resources field — flexible JSON (object, string, array)
		// -----------------------------------------------------------------------
		{
			name: "resources as JSON object — returned in response",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{
					UserID:    strPtr("user-001"),
					Resources: map[string]interface{}{"accountIds": []string{"acc-1", "acc-2"}},
				}
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *AuthResourceResponse) {
				ts.NotNil(resp.Resources, "resources must be present in response")
			},
		},
		{
			name: "resources as plain string — round-trips correctly",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Resources: "read-only"}
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *AuthResourceResponse) {
				ts.Require().NotNil(resp.Resources)
				ts.Equal("read-only", resp.Resources)
			},
		},
		{
			name: "resources as JSON array — returned in response",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{
					UserID:    strPtr("user-001"),
					Resources: []string{"scope:read", "scope:write"},
				}
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, resp *AuthResourceResponse) {
				ts.NotNil(resp.Resources)
			},
		},

		// -----------------------------------------------------------------------
		// Consent status derivation
		// -----------------------------------------------------------------------
		{
			name: "APPROVED auth on empty consent — consent status becomes ACTIVE",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "APPROVED"}
			},
			wantStatus: http.StatusOK,
			checkResult: func(orgID, consentID string, _ *AuthResourceResponse) {
				ts.Equal("ACTIVE", ts.getConsentStatus(orgID, consentID),
					"consent must become ACTIVE after one APPROVED auth")
			},
		},
		{
			name: "REJECTED auth on empty consent — consent status becomes REJECTED",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "REJECTED"}
			},
			wantStatus: http.StatusOK,
			checkResult: func(orgID, consentID string, _ *AuthResourceResponse) {
				ts.Equal("REJECTED", ts.getConsentStatus(orgID, consentID),
					"consent must become REJECTED after one REJECTED auth")
			},
		},
		{
			name: "CREATED auth on empty consent — consent status stays CREATED",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "CREATED"}
			},
			wantStatus: http.StatusOK,
			checkResult: func(orgID, consentID string, _ *AuthResourceResponse) {
				ts.Equal("CREATED", ts.getConsentStatus(orgID, consentID),
					"consent status must remain CREATED after one CREATED auth")
			},
		},

		// -----------------------------------------------------------------------
		// Multiple auth resources for the same consent
		// -----------------------------------------------------------------------
		{
			name: "second auth resource for same consent — both are returned by list",
			buildBody: func(orgID, consentID string) any {
				ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{
					UserID: strPtr("user-001"), Type: "primary", Status: "APPROVED",
				})
				return AuthResourceCreateRequest{UserID: strPtr("user-002"), Type: "secondary", Status: "CREATED"}
			},
			wantStatus: http.StatusOK,
			checkResult: func(orgID, consentID string, _ *AuthResourceResponse) {
				_, list := ts.doListAuthResources(orgID, consentID)
				ts.Require().Len(list, 2, "both auth resources must be listed for the consent")
			},
		},

		// -----------------------------------------------------------------------
		// Validation errors — system-reserved statuses
		// -----------------------------------------------------------------------
		{
			name: "system-reserved status 'SYS_EXPIRED' → 400 AR-4002",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "SYS_EXPIRED"}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "AR-4002",
		},
		{
			name: "system-reserved status 'SYS_REVOKED' → 400 AR-4002",
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "SYS_REVOKED"}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "AR-4002",
		},

		// -----------------------------------------------------------------------
		// Header errors
		// -----------------------------------------------------------------------
		{
			name:      "missing org-id header → 400 AR-4007",
			omitOrgID: true,
			buildBody: func(_, _ string) any {
				return AuthResourceCreateRequest{}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "AR-4007",
		},

		// -----------------------------------------------------------------------
		// Body parse errors
		// -----------------------------------------------------------------------
		{
			name: "malformed JSON body → 400 AR-4001",
			buildBody: func(_, _ string) any {
				return `{invalid-json`
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "AR-4001",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()
			consentID := ts.mustCreateConsent(orgID, "grp-ar-create")

			orgForReq := orgID
			if tc.omitOrgID {
				orgForReq = ""
			}

			status, body := ts.doRequest(
				http.MethodPost,
				"/api/v1/consents/"+consentID+"/authorizations",
				orgForReq,
				tc.buildBody(orgID, consentID),
			)

			ts.Equal(tc.wantStatus, status)
			if tc.wantErrorCode != "" {
				ts.assertAPIError(body, tc.wantErrorCode)
			} else if tc.checkResult != nil {
				var resp AuthResourceResponse
				ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal response: %s", body)
				tc.checkResult(orgID, consentID, &resp)
			}
		})
	}
}
