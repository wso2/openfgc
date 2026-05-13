/*
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package consent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/consent/model"
)

// TestRunExpirationJob_NoExpiredConsents
// When GetExpiredConsents returns an empty list, ExpireConsent must never be called.
func TestRunExpirationJob_NoExpiredConsents(t *testing.T) {
	svc := NewMockConsentService(t)
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE", "CREATED"}}

	svc.On("GetExpiredConsents",
		mock.Anything,
		mock.AnythingOfType("int64"),
		statuses.ExpirableConsentStatuses,
	).Return([]model.Consent{}, nil)

	RunExpirationJob(context.Background(), svc, statuses)

	svc.AssertNotCalled(t, "ExpireConsent")
}

// TestRunExpirationJob_GetExpiredConsentsFails
// When GetExpiredConsents returns an error, the job must abort early and never call ExpireConsent.
func TestRunExpirationJob_GetExpiredConsentsFails(t *testing.T) {
	svc := NewMockConsentService(t)
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	svc.On("GetExpiredConsents",
		mock.Anything,
		mock.AnythingOfType("int64"),
		statuses.ExpirableConsentStatuses,
	).Return(nil, errors.New("db connection failed"))

	RunExpirationJob(context.Background(), svc, statuses)

	svc.AssertNotCalled(t, "ExpireConsent")
}

// TestRunExpirationJob_ExpiresAllConsents
// When GetExpiredConsents returns N consents, ExpireConsent must be called exactly N times.
// Note: RunExpirationJob copies each consent before passing &c to ExpireConsent, so matching by ConsentID is required.
func TestRunExpirationJob_ExpiresAllConsents(t *testing.T) {
	svc := NewMockConsentService(t)
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	consents := []model.Consent{
		{ConsentID: "consent-aaa", OrgID: "org-1", CurrentStatus: "ACTIVE"},
		{ConsentID: "consent-bbb", OrgID: "org-2", CurrentStatus: "ACTIVE"},
	}

	svc.On("GetExpiredConsents",
		mock.Anything,
		mock.AnythingOfType("int64"),
		statuses.ExpirableConsentStatuses,
	).Return(consents, nil)

	svc.On("ExpireConsent",
		mock.Anything,
		mock.MatchedBy(func(c *model.Consent) bool { return c.ConsentID == "consent-aaa" }),
		"org-1",
	).Return(nil)

	svc.On("ExpireConsent",
		mock.Anything,
		mock.MatchedBy(func(c *model.Consent) bool { return c.ConsentID == "consent-bbb" }),
		"org-2",
	).Return(nil)

	RunExpirationJob(context.Background(), svc, statuses)

	svc.AssertNumberOfCalls(t, "ExpireConsent", 2)
}

// TestRunExpirationJob_ContinuesOnExpireError
// When ExpireConsent fails for one consent, the job must continue and still attempt to expire the remaining consents.
func TestRunExpirationJob_ContinuesOnExpireError(t *testing.T) {
	svc := NewMockConsentService(t)
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	consents := []model.Consent{
		{ConsentID: "consent-fail", OrgID: "org-1", CurrentStatus: "ACTIVE"},
		{ConsentID: "consent-ok", OrgID: "org-2", CurrentStatus: "ACTIVE"},
	}

	svc.On("GetExpiredConsents",
		mock.Anything,
		mock.AnythingOfType("int64"),
		statuses.ExpirableConsentStatuses,
	).Return(consents, nil)

	svc.On("ExpireConsent",
		mock.Anything,
		mock.MatchedBy(func(c *model.Consent) bool { return c.ConsentID == "consent-fail" }),
		"org-1",
	).Return(errors.New("expire failed"))

	svc.On("ExpireConsent",
		mock.Anything,
		mock.MatchedBy(func(c *model.Consent) bool { return c.ConsentID == "consent-ok" }),
		"org-2",
	).Return(nil)

	RunExpirationJob(context.Background(), svc, statuses)

	// Both consents must have been attempted despite the first failure.
	svc.AssertNumberOfCalls(t, "ExpireConsent", 2)
}

// TestRunExpirationJob_PanicRecovery
// RunExpirationJob has a deferred recover(). A panic inside GetExpiredConsents must be absorbed and must not propagate.
func TestRunExpirationJob_PanicRecovery(t *testing.T) {
	svc := NewMockConsentService(t)
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	svc.On("GetExpiredConsents",
		mock.Anything,
		mock.AnythingOfType("int64"),
		statuses.ExpirableConsentStatuses,
	).Run(func(_ mock.Arguments) {
		panic("intentional panic for test")
	}).Return(nil, nil)

	require.NotPanics(t, func() {
		RunExpirationJob(context.Background(), svc, statuses)
	})
}
