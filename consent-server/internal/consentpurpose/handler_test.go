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

	"github.com/wso2/openfgc/consent-server/internal/consentpurpose/model"
	"github.com/wso2/openfgc/consent-server/internal/system/constants"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
)

const (
	testOrgID     = "test-org-123"
	testPurposeID = "purpose-123"
)

func strPtr(s string) *string { return &s }

// =============================================================================
// createPurpose
// =============================================================================

func TestCreatePurpose_Success(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	pv := &model.PurposeOutput{
		ID:         testPurposeID,
		Name:       "Test Purpose",
		GroupID:    testOrgID,
		VersionNum: 1,
	}
	mockSvc.On("CreatePurpose", mock.Anything, mock.Anything, testOrgID).
		Return(pv, nil)

	handler := newConsentPurposeHandler(mockSvc)
	body, _ := json.Marshal(model.CreatePurposeRequest{Name: "Test Purpose"})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Contains(t, rr.Header().Get(constants.HeaderContentType), "application/json")

	var resp model.PurposeResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, testPurposeID, resp.PurposeID)
	require.Equal(t, "Test Purpose", resp.Name)
	require.Equal(t, "v1", resp.Version)
}

func TestCreatePurpose_GroupIDFromHeader(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	pv := &model.PurposeOutput{ID: testPurposeID, Name: "P", GroupID: "grp-1", VersionNum: 1}
	mockSvc.On("CreatePurpose", mock.Anything,
		mock.MatchedBy(func(input model.CreatePurposeInput) bool {
			return input.GroupID == "grp-1"
		}),
		testOrgID).
		Return(pv, nil)

	handler := newConsentPurposeHandler(mockSvc)
	body, _ := json.Marshal(model.CreatePurposeRequest{Name: "P"})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.Header.Set(constants.HeaderGroupID, "grp-1")
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	mockSvc.AssertExpectations(t)
}

func TestCreatePurpose_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	body, _ := json.Marshal(model.CreatePurposeRequest{Name: "Test"})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreatePurpose")
}

func TestCreatePurpose_InvalidJSON(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBufferString("{invalid"))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreatePurpose")
}

func TestCreatePurpose_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	svcErr := &serviceerror.ServiceError{
		Type: serviceerror.ClientErrorType, Code: "CP-4001", Message: "validation failed",
	}
	mockSvc.On("CreatePurpose", mock.Anything, mock.Anything, testOrgID).
		Return(nil, svcErr)

	handler := newConsentPurposeHandler(mockSvc)
	body, _ := json.Marshal(model.CreatePurposeRequest{Name: "Test"})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// =============================================================================
// getPurpose
// =============================================================================

func TestGetPurpose_Success(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	pv := &model.PurposeOutput{
		ID: testPurposeID, Name: "My Purpose", GroupID: testOrgID, VersionNum: 2,
	}
	mockSvc.On("GetPurpose", mock.Anything, testPurposeID, testOrgID).Return(pv, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.getPurpose(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.PurposeResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, testPurposeID, resp.PurposeID)
	require.Equal(t, "v2", resp.Version)
}

func TestGetPurpose_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID, nil)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.getPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "GetPurpose")
}

