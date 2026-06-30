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
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

// TestSearchConsents covers GET /consents with the advanced filter parameters
// that are not exercised by TestListConsents:
//   - userIds            — filter by authorization user ID
//   - fromTime / toTime  — time-window filter (Unix milliseconds against updatedTime)
//   - elementName        — filter by element name (exact match)
//   - elementNamespace   — filter by element namespace (exact match)
//   - elementVersion     — filter by element version (requires elementName or elementNamespace)
//   - purposeVersion     — filter by purpose version (requires purposeName)
//   - combined filters   — multiple filters together
func (ts *ConsentAPITestSuite) TestSearchConsents() {
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
		// userIds filter
		// -----------------------------------------------------------------------
		{
			name: "userIds filter — returns only consents whose auth has that userId",
			setup: func(orgID string) {
				// Consent with userId "alice"
				ts.mustCreateConsent(orgID, "grp-uid-1", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "alice", Type: "accounts", Status: "APPROVED"},
					},
				})
				// Consent with userId "bob" — must NOT appear
				ts.mustCreateConsent(orgID, "grp-uid-2", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "bob", Type: "accounts", Status: "APPROVED"},
					},
				})
			},
			params:     url.Values{"userIds": {"alice"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total)
				ts.Require().Len(resp.Data, 1)
				ts.Require().Len(resp.Data[0].Authorizations, 1)
				ts.Require().NotNil(resp.Data[0].Authorizations[0].UserID)
				ts.Equal("alice", *resp.Data[0].Authorizations[0].UserID)
			},
		},
		{
			name: "userIds filter — comma-separated list returns all matching",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-uids-1", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "alice", Type: "accounts", Status: "APPROVED"}},
				})
				ts.mustCreateConsent(orgID, "grp-uids-2", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "bob", Type: "accounts", Status: "APPROVED"}},
				})
				// charlie — not in filter, must NOT appear
				ts.mustCreateConsent(orgID, "grp-uids-3", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "charlie", Type: "accounts", Status: "APPROVED"}},
				})
			},
			params:     url.Values{"userIds": {"alice,bob"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(2, resp.Metadata.Total)
				for _, c := range resp.Data {
					ts.Require().Len(c.Authorizations, 1)
					ts.Require().NotNil(c.Authorizations[0].UserID)
					ts.Contains([]string{"alice", "bob"}, *c.Authorizations[0].UserID)
				}
			},
		},
		{
			name: "userIds filter — no match returns empty result",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-uid-nm", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "alice", Type: "accounts", Status: "APPROVED"}},
				})
			},
			params:     url.Values{"userIds": {"no-such-user"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(0, resp.Metadata.Total)
				ts.Empty(resp.Data)
			},
		},

		// -----------------------------------------------------------------------
		// fromTime / toTime time-window filter
		// The server compares against UPDATED_TIME which is stored in milliseconds.
		// -----------------------------------------------------------------------
		{
			name: "fromTime in the past — includes consents created after that time",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-ft-1", ConsentCreateRequest{Type: "accounts"})
			},
			// Use a fromTime 60 seconds in the past (in ms) — current consent is newer.
			params: url.Values{
				"fromTime": {fmt.Sprintf("%d", time.Now().Add(-60*time.Second).UnixMilli())},
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.GreaterOrEqual(resp.Metadata.Total, 1,
					"consent created after fromTime must be included")
			},
		},
		{
			name: "toTime in the past — excludes consents created after that time",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-tt-excl", ConsentCreateRequest{Type: "accounts"})
			},
			// Use a toTime 60 seconds ago — the consent we just created is newer, must be excluded.
			params: url.Values{
				"toTime": {fmt.Sprintf("%d", time.Now().Add(-60*time.Second).UnixMilli())},
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(0, resp.Metadata.Total,
					"consent created after toTime must NOT be included")
			},
		},
		{
			name: "fromTime + toTime window — includes only consents within range",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-window", ConsentCreateRequest{Type: "accounts"})
			},
			// Window: 2 minutes in the past → 2 minutes in the future.
			params: url.Values{
				"fromTime": {fmt.Sprintf("%d", time.Now().Add(-2*time.Minute).UnixMilli())},
				"toTime":   {fmt.Sprintf("%d", time.Now().Add(2*time.Minute).UnixMilli())},
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.GreaterOrEqual(resp.Metadata.Total, 1, "consent within the time window must be included")
			},
		},

		// -----------------------------------------------------------------------
		// elementName / elementNamespace / elementVersion filters
		// -----------------------------------------------------------------------
		{
			name: "elementName filter — returns only consents that include the named element",
			setup: func(orgID string) {
				ts.mustCreateElement(orgID, "search-elem-alpha", "basic")
				ts.mustCreateElement(orgID, "search-elem-beta", "basic")
				ts.mustCreatePurpose(orgID, "search-purp-alpha", "search-elem-alpha")
				ts.mustCreatePurpose(orgID, "search-purp-beta", "search-elem-beta")

				// Consent with alpha element
				ts.mustCreateConsent(orgID, "grp-en-1", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{Name: "search-purp-alpha", Elements: []ElementApprovalRequest{{Name: "search-elem-alpha", Approved: true}}},
					},
				})
				// Consent with beta element — must NOT appear in filter for alpha
				ts.mustCreateConsent(orgID, "grp-en-2", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{Name: "search-purp-beta", Elements: []ElementApprovalRequest{{Name: "search-elem-beta", Approved: false}}},
					},
				})
			},
			params:     url.Values{"elementName": {"search-elem-alpha"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total)
				ts.Require().Len(resp.Data, 1)
				ts.Require().NotEmpty(resp.Data[0].Purposes)
				ts.Equal("search-purp-alpha", resp.Data[0].Purposes[0].Name)
			},
		},
		{
			name: "elementNamespace filter — returns consents whose elements match the namespace",
			setup: func(orgID string) {
				ts.mustCreateElement(orgID, "search-ns-elem", "basic")
				ts.mustCreatePurpose(orgID, "search-ns-purp", "search-ns-elem")
				ts.mustCreateConsent(orgID, "grp-ns-1", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{Name: "search-ns-purp", Elements: []ElementApprovalRequest{{Name: "search-ns-elem", Approved: true}}},
					},
				})
				// Consent without any purpose — must NOT match namespace filter
				ts.mustCreateConsent(orgID, "grp-ns-2", ConsentCreateRequest{Type: "accounts"})
			},
			// Elements created without an explicit namespace get "default"
			params:     url.Values{"elementNamespace": {"default"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.GreaterOrEqual(resp.Metadata.Total, 1)
				// All returned consents must have at least one purpose
				for _, c := range resp.Data {
					ts.NotEmpty(c.Purposes, "consents returned for namespace filter must have purposes")
				}
			},
		},
		{
			name: "elementVersion filter with elementName — returns consents matching version",
			setup: func(orgID string) {
				ts.mustCreateElement(orgID, "search-ev-elem", "basic")
				ts.mustCreatePurpose(orgID, "search-ev-purp", "search-ev-elem")
				// Consent bound to v1 of the element (no pin — resolved to latest = v1)
				ts.mustCreateConsent(orgID, "grp-ev-1", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{Name: "search-ev-purp", Elements: []ElementApprovalRequest{{Name: "search-ev-elem", Approved: true}}},
					},
				})
			},
			params:     url.Values{"elementName": {"search-ev-elem"}, "elementVersion": {"v1"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total, "consent bound to v1 of element must be returned")
			},
		},
		{
			name: "elementVersion filter — v2 filter returns nothing when only v1 consent exists",
			setup: func(orgID string) {
				ts.mustCreateElement(orgID, "search-ev2-elem", "basic")
				ts.mustCreatePurpose(orgID, "search-ev2-purp", "search-ev2-elem")
				// Consent bound to v1 of the element
				ts.mustCreateConsent(orgID, "grp-ev2-1", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{Name: "search-ev2-purp", Elements: []ElementApprovalRequest{{Name: "search-ev2-elem", Approved: true}}},
					},
				})
			},
			// Filter for v2 — no consents were bound to v2
			params:     url.Values{"elementName": {"search-ev2-elem"}, "elementVersion": {"v2"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(0, resp.Metadata.Total, "no consents bound to v2 — must return empty")
			},
		},

		// -----------------------------------------------------------------------
		// purposeVersion filter (requires purposeName)
		// -----------------------------------------------------------------------
		{
			name: "purposeVersion filter with purposeName — returns consents at that purpose version",
			setup: func(orgID string) {
				ts.mustCreateElement(orgID, "search-pv-elem", "basic")
				ts.mustCreatePurpose(orgID, "search-pv-purp", "search-pv-elem")
				// Consent bound to v1 of the purpose (created with no version pin → resolves to v1)
				ts.mustCreateConsent(orgID, "grp-pv-1", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{Name: "search-pv-purp", Elements: []ElementApprovalRequest{{Name: "search-pv-elem", Approved: true}}},
					},
				})
			},
			params:     url.Values{"purposeName": {"search-pv-purp"}, "purposeVersion": {"v1"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total, "consent bound to v1 of purpose must be returned")
				ts.Equal("search-pv-purp", resp.Data[0].Purposes[0].Name)
			},
		},
		{
			name: "purposeVersion filter — v2 filter returns nothing when only v1 consent exists",
			setup: func(orgID string) {
				ts.mustCreateElement(orgID, "search-pv2-elem", "basic")
				ts.mustCreatePurpose(orgID, "search-pv2-purp", "search-pv2-elem")
				ts.mustCreateConsent(orgID, "grp-pv2-1", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{Name: "search-pv2-purp", Elements: []ElementApprovalRequest{{Name: "search-pv2-elem", Approved: true}}},
					},
				})
			},
			params:     url.Values{"purposeName": {"search-pv2-purp"}, "purposeVersion": {"v2"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(0, resp.Metadata.Total, "no consents bound to purpose v2 — must return empty")
			},
		},

		// -----------------------------------------------------------------------
		// Combined filters
		// -----------------------------------------------------------------------
		{
			name: "combined consentTypes + groupIds — only matching consents returned",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-combo-1", ConsentCreateRequest{Type: "accounts"})
				ts.mustCreateConsent(orgID, "grp-combo-2", ConsentCreateRequest{Type: "payments"})
				ts.mustCreateConsent(orgID, "grp-combo-3", ConsentCreateRequest{Type: "accounts"})
			},
			params: url.Values{
				"consentTypes": {"accounts"},
				"groupIds":     {"grp-combo-1"},
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total)
				ts.Require().Len(resp.Data, 1)
				ts.Equal("accounts", resp.Data[0].Type)
				ts.Equal("grp-combo-1", resp.Data[0].GroupID)
			},
		},
		{
			name: "combined elementName + consentStatuses — must match both conditions",
			setup: func(orgID string) {
				ts.mustCreateElement(orgID, "search-combo-elem", "basic")
				ts.mustCreatePurpose(orgID, "search-combo-purp", "search-combo-elem")
				// ACTIVE consent with element
				ts.mustCreateConsent(orgID, "grp-combo-active", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Type: "accounts", Status: "APPROVED"},
					},
					Purposes: []PurposeRefRequest{
						{Name: "search-combo-purp", Elements: []ElementApprovalRequest{{Name: "search-combo-elem", Approved: true}}},
					},
				})
				// CREATED consent with element (no auth → CREATED status)
				ts.mustCreateConsent(orgID, "grp-combo-created", ConsentCreateRequest{
					Type: "accounts",
					Purposes: []PurposeRefRequest{
						{Name: "search-combo-purp", Elements: []ElementApprovalRequest{{Name: "search-combo-elem", Approved: true}}},
					},
				})
			},
			params: url.Values{
				"elementName":     {"search-combo-elem"},
				"consentStatuses": {"ACTIVE"},
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total,
					"only the ACTIVE consent with that element must be returned")
				ts.Equal("ACTIVE", resp.Data[0].Status)
			},
		},

		// -----------------------------------------------------------------------
		// Org isolation
		// -----------------------------------------------------------------------
		{
			name: "org isolation — search results are scoped to the requesting org",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-iso-search", ConsentCreateRequest{Type: "accounts"})
				// Create the same type of consent in a different org — must NOT appear.
				differentOrg := freshOrgID()
				ts.mustCreateConsent(differentOrg, "grp-iso-other", ConsentCreateRequest{Type: "accounts"})
			},
			params:     url.Values{"consentTypes": {"accounts"}},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentListResponse) {
				ts.Equal(1, resp.Metadata.Total,
					"only consents from the requesting org must be returned")
			},
		},

		// -----------------------------------------------------------------------
		// Dependent-parameter validation
		// -----------------------------------------------------------------------
		{
			name:       "elementVersion without elementName or elementNamespace → 400 CS-4002",
			params:     url.Values{"elementVersion": {"v1"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "elementVersion with elementName → 200 (dependency satisfied)",
			params:     url.Values{"elementVersion": {"v1"}, "elementName": {"any-elem"}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "elementVersion with elementNamespace → 200 (dependency satisfied)",
			params:     url.Values{"elementVersion": {"v1"}, "elementNamespace": {"default"}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "purposeVersion without purposeName → 400 CS-4002",
			params:     url.Values{"purposeVersion": {"v1"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "purposeVersion with purposeName → 200 (dependency satisfied)",
			params:     url.Values{"purposeVersion": {"v1"}, "purposeName": {"any-purpose"}},
			wantStatus: http.StatusOK,
		},

		// -----------------------------------------------------------------------
		// Header validation
		// -----------------------------------------------------------------------
		{
			name:       "missing org-id → 400 CS-4002",
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

			searchOrgID := orgID
			if tc.omitOrgID {
				searchOrgID = ""
			}

			status, body := ts.doSearchConsents(searchOrgID, tc.params)
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

// TestSearchConsentsByAttribute covers GET /consents/attributes.
//
// Contract:
//   - key is required; value is optional.
//   - By key only: returns all consent IDs that have any value for that attribute key.
//   - By key + value: returns only consent IDs with an exact key-value match.
//   - Results are sorted alphabetically by consent ID and include a count field.
//   - Results are scoped to the requesting org.
//   - Missing key → 400 CS-4002.
//   - Missing org-id → 400 CS-4002.
func (ts *ConsentAPITestSuite) TestSearchConsentsByAttribute() {
	type testCase struct {
		name        string
		setup       func(orgID string)
		key         string
		value       string // empty = omit value param (key-only search)
		omitOrgID   bool
		wantStatus  int
		wantError   string
		checkResult func(resp *ConsentAttributeSearchResponse)
	}

	cases := []testCase{
		// -----------------------------------------------------------------------
		// Key-only search
		// -----------------------------------------------------------------------
		{
			name: "key only — returns all consent IDs that have that attribute key",
			setup: func(orgID string) {
				c1 := ts.mustCreateConsent(orgID, "grp-attr-1", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"department": "sales", "region": "us"},
				})
				c2 := ts.mustCreateConsent(orgID, "grp-attr-2", ConsentCreateRequest{
					Type:       "payments",
					Attributes: map[string]string{"department": "engineering"},
				})
				// Store IDs as test context via closure — just verify count
				_ = c1
				_ = c2
				// Consent without the "department" key — must NOT appear
				ts.mustCreateConsent(orgID, "grp-attr-3", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"tier": "gold"},
				})
			},
			key:        "department",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Equal(2, resp.Count)
				ts.Require().Len(resp.ConsentIDs, 2)
			},
		},
		{
			name: "key + value — returns only exact key-value matches",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-kv-1", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"env": "prod"},
				})
				// Different value — must NOT appear
				ts.mustCreateConsent(orgID, "grp-kv-2", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"env": "staging"},
				})
				// Key absent entirely — must NOT appear
				ts.mustCreateConsent(orgID, "grp-kv-3", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"other": "prod"},
				})
			},
			key:        "env",
			value:      "prod",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Equal(1, resp.Count)
				ts.Require().Len(resp.ConsentIDs, 1)
			},
		},
		{
			name: "key + value — multiple consents with exact same key-value are all returned",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-multi-attr-1", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"tier": "gold"},
				})
				ts.mustCreateConsent(orgID, "grp-multi-attr-2", ConsentCreateRequest{
					Type:       "payments",
					Attributes: map[string]string{"tier": "gold"},
				})
				ts.mustCreateConsent(orgID, "grp-multi-attr-3", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"tier": "silver"},
				})
			},
			key:        "tier",
			value:      "gold",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Equal(2, resp.Count)
				ts.Require().Len(resp.ConsentIDs, 2)
			},
		},
		{
			name: "key matches but value does not — empty result",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-val-miss", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"color": "blue"},
				})
			},
			key:        "color",
			value:      "red", // not stored
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Equal(0, resp.Count)
				ts.Empty(resp.ConsentIDs)
			},
		},
		{
			name: "key does not exist in org — empty result",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-no-key", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"something": "else"},
				})
			},
			key:        "nonexistent-key-xyz",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Equal(0, resp.Count)
				ts.Empty(resp.ConsentIDs)
			},
		},
		{
			name: "consent without attributes — not returned for any key search",
			setup: func(orgID string) {
				// No attributes at all
				ts.mustCreateConsent(orgID, "grp-no-attrs", ConsentCreateRequest{Type: "accounts"})
			},
			key:        "anything",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Equal(0, resp.Count)
				ts.Empty(resp.ConsentIDs)
			},
		},

		// -----------------------------------------------------------------------
		// Result ordering
		// -----------------------------------------------------------------------
		{
			name: "results are sorted alphabetically by consent ID",
			setup: func(orgID string) {
				// Create three consents with the same attribute key
				ts.mustCreateConsent(orgID, "grp-sort-1", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"sortkey": "val"},
				})
				ts.mustCreateConsent(orgID, "grp-sort-2", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"sortkey": "val"},
				})
				ts.mustCreateConsent(orgID, "grp-sort-3", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"sortkey": "val"},
				})
			},
			key:        "sortkey",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Require().Len(resp.ConsentIDs, 3)
				sorted := make([]string, len(resp.ConsentIDs))
				copy(sorted, resp.ConsentIDs)
				sort.Strings(sorted)
				ts.Equal(sorted, resp.ConsentIDs, "consent IDs must be sorted alphabetically")
			},
		},

		// -----------------------------------------------------------------------
		// Org isolation
		// -----------------------------------------------------------------------
		{
			name: "org isolation — returns only consents from the requesting org",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "grp-iso-attr-1", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"scope": "internal"},
				})
				// Same attribute in a different org — must NOT appear
				differentOrg := freshOrgID()
				ts.mustCreateConsent(differentOrg, "grp-iso-attr-other", ConsentCreateRequest{
					Type:       "accounts",
					Attributes: map[string]string{"scope": "internal"},
				})
			},
			key:        "scope",
			value:      "internal",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Equal(1, resp.Count,
					"only consents from the requesting org must be returned")
			},
		},

		// -----------------------------------------------------------------------
		// count field matches len(consentIds)
		// -----------------------------------------------------------------------
		{
			name: "count field equals length of consentIds array",
			setup: func(orgID string) {
				for i := 0; i < 4; i++ {
					ts.mustCreateConsent(orgID, fmt.Sprintf("grp-count-%d", i), ConsentCreateRequest{
						Type:       "accounts",
						Attributes: map[string]string{"batch": "run1"},
					})
				}
			},
			key:        "batch",
			value:      "run1",
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentAttributeSearchResponse) {
				ts.Equal(resp.Count, len(resp.ConsentIDs),
					"count field must equal the length of consentIds")
				ts.Equal(4, resp.Count)
			},
		},

		// -----------------------------------------------------------------------
		// Validation errors
		// -----------------------------------------------------------------------
		{
			name:       "missing key parameter → 400 CS-4002",
			key:        "", // doSearchByAttribute treats "" as omitting the key param
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "missing org-id header → 400 CS-4002",
			key:        "any",
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

			searchOrgID := orgID
			if tc.omitOrgID {
				searchOrgID = ""
			}

			status, body := ts.doSearchByAttribute(searchOrgID, tc.key, tc.value)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", body)

			if tc.wantError != "" {
				ts.assertAPIError(body, tc.wantError)
				return
			}

			var resp ConsentAttributeSearchResponse
			ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentAttributeSearchResponse: %s", body)
			if tc.checkResult != nil {
				tc.checkResult(&resp)
			}
		})
	}
}

