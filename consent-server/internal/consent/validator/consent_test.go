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

package validator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/system/config"
)

// =============================================================================
// ValidateConsentCreateRequest
// =============================================================================

func TestValidateConsentCreateRequest_Success(t *testing.T) {
	// Status omitted — it is optional; service applies the default. UserID is required.
	req := model.ConsentCreateRequest{
		Type:           "accounts",
		Authorizations: []model.AuthorizationRequest{{UserID: "user-001", Type: "accounts"}},
	}
	err := ValidateConsentCreateRequest(req, "grp-1", "org-1")
	require.NoError(t, err)
}

func TestValidateConsentCreateRequest_AuthTypeOptional(t *testing.T) {
	// Authorization type is optional — service defaults to "default". UserID is required.
	req := model.ConsentCreateRequest{
		Type:           "accounts",
		Authorizations: []model.AuthorizationRequest{{UserID: "user-001"}},
	}
	err := ValidateConsentCreateRequest(req, "grp-1", "org-1")
	require.NoError(t, err)
}

func TestValidateConsentCreateRequest_MissingAuthUserID(t *testing.T) {
	// Omitting userId in an authorization object must return a validation error.
	req := model.ConsentCreateRequest{
		Type:           "accounts",
		Authorizations: []model.AuthorizationRequest{{Type: "accounts"}},
	}
	err := ValidateConsentCreateRequest(req, "grp-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorizations[0]: userId is required")
}