func TestGetPurpose_NotFound(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	mockSvc.On("GetPurpose", mock.Anything, testPurposeID, testOrgID).
		Return(nil, &ErrorPurposeNotFound)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.getPurpose(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// =============================================================================
// listPurposes
// =============================================================================

func TestListPurposes_Success(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	out := &model.PurposeListOutput{
		Data: []model.PurposeOutput{
			{ID: "p-1", Name: "Purpose 1", GroupID: testOrgID, VersionNum: 1},
			{ID: "p-2", Name: "Purpose 2", GroupID: testOrgID, VersionNum: 1},
		},
		Total: 2, Count: 2, Limit: 100, Offset: 0,
	}
	mockSvc.On("ListPurposes", mock.Anything, testOrgID, mock.Anything).Return(out, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.PurposeListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Data, 2)
	require.Equal(t, 2, resp.Metadata.Total)
	require.Equal(t, 100, resp.Metadata.Limit)
}

func TestListPurposes_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes", nil)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "ListPurposes")
}

func TestListPurposes_PurposeVersionWithoutPurposeName(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes?purposeVersion=1", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "ListPurposes")
}

func TestListPurposes_PurposeVersionWithPurposeName(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	out := &model.PurposeListOutput{Total: 0, Count: 0, Limit: 100, Offset: 0}
	mockSvc.On("ListPurposes", mock.Anything, testOrgID, mock.Anything).Return(out, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes?purposeVersion=1&purposeName=Marketing", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestListPurposes_ElementVersionWithoutElementNameOrNamespace(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes?elementVersion=2", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "ListPurposes")
}

func TestListPurposes_ElementVersionWithElementName(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	out := &model.PurposeListOutput{Total: 0, Count: 0, Limit: 100, Offset: 0}
	mockSvc.On("ListPurposes", mock.Anything, testOrgID, mock.Anything).Return(out, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes?elementVersion=2&elementName=user_email", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestListPurposes_ElementVersionWithElementNamespace(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	out := &model.PurposeListOutput{Total: 0, Count: 0, Limit: 100, Offset: 0}
	mockSvc.On("ListPurposes", mock.Anything, testOrgID, mock.Anything).Return(out, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes?elementVersion=2&elementNamespace=billing", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestListPurposes_InvalidPagination(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	out := &model.PurposeListOutput{Total: 0, Count: 0, Limit: 100, Offset: 0}
	mockSvc.On("ListPurposes", mock.Anything, testOrgID, mock.Anything).Return(out, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes?limit=bad&offset=bad", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	// Falls back to defaults — still returns 200
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestListPurposes_InvalidPurposeVersion(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	for _, bad := range []string{"vbad", "v0", "v-1", "abc"} {
		req := httptest.NewRequest(http.MethodGet, "/consent-purposes?purposeVersion="+bad, nil)
		req.Header.Set(constants.HeaderOrgID, testOrgID)
		rr := httptest.NewRecorder()
		handler.listPurposes(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code, "purposeVersion=%q should return 400", bad)
	}
}

func TestListPurposes_InvalidElementVersion(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	for _, bad := range []string{"vbad", "v0", "v-1", "abc"} {
		req := httptest.NewRequest(http.MethodGet, "/consent-purposes?elementVersion="+bad, nil)
		req.Header.Set(constants.HeaderOrgID, testOrgID)
		rr := httptest.NewRecorder()
		handler.listPurposes(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code, "elementVersion=%q should return 400", bad)
	}
}

// =============================================================================
// listPurposeVersions
// =============================================================================

func TestListPurposeVersions_Success(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	out := &model.PurposeVersionListOutput{
		PurposeID: testPurposeID, Name: "My Purpose", GroupID: testOrgID,
		Versions: []model.PurposeOutput{
			{ID: testPurposeID, VersionNum: 1},
			{ID: testPurposeID, VersionNum: 2},
		},
	}
	mockSvc.On("GetPurposeVersions", mock.Anything, testPurposeID, testOrgID).Return(out, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID+"/versions", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.listPurposeVersions(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.PurposeVersionListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, testPurposeID, resp.PurposeID)
	require.Len(t, resp.Versions, 2)
	require.Equal(t, "v1", resp.Versions[0].Version)
	require.Equal(t, "v2", resp.Versions[1].Version)
}

// =============================================================================
// createPurposeVersion
// =============================================================================

func TestCreatePurposeVersion_Success(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	pv := &model.PurposeOutput{ID: testPurposeID, Name: "My Purpose", GroupID: testOrgID, VersionNum: 2}
	mockSvc.On("CreatePurposeVersion", mock.Anything, testPurposeID, mock.Anything, testOrgID).
		Return(pv, nil)

	handler := newConsentPurposeHandler(mockSvc)
	body, _ := json.Marshal(model.CreatePurposeVersionRequest{Description: strPtr("v2 description")})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes/"+testPurposeID+"/versions", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.createPurposeVersion(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp model.PurposeResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, "v2", resp.Version)
}

// =============================================================================
// getPurposeVersion
// =============================================================================

func TestGetPurposeVersion_Success(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	pv := &model.PurposeOutput{ID: testPurposeID, Name: "My Purpose", GroupID: testOrgID, VersionNum: 1}
	mockSvc.On("GetPurposeVersion", mock.Anything, testPurposeID, 1, testOrgID).Return(pv, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID+"/versions/v1", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	req.SetPathValue("version", "v1")
	rr := httptest.NewRecorder()

	handler.getPurposeVersion(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp model.PurposeResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, "v1", resp.Version)
}

func TestGetPurposeVersion_InvalidFormat(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID+"/versions/1", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	req.SetPathValue("version", "1") // missing "v" prefix
	rr := httptest.NewRecorder()

	handler.getPurposeVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "GetPurposeVersion")
}

// =============================================================================
// deletePurposeVersion
// =============================================================================

func TestDeletePurposeVersion_Success(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	mockSvc.On("DeletePurposeVersion", mock.Anything, testPurposeID, 1, testOrgID).Return(nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID+"/versions/v1", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	req.SetPathValue("version", "v1")
	rr := httptest.NewRecorder()

	handler.deletePurposeVersion(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
}

// =============================================================================
// deletePurpose
// =============================================================================

func TestDeletePurpose_Success(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	mockSvc.On("DeletePurpose", mock.Anything, testPurposeID, testOrgID).Return(nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.deletePurpose(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Empty(t, rr.Body.String())
}

func TestDeletePurpose_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID, nil)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.deletePurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "DeletePurpose")
}

func TestDeletePurpose_NotFound(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	mockSvc.On("DeletePurpose", mock.Anything, testPurposeID, testOrgID).
		Return(&ErrorPurposeNotFound)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.deletePurpose(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// =============================================================================
// listPurposes — remaining gaps
// =============================================================================

func TestListPurposes_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	svcErr := &serviceerror.ServiceError{Type: serviceerror.ServerErrorType, Code: "CP-5004", Message: "list failed"}
	mockSvc.On("ListPurposes", mock.Anything, testOrgID, mock.Anything).Return(nil, svcErr)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestListPurposes_GroupIdsAndPagination(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	out := &model.PurposeListOutput{Total: 1, Count: 1, Limit: 10, Offset: 5}
	mockSvc.On("ListPurposes", mock.Anything, testOrgID, mock.Anything).Return(out, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes?groupIds=g1,g2&limit=10&offset=5", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.listPurposes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

// =============================================================================
// listPurposeVersions — remaining gaps
// =============================================================================

func TestListPurposeVersions_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID+"/versions", nil)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.listPurposeVersions(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "GetPurposeVersions")
}

func TestListPurposeVersions_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	svcErr := &serviceerror.ServiceError{Type: serviceerror.ClientErrorType, Code: "CP-4040", Message: "not found"}
	mockSvc.On("GetPurposeVersions", mock.Anything, testPurposeID, testOrgID).Return(nil, svcErr)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID+"/versions", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.listPurposeVersions(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// =============================================================================
// createPurposeVersion — remaining gaps
// =============================================================================

func TestCreatePurposeVersion_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	body, _ := json.Marshal(model.CreatePurposeVersionRequest{})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes/"+testPurposeID+"/versions", bytes.NewBuffer(body))
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.createPurposeVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreatePurposeVersion")
}

func TestCreatePurposeVersion_InvalidJSON(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/consent-purposes/"+testPurposeID+"/versions", bytes.NewBufferString("{invalid"))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.createPurposeVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreatePurposeVersion")
}

func TestCreatePurposeVersion_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	svcErr := &serviceerror.ServiceError{Type: serviceerror.ClientErrorType, Code: "CP-4040", Message: "not found"}
	mockSvc.On("CreatePurposeVersion", mock.Anything, testPurposeID, mock.Anything, testOrgID).Return(nil, svcErr)

	handler := newConsentPurposeHandler(mockSvc)
	body, _ := json.Marshal(model.CreatePurposeVersionRequest{})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes/"+testPurposeID+"/versions", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.createPurposeVersion(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// =============================================================================
// getPurposeVersion — remaining gaps
// =============================================================================

func TestGetPurposeVersion_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID+"/versions/v1", nil)
	req.SetPathValue("purposeId", testPurposeID)
	req.SetPathValue("version", "v1")
	rr := httptest.NewRecorder()

	handler.getPurposeVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "GetPurposeVersion")
}

func TestGetPurposeVersion_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	svcErr := &serviceerror.ServiceError{Type: serviceerror.ClientErrorType, Code: "CP-4040", Message: "not found"}
	mockSvc.On("GetPurposeVersion", mock.Anything, testPurposeID, 1, testOrgID).Return(nil, svcErr)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID+"/versions/v1", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	req.SetPathValue("version", "v1")
	rr := httptest.NewRecorder()

	handler.getPurposeVersion(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// =============================================================================
// deletePurposeVersion — remaining gaps
// =============================================================================

func TestDeletePurposeVersion_MissingOrgID(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID+"/versions/v1", nil)
	req.SetPathValue("purposeId", testPurposeID)
	req.SetPathValue("version", "v1")
	rr := httptest.NewRecorder()

	handler.deletePurposeVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "DeletePurposeVersion")
}

func TestDeletePurposeVersion_ZeroVersion(t *testing.T) {
	// "v0" is syntactically valid but semantically invalid (n < 1).
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID+"/versions/v0", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	req.SetPathValue("version", "v0")
	rr := httptest.NewRecorder()

	handler.deletePurposeVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "DeletePurposeVersion")
}

func TestDeletePurposeVersion_ServiceError(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	svcErr := &serviceerror.ServiceError{Type: serviceerror.ClientErrorType, Code: "CP-4091", Message: "version in use"}
	mockSvc.On("DeletePurposeVersion", mock.Anything, testPurposeID, 1, testOrgID).Return(svcErr)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodDelete, "/consent-purposes/"+testPurposeID+"/versions/v1", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	req.SetPathValue("version", "v1")
	rr := httptest.NewRecorder()

	handler.deletePurposeVersion(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)
}

// =============================================================================
// purposeToResponse / purposeToItem — element-populated paths
// =============================================================================

func TestGetPurpose_WithElements(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	pv := &model.PurposeOutput{
		ID: testPurposeID, Name: "Marketing", GroupID: testOrgID, VersionNum: 1,
		Elements: []model.PurposeElementOutput{
			{ElementID: "eid-1", Name: "email", Namespace: "default", VersionNum: 2, Mandatory: true},
		},
	}
	mockSvc.On("GetPurpose", mock.Anything, testPurposeID, testOrgID).Return(pv, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID, nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.getPurpose(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.PurposeResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Elements, 1)
	require.Equal(t, "email", resp.Elements[0].Name)
	require.Equal(t, "v2", resp.Elements[0].Version)
	require.True(t, resp.Elements[0].Mandatory)
}

func TestListPurposeVersions_WithElements(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	out := &model.PurposeVersionListOutput{
		PurposeID: testPurposeID, Name: "Marketing", GroupID: testOrgID,
		Versions: []model.PurposeOutput{
			{
				ID: testPurposeID, VersionNum: 1,
				Elements: []model.PurposeElementOutput{
					{ElementID: "eid-1", Name: "email", Namespace: "default", VersionNum: 1, Mandatory: false},
				},
			},
		},
	}
	mockSvc.On("GetPurposeVersions", mock.Anything, testPurposeID, testOrgID).Return(out, nil)

	handler := newConsentPurposeHandler(mockSvc)
	req := httptest.NewRequest(http.MethodGet, "/consent-purposes/"+testPurposeID+"/versions", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	req.SetPathValue("purposeId", testPurposeID)
	rr := httptest.NewRecorder()

	handler.listPurposeVersions(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.PurposeVersionListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Versions[0].Elements, 1)
	require.Equal(t, "email", resp.Versions[0].Elements[0].Name)
}

// =============================================================================
// toElementRefs — loop body via createPurpose with elements in body
// =============================================================================

func TestCreatePurpose_WithElements(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)

	pv := &model.PurposeOutput{ID: testPurposeID, Name: "Marketing", GroupID: testOrgID, VersionNum: 1}
	mockSvc.On("CreatePurpose", mock.Anything, mock.Anything, testOrgID).Return(pv, nil)

	handler := newConsentPurposeHandler(mockSvc)
	v := "v1"
	body, _ := json.Marshal(model.CreatePurposeRequest{
		Name: "Marketing",
		Elements: []model.ElementRefRequest{
			{Name: "email", Namespace: "default", Version: &v, Mandatory: true},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
}

// parseVersionString nil branch: element without a version field → handler passes it through as nil ref.
func TestCreatePurpose_ElementWithNilVersion(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	pv := &model.PurposeOutput{ID: testPurposeID, Name: "Marketing", GroupID: testOrgID, VersionNum: 1}
	mockSvc.On("CreatePurpose", mock.Anything, mock.Anything, testOrgID).Return(pv, nil)

	handler := newConsentPurposeHandler(mockSvc)
	body, _ := json.Marshal(model.CreatePurposeRequest{
		Name: "Marketing",
		Elements: []model.ElementRefRequest{
			{Name: "email", Mandatory: true}, // Version omitted → nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
}

// parseVersionString format error: "abc" has no leading "v".
func TestCreatePurpose_InvalidElementVersionString(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	bad := "abc"
	body, _ := json.Marshal(model.CreatePurposeRequest{
		Name: "Marketing",
		Elements: []model.ElementRefRequest{
			{Name: "email", Version: &bad},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreatePurpose")
}

// parseVersionString n<1 error: "v0" has valid prefix but version number is zero.
func TestCreatePurpose_ElementVersionZero(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	v0 := "v0"
	body, _ := json.Marshal(model.CreatePurposeRequest{
		Name: "Marketing",
		Elements: []model.ElementRefRequest{
			{Name: "email", Version: &v0},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurpose(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreatePurpose")
}

// Same invalid version path for createPurposeVersion.
func TestCreatePurposeVersion_InvalidElementVersionString(t *testing.T) {
	mockSvc := NewMockConsentPurposeService(t)
	handler := newConsentPurposeHandler(mockSvc)

	bad := "bad"
	body, _ := json.Marshal(model.CreatePurposeVersionRequest{
		Elements: []model.ElementRefRequest{
			{Name: "email", Version: &bad},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/consent-purposes/"+testPurposeID+"/versions", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	handler.createPurposeVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockSvc.AssertNotCalled(t, "CreatePurposeVersion")
}
