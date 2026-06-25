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
	"net/url"
)

// =============================================================================
// TestCreateConsentDelegation covers POST /consents with delegation auth types.
// =============================================================================

func (ts *ConsentAPITestSuite) TestCreateConsentDelegation() {
	type testCase struct {
		name          string
		groupID       string
		buildBody     func(orgID string) any
		wantStatus    int
		wantErrorCode string
		checkResult   func(resp *ConsentResponse)
	}

	cases := []testCase{

		{
			name:    "delegation: delegate + delegate_subject with RECORDED → ACTIVE",
			groupID: "grp-deleg-1",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(resp *ConsentResponse) {
				ts.Equal("ACTIVE", resp.Status,
					"delegate APPROVED + delegate_subject RECORDED → consent ACTIVE")
				ts.Require().Len(resp.Authorizations, 2)
			},
		},
		{
			name:    "delegation: delegate CREATED + delegate_subject RECORDED → CREATED",
			groupID: "grp-deleg-pending",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "CREATED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(resp *ConsentResponse) {
				ts.Equal("CREATED", resp.Status,
					"delegate CREATED + delegate_subject RECORDED → consent stays CREATED")
			},
		},
		{
			name:    "self-consent: primary with APPROVED → ACTIVE",
			groupID: "grp-primary",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-111", Type: "primary", Status: "APPROVED"},
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(resp *ConsentResponse) {
				ts.Equal("ACTIVE", resp.Status)
				ts.Require().Len(resp.Authorizations, 1)
				ts.Equal("primary", resp.Authorizations[0].Type)
			},
		},
		{
			name:    "custom types: owner APPROVED + agent RECORDED → ACTIVE",
			groupID: "grp-custom",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-111", Type: "owner", Status: "APPROVED"},
						{UserID: "agent-ai", Type: "agent", Status: "RECORDED"},
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(resp *ConsentResponse) {
				ts.Equal("ACTIVE", resp.Status,
					"custom types: owner APPROVED + agent RECORDED → ACTIVE")
				ts.Require().Len(resp.Authorizations, 2)
			},
		},
		{
			name:    "multiple delegates + multiple subjects → ACTIVE",
			groupID: "grp-multi-deleg",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "parent-1", Type: "delegate", Status: "APPROVED"},
						{UserID: "parent-2", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-1", Type: "delegate_subject", Status: "RECORDED"},
						{UserID: "child-2", Type: "delegate_subject", Status: "RECORDED"},
					},
				}
			},
			wantStatus: http.StatusCreated,
			checkResult: func(resp *ConsentResponse) {
				ts.Equal("ACTIVE", resp.Status)
				ts.Require().Len(resp.Authorizations, 4)
			},
		},

		// -----------------------------------------------------------------
		// Validation errors
		// -----------------------------------------------------------------
		{
			name:    "delegate without delegate_subject → 400",
			groupID: "grp-deleg-err-1",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
					},
				}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CS-4002",
		},
		{
			name:    "delegate_subject without delegate → 400",
			groupID: "grp-deleg-err-2",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CS-4002",
		},
		{
			name:    "primary mixed with delegate → 400",
			groupID: "grp-deleg-err-3",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-111", Type: "primary", Status: "APPROVED"},
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CS-4002",
		},
		{
			name:    "all RECORDED (agent only) → 400",
			groupID: "grp-deleg-err-4",
			buildBody: func(_ string) any {
				return ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "agent-ai", Type: "agent", Status: "RECORDED"},
					},
				}
			},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "CS-4002",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()
			body := tc.buildBody(orgID)

			status, respBody := ts.doCreateConsentRaw(orgID, tc.groupID, body)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", respBody)

			if tc.wantErrorCode != "" {
				ts.assertAPIError(respBody, tc.wantErrorCode)
				return
			}

			var resp ConsentResponse
			ts.Require().NoError(json.Unmarshal(respBody, &resp))
			if tc.checkResult != nil {
				tc.checkResult(&resp)
			}
		})
	}
}

// =============================================================================
// TestSearchConsentsDelegation covers GET /consents with delegation query params.
// =============================================================================

