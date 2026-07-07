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

	"github.com/wso2/openfgc/consent-server/internal/consentelement/model"
	"github.com/wso2/openfgc/consent-server/internal/system/constants"
)

const (
	testOrgID     = "test-org-123"
	testElementID = "elem-123"
)

func stringPtr(s string) *string { return &s }

// --- createElements ---

func TestCreateElement_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	output := &model.BatchCreateOutput{
		Results: []model.CreateElementOutput{
			{Status: "SUCCESS", Element: &model.ElementVersion{ID: testElementID, Name: "email", Type: "basic", VersionNum: 1}},
		},
	}
	mockService.On("CreateElementsInBatch", mock.Anything, mock.Anything, testOrgID).Return(output, nil)

	body, _ := json.Marshal([]model.CreateElementRequest{{Name: "email", Type: "basic"}})
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElements(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.BatchCreateResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Results, 1)
	require.Equal(t, "SUCCESS", resp.Results[0].Status)
}

func TestCreateElement_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	body, _ := json.Marshal([]model.CreateElementRequest{{Name: "x", Type: "basic"}})
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElements(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	mockService.AssertNotCalled(t, "CreateElementsInBatch")
}

func TestCreateElement_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBufferString("{bad"))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElements(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateElement_EmptyArray(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	body, _ := json.Marshal([]model.CreateElementRequest{})
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElements(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateElement_ServiceError(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	mockService.On("CreateElementsInBatch", mock.Anything, mock.Anything, testOrgID).Return(nil, &ErrorCreateElement)

	body, _ := json.Marshal([]model.CreateElementRequest{{Name: "email", Type: "basic"}})
	req := httptest.NewRequest(http.MethodPost, "/consent-elements", bytes.NewBuffer(body))
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElements(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- getElement ---

func TestGetElement_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	expected := &model.ElementVersion{ID: testElementID, Name: "email", Type: "basic", VersionNum: 1}
	mockService.On("GetElement", mock.Anything, testElementID, testOrgID).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).getElement(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.ElementResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, testElementID, resp.ElementID)
	require.Equal(t, "v1", resp.Version)
}

func TestGetElement_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).getElement(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetElement_NotFound(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	mockService.On("GetElement", mock.Anything, testElementID, testOrgID).Return(nil, &ErrorElementNotFound)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID, nil)
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).getElement(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// --- listElements ---

func TestListElements_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	output := &model.ElementListOutput{
		Data:  []model.ElementVersion{{ID: "e1", Name: "email", VersionNum: 1}, {ID: "e2", Name: "age", VersionNum: 1}},
		Total: 2,
	}
	mockService.On("ListElements", mock.Anything, testOrgID, model.ElementListFilter{Limit: 100}).Return(output, nil)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).listElements(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.ElementListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Data, 2)
	require.Equal(t, 2, resp.Metadata.Total)
}

func TestListElements_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	req := httptest.NewRequest(http.MethodGet, "/consent-elements", nil)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).listElements(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListElements_ServiceError(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	mockService.On("ListElements", mock.Anything, testOrgID, model.ElementListFilter{Limit: 100}).Return(nil, &ErrorReadElement)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).listElements(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestListElements_WithFilters(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	v := 2
	expected := model.ElementListFilter{Name: "em", Namespace: "ns1", Type: "basic", Version: &v, Details: true, Limit: 10, Offset: 5}
	mockService.On("ListElements", mock.Anything, testOrgID, expected).Return(&model.ElementListOutput{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements?name=em&namespace=ns1&type=basic&version=2&details=true&limit=10&offset=5", nil)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).listElements(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

// --- listElementVersions ---

func TestListElementVersions_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	output := &model.ElementVersionListOutput{
		ElementID: testElementID,
		Versions: []model.ElementVersion{
			{ID: testElementID, VersionNum: 1},
			{ID: testElementID, VersionNum: 2},
		},
	}
	mockService.On("ListElementVersions", mock.Anything, testElementID, testOrgID).Return(output, nil)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID+"/versions", nil)
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).listElementVersions(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.ElementVersionListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, testElementID, resp.ElementID)
	require.Len(t, resp.Versions, 2)
	require.Equal(t, "v1", resp.Versions[0].Version)
	require.Equal(t, "v2", resp.Versions[1].Version)
}

func TestListElementVersions_NotFound(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	mockService.On("ListElementVersions", mock.Anything, testElementID, testOrgID).Return(nil, &ErrorElementNotFound)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID+"/versions", nil)
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).listElementVersions(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// --- createElementVersion ---

func TestCreateElementVersion_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	input := model.CreateElementVersionInput{DisplayName: stringPtr("v2")}
	created := &model.ElementVersion{ID: testElementID, VersionNum: 2, DisplayName: stringPtr("v2")}
	mockService.On("CreateElementVersion", mock.Anything, testElementID, input, testOrgID).Return(created, nil)

	body, _ := json.Marshal(model.CreateElementVersionRequest{DisplayName: stringPtr("v2")})
	req := httptest.NewRequest(http.MethodPost, "/consent-elements/"+testElementID+"/versions", bytes.NewBuffer(body))
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElementVersion(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.ElementResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, "v2", resp.Version)
}

func TestCreateElementVersion_InvalidJSON(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	req := httptest.NewRequest(http.MethodPost, "/consent-elements/"+testElementID+"/versions", bytes.NewBufferString("{bad"))
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElementVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateElementVersion_ServiceError(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	input := model.CreateElementVersionInput{}
	mockService.On("CreateElementVersion", mock.Anything, testElementID, input, testOrgID).Return(nil, &ErrorCreateElement)

	body, _ := json.Marshal(model.CreateElementVersionRequest{})
	req := httptest.NewRequest(http.MethodPost, "/consent-elements/"+testElementID+"/versions", bytes.NewBuffer(body))
	req.SetPathValue("elementId", testElementID)
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElementVersion(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestCreateElementVersion_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	body, _ := json.Marshal(model.CreateElementVersionRequest{})
	req := httptest.NewRequest(http.MethodPost, "/consent-elements/"+testElementID+"/versions", bytes.NewBuffer(body))
	req.SetPathValue("elementId", testElementID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).createElementVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// --- getElementVersion ---

func TestGetElementVersion_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)

	expected := &model.ElementVersion{ID: testElementID, VersionNum: 2}
	mockService.On("GetElementVersion", mock.Anything, testElementID, 2, testOrgID).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID+"/versions/v2", nil)
	req.SetPathValue("elementId", testElementID)
	req.SetPathValue("version", "v2")
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).getElementVersion(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.ElementResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Equal(t, "v2", resp.Version)
}

func TestGetElementVersion_InvalidVersion(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID+"/versions/abc", nil)
	req.SetPathValue("elementId", testElementID)
	req.SetPathValue("version", "abc")
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).getElementVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetElementVersion_NotFound(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	mockService.On("GetElementVersion", mock.Anything, testElementID, 99, testOrgID).Return(nil, &ErrorElementNotFound)

	req := httptest.NewRequest(http.MethodGet, "/consent-elements/"+testElementID+"/versions/v99", nil)
	req.SetPathValue("elementId", testElementID)
	req.SetPathValue("version", "v99")
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).getElementVersion(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

// --- deleteElementVersion ---

func TestDeleteElementVersion_Success(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	mockService.On("DeleteElementVersion", mock.Anything, testElementID, 1, testOrgID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/consent-elements/"+testElementID+"/versions/v1", nil)
	req.SetPathValue("elementId", testElementID)
	req.SetPathValue("version", "v1")
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).deleteElementVersion(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDeleteElementVersion_MissingOrgID(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	req := httptest.NewRequest(http.MethodDelete, "/consent-elements/"+testElementID+"/versions/v1", nil)
	req.SetPathValue("elementId", testElementID)
	req.SetPathValue("version", "v1")
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).deleteElementVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteElementVersion_InvalidVersion(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	req := httptest.NewRequest(http.MethodDelete, "/consent-elements/"+testElementID+"/versions/0", nil)
	req.SetPathValue("elementId", testElementID)
	req.SetPathValue("version", "0")
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).deleteElementVersion(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteElementVersion_ReferencedByPurpose(t *testing.T) {
	mockService := NewMockConsentElementService(t)
	mockService.On("DeleteElementVersion", mock.Anything, testElementID, 1, testOrgID).Return(&ErrorVersionReferencedByPurpose)

	req := httptest.NewRequest(http.MethodDelete, "/consent-elements/"+testElementID+"/versions/v1", nil)
	req.SetPathValue("elementId", testElementID)
	req.SetPathValue("version", "v1")
	req.Header.Set(constants.HeaderOrgID, testOrgID)
	rr := httptest.NewRecorder()

	newConsentElementHandler(mockService).deleteElementVersion(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)
}

// =============================================================================
// schemaToString
// =============================================================================

func TestSchemaToString(t *testing.T) {
	cases := []struct {
		name    string
		raw     json.RawMessage
		wantNil bool
		want    string
	}{
		{"nil/absent payload → nil", nil, true, ""},
		{"empty payload → nil", json.RawMessage{}, true, ""},
		{"JSON null → nil (not the literal string null)", json.RawMessage(`null`), true, ""},
		{"JSON null with surrounding whitespace → nil", json.RawMessage("  null  "), true, ""},
		{"JSON string → unwrapped value", json.RawMessage(`"my-schema"`), false, "my-schema"},
		{"JSON object → kept as-is", json.RawMessage(`{"type":"object"}`), false, `{"type":"object"}`},
		{"JSON array → kept as-is", json.RawMessage(`["a","b"]`), false, `["a","b"]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := schemaToString(tc.raw)
			if tc.wantNil {
				require.Nil(t, got, "expected nil but got %q", got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, tc.want, *got)
			}
		})
	}
}
