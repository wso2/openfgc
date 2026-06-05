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

// TestGetConsent covers GET /consents/{consentId}.
func (ts *ConsentAPITestSuite) TestGetConsent() {
	type testCase struct {
		name string

		// setup creates the consent and returns (consentID, fetchOrgID).
		// fetchOrgID may differ from the setup orgID for isolation tests.
		setup func(orgID string) (consentID, fetchOrgID string)

		consentID  string // static consentID for error cases that skip setup
		omitOrgID  bool
		wantStatus int
		wantError  string
		checkResult func(resp *ConsentResponse)
	}

	cases := []testCase{
		// -----------------------------------------------------------------------
		// Happy paths
		// -----------------------------------------------------------------------
		{
			name: "returns full consent with all fields",
			setup: func(orgID string) (string, string) {
				c := ts.mustCreateConsent(orgID, "grp-get-full", ConsentCreateRequest{
					Type:                       "accounts",
					Frequency:                  intPtr(3),
					RecurringIndicator:         boolPtr(true),
					DataAccessValidityDuration: int64Ptr(86400000),
					Attributes:                 map[string]string{"k": "v"},
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Type: "accounts", Status: "APPROVED"},
					},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentResponse) {
				ts.Equal("accounts", resp.Type)
				ts.Require().NotNil(resp.Frequency)
				ts.Equal(3, *resp.Frequency)
				ts.Require().NotNil(resp.RecurringIndicator)
				ts.True(*resp.RecurringIndicator)
				ts.Require().NotNil(resp.DataAccessValidityDuration)
				ts.Equal(int64(86400000), *resp.DataAccessValidityDuration)
				ts.Len(resp.Attributes, 1)
				ts.Len(resp.Authorizations, 1)
			},
		},
		{
			name: "get returns same data as create",
			setup: func(orgID string) (string, string) {
				c := ts.mustCreateConsent(orgID, "grp-roundtrip", ConsentCreateRequest{
					Type: "payments",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-42", Type: "payments", Status: "APPROVED"},
					},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentResponse) {
				ts.Equal("payments", resp.Type)
				ts.Equal("ACTIVE", resp.Status)
				ts.Require().Len(resp.Authorizations, 1)
				ts.Equal("payments", resp.Authorizations[0].Type)
				ts.Require().NotNil(resp.Authorizations[0].UserID)
				ts.Equal("user-42", *resp.Authorizations[0].UserID)
			},
		},
		{
			name: "get consent with purpose — purpose and elements are returned",
			setup: func(orgID string) (string, string) {
				ts.mustCreateElement(orgID, "get-elem", "basic")
				ts.mustCreatePurpose(orgID, "get-purpose", "get-elem")
				c := ts.mustCreateConsent(orgID, "grp-get-purpose", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{
							Name:     "get-purpose",
							Elements: []ElementApprovalRequest{{Name: "get-elem", Approved: true}},
						},
					},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentResponse) {
				ts.Require().Len(resp.Purposes, 1)
				ts.Equal("get-purpose", resp.Purposes[0].Name)
				ts.Require().Len(resp.Purposes[0].Elements, 1)
				ts.Equal("get-elem", resp.Purposes[0].Elements[0].Name)
			},
		},
		{
			name: "org isolation — consent not visible across orgs → 404",
			setup: func(orgID string) (string, string) {
				c := ts.mustCreateConsent(orgID, "grp-iso", ConsentCreateRequest{Type: "accounts"})
				differentOrg := freshOrgID()
				return c.ID, differentOrg
			},
			wantStatus: http.StatusNotFound,
			wantError:  "CS-4040",
		},

		// -----------------------------------------------------------------------
		// Error cases
		// -----------------------------------------------------------------------
		{
			name:       "non-existent consent ID → 404 CS-4040",
			consentID:  "00000000-0000-0000-0000-000000000000",
			wantStatus: http.StatusNotFound,
			wantError:  "CS-4040",
		},
		{
			name:       "missing org-id header → 400 CS-4002",
			consentID:  "00000000-0000-0000-0000-000000000001",
			omitOrgID:  true,
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "malformed consent ID (not UUID) → 400 CS-4002",
			consentID:  "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()

			consentID := tc.consentID
			fetchOrgID := orgID
			if tc.setup != nil {
				consentID, fetchOrgID = tc.setup(orgID)
			}
			if tc.omitOrgID {
				fetchOrgID = ""
			}

			status, body := ts.doRequest(http.MethodGet, "/api/v1/consents/"+consentID, fetchOrgID, "", nil)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", body)

			if tc.wantError != "" {
				ts.assertAPIError(body, tc.wantError)
				return
			}

			var resp ConsentResponse
			ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentResponse: %s", body)
			if tc.checkResult != nil {
				tc.checkResult(&resp)
			}
		})
	}
}

