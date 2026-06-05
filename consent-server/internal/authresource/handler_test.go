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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wso2/openfgc/internal/authresource/model"
	"github.com/wso2/openfgc/internal/system/constants"
)

const (
	testOrgID     = "org-123"
	testConsentID = "consent-456"
	testAuthID    = "auth-789"
)

func TestHandleCreate_Success(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	userID := "user-123"
	// Status is intentionally omitted — it is optional; the service applies the default.
	// Including a status here would require config to be initialised (validator checks it).
	request := model.AuthResourceCreateRequest{
		Type:   "accounts",
		UserID: &userID,
	}

	expectedOutput := &model.AuthResourceOutput{
		AuthID:     testAuthID,
		AuthType:   "accounts",
		AuthStatus: "APPROVED",
		UserID:     &userID,
	}

	mockService.On("CreateAuthResource", mock.Anything, testConsentID, testOrgID,
		mock.AnythingOfType("model.CreateAuthResourceInput")).
		Return(expectedOutput, nil)

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents/"+testConsentID+"/authorizations", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	rr := httptest.NewRecorder()

	handler.handleCreate(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestHandleCreate_MissingOrgID(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	request := model.AuthResourceCreateRequest{
		Type:   "accounts",
		Status: "authorized",
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents/"+testConsentID+"/authorizations", bytes.NewBuffer(body))
	req.SetPathValue("consentId", testConsentID)
	rr := httptest.NewRecorder()

	handler.handleCreate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleCreate_MissingConsentID(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	request := model.AuthResourceCreateRequest{
		Type:   "accounts",
		Status: "authorized",
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/consents//authorizations", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.handleCreate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleCreate_InvalidJSON(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/consents/"+testConsentID+"/authorizations", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	rr := httptest.NewRecorder()

	handler.handleCreate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGet_Success(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	uid := "user-1"
	expectedOutput := &model.AuthResourceOutput{
		AuthID:     testAuthID,
		ConsentID:  testConsentID,
		AuthType:   "authorisation",
		UserID:     &uid,
		AuthStatus: "APPROVED",
	}

	mockService.On("GetAuthResource", mock.Anything, testAuthID, testConsentID, testOrgID).
		Return(expectedOutput, nil)

	req := httptest.NewRequest(http.MethodGet, "/consents/"+testConsentID+"/authorizations/"+testAuthID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	req.SetPathValue("authorizationId", testAuthID)
	rr := httptest.NewRecorder()

	handler.handleGet(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestHandleGet_ServiceError(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	svcErr := &ErrorAuthResourceNotFound

	mockService.On("GetAuthResource", mock.Anything, testAuthID, testConsentID, testOrgID).
		Return((*model.AuthResourceOutput)(nil), svcErr)

	req := httptest.NewRequest(http.MethodGet, "/consents/"+testConsentID+"/authorizations/"+testAuthID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	req.SetPathValue("authorizationId", testAuthID)
	rr := httptest.NewRecorder()

	handler.handleGet(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleGet_MissingOrgID(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consents/"+testConsentID+"/authorizations/"+testAuthID, nil)
	req.SetPathValue("consentId", testConsentID)
	req.SetPathValue("authorizationId", testAuthID)
	rr := httptest.NewRecorder()

	handler.handleGet(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGet_MissingAuthID(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consents/"+testConsentID+"/authorizations/", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	rr := httptest.NewRecorder()

	handler.handleGet(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleListByConsent_Success(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	expectedOutput := &model.AuthResourceListOutput{
		Data: []model.AuthResourceOutput{
			{
				AuthID:     testAuthID,
				AuthType:   "accounts",
				AuthStatus: "authorized",
			},
		},
	}

	mockService.On("GetAuthResourcesByConsentID", mock.Anything, testConsentID, testOrgID).
		Return(expectedOutput, nil)

	req := httptest.NewRequest(http.MethodGet, "/consents/"+testConsentID+"/authorizations", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	rr := httptest.NewRecorder()

	handler.handleListByConsent(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestHandleListByConsent_MissingOrgID(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consents/"+testConsentID+"/authorizations", nil)
	req.SetPathValue("consentId", testConsentID)
	rr := httptest.NewRecorder()

	handler.handleListByConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleListByConsent_MissingConsentID(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/consents//authorizations", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.handleListByConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleUpdate_Success(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	userID := "user-001"
	updateReq := model.AuthResourceUpdateRequest{
		UserID: &userID,
		Type:   "re-authorisation",
	}
	expectedOutput := &model.AuthResourceOutput{
		AuthID:     testAuthID,
		ConsentID:  testConsentID,
		AuthType:   "re-authorisation",
		AuthStatus: "APPROVED",
	}

	mockService.On("UpdateAuthResource", mock.Anything, testAuthID, testConsentID, testOrgID,
		mock.AnythingOfType("model.UpdateAuthResourceInput")).
		Return(expectedOutput, nil)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/consents/"+testConsentID+"/authorizations/"+testAuthID, bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	req.SetPathValue("authorizationId", testAuthID)
	rr := httptest.NewRecorder()

	handler.handleUpdate(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestHandleUpdate_ServiceError(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	userID := "user-001"
	updateReq := model.AuthResourceUpdateRequest{
		UserID: &userID,
		Type:   "re-authorisation",
	}
	svcErr := &ErrorAuthResourceNotFound

	mockService.On("UpdateAuthResource", mock.Anything, testAuthID, testConsentID, testOrgID,
		mock.AnythingOfType("model.UpdateAuthResourceInput")).
		Return((*model.AuthResourceOutput)(nil), svcErr)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/consents/"+testConsentID+"/authorizations/"+testAuthID, bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	req.SetPathValue("authorizationId", testAuthID)
	rr := httptest.NewRecorder()

	handler.handleUpdate(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleUpdate_MissingUserID(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	// userId is required — omitting it must yield 400.
	updateReq := model.AuthResourceUpdateRequest{
		Status: "revoked",
	}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/consents/"+testConsentID+"/authorizations/"+testAuthID, bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	req.SetPathValue("authorizationId", testAuthID)
	rr := httptest.NewRecorder()

	handler.handleUpdate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleUpdate_MissingOrgID(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	updateReq := model.AuthResourceUpdateRequest{
		Status: "revoked",
	}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/consents/"+testConsentID+"/authorizations/"+testAuthID, bytes.NewBuffer(body))
	req.SetPathValue("consentId", testConsentID)
	req.SetPathValue("authorizationId", testAuthID)
	rr := httptest.NewRecorder()

	handler.handleUpdate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleUpdate_InvalidJSON(t *testing.T) {
	mockService := NewMockAuthResourceService(t)
	handler := newAuthResourceHandler(mockService)

	req := httptest.NewRequest(http.MethodPut, "/consents/"+testConsentID+"/authorizations/"+testAuthID, bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("consentId", testConsentID)
	req.SetPathValue("authorizationId", testAuthID)
	rr := httptest.NewRecorder()

	handler.handleUpdate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}
