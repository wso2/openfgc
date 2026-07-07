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

package consentpurpose

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	elementmodel "github.com/wso2/openfgc/consent-server/internal/consentelement/model"
	purposemodel "github.com/wso2/openfgc/consent-server/internal/consentpurpose/model"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
	"github.com/wso2/openfgc/consent-server/tests/mocks/stores/interfacesmock"
)

// =============================================================================
// Test helpers
// =============================================================================

const (
	svcOrgID     = "org-001"
	svcPurposeID = "purpose-001"
)

var errStore = errors.New("store error")

func pStr(s string) *string { return &s }

func newSvc(t *testing.T, ps *interfacesmock.ConsentPurposeStore, es *interfacesmock.ConsentElementStore) *consentPurposeService {
	t.Helper()
	return &consentPurposeService{stores: &stores.StoreRegistry{
		ConsentPurpose: ps,
		ConsentElement: es,
	}}
}

func makeEV(id, versionID string, versionNum int) *elementmodel.ElementVersion {
	return &elementmodel.ElementVersion{ID: id, VersionID: versionID, VersionNum: versionNum}
}

func makePV(id, versionID, name string, versionNum int) *purposemodel.PurposeVersion {
	return &purposemodel.PurposeVersion{
		ID: id, VersionID: versionID, Name: name,
		GroupID: svcOrgID, VersionNum: versionNum, OrgID: svcOrgID,
	}
}

// =============================================================================
// validatePurposeInput
// =============================================================================

func TestValidatePurposeInput_Name(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	t.Run("valid name", func(t *testing.T) {
		require.Nil(t, svc.validatePurposeInput("Marketing", nil))
	})

	t.Run("empty name is rejected", func(t *testing.T) {
		svcErr := svc.validatePurposeInput("", nil)
		require.NotNil(t, svcErr)
		require.Contains(t, svcErr.Description, "name is required")
	})

	t.Run("255-char name is accepted", func(t *testing.T) {
		require.Nil(t, svc.validatePurposeInput(strings.Repeat("a", 255), nil))
	})

	t.Run("256-char name is rejected", func(t *testing.T) {
		svcErr := svc.validatePurposeInput(strings.Repeat("a", 256), nil)
		require.NotNil(t, svcErr)
		require.Contains(t, svcErr.Description, "name must not exceed 255 characters")
	})
}

func TestValidatePurposeInput_Description(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	t.Run("nil description is accepted", func(t *testing.T) {
		require.Nil(t, svc.validatePurposeInput("Name", nil))
	})

	t.Run("1024-char description is accepted", func(t *testing.T) {
		require.Nil(t, svc.validatePurposeInput("Name", pStr(strings.Repeat("a", 1024))))
	})

	t.Run("1025-char description is rejected", func(t *testing.T) {
		svcErr := svc.validatePurposeInput("Name", pStr(strings.Repeat("a", 1025)))
		require.NotNil(t, svcErr)
		require.Contains(t, svcErr.Description, "description must not exceed 1024 characters")
	})
}

// =============================================================================
// pvToOutput
// =============================================================================

func TestPvToOutput_Nil(t *testing.T) {
	require.Nil(t, pvToOutput(nil))
}

func TestPvToOutput_WithoutElements(t *testing.T) {
	pv := makePV(svcPurposeID, "vid-1", "Marketing", 2)
	pv.DisplayName = pStr("Marketing Display")
	pv.Properties = map[string]string{"k": "v"}

	out := pvToOutput(pv)
	require.NotNil(t, out)
	require.Equal(t, svcPurposeID, out.ID)
	require.Equal(t, "vid-1", out.VersionID)
	require.Equal(t, "Marketing", out.Name)
	require.Equal(t, svcOrgID, out.GroupID)
	require.Equal(t, 2, out.VersionNum)
	require.Equal(t, "Marketing Display", *out.DisplayName)
	require.Equal(t, map[string]string{"k": "v"}, out.Properties)
	require.Empty(t, out.Elements)
}

