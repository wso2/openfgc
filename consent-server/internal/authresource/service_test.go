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
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/consent-server/internal/authresource/model"
	"github.com/wso2/openfgc/consent-server/internal/system/config"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
	"github.com/wso2/openfgc/consent-server/tests/mocks/stores/interfacesmock"
)

func TestMain(m *testing.M) {
	cfg := &config.Config{
		Consent: config.ConsentConfig{
			AuthStatusMappings: config.AuthStatusMappings{
				ApprovedState:      "APPROVED",
				RejectedState:      "REJECTED",
				CreatedState:       "CREATED",
				SystemExpiredState: "SYSTEM_EXPIRED",
				SystemRevokedState: "SYSTEM_REVOKED",
			},
		},
	}
	config.SetGlobal(cfg)
	os.Exit(m.Run())
}

// newTestSvc wires a service with mock stores. Pass nil for stores not used by the test.
func newTestSvc(
	ar *interfacesmock.AuthResourceStore,
	cs *interfacesmock.ConsentStore,
) *authResourceService {
	return &authResourceService{stores: &stores.StoreRegistry{
		AuthResource: ar,
		Consent:      cs,
	}}
}

// =============================================================================
// Model struct construction — quick sanity checks
// =============================================================================

func TestAuthResourceStructure(t *testing.T) {
	userID := "user-123"
	resources := `{"accounts": ["acc1", "acc2"]}`

	authResource := model.AuthResource{
		AuthID:      "auth-123",
		ConsentID:   "consent-456",
		AuthType:    "accounts",
		UserID:      &userID,
		AuthStatus:  "authorized",
		UpdatedTime: 1234567890,
		Resources:   &resources,
		OrgID:       "org-123",
	}

	require.Equal(t, "auth-123", authResource.AuthID)
	require.Equal(t, "consent-456", authResource.ConsentID)
	require.Equal(t, "accounts", authResource.AuthType)
	require.NotNil(t, authResource.UserID)
	require.Equal(t, "user-123", *authResource.UserID)
	require.Equal(t, "authorized", authResource.AuthStatus)
	require.Equal(t, int64(1234567890), authResource.UpdatedTime)
	require.NotNil(t, authResource.Resources)
	require.Equal(t, "org-123", authResource.OrgID)
}

func TestCreateRequestStructure(t *testing.T) {
	userID := "user-123"
	createReq := model.AuthResourceCreateRequest{
		Type:      "accounts",
		UserID:    &userID,
		Status:    "authorized",
		Resources: map[string]interface{}{"accounts": []string{"acc1"}},
	}

	require.Equal(t, "accounts", createReq.Type)
	require.NotNil(t, createReq.UserID)
	require.Equal(t, "user-123", *createReq.UserID)
	require.Equal(t, "authorized", createReq.Status)
	require.NotNil(t, createReq.Resources)
}

func TestUpdateRequestStructure(t *testing.T) {
	updateReq := model.AuthResourceUpdateRequest{
		Status:    "revoked",
		Resources: map[string]interface{}{"reason": "user revoked"},
	}

	require.Equal(t, "revoked", updateReq.Status)
	require.NotNil(t, updateReq.Resources)
}

func TestResponseStructure(t *testing.T) {
	userID := "user-123"
	out := model.AuthResourceOutput{
		AuthID:      "auth-123",
		AuthType:    "accounts",
		UserID:      &userID,
		AuthStatus:  "authorized",
		UpdatedTime: 1234567890,
		Resources:   map[string]interface{}{"accounts": []string{"acc1"}},
	}

	require.Equal(t, "auth-123", out.AuthID)
	require.Equal(t, "accounts", out.AuthType)
	require.NotNil(t, out.UserID)
	require.Equal(t, "authorized", out.AuthStatus)
}

