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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/system/config"
	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
)

const (
	testOrgID    = "org-123"
	testClientID = "client-456"
)

func TestCreateConsent_Success(t *testing.T) {
	mockService := NewMockConsentService(t)

	request := model.ConsentAPIRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}

	expectedResponse := &model.ConsentResponse{
		ConsentID:     "consent-123",
		ConsentType:   "accounts",
		CurrentStatus: "awaitingAuthorization",
		CreatedTime:   1234567890,
		UpdatedTime:   1234567890,
	}

	mockService.On("CreateConsent", mock.Anything, request, testClientID, testOrgID).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	mockService.AssertExpectations(t)
}

func TestCreateConsent_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	request := model.ConsentAPIRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	// Missing org-id header
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetConsent_Success(t *testing.T) {
	mockService := NewMockConsentService(t)

	expectedResponse := &model.ConsentResponse{
		ConsentID:     "550e8400-e29b-41d4-a716-446655440000",
		ConsentType:   "accounts",
		CurrentStatus: "active",
		ClientID:      testClientID,
		OrgID:         testOrgID,
	}

	mockService.On("GetConsent", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", testOrgID).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}", handler.getConsent)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/consents/550e8400-e29b-41d4-a716-446655440000", nil)
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response model.ConsentAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", response.ID)
	require.Equal(t, "accounts", response.Type)
	mockService.AssertExpectations(t)
}

func TestGetConsent_NotFound(t *testing.T) {
	mockService := NewMockConsentService(t)

	mockService.On("GetConsent", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", testOrgID).
		Return(nil, serviceerror.CustomServiceError(
			ErrorConsentNotFound,
			"Consent not found",
		))

	handler := newConsentHandler(mockService)
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}", handler.getConsent)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/consents/550e8400-e29b-41d4-a716-446655440001", nil)
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestListConsents_Success(t *testing.T) {
	mockService := NewMockConsentService(t)

	expectedResponse := &model.ConsentDetailSearchResponse{
		Data: []model.ConsentDetailResponse{
			{
				ID:     "consent-1",
				Type:   "accounts",
				Status: "active",
			},
		},
		Metadata: model.ConsentSearchMetadata{
			Total:  1,
			Limit:  10,
			Offset: 0,
			Count:  1,
		},
	}

	mockService.On("SearchConsentsDetailed", mock.Anything, mock.AnythingOfType("model.ConsentSearchFilters")).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ConsentDetailSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Len(t, response.Data, 1)
	mockService.AssertExpectations(t)
}

func TestRevokeConsent_Success(t *testing.T) {
	mockService := NewMockConsentService(t)

	revokeRequest := model.ConsentRevokeRequest{
		ActionBy:         "user-123",
		RevocationReason: "User requested revocation",
	}

	expectedResponse := &model.ConsentRevokeResponse{
		ActionTime:       1234567890,
		ActionBy:         "user-123",
		RevocationReason: "User requested revocation",
	}

	mockService.On("RevokeConsent", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", testOrgID, revokeRequest).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consents/{consentId}/revoke", handler.revokeConsent)

	server := httptest.NewServer(mux)
	defer server.Close()

	body, err := json.Marshal(revokeRequest)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, server.URL+"/api/v1/consents/550e8400-e29b-41d4-a716-446655440000/revoke", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderContentType, constants.ContentTypeJSON)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response model.ConsentRevokeResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "user-123", response.ActionBy)
	require.Equal(t, "User requested revocation", response.RevocationReason)
	mockService.AssertExpectations(t)
}

func TestValidateConsent_Success(t *testing.T) {
	mockService := NewMockConsentService(t)

	validateRequest := model.ValidateRequest{
		ConsentID: "consent-123",
	}

	expectedResponse := &model.ValidateResponse{
		IsValid: true,
		ConsentInformation: &model.ValidateConsentAPIResponse{
			ID:   "consent-123",
			Type: "accounts",
		},
	}

	mockService.On("ValidateConsent", mock.Anything, validateRequest, testOrgID).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	body, _ := json.Marshal(validateRequest)
	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ValidateResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.True(t, response.IsValid)
	mockService.AssertExpectations(t)
}