func TestPvToOutput_WithElements(t *testing.T) {
	pv := makePV(svcPurposeID, "vid-1", "Marketing", 1)
	pv.Elements = []purposemodel.PurposeMappedElement{
		{ElementVersionID: "evid-1", ElementID: "eid-1", Name: "email", Namespace: "default", VersionNum: 1, Mandatory: true},
		{ElementVersionID: "evid-2", ElementID: "eid-2", Name: "phone", Namespace: "contact", VersionNum: 3, Mandatory: false},
	}

	out := pvToOutput(pv)
	require.Len(t, out.Elements, 2)
	require.Equal(t, "evid-1", out.Elements[0].ElementVersionID)
	require.Equal(t, "email", out.Elements[0].Name)
	require.Equal(t, "default", out.Elements[0].Namespace)
	require.Equal(t, 1, out.Elements[0].VersionNum)
	require.True(t, out.Elements[0].Mandatory)
	require.Equal(t, "phone", out.Elements[1].Name)
	require.False(t, out.Elements[1].Mandatory)
}

// =============================================================================
// validateAndResolveElements
// =============================================================================

func TestValidateAndResolveElements_EmptyRefs(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	_, svcErr := svc.validateAndResolveElements(context.Background(), nil, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInvalidPurposeElements.Code, svcErr.Code)
}

func TestValidateAndResolveElements_DuplicateRef(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	ev := makeEV("eid-1", "evid-1", 1)
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(ev, nil)

	refs := []purposemodel.ElementRef{
		{Name: "email", Namespace: "default"},
		{Name: "email", Namespace: "default"},
	}
	_, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.NotNil(t, svcErr)
	require.Contains(t, svcErr.Description, "duplicate element reference")
}

func TestValidateAndResolveElements_SameNameDifferentVersion(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	ev := makeEV("eid-1", "evid-1", 2)
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(ev, nil)
	es.On("GetVersion", mock.Anything, "eid-1", 1, svcOrgID).Return(makeEV("eid-1", "evid-v1", 1), nil)
	es.On("GetVersion", mock.Anything, "eid-1", 2, svcOrgID).Return(makeEV("eid-1", "evid-v2", 2), nil)

	v1, v2 := 1, 2
	refs := []purposemodel.ElementRef{
		{Name: "email", Namespace: "default", Version: &v1},
		{Name: "email", Namespace: "default", Version: &v2},
	}
	resolved, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.Nil(t, svcErr)
	require.Len(t, resolved, 2)
}

func TestValidateAndResolveElements_DefaultNamespace(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	ev := makeEV("eid-1", "evid-1", 1)
	// Namespace should default to "default" when empty.
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(ev, nil)

	refs := []purposemodel.ElementRef{{Name: "email"}} // no namespace
	resolved, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.Nil(t, svcErr)
	require.Len(t, resolved, 1)
	require.Equal(t, "default", resolved[0].Namespace)
}

func TestValidateAndResolveElements_ElementStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(nil, errStore)

	refs := []purposemodel.ElementRef{{Name: "email"}}
	_, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
}

func TestValidateAndResolveElements_ElementNotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(nil, nil)

	refs := []purposemodel.ElementRef{{Name: "email"}}
	_, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.NotNil(t, svcErr)
	require.Contains(t, svcErr.Description, "does not exist")
}

func TestValidateAndResolveElements_SpecificVersionNotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	ev := makeEV("eid-1", "evid-latest", 3)
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(ev, nil)
	v := 99
	es.On("GetVersion", mock.Anything, "eid-1", 99, svcOrgID).Return(nil, nil)

	refs := []purposemodel.ElementRef{{Name: "email", Version: &v}}
	_, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.NotNil(t, svcErr)
	require.Contains(t, svcErr.Description, "version v99 does not exist")
}

func TestValidateAndResolveElements_SpecificVersionStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	ev := makeEV("eid-1", "evid-latest", 3)
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(ev, nil)
	v := 2
	es.On("GetVersion", mock.Anything, "eid-1", 2, svcOrgID).Return(nil, errStore)

	refs := []purposemodel.ElementRef{{Name: "email", Version: &v}}
	_, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInternalServerError.Code, svcErr.Code)
}

func TestValidateAndResolveElements_Success_LatestVersion(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	ev := makeEV("eid-1", "evid-3", 3)
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(ev, nil)

	refs := []purposemodel.ElementRef{{Name: "email", Mandatory: true}}
	resolved, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.Nil(t, svcErr)
	require.Len(t, resolved, 1)
	require.Equal(t, "evid-3", resolved[0].ElementVersionID)
	require.Equal(t, "eid-1", resolved[0].ElementID)
	require.Equal(t, 3, resolved[0].VersionNum)
	require.True(t, resolved[0].Mandatory)
}

