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

package consentelement

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wso2/openfgc/consent-server/internal/consentelement/model"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
	interfacesmock "github.com/wso2/openfgc/consent-server/tests/mocks/stores/interfacesmock"
)

// --- CreateElementsInBatch ---

func TestService_CreateElementsInBatch_EmptyRequest(t *testing.T) {
	service := NewConsentElementService(&stores.StoreRegistry{})
	resp, err := service.CreateElementsInBatch(context.Background(), []model.CreateElementInput{}, testOrgID)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, "CE-1002", err.Code)
}

func TestService_CreateElementsInBatch_ValidationError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	// Missing name — validation fails before any store call
	inputs := []model.CreateElementInput{{Type: "basic"}}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err) // batch never returns a top-level error
	require.NotNil(t, resp)
	require.Len(t, resp.Results, 1)
	require.Equal(t, "FAILED", resp.Results[0].Status)
	require.NotNil(t, resp.Results[0].Error)
}

func TestService_CreateElementsInBatch_NameAlreadyExists(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	existing := &model.ElementVersion{ID: "old-id", Name: "email", Namespace: "default"}
	mockStore.On("GetByNameAndNamespace", mock.Anything, "email", "default", testOrgID).Return(existing, nil)

	inputs := []model.CreateElementInput{{Name: "email", Type: "basic"}}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Len(t, resp.Results, 1)
	require.Equal(t, "FAILED", resp.Results[0].Status)
}

func TestService_CreateElementsInBatch_InvalidType(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	inputs := []model.CreateElementInput{{Name: "email", Type: "invalid-type"}}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Len(t, resp.Results, 1)
	require.Equal(t, "FAILED", resp.Results[0].Status)
}

func TestService_CreateElementsInBatch_MissingSchemaForJSON(t *testing.T) {
	// json type with nil schema → validator rejects it at service level.
	// Array/object format checks are the handler's responsibility.
	service := NewConsentElementService(&stores.StoreRegistry{})

	inputs := []model.CreateElementInput{{Name: "email", Type: "json"}} // Schema is nil
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Len(t, resp.Results, 1)
	require.Equal(t, "FAILED", resp.Results[0].Status)
}

func TestService_CreateElementsInBatch_SchemaRequiredForJSONAndXML(t *testing.T) {
	// json and xml without schema → FAILED
	service := NewConsentElementService(&stores.StoreRegistry{})

	inputs := []model.CreateElementInput{
		{Name: "doc", Type: "json"},  // missing schema → FAILED
		{Name: "feed", Type: "xml"},  // missing schema → FAILED
	}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Len(t, resp.Results, 2)
	require.Equal(t, "FAILED", resp.Results[0].Status)
	require.Equal(t, "FAILED", resp.Results[1].Status)
}

func TestService_CreateElementsInBatch_ContinuesOnValidationFailure(t *testing.T) {
	// Verify that a validation error on one item does not abort processing the rest.
	service := NewConsentElementService(&stores.StoreRegistry{})

	inputs := []model.CreateElementInput{
		{Type: "basic"},          // missing name → FAILED
		{Name: "x", Type: "???"}, // invalid type → FAILED
	}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Len(t, resp.Results, 2)
	require.Equal(t, "FAILED", resp.Results[0].Status)
	require.Equal(t, "FAILED", resp.Results[1].Status)
}

// --- GetElement ---

func TestService_GetElement_Success(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	expected := &model.ElementVersion{ID: testElementID, Name: "email", Type: "basic", VersionNum: 1}
	mockStore.On("GetLatestVersion", mock.Anything, testElementID, testOrgID).Return(expected, nil)

	v, err := service.GetElement(context.Background(), testElementID, testOrgID)
	require.Nil(t, err)
	require.Equal(t, testElementID, v.ID)
}

func TestService_GetElement_NotFound(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetLatestVersion", mock.Anything, testElementID, testOrgID).Return(nil, nil)

	v, err := service.GetElement(context.Background(), testElementID, testOrgID)
	require.Error(t, err)
	require.Nil(t, v)
	require.Equal(t, "CE-1016", err.Code)
}

func TestService_GetElement_DBError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetLatestVersion", mock.Anything, testElementID, testOrgID).Return(nil, errors.New("db error"))

	v, err := service.GetElement(context.Background(), testElementID, testOrgID)
	require.Error(t, err)
	require.Nil(t, v)
}

// --- GetElementVersion ---

func TestService_GetElementVersion_Success(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	expected := &model.ElementVersion{ID: testElementID, VersionNum: 2}
	mockStore.On("GetVersion", mock.Anything, testElementID, 2, testOrgID).Return(expected, nil)

	v, err := service.GetElementVersion(context.Background(), testElementID, 2, testOrgID)
	require.Nil(t, err)
	require.Equal(t, 2, v.VersionNum)
}

func TestService_GetElementVersion_NotFound(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetVersion", mock.Anything, testElementID, 99, testOrgID).Return(nil, nil)

	v, err := service.GetElementVersion(context.Background(), testElementID, 99, testOrgID)
	require.Error(t, err)
	require.Nil(t, v)
	require.Equal(t, "CE-1016", err.Code)
}

