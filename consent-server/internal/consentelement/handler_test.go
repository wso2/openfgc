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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wso2/openfgc/internal/consentelement/model"
	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
)

const (
	testOrgID     = "test-org-123"
	testElementID = "elem-123"
)

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// TestCreateElement_Success tests successful element creation
func TestCreateElement_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	requests := []model.ConsentElementCreateRequest{
		{
			Name:        "test_element",
			Type:        "basic",
			Description: "Test element",
			Properties:  map[string]string{"value": "test"},
		},
	}

	expectedElements := []model.ConsentElement{
		{
			ID:          testElementID,
			Name:        "test_element",
			Description: stringPtr("Test element"),
			Type:        "basic",
			OrgID:       testOrgID,
			Properties:  map[string]string{"value": "test"},
		},
	}

	mockService.On("CreateElementsInBatch", mock.Anything, requests, testOrgID).
		Return(expectedElements, nil)

	handler := newConsentElementHandler(mockService)
	body, _ := json.Marshal(requests)
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createElement(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Contains(t, rr.Header().Get(constants.HeaderContentType), "application/json")

	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Contains(t, response, "data")
	require.Contains(t, response, "message")

	data := response["data"].([]interface{})
	require.Len(t, data, 1)
}

// TestCreateElement_MissingOrgID tests missing org-id header
func TestCreateElement_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	handler := newConsentElementHandler(mockService)

	requests := []model.ConsentElementCreateRequest{{Name: "test", Type: "basic"}}
	body, _ := json.Marshal(requests)
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.createElement(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "CreateElementsInBatch")
}

// TestCreateElement_InvalidJSON tests malformed JSON request
func TestCreateElement_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	handler := newConsentElementHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBufferString("{invalid json"))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createElement(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "CreateElementsInBatch")
}

// TestCreateElement_EmptyArray tests empty request array
func TestCreateElement_EmptyArray(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	handler := newConsentElementHandler(mockService)

	body, _ := json.Marshal([]model.ConsentElementCreateRequest{})
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createElement(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "CreateElementsInBatch")
}

// TestCreateElement_ServiceError tests service layer error handling
func TestCreateElement_ServiceError(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	requests := []model.ConsentElementCreateRequest{
		{Name: "test_element", Type: "basic"},
	}

	serviceErr := &serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1011",
		Message:     "Element name already exists",
		Description: "An element with the same name already exists",
	}

	mockService.On("CreateElementsInBatch", mock.Anything, requests, testOrgID).
		Return(nil, serviceErr)

	handler := newConsentElementHandler(mockService)
	body, _ := json.Marshal(requests)
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createElement(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)
}

// TestGetElement_Success tests successful element retrieval
func TestGetElement_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	expectedElement := &model.ConsentElement{
		ID:          testElementID,
		Name:        "test_element",
		Description: stringPtr("Test element"),
		Type:        "basic",
		OrgID:       testOrgID,
		Properties:  map[string]string{"value": "test"},
	}

	mockService.On("GetElement", mock.Anything, testElementID, testOrgID).
		Return(expectedElement, nil)

	handler := newConsentElementHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.getElement(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ConsentElementResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, testElementID, response.ID)
	require.Equal(t, "test_element", response.Name)
}

// TestGetElement_MissingOrgID tests missing org-id header
func TestGetElement_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	handler := newConsentElementHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	rr := httptest.NewRecorder()

	handler.getElement(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "GetElement")
}

// TestGetElement_NotFound tests element not found scenario
func TestGetElement_NotFound(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	serviceErr := &ErrorElementNotFound
	mockService.On("GetElement", mock.Anything, testElementID, testOrgID).
		Return(nil, serviceErr)

	handler := newConsentElementHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.getElement(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// TestListElements_Success tests successful element listing
func TestListElements_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	expectedElements := []model.ConsentElement{
		{
			ID:         "elem-1",
			Name:       "element_1",
			Type:       "basic",
			OrgID:      testOrgID,
			Properties: map[string]string{},
		},
		{
			ID:         "elem-2",
			Name:       "element_2",
			Type:       "json-payload",
			OrgID:      testOrgID,
			Properties: map[string]string{"validationSchema": "{}"},
		},
	}

	mockService.On("ListElements", mock.Anything, testOrgID, 100, 0, "").
		Return(expectedElements, 2, nil)

	handler := newConsentElementHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-elements", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listElements(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Contains(t, response, "data")
	require.Contains(t, response, "metadata")

	data := response["data"].([]interface{})
	require.Len(t, data, 2)

	metadata := response["metadata"].(map[string]interface{})
	require.Equal(t, float64(2), metadata["total"])
	require.Equal(t, float64(0), metadata["offset"])
	require.Equal(t, float64(100), metadata["limit"])
}

// TestListElements_MissingOrgID tests missing org-id header
func TestListElements_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	handler := newConsentElementHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements", nil)
	rr := httptest.NewRecorder()

	handler.listElements(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "ListElements")
}

