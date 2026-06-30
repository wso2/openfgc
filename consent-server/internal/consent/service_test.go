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
	"database/sql"

	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authmodel "github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/system/config"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/tests/mocks/stores/interfacesmock"
)

// TestMain sets the global config before any test runs so that the DB provider
// initialises with SQLite in-memory when ExecuteTransaction first calls GetConsentDBClient.
func TestMain(m *testing.M) {
	config.SetGlobal(makeTestConfig())
	os.Exit(m.Run())
}

// =============================================================================
// Test helpers
// =============================================================================

const (
	testOrgID     = "org-001"
	testConsentID = "consent-001"
)

var errStoreConsent = errors.New("store error")

type noopTx struct{}

func (noopTx) Exec(dbmodel.DBQuery, ...any) (sql.Result, error) { return nil, nil }
func (noopTx) Query(dbmodel.DBQuery, ...any) (*sql.Rows, error) { return nil, nil }
func (noopTx) Commit() error                                    { return nil }
func (noopTx) Rollback() error                                  { return nil }

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

// =============================================================================
// GetGroupIDsByUserID
// =============================================================================

func TestGetGroupIDsByUserID_Success(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetGroupIDsByUserID", mock.Anything, "user-001", testOrgID).
		Return([]string{"group-001", "group-002"}, nil)

	out, svcErr := svc.GetGroupIDsByUserID(context.Background(), "user-001", testOrgID)
	require.Nil(t, svcErr)
	require.NotNil(t, out)
	require.Equal(t, 2, out.Count)
	require.Equal(t, []string{"group-001", "group-002"}, out.GroupIDs)
}

func TestGetGroupIDsByUserID_EmptyResult(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetGroupIDsByUserID", mock.Anything, "user-001", testOrgID).
		Return([]string{}, nil)

	out, svcErr := svc.GetGroupIDsByUserID(context.Background(), "user-001", testOrgID)
	require.Nil(t, svcErr)
	require.NotNil(t, out)
	require.Equal(t, 0, out.Count)
	require.Empty(t, out.GroupIDs)
}

func TestGetGroupIDsByUserID_StoreError(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetGroupIDsByUserID", mock.Anything, "user-001", testOrgID).
		Return(nil, errStoreConsent)

	out, svcErr := svc.GetGroupIDsByUserID(context.Background(), "user-001", testOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
	require.Equal(t, ErrorInternalServerError.Message, svcErr.Message)
}

// =============================================================================
// GetExpiredConsents
// =============================================================================

func TestGetExpiredConsents_StoreError(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetExpiredConsents", mock.Anything, int64(1000), []string{"ACTIVE"}).
		Return(nil, errStoreConsent)

	consents, svcErr := svc.GetExpiredConsents(context.Background(), 1000, []string{"ACTIVE"})
	require.Nil(t, consents)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, errStoreConsent.Error())
}

func TestGetExpiredConsents_Success(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	expected := []model.Consent{
		{ConsentID: "c-1", OrgID: testOrgID, CurrentStatus: "ACTIVE"},
		{ConsentID: "c-2", OrgID: testOrgID, CurrentStatus: "ACTIVE"},
	}
	cs.On("GetExpiredConsents", mock.Anything, int64(1000), []string{"ACTIVE"}).
		Return(expected, nil)

	consents, svcErr := svc.GetExpiredConsents(context.Background(), 1000, []string{"ACTIVE"})
	require.Nil(t, svcErr)
	require.Len(t, consents, 2)
	require.Equal(t, "c-1", consents[0].ConsentID)
	require.Equal(t, "c-2", consents[1].ConsentID)
}

func TestGetExpiredConsents_Empty(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	cs.On("GetExpiredConsents", mock.Anything, int64(999), []string{"ACTIVE", "CREATED"}).
		Return([]model.Consent{}, nil)

	consents, svcErr := svc.GetExpiredConsents(context.Background(), 999, []string{"ACTIVE", "CREATED"})
	require.Nil(t, svcErr)
	require.Empty(t, consents)
}

// =============================================================================
// ExpireConsent
// =============================================================================

