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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/consent-server/internal/consent/model"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
)

// expirationConsentService is a test double for ConsentService that covers only the
// two methods exercised by the expiration job. All other methods panic via the embed.
type expirationConsentService struct {
	unimplementedConsentService
	expiredConsents []model.Consent
	expiredErr      error
	expireErrMap    map[string]error
	expiredCalls    []string
}

func (s *expirationConsentService) GetExpiredConsents(_ context.Context, _ int64, _ []string) ([]model.Consent, *serviceerror.ServiceError) {
	if s.expiredErr != nil {
		return nil, serviceerror.CustomServiceError(ErrorInternalServerError, s.expiredErr.Error())
	}
	return s.expiredConsents, nil
}

func (s *expirationConsentService) ExpireConsent(_ context.Context, consent *model.Consent, _ string) *serviceerror.ServiceError {
	s.expiredCalls = append(s.expiredCalls, consent.ConsentID)
	if s.expireErrMap != nil {
		if err, ok := s.expireErrMap[consent.ConsentID]; ok {
			return serviceerror.CustomServiceError(ErrorInternalServerError, err.Error())
		}
	}
	return nil
}

// TestRunExpirationJob_NoExpiredConsents
// When GetExpiredConsents returns an empty list, ExpireConsent must never be called.
func TestRunExpirationJob_NoExpiredConsents(t *testing.T) {
	svc := &expirationConsentService{expiredConsents: []model.Consent{}}
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE", "CREATED"}}

	RunExpirationJob(context.Background(), svc, statuses)

	require.Empty(t, svc.expiredCalls, "ExpireConsent must not be called when no consents are expired")
}

// TestRunExpirationJob_GetExpiredConsentsFails
// When GetExpiredConsents returns an error, the job must abort early and never call ExpireConsent.
func TestRunExpirationJob_GetExpiredConsentsFails(t *testing.T) {
	svc := &expirationConsentService{expiredErr: errors.New("db connection failed")}
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	RunExpirationJob(context.Background(), svc, statuses)

	require.Empty(t, svc.expiredCalls, "ExpireConsent must not be called when GetExpiredConsents fails")
}

// TestRunExpirationJob_ExpiresAllConsents
// When GetExpiredConsents returns N consents, ExpireConsent must be called exactly N times.
func TestRunExpirationJob_ExpiresAllConsents(t *testing.T) {
	svc := &expirationConsentService{
		expiredConsents: []model.Consent{
			{ConsentID: "consent-aaa", OrgID: "org-1", CurrentStatus: "ACTIVE"},
			{ConsentID: "consent-bbb", OrgID: "org-2", CurrentStatus: "ACTIVE"},
		},
		expireErrMap: map[string]error{},
	}
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	RunExpirationJob(context.Background(), svc, statuses)

	require.Len(t, svc.expiredCalls, 2, "ExpireConsent must be called for each expired consent")
	require.Contains(t, svc.expiredCalls, "consent-aaa")
	require.Contains(t, svc.expiredCalls, "consent-bbb")
}

// TestRunExpirationJob_ContinuesOnExpireError
// When ExpireConsent fails for one consent, the job must continue and still attempt the remaining consents.
func TestRunExpirationJob_ContinuesOnExpireError(t *testing.T) {
	svc := &expirationConsentService{
		expiredConsents: []model.Consent{
			{ConsentID: "consent-fail", OrgID: "org-1", CurrentStatus: "ACTIVE"},
			{ConsentID: "consent-ok", OrgID: "org-2", CurrentStatus: "ACTIVE"},
		},
		expireErrMap: map[string]error{
			"consent-fail": errors.New("expire failed"),
		},
	}
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	RunExpirationJob(context.Background(), svc, statuses)

	require.Contains(t, svc.expiredCalls, "consent-ok", "second consent must still be attempted after first fails")
}

// TestRunExpirationJob_PanicRecovery
// A panic inside GetExpiredConsents must be absorbed and must not propagate.
func TestRunExpirationJob_PanicRecovery(t *testing.T) {
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	require.NotPanics(t, func() {
		RunExpirationJob(context.Background(), &panicConsentService{}, statuses)
	})
}

// panicConsentService satisfies ConsentService and panics on GetExpiredConsents.
type panicConsentService struct {
	unimplementedConsentService
}

func (p *panicConsentService) GetExpiredConsents(_ context.Context, _ int64, _ []string) ([]model.Consent, *serviceerror.ServiceError) {
	panic("intentional panic for test")
}
