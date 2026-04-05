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

	// Set the header so the handler overwrites ActionBy correctly.
	req.Header.Set("X-User-ID", "user-123")

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

// ── GET /consents/{consentId}/delegates ──────────────────────────────────────

func TestGetDelegates_Success(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	consentID := "550e8400-e29b-41d4-a716-446655440000"
	expectedResponse := &model.DelegateListResponse{
		ConsentID:        consentID,
		DataPrincipalID:  "child_user_123",
		RevocationPolicy: "ANY",
		DelegateCount:    2,
		Delegates: []model.DelegateInfo{
			{
				AuthID:         "auth-001",
				UserID:         "parent_mom_456",
				DelegationType: "parental_biological",
				Status:         "approved",
				CanRevoke:      true,
				CanModify:      true,
				OnBehalfOf:     "child_user_123",
				UpdatedTime:    1700000000000,
			},
			{
				AuthID:         "auth-002",
				UserID:         "parent_dad_789",
				DelegationType: "parental_biological",
				Status:         "approved",
				CanRevoke:      true,
				CanModify:      true,
				OnBehalfOf:     "child_user_123",
				UpdatedTime:    1700000000000,
			},
		},
	}

	mockService.On("GetConsentDelegates", mock.Anything, consentID, testOrgID).
		Return(expectedResponse, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}/delegates", handler.getDelegates)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/consents/"+consentID+"/delegates", nil)
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response model.DelegateListResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, consentID, response.ConsentID)
	require.Equal(t, 2, response.DelegateCount)
	require.Equal(t, "child_user_123", response.DataPrincipalID)
	require.Equal(t, "ANY", response.RevocationPolicy)
	mockService.AssertExpectations(t)
}

func TestGetDelegates_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/consents/550e8400-e29b-41d4-a716-446655440000/delegates", nil)
	// No org-id header
	rr := httptest.NewRecorder()

	handler.getDelegates(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetDelegates_InvalidConsentID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	// Calling handler.getDelegates directly bypasses the router, meaning
	// PathValue always returns "" and the test passes for the wrong reason
	// (missing field) instead of exercising the malformed-UUID validation path.
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}/delegates", handler.getDelegates)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/consents/not-a-valid-uuid!!/delegates", nil)
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetDelegates_NotFound(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	consentID := "550e8400-e29b-41d4-a716-446655440001"

	mockService.On("GetConsentDelegates", mock.Anything, consentID, testOrgID).
		Return(nil, serviceerror.CustomServiceError(ErrorConsentNotFound, "Consent not found"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}/delegates", handler.getDelegates)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/consents/"+consentID+"/delegates", nil)
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestGetDelegates_ServiceError(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	consentID := "550e8400-e29b-41d4-a716-446655440002"

	mockService.On("GetConsentDelegates", mock.Anything, consentID, testOrgID).
		Return(nil, serviceerror.CustomServiceError(ErrorInternalServerError, "database error"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}/delegates", handler.getDelegates)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/consents/"+consentID+"/delegates", nil)
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestGetDelegates_NoDelegates(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	consentID := "550e8400-e29b-41d4-a716-446655440003"
	expectedResponse := &model.DelegateListResponse{
		ConsentID:     consentID,
		DelegateCount: 0,
		Delegates:     []model.DelegateInfo{},
	}

	mockService.On("GetConsentDelegates", mock.Anything, consentID, testOrgID).
		Return(expectedResponse, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}/delegates", handler.getDelegates)

	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/consents/"+consentID+"/delegates", nil)
	require.NoError(t, err)
	req.Header.Set(constants.HeaderOrgID, testOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response model.DelegateListResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, 0, response.DelegateCount)
	require.Empty(t, response.Delegates)
	mockService.AssertExpectations(t)
}

// ── GET /consents?dataPrincipalId= ───────────────────────────────────────────

func TestListConsents_WithDataPrincipalID(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	expectedResponse := &model.ConsentDetailSearchResponse{
		Data: []model.ConsentDetailResponse{},
		Metadata: model.ConsentSearchMetadata{
			Total:  0,
			Limit:  10,
			Offset: 0,
			Count:  0,
		},
	}

	// to ConsentSearchFilters.CallerID so the delegation authorization path
	// is exercised (X-User-ID is required for dataPrincipalId queries).
	mockService.On("SearchConsentsDetailed", mock.Anything,
		mock.MatchedBy(func(f model.ConsentSearchFilters) bool {
			return f.DataPrincipalID == "child_user_123" &&
				f.OrgID == testOrgID &&
				f.CallerID == "parent_user_456"
		}),
	).Return(expectedResponse, nil)

	req := httptest.NewRequest(http.MethodGet, "/consents?dataPrincipalId=child_user_123", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set("X-User-ID", "parent_user_456")
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

// GET /consents?dataPrincipalId=X without being a delegate → 403

func TestListConsents_UnauthorizedForPrincipal(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	// Simulate unauthorized access error from service layer
	// service-layer authorization check is properly exercised.
	mockService.On("SearchConsentsDetailed", mock.Anything,
		mock.MatchedBy(func(f model.ConsentSearchFilters) bool {
			return f.DataPrincipalID == "unauthorized_user" &&
				f.OrgID == testOrgID &&
				f.CallerID == "some_caller_999"
		}),
	).Return(nil, serviceerror.CustomServiceError(
		ErrorUnauthorized,
		"not allowed to access this principal",
	))

	req := httptest.NewRequest(http.MethodGet, "/consents?dataPrincipalId=unauthorized_user", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set("X-User-ID", "some_caller_999")
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
	mockService.AssertExpectations(t)
}

// — POST /consents with delegation.type but no principal_id → 400

func TestCreateConsent_InvalidDelegationAttributes(t *testing.T) {
	mockService := NewMockConsentService(t)
	handler := newConsentHandler(mockService)

	// ValidateDelegationAttributes checks req.Attributes["delegation.type"].
	// When delegation.type is set but delegation.principal_id is missing,
	// the validator rejects the request before CreateConsent is ever called —
	// so no Mock.On("CreateConsent") is needed here.
	request := model.ConsentAPIRequest{
		Type: "accounts",
		// delegation.type is set → triggers ValidateDelegationAttributes
		// delegation.principal_id is intentionally absent → Rule 1 rejects the request
		Attributes: map[string]string{
			"delegation.type": "guardian",
			// "delegation.principal_id" deliberately omitted
		},
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
	req.Header.Set(constants.HeaderTPPClientID, testClientID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	// Expect 400 Bad Request — delegation.principal_id missing (Rule 1 of ValidateDelegationAttributes)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