func TestListResponseStructure(t *testing.T) {
	listOut := model.AuthResourceListOutput{
		Data: []model.AuthResourceOutput{
			{AuthID: "auth-1", AuthType: "accounts", AuthStatus: "authorized"},
			{AuthID: "auth-2", AuthType: "payments", AuthStatus: "pending"},
		},
	}

	require.Len(t, listOut.Data, 2)
	require.Equal(t, "auth-1", listOut.Data[0].AuthID)
	require.Equal(t, "auth-2", listOut.Data[1].AuthID)
}

func TestAuthResourceNilFields(t *testing.T) {
	authResource := model.AuthResource{
		AuthID:      "auth-123",
		ConsentID:   "consent-456",
		AuthType:    "accounts",
		AuthStatus:  "authorized",
		UpdatedTime: 1234567890,
		OrgID:       "org-123",
	}

	require.Nil(t, authResource.UserID)
	require.Nil(t, authResource.Resources)
}

func TestCreateRequestOptionalFields(t *testing.T) {
	createReq := model.AuthResourceCreateRequest{
		Type:   "accounts",
		Status: "authorized",
	}

	require.Nil(t, createReq.UserID)
	require.Nil(t, createReq.Resources)
}

func TestUpdateRequestOptionalFields(t *testing.T) {
	updateReq := model.AuthResourceUpdateRequest{}

	require.Empty(t, updateReq.Status)
	require.Nil(t, updateReq.UserID)
	require.Nil(t, updateReq.Resources)
}

// =============================================================================
// buildAuthResourceOutput
// =============================================================================

func TestBuildAuthResourceOutput_WithResources(t *testing.T) {
	resources := `{"accounts":["acc-1"]}`
	ar := &model.AuthResource{
		AuthID:      "auth-1",
		ConsentID:   "consent-1",
		AuthType:    "authorisation",
		AuthStatus:  "APPROVED",
		UpdatedTime: 1000,
		Resources:   &resources,
		OrgID:       "org-1",
	}

	out := buildAuthResourceOutput(ar)

	require.Equal(t, "auth-1", out.AuthID)
	require.Equal(t, "consent-1", out.ConsentID)
	require.Equal(t, "authorisation", out.AuthType)
	require.Equal(t, "APPROVED", out.AuthStatus)
	require.Equal(t, int64(1000), out.UpdatedTime)
	require.Equal(t, "org-1", out.OrgID)
	require.NotNil(t, out.Resources)
}

func TestBuildAuthResourceOutput_NilResources(t *testing.T) {
	ar := &model.AuthResource{AuthID: "auth-2", Resources: nil}
	out := buildAuthResourceOutput(ar)
	require.Nil(t, out.Resources)
}

func TestBuildAuthResourceOutput_EmptyResourcesString(t *testing.T) {
	empty := ""
	ar := &model.AuthResource{AuthID: "auth-3", Resources: &empty}
	out := buildAuthResourceOutput(ar)
	require.Nil(t, out.Resources)
}

func TestBuildAuthResourceOutput_InvalidResourcesJSON(t *testing.T) {
	// Invalid JSON is logged and resources is left nil — no panic.
	bad := "not-json"
	ar := &model.AuthResource{AuthID: "auth-4", Resources: &bad}
	out := buildAuthResourceOutput(ar)
	require.Nil(t, out.Resources)
}

func TestBuildAuthResourceOutput_WithUserID(t *testing.T) {
	uid := "user@example.com"
	ar := &model.AuthResource{AuthID: "auth-5", UserID: &uid}
	out := buildAuthResourceOutput(ar)
	require.NotNil(t, out.UserID)
	require.Equal(t, "user@example.com", *out.UserID)
}

// =============================================================================
// validateOrgID
// =============================================================================

func TestValidateOrgID(t *testing.T) {
	svc := &authResourceService{}

	require.NotNil(t, svc.validateOrgID(""))
	require.NotNil(t, svc.validateOrgID(strings.Repeat("x", 256)))
	require.Nil(t, svc.validateOrgID("org-001"))
	require.Nil(t, svc.validateOrgID(strings.Repeat("x", 255)))
}