func TestHandler_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateConsent_MissingClientID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	request := model.ConsentAPIRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	// Missing client-id header
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListConsents_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	// Missing org-id header
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListConsents_InvalidLimit(t *testing.T) {
	mockService := NewMockConsentService(t)

	// Mock the call with default limit (10) since invalid param uses default
	mockService.On("SearchConsentsDetailed", mock.Anything, mock.AnythingOfType("model.ConsentSearchFilters")).
		Return(&model.ConsentDetailSearchResponse{
			Data:     []model.ConsentDetailResponse{},
			Metadata: model.ConsentSearchMetadata{Total: 0, Limit: 10, Offset: 0, Count: 0},
		}, nil)

	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consents?limit=invalid", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	// Should succeed with default limit
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestListConsents_InvalidOffset(t *testing.T) {
	mockService := NewMockConsentService(t)

	// Mock the call with default offset (0) since invalid param uses default
	mockService.On("SearchConsentsDetailed", mock.Anything, mock.AnythingOfType("model.ConsentSearchFilters")).
		Return(&model.ConsentDetailSearchResponse{
			Data:     []model.ConsentDetailResponse{},
			Metadata: model.ConsentSearchMetadata{Total: 0, Limit: 10, Offset: 0, Count: 0},
		}, nil)

	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consents?offset=invalid", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	// Should succeed with default offset
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestListConsents_ServiceError(t *testing.T) {
	mockService := NewMockConsentService(t)

	mockService.On("SearchConsentsDetailed", mock.Anything, mock.AnythingOfType("model.ConsentSearchFilters")).
		Return(nil, &ErrorInternalServerError)

	handler := newConsentHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

func TestValidateConsent_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestValidateConsent_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	validateRequest := model.ValidateRequest{
		ConsentID: "consent-123",
	}

	body, _ := json.Marshal(validateRequest)
	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBuffer(body))
	// Missing org-id header
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestValidateConsent_ServiceError(t *testing.T) {
	mockService := NewMockConsentService(t)

	validateRequest := model.ValidateRequest{
		ConsentID: "consent-123",
	}

	mockService.On("ValidateConsent", mock.Anything, validateRequest, testOrgID).
		Return(nil, &ErrorInternalServerError)

	handler := newConsentHandler(mockService)
	body, _ := json.Marshal(validateRequest)
	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

func TestCreateConsent_ServiceValidationError(t *testing.T) {
	mockService := NewMockConsentService(t)

	request := model.ConsentAPIRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "accounts"},
		},
	}

	mockService.On("CreateConsent", mock.Anything, request, testClientID, testOrgID).
		Return(nil, &ErrorValidationFailed)

	handler := newConsentHandler(mockService)
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertExpectations(t)
}

func TestListConsents_WithFilters(t *testing.T) {
	mockService := NewMockConsentService(t)

	expectedResponse := &model.ConsentDetailSearchResponse{
		Data: []model.ConsentDetailResponse{
			{
				ID:     "consent-1",
				Type:   "accounts",
				Status: "active",
			},
			{
				ID:     "consent-2",
				Type:   "payments",
				Status: "active",
			},
		},
		Metadata: model.ConsentSearchMetadata{
			Total:  2,
			Limit:  10,
			Offset: 0,
			Count:  2,
		},
	}

	mockService.On("SearchConsentsDetailed", mock.Anything, mock.AnythingOfType("model.ConsentSearchFilters")).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consents?consentType=accounts&status=active&limit=10&offset=0", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ConsentDetailSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Len(t, response.Data, 2)
	require.Equal(t, 2, response.Metadata.Total)
	mockService.AssertExpectations(t)
}

