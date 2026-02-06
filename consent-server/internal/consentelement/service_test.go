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

	"github.com/wso2/openfgc/internal/consentelement/model"
	"github.com/wso2/openfgc/internal/system/stores"
	interfacesmock "github.com/wso2/openfgc/tests/mocks/stores/interfacesmock"
)

// TestService_CreateElementsInBatch_Success tests successful batch creation - skipped due to transaction complexity
// This functionality is covered by integration tests
func TestService_CreateElementsInBatch_Success(t *testing.T) {
	t.Skip("Transaction-based tests covered by integration tests")
}

// TestService_CreateElementsInBatch_EmptyRequest tests empty request array
func TestService_CreateElementsInBatch_EmptyRequest(t *testing.T) {
	registry := &stores.StoreRegistry{}
	service := newConsentElementService(registry)

	ctx := context.Background()
	elements, err := service.CreateElementsInBatch(ctx, []model.ConsentElementCreateRequest{}, testOrgID)

	require.Error(t, err)
	require.Nil(t, elements)
	require.Equal(t, "CE-1002", err.Code)
}

// TestService_CreateElementsInBatch_DuplicateNameInBatch tests duplicate names in batch
func TestService_CreateElementsInBatch_DuplicateNameInBatch(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
	}
	service := newConsentElementService(registry)

	requests := []model.ConsentElementCreateRequest{
		{Name: "duplicate", Type: "basic"},
		{Name: "duplicate", Type: "basic"},
	}

	// Mock CheckNameExists for first occurrence - service checks DB before detecting duplicate in batch
	mockElementStore.On("CheckNameExists", mock.Anything, "duplicate", testOrgID).
		Return(false, nil).Maybe()

	ctx := context.Background()
	elements, err := service.CreateElementsInBatch(ctx, requests, testOrgID)

	require.Error(t, err)
	require.Nil(t, elements)
	require.Contains(t, err.Description, "duplicate element name")
}

// TestService_CreateElementsInBatch_NameAlreadyExists tests existing name in database
func TestService_CreateElementsInBatch_NameAlreadyExists(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
	}
	service := newConsentElementService(registry)

	requests := []model.ConsentElementCreateRequest{
		{Name: "existing", Type: "basic"},
	}

	// Mock name existence check - returns true
	mockElementStore.On("CheckNameExists", mock.Anything, "existing", testOrgID).
		Return(true, nil)

	ctx := context.Background()
	elements, err := service.CreateElementsInBatch(ctx, requests, testOrgID)

	require.Error(t, err)
	require.Nil(t, elements)
	require.Equal(t, "CE-1011", err.Code)
	require.Contains(t, err.Description, "already exists")
}

// TestService_GetElement_Success tests successful element retrieval
func TestService_GetElement_Success(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
	}
	service := newConsentElementService(registry)

	expectedElement := &model.ConsentElement{
		ID:         testElementID,
		Name:       "test_element",
		Type:       "basic",
		OrgID:      testOrgID,
		Properties: make(map[string]string),
	}

	mockElementStore.On("GetByID", mock.Anything, testElementID, testOrgID).
		Return(expectedElement, nil)
	mockElementStore.On("GetPropertiesByElementID", mock.Anything, testElementID, testOrgID).
		Return([]model.ConsentElementProperty{}, nil)

	ctx := context.Background()
	element, err := service.GetElement(ctx, testElementID, testOrgID)

	if err != nil {
		t.Logf("Error is not nil: %v (type: %T)", err, err)
	}
	require.Nil(t, err)
	require.NotNil(t, element)
	require.Equal(t, testElementID, element.ID)
	require.Equal(t, "test_element", element.Name)
	mockElementStore.AssertExpectations(t)
}

// TestService_GetElement_NotFound tests element not found
func TestService_GetElement_NotFound(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
	}
	service := newConsentElementService(registry)

	mockElementStore.On("GetByID", mock.Anything, testElementID, testOrgID).
		Return(nil, nil)

	ctx := context.Background()
	element, err := service.GetElement(ctx, testElementID, testOrgID)

	require.Error(t, err)
	require.Nil(t, element)
	require.Equal(t, "CE-1016", err.Code)
}