// =============================================================================
// validateConsentIDAndOrgID
// =============================================================================

func TestValidateConsentIDAndOrgID(t *testing.T) {
	svc := &authResourceService{}

	require.NotNil(t, svc.validateConsentIDAndOrgID("", "org-1"))
	require.NotNil(t, svc.validateConsentIDAndOrgID(strings.Repeat("x", 256), "org-1"))
	require.NotNil(t, svc.validateConsentIDAndOrgID("consent-1", ""))
	require.Nil(t, svc.validateConsentIDAndOrgID("consent-1", "org-1"))
}

// =============================================================================
// validateAuthIDAndOrgID
// =============================================================================

func TestValidateAuthIDAndOrgID(t *testing.T) {
	svc := &authResourceService{}

	require.NotNil(t, svc.validateAuthIDAndOrgID("", "org-1"))
	require.NotNil(t, svc.validateAuthIDAndOrgID(strings.Repeat("x", 256), "org-1"))
	require.NotNil(t, svc.validateAuthIDAndOrgID("auth-1", ""))
	require.Nil(t, svc.validateAuthIDAndOrgID("auth-1", "org-1"))
}

// =============================================================================
// GetAuthResource
// =============================================================================

func TestGetAuthResource_ValidationFailures(t *testing.T) {
	svc := &authResourceService{}
	ctx := context.Background()

	_, err := svc.GetAuthResource(ctx, "", "consent-1", "org-1")
	require.NotNil(t, err)

	_, err = svc.GetAuthResource(ctx, "auth-1", "", "org-1")
	require.NotNil(t, err)

	_, err = svc.GetAuthResource(ctx, "auth-1", "consent-1", "")
	require.NotNil(t, err)
}

func TestGetAuthResource_StoreError(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").
		Return(nil, errors.New("db connection failed"))

	svc := newTestSvc(arStore, nil)
	_, err := svc.GetAuthResource(context.Background(), "auth-1", "consent-1", "org-1")
	require.NotNil(t, err)
}

func TestGetAuthResource_NotFound(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").
		Return((*model.AuthResource)(nil), nil)

	svc := newTestSvc(arStore, nil)
	_, err := svc.GetAuthResource(context.Background(), "auth-1", "consent-1", "org-1")
	require.NotNil(t, err)
}

func TestGetAuthResource_WrongConsent(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").
		Return(&model.AuthResource{AuthID: "auth-1", ConsentID: "other-consent"}, nil)

	svc := newTestSvc(arStore, nil)
	_, err := svc.GetAuthResource(context.Background(), "auth-1", "consent-1", "org-1")
	require.NotNil(t, err)
}

func TestGetAuthResource_Success(t *testing.T) {
	resources := `{"key":"value"}`
	uid := "user-1"
	ar := &model.AuthResource{
		AuthID:      "auth-1",
		ConsentID:   "consent-1",
		AuthType:    "authorisation",
		UserID:      &uid,
		AuthStatus:  "APPROVED",
		UpdatedTime: 9999,
		Resources:   &resources,
		OrgID:       "org-1",
	}

	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").Return(ar, nil)

	svc := newTestSvc(arStore, nil)
	out, svcErr := svc.GetAuthResource(context.Background(), "auth-1", "consent-1", "org-1")
	require.Nil(t, svcErr)
	require.Equal(t, "auth-1", out.AuthID)
	require.Equal(t, "APPROVED", out.AuthStatus)
	require.NotNil(t, out.Resources)
}

// =============================================================================
// GetAuthResourcesByConsentID
// =============================================================================

func TestGetAuthResourcesByConsentID_ValidationFailures(t *testing.T) {
	svc := &authResourceService{}
	ctx := context.Background()

	_, err := svc.GetAuthResourcesByConsentID(ctx, "", "org-1")
	require.NotNil(t, err)

	_, err = svc.GetAuthResourcesByConsentID(ctx, "consent-1", "")
	require.NotNil(t, err)
}