func TestExpireConsent_TransactionError(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	as := interfacesmock.NewAuthResourceStore(t)
	svc := newConsentSvc(t, cs, as)

	consent := makeTestConsent(testConsentID, testOrgID, "ACTIVE")
	cs.On("GetByIDForUpdate", mock.Anything, testConsentID, testOrgID).Return(consent, nil)
	// Mutation step in the transaction fails — should trigger rollback and return ServiceError.
	cs.On("UpdateStatus", mock.Anything, testConsentID, testOrgID, "EXPIRED", mock.AnythingOfType("int64")).
		Return(errStoreConsent)

	svcErr := svc.ExpireConsent(context.Background(), consent, testOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
	// Consent status must not be mutated on failure.
	require.Equal(t, "ACTIVE", consent.CurrentStatus)
}

func TestExpireConsent_Success(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	as := interfacesmock.NewAuthResourceStore(t)
	svc := newConsentSvc(t, cs, as)

	consent := makeTestConsent(testConsentID, testOrgID, "ACTIVE")
	cs.On("GetByIDForUpdate", mock.Anything, testConsentID, testOrgID).Return(consent, nil)
	cs.On("UpdateStatus", mock.Anything, testConsentID, testOrgID, "EXPIRED", mock.AnythingOfType("int64")).
		Return(nil)
	as.On("UpdateAllStatusByConsentID", mock.Anything, testConsentID, testOrgID, "SYSTEM_EXPIRED", mock.AnythingOfType("int64")).
		Return(nil)
	cs.On("CreateStatusAudit", mock.Anything, mock.AnythingOfType("*model.ConsentStatusAudit")).
		Return(nil)

	svcErr := svc.ExpireConsent(context.Background(), consent, testOrgID)
	require.Nil(t, svcErr)
	// Consent should be mutated in-place to reflect the new status.
	require.Equal(t, "EXPIRED", consent.CurrentStatus)
}

// =============================================================================
// Consent History
// =============================================================================

func setConsentHistoryEnabled(t *testing.T, enabled bool) {
	t.Helper()
	cfg := config.Get()
	require.NotNil(t, cfg)
	previous := cfg.Consent.History.Enabled
	cfg.Consent.History.Enabled = enabled
	t.Cleanup(func() { cfg.Consent.History.Enabled = previous })
}

func mockConsentSnapshotLoad(
	cs *interfacesmock.ConsentStore,
	as *interfacesmock.AuthResourceStore,
	consentID string,
	orgID string,
) {
	value := `{"level":"gold"}`
	userID := "user-001"
	resources := `{"accountIds":["acc-1"]}`
	cs.On("GetAttributesByConsentIDTx", mock.Anything, consentID, orgID).
		Return([]model.ConsentAttribute{{ConsentID: consentID, AttKey: "region", AttValue: "EU", OrgID: orgID}}, nil)
	as.On("GetByConsentIDTx", mock.Anything, consentID, orgID).
		Return([]authmodel.AuthResource{{
			AuthID:      "auth-001",
			ConsentID:   consentID,
			AuthType:    "authorisation",
			UserID:      &userID,
			AuthStatus:  "APPROVED",
			UpdatedTime: 1200,
			Resources:   &resources,
			OrgID:       orgID,
		}}, nil)
	cs.On("GetPurposesByConsentIDTx", mock.Anything, consentID, orgID).
		Return([]model.ConsentPurposeRow{{
			ConsentID:        consentID,
			PurposeVersionID: "purpose-version-001",
			PurposeID:        "purpose-001",
			PurposeName:      "beneficiary-access",
			PurposeGroupID:   "group-001",
			PurposeVersion:   1,
			OrgID:            orgID,
		}}, nil)
	cs.On("GetElementApprovalsByConsentIDTx", mock.Anything, consentID, orgID).
		Return([]model.ConsentApprovalRow{{
			ConsentID:         consentID,
			PurposeVersionID:  "purpose-version-001",
			ElementVersionID:  "element-version-001",
			ElementID:         "element-001",
			ElementName:       "payee-id",
			ElementNamespace:  "payments",
			ElementVersionNum: 1,
			ElementType:       "json",
			Mandatory:         true,
			Approved:          true,
			Value:             &value,
			OrgID:             orgID,
		}}, nil)
	cs.On("GetElementPropertiesByConsentIDTx", mock.Anything, consentID, orgID).
		Return(map[string]map[string]string{}, nil)
	cs.On("GetPurposePropertiesByConsentIDTx", mock.Anything, consentID, orgID).
		Return(map[string]map[string]string{}, nil)
}

func TestRecordConsentHistory_BuildsFullSnapshot(t *testing.T) {
	setConsentHistoryEnabled(t, true)

	cs := interfacesmock.NewConsentStore(t)
	as := interfacesmock.NewAuthResourceStore(t)
	svc := newConsentSvc(t, cs, as)

	frequency := 5
	recurring := true
	dataAccessDuration := int64(3600)
	consent := makeTestConsent(testConsentID, testOrgID, "ACTIVE")
	consent.ConsentFrequency = &frequency
	consent.RecurringIndicator = &recurring
	consent.DataAccessValidityDuration = &dataAccessDuration
	actionBy := "group-001"

	cs.On("GetByIDForUpdate", mock.Anything, testConsentID, testOrgID).Return(consent, nil)
	mockConsentSnapshotLoad(cs, as, testConsentID, testOrgID)

	var captured *model.ConsentHistory
	cs.On("CreateHistory", mock.Anything, mock.AnythingOfType("*model.ConsentHistory")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*model.ConsentHistory)
		}).
		Return(nil)

	err := svc.recordConsentHistory(context.Background(), noopTx{}, testConsentID, testOrgID, &actionBy, HistoryReasonConsentUpdated)
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Equal(t, testConsentID, captured.ConsentID)
	require.Equal(t, testOrgID, captured.OrgID)
	require.Equal(t, actionBy, *captured.ActionBy)
	require.Equal(t, string(HistoryReasonConsentUpdated), *captured.Reason)

	var snapshot map[string]interface{}
	require.NoError(t, json.Unmarshal(captured.Snapshot, &snapshot))
	require.Equal(t, testConsentID, snapshot["id"])
	require.Equal(t, "GENERAL", snapshot["type"])
	require.Equal(t, "ACTIVE", snapshot["status"])
	require.Equal(t, "EU", snapshot["attributes"].(map[string]interface{})["region"])
	require.Len(t, snapshot["purposes"].([]interface{}), 1)
	require.Len(t, snapshot["authorizations"].([]interface{}), 1)
}