// TestDeleteElement_Success tests successful element deletion
func TestDeleteElement_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	mockService.On("DeleteElement", mock.Anything, testElementID, testOrgID).
		Return(nil)

	handler := newConsentElementHandler(mockService)
	req := httptest.NewRequest(http.MethodDelete, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.deleteElement(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Empty(t, rr.Body.String())
}

// TestValidateElements_Success tests successful element name validation
func TestValidateElements_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	elementNames := []string{"element_1", "element_2", "element_3"}
	validNames := []string{"element_1", "element_3"}

	mockService.On("ValidateElementNames", mock.Anything, testOrgID, elementNames).
		Return(validNames, nil)

	handler := newConsentElementHandler(mockService)
	body, _ := json.Marshal(elementNames)
	req := httptest.NewRequest(http.MethodPost, "/consent-elements/validate", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.validateElements(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response []string
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, validNames, response)
}

// TestUpdateElement_Success tests successful element update
func TestUpdateElement_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	updateReq := model.ConsentElementUpdateRequest{
		Name:        "updated_element",
		Description: stringPtr("Updated description"),
		Type:        "basic",
		Properties:  map[string]string{"value": "updated"},
	}

	updatedElement := &model.ConsentElement{
		ID:          testElementID,
		Name:        "updated_element",
		Description: stringPtr("Updated description"),
		Type:        "basic",
		OrgID:       testOrgID,
		Properties:  map[string]string{"value": "updated"},
	}

	mockService.On("UpdateElement", mock.Anything, testElementID, updateReq, testOrgID).
		Return(updatedElement, nil)

	handler := newConsentElementHandler(mockService)
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/consent-elements/"+testElementID, bytes.NewBuffer(body))
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.updateElement(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ConsentElementResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "updated_element", response.Name)
}

// TestUpdateElement_MissingOrgID tests missing org-id header
func TestUpdateElement_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	handler := newConsentElementHandler(mockService)

	updateReq := model.ConsentElementUpdateRequest{Name: "test", Type: "basic"}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/consent-elements/"+testElementID, bytes.NewBuffer(body))
	req.SetPathValue("elementId", testElementID)
	rr := httptest.NewRecorder()

	handler.updateElement(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "UpdateElement")
}

// TestDeleteElement_MissingOrgID tests missing org-id header
func TestDeleteElement_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	handler := newConsentElementHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	rr := httptest.NewRecorder()

	handler.deleteElement(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "DeleteElement")
}

// TestDeleteElement_NotFound tests element not found scenario
func TestDeleteElement_NotFound(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	serviceErr := &ErrorElementNotFound
	mockService.On("DeleteElement", mock.Anything, testElementID, testOrgID).
		Return(serviceErr)

	handler := newConsentElementHandler(mockService)
	req := httptest.NewRequest(http.MethodDelete, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.deleteElement(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// TestListElements_WithPagination tests pagination parameters
func TestListElements_WithPagination(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	mockService.On("ListElements", mock.Anything, testOrgID, 10, 5, "").
		Return([]model.ConsentElement{}, 0, nil)

	handler := newConsentElementHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-elements?limit=10&offset=5", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listElements(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

// TestListElements_WithNameFilter tests name filter parameter
func TestListElements_WithNameFilter(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	mockService.On("ListElements", mock.Anything, testOrgID, 100, 0, "test").
		Return([]model.ConsentElement{}, 0, nil)

	handler := newConsentElementHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-elements?name=test", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listElements(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}