func TestValidateConsentCreateRequest_MissingType(t *testing.T) {
	req := model.ConsentCreateRequest{}
	err := ValidateConsentCreateRequest(req, "grp-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "type is required")
}

func TestValidateConsentCreateRequest_TypeTooLong(t *testing.T) {
	req := model.ConsentCreateRequest{Type: string(make([]byte, 65))}
	err := ValidateConsentCreateRequest(req, "grp-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "type must be at most 64 characters")
}

func TestValidateConsentCreateRequest_MissingGroupID(t *testing.T) {
	req := model.ConsentCreateRequest{Type: "accounts"}
	err := ValidateConsentCreateRequest(req, "", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "group-id header is required")
}

func TestValidateConsentCreateRequest_MissingOrgID(t *testing.T) {
	req := model.ConsentCreateRequest{Type: "accounts"}
	err := ValidateConsentCreateRequest(req, "grp-1", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "orgID is required")
}

func TestValidateConsentCreateRequest_NegativeExpirationTime(t *testing.T) {
	neg := int64(-1)
	req := model.ConsentCreateRequest{Type: "accounts", ExpirationTime: &neg}
	err := ValidateConsentCreateRequest(req, "grp-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "expirationTime must be non-negative")
}

func TestValidateConsentCreateRequest_ExpirationTimeTooSmall(t *testing.T) {
	// Values with too few digits are not valid Unix timestamps in either seconds
	// or milliseconds format and must be rejected.
	for _, v := range []int64{1, 123, 999_999_999} {
		v := v
		t.Run(fmt.Sprintf("value=%d", v), func(t *testing.T) {
			req := model.ConsentCreateRequest{Type: "accounts", ExpirationTime: &v}
			err := ValidateConsentCreateRequest(req, "grp-1", "org-1")
			require.Error(t, err)
			require.Contains(t, err.Error(), "expirationTime is not a valid Unix timestamp")
		})
	}
}

func TestValidateConsentCreateRequest_NegativeFrequency(t *testing.T) {
	neg := -5
	req := model.ConsentCreateRequest{Type: "accounts", Frequency: &neg}
	err := ValidateConsentCreateRequest(req, "grp-1", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "frequency must be non-negative")
}

// =============================================================================
// ValidateConsentUpdateRequest
// =============================================================================

func TestValidateConsentUpdateRequest_Success(t *testing.T) {
	req := model.ConsentUpdateRequest{Type: "payments"}
	err := ValidateConsentUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateConsentUpdateRequest_NoFieldsProvided(t *testing.T) {
	req := model.ConsentUpdateRequest{}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one field must be provided")
}

func TestValidateConsentUpdateRequest_EmptySlicesAreValid(t *testing.T) {
	// Explicitly empty slices/maps must count as provided — nil is the "absent" sentinel.
	require.NoError(t, ValidateConsentUpdateRequest(model.ConsentUpdateRequest{
		Authorizations: []model.AuthorizationRequest{},
	}), "empty Authorizations slice must be a valid update")

	require.NoError(t, ValidateConsentUpdateRequest(model.ConsentUpdateRequest{
		Purposes: []model.ConsentPurposeRefRequest{},
	}), "empty Purposes slice must be a valid update")

	require.NoError(t, ValidateConsentUpdateRequest(model.ConsentUpdateRequest{
		Attributes: map[string]string{},
	}), "empty Attributes map must be a valid update")
}

func TestValidateConsentUpdateRequest_AllNilIsStillRejected(t *testing.T) {
	// All fields nil/zero — still rejected; nil slices/maps are "not provided".
	err := ValidateConsentUpdateRequest(model.ConsentUpdateRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one field must be provided")
}

func TestValidateConsentUpdateRequest_TypeTooLong(t *testing.T) {
	req := model.ConsentUpdateRequest{Type: string(make([]byte, 65))}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "type must be at most 64 characters")
}

func TestValidateConsentUpdateRequest_NegativeExpirationTime(t *testing.T) {
	neg := int64(-1)
	req := model.ConsentUpdateRequest{ExpirationTime: &neg}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expirationTime must be non-negative")
}

func TestValidateConsentUpdateRequest_ExpirationTimeTooSmall(t *testing.T) {
	small := int64(123)
	req := model.ConsentUpdateRequest{ExpirationTime: &small}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expirationTime is not a valid Unix timestamp")
}

func TestValidateConsentUpdateRequest_NegativeFrequency(t *testing.T) {
	neg := -5
	req := model.ConsentUpdateRequest{Frequency: &neg}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "frequency must be non-negative")
}

func TestValidateConsentUpdateRequest_AuthTypeOptional(t *testing.T) {
	// Authorization type is optional in update too; status omitted to avoid config dependency.
	// UserID is required.
	req := model.ConsentUpdateRequest{
		Authorizations: []model.AuthorizationRequest{{UserID: "user-001"}},
	}
	err := ValidateConsentUpdateRequest(req)
	require.NoError(t, err)
}

func TestValidateConsentUpdateRequest_MissingAuthUserID(t *testing.T) {
	// Omitting userId in an authorization object must return a validation error.
	req := model.ConsentUpdateRequest{
		Authorizations: []model.AuthorizationRequest{{Type: "accounts"}},
	}
	err := ValidateConsentUpdateRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorizations[0]: userId is required")
}

// =============================================================================
// ValidateConsentGetRequest
// =============================================================================

func TestValidateConsentGetRequest_Success(t *testing.T) {
	require.NoError(t, ValidateConsentGetRequest("consent-123", "org-1"))
}

func TestValidateConsentGetRequest_EmptyConsentID(t *testing.T) {
	err := ValidateConsentGetRequest("", "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "consent ID cannot be empty")
}

func TestValidateConsentGetRequest_EmptyOrgID(t *testing.T) {
	err := ValidateConsentGetRequest("consent-123", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "organization ID cannot be empty")
}

func TestValidateConsentGetRequest_ConsentIDTooLong(t *testing.T) {
	err := ValidateConsentGetRequest(string(make([]byte, 256)), "org-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "consent ID too long")
}

func TestValidateConsentGetRequest_OrgIDTooLong(t *testing.T) {
	err := ValidateConsentGetRequest("consent-123", string(make([]byte, 256)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "organization ID too long")
}

// =============================================================================
// IsConsentExpired
// =============================================================================

func TestIsConsentExpired_ZeroMeansNoExpiry(t *testing.T) {
	require.False(t, IsConsentExpired(0))
}

func TestIsConsentExpired_FutureTimestamp(t *testing.T) {
	// Far-future timestamp in ms
	require.False(t, IsConsentExpired(9_999_999_999_999))
}

func TestIsConsentExpired_PastTimestamp(t *testing.T) {
	// Well-past timestamp in ms (year 2001)
	require.True(t, IsConsentExpired(1_000_000_000_000))
}

// =============================================================================
// EvaluateConsentStatusFromAuthStatuses
// =============================================================================

// setTestConfig initialises the global config with standard status labels used
// by EvaluateConsentStatusFromAuthStatuses. Called at the start of each sub-test
// that exercises the derivation logic so the config is always present.
func setTestConfig() {
	config.SetGlobal(&config.Config{
		Consent: config.ConsentConfig{
			StatusMappings: config.ConsentStatusMappings{
				ActiveStatus:   "ACTIVE",
				CreatedStatus:  "CREATED",
				RejectedStatus: "REJECTED",
			},
			AuthStatusMappings: config.AuthStatusMappings{
				ApprovedState:      "APPROVED",
				RejectedState:      "REJECTED",
				CreatedState:       "CREATED",
				RecordedState:      "RECORDED",
				SystemExpiredState: "SYS_EXPIRED",
				SystemRevokedState: "SYS_REVOKED",
			},
		},
	})
}

func TestEvaluateConsentStatus_EmptyList(t *testing.T) {
	setTestConfig()
	// No auth resources → consent stays in CREATED
	got := EvaluateConsentStatusFromAuthStatuses(nil)
	require.Equal(t, "CREATED", got)
}

func TestEvaluateConsentStatus_AllApproved(t *testing.T) {
	setTestConfig()
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "APPROVED"})
	require.Equal(t, "ACTIVE", got)
}

func TestEvaluateConsentStatus_AllCreated(t *testing.T) {
	setTestConfig()
	got := EvaluateConsentStatusFromAuthStatuses([]string{"CREATED", "CREATED"})
	require.Equal(t, "CREATED", got)
}

func TestEvaluateConsentStatus_AllRejected(t *testing.T) {
	setTestConfig()
	got := EvaluateConsentStatusFromAuthStatuses([]string{"REJECTED", "REJECTED"})
	require.Equal(t, "REJECTED", got)
}

func TestEvaluateConsentStatus_RejectedTakesPriorityOverApproved(t *testing.T) {
	setTestConfig()
	// Even one REJECTED makes the whole consent REJECTED.
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "REJECTED"})
	require.Equal(t, "REJECTED", got)
}

func TestEvaluateConsentStatus_RejectedTakesPriorityOverCreated(t *testing.T) {
	setTestConfig()
	got := EvaluateConsentStatusFromAuthStatuses([]string{"CREATED", "REJECTED"})
	require.Equal(t, "REJECTED", got)
}

func TestEvaluateConsentStatus_CreatedTakesPriorityOverApproved(t *testing.T) {
	setTestConfig()
	// Mix of APPROVED and CREATED → CREATED wins over APPROVED.
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "CREATED"})
	require.Equal(t, "CREATED", got)
}