func TestValidateAndResolveElements_Success_SpecificVersion(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	latestEV := makeEV("eid-1", "evid-latest", 3)
	specificEV := makeEV("eid-1", "evid-v1", 1)
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(latestEV, nil)
	v := 1
	es.On("GetVersion", mock.Anything, "eid-1", 1, svcOrgID).Return(specificEV, nil)

	refs := []purposemodel.ElementRef{{Name: "email", Version: &v}}
	resolved, svcErr := svc.validateAndResolveElements(context.Background(), refs, svcOrgID)
	require.Nil(t, svcErr)
	require.Equal(t, "evid-v1", resolved[0].ElementVersionID)
	require.Equal(t, 1, resolved[0].VersionNum)
}

// =============================================================================
// CreatePurpose
// =============================================================================

func TestCreatePurpose_ValidationFailure(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	input := purposemodel.CreatePurposeInput{Name: ""}
	out, svcErr := svc.CreatePurpose(context.Background(), input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Contains(t, svcErr.Description, "name is required")
}

func TestCreatePurpose_EmptyElements(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	// GroupID = "" (group-scoped path): check (name, groupID) then check org-level.
	ps.On("GetByNameAndGroupID", mock.Anything, "Marketing", "", svcOrgID).Return(nil, nil)
	ps.On("GetByNameAndGroupID", mock.Anything, "Marketing", svcOrgID, svcOrgID).Return(nil, nil)

	input := purposemodel.CreatePurposeInput{Name: "Marketing", Elements: nil}
	out, svcErr := svc.CreatePurpose(context.Background(), input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorInvalidPurposeElements.Code, svcErr.Code)
}

func TestCreatePurpose_ElementNotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	// GroupID = "" (group-scoped path): check (name, groupID) then check org-level.
	ps.On("GetByNameAndGroupID", mock.Anything, "Marketing", "", svcOrgID).Return(nil, nil)
	ps.On("GetByNameAndGroupID", mock.Anything, "Marketing", svcOrgID, svcOrgID).Return(nil, nil)
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(nil, nil)

	input := purposemodel.CreatePurposeInput{
		Name:     "Marketing",
		Elements: []purposemodel.ElementRef{{Name: "email"}},
	}
	out, svcErr := svc.CreatePurpose(context.Background(), input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Contains(t, svcErr.Description, "does not exist")
}

// TestCreatePurpose_OrgLevelBlockedByGroupScoped verifies that creating an org-level purpose
// is rejected when a group-scoped purpose with the same name already exists anywhere in the org.
func TestCreatePurpose_OrgLevelBlockedByGroupScoped(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	// Org-level path (GroupID = orgID): no same-name org-level exists, but a group-scoped does.
	ps.On("GetByNameAndGroupID", mock.Anything, "Marketing", svcOrgID, svcOrgID).Return(nil, nil)
	ps.On("ExistsByNameInOrg", mock.Anything, "Marketing", svcOrgID).Return(true, nil)

	input := purposemodel.CreatePurposeInput{Name: "Marketing", GroupID: svcOrgID}
	out, svcErr := svc.CreatePurpose(context.Background(), input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorPurposeNameExists.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, "Marketing")
}

// TestCreatePurpose_OrgLevelExistsByNameInOrgStoreError verifies the error path for ExistsByNameInOrg.
func TestCreatePurpose_OrgLevelExistsByNameInOrgStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetByNameAndGroupID", mock.Anything, "Marketing", svcOrgID, svcOrgID).Return(nil, nil)
	ps.On("ExistsByNameInOrg", mock.Anything, "Marketing", svcOrgID).Return(false, errors.New("db error"))

	input := purposemodel.CreatePurposeInput{Name: "Marketing", GroupID: svcOrgID}
	out, svcErr := svc.CreatePurpose(context.Background(), input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorCheckNameExistence.Code, svcErr.Code)
}

