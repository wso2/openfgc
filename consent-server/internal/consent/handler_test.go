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
	"fmt"
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
	handlerTestOrgID     = "test-org-123"
	handlerTestGroupID   = "test-group-456"
	handlerTestConsentID = "550e8400-e29b-41d4-a716-446655440000"
)

// =============================================================================
// createConsent
// =============================================================================

func TestHandlerCreateConsent_Success(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	out := &model.ConsentOutput{
		ConsentID:     handlerTestConsentID,
		GroupID:       handlerTestGroupID,
		ConsentType:   "accounts",
		CurrentStatus: "ACTIVE",
	}
	mockSvc.On("CreateConsent", mock.Anything, mock.Anything, handlerTestOrgID).
		Return(out, nil)

	handler := newConsentHandler(mockSvc)
	reqBody := model.ConsentCreateRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeRefRequest{
			{Name: "Marketing", Elements: []model.ConsentPurposeElementApprovalRequest{}},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	req.Header.Set(constants.HeaderGroupID, handlerTestGroupID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Contains(t, rr.Header().Get(constants.HeaderContentType), "application/json")

	var resp model.ConsentResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, handlerTestConsentID, resp.ConsentID)
}

func TestHandlerCreateConsent_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	reqBody := model.ConsentCreateRequest{Type: "accounts"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderGroupID, handlerTestGroupID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreateConsent")
}

func TestHandlerCreateConsent_MissingGroupID(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	reqBody := model.ConsentCreateRequest{Type: "accounts"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	// No group-id header
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreateConsent")
}

func TestHandlerCreateConsent_InvalidJSON(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBufferString("{invalid"))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	req.Header.Set(constants.HeaderGroupID, handlerTestGroupID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreateConsent")
}

func TestHandlerCreateConsent_ValidationError_EmptyType(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	// type is empty — validator should reject this
	reqBody := model.ConsentCreateRequest{Type: ""}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	req.Header.Set(constants.HeaderGroupID, handlerTestGroupID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreateConsent")
}

func TestHandlerCreateConsent_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	svcErr := &serviceerror.ServiceError{
		Type:    serviceerror.ClientErrorType,
		Code:    "CS-4002",
		Message: "validation failed",
	}
	mockSvc.On("CreateConsent", mock.Anything, mock.Anything, handlerTestOrgID).
		Return(nil, svcErr)

	handler := newConsentHandler(mockSvc)
	reqBody := model.ConsentCreateRequest{
		Type: "accounts",
		Purposes: []model.ConsentPurposeRefRequest{
			{Name: "Marketing", Elements: []model.ConsentPurposeElementApprovalRequest{}},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	req.Header.Set(constants.HeaderGroupID, handlerTestGroupID)
	rr := httptest.NewRecorder()

	handler.createConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// =============================================================================
// getConsent
// =============================================================================

func TestHandlerGetConsent_Success(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	out := &model.ConsentOutput{
		ConsentID:     handlerTestConsentID,
		GroupID:       handlerTestGroupID,
		ConsentType:   "accounts",
		CurrentStatus: "ACTIVE",
	}
	mockSvc.On("GetConsent", mock.Anything, handlerTestConsentID, handlerTestOrgID).Return(out, nil)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents/"+handlerTestConsentID, nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	req.SetPathValue("consentId", handlerTestConsentID)
	rr := httptest.NewRecorder()

	handler.getConsent(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.ConsentResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, handlerTestConsentID, resp.ConsentID)
}

func TestHandlerGetConsent_NotFound(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	mockSvc.On("GetConsent", mock.Anything, handlerTestConsentID, handlerTestOrgID).
		Return(nil, &ErrorConsentNotFound)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents/"+handlerTestConsentID, nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	req.SetPathValue("consentId", handlerTestConsentID)
	rr := httptest.NewRecorder()

	handler.getConsent(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerGetConsent_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents/"+handlerTestConsentID, nil)
	req.SetPathValue("consentId", handlerTestConsentID)
	rr := httptest.NewRecorder()

	handler.getConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "GetConsent")
}

// =============================================================================
// listConsents
// =============================================================================

func TestHandlerListConsents_Success(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	listOut := &model.ConsentListOutput{
		Data:   []model.ConsentOutput{{ConsentID: handlerTestConsentID, ConsentType: "accounts", CurrentStatus: "ACTIVE"}},
		Total:  1,
		Count:  1,
		Offset: 0,
		Limit:  10,
	}
	mockSvc.On("SearchConsents", mock.Anything, mock.Anything).Return(listOut, nil)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.ConsentListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
}

func TestHandlerListConsents_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "SearchConsents")
}

func TestHandlerListConsents_PurposeVersionWithoutPurposeName(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents?purposeVersion=v1", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "SearchConsents")
}

func TestHandlerListConsents_ElementVersionWithoutElementNameOrNamespace(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents?elementVersion=v1", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "SearchConsents")
}

func TestHandlerListConsents_ValidPurposeVersionAndName(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	listOut := &model.ConsentListOutput{
		Data:  []model.ConsentOutput{},
		Total: 0, Count: 0, Offset: 0, Limit: 10,
	}

	v1 := 1
	mockSvc.On("SearchConsents", mock.Anything, model.ConsentSearchFilter{
		OrgID:          handlerTestOrgID,
		Limit:          10,
		Offset:         0,
		PurposeName:    "Marketing",
		PurposeVersion: &v1,
	}).Return(listOut, nil)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents?purposeName=Marketing&purposeVersion=v1", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandlerListConsents_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	svcErr := &serviceerror.ServiceError{
		Type:    serviceerror.ServerErrorType,
		Code:    "CS-5000",
		Message: "internal server error",
	}
	mockSvc.On("SearchConsents", mock.Anything, mock.Anything).Return(nil, svcErr)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.listConsents(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

// =============================================================================
// updateConsent
// =============================================================================

func TestHandlerUpdateConsent_Success(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	out := &model.ConsentOutput{
		ConsentID:     handlerTestConsentID,
		GroupID:       handlerTestGroupID,
		ConsentType:   "accounts",
		CurrentStatus: "ACTIVE",
	}
	mockSvc.On("UpdateConsent", mock.Anything, handlerTestConsentID, mock.Anything, handlerTestOrgID, mock.Anything).
		Return(out, nil)

	handler := newConsentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /consents/{consentId}", handler.updateConsent)
	server := httptest.NewServer(mux)
	defer server.Close()

	reqBody := model.ConsentUpdateRequest{
		Type: "accounts",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/consents/%s", server.URL, handlerTestConsentID), bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	req.Header.Set(constants.HeaderGroupID, handlerTestGroupID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandlerUpdateConsent_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	handler := newConsentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /consents/{consentId}", handler.updateConsent)
	server := httptest.NewServer(mux)
	defer server.Close()

	reqBody := model.ConsentUpdateRequest{Type: "accounts"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/consents/%s", server.URL, handlerTestConsentID), bytes.NewBuffer(body))
	// No org-id header

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockSvc.AssertNotCalled(t, "UpdateConsent")
}

func TestHandlerUpdateConsent_InvalidJSON(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	handler := newConsentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /consents/{consentId}", handler.updateConsent)
	server := httptest.NewServer(mux)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/consents/%s", server.URL, handlerTestConsentID), bytes.NewBufferString("{invalid"))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockSvc.AssertNotCalled(t, "UpdateConsent")
}

// =============================================================================
// revokeConsent
// =============================================================================

func TestHandlerRevokeConsent_Success(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	revokeOut := &model.ConsentRevokeOutput{
		ActionTime: 1700000000000,
		ActionBy:   "admin-user",
		Reason:     "user request",
	}
	mockSvc.On("RevokeConsent", mock.Anything, handlerTestConsentID, handlerTestOrgID, mock.Anything).
		Return(revokeOut, nil)

	handler := newConsentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /consents/{consentId}/revoke", handler.revokeConsent)
	server := httptest.NewServer(mux)
	defer server.Close()

	reqBody := model.ConsentRevokeRequest{ActionBy: "admin-user", RevocationReason: "user request"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/consents/%s/revoke", server.URL, handlerTestConsentID), bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var revokeResp model.ConsentRevokeResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&revokeResp))
	require.Equal(t, "admin-user", revokeResp.ActionBy)
}

func TestHandlerRevokeConsent_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	handler := newConsentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /consents/{consentId}/revoke", handler.revokeConsent)
	server := httptest.NewServer(mux)
	defer server.Close()

	reqBody := model.ConsentRevokeRequest{ActionBy: "admin-user"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/consents/%s/revoke", server.URL, handlerTestConsentID), bytes.NewBuffer(body))
	// No org-id header

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockSvc.AssertNotCalled(t, "RevokeConsent")
}

func TestHandlerRevokeConsent_InvalidJSON(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	handler := newConsentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /consents/{consentId}/revoke", handler.revokeConsent)
	server := httptest.NewServer(mux)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/consents/%s/revoke", server.URL, handlerTestConsentID), bytes.NewBufferString("{invalid"))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockSvc.AssertNotCalled(t, "RevokeConsent")
}

func TestHandlerRevokeConsent_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	svcErr := &serviceerror.ServiceError{
		Type:    serviceerror.ClientErrorType,
		Code:    "CS-4041",
		Message: "consent already revoked",
	}
	mockSvc.On("RevokeConsent", mock.Anything, handlerTestConsentID, handlerTestOrgID, mock.Anything).
		Return(nil, svcErr)

	handler := newConsentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /consents/{consentId}/revoke", handler.revokeConsent)
	server := httptest.NewServer(mux)
	defer server.Close()

	reqBody := model.ConsentRevokeRequest{ActionBy: "admin-user"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/consents/%s/revoke", server.URL, handlerTestConsentID), bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

// =============================================================================
// validateConsent
// =============================================================================

func TestHandlerValidateConsent_Success_IsValid(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	validateOut := &model.ConsentValidateOutput{
		IsValid: true,
	}
	mockSvc.On("ValidateConsent", mock.Anything, mock.Anything, handlerTestOrgID).
		Return(validateOut, nil)

	handler := newConsentHandler(mockSvc)
	reqBody := model.ConsentValidateRequest{
		ConsentID: handlerTestConsentID,
		GroupID:   handlerTestGroupID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.ConsentValidateResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.True(t, resp.IsValid)
}

func TestHandlerValidateConsent_Success_IsInvalid(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	validateOut := &model.ConsentValidateOutput{
		IsValid:          false,
		ErrorCode:        1001,
		ErrorMessage:     "consent expired",
		ErrorDescription: "the consent has expired",
	}
	mockSvc.On("ValidateConsent", mock.Anything, mock.Anything, handlerTestOrgID).
		Return(validateOut, nil)

	handler := newConsentHandler(mockSvc)
	reqBody := model.ConsentValidateRequest{
		ConsentID: handlerTestConsentID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	// validate always returns 200; validity is in the body
	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.ConsentValidateResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.False(t, resp.IsValid)
}

func TestHandlerValidateConsent_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	reqBody := model.ConsentValidateRequest{ConsentID: handlerTestConsentID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "ValidateConsent")
}

func TestHandlerValidateConsent_InvalidJSON(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/consents/validate", bytes.NewBufferString("{invalid"))
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.validateConsent(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "ValidateConsent")
}

// =============================================================================
// searchConsentsByAttribute
// =============================================================================

func TestHandlerSearchConsentsByAttribute_SuccessWithKeyAndValue(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	attrOut := &model.ConsentAttributeSearchOutput{
		ConsentIDs: []string{handlerTestConsentID},
		Count:      1,
	}
	mockSvc.On("SearchConsentsByAttribute", mock.Anything, "purpose", "marketing", handlerTestOrgID).
		Return(attrOut, nil)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents/attributes?key=purpose&value=marketing", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.searchConsentsByAttribute(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.ConsentAttributeSearchResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, 1, resp.Count)
	require.Equal(t, handlerTestConsentID, resp.ConsentIDs[0])
}

func TestHandlerSearchConsentsByAttribute_SuccessWithKeyOnly(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	attrOut := &model.ConsentAttributeSearchOutput{
		ConsentIDs: []string{handlerTestConsentID, "consent-other"},
		Count:      2,
	}
	mockSvc.On("SearchConsentsByAttribute", mock.Anything, "purpose", "", handlerTestOrgID).
		Return(attrOut, nil)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents/attributes?key=purpose", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.searchConsentsByAttribute(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.ConsentAttributeSearchResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, 2, resp.Count)
}

func TestHandlerSearchConsentsByAttribute_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents/attributes?key=purpose", nil)
	rr := httptest.NewRecorder()

	handler.searchConsentsByAttribute(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "SearchConsentsByAttribute")
}

func TestHandlerSearchConsentsByAttribute_MissingKey(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents/attributes", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.searchConsentsByAttribute(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "SearchConsentsByAttribute")
}

// =============================================================================
// getGroupIDsByUserID
// =============================================================================

func TestHandlerGetGroupIDsByUserID_Success(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	groupOut := &model.ConsentGroupIDsOutput{
		GroupIDs: []string{"group-001", "group-002"},
		Count:    2,
	}
	mockSvc.On("GetGroupIDsByUserID", mock.Anything, "user-001", handlerTestOrgID).
		Return(groupOut, nil)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents/group-ids?userId=user-001", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.getGroupIDsByUserID(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.ConsentGroupIDsResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, 2, resp.Count)
	require.Equal(t, []string{"group-001", "group-002"}, resp.GroupIDs)
}

func TestHandlerGetGroupIDsByUserID_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents/group-ids?userId=user-001", nil)
	rr := httptest.NewRecorder()

	handler.getGroupIDsByUserID(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "GetGroupIDsByUserID")
}

func TestHandlerGetGroupIDsByUserID_MissingUserID(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents/group-ids", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.getGroupIDsByUserID(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "GetGroupIDsByUserID")
}

func TestHandlerGetGroupIDsByUserID_MultipleUserIDs(t *testing.T) {
	mockSvc := NewMockConsentService(t)
	handler := newConsentHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consents/group-ids?userId=user-001&userId=user-002", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.getGroupIDsByUserID(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "GetGroupIDsByUserID")
}

func TestHandlerGetGroupIDsByUserID_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentService(t)

	svcErr := &serviceerror.ServiceError{
		Type:    serviceerror.ServerErrorType,
		Code:    "CS-5000",
		Message: "internal server error",
	}
	mockSvc.On("GetGroupIDsByUserID", mock.Anything, "user-001", handlerTestOrgID).Return(nil, svcErr)

	handler := newConsentHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consents/group-ids?userId=user-001", nil)
	req.Header.Set(constants.HeaderOrgID, handlerTestOrgID)
	rr := httptest.NewRecorder()

	handler.getGroupIDsByUserID(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

// =============================================================================
// toExpirationMillis
// =============================================================================

func TestHandlerToExpirationMillis_Nil(t *testing.T) {
	result := toExpirationMillis(nil)
	require.Nil(t, result)
}

func TestHandlerToExpirationMillis_AlreadyMilliseconds(t *testing.T) {
	ms := int64(1_700_000_000_000) // already >= 100_000_000_000
	result := toExpirationMillis(&ms)
	require.NotNil(t, result)
	require.Equal(t, int64(1_700_000_000_000), *result)
}

func TestHandlerToExpirationMillis_Seconds_ConvertedToMilliseconds(t *testing.T) {
	secs := int64(1_000_000) // < 100_000_000_000 → treated as seconds
	result := toExpirationMillis(&secs)
	require.NotNil(t, result)
	require.Equal(t, int64(1_000_000_000), *result)
}

func TestHandlerToExpirationMillis_BoundaryValue(t *testing.T) {
	// Exactly at the cutoff (100_000_000_000) → already ms, unchanged
	v := int64(100_000_000_000)
	result := toExpirationMillis(&v)
	require.NotNil(t, result)
	require.Equal(t, int64(100_000_000_000), *result)
}

// =============================================================================
// valueStringToInterface
// =============================================================================

func TestValueStringToInterface_Nil(t *testing.T) {
	require.Nil(t, valueStringToInterface(nil, "basic"))
	require.Nil(t, valueStringToInterface(nil, "json"))
}

func TestValueStringToInterface_EmptyStringPreserved(t *testing.T) {
	empty := ""
	// Non-nil pointer to empty string must NOT be treated as absent.
	result := valueStringToInterface(&empty, "basic")
	require.NotNil(t, result)
	require.Equal(t, "", result)
}

func TestValueStringToInterface_NonEmptyBasic(t *testing.T) {
	v := "hello"
	require.Equal(t, "hello", valueStringToInterface(&v, "basic"))
}

func TestValueStringToInterface_JSONParsed(t *testing.T) {
	v := `{"key":"val"}`
	result := valueStringToInterface(&v, "json")
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "val", m["key"])
}

func TestValueStringToInterface_JSONInvalidFallsBackToString(t *testing.T) {
	v := "not-json"
	require.Equal(t, "not-json", valueStringToInterface(&v, "json"))
}

func TestValueStringToInterface_EmptyStringJSONFallsBackToString(t *testing.T) {
	// Empty string is invalid JSON — must fall back to returning the empty string, not nil.
	empty := ""
	result := valueStringToInterface(&empty, "json")
	require.NotNil(t, result)
	require.Equal(t, "", result)
}