func TestEvaluateConsentStatus_MixedAllThree_RejectedWins(t *testing.T) {
	setTestConfig()
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "CREATED", "REJECTED"})
	require.Equal(t, "REJECTED", got)
}

func TestEvaluateConsentStatus_SingleApproved(t *testing.T) {
	setTestConfig()
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED"})
	require.Equal(t, "ACTIVE", got)
}

func TestEvaluateConsentStatus_SingleRejected(t *testing.T) {
	setTestConfig()
	got := EvaluateConsentStatusFromAuthStatuses([]string{"REJECTED"})
	require.Equal(t, "REJECTED", got)
}

func TestEvaluateConsentStatus_UnknownStatusTreatedAsCreated(t *testing.T) {
	setTestConfig()
	// An unknown status value should not cause a panic and should produce CREATED.
	got := EvaluateConsentStatusFromAuthStatuses([]string{"SOME_UNKNOWN_STATUS"})
	require.Equal(t, "CREATED", got)
}

func TestEvaluateConsentStatus_CaseInsensitive(t *testing.T) {
	setTestConfig()
	// Status comparison must be case-insensitive.
	got := EvaluateConsentStatusFromAuthStatuses([]string{"approved", "approved"})
	require.Equal(t, "ACTIVE", got)
}

// =============================================================================
// EvaluateConsentStatusFromAuthStatuses — RECORDED filtering
// =============================================================================

func TestEvaluateConsentStatus_RecordedIsSkipped(t *testing.T) {
	setTestConfig()
	// Parent APPROVED + child RECORDED → skip RECORDED → only APPROVED remains → ACTIVE
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "RECORDED"})
	require.Equal(t, "ACTIVE", got)
}