func TestListConsents_EmptyResult(t *testing.T) {
	mockService := NewMockConsentService(t)

	expectedResponse := &model.ConsentDetailSearchResponse{
		Data: []model.ConsentDetailResponse{},
		Metadata: model.ConsentSearchMetadata{
			Total:  0,
			Limit:  10,
			Offset: 0,
			Count:  0,
		},
	}

	mockService.On("SearchConsentsDetailed", mock.Anything, mock.AnythingOfType("model.ConsentSearchFilters")).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ConsentDetailSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Empty(t, response.Data)
	require.Equal(t, 0, response.Metadata.Total)
	mockService.AssertExpectations(t)
}

func TestValidateConsent_InvalidConsentResponse(t *testing.T) {
	mockService := NewMockConsentService(t)

	validateRequest := model.ValidateRequest{
		ConsentID: "invalid-consent",
	}

	expectedResponse := &model.ValidateResponse{
		IsValid:          false,
		ErrorCode:        404,
		ErrorMessage:     "invalid_consent",
		ErrorDescription: "Consent not found",
	}

	mockService.On("ValidateConsent", mock.Anything, validateRequest, testOrgID).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	body, _ := json.Marshal(validateRequest)
	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ValidateResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.False(t, response.IsValid)
	require.Equal(t, 404, response.ErrorCode)
	mockService.AssertExpectations(t)
}

func TestRevokeConsent_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	revokeReq := model.ConsentRevokeRequest{
		ActionBy: "user-123",
	}
	body, _ := json.Marshal(revokeReq)
	req := httptest.NewRequest(http.MethodPost, "/consents/consent-123/revoke", bytes.NewBuffer(body))
	// Missing org-id header
	rr := httptest.NewRecorder()

	handler.revokeConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRevokeConsent_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/consents/consent-123/revoke", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.revokeConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateConsent_Success(t *testing.T) {
	mockService := NewMockConsentService(t)

	updateRequest := model.ConsentAPIUpdateRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
	}

	expectedResponse := &model.ConsentResponse{
		ConsentID:     "550e8400-e29b-41d4-a716-446655440000",
		ConsentType:   "accounts",
		CurrentStatus: "active",
		ClientID:      testClientID,
		OrgID:         testOrgID,
	}

	mockService.On("UpdateConsent", mock.Anything, updateRequest, testClientID, testOrgID, "550e8400-e29b-41d4-a716-446655440000").
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consents/{consentId}", handler.updateConsent)

	server := httptest.NewServer(mux)
	defer server.Close()

	body, err := json.Marshal(updateRequest)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, server.URL+"/api/v1/consents/550e8400-e29b-41d4-a716-446655440000", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	req.Header.Set(constants.HeaderContentType, constants.ContentTypeJSON)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response model.ConsentAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", response.ID)
	require.Equal(t, "accounts", response.Type)
	mockService.AssertExpectations(t)
}

