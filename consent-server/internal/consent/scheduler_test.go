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
	"testing"
	"time"

	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
)

// unimplementedConsentService satisfies ConsentService with panicking stubs for every method.
// Embed this in test doubles and override only the methods under test.
type unimplementedConsentService struct{}

func (unimplementedConsentService) CreateConsent(_ context.Context, _ model.ConsentAPIRequest, _, _ string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	panic("not implemented")
}
func (unimplementedConsentService) GetConsent(_ context.Context, _, _ string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	panic("not implemented")
}
func (unimplementedConsentService) SearchConsentsDetailed(_ context.Context, _ model.ConsentSearchFilters) (*model.ConsentDetailSearchResponse, *serviceerror.ServiceError) {
	panic("not implemented")
}
func (unimplementedConsentService) UpdateConsent(_ context.Context, _ model.ConsentAPIUpdateRequest, _, _, _ string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	panic("not implemented")
}
func (unimplementedConsentService) RevokeConsent(_ context.Context, _, _ string, _ model.ConsentRevokeRequest) (*model.ConsentRevokeResponse, *serviceerror.ServiceError) {
	panic("not implemented")
}
func (unimplementedConsentService) ValidateConsent(_ context.Context, _ model.ValidateRequest, _ string) (*model.ValidateResponse, *serviceerror.ServiceError) {
	panic("not implemented")
}
func (unimplementedConsentService) SearchConsentsByAttribute(_ context.Context, _, _, _ string) (*model.ConsentAttributeSearchResponse, *serviceerror.ServiceError) {
	panic("not implemented")
}
func (unimplementedConsentService) GetExpiredConsents(_ context.Context, _ int64, _ []string) ([]model.Consent, *serviceerror.ServiceError) {
	panic("not implemented")
}
func (unimplementedConsentService) ExpireConsent(_ context.Context, _ *model.Consent, _ string) *serviceerror.ServiceError {
	panic("not implemented")
}

// signalingConsentService satisfies ConsentService and signals when GetExpiredConsents is called.
type signalingConsentService struct {
	unimplementedConsentService
	fired chan struct{}
}

func (s *signalingConsentService) GetExpiredConsents(_ context.Context, _ int64, _ []string) ([]model.Consent, *serviceerror.ServiceError) {
	select {
	case s.fired <- struct{}{}:
	default:
	}
	return []model.Consent{}, nil
}

func (s *signalingConsentService) ExpireConsent(_ context.Context, _ *model.Consent, _ string) *serviceerror.ServiceError {
	return nil
}

// TestStartScheduler_FiresJobOnTick verifies that StartScheduler launches RunExpirationJob on each ticker tick.
func TestStartScheduler_FiresJobOnTick(t *testing.T) {
	svc := &signalingConsentService{fired: make(chan struct{}, 1)}
	statuses := ExpirationStatuses{ExpirableConsentStatuses: []string{"ACTIVE"}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go StartScheduler(ctx, svc, 50*time.Millisecond, statuses)

	select {
	case <-svc.fired:
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not fire the expiration job within 2 seconds")
	}
}

// TestExpirationStatuses_Fields confirms ExpirationStatuses carries its status list correctly.
func TestExpirationStatuses_Fields(t *testing.T) {
	statuses := ExpirationStatuses{
		ExpirableConsentStatuses: []string{"ACTIVE", "CREATED"},
	}

	if len(statuses.ExpirableConsentStatuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses.ExpirableConsentStatuses))
	}
	if statuses.ExpirableConsentStatuses[0] != "ACTIVE" {
		t.Errorf("expected ACTIVE, got %s", statuses.ExpirableConsentStatuses[0])
	}
	if statuses.ExpirableConsentStatuses[1] != "CREATED" {
		t.Errorf("expected CREATED, got %s", statuses.ExpirableConsentStatuses[1])
	}
}
