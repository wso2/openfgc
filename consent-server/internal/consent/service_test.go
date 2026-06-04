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

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authmodel "github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/system/config"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/tests/mocks/stores/interfacesmock"
)

// =============================================================================
// Test helpers
// =============================================================================

const (
	testOrgID     = "org-001"
	testConsentID = "consent-001"
)

var errStoreConsent = errors.New("store error")

func newConsentSvc(t *testing.T, cs *interfacesmock.ConsentStore, as *interfacesmock.AuthResourceStore) *consentService {
	t.Helper()
	return &consentService{stores: &stores.StoreRegistry{
		Consent:      cs,
		AuthResource: as,
	}}
}

func makeTestConsent(id, orgID, status string) *model.Consent {
	return &model.Consent{
		ConsentID:     id,
		OrgID:         orgID,
		GroupID:       "group-001",
		ConsentType:   "GENERAL",
		CurrentStatus: status,
		CreatedTime:   1000,
		UpdatedTime:   1000,
	}
}

func makeTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Port: 9090},
		Database: config.DatabasesConfig{
			Consent: config.DatabaseConfig{
				Type:     "sqlite",
				Path:     ":memory:",
				Hostname: "localhost",
				Database: "consent_mgt",
			},
		},
		Consent: config.ConsentConfig{
			StatusMappings: config.ConsentStatusMappings{
				ActiveStatus:   "ACTIVE",
				ExpiredStatus:  "EXPIRED",
				RevokedStatus:  "REVOKED",
				CreatedStatus:  "CREATED",
				RejectedStatus: "REJECTED",
			},
			AuthStatusMappings: config.AuthStatusMappings{
				ApprovedState:      "APPROVED",
				RejectedState:      "REJECTED",
				CreatedState:       "CREATED",
				SystemExpiredState: "SYSTEM_EXPIRED",
				SystemRevokedState: "SYSTEM_REVOKED",
			},
		},
	}
}

// =============================================================================
// parseVersionString
// =============================================================================

func TestParseVersionString_Valid(t *testing.T) {
	n, err := parseVersionString("v1")
	require.NoError(t, err)
	require.Equal(t, 1, n)
}

func TestParseVersionString_LargerNumber(t *testing.T) {
	n, err := parseVersionString("v42")
	require.NoError(t, err)
	require.Equal(t, 42, n)
}

func TestParseVersionString_EmptyString(t *testing.T) {
	_, err := parseVersionString("")
	require.Error(t, err)
}

func TestParseVersionString_NoPrefix(t *testing.T) {
	_, err := parseVersionString("1")
	require.Error(t, err)
}

func TestParseVersionString_Zero(t *testing.T) {
	_, err := parseVersionString("v0")
	require.Error(t, err)
}

func TestParseVersionString_Negative(t *testing.T) {
	_, err := parseVersionString("v-1")
	require.Error(t, err)
}

func TestParseVersionString_NonNumeric(t *testing.T) {
	_, err := parseVersionString("vabc")
	require.Error(t, err)
}

// =============================================================================
// formatVersion
// =============================================================================

func TestFormatVersion_One(t *testing.T) {
	require.Equal(t, "v1", formatVersion(1))
}

func TestFormatVersion_Three(t *testing.T) {
	require.Equal(t, "v3", formatVersion(3))
}

func TestFormatVersion_LargeNumber(t *testing.T) {
	require.Equal(t, "v100", formatVersion(100))
}

// =============================================================================
// valueToString
// =============================================================================

func TestValueToString_Nil(t *testing.T) {
	require.Equal(t, "", valueToString(nil))
}

func TestValueToString_String(t *testing.T) {
	require.Equal(t, "hello", valueToString("hello"))
}

func TestValueToString_EmptyString(t *testing.T) {
	require.Equal(t, "", valueToString(""))
}

func TestValueToString_Int(t *testing.T) {
	result := valueToString(42)
	require.Equal(t, "42", result)
}

func TestValueToString_Map(t *testing.T) {
	m := map[string]interface{}{"key": "value"}
	result := valueToString(m)
	require.Contains(t, result, `"key"`)
	require.Contains(t, result, `"value"`)
}

func TestValueToString_Bool(t *testing.T) {
	require.Equal(t, "true", valueToString(true))
}

// =============================================================================
// authResourceToOutput
// =============================================================================