func TestGetConsentHistory_MapsStoreRecordsToOutput(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	actionBy := "user-001"
	reason := string(HistoryReasonConsentRevoked)
	items := []model.ConsentHistory{{
		HistoryID:  "history-001",
		ConsentID:  testConsentID,
		OrgID:      testOrgID,
		ActionTime: 1700000000000,
		ActionBy:   &actionBy,
		Reason:     &reason,
	}}

	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(makeTestConsent(testConsentID, testOrgID, "ACTIVE"), nil)
	cs.On("GetHistoryByConsentID", mock.Anything, testConsentID, testOrgID, false).Return(items, nil)

	out, svcErr := svc.GetConsentHistory(context.Background(), testConsentID, testOrgID, false)
	require.Nil(t, svcErr)
	require.NotNil(t, out)
	require.Equal(t, testConsentID, out.ID)
	require.Len(t, out.History, 1)
	require.Equal(t, "history-001", out.History[0].HistoryID)
	require.Equal(t, testConsentID, out.History[0].ConsentID)
	require.Equal(t, testOrgID, out.History[0].OrgID)
	require.Equal(t, int64(1700000000000), out.History[0].ActionTime)
	require.Equal(t, actionBy, *out.History[0].ActionBy)
	require.Equal(t, reason, *out.History[0].Reason)
}

func TestGetConsentHistory_ExcludesSnapshotsWhenDisabled(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	items := []model.ConsentHistory{{
		HistoryID: "history-001",
		ConsentID: testConsentID,
		OrgID:     testOrgID,
		Snapshot:  []byte(`{"id":"old"}`),
	}}
	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(makeTestConsent(testConsentID, testOrgID, "ACTIVE"), nil)
	cs.On("GetHistoryByConsentID", mock.Anything, testConsentID, testOrgID, false).Return(items, nil)

	out, svcErr := svc.GetConsentHistory(context.Background(), testConsentID, testOrgID, false)
	require.Nil(t, svcErr)
	require.Len(t, out.History, 1)
	require.Empty(t, out.History[0].Snapshot)
}

func TestGetConsentHistory_IncludesSnapshotsWhenRequested(t *testing.T) {
	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	snapshot := []byte(`{"id":"old"}`)
	items := []model.ConsentHistory{{
		HistoryID: "history-001",
		ConsentID: testConsentID,
		OrgID:     testOrgID,
		Snapshot:  snapshot,
	}}
	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(makeTestConsent(testConsentID, testOrgID, "ACTIVE"), nil)
	cs.On("GetHistoryByConsentID", mock.Anything, testConsentID, testOrgID, true).Return(items, nil)

	out, svcErr := svc.GetConsentHistory(context.Background(), testConsentID, testOrgID, true)
	require.Nil(t, svcErr)
	require.Len(t, out.History, 1)
	require.JSONEq(t, string(snapshot), string(out.History[0].Snapshot))
}

