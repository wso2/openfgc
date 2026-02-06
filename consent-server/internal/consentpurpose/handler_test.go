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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wso2/openfgc/internal/consentpurpose/model"
	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
)

const (
	testOrgID     = "test-org-123"
	testClientID  = "test-client-456"
	testPurposeID = "purpose-123"
)

func stringPtr(s string) *string {
	return &s
}

// TestCreatePurpose_Success tests successful purpose creation
func TestCreatePurpose_Success(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	request := model.CreateRequest{
		Name:        "Test Purpose",
		Description: "Test Description",
		Elements: []model.ElementInput{
			{ElementName: "element1", IsMandatory: true},
		},
	}

	expectedPurpose := &model.ConsentPurpose{
		ID:          testPurposeID,
		Name:        "Test Purpose",
		Description: stringPtr("Test Description"),
		ClientID:    testClientID,
		Elements: []model.PurposeElement{
			{ElementID: "elem-1", ElementName: "element1", IsMandatory: true},
		},
		CreatedTime: 1234567890,
		UpdatedTime: 1234567890,
		OrgID:       testOrgID,
	}

	mockService.On("CreatePurpose", mock.Anything, request, testOrgID, testClientID).
		Return(expectedPurpose, nil)

	handler := newConsentPurposeHandler(mockService)
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Contains(t, rr.Header().Get(constants.HeaderContentType), "application/json")

	var response model.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, testPurposeID, response.ID)
	require.Equal(t, "Test Purpose", response.Name)
}

// TestCreatePurpose_MissingOrgID tests missing org-id header
func TestCreatePurpose_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	request := model.CreateRequest{Name: "Test", Elements: []model.ElementInput{{ElementName: "elem"}}}
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "CreatePurpose")
}

// TestCreatePurpose_MissingClientID tests missing client-id header
func TestCreatePurpose_MissingClientID(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	request := model.CreateRequest{Name: "Test", Elements: []model.ElementInput{{ElementName: "elem"}}}
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "CreatePurpose")
}

// TestCreatePurpose_InvalidJSON tests malformed JSON request
func TestCreatePurpose_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBufferString("{invalid json"))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "CreatePurpose")
}

// TestCreatePurpose_ServiceError tests service layer error handling
func TestCreatePurpose_ServiceError(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	request := model.CreateRequest{
		Name:     "Test Purpose",
		Elements: []model.ElementInput{{ElementName: "elem"}},
	}

	serviceErr := &serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4001",
		Message:     "Validation failed",
		Description: "Invalid request",
	}

	mockService.On("CreatePurpose", mock.Anything, request, testOrgID, testClientID).
		Return(nil, serviceErr)

	handler := newConsentPurposeHandler(mockService)
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestGetPurpose_Success tests successful purpose retrieval
func TestGetPurpose_Success(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	expectedPurpose := &model.ConsentPurpose{
		ID:          testPurposeID,
		Name:        "Test Purpose",
		Description: stringPtr("Test Description"),
		ClientID:    testClientID,
		Elements:    []model.PurposeElement{},
		CreatedTime: 1234567890,
		UpdatedTime: 1234567890,
		OrgID:       testOrgID,
	}

	mockService.On("GetPurpose", mock.Anything, testPurposeID, testOrgID).
		Return(expectedPurpose, nil)

	handler := newConsentPurposeHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.getPurpose(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Header().Get(constants.HeaderContentType), "application/json")

	var response model.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, testPurposeID, response.ID)
}

// TestGetPurpose_MissingOrgID tests missing org-id header
func TestGetPurpose_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID, nil)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.getPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "GetPurpose")
}

