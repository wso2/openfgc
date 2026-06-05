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

// TestGetAuthResource covers GET /consents/{consentId}/authorizations/{authorizationId}.
//
// Isolation: each sub-test gets a fresh org and a dedicated consent.
func (ts *AuthResourceAPITestSuite) TestGetAuthResource() {
	type testCase struct {
		name string

		// setup creates pre-conditions and returns the consentID and authID to GET.
		// May return a different authID / consentID than the created resource to exercise error cases.
		setup func(orgID string) (consentID, authID string)

		// useAltOrg makes the GET request with a different fresh org (cross-org isolation tests).
		useAltOrg bool

		omitOrgID     bool
		wantStatus    int
		wantErrorCode string
		checkResult   func(resp *AuthResourceResponse)
	}

	cases := []testCase{

		// -----------------------------------------------------------------------
		// Happy path
		// -----------------------------------------------------------------------
		{
			name: "get created auth resource — all fields returned correctly",
			setup: func(orgID string) (string, string) {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-get-ok")
				ar := ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{
					UserID: strPtr("user-bob"),
					Type:   "payments",
					Status: "APPROVED",
				})
				return consentID, ar.ID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *AuthResourceResponse) {
				ts.assertAuthResourceResponse(resp)
				ts.Equal("payments", resp.Type)
				ts.Equal("APPROVED", resp.Status)
				ts.Require().NotNil(resp.UserID)
				ts.Equal("user-bob", *resp.UserID)
			},
		},

		// -----------------------------------------------------------------------
		// Not found
		// -----------------------------------------------------------------------
		{
			name: "unknown auth ID → 404 AR-4040",
			setup: func(orgID string) (string, string) {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-get-nf")
				return consentID, "00000000-0000-0000-0000-000000000000"
			},
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "AR-4040",
		},

		// -----------------------------------------------------------------------
		// Cross-consent isolation
		// Auth resource belongs to consent A; request uses consent B's URL.
		// -----------------------------------------------------------------------
		{
			name: "auth from consent A fetched via consent B's URL → 404 AR-4040",
			setup: func(orgID string) (string, string) {
				consentA := ts.mustCreateConsent(orgID, "grp-ar-cross-a")
				consentB := ts.mustCreateConsent(orgID, "grp-ar-cross-b")
				ar := ts.mustCreateAuthResource(orgID, consentA, AuthResourceCreateRequest{UserID: strPtr("user-001")})
				// Return consentB's path but authA's ID — server must reject this.
				return consentB, ar.ID
			},
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "AR-4040",
		},

		// -----------------------------------------------------------------------
		// Cross-org isolation
		// -----------------------------------------------------------------------
		{
			name: "auth created under org A, fetched with org B's header → 404 AR-4040",
			setup: func(orgID string) (string, string) {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-get-org")
				ar := ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{UserID: strPtr("user-001")})
				return consentID, ar.ID
			},
			useAltOrg:     true,
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "AR-4040",
		},

		// -----------------------------------------------------------------------
		// Header errors
		// -----------------------------------------------------------------------
		{
			name: "missing org-id header → 400 AR-4007",
			setup: func(orgID string) (string, string) {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-get-hdr")
				return consentID, "some-auth-id"
			},
			omitOrgID:     true,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "AR-4007",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()
			consentID, authID := tc.setup(orgID)

			orgForReq := orgID
			switch {
			case tc.omitOrgID:
				orgForReq = ""
			case tc.useAltOrg:
				orgForReq = freshOrgID()
			}

			status, body := ts.doRequest(
				http.MethodGet,
				"/api/v1/consents/"+consentID+"/authorizations/"+authID,
				orgForReq,
				nil,
			)

			ts.Equal(tc.wantStatus, status)
			if tc.wantErrorCode != "" {
				ts.assertAPIError(body, tc.wantErrorCode)
			} else if tc.checkResult != nil {
				var resp AuthResourceResponse
				ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal response: %s", body)
				tc.checkResult(&resp)
			}
		})
	}
}