// --- ListElementVersions ---

func TestService_ListElementVersions_Success(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("ElementExists", mock.Anything, testElementID, testOrgID).Return(true, nil)
	mockStore.On("ListVersions", mock.Anything, testElementID, testOrgID).Return([]model.ElementVersion{
		{ID: testElementID, VersionNum: 1},
		{ID: testElementID, VersionNum: 2},
	}, nil)

	resp, err := service.ListElementVersions(context.Background(), testElementID, testOrgID)
	require.Nil(t, err)
	require.Equal(t, testElementID, resp.ElementID)
	require.Len(t, resp.Versions, 2)
}

func TestService_ListElementVersions_ElementNotFound(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("ElementExists", mock.Anything, testElementID, testOrgID).Return(false, nil)

	resp, err := service.ListElementVersions(context.Background(), testElementID, testOrgID)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, "CE-1016", err.Code)
}

// --- ListElements ---

func TestService_ListElements_Success(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	filters := model.ElementListFilter{Limit: 10, Offset: 0}
	mockStore.On("List", mock.Anything, testOrgID, filters).Return([]model.ElementVersion{
		{ID: "e1"}, {ID: "e2"},
	}, 2, nil)

	resp, err := service.ListElements(context.Background(), testOrgID, filters)
	require.Nil(t, err)
	require.Equal(t, 2, resp.Total)
	require.Len(t, resp.Data, 2)
}

func TestService_ListElements_DefaultLimit(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	// Limit 0 → defaults to 100
	mockStore.On("List", mock.Anything, testOrgID, model.ElementListFilter{Limit: 100}).Return([]model.ElementVersion{}, 0, nil)

	resp, err := service.ListElements(context.Background(), testOrgID, model.ElementListFilter{Limit: 0})
	require.Nil(t, err)
	require.Equal(t, 0, resp.Total)
}

// --- CreateElementVersion ---

func TestService_CreateElementVersion_ElementNotFound(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetLatestVersion", mock.Anything, testElementID, testOrgID).Return(nil, nil)

	v, err := service.CreateElementVersion(context.Background(), testElementID, model.CreateElementVersionInput{}, testOrgID)
	require.Error(t, err)
	require.Nil(t, v)
	require.Equal(t, "CE-1016", err.Code)
}

// --- DeleteElementVersion ---

func TestService_DeleteElementVersion_NotFound(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetVersion", mock.Anything, testElementID, 1, testOrgID).Return(nil, nil)

	err := service.DeleteElementVersion(context.Background(), testElementID, 1, testOrgID)
	require.Error(t, err)
	require.Equal(t, "CE-1016", err.Code)
}

func TestService_DeleteElementVersion_ReferencedByPurpose(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	v := &model.ElementVersion{ID: testElementID, VersionID: "v-uuid-1"}
	mockStore.On("GetVersion", mock.Anything, testElementID, 1, testOrgID).Return(v, nil)
	mockStore.On("IsVersionReferencedByPurpose", mock.Anything, "v-uuid-1", testOrgID).Return(true, nil)

	err := service.DeleteElementVersion(context.Background(), testElementID, 1, testOrgID)
	require.Error(t, err)
	require.Equal(t, "CE-4090", err.Code)
}

// --- CreateElementsInBatch / validateCreateInput — additional validation paths ---

func TestService_CreateElementsInBatch_NameTooLong(t *testing.T) {
	service := NewConsentElementService(&stores.StoreRegistry{})

	inputs := []model.CreateElementInput{{Name: string(make([]byte, 256)), Type: "basic"}}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Equal(t, "FAILED", resp.Results[0].Status)
}

func TestService_CreateElementsInBatch_DescriptionTooLong(t *testing.T) {
	service := NewConsentElementService(&stores.StoreRegistry{})

	desc := string(make([]byte, 1025))
	inputs := []model.CreateElementInput{{Name: "email", Type: "basic", Description: &desc}}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Equal(t, "FAILED", resp.Results[0].Status)
}

func TestService_CreateElementsInBatch_EmptyType(t *testing.T) {
	service := NewConsentElementService(&stores.StoreRegistry{})

	inputs := []model.CreateElementInput{{Name: "email", Type: ""}}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Equal(t, "FAILED", resp.Results[0].Status)
}

func TestService_CreateElementsInBatch_NameCheckDBError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetByNameAndNamespace", mock.Anything, "email", "default", testOrgID).
		Return(nil, errors.New("db error"))

	inputs := []model.CreateElementInput{{Name: "email", Type: "basic"}}
	resp, err := service.CreateElementsInBatch(context.Background(), inputs, testOrgID)

	require.Nil(t, err)
	require.Equal(t, "FAILED", resp.Results[0].Status)
}

// --- GetElementVersion — additional paths ---