// TestGetPurpose_NotFound tests purpose not found scenario
func TestGetPurpose_NotFound(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	serviceErr := &ErrorPurposeNotFound

	mockService.On("GetPurpose", mock.Anything, testPurposeID, testOrgID).
		Return(nil, serviceErr)

	handler := newConsentPurposeHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.getPurpose(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// TestListPurposes_Success tests successful listing with no filters
func TestListPurposes_Success(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	purposes := []model.ConsentPurpose{
		{
			ID:       "purpose-1",
			Name:     "Purpose 1",
			ClientID: testClientID,
			OrgID:    testOrgID,
		},
		{
			ID:       "purpose-2",
			Name:     "Purpose 2",
			ClientID: testClientID,
			OrgID:    testOrgID,
		},
	}

	mockService.On("ListPurposes", mock.Anything, testOrgID, "", []string(nil), []string(nil), 0, 100).
		Return(purposes, 2, nil)

	handler := newConsentPurposeHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ListResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Len(t, response.Data, 2)
	require.Equal(t, 2, response.Metadata.Total)
	require.Equal(t, 0, response.Metadata.Offset)
	require.Equal(t, 100, response.Metadata.Limit)
}

// TestListPurposes_WithFilters tests listing with filters
func TestListPurposes_WithFilters(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	purposes := []model.ConsentPurpose{
		{ID: "purpose-1", Name: "Test Purpose", ClientID: testClientID, OrgID: testOrgID},
	}

	mockService.On("ListPurposes", mock.Anything, testOrgID, "Test Purpose",
		[]string{"client-1", "client-2"}, []string{"elem1"}, 10, 20).
		Return(purposes, 1, nil)

	handler := newConsentPurposeHandler(mockService)
	req := httptest.NewRequest(http.MethodGet,
		"/consent-purposes?name=Test%20Purpose&clientIds=client-1,client-2&elementNames=elem1&offset=10&limit=20", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ListResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Len(t, response.Data, 1)
	require.Equal(t, 10, response.Metadata.Offset)
	require.Equal(t, 20, response.Metadata.Limit)
}

// TestListPurposes_MissingOrgID tests missing org-id header
func TestListPurposes_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes", nil)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "ListPurposes")
}

// TestListPurposes_InvalidPagination tests invalid pagination parameters
func TestListPurposes_InvalidPagination(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	purposes := []model.ConsentPurpose{}

	// Invalid values should fall back to defaults: limit=100, offset=0
	mockService.On("ListPurposes", mock.Anything, testOrgID, "", []string(nil), []string(nil), 0, 100).
		Return(purposes, 0, nil)

	handler := newConsentPurposeHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes?limit=invalid&offset=invalid", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

// TestUpdatePurpose_Success tests successful purpose update
func TestUpdatePurpose_Success(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	request := model.UpdateRequest{
		Name:        "Updated Purpose",
		Description: "Updated Description",
		Elements: []model.ElementInput{
			{ElementName: "element1", IsMandatory: true},
		},
	}

	expectedPurpose := &model.ConsentPurpose{
		ID:          testPurposeID,
		Name:        "Updated Purpose",
		Description: stringPtr("Updated Description"),
		ClientID:    testClientID,
		Elements:    []model.PurposeElement{},
		UpdatedTime: 1234567900,
		OrgID:       testOrgID,
	}

	mockService.On("UpdatePurpose", mock.Anything, testPurposeID, request, testOrgID, testClientID).
		Return(expectedPurpose, nil)

	handler := newConsentPurposeHandler(mockService)
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPut, "/consent-purposes/"+testPurposeID, bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.updatePurpose(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "Updated Purpose", response.Name)
}

// TestUpdatePurpose_MissingHeaders tests missing required headers
func TestUpdatePurpose_MissingHeaders(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	request := model.UpdateRequest{Name: "Test", Elements: []model.ElementInput{{ElementName: "elem"}}}
	body, _ := json.Marshal(request)

	// Missing org-id
	req := httptest.NewRequest(http.MethodPut, "/consent-purposes/"+testPurposeID, bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()
	handler.updatePurpose(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	// Missing client-id
	req = httptest.NewRequest(http.MethodPut, "/consent-purposes/"+testPurposeID, bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr = httptest.NewRecorder()
	handler.updatePurpose(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	mockService.AssertNotCalled(t, "UpdatePurpose")
}

// TestUpdatePurpose_InvalidJSON tests malformed JSON request
func TestUpdatePurpose_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	req := httptest.NewRequest(http.MethodPut, "/consent-purposes/"+testPurposeID, bytes.NewBufferString("{invalid"))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.updatePurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "UpdatePurpose")
}

// TestDeletePurpose_Success tests successful purpose deletion
func TestDeletePurpose_Success(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	mockService.On("DeletePurpose", mock.Anything, testPurposeID, testOrgID).
		Return(nil)

	handler := newConsentPurposeHandler(mockService)
	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.deletePurpose(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Empty(t, rr.Body.String())
}

// TestDeletePurpose_MissingOrgID tests missing org-id header
func TestDeletePurpose_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID, nil)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.deletePurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "DeletePurpose")
}

// TestDeletePurpose_NotFound tests deleting non-existent purpose
func TestDeletePurpose_NotFound(t *testing.T) {
	mockService := NewMockConsentPurposeService(t)

	serviceErr := &ErrorPurposeNotFound

	mockService.On("DeletePurpose", mock.Anything, testPurposeID, testOrgID).
		Return(serviceErr)

	handler := newConsentPurposeHandler(mockService)
	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.deletePurpose(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}