func (ts *ConsentAPITestSuite) TestSearchConsentsDelegation() {
	type testCase struct {
		name        string
		setup       func(orgID string)
		params      url.Values
		wantStatus  int
		wantError   string
		checkResult func(resp *ConsentListResponse)
	}

	cases := []testCase{

		// -----------------------------------------------------------------
		// delegation=false — self-consents only
		// -----------------------------------------------------------------
		{
			name: "delegation=false — returns only self-consents for the user",
			setup: func(orgID string) {
				// Father's self-consent (primary)
				ts.mustCreateConsent(orgID, "grp-self-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "primary", Status: "APPROVED"},
					},
				})
				// Father is a delegate on another consent — must NOT appear
				ts.mustCreateConsent(orgID, "grp-deleg-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
			},
			params:     url.Values{"delegation": {"false"}, "userIds": {"father-111"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total, "only self-consent should be returned")
				ts.Require().Len(resp.Data, 1)
				// The returned consent should have a primary auth for father
				found := false
				for _, auth := range resp.Data[0].Authorizations {
					if auth.UserID != nil && *auth.UserID == "father-111" && auth.Type == "primary" {
						found = true
					}
				}
				ts.True(found, "returned consent must have father as primary")
			},
		},

		// -----------------------------------------------------------------
		// delegation=true — delegation consents
		// -----------------------------------------------------------------
		{
			name: "delegation=true — returns consents where user is a delegate",
			setup: func(orgID string) {
				// Father's self-consent — must NOT appear
				ts.mustCreateConsent(orgID, "grp-self-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "primary", Status: "APPROVED"},
					},
				})
				// Father is a delegate
				ts.mustCreateConsent(orgID, "grp-deleg-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
			},
			params:     url.Values{"delegation": {"true"}, "userIds": {"father-111"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total, "only delegation consent should be returned")
				ts.Require().Len(resp.Data, 1)
				// The returned consent should have father as delegate
				found := false
				for _, auth := range resp.Data[0].Authorizations {
					if auth.UserID != nil && *auth.UserID == "father-111" && auth.Type == "delegate" {
						found = true
					}
				}
				ts.True(found, "returned consent must have father as delegate")
			},
		},

		// -----------------------------------------------------------------
		// delegateSubject — consents about a subject's data
		// -----------------------------------------------------------------
		{
			name: "delegateSubject — returns consents where this user is the subject",
			setup: func(orgID string) {
				// Consent about child-333's data
				ts.mustCreateConsent(orgID, "grp-ds-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
				// Consent about child-444's data — must NOT appear
				ts.mustCreateConsent(orgID, "grp-ds-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "mother-222", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-444", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
			},
			params:     url.Values{"delegateSubject": {"child-333"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total, "only consents about child-333 should be returned")
			},
		},

		// -----------------------------------------------------------------
		// Combined: delegation=true + delegateSubject
		// -----------------------------------------------------------------
		{
			name: "delegation=true + delegateSubject — father manages son's consent",
			setup: func(orgID string) {
				// Father delegates for child-333
				ts.mustCreateConsent(orgID, "grp-combo-ds-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
				// Mother delegates for child-333 — must NOT appear (different delegate)
				ts.mustCreateConsent(orgID, "grp-combo-ds-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "mother-222", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
			},
			params: url.Values{
				"delegation":      {"true"},
				"userIds":         {"father-111"},
				"delegateSubject": {"child-333"},
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total,
					"only consent where father is delegate AND child-333 is subject")
			},
		},

		// -----------------------------------------------------------------
		// authTypes — custom type filtering
		// -----------------------------------------------------------------
		{
			name: "authTypes=agent — returns consents with agent auth type",
			setup: func(orgID string) {
				// Consent with agent
				ts.mustCreateConsent(orgID, "grp-at-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-111", Type: "owner", Status: "APPROVED"},
						{UserID: "agent-ai", Type: "agent", Status: "RECORDED"},
					},
				})
				// Consent without agent — must NOT appear
				ts.mustCreateConsent(orgID, "grp-at-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-222", Type: "primary", Status: "APPROVED"},
					},
				})
			},
			params:     url.Values{"authTypes": {"agent"}, "userIds": {"agent-ai"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total, "only consent with agent-ai as agent")
			},
		},

		// -----------------------------------------------------------------
		// delegation=true without userIds — all delegation consents in system
		// -----------------------------------------------------------------
		{
			name: "delegation=true without userIds — all delegation consents",
			setup: func(orgID string) {
				// Delegation consent
				ts.mustCreateConsent(orgID, "grp-all-deleg-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
				// Self-consent — must NOT appear
				ts.mustCreateConsent(orgID, "grp-all-self-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-111", Type: "primary", Status: "APPROVED"},
					},
				})
			},
			params:     url.Values{"delegation": {"true"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total, "only delegation consent should appear")
			},
		},

		// -----------------------------------------------------------------
		// delegation=false without userIds — all self-consents in system
		// -----------------------------------------------------------------
		{
			name: "delegation=false without userIds — all self-consents",
			setup: func(orgID string) {
				// Self-consent
				ts.mustCreateConsent(orgID, "grp-all-self-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-111", Type: "primary", Status: "APPROVED"},
					},
				})
				// Delegation consent — must NOT appear
				ts.mustCreateConsent(orgID, "grp-all-deleg-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
			},
			params:     url.Values{"delegation": {"false"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total, "only self-consent should appear")
			},
		},

		// -----------------------------------------------------------------
		// userIds without delegation — all consents involving user (any role)
		// -----------------------------------------------------------------
		{
			name: "userIds without delegation — returns all consents involving the user",
			setup: func(orgID string) {
				// Father's self-consent
				ts.mustCreateConsent(orgID, "grp-all-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "primary", Status: "APPROVED"},
					},
				})
				// Father as delegate
				ts.mustCreateConsent(orgID, "grp-all-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
						{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
					},
				})
				// Unrelated consent — must NOT appear
				ts.mustCreateConsent(orgID, "grp-all-3", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "other-user", Type: "primary", Status: "APPROVED"},
					},
				})
			},
			params:     url.Values{"userIds": {"father-111"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(2, resp.Metadata.Total,
					"both self-consent and delegation consent should appear")
			},
		},

		// -----------------------------------------------------------------
		// Validation errors
		// -----------------------------------------------------------------
		{
			name:       "delegation invalid value → 400",
			params:     url.Values{"delegation": {"maybe"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "delegation + authTypes together → 400 (mutually exclusive)",
			params:     url.Values{"delegation": {"true"}, "authTypes": {"agent"}, "userIds": {"user-1"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()

			if tc.setup != nil {
				tc.setup(orgID)
			}

			status, body := ts.doSearchConsents(orgID, tc.params)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", body)

			if tc.wantError != "" {
				ts.assertAPIError(body, tc.wantError)
				return
			}

			var resp ConsentListResponse
			ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal: %s", body)
			if tc.checkResult != nil {
				tc.checkResult(&resp)
			}
		})
	}
}