func TestRecordConsentHistory_SkipsWhenHistoryDisabled(t *testing.T) {
	setConsentHistoryEnabled(t, false)

	cs := interfacesmock.NewConsentStore(t)
	svc := newConsentSvc(t, cs, nil)

	err := svc.recordConsentHistory(context.Background(), nil, testConsentID, testOrgID, nil, HistoryReasonConsentUpdated)
	require.NoError(t, err)
	cs.AssertNotCalled(t, "GetByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
	cs.AssertNotCalled(t, "CreateHistory", mock.Anything, mock.Anything)
}

func TestUpdateConsent_HistoryInsertFailureAbortsUpdate(t *testing.T) {
	setConsentHistoryEnabled(t, true)

	cs := interfacesmock.NewConsentStore(t)
	as := interfacesmock.NewAuthResourceStore(t)
	svc := newConsentSvc(t, cs, as)

	existing := makeTestConsent(testConsentID, testOrgID, "ACTIVE")
	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(existing, nil).Once()
	cs.On("GetByIDForUpdate", mock.Anything, testConsentID, testOrgID).Return(existing, nil)
	mockConsentSnapshotLoad(cs, as, testConsentID, testOrgID)

	var captured *model.ConsentHistory
	cs.On("CreateHistory", mock.Anything, mock.AnythingOfType("*model.ConsentHistory")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*model.ConsentHistory)
		}).
		Return(errStoreConsent)

	out, svcErr := svc.UpdateConsent(context.Background(), testConsentID, "group-001", testOrgID, model.UpdateConsentInput{ConsentType: "UPDATED"})
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.NotNil(t, captured)
	require.Equal(t, string(HistoryReasonConsentUpdated), *captured.Reason)
	require.Equal(t, "group-001", *captured.ActionBy)
	cs.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestRevokeConsent_UsesRevokedHistoryReason(t *testing.T) {
	setConsentHistoryEnabled(t, true)

	cs := interfacesmock.NewConsentStore(t)
	as := interfacesmock.NewAuthResourceStore(t)
	svc := newConsentSvc(t, cs, as)

	existing := makeTestConsent(testConsentID, testOrgID, "ACTIVE")
	cs.On("GetByID", mock.Anything, testConsentID, testOrgID).Return(existing, nil)
	cs.On("GetByIDForUpdate", mock.Anything, testConsentID, testOrgID).Return(existing, nil)
	mockConsentSnapshotLoad(cs, as, testConsentID, testOrgID)

	var captured *model.ConsentHistory
	cs.On("CreateHistory", mock.Anything, mock.AnythingOfType("*model.ConsentHistory")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*model.ConsentHistory)
		}).
		Return(errStoreConsent)

	out, svcErr := svc.RevokeConsent(context.Background(), testConsentID, testOrgID, model.ConsentRevokeInput{ActionBy: "admin-user", Reason: "user request"})
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.NotNil(t, captured)
	require.Equal(t, string(HistoryReasonConsentRevoked), *captured.Reason)
	require.Equal(t, "admin-user", *captured.ActionBy)
	cs.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestExpireConsent_UsesExpiredHistoryReason(t *testing.T) {
	setConsentHistoryEnabled(t, true)

	cs := interfacesmock.NewConsentStore(t)
	as := interfacesmock.NewAuthResourceStore(t)
	svc := newConsentSvc(t, cs, as)

	consent := makeTestConsent(testConsentID, testOrgID, "ACTIVE")
	cs.On("GetByIDForUpdate", mock.Anything, testConsentID, testOrgID).Return(consent, nil)
	mockConsentSnapshotLoad(cs, as, testConsentID, testOrgID)

	var captured *model.ConsentHistory
	cs.On("CreateHistory", mock.Anything, mock.AnythingOfType("*model.ConsentHistory")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*model.ConsentHistory)
		}).
		Return(errStoreConsent)

	svcErr := svc.ExpireConsent(context.Background(), consent, testOrgID)
	require.NotNil(t, svcErr)
	require.NotNil(t, captured)
	require.Equal(t, string(HistoryReasonConsentExpired), *captured.Reason)
	require.Equal(t, "SYSTEM", *captured.ActionBy)
	cs.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	as.AssertNotCalled(t, "UpdateAllStatusByConsentID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