func TestUpdateConsent_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	updateReq := model.ConsentAPIUpdateRequest{}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/consents/consent-123", bytes.NewBuffer(body))
	// Missing org-id header
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.updateConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateConsent_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodPut, "/consents/consent-123", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.updateConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSearchConsentsByAttribute_Success(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	expectedResponse := &model.ConsentAttributeSearchResponse{
		ConsentIDs: []string{"consent-123", "consent-456"},
		Count:      2,
	}

	mockService.On("SearchConsentsByAttribute", mock.Anything, "key1", "value1", testOrgID).
		Return(expectedResponse, nil)

	req := httptest.NewRequest(http.MethodGet, "/consents/search?key=key1&value=value1", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.searchConsentsByAttribute(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestSearchConsentsByAttribute_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consents/search?key=key1", nil)
	// Missing org-id header
	rr := httptest.NewRecorder()

	handler.searchConsentsByAttribute(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// ==================== Delegation Tests ====================

// TestCreateDelegatedConsent_Success tests creating a consent with delegation object
func TestCreateDelegatedConsent_Success(t *testing.T) {
	mockService := NewMockConsentService(t)

	request := model.ConsentAPIRequest{
		Type: "food_delivery_consent",
		Delegation: &model.DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-456",
		},
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "food_delivery",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "child_name", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{UserID: "mother-111", Status: "APPROVED", Resources: map[string]interface{}{}},
			{UserID: "father-222", Status: "APPROVED", Resources: map[string]interface{}{}},
		},
	}

	expectedResponse := &model.ConsentResponse{
		ConsentID:   "consent-delegation-123",
		ConsentType: "food_delivery_consent",
		Delegation: &model.ConsentDelegation{
			ConsentID:        "consent-delegation-123",
			DelegationType:   "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-456",
		},
	}

	mockService.On("CreateConsent", mock.Anything, request, testClientID, testOrgID).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var response model.ConsentAPIResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.NotNil(t, response.Delegation, "Delegation should be present in response")
	require.Equal(t, "parental_biological", response.Delegation.DelegationType)
	require.Equal(t, "ANY", response.Delegation.RevocationPolicy)
	require.Equal(t, "child-456", response.Delegation.OnBehalfOf)
	mockService.AssertExpectations(t)
}

// TestCreateConsent_WithoutDelegation_NoDelegationInResponse tests that normal consent has no delegation
func TestCreateConsent_WithoutDelegation_NoDelegationInResponse(t *testing.T) {
	mockService := NewMockConsentService(t)

	request := model.ConsentAPIRequest{
		Type: "marketing",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements: []model.ConsentElementApprovalItem{
					{ElementName: "element-1", IsUserApproved: true},
				},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "authorisation", UserID: "user@example.com", Status: "APPROVED"},
		},
	}

	expectedResponse := &model.ConsentResponse{
		ConsentID:   "consent-normal-123",
		ConsentType: "marketing",
		Delegation:  nil,
	}

	mockService.On("CreateConsent", mock.Anything, request, testClientID, testOrgID).
		Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var response model.ConsentAPIResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Nil(t, response.Delegation, "Delegation should be nil for non-delegated consent")
	mockService.AssertExpectations(t)
}

// TestListConsents_WithDelegationFilter tests the delegation=true query parameter
func TestListConsents_WithDelegationFilter(t *testing.T) {
	mockService := NewMockConsentService(t)

	expectedResponse := &model.ConsentDetailSearchResponse{
		Data: []model.ConsentDetailResponse{
			{
				ID:     "consent-delegation-123",
				Type:   "food_delivery_consent",
				Status: "ACTIVE",
				Delegation: &model.ConsentDelegation{
					DelegationType:   "parental_biological",
					RevocationPolicy: "ANY",
					OnBehalfOf:       "child-456",
				},
			},
		},
		Metadata: model.ConsentSearchMetadata{
			Total:  1,
			Limit:  25,
			Offset: 0,
			Count:  1,
		},
	}

	mockService.On("SearchConsentsDetailed", mock.Anything, mock.MatchedBy(func(filters model.ConsentSearchFilters) bool {
		return filters.IsDelegated != nil && *filters.IsDelegated == true
	})).Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consents?delegation=true", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response model.ConsentDetailSearchResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	require.Len(t, response.Data, 1)
	require.NotNil(t, response.Data[0].Delegation)
	require.Equal(t, "child-456", response.Data[0].Delegation.OnBehalfOf)
	mockService.AssertExpectations(t)
}