func TestGetAuthResourcesByConsentID_StoreError(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByConsentID", context.Background(), "consent-1", "org-1").
		Return(nil, errors.New("store error"))

	svc := newTestSvc(arStore, nil)
	_, err := svc.GetAuthResourcesByConsentID(context.Background(), "consent-1", "org-1")
	require.NotNil(t, err)
}

func TestGetAuthResourcesByConsentID_Empty(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByConsentID", context.Background(), "consent-1", "org-1").
		Return([]model.AuthResource{}, nil)

	svc := newTestSvc(arStore, nil)
	out, svcErr := svc.GetAuthResourcesByConsentID(context.Background(), "consent-1", "org-1")
	require.Nil(t, svcErr)
	require.Empty(t, out.Data)
}

func TestGetAuthResourcesByConsentID_MultipleResults(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByConsentID", context.Background(), "consent-1", "org-1").
		Return([]model.AuthResource{
			{AuthID: "auth-1", ConsentID: "consent-1", AuthStatus: "APPROVED", OrgID: "org-1"},
			{AuthID: "auth-2", ConsentID: "consent-1", AuthStatus: "CREATED", OrgID: "org-1"},
		}, nil)

	svc := newTestSvc(arStore, nil)
	out, svcErr := svc.GetAuthResourcesByConsentID(context.Background(), "consent-1", "org-1")
	require.Nil(t, svcErr)
	require.Len(t, out.Data, 2)
	require.Equal(t, "auth-1", out.Data[0].AuthID)
	require.Equal(t, "auth-2", out.Data[1].AuthID)
}

// =============================================================================
// UpdateAuthResource — validation and pre-transaction paths
// =============================================================================

func TestUpdateAuthResource_ValidationFailures(t *testing.T) {
	svc := &authResourceService{}
	ctx := context.Background()

	_, err := svc.UpdateAuthResource(ctx, "", "consent-1", "org-1", model.UpdateAuthResourceInput{})
	require.NotNil(t, err)

	_, err = svc.UpdateAuthResource(ctx, "auth-1", "", "org-1", model.UpdateAuthResourceInput{})
	require.NotNil(t, err)

	_, err = svc.UpdateAuthResource(ctx, "auth-1", "consent-1", "", model.UpdateAuthResourceInput{})
	require.NotNil(t, err)
}

func TestUpdateAuthResource_StoreError(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").
		Return(nil, errors.New("db error"))

	svc := newTestSvc(arStore, nil)
	_, err := svc.UpdateAuthResource(context.Background(), "auth-1", "consent-1", "org-1",
		model.UpdateAuthResourceInput{AuthType: "re-authorisation"})
	require.NotNil(t, err)
}

func TestUpdateAuthResource_NotFound(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").
		Return((*model.AuthResource)(nil), nil)

	svc := newTestSvc(arStore, nil)
	_, err := svc.UpdateAuthResource(context.Background(), "auth-1", "consent-1", "org-1",
		model.UpdateAuthResourceInput{AuthType: "re-authorisation"})
	require.NotNil(t, err)
}

func TestUpdateAuthResource_WrongConsent(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").
		Return(&model.AuthResource{AuthID: "auth-1", ConsentID: "other-consent"}, nil)

	svc := newTestSvc(arStore, nil)
	_, err := svc.UpdateAuthResource(context.Background(), "auth-1", "consent-1", "org-1",
		model.UpdateAuthResourceInput{AuthType: "re-authorisation"})
	require.NotNil(t, err)
}