// TestGetGroupIDsByUserID covers GET /consents/group-ids.
//
// Contract:
//   - userId is required and exactly one value must be supplied.
//   - Returns distinct group IDs associated with the specified user.
//   - Results are scoped to the requesting org.
//   - Results are sorted alphabetically by group ID.
//   - Missing userId, repeated userId, or missing org-id -> 400 CS-4002.
func (ts *ConsentAPITestSuite) TestGetGroupIDsByUserID() {
	type testCase struct {
		name        string
		setup       func(orgID string)
		userIDs     []string
		omitOrgID   bool
		wantStatus  int
		wantError   string
		checkResult func(resp *ConsentGroupIDsResponse)
	}

	cases := []testCase{
		{
			name: "one user across multiple groups returns distinct group IDs",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "group-gamma", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-multi-001", Status: "APPROVED"}},
				})
				ts.mustCreateConsent(orgID, "group-alpha", ConsentCreateRequest{
					Type:           "payments",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-multi-001", Status: "APPROVED"}},
				})
				ts.mustCreateConsent(orgID, "group-beta", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-multi-001", Status: "APPROVED"}},
				})
			},
			userIDs:    []string{"gid-user-multi-001"},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentGroupIDsResponse) {
				ts.Equal(3, resp.Count)
				ts.Equal([]string{"group-alpha", "group-beta", "group-gamma"}, resp.GroupIDs)
			},
		},
		{
			name: "duplicate group IDs across multiple consents are returned once",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "group-dup", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-dup-001", Status: "APPROVED"}},
				})
				ts.mustCreateConsent(orgID, "group-dup", ConsentCreateRequest{
					Type:           "payments",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-dup-001", Status: "APPROVED"}},
				})
				ts.mustCreateConsent(orgID, "group-other", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-dup-001", Status: "APPROVED"}},
				})
			},
			userIDs:    []string{"gid-user-dup-001"},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentGroupIDsResponse) {
				ts.Equal(2, resp.Count)
				ts.Equal([]string{"group-dup", "group-other"}, resp.GroupIDs)
			},
		},
		{
			name: "user with no matching authorizations returns empty result",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "group-alpha", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-other-001", Status: "APPROVED"}},
				})
			},
			userIDs:    []string{"gid-user-none-001"},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentGroupIDsResponse) {
				ts.Equal(0, resp.Count)
				ts.Empty(resp.GroupIDs)
			},
		},
		{
			name: "org isolation returns only groups from the requesting org",
			setup: func(orgID string) {
				ts.mustCreateConsent(orgID, "group-alpha", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-org-001", Status: "APPROVED"}},
				})
				differentOrg := freshOrgID()
				ts.mustCreateConsent(differentOrg, "group-foreign", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "gid-user-org-001", Status: "APPROVED"}},
				})
			},
			userIDs:    []string{"gid-user-org-001"},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentGroupIDsResponse) {
				ts.Equal(1, resp.Count)
				ts.Equal([]string{"group-alpha"}, resp.GroupIDs)
			},
		},
		{
			name:       "missing userId parameter -> 400 CS-4002",
			userIDs:    nil,
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "repeated userId parameters -> 400 CS-4002",
			userIDs:    []string{"gid-user-repeat-001", "gid-user-repeat-002"},
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "missing org-id header -> 400 CS-4002",
			userIDs:    []string{"gid-user-header-001"},
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

			requestOrgID := orgID
			if tc.omitOrgID {
				requestOrgID = ""
			}

			status, body := ts.doGetGroupIDsByUserID(requestOrgID, tc.userIDs)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", body)

			if tc.wantError != "" {
				ts.assertAPIError(body, tc.wantError)
				return
			}

			var resp ConsentGroupIDsResponse
			ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentGroupIDsResponse: %s", body)
			if tc.checkResult != nil {
				tc.checkResult(&resp)
			}
		})
	}
}