// TestListConsents_DelegationFilterNotSet tests that without delegation param, IsDelegated is nil
func TestListConsents_DelegationFilterNotSet(t *testing.T) {
	mockService := NewMockConsentService(t)

	expectedResponse := &model.ConsentDetailSearchResponse{
		Data:     []model.ConsentDetailResponse{},
		Metadata: model.ConsentSearchMetadata{Total: 0, Limit: 25, Offset: 0, Count: 0},
	}

	mockService.On("SearchConsentsDetailed", mock.Anything, mock.MatchedBy(func(filters model.ConsentSearchFilters) bool {
		return filters.IsDelegated == nil
	})).Return(expectedResponse, nil)

	handler := newConsentHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

// TestDelegationRequest_Structure tests DelegationRequest struct fields
func TestDelegationRequest_Structure(t *testing.T) {
	delegation := model.DelegationRequest{
		Type:             "parental_biological",
		RevocationPolicy: "ANY",
		OnBehalfOf:       "child-456",
	}

	require.Equal(t, "parental_biological", delegation.Type)
	require.Equal(t, "ANY", delegation.RevocationPolicy)
	require.Equal(t, "child-456", delegation.OnBehalfOf)
}

// TestConsentDelegation_Structure tests ConsentDelegation DB model struct
func TestConsentDelegation_Structure(t *testing.T) {
	delegation := model.ConsentDelegation{
		ConsentID:        "consent-123",
		DelegationType:   "registered_carer",
		RevocationPolicy: "BOTH",
		OnBehalfOf:       "patient-321",
		OrgID:            "hospital-org",
	}

	require.Equal(t, "consent-123", delegation.ConsentID)
	require.Equal(t, "registered_carer", delegation.DelegationType)
	require.Equal(t, "BOTH", delegation.RevocationPolicy)
	require.Equal(t, "patient-321", delegation.OnBehalfOf)
	require.Equal(t, "hospital-org", delegation.OrgID)
}

// TestDelegationRequest_JSONMarshaling tests JSON serialization of DelegationRequest
func TestDelegationRequest_JSONMarshaling(t *testing.T) {
	delegation := model.DelegationRequest{
		Type:             "parental_biological",
		RevocationPolicy: "ANY",
		OnBehalfOf:       "child-456",
	}

	data, err := json.Marshal(delegation)
	require.NoError(t, err)

	var parsed model.DelegationRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	require.Equal(t, delegation.Type, parsed.Type)
	require.Equal(t, delegation.RevocationPolicy, parsed.RevocationPolicy)
	require.Equal(t, delegation.OnBehalfOf, parsed.OnBehalfOf)
}

// TestConsentAPIRequest_WithDelegation tests ConsentAPIRequest with delegation field
func TestConsentAPIRequest_WithDelegation(t *testing.T) {
	req := model.ConsentAPIRequest{
		Type: "food_delivery_consent",
		Delegation: &model.DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-456",
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{UserID: "mother-111", Status: "APPROVED"},
		},
	}

	require.NotNil(t, req.Delegation)
	require.Equal(t, "parental_biological", req.Delegation.Type)
	require.Equal(t, "child-456", req.Delegation.OnBehalfOf)
}

// TestConsentAPIRequest_WithoutDelegation tests ConsentAPIRequest without delegation
func TestConsentAPIRequest_WithoutDelegation(t *testing.T) {
	req := model.ConsentAPIRequest{
		Type: "marketing",
		Authorizations: []model.AuthorizationAPIRequest{
			{Type: "authorisation", UserID: "user@example.com"},
		},
	}

	require.Nil(t, req.Delegation)
}

// TestToConsentCreateRequest_InfersAuthType tests that auth type is inferred as "delegate"
func TestToConsentCreateRequest_InfersAuthType(t *testing.T) {
	// Initialize config for ToConsentCreateRequest
	prev := config.Get()
	t.Cleanup(func() { config.SetGlobal(prev) })
	config.SetGlobal(&config.Config{
		Consent: config.ConsentConfig{
			AuthStatusMappings: config.AuthStatusMappings{
				ApprovedState: "APPROVED",
				CreatedState:  "CREATED",
				RejectedState: "REJECTED",
			},
		},
	})

	req := model.ConsentAPIRequest{
		Type: "food_delivery_consent",
		Delegation: &model.DelegationRequest{
			Type:             "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-456",
		},
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "food_delivery",
				Elements:    []model.ConsentElementApprovalItem{{ElementName: "child_name", IsUserApproved: true}},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{UserID: "mother-111", Type: "authorisation", Status: "APPROVED"},
			{UserID: "father-222", Status: "APPROVED"},
		},
	}

	createReq, err := req.ToConsentCreateRequest()
	require.NoError(t, err)

	for _, auth := range createReq.AuthResources {
		require.Equal(t, "delegate", auth.AuthType, "Auth type should be inferred as 'delegate' when delegation is present")
	}
}