// TestListConsents covers GET /consents.
func (ts *ConsentAPITestSuite) TestListConsents() {
	type testCase struct {
		name        string
		setup       func(orgID string)
		params      url.Values
		omitOrgID   bool
		wantStatus  int
		wantError   string
		checkResult func(resp *ConsentListResponse)
	}

	cases := []testCase{
		// -----------------------------------------------------------------------
		// Happy paths
		// -----------------------------------------------------------------------
		{
			name:       "empty org — returns empty data with correct metadata",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Empty(resp.Data)
				ts.Equal(0, resp.Metadata.Total)
				ts.Equal(0, resp.Metadata.Count)
				ts.Equal(0, resp.Metadata.Offset)
			},
		},
		{
			name: "single consent — returned in list",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-list-single", ConsentCreateRequest{Type: "accounts"})
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total)
				ts.Equal(1, resp.Metadata.Count)
				ts.Require().Len(resp.Data, 1)
				ts.Equal("accounts", resp.Data[0].Type)
			},
		},
		{
			name: "multiple consents — all returned in list",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-multi-1", ConsentCreateRequest{Type: "accounts"})
				ts.mustCreateConsent(orgID, "grp-multi-2", ConsentCreateRequest{Type: "payments"})
				ts.mustCreateConsent(orgID, "grp-multi-3", ConsentCreateRequest{Type: "investments"})
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(3, resp.Metadata.Total)
				ts.Len(resp.Data, 3)
			},
		},
		{
			name: "limit — returns only requested count",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-lim-1", ConsentCreateRequest{Type: "accounts"})
				ts.mustCreateConsent(orgID, "grp-lim-2", ConsentCreateRequest{Type: "payments"})
				ts.mustCreateConsent(orgID, "grp-lim-3", ConsentCreateRequest{Type: "investments"})
			},
			params:     url.Values{"limit": {"2"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(3, resp.Metadata.Total)
				ts.Equal(2, resp.Metadata.Count)
				ts.Len(resp.Data, 2)
			},
		},
		{
			name: "offset — skips the first N consents",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-off-1", ConsentCreateRequest{Type: "accounts"})
				ts.mustCreateConsent(orgID, "grp-off-2", ConsentCreateRequest{Type: "payments"})
			},
			params:     url.Values{"offset": {"1"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(2, resp.Metadata.Total)
				ts.Equal(1, resp.Metadata.Count)
				ts.Len(resp.Data, 1)
				ts.Equal(1, resp.Metadata.Offset)
			},
		},
		{
			name: "groupIds filter — returns only matching consents",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "group-alpha", ConsentCreateRequest{Type: "accounts"})
				ts.mustCreateConsent(orgID, "group-beta", ConsentCreateRequest{Type: "accounts"})
				ts.mustCreateConsent(orgID, "group-gamma", ConsentCreateRequest{Type: "accounts"})
			},
			params:     url.Values{"groupIds": {"group-alpha,group-beta"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(2, resp.Metadata.Total)
				for _, c := range resp.Data {
					ts.Contains([]string{"group-alpha", "group-beta"}, c.GroupID)
				}
			},
		},
		{
			name: "consentTypes filter — returns only matching types",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-ct-1", ConsentCreateRequest{Type: "accounts"})
				ts.mustCreateConsent(orgID, "grp-ct-2", ConsentCreateRequest{Type: "payments"})
				ts.mustCreateConsent(orgID, "grp-ct-3", ConsentCreateRequest{Type: "payments"})
			},
			params:     url.Values{"consentTypes": {"payments"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(2, resp.Metadata.Total)
				for _, c := range resp.Data {
					ts.Equal("payments", c.Type)
				}
			},
		},
		{
			name: "consentStatuses filter — returns only matching statuses",
			setup: func(orgID string) {
				// ACTIVE consent: all auths APPROVED
				ts.mustCreateConsent(orgID, "grp-cs-active", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				// CREATED consent: no auth
				ts.mustCreateConsent(orgID, "grp-cs-created", ConsentCreateRequest{Type: "accounts"})
			},
			params:     url.Values{"consentStatuses": {"ACTIVE"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total)
				ts.Require().Len(resp.Data, 1)
				ts.Equal("ACTIVE", resp.Data[0].Status)
			},
		},
		{
			name: "purposeName filter — returns only consents referencing that purpose",
			setup: func(orgID string) {
				ts.mustCreateElement(orgID, "list-filter-elem", "basic")
				ts.mustCreatePurpose(orgID, "list-filter-purpose", "list-filter-elem")
				ts.mustCreateConsent(orgID, "grp-pf-1", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{
							Name:     "list-filter-purpose",
							Elements: []ElementApprovalRequest{{Name: "list-filter-elem", Approved: true}},
						},
					},
				})
				// Consent without purpose — should not appear in filtered results
				ts.mustCreateConsent(orgID, "grp-pf-2", ConsentCreateRequest{Type: "accounts"})
			},
			params:     url.Values{"purposeName": {"list-filter-purpose"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total)
				ts.Require().Len(resp.Data, 1)
				ts.Require().NotEmpty(resp.Data[0].Purposes)
				ts.Equal("list-filter-purpose", resp.Data[0].Purposes[0].Name)
			},
		},

		// -----------------------------------------------------------------------
		// Filter validation errors
		// -----------------------------------------------------------------------
		{
			name:       "purposeVersion without purposeName → 400 CS-4002",
			params:     url.Values{"purposeVersion": {"v1"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "elementVersion without elementName or elementNamespace → 400 CS-4002",
			params:     url.Values{"elementVersion": {"v1"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},

		// -----------------------------------------------------------------------
		// Header errors
		// -----------------------------------------------------------------------
		{
			name:       "missing org-id header → 400 CS-4002",
			omitOrgID:  true,
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

			listOrgID := orgID
			if tc.omitOrgID {
				listOrgID = ""
			}

			path := "/api/v1/consents"
			if len(tc.params) > 0 {
				path += "?" + tc.params.Encode()
			}
			status, body := ts.doRequest(http.MethodGet, path, listOrgID, "", nil)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", body)

			if tc.wantError != "" {
				ts.assertAPIError(body, tc.wantError)
				return
			}

			var resp ConsentListResponse
			ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentListResponse: %s", body)
			if tc.checkResult != nil {
				tc.checkResult(&resp)
			}
		})
	}
}