func TestAuthResourceToOutput_NoResources(t *testing.T) {
	ar := authmodel.AuthResource{
		AuthID:      "auth-001",
		ConsentID:   testConsentID,
		AuthType:    "default",
		AuthStatus:  "APPROVED",
		UpdatedTime: 1000,
		OrgID:       testOrgID,
	}
	out := authResourceToOutput(ar)
	require.Equal(t, "auth-001", out.AuthID)
	require.Equal(t, testConsentID, out.ConsentID)
	require.Equal(t, "default", out.AuthType)
	require.Equal(t, "APPROVED", out.AuthStatus)
	require.Equal(t, int64(1000), out.UpdatedTime)
	require.Equal(t, testOrgID, out.OrgID)
	require.Nil(t, out.Resources)
}

func TestAuthResourceToOutput_WithJSONResources(t *testing.T) {
	jsonStr := `{"scope":"read"}`
	ar := authmodel.AuthResource{
		AuthID:    "auth-002",
		ConsentID: testConsentID,
		OrgID:     testOrgID,
		Resources: &jsonStr,
	}
	out := authResourceToOutput(ar)
	require.NotNil(t, out.Resources)
	resourceMap, ok := out.Resources.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "read", resourceMap["scope"])
}

func TestAuthResourceToOutput_EmptyResourcesString(t *testing.T) {
	empty := ""
	ar := authmodel.AuthResource{
		AuthID:    "auth-003",
		ConsentID: testConsentID,
		OrgID:     testOrgID,
		Resources: &empty,
	}
	out := authResourceToOutput(ar)
	require.Nil(t, out.Resources)
}

func TestAuthResourceToOutput_WithUserID(t *testing.T) {
	userID := "user-001"
	ar := authmodel.AuthResource{
		AuthID:    "auth-004",
		ConsentID: testConsentID,
		OrgID:     testOrgID,
		UserID:    &userID,
	}
	out := authResourceToOutput(ar)
	require.NotNil(t, out.UserID)
	require.Equal(t, "user-001", *out.UserID)
}

// =============================================================================
// GetConsent
// =============================================================================

func TestGetConsent_NotFound(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(nil, nil)

	out, svcErr := svc.GetConsent(context.Background(), testConsentID, testOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorConsentNotFound.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, testConsentID)
}

func TestGetConsent_StoreError(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(nil, errStoreConsent)

	out, svcErr := svc.GetConsent(context.Background(), testConsentID, testOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
}

func TestGetConsent_Success(t *testing.T) {
	config.SetGlobal(makeTestConfig())

	cs := interfacesmock.NewConsentStore(t)
	as := interfacesmock.NewAuthResourceStore(t)
	svc := newConsentSvc(t, cs, as)

	consent := makeTestConsent(testConsentID, testOrgID, "ACTIVE")
	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(consent, nil)
	cs.On("GetAttributesByConsentID", mock.Anything, testConsentID, testOrgID).Return([]model.ConsentAttribute{}, nil)
	cs.On("GetPurposesByConsentID", mock.Anything, testConsentID, testOrgID).Return([]model.ConsentPurposeRow{}, nil)
	cs.On("GetElementApprovalsByConsentID", mock.Anything, testConsentID, testOrgID).Return([]model.ConsentApprovalRow{}, nil)
	cs.On("GetElementPropertiesByConsentID", mock.Anything, testConsentID, testOrgID).Return(map[string]map[string]string{}, nil)
	cs.On("GetPurposePropertiesByConsentID", mock.Anything, testConsentID, testOrgID).Return(map[string]map[string]string{}, nil)
	as.On("GetByConsentID", mock.Anything, testConsentID, testOrgID).Return([]authmodel.AuthResource{}, nil)

	out, svcErr := svc.GetConsent(context.Background(), testConsentID, testOrgID)
	require.Nil(t, svcErr)
	require.NotNil(t, out)
	require.Equal(t, testConsentID, out.ConsentID)
	require.Equal(t, testOrgID, out.OrgID)
	require.Equal(t, "ACTIVE", out.CurrentStatus)
	require.Equal(t, "GENERAL", out.ConsentType)
	require.Empty(t, out.Purposes)
	require.Empty(t, out.Authorizations)
}

// =============================================================================
// RevokeConsent — pre-transaction validation paths
// =============================================================================

func TestRevokeConsent_EmptyActionBy(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	out, svcErr := svc.RevokeConsent(context.Background(), testConsentID, testOrgID,
		model.ConsentRevokeInput{ActionBy: ""})
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorValidationFailed.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, "actionBy is required")
}