func TestCreatePurpose_NameAlreadyExists(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	existing := &purposemodel.PurposeVersion{ID: "existing-id", Name: "Marketing", GroupID: svcOrgID}
	ps.On("GetByNameAndGroupID", mock.Anything, "Marketing", svcOrgID, svcOrgID).Return(existing, nil)

	input := purposemodel.CreatePurposeInput{Name: "Marketing", GroupID: svcOrgID}
	out, svcErr := svc.CreatePurpose(context.Background(), input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorPurposeNameExists.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, "Marketing")
}

func TestCreatePurpose_NameCheckStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetByNameAndGroupID", mock.Anything, "Marketing", svcOrgID, svcOrgID).Return(nil, errors.New("db error"))

	input := purposemodel.CreatePurposeInput{Name: "Marketing", GroupID: svcOrgID}
	out, svcErr := svc.CreatePurpose(context.Background(), input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorCheckNameExistence.Code, svcErr.Code)
}

// Note: CreatePurpose success and transaction-failure paths require a real DB connection
// (ExecuteTransaction calls logger.Fatal on missing config). These are covered by integration tests.

// =============================================================================
// CreatePurposeVersion
// =============================================================================

func TestCreatePurposeVersion_DescriptionTooLong(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	input := purposemodel.CreatePurposeVersionInput{Description: pStr(strings.Repeat("x", 1025))}
	out, svcErr := svc.CreatePurposeVersion(context.Background(), svcPurposeID, input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Contains(t, svcErr.Description, "description must not exceed 1024 characters")
}

func TestCreatePurposeVersion_GetLatestStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetLatestVersion", mock.Anything, svcPurposeID, svcOrgID).Return(nil, errStore)

	input := purposemodel.CreatePurposeVersionInput{}
	out, svcErr := svc.CreatePurposeVersion(context.Background(), svcPurposeID, input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorRetrievePurpose.Code, svcErr.Code)
}

func TestCreatePurposeVersion_PurposeNotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetLatestVersion", mock.Anything, svcPurposeID, svcOrgID).Return(nil, nil)

	input := purposemodel.CreatePurposeVersionInput{}
	out, svcErr := svc.CreatePurposeVersion(context.Background(), svcPurposeID, input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorPurposeNotFound.Code, svcErr.Code)
}

func TestCreatePurposeVersion_ElementNotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	es := interfacesmock.NewConsentElementStore(t)
	svc := newSvc(t, ps, es)

	ps.On("GetLatestVersion", mock.Anything, svcPurposeID, svcOrgID).Return(makePV(svcPurposeID, "vid-1", "Marketing", 1), nil)
	es.On("GetByNameAndNamespace", mock.Anything, "email", "default", svcOrgID).Return(nil, nil)

	input := purposemodel.CreatePurposeVersionInput{
		Elements: []purposemodel.ElementRef{{Name: "email"}},
	}
	out, svcErr := svc.CreatePurposeVersion(context.Background(), svcPurposeID, input, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Contains(t, svcErr.Description, "does not exist")
}

// Note: CreatePurposeVersion success and transaction-failure paths are covered by integration tests.

// =============================================================================
// GetPurpose
// =============================================================================

func TestGetPurpose_StoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetLatestVersion", mock.Anything, svcPurposeID, svcOrgID).Return(nil, errStore)

	out, svcErr := svc.GetPurpose(context.Background(), svcPurposeID, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorRetrievePurpose.Code, svcErr.Code)
}

func TestSvcGetPurpose_NotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetLatestVersion", mock.Anything, svcPurposeID, svcOrgID).Return(nil, nil)

	out, svcErr := svc.GetPurpose(context.Background(), svcPurposeID, svcOrgID)
	require.Nil(t, out)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorPurposeNotFound.Code, svcErr.Code)
}

