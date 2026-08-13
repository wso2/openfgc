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
	"net/http"
)

// =============================================================================
// Authorization writes against a consent whose lifecycle has ended.
//
// Revocation records that consent was withdrawn and marks every authorization
// system-revoked. Because system-revoked statuses are excluded from consent
// status derivation, a later authorization write would derive a status from the
// incoming authorization alone and could return the consent to an active state.
// Both write endpoints therefore refuse a revoked consent.
// =============================================================================

// revokeConsent calls POST /consents/{consentId}/revoke.
func (ts *AuthResourceAPITestSuite) revokeConsent(orgID, consentID string) {
	status, body := ts.doRequest(http.MethodPost, "/api/v1/consents/"+consentID+"/revoke", orgID,
		map[string]any{"actionBy": "tester"})
	ts.Require().Equal(http.StatusOK, status, "revokeConsent: unexpected status: %s", body)
}

func (ts *AuthResourceAPITestSuite) TestAuthResourceWritesOnRevokedConsent() {
	ts.Run("create is refused", func() {
		orgID := freshOrgID()
		consentID := ts.mustCreateConsent(orgID, "grp-revoked-create")
		ts.mustCreateAuthResource(orgID, consentID,
			AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "APPROVED"})

		ts.revokeConsent(orgID, consentID)
		ts.Require().Equal("REVOKED", ts.getConsentStatus(orgID, consentID))

		status, body := ts.doRequest(http.MethodPost, "/api/v1/consents/"+consentID+"/authorizations", orgID,
			AuthResourceCreateRequest{UserID: strPtr("user-002"), Status: "APPROVED"})
		ts.Equal(http.StatusBadRequest, status, "unexpected status; body: %s", body)
		ts.assertAPIError(body, "AR-4002")
		ts.Equal("REVOKED", ts.getConsentStatus(orgID, consentID),
			"a revoked consent must not return to an active state")
	})

	ts.Run("update is refused", func() {
		orgID := freshOrgID()
		consentID := ts.mustCreateConsent(orgID, "grp-revoked-update")
		ar := ts.mustCreateAuthResource(orgID, consentID,
			AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "CREATED"})

		ts.revokeConsent(orgID, consentID)
		ts.Require().Equal("REVOKED", ts.getConsentStatus(orgID, consentID))

		status, body := ts.doRequest(http.MethodPut,
			"/api/v1/consents/"+consentID+"/authorizations/"+ar.ID, orgID,
			AuthResourceUpdateRequest{UserID: strPtr("user-001"), Status: "APPROVED"})
		ts.Equal(http.StatusBadRequest, status, "unexpected status; body: %s", body)
		ts.assertAPIError(body, "AR-4002")
		ts.Equal("REVOKED", ts.getConsentStatus(orgID, consentID),
			"a revoked consent must not return to an active state")
	})

	ts.Run("writes are permitted before revocation", func() {
		orgID := freshOrgID()
		consentID := ts.mustCreateConsent(orgID, "grp-revoked-control")
		ar := ts.mustCreateAuthResource(orgID, consentID,
			AuthResourceCreateRequest{UserID: strPtr("user-001"), Status: "CREATED"})

		status, _ := ts.doRequest(http.MethodPut,
			"/api/v1/consents/"+consentID+"/authorizations/"+ar.ID, orgID,
			AuthResourceUpdateRequest{UserID: strPtr("user-001"), Status: "APPROVED"})
		ts.Equal(http.StatusOK, status)
		ts.Equal("ACTIVE", ts.getConsentStatus(orgID, consentID))
	})
}