func TestEvaluateConsentStatus_RecordedWithCreated(t *testing.T) {
	setTestConfig()
	// Delegate CREATED + child RECORDED → skip RECORDED → CREATED remains → CREATED
	got := EvaluateConsentStatusFromAuthStatuses([]string{"CREATED", "RECORDED"})
	require.Equal(t, "CREATED", got)
}

func TestEvaluateConsentStatus_RecordedWithRejected(t *testing.T) {
	setTestConfig()
	// Delegate REJECTED + child RECORDED → skip RECORDED → REJECTED remains → REJECTED
	got := EvaluateConsentStatusFromAuthStatuses([]string{"REJECTED", "RECORDED"})
	require.Equal(t, "REJECTED", got)
}

func TestEvaluateConsentStatus_MultipleRecorded(t *testing.T) {
	setTestConfig()
	// One APPROVED + multiple RECORDED → all RECORDED skipped → ACTIVE
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "RECORDED", "RECORDED"})
	require.Equal(t, "ACTIVE", got)
}

func TestEvaluateConsentStatus_SysExpiredIsSkipped(t *testing.T) {
	setTestConfig()
	// SYS_EXPIRED should be filtered out, not treated as unknown/CREATED.
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "SYS_EXPIRED"})
	require.Equal(t, "ACTIVE", got)
}

func TestEvaluateConsentStatus_SysRevokedIsSkipped(t *testing.T) {
	setTestConfig()
	// SYS_REVOKED should be filtered out, not treated as unknown/CREATED.
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "SYS_REVOKED"})
	require.Equal(t, "ACTIVE", got)
}

func TestEvaluateConsentStatus_AllNonParticipating_FallsBackToCreated(t *testing.T) {
	setTestConfig()
	// If every status is filtered (RECORDED + SYS_EXPIRED), the safety fallback is CREATED.
	// This shouldn't happen in practice because validation prevents all-RECORDED consents.
	got := EvaluateConsentStatusFromAuthStatuses([]string{"RECORDED", "SYS_EXPIRED"})
	require.Equal(t, "CREATED", got)
}

func TestEvaluateConsentStatus_RecordedCaseInsensitive(t *testing.T) {
	setTestConfig()
	// RECORDED filtering must be case-insensitive.
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "recorded"})
	require.Equal(t, "ACTIVE", got)
}

func TestEvaluateConsentStatus_DelegationScenario_ParentChild(t *testing.T) {
	setTestConfig()
	// Realistic delegation: parent APPROVED, child RECORDED → ACTIVE
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "RECORDED"})
	require.Equal(t, "ACTIVE", got, "parent-child delegation: parent approved, child recorded → ACTIVE")
}

func TestEvaluateConsentStatus_DelegationScenario_ParentPending(t *testing.T) {
	setTestConfig()
	// Parent hasn't approved yet (CREATED), child RECORDED → CREATED
	got := EvaluateConsentStatusFromAuthStatuses([]string{"CREATED", "RECORDED"})
	require.Equal(t, "CREATED", got, "parent-child delegation: parent pending, child recorded → CREATED")
}

func TestEvaluateConsentStatus_CustomTypeScenario_OwnerAndAgent(t *testing.T) {
	setTestConfig()
	// owner APPROVED + agent RECORDED → skip RECORDED → ACTIVE
	got := EvaluateConsentStatusFromAuthStatuses([]string{"APPROVED", "RECORDED"})
	require.Equal(t, "ACTIVE", got, "custom types: owner approved, agent recorded → ACTIVE")
}

// =============================================================================
// ValidateAuthTypeConstraints
// =============================================================================

func TestValidateAuthTypeConstraints_EmptyList(t *testing.T) {
	setTestConfig()
	// No authorizations — nothing to validate, passes.
	err := ValidateAuthTypeConstraints(nil)
	require.NoError(t, err)
}

func TestValidateAuthTypeConstraints_PrimaryOnly(t *testing.T) {
	setTestConfig()
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "user-1", Type: "primary", Status: "APPROVED"},
	})
	require.NoError(t, err)
}

func TestValidateAuthTypeConstraints_DefaultTreatedAsPrimary(t *testing.T) {
	setTestConfig()
	// Empty type defaults to "default" which is treated as self-consent (like primary).
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "user-1", Status: "APPROVED"},
	})
	require.NoError(t, err)
}