func TestSvcGetPurpose_Success(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	pv := makePV(svcPurposeID, "vid-2", "Marketing", 2)
	pv.DisplayName = pStr("Marketing Display")
	pv.Elements = []purposemodel.PurposeMappedElement{
		{ElementVersionID: "evid-1", ElementID: "eid-1", Name: "email", Namespace: "default", VersionNum: 1, Mandatory: true},
	}
	ps.On("GetLatestVersion", mock.Anything, svcPurposeID, svcOrgID).Return(pv, nil)

	out, svcErr := svc.GetPurpose(context.Background(), svcPurposeID, svcOrgID)
	require.Nil(t, svcErr)
	require.NotNil(t, out)
	require.Equal(t, svcPurposeID, out.ID)
	require.Equal(t, "vid-2", out.VersionID)
	require.Equal(t, "Marketing", out.Name)
	require.Equal(t, 2, out.VersionNum)
	require.Equal(t, "Marketing Display", *out.DisplayName)
	require.Len(t, out.Elements, 1)
	require.Equal(t, "email", out.Elements[0].Name)
	require.True(t, out.Elements[0].Mandatory)
}

// =============================================================================
// GetPurposeVersion
// =============================================================================

func TestGetPurposeVersion_StoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetVersion", mock.Anything, svcPurposeID, 1, svcOrgID).Return(nil, errStore)

	out, svcErr := svc.GetPurposeVersion(context.Background(), svcPurposeID, 1, svcOrgID)
	require.Nil(t, out)
	require.Equal(t, ErrorRetrievePurpose.Code, svcErr.Code)
}

func TestGetPurposeVersion_NotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetVersion", mock.Anything, svcPurposeID, 1, svcOrgID).Return(nil, nil)

	out, svcErr := svc.GetPurposeVersion(context.Background(), svcPurposeID, 1, svcOrgID)
	require.Nil(t, out)
	require.Equal(t, ErrorPurposeNotFound.Code, svcErr.Code)
}

func TestSvcGetPurposeVersion_Success(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	pv := makePV(svcPurposeID, "vid-1", "Marketing", 1)
	ps.On("GetVersion", mock.Anything, svcPurposeID, 1, svcOrgID).Return(pv, nil)

	out, svcErr := svc.GetPurposeVersion(context.Background(), svcPurposeID, 1, svcOrgID)
	require.Nil(t, svcErr)
	require.Equal(t, svcPurposeID, out.ID)
	require.Equal(t, 1, out.VersionNum)
}

// =============================================================================
// GetPurposeVersions
// =============================================================================

func TestGetPurposeVersions_ExistsStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(false, errStore)

	out, svcErr := svc.GetPurposeVersions(context.Background(), svcPurposeID, svcOrgID)
	require.Nil(t, out)
	require.Equal(t, ErrorRetrievePurpose.Code, svcErr.Code)
}

func TestGetPurposeVersions_NotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(false, nil)

	out, svcErr := svc.GetPurposeVersions(context.Background(), svcPurposeID, svcOrgID)
	require.Nil(t, out)
	require.Equal(t, ErrorPurposeNotFound.Code, svcErr.Code)
}

func TestGetPurposeVersions_ListStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(true, nil)
	ps.On("ListVersions", mock.Anything, svcPurposeID, svcOrgID).Return(nil, errStore)

	out, svcErr := svc.GetPurposeVersions(context.Background(), svcPurposeID, svcOrgID)
	require.Nil(t, out)
	require.Equal(t, ErrorListPurposes.Code, svcErr.Code)
}

func TestGetPurposeVersions_Success(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(true, nil)
	ps.On("ListVersions", mock.Anything, svcPurposeID, svcOrgID).Return([]purposemodel.PurposeVersion{
		*makePV(svcPurposeID, "vid-1", "Marketing", 1),
		*makePV(svcPurposeID, "vid-2", "Marketing", 2),
	}, nil)

	out, svcErr := svc.GetPurposeVersions(context.Background(), svcPurposeID, svcOrgID)
	require.Nil(t, svcErr)
	require.Equal(t, svcPurposeID, out.PurposeID)
	require.Equal(t, "Marketing", out.Name)
	require.Equal(t, svcOrgID, out.GroupID)
	require.Len(t, out.Versions, 2)
	require.Equal(t, 1, out.Versions[0].VersionNum)
	require.Equal(t, 2, out.Versions[1].VersionNum)
}

func TestGetPurposeVersions_EmptyVersions(t *testing.T) {
	// Exists but no versions stored — edge case.
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(true, nil)
	ps.On("ListVersions", mock.Anything, svcPurposeID, svcOrgID).Return([]purposemodel.PurposeVersion{}, nil)

	out, svcErr := svc.GetPurposeVersions(context.Background(), svcPurposeID, svcOrgID)
	require.Nil(t, svcErr)
	require.Equal(t, svcPurposeID, out.PurposeID)
	require.Empty(t, out.Name)
	require.Empty(t, out.Versions)
}