func TestRevokeConsent_ConsentNotFound(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(nil, nil)

	out, svcErr := svc.RevokeConsent(context.Background(), testConsentID, testOrgID,
		model.ConsentRevokeInput{ActionBy: "user-001"})
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorConsentNotFound.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, testConsentID)
}

func TestRevokeConsent_StoreError(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(nil, errStoreConsent)

	out, svcErr := svc.RevokeConsent(context.Background(), testConsentID, testOrgID,
		model.ConsentRevokeInput{ActionBy: "user-001"})
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
}

func TestRevokeConsent_AlreadyRevoked(t *testing.T) {
	config.SetGlobal(makeTestConfig())

	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	consent := makeTestConsent(testConsentID, testOrgID, "REVOKED")
	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(consent, nil)

	out, svcErr := svc.RevokeConsent(context.Background(), testConsentID, testOrgID,
		model.ConsentRevokeInput{ActionBy: "user-001"})
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorConsentAlreadyRevoked.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, testConsentID)
}

// =============================================================================
// ValidateConsent — pre-store validation paths
// =============================================================================

func TestValidateConsent_EmptyConsentID(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	out, svcErr := svc.ValidateConsent(context.Background(),
		model.ConsentValidateInput{ConsentID: ""}, testOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorValidationFailed.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, "consentId is required")
}

func TestValidateConsent_ConsentNotFound(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(nil, nil)

	out, svcErr := svc.ValidateConsent(context.Background(),
		model.ConsentValidateInput{ConsentID: testConsentID}, testOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorConsentNotFound.Code, svcErr.Code)
}

func TestValidateConsent_StoreError(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(nil, errStoreConsent)

	out, svcErr := svc.ValidateConsent(context.Background(),
		model.ConsentValidateInput{ConsentID: testConsentID}, testOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
}

// =============================================================================
// SearchConsentsByAttribute
// =============================================================================

func TestSearchConsentsByAttribute_WithValue(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetConsentIDsByAttribute", mock.Anything, "userId", "user-001", testOrgID).
		Return([]string{"c-1", "c-2"}, nil)

	out, svcErr := svc.SearchConsentsByAttribute(context.Background(), "userId", "user-001", testOrgID)
	require.Nil(t, svcErr)
	require.NotNil(t, out)
	require.Equal(t, 2, out.Count)
	require.ElementsMatch(t, []string{"c-1", "c-2"}, out.ConsentIDs)
}

func TestSearchConsentsByAttribute_WithoutValue(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetConsentIDsByAttributeKey", mock.Anything, "userId", testOrgID).
		Return([]string{"c-1", "c-2", "c-3"}, nil)

	out, svcErr := svc.SearchConsentsByAttribute(context.Background(), "userId", "", testOrgID)
	require.Nil(t, svcErr)
	require.NotNil(t, out)
	require.Equal(t, 3, out.Count)
	require.Len(t, out.ConsentIDs, 3)
}

func TestSearchConsentsByAttribute_WithValueStoreError(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetConsentIDsByAttribute", mock.Anything, "userId", "user-001", testOrgID).
		Return(nil, errStoreConsent)

	out, svcErr := svc.SearchConsentsByAttribute(context.Background(), "userId", "user-001", testOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
}

func TestSearchConsentsByAttribute_WithoutValueStoreError(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetConsentIDsByAttributeKey", mock.Anything, "userId", testOrgID).
		Return(nil, errStoreConsent)

	out, svcErr := svc.SearchConsentsByAttribute(context.Background(), "userId", "", testOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
}

func TestSearchConsentsByAttribute_EmptyResult(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetConsentIDsByAttribute", mock.Anything, "unknownKey", "someValue", testOrgID).
		Return([]string{}, nil)

	out, svcErr := svc.SearchConsentsByAttribute(context.Background(), "unknownKey", "someValue", testOrgID)
	require.Nil(t, svcErr)
	require.NotNil(t, out)
	require.Equal(t, 0, out.Count)
	require.Empty(t, out.ConsentIDs)
}