func TestUpdateAuthResource_SystemReservedStatusRejected(t *testing.T) {
	// Covers the `if input.AuthStatus != ""` validation branch after GetByID succeeds.
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").
		Return(&model.AuthResource{AuthID: "auth-1", ConsentID: "consent-1"}, nil)

	svc := newTestSvc(arStore, nil)
	_, err := svc.UpdateAuthResource(context.Background(), "auth-1", "consent-1", "org-1",
		model.UpdateAuthResourceInput{AuthStatus: "SYSTEM_REVOKED"})
	require.NotNil(t, err)
}

func TestUpdateAuthResource_ResourcesMarshalError(t *testing.T) {
	// Covers the json.Marshal error branch for resources.
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByID", context.Background(), "auth-1", "org-1").
		Return(&model.AuthResource{AuthID: "auth-1", ConsentID: "consent-1"}, nil)

	svc := newTestSvc(arStore, nil)
	_, err := svc.UpdateAuthResource(context.Background(), "auth-1", "consent-1", "org-1",
		model.UpdateAuthResourceInput{Resources: make(chan int)})
	require.NotNil(t, err)
}

// =============================================================================
// UpdateAllStatusByConsentID — validation paths
// =============================================================================

func TestUpdateAllStatusByConsentID_ValidationFailures(t *testing.T) {
	svc := &authResourceService{}
	ctx := context.Background()

	require.NotNil(t, svc.UpdateAllStatusByConsentID(ctx, "", "org-1", "REVOKED"))
	require.NotNil(t, svc.UpdateAllStatusByConsentID(ctx, "consent-1", "", "REVOKED"))
	require.NotNil(t, svc.UpdateAllStatusByConsentID(ctx, "consent-1", "org-1", ""))
}

// =============================================================================
// CreateAuthResource — validation and pre-transaction paths
// =============================================================================

func TestCreateAuthResource_ValidationFailures(t *testing.T) {
	svc := &authResourceService{}
	ctx := context.Background()

	_, err := svc.CreateAuthResource(ctx, "", "org-1", model.CreateAuthResourceInput{})
	require.NotNil(t, err)

	_, err = svc.CreateAuthResource(ctx, "consent-1", "", model.CreateAuthResourceInput{})
	require.NotNil(t, err)
}

func TestCreateAuthResource_SystemReservedStatusRejected(t *testing.T) {
	svc := &authResourceService{}
	ctx := context.Background()

	_, err := svc.CreateAuthResource(ctx, "consent-1", "org-1", model.CreateAuthResourceInput{
		AuthStatus: "SYSTEM_EXPIRED",
	})
	require.NotNil(t, err)
}

func TestCreateAuthResource_ResourcesMarshalError(t *testing.T) {
	svc := &authResourceService{}
	ctx := context.Background()

	// A channel is not JSON-marshalable.
	_, err := svc.CreateAuthResource(ctx, "consent-1", "org-1", model.CreateAuthResourceInput{
		Resources: make(chan int),
	})
	require.NotNil(t, err)
}

func TestCreateAuthResource_GetByConsentIDError(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByConsentID", context.Background(), "consent-1", "org-1").
		Return(nil, errors.New("store error"))

	svc := newTestSvc(arStore, nil)
	_, err := svc.CreateAuthResource(context.Background(), "consent-1", "org-1",
		model.CreateAuthResourceInput{})
	require.NotNil(t, err)
}

func TestCreateAuthResource_ConsentGetByIDError(t *testing.T) {
	arStore := interfacesmock.NewAuthResourceStore(t)
	arStore.On("GetByConsentID", context.Background(), "consent-1", "org-1").
		Return([]model.AuthResource{}, nil)

	csStore := interfacesmock.NewConsentStore(t)
	csStore.On("GetByID", context.Background(), "consent-1", "org-1").
		Return(nil, errors.New("consent store error"))

	svc := newTestSvc(arStore, csStore)
	_, err := svc.CreateAuthResource(context.Background(), "consent-1", "org-1",
		model.CreateAuthResourceInput{})
	require.NotNil(t, err)
}