// TestToConsentCreateRequest_NoInferenceWithoutDelegation tests auth type is NOT inferred without delegation
func TestToConsentCreateRequest_NoInferenceWithoutDelegation(t *testing.T) {
	// Initialize config for ToConsentCreateRequest
	prev := config.Get()
	t.Cleanup(func() { config.SetGlobal(prev) })
	config.SetGlobal(&config.Config{
		Consent: config.ConsentConfig{
			AuthStatusMappings: config.AuthStatusMappings{
				ApprovedState: "APPROVED",
				CreatedState:  "CREATED",
				RejectedState: "REJECTED",
			},
		},
	})

	req := model.ConsentAPIRequest{
		Type: "marketing",
		Purposes: []model.ConsentPurposeItem{
			{
				PurposeName: "purpose-1",
				Elements:    []model.ConsentElementApprovalItem{{ElementName: "elem-1", IsUserApproved: true}},
			},
		},
		Authorizations: []model.AuthorizationAPIRequest{
			{UserID: "user@example.com", Type: "authorisation", Status: "APPROVED"},
		},
	}

	createReq, err := req.ToConsentCreateRequest()
	require.NoError(t, err)

	require.Equal(t, "authorisation", createReq.AuthResources[0].AuthType)
}

// TestConsentSearchFilters_DelegationFilter tests IsDelegated filter field
func TestConsentSearchFilters_DelegationFilter(t *testing.T) {
	isDelegated := true
	filters := model.ConsentSearchFilters{
		OrgID:       "test-org",
		IsDelegated: &isDelegated,
	}

	require.NotNil(t, filters.IsDelegated)
	require.True(t, *filters.IsDelegated)
}

// TestConsentSearchFilters_ConsentIDsFilter tests ConsentIDs filter field
func TestConsentSearchFilters_ConsentIDsFilter(t *testing.T) {
	filters := model.ConsentSearchFilters{
		OrgID:      "test-org",
		ConsentIDs: []string{"consent-1", "consent-2"},
	}

	require.Len(t, filters.ConsentIDs, 2)
	require.Equal(t, "consent-1", filters.ConsentIDs[0])
}

// TestConsentResponse_WithDelegation tests ConsentResponse includes delegation
func TestConsentResponse_WithDelegation(t *testing.T) {
	response := model.ConsentResponse{
		ConsentID:   "consent-123",
		ConsentType: "food_delivery_consent",
		Delegation: &model.ConsentDelegation{
			ConsentID:        "consent-123",
			DelegationType:   "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-456",
		},
	}

	require.NotNil(t, response.Delegation)
	require.Equal(t, "child-456", response.Delegation.OnBehalfOf)
}

// TestConsentDetailResponse_WithDelegation tests ConsentDetailResponse includes delegation
func TestConsentDetailResponse_WithDelegation(t *testing.T) {
	response := model.ConsentDetailResponse{
		ID:   "consent-123",
		Type: "food_delivery_consent",
		Delegation: &model.ConsentDelegation{
			DelegationType:   "parental_biological",
			RevocationPolicy: "ANY",
			OnBehalfOf:       "child-456",
		},
	}

	require.NotNil(t, response.Delegation)
	require.Equal(t, "parental_biological", response.Delegation.DelegationType)
}