// =============================================================================
// ListPurposes
// =============================================================================

func TestListPurposes_StoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("List", mock.Anything, svcOrgID, mock.Anything).Return(nil, 0, errStore)

	out, svcErr := svc.ListPurposes(context.Background(), svcOrgID, purposemodel.PurposeListFilter{})
	require.Nil(t, out)
	require.Equal(t, ErrorListPurposes.Code, svcErr.Code)
}

func TestSvcListPurposes_Success(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	versions := []purposemodel.PurposeVersion{
		*makePV("p-1", "vid-1", "Marketing", 1),
		*makePV("p-2", "vid-2", "Analytics", 1),
	}
	filters := purposemodel.PurposeListFilter{Limit: 10, Offset: 0}
	ps.On("List", mock.Anything, svcOrgID, filters).Return(versions, 2, nil)

	out, svcErr := svc.ListPurposes(context.Background(), svcOrgID, filters)
	require.Nil(t, svcErr)
	require.Len(t, out.Data, 2)
	require.Equal(t, 2, out.Total)
	require.Equal(t, 2, out.Count)
	require.Equal(t, 10, out.Limit)
	require.Equal(t, 0, out.Offset)
	require.Equal(t, "Marketing", out.Data[0].Name)
	require.Equal(t, "Analytics", out.Data[1].Name)
}

func TestListPurposes_EmptyResult(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	filters := purposemodel.PurposeListFilter{Limit: 100}
	ps.On("List", mock.Anything, svcOrgID, filters).Return([]purposemodel.PurposeVersion{}, 0, nil)

	out, svcErr := svc.ListPurposes(context.Background(), svcOrgID, filters)
	require.Nil(t, svcErr)
	require.Empty(t, out.Data)
	require.Equal(t, 0, out.Total)
	require.Equal(t, 0, out.Count)
}

// =============================================================================
// DeletePurposeVersion
// =============================================================================

func TestDeletePurposeVersion_GetVersionStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetVersion", mock.Anything, svcPurposeID, 1, svcOrgID).Return(nil, errStore)

	svcErr := svc.DeletePurposeVersion(context.Background(), svcPurposeID, 1, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorRetrievePurpose.Code, svcErr.Code)
}

func TestDeletePurposeVersion_NotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("GetVersion", mock.Anything, svcPurposeID, 1, svcOrgID).Return(nil, nil)

	svcErr := svc.DeletePurposeVersion(context.Background(), svcPurposeID, 1, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorPurposeNotFound.Code, svcErr.Code)
}

func TestDeletePurposeVersion_UsageCheckError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	pv := makePV(svcPurposeID, "vid-1", "Marketing", 1)
	ps.On("GetVersion", mock.Anything, svcPurposeID, 1, svcOrgID).Return(pv, nil)
	ps.On("IsVersionUsedInConsents", mock.Anything, "vid-1", svcOrgID).Return(false, errStore)

	svcErr := svc.DeletePurposeVersion(context.Background(), svcPurposeID, 1, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorCheckPurposeUsage.Code, svcErr.Code)
}

func TestDeletePurposeVersion_InUse(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	pv := makePV(svcPurposeID, "vid-1", "Marketing", 1)
	ps.On("GetVersion", mock.Anything, svcPurposeID, 1, svcOrgID).Return(pv, nil)
	ps.On("IsVersionUsedInConsents", mock.Anything, "vid-1", svcOrgID).Return(true, nil)

	svcErr := svc.DeletePurposeVersion(context.Background(), svcPurposeID, 1, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorPurposeVersionInUse.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, "v1")
}

func TestDeletePurposeVersion_ListVersionsError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	pv := makePV(svcPurposeID, "vid-1", "Marketing", 1)
	ps.On("GetVersion", mock.Anything, svcPurposeID, 1, svcOrgID).Return(pv, nil)
	ps.On("IsVersionUsedInConsents", mock.Anything, "vid-1", svcOrgID).Return(false, nil)
	ps.On("ListVersions", mock.Anything, svcPurposeID, svcOrgID).Return(nil, errStore)

	svcErr := svc.DeletePurposeVersion(context.Background(), svcPurposeID, 1, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorRetrievePurpose.Code, svcErr.Code)
}