// TestService_ListElements_Success tests successful element listing
func TestService_ListElements_Success(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
	}
	service := newConsentElementService(registry)

	expectedElements := []model.ConsentElement{
		{ID: "elem-1", Name: "element_1", OrgID: testOrgID, Properties: make(map[string]string)},
		{ID: "elem-2", Name: "element_2", OrgID: testOrgID, Properties: make(map[string]string)},
	}

	mockElementStore.On("List", mock.Anything, testOrgID, 100, 0, "").
		Return(expectedElements, 2, nil)
	mockElementStore.On("GetPropertiesByElementID", mock.Anything, "elem-1", testOrgID).
		Return([]model.ConsentElementProperty{}, nil)
	mockElementStore.On("GetPropertiesByElementID", mock.Anything, "elem-2", testOrgID).
		Return([]model.ConsentElementProperty{}, nil)

	ctx := context.Background()
	elements, total, err := service.ListElements(ctx, testOrgID, 100, 0, "")

	require.Nil(t, err)
	require.Len(t, elements, 2)
	require.Equal(t, 2, total)
	mockElementStore.AssertExpectations(t)
}

// TestService_DeleteElement_Success tests successful element deletion - skipped due to transaction complexity
// This functionality is covered by integration tests
func TestService_DeleteElement_Success(t *testing.T) {
	t.Skip("Transaction-based tests covered by integration tests")
}

// TestService_DeleteElement_NotFound tests deleting non-existent element
func TestService_DeleteElement_NotFound(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
	}
	service := newConsentElementService(registry)

	mockElementStore.On("GetByID", mock.Anything, testElementID, testOrgID).
		Return(nil, nil)

	ctx := context.Background()
	err := service.DeleteElement(ctx, testElementID, testOrgID)

	require.Error(t, err)
	require.Equal(t, "CE-1016", err.Code)
}

// TestService_DeleteElement_UsedInPurposes tests deleting element used in purposes
func TestService_DeleteElement_UsedInPurposes(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	mockPurposeStore := interfacesmock.NewConsentPurposeStore(t)

	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
		ConsentPurpose: mockPurposeStore,
	}
	service := newConsentElementService(registry)

	// Element exists
	existingElement := &model.ConsentElement{
		ID:         testElementID,
		Properties: make(map[string]string),
	}
	mockElementStore.On("GetByID", mock.Anything, testElementID, testOrgID).
		Return(existingElement, nil)

	// Element is used in purposes
	mockPurposeStore.On("IsElementUsedInPurposes", mock.Anything, testElementID, testOrgID).
		Return(true, nil)

	ctx := context.Background()
	err := service.DeleteElement(ctx, testElementID, testOrgID)

	require.Error(t, err)
	require.Equal(t, "CE-5009", err.Code)
	require.Contains(t, err.Description, "used in one or more consent purposes")
	mockElementStore.AssertExpectations(t)
	mockPurposeStore.AssertExpectations(t)
}

// TestValidateElementNames_Success tests successful name validation
func TestValidateElementNames_Success(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
	}
	service := newConsentElementService(registry)

	ctx := context.Background()

	// Mock GetIDsByNames - returns map of name->ID for existing elements
	elementIDMap := map[string]string{
		"element_1": "id-1",
		"element_3": "id-3",
		// element_2 not included - doesn't exist
	}
	mockElementStore.On("GetIDsByNames", mock.Anything, []string{"element_1", "element_2", "element_3"}, testOrgID).
		Return(elementIDMap, nil)

	validNames, err := service.ValidateElementNames(ctx, testOrgID, []string{"element_1", "element_2", "element_3"})

	require.Nil(t, err)
	require.Len(t, validNames, 2)
	require.Contains(t, validNames, "element_1")
	require.Contains(t, validNames, "element_3")
	require.NotContains(t, validNames, "element_2")
	mockElementStore.AssertExpectations(t)
}

// TestValidateElementNames_EmptyInput tests empty input
func TestValidateElementNames_EmptyInput(t *testing.T) {
	registry := &stores.StoreRegistry{}
	service := newConsentElementService(registry)

	ctx := context.Background()
	validNames, err := service.ValidateElementNames(ctx, testOrgID, []string{})

	require.Error(t, err)
	require.Nil(t, validNames)
}

// TestValidateElementNames_DatabaseError tests database error during validation
func TestValidateElementNames_DatabaseError(t *testing.T) {
	mockElementStore := interfacesmock.NewConsentElementStore(t)
	registry := &stores.StoreRegistry{
		ConsentElement: mockElementStore,
	}
	service := newConsentElementService(registry)

	ctx := context.Background()

	mockElementStore.On("GetIDsByNames", mock.Anything, []string{"element_1"}, testOrgID).
		Return(nil, errors.New("database error"))

	validNames, err := service.ValidateElementNames(ctx, testOrgID, []string{"element_1"})

	require.Error(t, err)
	require.Nil(t, validNames)
	mockElementStore.AssertExpectations(t)
}