// TestListAuthResources covers GET /consents/{consentId}/authorizations.
//
// The endpoint returns a plain JSON array (no wrapper object).
// Isolation: each sub-test gets a fresh org.
func (ts *AuthResourceAPITestSuite) TestListAuthResources() {
	type testCase struct {
		name string

		// setup creates any pre-conditions and returns the consentID to list against.
		setup func(orgID string) string

		// useAltOrg sends the list request under a different fresh org.
		useAltOrg bool

		omitOrgID     bool
		wantStatus    int
		wantErrorCode string
		checkResult   func(list []AuthResourceResponse)
	}

	cases := []testCase{

		// -----------------------------------------------------------------------
		// Happy path
		// -----------------------------------------------------------------------
		{
			name: "consent with no auth resources — returns empty array",
			setup: func(orgID string) string {
				return ts.mustCreateConsent(orgID, "grp-ar-list-empty")
			},
			wantStatus: http.StatusOK,
			checkResult: func(list []AuthResourceResponse) {
				ts.Empty(list, "must return [] when no auth resources exist")
			},
		},
		{
			name: "consent with one auth resource — returned in list",
			setup: func(orgID string) string {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-list-one")
				ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{
					UserID: strPtr("user-001"), Type: "accounts", Status: "APPROVED",
				})
				return consentID
			},
			wantStatus: http.StatusOK,
			checkResult: func(list []AuthResourceResponse) {
				ts.Require().Len(list, 1)
				ts.assertAuthResourceResponse(&list[0])
				ts.Equal("accounts", list[0].Type)
				ts.Equal("APPROVED", list[0].Status)
			},
		},
		{
			name: "consent with three auth resources — all returned with distinct IDs",
			setup: func(orgID string) string {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-list-multi")
				ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{UserID: strPtr("user-001"), Type: "a", Status: "APPROVED"})
				ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{UserID: strPtr("user-001"), Type: "b", Status: "CREATED"})
				ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{UserID: strPtr("user-001"), Type: "c", Status: "REJECTED"})
				return consentID
			},
			wantStatus: http.StatusOK,
			checkResult: func(list []AuthResourceResponse) {
				ts.Require().Len(list, 3)
				seen := map[string]bool{}
				for i := range list {
					ts.assertAuthResourceResponse(&list[i])
					ts.False(seen[list[i].ID], "all auth resource IDs must be distinct")
					seen[list[i].ID] = true
				}
			},
		},

		// -----------------------------------------------------------------------
		// Cross-org isolation
		// -----------------------------------------------------------------------
		{
			name: "auth resources created under org A — listing under org B returns empty array",
			setup: func(orgID string) string {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-list-org")
				ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{UserID: strPtr("user-001")})
				return consentID
			},
			useAltOrg:  true,
			wantStatus: http.StatusOK,
			checkResult: func(list []AuthResourceResponse) {
				ts.Empty(list, "org B must not see org A's auth resources")
			},
		},

		// -----------------------------------------------------------------------
		// Header errors
		// -----------------------------------------------------------------------
		{
			name: "missing org-id header → 400 AR-4007",
			setup: func(orgID string) string {
				return ts.mustCreateConsent(orgID, "grp-ar-list-hdr")
			},
			omitOrgID:     true,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "AR-4007",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()
			consentID := tc.setup(orgID)

			orgForReq := orgID
			switch {
			case tc.omitOrgID:
				orgForReq = ""
			case tc.useAltOrg:
				orgForReq = freshOrgID()
			}

			status, body := ts.doRequest(
				http.MethodGet,
				"/api/v1/consents/"+consentID+"/authorizations",
				orgForReq,
				nil,
			)

			ts.Equal(tc.wantStatus, status)
			if tc.wantErrorCode != "" {
				ts.assertAPIError(body, tc.wantErrorCode)
			} else if tc.checkResult != nil {
				var list []AuthResourceResponse
				ts.Require().NoError(json.Unmarshal(body, &list), "unmarshal list response: %s", body)
				tc.checkResult(list)
			}
		})
	}
}

// TestGetAuthResource_NotFoundUniformMessage verifies that all three AR-4040 conditions
// (non-existent ID, wrong consent, wrong org) return the same opaque description.
// This guards against the leaky messages that previously exposed internal state:
//   - "auth resource not found: {authID}"        — revealed resource existence
//   - "auth resource {authID} does not belong to consent {consentID}"  — confirmed existence
func (ts *AuthResourceAPITestSuite) TestGetAuthResource_NotFoundUniformMessage() {
	const wantDescription = "the authorization resource does not exist, does not belong to the specified consent, or is not accessible in this organization"

	type testCase struct {
		name  string
		setup func(orgID string) (consentID, authID, orgForReq string)
	}

	cases := []testCase{
		{
			name: "non-existent auth ID — description is opaque",
			setup: func(orgID string) (string, string, string) {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-msg-nf")
				return consentID, "00000000-0000-0000-0000-000000000000", orgID
			},
		},
		{
			name: "auth belongs to consent A but requested via consent B — description is opaque",
			setup: func(orgID string) (string, string, string) {
				consentA := ts.mustCreateConsent(orgID, "grp-ar-msg-ca")
				consentB := ts.mustCreateConsent(orgID, "grp-ar-msg-cb")
				ar := ts.mustCreateAuthResource(orgID, consentA, AuthResourceCreateRequest{UserID: strPtr("user-001")})
				return consentB, ar.ID, orgID
			},
		},
		{
			name: "auth exists but org-id header belongs to a different org — description is opaque",
			setup: func(orgID string) (string, string, string) {
				consentID := ts.mustCreateConsent(orgID, "grp-ar-msg-org")
				ar := ts.mustCreateAuthResource(orgID, consentID, AuthResourceCreateRequest{UserID: strPtr("user-001")})
				return consentID, ar.ID, freshOrgID()
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()
			consentID, authID, orgForReq := tc.setup(orgID)

			status, body := ts.doRequest(
				http.MethodGet,
				"/api/v1/consents/"+consentID+"/authorizations/"+authID,
				orgForReq,
				nil,
			)

			ts.Equal(http.StatusNotFound, status)
			errResp := ts.assertAPIError(body, "AR-4040")
			ts.Equal(wantDescription, errResp.Description,
				"all not-found conditions must return the same opaque description")
		})
	}
}