// Note: DeletePurposeVersion transaction paths (single and last version) are covered by integration tests.

// =============================================================================
// DeletePurpose
// =============================================================================

func TestDeletePurpose_ExistsStoreError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(false, errStore)

	svcErr := svc.DeletePurpose(context.Background(), svcPurposeID, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorRetrievePurpose.Code, svcErr.Code)
}

func TestSvcDeletePurpose_NotFound(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(false, nil)

	svcErr := svc.DeletePurpose(context.Background(), svcPurposeID, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorPurposeNotFound.Code, svcErr.Code)
}

func TestDeletePurpose_ListVersionsError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(true, nil)
	ps.On("ListVersions", mock.Anything, svcPurposeID, svcOrgID).Return(nil, errStore)

	svcErr := svc.DeletePurpose(context.Background(), svcPurposeID, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorRetrievePurpose.Code, svcErr.Code)
}

func TestDeletePurpose_VersionUsageCheckError(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(true, nil)
	ps.On("ListVersions", mock.Anything, svcPurposeID, svcOrgID).Return([]purposemodel.PurposeVersion{
		*makePV(svcPurposeID, "vid-1", "Marketing", 1),
	}, nil)
	ps.On("IsVersionUsedInConsents", mock.Anything, "vid-1", svcOrgID).Return(false, errStore)

	svcErr := svc.DeletePurpose(context.Background(), svcPurposeID, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorCheckPurposeUsage.Code, svcErr.Code)
}

func TestDeletePurpose_VersionInUse(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)

	ps.On("PurposeExists", mock.Anything, svcPurposeID, svcOrgID).Return(true, nil)
	ps.On("ListVersions", mock.Anything, svcPurposeID, svcOrgID).Return([]purposemodel.PurposeVersion{
		*makePV(svcPurposeID, "vid-1", "Marketing", 1),
		*makePV(svcPurposeID, "vid-2", "Marketing", 2),
	}, nil)
	// First version is free, second is in use.
	ps.On("IsVersionUsedInConsents", mock.Anything, "vid-1", svcOrgID).Return(false, nil)
	ps.On("IsVersionUsedInConsents", mock.Anything, "vid-2", svcOrgID).Return(true, nil)

	svcErr := svc.DeletePurpose(context.Background(), svcPurposeID, svcOrgID)
	require.NotNil(t, svcErr)
	require.Equal(t, ErrorPurposeVersionInUse.Code, svcErr.Code)
	require.Contains(t, svcErr.Description, "v2")
}

// Note: DeletePurpose transaction path is covered by integration tests.

// =============================================================================
// isMySQLDuplicateKeyError
// =============================================================================

func TestIsMySQLDuplicateKeyError(t *testing.T) {
	// nil is never a MySQL error
	require.False(t, isMySQLDuplicateKeyError(nil))

	// plain Go error — not a MySQL error at all
	require.False(t, isMySQLDuplicateKeyError(errors.New("something went wrong")))

	// MySQL error 1062 → duplicate key
	require.True(t, isMySQLDuplicateKeyError(&mysql.MySQLError{Number: 1062}))

	// MySQL error with a different number → not a duplicate key
	require.False(t, isMySQLDuplicateKeyError(&mysql.MySQLError{Number: 1054}))
}

// =============================================================================
// buildCreateVersionTx
// =============================================================================

func TestBuildCreateVersionTx_QueryCount(t *testing.T) {
	ps := interfacesmock.NewConsentPurposeStore(t)
	svc := newSvc(t, ps, nil)
	pv := makePV(svcPurposeID, "vid-1", "Marketing", 1)

	// No elements → one query: CreateVersion only.
	queries := svc.buildCreateVersionTx(pv, nil)
	require.Len(t, queries, 1)

	// Two elements → three queries: CreateVersion + two LinkElementVersion calls.
	elems := []purposemodel.PurposeMappedElement{
		{ElementVersionID: "ev-1"},
		{ElementVersionID: "ev-2"},
	}
	queries = svc.buildCreateVersionTx(pv, elems)
	require.Len(t, queries, 3)
}
