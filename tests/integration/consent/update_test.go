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

// TestUpdateConsent covers PUT /consents/{consentId}.
//
// Key semantics:
//   - Authorizations/Purposes/Attributes: nil = keep existing; non-nil (even empty) = replace all.
//   - Authorization IDs change after an update that includes authorizations (delete + re-create).
//   - group-id header must match the consent's existing groupId or be omitted (server uses the stored group).
func (ts *ConsentAPITestSuite) TestUpdateConsent() {
	type testCase struct {
		name string

		// setup creates the consent to update and returns (consentID, groupID).
		// The orgID is provided by the test loop.
		setup func(orgID string) (consentID, groupID string, created *ConsentResponse)

		req       ConsentUpdateRequest
		rawBody   string // used for parse errors (skips setup)
		consentID string // override for error cases that skip setup
		groupID   string // override group-id header (e.g. mismatch tests)
		omitOrgID bool

		wantStatus  int
		wantError   string
		checkResult func(orgID, consentID string, created *ConsentResponse, updated *ConsentResponse)
	}

	cases := []testCase{
		// -----------------------------------------------------------------------
		// Type update
		// -----------------------------------------------------------------------
		{
			name: "update type — new type is returned",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-type", ConsentCreateRequest{Type: "accounts"})
				return c.ID, "grp-upd-type", c
			},
			req:        ConsentUpdateRequest{Type: "payments"},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Equal("payments", updated.Type)
			},
		},

		// -----------------------------------------------------------------------
		// Optional scalar fields
		// -----------------------------------------------------------------------
		{
			name: "update frequency — new value returned",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-freq", ConsentCreateRequest{
					Type:      "accounts",
					Frequency: intPtr(1),
				})
				return c.ID, "grp-upd-freq", c
			},
			req:        ConsentUpdateRequest{Frequency: intPtr(10)},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Require().NotNil(updated.Frequency)
				ts.Equal(10, *updated.Frequency)
			},
		},
		{
			name: "update recurringIndicator — toggled value returned",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-ri", ConsentCreateRequest{
					Type:               "accounts",
					RecurringIndicator: boolPtr(false),
				})
				return c.ID, "grp-upd-ri", c
			},
			req:        ConsentUpdateRequest{RecurringIndicator: boolPtr(true)},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Require().NotNil(updated.RecurringIndicator)
				ts.True(*updated.RecurringIndicator)
			},
		},
		{
			name: "update dataAccessValidityDuration — new value returned",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-davd", ConsentCreateRequest{Type: "accounts"})
				return c.ID, "grp-upd-davd", c
			},
			req:        ConsentUpdateRequest{DataAccessValidityDuration: int64Ptr(7200000)},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Require().NotNil(updated.DataAccessValidityDuration)
				ts.Equal(int64(7200000), *updated.DataAccessValidityDuration)
			},
		},

		// -----------------------------------------------------------------------
		// Attributes — replace semantics
		// -----------------------------------------------------------------------
		{
			name: "update attributes — old attributes replaced by new ones",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-attr", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"old-key": "old-val"},
				})
				return c.ID, "grp-upd-attr", c
			},
			req:        ConsentUpdateRequest{Attributes: map[string]string{"new-key": "new-val"}},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Len(updated.Attributes, 1)
				ts.Equal("new-val", updated.Attributes["new-key"])
				ts.Empty(updated.Attributes["old-key"])
			},
		},
		{
			name: "update attributes with empty map — clears all attributes",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-attr-clear", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"k": "v"},
				})
				return c.ID, "grp-upd-attr-clear", c
			},
			// Type is required alongside empty Attributes so the validator's
			// "at least one field must be provided" check passes.
			req:        ConsentUpdateRequest{Type: "payments", Attributes: map[string]string{}},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Empty(updated.Attributes)
			},
		},

		// -----------------------------------------------------------------------
		// Authorizations — replace semantics
		// -----------------------------------------------------------------------
		{
			name: "update authorizations — old ones replaced, status re-derived",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-auth", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Type: "old", Status: "REJECTED"},
					},
				})
				return c.ID, "grp-upd-auth", c
			},
			req: ConsentUpdateRequest{
				Authorizations: []AuthorizationRequest{
					{UserID: "user-001", Type: "new", Status: "APPROVED"},
				},
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Require().Len(updated.Authorizations, 1)
				ts.Equal("new", updated.Authorizations[0].Type)
				ts.Equal("APPROVED", updated.Authorizations[0].Status)
				// Status re-derived from new auths: all APPROVED → ACTIVE
				ts.Equal("ACTIVE", updated.Status)
			},
		},
		{
			name: "update authorizations with empty slice — clears all auths, status becomes CREATED",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-auth-clear", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				return c.ID, "grp-upd-auth-clear", c
			},
			// Type is required alongside empty Authorizations so the validator's
			// "at least one field must be provided" check passes.
			req:        ConsentUpdateRequest{Type: "payments", Authorizations: []AuthorizationRequest{}},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Empty(updated.Authorizations)
				ts.Equal("CREATED", updated.Status)
			},
		},
		{
			name: "auth IDs change after update with authorizations (replace semantics)",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-auth-ids", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				return c.ID, "grp-upd-auth-ids", c
			},
			req: ConsentUpdateRequest{
				Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
			},
			wantStatus: http.StatusOK,
			checkResult: func(orgID, consentID string, original *ConsentResponse, updated *ConsentResponse) {
				ts.Require().Len(updated.Authorizations, 1)
				ts.Require().Len(original.Authorizations, 1)
				// New auth resource has a different ID from the original
				ts.NotEqual(original.Authorizations[0].ID, updated.Authorizations[0].ID,
					"update with authorizations must re-create auth resources with new IDs")
			},
		},

		// -----------------------------------------------------------------------
		// Purposes — replace semantics
		// -----------------------------------------------------------------------
		{
			name: "update purposes — old purposes replaced by new ones",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				ts.mustCreateElement(orgID, "upd-elem1", "basic")
				ts.mustCreateElement(orgID, "upd-elem2", "basic")
				ts.mustCreatePurpose(orgID, "upd-purpose1", "upd-elem1")
				ts.mustCreatePurpose(orgID, "upd-purpose2", "upd-elem2")
				c := ts.mustCreateConsent(orgID, "grp-upd-purpose", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{
							Name:     "upd-purpose1",
							Elements: []ElementApprovalRequest{{Name: "upd-elem1", Approved: true}},
						},
					},
				})
				return c.ID, "grp-upd-purpose", c
			},
			req: ConsentUpdateRequest{
				Purposes: []PurposeRefRequest{
					{
						Name:     "upd-purpose2",
						Elements: []ElementApprovalRequest{{Name: "upd-elem2", Approved: false}},
					},
				},
			},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Require().Len(updated.Purposes, 1)
				ts.Equal("upd-purpose2", updated.Purposes[0].Name)
			},
		},
		{
			name: "update purposes with empty slice — removes all purposes",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				ts.mustCreateElement(orgID, "upd-clr-elem", "basic")
				ts.mustCreatePurpose(orgID, "upd-clr-purpose", "upd-clr-elem")
				c := ts.mustCreateConsent(orgID, "grp-upd-purpose-clear", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{
							Name:     "upd-clr-purpose",
							Elements: []ElementApprovalRequest{{Name: "upd-clr-elem", Approved: true}},
						},
					},
				})
				return c.ID, "grp-upd-purpose-clear", c
			},
			// Type is required alongside empty Purposes so the validator's
			// "at least one field must be provided" check passes.
			req:        ConsentUpdateRequest{Type: "payments", Purposes: []PurposeRefRequest{}},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, _ *ConsentResponse, updated *ConsentResponse) {
				ts.Empty(updated.Purposes)
			},
		},

		// -----------------------------------------------------------------------
		// updatedTime advances
		// -----------------------------------------------------------------------
		{
			name: "updatedTime advances after update",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-upd-time", ConsentCreateRequest{Type: "accounts"})
				return c.ID, "grp-upd-time", c
			},
			req:        ConsentUpdateRequest{Type: "payments"},
			wantStatus: http.StatusOK,
			checkResult: func(_, _ string, created *ConsentResponse, updated *ConsentResponse) {
				// updatedTime must be >= createdTime (may be equal in fast tests at ms resolution)
				ts.GreaterOrEqual(updated.UpdatedTime, created.CreatedTime,
					"updatedTime must not precede createdTime")
				ts.Greater(updated.UpdatedTime, int64(946684800000),
					"updatedTime must be a Unix millisecond timestamp")
			},
		},

		// -----------------------------------------------------------------------
		// Group mismatch
		// -----------------------------------------------------------------------
		{
			name: "wrong group-id header → 400 CS-4002",
			setup: func(orgID string) (string, string, *ConsentResponse) {
				c := ts.mustCreateConsent(orgID, "grp-correct", ConsentCreateRequest{Type: "accounts"})
				return c.ID, "grp-wrong", c // intentional mismatch
			},
			req:        ConsentUpdateRequest{Type: "payments"},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},

		// -----------------------------------------------------------------------
		// Not-found / header / body errors
		// -----------------------------------------------------------------------
		{
			name:       "non-existent consent ID → 404 CS-4040",
			consentID:  "00000000-0000-0000-0000-000000000000",
			req:        ConsentUpdateRequest{Type: "payments"},
			wantStatus: http.StatusNotFound,
			wantError:  "CS-4040",
		},
		{
			name:       "missing org-id header → 400 CS-4002",
			consentID:  "00000000-0000-0000-0000-000000000001",
			omitOrgID:  true,
			req:        ConsentUpdateRequest{Type: "payments"},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "malformed consent ID → 400 CS-4002",
			consentID:  "not-a-uuid",
			req:        ConsentUpdateRequest{Type: "payments"},
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
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()

			consentID := tc.consentID
			groupID := tc.groupID
			var created *ConsentResponse
			if tc.setup != nil {
				consentID, groupID, created = tc.setup(orgID)
			}

			updateOrgID := orgID
			if tc.omitOrgID {
				updateOrgID = ""
			}

			var body any = tc.req
			if tc.rawBody != "" {
				body = tc.rawBody
			}

			status, respBody := ts.doRequest(http.MethodPut, "/api/v1/consents/"+consentID, updateOrgID, groupID, body)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", respBody)

			if tc.wantError != "" {
				ts.assertAPIError(respBody, tc.wantError)
				return
			}

			var resp ConsentResponse
			ts.Require().NoError(json.Unmarshal(respBody, &resp), "unmarshal ConsentResponse: %s", respBody)
			if tc.checkResult != nil {
				tc.checkResult(orgID, consentID, created, &resp)
			}
		})
	}
}