func TestValidateAuthTypeConstraints_DelegateWithDelegateSubject(t *testing.T) {
	setTestConfig()
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
		{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
	})
	require.NoError(t, err)
}

func TestValidateAuthTypeConstraints_DelegateWithoutDelegateSubject(t *testing.T) {
	setTestConfig()
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "'delegate' requires at least one 'delegate_subject'")
}

func TestValidateAuthTypeConstraints_DelegateSubjectWithoutDelegate(t *testing.T) {
	setTestConfig()
	// This also fails the universal rule (only RECORDED = no active participant),
	// but the first-class pairing check fires first.
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
	})
	require.Error(t, err)
	// Should hit either the universal rule or the pairing rule
}

func TestValidateAuthTypeConstraints_PrimaryMixedWithDelegate(t *testing.T) {
	setTestConfig()
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "user-1", Type: "primary", Status: "APPROVED"},
		{UserID: "father-111", Type: "delegate", Status: "APPROVED"},
		{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "'primary' cannot be mixed with 'delegate' or 'delegate_subject'")
}

func TestValidateAuthTypeConstraints_PrimaryMixedWithDelegateSubject(t *testing.T) {
	setTestConfig()
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "user-1", Type: "primary", Status: "APPROVED"},
		{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "'primary' cannot be mixed with 'delegate' or 'delegate_subject'")
}

func TestValidateAuthTypeConstraints_CustomTypesSkipValidation(t *testing.T) {
	setTestConfig()
	// Custom types "owner" and "agent" skip first-class pairing rules entirely.
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "user-111", Type: "owner", Status: "APPROVED"},
		{UserID: "agent-ai", Type: "agent", Status: "RECORDED"},
	})
	require.NoError(t, err)
}

func TestValidateAuthTypeConstraints_AllRecordedRejected(t *testing.T) {
	setTestConfig()
	// Universal rule: at least one authorization must have a non-RECORDED status.
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "agent-ai", Type: "agent", Status: "RECORDED"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one authorization must have an active status")
}

func TestValidateAuthTypeConstraints_AllRecordedMultiple(t *testing.T) {
	setTestConfig()
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "agent-1", Type: "agent", Status: "RECORDED"},
		{UserID: "agent-2", Type: "carer", Status: "RECORDED"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one authorization must have an active status")
}

func TestValidateAuthTypeConstraints_CustomWithApprovedAndRecorded(t *testing.T) {
	setTestConfig()
	// owner:APPROVED + agent:RECORDED → passes both universal and custom rules
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "user-111", Type: "owner", Status: "APPROVED"},
		{UserID: "agent-ai", Type: "agent", Status: "RECORDED"},
	})
	require.NoError(t, err)
}

func TestValidateAuthTypeConstraints_EmptyStatusDefaultsToApproved(t *testing.T) {
	setTestConfig()
	// Empty status defaults to APPROVED at the service layer, so it counts as non-RECORDED.
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "user-1", Type: "primary"},
	})
	require.NoError(t, err)
}

func TestValidateAuthTypeConstraints_MultipleDelegatesAndSubjects(t *testing.T) {
	setTestConfig()
	// Multiple delegates + multiple subjects is valid.
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "parent-1", Type: "delegate", Status: "APPROVED"},
		{UserID: "parent-2", Type: "delegate", Status: "APPROVED"},
		{UserID: "child-1", Type: "delegate_subject", Status: "RECORDED"},
		{UserID: "child-2", Type: "delegate_subject", Status: "RECORDED"},
	})
	require.NoError(t, err)
}

func TestValidateAuthTypeConstraints_DefaultMixedWithDelegate(t *testing.T) {
	setTestConfig()
	// "default" is treated as primary, so mixing with delegate should fail.
	err := ValidateAuthTypeConstraints([]model.AuthorizationRequest{
		{UserID: "user-1", Status: "APPROVED"},                              // type="" → "default" → primary
		{UserID: "father-111", Type: "delegate", Status: "APPROVED"},        // delegate
		{UserID: "child-333", Type: "delegate_subject", Status: "RECORDED"}, // delegate_subject
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "'primary' cannot be mixed with 'delegate'")
}
