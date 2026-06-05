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
	"time"
)

// =============================================================================
// Scheduler expiry helper
// =============================================================================

// pollUntilExpiredInSearch polls GET /consents?consentStatuses=EXPIRED until
// the given consent ID appears in the results, or until timeout is exceeded.
// Returns true if the consent was found in the EXPIRED list.
//
// Using the list endpoint (not GET /consents/{id}) is deliberate: the GET
// by-ID endpoint triggers the on-read expiry path, which would make it
// impossible to tell whether the scheduler or the read handler expired the
// consent. The list endpoint returns persisted state without side effects.
func (ts *ConsentAPITestSuite) pollUntilExpiredInSearch(orgID, consentID string, timeout, interval time.Duration) bool {
	ts.T().Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, resp := ts.doListConsents(orgID, url.Values{"consentStatuses": {"EXPIRED"}})
		if status == http.StatusOK && resp != nil {
			for _, c := range resp.Data {
				if c.ID == consentID {
					return true
				}
			}
		}
		time.Sleep(interval)
	}
	return false
}

// =============================================================================
// TestConsentExpiry — on-read expiry triggered by GET and validate
// =============================================================================

// TestConsentExpiry covers the auto-expiry behaviour triggered on read operations.
//
// The service checks whether a consent's expirationTime has passed whenever the
// consent is loaded (GET, validate). If so, it atomically transitions the consent
// and all its auth resources to their respective expired statuses before returning
// the response.
//
// Rules under test:
//   - Creating a consent with a past expirationTime immediately marks it EXPIRED.
//   - GET on an EXPIRED consent returns status EXPIRED.
//   - Auth resources of an expired consent are marked SYS_EXPIRED.
//   - Validate on an EXPIRED consent returns isValid=false with errorCode=401.
//   - Creating a consent with a future expirationTime keeps it CREATED/ACTIVE.
//   - List (/consents) with consentStatuses=EXPIRED returns expired consents.
func (ts *ConsentAPITestSuite) TestConsentExpiry() {
	pastMs := func(d time.Duration) int64 { return time.Now().Add(-d).UnixMilli() }
	futureMs := func(d time.Duration) int64 { return time.Now().Add(d).UnixMilli() }

	type testCase struct {
		name string
		run  func(orgID string)
	}

	cases := []testCase{

		// -----------------------------------------------------------------------
		// Create with past expirationTime → immediately EXPIRED
		// -----------------------------------------------------------------------
		{
			name: "create with past expirationTime → create response has status EXPIRED",
			run: func(orgID string) {
				past := pastMs(1 * time.Minute)
				c := ts.mustCreateConsent(orgID, "grp-exp-create", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &past,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				ts.Equal("EXPIRED", c.Status,
					"consent with past expirationTime must be EXPIRED immediately")
			},
		},

		// -----------------------------------------------------------------------
		// GET on an expired consent
		// -----------------------------------------------------------------------
		{
			name: "GET expired consent returns status EXPIRED",
			run: func(orgID string) {
				past := pastMs(1 * time.Minute)
				created := ts.mustCreateConsent(orgID, "grp-exp-get", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &past,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})

				status, got := ts.doGetConsent(orgID, created.ID)
				ts.Require().Equal(http.StatusOK, status)
				ts.Require().NotNil(got)
				ts.Equal("EXPIRED", got.Status, "GET on expired consent must return EXPIRED status")
			},
		},
		{
			name: "GET expired consent — expirationTime is present in response",
			run: func(orgID string) {
				past := pastMs(30 * time.Second)
				created := ts.mustCreateConsent(orgID, "grp-exp-etime", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &past,
				})

				_, got := ts.doGetConsent(orgID, created.ID)
				ts.Require().NotNil(got)
				ts.Require().NotNil(got.ExpirationTime, "expirationTime must be returned in response")
				ts.Equal(past, *got.ExpirationTime)
			},
		},

		// -----------------------------------------------------------------------
		// Auth resources get SYS_EXPIRED when consent expires
		// -----------------------------------------------------------------------
		{
			name: "auth resources of expired consent are marked SYS_EXPIRED",
			run: func(orgID string) {
				past := pastMs(1 * time.Minute)
				created := ts.mustCreateConsent(orgID, "grp-exp-auth", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &past,
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Type: "accounts", Status: "APPROVED"},
						{UserID: "user-002", Type: "accounts", Status: "APPROVED"},
					},
				})

				// GET triggers expiry check; auth resources should be SYS_EXPIRED.
				_, got := ts.doGetConsent(orgID, created.ID)
				ts.Require().NotNil(got)
				ts.Equal("EXPIRED", got.Status)
				ts.Require().Len(got.Authorizations, 2,
					"both auth resources must still be present after expiry")
				for _, auth := range got.Authorizations {
					ts.Equal("SYS_EXPIRED", auth.Status,
						"auth resource must be SYS_EXPIRED after consent expires")
				}
			},
		},

		// -----------------------------------------------------------------------
		// Validate on an expired consent
		// -----------------------------------------------------------------------
		{
			name: "validate expired consent → isValid=false, errorCode=401, status EXPIRED in consentInfo",
			run: func(orgID string) {
				past := pastMs(1 * time.Minute)
				created := ts.mustCreateConsent(orgID, "grp-exp-val", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &past,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})

				_, body := ts.doValidateConsent(orgID, ConsentValidateRequest{ConsentID: created.ID})
				var resp ConsentValidateResponse
				ts.Require().NoError(json.Unmarshal(body, &resp))
				ts.False(resp.IsValid)
				ts.Equal(401, resp.ErrorCode)
				ts.Equal("invalid_consent_status", resp.ErrorMessage)
				ts.Require().NotNil(resp.ConsentInfo)
				ts.Equal("EXPIRED", resp.ConsentInfo.Status)
			},
		},

		// -----------------------------------------------------------------------
		// Future expirationTime does not trigger expiry
		// -----------------------------------------------------------------------
		{
			name: "future expirationTime — consent status is not EXPIRED",
			run: func(orgID string) {
				future := futureMs(24 * time.Hour)
				c := ts.mustCreateConsent(orgID, "grp-exp-future", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &future,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				ts.NotEqual("EXPIRED", c.Status,
					"consent with future expirationTime must not be EXPIRED")
				ts.Equal("ACTIVE", c.Status)
			},
		},
		{
			name: "no expirationTime — consent never expires",
			run: func(orgID string) {
				c := ts.mustCreateConsent(orgID, "grp-no-exp", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				ts.Equal("ACTIVE", c.Status)
				ts.Nil(c.ExpirationTime, "expirationTime must be absent when not set")
			},
		},

		// -----------------------------------------------------------------------
		// Expiry status visible in list / search
		// -----------------------------------------------------------------------
		{
			name: "list with consentStatuses=EXPIRED returns expired consents",
			run: func(orgID string) {
				past := pastMs(1 * time.Minute)
				ts.mustCreateConsent(orgID, "grp-list-exp", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &past,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				// Non-expired consent — must NOT appear in EXPIRED filter.
				ts.mustCreateConsent(orgID, "grp-list-active", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})

				status, resp := ts.doListConsents(orgID, url.Values{"consentStatuses": {"EXPIRED"}})
				ts.Require().Equal(http.StatusOK, status)
				ts.Require().NotNil(resp)
				ts.Equal(1, resp.Metadata.Total)
				ts.Require().Len(resp.Data, 1)
				ts.Equal("EXPIRED", resp.Data[0].Status)
			},
		},
		{
			name: "list without status filter includes expired consents alongside active ones",
			run: func(orgID string) {
				past := pastMs(1 * time.Minute)
				ts.mustCreateConsent(orgID, "grp-all-exp", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &past,
				})
				ts.mustCreateConsent(orgID, "grp-all-act", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})

				status, resp := ts.doListConsents(orgID, nil)
				ts.Require().Equal(http.StatusOK, status)
				ts.Require().NotNil(resp)
				ts.Equal(2, resp.Metadata.Total)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			tc.run(freshOrgID())
		})
	}
}