func TestService_GetElementVersion_DBError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetVersion", mock.Anything, testElementID, 1, testOrgID).Return(nil, errors.New("db error"))

	v, err := service.GetElementVersion(context.Background(), testElementID, 1, testOrgID)
	require.Error(t, err)
	require.Nil(t, v)
}

// --- ListElementVersions — additional paths ---

func TestService_ListElementVersions_ElementExistsError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("ElementExists", mock.Anything, testElementID, testOrgID).Return(false, errors.New("db error"))

	resp, err := service.ListElementVersions(context.Background(), testElementID, testOrgID)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestService_ListElementVersions_ListVersionsError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("ElementExists", mock.Anything, testElementID, testOrgID).Return(true, nil)
	mockStore.On("ListVersions", mock.Anything, testElementID, testOrgID).Return(nil, errors.New("db error"))

	resp, err := service.ListElementVersions(context.Background(), testElementID, testOrgID)
	require.Error(t, err)
	require.Nil(t, resp)
}

// --- ListElements — additional paths ---

func TestService_ListElements_DBError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("List", mock.Anything, testOrgID, model.ElementListFilter{Limit: 10}).
		Return(nil, 0, errors.New("db error"))

	resp, err := service.ListElements(context.Background(), testOrgID, model.ElementListFilter{Limit: 10})
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestService_ListElements_NegativeOffset(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	// Negative offset must be clamped to 0 before the store call.
	mockStore.On("List", mock.Anything, testOrgID, model.ElementListFilter{Limit: 10, Offset: 0}).
		Return([]model.ElementVersion{}, 0, nil)

	resp, err := service.ListElements(context.Background(), testOrgID, model.ElementListFilter{Limit: 10, Offset: -5})
	require.Nil(t, err)
	require.NotNil(t, resp)
}

func TestService_CreateElementVersion_DescriptionTooLong(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	latest := &model.ElementVersion{ID: testElementID, Name: "x", Type: "basic", VersionNum: 1}
	mockStore.On("GetLatestVersion", mock.Anything, testElementID, testOrgID).Return(latest, nil)

	desc := string(make([]byte, 1025))
	v, err := service.CreateElementVersion(context.Background(), testElementID, model.CreateElementVersionInput{Description: &desc}, testOrgID)
	require.Error(t, err)
	require.Nil(t, v)
	require.Equal(t, "CE-1008", err.Code)
}

func TestService_CreateElementVersion_InvalidSchemaForJSON(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	// json type requires a non-nil schema — nil input.Schema triggers CE-5010.
	latest := &model.ElementVersion{ID: testElementID, Name: "doc", Type: "json", VersionNum: 1}
	mockStore.On("GetLatestVersion", mock.Anything, testElementID, testOrgID).Return(latest, nil)

	v, err := service.CreateElementVersion(context.Background(), testElementID, model.CreateElementVersionInput{}, testOrgID)
	require.Error(t, err)
	require.Nil(t, v)
	require.Equal(t, "CE-5010", err.Code)
}

// --- CreateElementVersion — additional paths ---

func TestService_CreateElementVersion_DBError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetLatestVersion", mock.Anything, testElementID, testOrgID).Return(nil, errors.New("db error"))

	v, err := service.CreateElementVersion(context.Background(), testElementID, model.CreateElementVersionInput{}, testOrgID)
	require.Error(t, err)
	require.Nil(t, v)
}

// --- DeleteElementVersion — additional paths ---

func TestService_DeleteElementVersion_GetVersionError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	mockStore.On("GetVersion", mock.Anything, testElementID, 1, testOrgID).Return(nil, errors.New("db error"))

	err := service.DeleteElementVersion(context.Background(), testElementID, 1, testOrgID)
	require.Error(t, err)
}

func TestService_DeleteElementVersion_ReferenceCheckError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	v := &model.ElementVersion{ID: testElementID, VersionID: "v-uuid-1"}
	mockStore.On("GetVersion", mock.Anything, testElementID, 1, testOrgID).Return(v, nil)
	mockStore.On("IsVersionReferencedByPurpose", mock.Anything, "v-uuid-1", testOrgID).Return(false, errors.New("db error"))

	err := service.DeleteElementVersion(context.Background(), testElementID, 1, testOrgID)
	require.Error(t, err)
}

func TestService_DeleteElementVersion_ListVersionsError(t *testing.T) {
	mockStore := interfacesmock.NewConsentElementStore(t)
	service := NewConsentElementService(&stores.StoreRegistry{ConsentElement: mockStore})

	v := &model.ElementVersion{ID: testElementID, VersionID: "v-uuid-1"}
	mockStore.On("GetVersion", mock.Anything, testElementID, 1, testOrgID).Return(v, nil)
	mockStore.On("IsVersionReferencedByPurpose", mock.Anything, "v-uuid-1", testOrgID).Return(false, nil)
	mockStore.On("ListVersions", mock.Anything, testElementID, testOrgID).Return(nil, errors.New("db error"))

	err := service.DeleteElementVersion(context.Background(), testElementID, 1, testOrgID)
	require.Error(t, err)
}
