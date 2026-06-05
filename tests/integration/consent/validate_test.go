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

// TestValidateConsent covers POST /consents/validate.
//
// Contract:
//   - The endpoint always returns HTTP 200 when the consent is found.
//     The isValid field in the body indicates success or failure.
//   - consentInformation is always present in the response body, regardless of validity.
//   - ACTIVE consent with all mandatory elements approved → isValid=true.
//   - Non-ACTIVE status (CREATED, REVOKED, EXPIRED) → isValid=false, errorCode=401.
//   - Mandatory element not approved → isValid=false, errorCode=403.
//   - Expired consent → auto-expired on validate, then isValid=false, errorCode=401.
//   - consentInformation elements include enriched fields: type.
//   - Missing or bad input → 400 CS-4002 / CS-4001.
//   - Non-existent consentId → 404 CS-4040.
func (ts *ConsentAPITestSuite) TestValidateConsent() {
	type testCase struct {
		name string

		// setup creates the consent and returns (consentID, orgID used for validate call).
		// orgID is the freshOrgID created for this test; setup may use a different org for
		// isolation tests.
		setup func(orgID string) (consentID, validateOrgID string)

		// rawBody is used for static error-cases that skip setup.
		rawBody   string
		omitOrgID bool

		wantStatus  int
		wantError   string
		checkResult func(resp *ConsentValidateResponse)
	}

	cases := []testCase{
		// -----------------------------------------------------------------------
		// Valid — ACTIVE consent
		// -----------------------------------------------------------------------
		{
			name: "ACTIVE consent → isValid=true, consentInformation present",
			setup: func(orgID string) (string, string) {
				c := ts.mustCreateConsent(orgID, "grp-val-active", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Type: "accounts", Status: "APPROVED"},
					},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentValidateResponse) {
				ts.True(resp.IsValid, "ACTIVE consent must be valid")
				ts.Equal(0, resp.ErrorCode, "no errorCode for valid consent")
				ts.Empty(resp.ErrorMessage)
				ts.Require().NotNil(resp.ConsentInfo, "consentInformation must always be present")
				ts.Equal("ACTIVE", resp.ConsentInfo.Status)
			},
		},
		{
			name: "consentInformation contains all top-level consent fields",
			setup: func(orgID string) (string, string) {
				c := ts.mustCreateConsent(orgID, "grp-val-fields", ConsentCreateRequest{
					Type:                       "payments",
					Frequency:                  intPtr(5),
					RecurringIndicator:         boolPtr(true),
					DataAccessValidityDuration: int64Ptr(3600000),
					Attributes:                 map[string]string{"k": "v"},
					Authorizations:             []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentValidateResponse) {
				ts.True(resp.IsValid)
				ts.Require().NotNil(resp.ConsentInfo)
				info := resp.ConsentInfo
				ts.Equal("payments", info.Type)
				ts.Equal("ACTIVE", info.Status)
				ts.Require().NotNil(info.Frequency)
				ts.Equal(5, *info.Frequency)
				ts.Require().NotNil(info.RecurringIndicator)
				ts.True(*info.RecurringIndicator)
				ts.Require().NotNil(info.DataAccessValidityDuration)
				ts.Equal(int64(3600000), *info.DataAccessValidityDuration)
				ts.Equal("v", info.Attributes["k"])
			},
		},

		// -----------------------------------------------------------------------
		// Enriched element data in consentInformation
		// -----------------------------------------------------------------------
		{
			name: "consentInformation includes enriched element fields (type)",
			setup: func(orgID string) (string, string) {
				ts.mustCreateElement(orgID, "val-enrich-elem", "basic")
				ts.mustCreatePurpose(orgID, "val-enrich-purp", "val-enrich-elem")
				c := ts.mustCreateConsent(orgID, "grp-val-enrich", ConsentCreateRequest{
					Type: "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
					Purposes: []PurposeRefRequest{
						{
							Name:     "val-enrich-purp",
							Elements: []ElementApprovalRequest{{Name: "val-enrich-elem", Approved: true}},
						},
					},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentValidateResponse) {
				ts.True(resp.IsValid)
				ts.Require().NotNil(resp.ConsentInfo)
				ts.Require().Len(resp.ConsentInfo.Purposes, 1)
				ts.Require().Len(resp.ConsentInfo.Purposes[0].Elements, 1)
				elem := resp.ConsentInfo.Purposes[0].Elements[0]
				ts.Equal("val-enrich-elem", elem.Name)
				ts.NotEmpty(elem.Type, "validate response must include the element type")
				ts.Equal("basic", elem.Type)
			},
		},

		// -----------------------------------------------------------------------
		// Invalid — wrong status
		// -----------------------------------------------------------------------
		{
			name: "CREATED consent (no auths) → isValid=false, errorCode=401, errorMessage=invalid_consent_status",
			setup: func(orgID string) (string, string) {
				// No authorizations → status remains CREATED
				c := ts.mustCreateConsent(orgID, "grp-val-created", ConsentCreateRequest{
					Type: "accounts",
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentValidateResponse) {
				ts.False(resp.IsValid)
				ts.Equal(401, resp.ErrorCode)
				ts.Equal("invalid_consent_status", resp.ErrorMessage)
				ts.Require().NotNil(resp.ConsentInfo, "consentInformation must be present even on failure")
				ts.Equal("CREATED", resp.ConsentInfo.Status)
			},
		},
		{
			name: "REVOKED consent → isValid=false, errorCode=401",
			setup: func(orgID string) (string, string) {
				c := ts.mustCreateConsent(orgID, "grp-val-rev", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				ts.doRevokeConsent(orgID, c.ID, ConsentRevokeRequest{ActionBy: "tester"})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentValidateResponse) {
				ts.False(resp.IsValid)
				ts.Equal(401, resp.ErrorCode)
				ts.Equal("invalid_consent_status", resp.ErrorMessage)
				ts.Require().NotNil(resp.ConsentInfo)
				ts.Equal("REVOKED", resp.ConsentInfo.Status)
			},
		},
		{
			name: "EXPIRED consent → isValid=false, errorCode=401, auto-expired during validate",
			setup: func(orgID string) (string, string) {
				// expirationTime 1 ms ago → consent expires on create (and again on validate if needed)
				past := time.Now().Add(-1*time.Second).UnixMilli()
				c := ts.mustCreateConsent(orgID, "grp-val-exp", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &past,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentValidateResponse) {
				ts.False(resp.IsValid)
				ts.Equal(401, resp.ErrorCode)
				ts.Equal("invalid_consent_status", resp.ErrorMessage)
				ts.Require().NotNil(resp.ConsentInfo)
				ts.Equal("EXPIRED", resp.ConsentInfo.Status)
			},
		},

		// -----------------------------------------------------------------------
		// Mandatory elements
		// -----------------------------------------------------------------------
		{
			name: "mandatory element approved → isValid=true",
			setup: func(orgID string) (string, string) {
				ts.mustCreateElement(orgID, "val-mand-elem", "basic")
				// Create purpose with element marked mandatory=true
				ts.doRequest(http.MethodPost, "/api/v1/consent-purposes", orgID, "", map[string]any{
					"name":     "val-mand-purp",
					"elements": []map[string]any{{"name": "val-mand-elem", "mandatory": true}},
				})
				c := ts.mustCreateConsent(orgID, "grp-mand-ok", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
					Purposes: []PurposeRefRequest{
						{
							Name:     "val-mand-purp",
							Elements: []ElementApprovalRequest{{Name: "val-mand-elem", Approved: true}},
						},
					},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentValidateResponse) {
				ts.True(resp.IsValid, "all mandatory elements approved → must be valid")
				ts.Equal(0, resp.ErrorCode)
			},
		},
		{
			name: "mandatory element NOT approved → isValid=false, errorCode=403, errorMessage=mandatory_elements_not_approved",
			setup: func(orgID string) (string, string) {
				ts.mustCreateElement(orgID, "val-mand2-elem", "basic")
				ts.doRequest(http.MethodPost, "/api/v1/consent-purposes", orgID, "", map[string]any{
					"name":     "val-mand2-purp",
					"elements": []map[string]any{{"name": "val-mand2-elem", "mandatory": true}},
				})
				c := ts.mustCreateConsent(orgID, "grp-mand-fail", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
					Purposes: []PurposeRefRequest{
						{
							Name: "val-mand2-purp",
							// approved=false (default) — mandatory element not approved
							Elements: []ElementApprovalRequest{{Name: "val-mand2-elem", Approved: false}},
						},
					},
				})
				return c.ID, orgID
			},
			wantStatus: http.StatusOK,
			checkResult: func(resp *ConsentValidateResponse) {
				ts.False(resp.IsValid)
				ts.Equal(403, resp.ErrorCode)
				ts.Equal("mandatory_elements_not_approved", resp.ErrorMessage)
				ts.NotEmpty(resp.ErrorDescription,
					"errorDescription must name the unapproved element")
				ts.Require().NotNil(resp.ConsentInfo)
			},
		},

		// -----------------------------------------------------------------------
		// Org isolation
		// -----------------------------------------------------------------------
		{
			name: "consent exists in org-A — validate with org-B → 404 CS-4040",
			setup: func(orgID string) (string, string) {
				c := ts.mustCreateConsent(orgID, "grp-val-iso", ConsentCreateRequest{
					Type: "accounts",
				})
				differentOrg := freshOrgID()
				return c.ID, differentOrg // validate with wrong org
			},
			wantStatus: http.StatusNotFound,
			wantError:  "CS-4040",
		},

		// -----------------------------------------------------------------------
		// Error cases — HTTP-level failures (not isValid=false)
		// -----------------------------------------------------------------------
		{
			name:       "missing consentId in body → 400 CS-4002",
			rawBody:    `{}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
		{
			name:       "non-existent consentId → 404 CS-4040",
			rawBody:    `{"consentId":"00000000-0000-0000-0000-000000000000"}`,
			wantStatus: http.StatusNotFound,
			wantError:  "CS-4040",
		},
		{
			name:       "malformed JSON body → 400 CS-4001",
			rawBody:    `{bad json`,
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4001",
		},
		{
			name:       "missing org-id header → 400 CS-4002",
			rawBody:    `{"consentId":"00000000-0000-0000-0000-000000000001"}`,
			omitOrgID:  true,
			wantStatus: http.StatusBadRequest,
			wantError:  "CS-4002",
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			orgID := freshOrgID()

			var consentID string
			validateOrgID := orgID
			if tc.setup != nil {
				consentID, validateOrgID = tc.setup(orgID)
			}
			if tc.omitOrgID {
				validateOrgID = ""
			}

			var reqBody any
			if tc.rawBody != "" {
				reqBody = tc.rawBody
			} else if consentID != "" {
				reqBody = ConsentValidateRequest{ConsentID: consentID}
			}

			status, body := ts.doValidateConsent(validateOrgID, reqBody)
			ts.Require().Equal(tc.wantStatus, status, "unexpected status; body: %s", body)

			if tc.wantError != "" {
				ts.assertAPIError(body, tc.wantError)
				return
			}

			var resp ConsentValidateResponse
			ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentValidateResponse: %s", body)
			if tc.checkResult != nil {
				tc.checkResult(&resp)
			}
		})
	}
}