// =============================================================================
// TestSchedulerExpiration — background scheduler expiry
// =============================================================================

// TestSchedulerExpiration covers the background consent expiration scheduler.
//
// Contrast with TestConsentExpiry above, which tests the *on-read* expiry path
// (GET /consents/{id} and POST /consents/validate trigger inline expiry before
// returning). Here we verify that the background scheduler independently
// discovers and expires consents whose expirationTime has passed.
//
// Observation strategy:
//   - Do NOT call GET /consents/{id} before asserting scheduler behaviour.
//     That endpoint triggers on-read expiry, making it impossible to tell
//     which path caused the status transition.
//   - Use GET /consents (list/search) instead. The list endpoint returns the
//     persisted DB state without triggering any inline expiry logic.
//
// Each test creates consents with expirationTime = now + 3 s (positive offset
// so on-create expiry does not fire), then polls the list endpoint for up to
// 20 s. Tests pass as soon as the scheduler fires — typically within a few
// seconds in a CI environment.
//
// Negative-case tests (revoked, no expiry) pair a *sentinel* consent that
// should expire alongside the subject under test. Only once the sentinel
// appears as EXPIRED in the list (proving the scheduler has run at least once)
// is the assertion on the subject meaningful.
func (ts *ConsentAPITestSuite) TestSchedulerExpiration() {
	const (
		schedExpiry  = 3 * time.Second
		pollTimeout  = 20 * time.Second
		pollInterval = 500 * time.Millisecond
	)

	type testCase struct {
		name string
		run  func(orgID string)
	}

	cases := []testCase{

		// -----------------------------------------------------------------------
		// Happy path: ACTIVE consent transitions to EXPIRED
		// -----------------------------------------------------------------------
		{
			name: "ACTIVE consent is moved to EXPIRED after expirationTime passes",
			run: func(orgID string) {
				expiry := time.Now().Add(schedExpiry).UnixMilli()

				c := ts.mustCreateConsent(orgID, "grp-sched-active", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &expiry,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				ts.Require().Equal("ACTIVE", c.Status,
					"consent must start ACTIVE — expirationTime is still in the future")

				ts.Require().True(
					ts.pollUntilExpiredInSearch(orgID, c.ID, pollTimeout, pollInterval),
					"scheduler must transition consent to EXPIRED within %s", pollTimeout,
				)
			},
		},

		// -----------------------------------------------------------------------
		// Auth resources are SYS_EXPIRED after scheduler expiry
		// -----------------------------------------------------------------------
		{
			name: "auth resources are set to SYS_EXPIRED after scheduler expiry",
			run: func(orgID string) {
				expiry := time.Now().Add(schedExpiry).UnixMilli()

				c := ts.mustCreateConsent(orgID, "grp-sched-auth", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &expiry,
					Authorizations: []AuthorizationRequest{
						{UserID: "user-001", Type: "accounts", Status: "APPROVED"},
						{UserID: "user-002", Type: "savings", Status: "APPROVED"},
					},
				})
				ts.Require().Equal("ACTIVE", c.Status)
				ts.Require().Len(c.Authorizations, 2)

				ts.Require().True(
					ts.pollUntilExpiredInSearch(orgID, c.ID, pollTimeout, pollInterval),
					"scheduler must expire the consent",
				)

				// The list endpoint batch-fetches auth resources — no need for GET.
				_, list := ts.doListConsents(orgID, url.Values{"consentStatuses": {"EXPIRED"}})
				ts.Require().NotNil(list)

				var found *ConsentResponse
				for i := range list.Data {
					if list.Data[i].ID == c.ID {
						found = &list.Data[i]
						break
					}
				}
				ts.Require().NotNil(found, "expired consent must appear in the EXPIRED search results")
				ts.Require().Len(found.Authorizations, 2, "all auth resources must still be returned after expiry")
				for _, auth := range found.Authorizations {
					ts.Equal("SYS_EXPIRED", auth.Status,
						"auth resource %s must be SYS_EXPIRED after scheduler expires the consent", auth.ID)
				}
			},
		},

		// -----------------------------------------------------------------------
		// CREATED consent (no authorizations) is also expired by the scheduler
		// -----------------------------------------------------------------------
		{
			name: "CREATED consent with no authorizations is moved to EXPIRED by scheduler",
			run: func(orgID string) {
				expiry := time.Now().Add(schedExpiry).UnixMilli()

				c := ts.mustCreateConsent(orgID, "grp-sched-created", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &expiry,
					// No authorizations → status starts as CREATED.
				})
				ts.Require().Equal("CREATED", c.Status,
					"consent with no authorizations must start as CREATED")

				ts.Require().True(
					ts.pollUntilExpiredInSearch(orgID, c.ID, pollTimeout, pollInterval),
					"scheduler must expire a CREATED consent after its expirationTime passes",
				)
			},
		},

		// -----------------------------------------------------------------------
		// Negative: REVOKED consent is not re-expired by the scheduler
		// -----------------------------------------------------------------------
		{
			name: "REVOKED consent stays REVOKED after its expirationTime passes",
			run: func(orgID string) {
				expiry := time.Now().Add(schedExpiry).UnixMilli()

				// Sentinel: an ACTIVE consent that will be expired by the scheduler.
				// When the sentinel appears as EXPIRED in search, we know the scheduler
				// has run at least once, making the negative assertion on the subject valid.
				sentinel := ts.mustCreateConsent(orgID, "grp-sched-rev-s", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &expiry,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})

				// Subject: revoke it before its expirationTime passes.
				subject := ts.mustCreateConsent(orgID, "grp-sched-rev", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &expiry,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				_, revokeResp := ts.doRevokeConsent(orgID, subject.ID, ConsentRevokeRequest{ActionBy: "test"})
				ts.Require().NotNil(revokeResp, "revoke must succeed before asserting scheduler behaviour")

				// Wait for the sentinel to expire — proves the scheduler has run.
				ts.Require().True(
					ts.pollUntilExpiredInSearch(orgID, sentinel.ID, pollTimeout, pollInterval),
					"sentinel consent must be expired by the scheduler to validate the negative case",
				)

				// Subject must still appear as REVOKED, not EXPIRED.
				_, list := ts.doListConsents(orgID, url.Values{"consentStatuses": {"REVOKED"}})
				ts.Require().NotNil(list)
				found := false
				for _, c := range list.Data {
					if c.ID == subject.ID {
						ts.Equal("REVOKED", c.Status,
							"REVOKED consent must not be re-expired by the scheduler")
						found = true
						break
					}
				}
				ts.True(found, "REVOKED consent must still appear under the REVOKED status filter")
			},
		},

		// -----------------------------------------------------------------------
		// Negative: consent with no expirationTime is never expired by the scheduler
		// -----------------------------------------------------------------------
		{
			name: "consent with no expirationTime is never expired by scheduler",
			run: func(orgID string) {
				expiry := time.Now().Add(schedExpiry).UnixMilli()

				// Sentinel: will be expired, proving the scheduler has run.
				sentinel := ts.mustCreateConsent(orgID, "grp-sched-noexp-s", ConsentCreateRequest{
					Type:           "accounts",
					ExpirationTime: &expiry,
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})

				// Subject: no expirationTime set — should never be expired.
				subject := ts.mustCreateConsent(orgID, "grp-sched-noexp", ConsentCreateRequest{
					Type:           "accounts",
					Authorizations: []AuthorizationRequest{{UserID: "user-001", Status: "APPROVED"}},
				})
				ts.Nil(subject.ExpirationTime, "subject must have no expirationTime")

				// Wait for sentinel to expire.
				ts.Require().True(
					ts.pollUntilExpiredInSearch(orgID, sentinel.ID, pollTimeout, pollInterval),
					"sentinel must expire to confirm the scheduler has run at least once",
				)

				// Subject must remain ACTIVE.
				_, list := ts.doListConsents(orgID, url.Values{"consentStatuses": {"ACTIVE"}})
				ts.Require().NotNil(list)
				found := false
				for _, c := range list.Data {
					if c.ID == subject.ID {
						ts.Equal("ACTIVE", c.Status)
						found = true
						break
					}
				}
				ts.True(found, "consent with no expirationTime must remain ACTIVE after scheduler runs")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		ts.Run(tc.name, func() {
			tc.run(freshOrgID())
		})
	}
}
