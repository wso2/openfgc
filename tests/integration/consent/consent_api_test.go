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
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wso2/openfgc/tests/integration/testutils"
)

var serverURL = testutils.GetTestServerURL()

// orgCounter drives freshOrgID — monotonically increasing within a test run.
var orgCounter atomic.Int64

// freshOrgID returns a unique org ID for each call.
// Using a fresh org per test means tests never share DB state and never need cleanup.
func freshOrgID() string {
	return fmt.Sprintf("test-cs-%d", orgCounter.Add(1))
}

// ptr converts a string to *string.
func ptr(s string) *string { return &s }

// intPtr converts an int to *int.
func intPtr(i int) *int { return &i }

// int64Ptr converts an int64 to *int64.
func int64Ptr(i int64) *int64 { return &i }

// boolPtr converts a bool to *bool.
func boolPtr(b bool) *bool { return &b }

// =============================================================================
// Suite
// =============================================================================

// ConsentAPITestSuite is the testify suite for all consent integration tests.
type ConsentAPITestSuite struct {
	suite.Suite
}

func TestConsentAPITestSuite(t *testing.T) {
	suite.Run(t, new(ConsentAPITestSuite))
}

func (ts *ConsentAPITestSuite) SetupSuite() {
	ts.T().Log("=== Consent Integration Test Suite Starting ===")
}

// =============================================================================
// Core HTTP helper
// =============================================================================

// doRequest executes an HTTP request and returns (statusCode, responseBody).
//
//   - orgID: written as the org-id header; pass "" to omit it entirely
//     (use this for missing-header error-case tests).
//   - groupID: written as the group-id header; pass "" to omit it entirely.
//   - body: nil for GET/DELETE; a struct (JSON-marshalled) or a raw string for POST/PUT.
func (ts *ConsentAPITestSuite) doRequest(method, path, orgID, groupID string, body any) (int, []byte) {
	var rawBody []byte
	if body != nil { //nolint:nestif
		if s, ok := body.(string); ok {
			rawBody = []byte(s)
		} else {
			var err error
			rawBody, err = json.Marshal(body)
			ts.Require().NoError(err, "marshal request body")
		}
	}

	req, err := http.NewRequest(method, serverURL+path, bytes.NewReader(rawBody))
	ts.Require().NoError(err)

	if orgID != "" {
		req.Header.Set(testutils.HeaderOrgID, orgID)
	}
	if groupID != "" {
		req.Header.Set("group-id", groupID)
	}
	if len(rawBody) > 0 {
		req.Header.Set(testutils.HeaderContentType, "application/json")
	}

	resp, err := testutils.GetHTTPClient().Do(req)
	ts.Require().NoError(err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	return resp.StatusCode, respBody
}

// =============================================================================
// Typed endpoint helpers
//
// Each typed helper returns (httpStatus, parsedResponse).
// The parsed response is nil when the status code does not match the expected
// success code — use doRequest directly to access the raw body in error cases.
// =============================================================================

// doCreateConsent handles POST /consents.
// groupID is the group-id header value; pass "" to omit it (triggers a validation error).
func (ts *ConsentAPITestSuite) doCreateConsent(orgID, groupID string, req ConsentCreateRequest) (int, *ConsentResponse) {
	status, body := ts.doRequest(http.MethodPost, "/api/v1/consents", orgID, groupID, req)
	if status != http.StatusCreated {
		return status, nil
	}
	var resp ConsentResponse
	ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentResponse: %s", body)
	return status, &resp
}

// doCreateConsentRaw sends any body (struct or raw string) and always returns the raw response.
// Use in table-driven tests where both success and error bodies need inspection.
func (ts *ConsentAPITestSuite) doCreateConsentRaw(orgID, groupID string, body any) (int, []byte) {
	return ts.doRequest(http.MethodPost, "/api/v1/consents", orgID, groupID, body)
}

func (ts *ConsentAPITestSuite) doGetConsent(orgID, consentID string) (int, *ConsentResponse) {
	status, body := ts.doRequest(http.MethodGet, "/api/v1/consents/"+consentID, orgID, "", nil)
	if status != http.StatusOK {
		return status, nil
	}
	var resp ConsentResponse
	ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentResponse (get)")
	return status, &resp
}

func (ts *ConsentAPITestSuite) doGetConsentWithStatusHistory(orgID, consentID string) (int, *ConsentResponse) {
	status, body := ts.doRequest(http.MethodGet, "/api/v1/consents/"+consentID+"?includeStatusHistory=true", orgID, "", nil)
	if status != http.StatusOK {
		return status, nil
	}
	var resp ConsentResponse
	ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentResponse (get with status history)")
	return status, &resp
}

func (ts *ConsentAPITestSuite) doGetConsentHistory(orgID, consentID string, includeSnapshots bool) (int, *ConsentHistoryListResponse) {
	path := "/api/v1/consents/" + consentID + "/history"
	if includeSnapshots {
		path += "?includeSnapshots=true"
	}
	status, body := ts.doRequest(http.MethodGet, path, orgID, "", nil)
	if status != http.StatusOK {
		return status, nil
	}
	var resp ConsentHistoryListResponse
	ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentHistoryListResponse")
	return status, &resp
}
func (ts *ConsentAPITestSuite) doListConsents(orgID string, params url.Values) (int, *ConsentListResponse) {
	path := "/api/v1/consents"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	status, body := ts.doRequest(http.MethodGet, path, orgID, "", nil)
	if status != http.StatusOK {
		return status, nil
	}
	var resp ConsentListResponse
	ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentListResponse")
	return status, &resp
}

// doSearchConsents calls GET /consents with the given query params, same endpoint as doListConsents
// but returns the raw body so callers can inspect it regardless of status code.
func (ts *ConsentAPITestSuite) doSearchConsents(orgID string, params url.Values) (int, []byte) {
	path := "/api/v1/consents"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	return ts.doRequest(http.MethodGet, path, orgID, "", nil)
}

// doSearchByAttribute calls GET /consents/attributes and returns (status, raw body).
func (ts *ConsentAPITestSuite) doSearchByAttribute(orgID, key, value string) (int, []byte) {
	params := url.Values{"key": {key}}
	if value != "" {
		params.Set("value", value)
	}
	return ts.doRequest(http.MethodGet, "/api/v1/consents/attributes?"+params.Encode(), orgID, "", nil)
}

// doGetGroupIDsByUserID calls GET /consents/group-ids and returns (status, raw body).
func (ts *ConsentAPITestSuite) doGetGroupIDsByUserID(orgID string, userIDs []string) (int, []byte) {
	params := url.Values{}
	for _, userID := range userIDs {
		params.Add("userId", userID)
	}

	path := "/api/v1/consents/group-ids"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return ts.doRequest(http.MethodGet, path, orgID, "", nil)
}

// doValidateConsent calls POST /consents/validate and returns (status, raw body).
// body can be a ConsentValidateRequest struct or a raw string for error cases.
func (ts *ConsentAPITestSuite) doValidateConsent(orgID string, body any) (int, []byte) {
	return ts.doRequest(http.MethodPost, "/api/v1/consents/validate", orgID, "", body)
}

// mustCreateElementFull creates a consent element using a full item map (name, type, schema, etc.)
// and returns its elementId. Use this when you need to set schema or other optional fields.
func (ts *ConsentAPITestSuite) mustCreateElementFull(orgID string, item map[string]any) string {
	body := []map[string]any{item}
	status, respBody := ts.doRequest(http.MethodPost, "/api/v1/consent-elements", orgID, "", body)
	ts.Require().Equal(http.StatusOK, status, "mustCreateElementFull: unexpected status for %v: %s", item["name"], respBody)

	var batchResp struct {
		Results []struct {
			Status  string `json:"status"`
			Element *struct {
				ElementID string `json:"elementId"`
			} `json:"element"`
			Error *string `json:"error"`
		} `json:"results"`
	}
	ts.Require().NoError(json.Unmarshal(respBody, &batchResp))
	ts.Require().Len(batchResp.Results, 1)
	ts.Require().Equal("SUCCESS", batchResp.Results[0].Status,
		"mustCreateElementFull: FAILED — error: %v", batchResp.Results[0].Error)
	ts.Require().NotNil(batchResp.Results[0].Element)
	return batchResp.Results[0].Element.ElementID
}

func (ts *ConsentAPITestSuite) doUpdateConsent(orgID, groupID, consentID string, req ConsentUpdateRequest) (int, *ConsentResponse) {
	status, body := ts.doRequest(http.MethodPut, "/api/v1/consents/"+consentID, orgID, groupID, req)
	if status != http.StatusOK {
		return status, nil
	}
	var resp ConsentResponse
	ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentResponse (update)")
	return status, &resp
}

func (ts *ConsentAPITestSuite) doRevokeConsent(orgID, consentID string, req ConsentRevokeRequest) (int, *ConsentRevokeResponse) {
	status, body := ts.doRequest(http.MethodPost, "/api/v1/consents/"+consentID+"/revoke", orgID, "", req)
	if status != http.StatusOK {
		return status, nil
	}
	var resp ConsentRevokeResponse
	ts.Require().NoError(json.Unmarshal(body, &resp), "unmarshal ConsentRevokeResponse")
	return status, &resp
}

// =============================================================================
// Must-helpers (test setup — use Require so the test stops on failure)
// =============================================================================

// mustCreateConsent creates a consent and returns it. Uses Require so the test
// fails immediately if the setup step fails.
func (ts *ConsentAPITestSuite) mustCreateConsent(orgID, groupID string, req ConsentCreateRequest) *ConsentResponse {
	status, resp := ts.doCreateConsent(orgID, groupID, req)
	ts.Require().Equal(http.StatusCreated, status, "mustCreateConsent: unexpected HTTP status")
	ts.Require().NotNil(resp)
	return resp
}

// mustCreateElement creates a consent element via the /consent-elements API and returns its elementId.
func (ts *ConsentAPITestSuite) mustCreateElement(orgID, name, elemType string) string {
	body := []map[string]any{{"name": name, "type": elemType}}
	status, respBody := ts.doRequest(http.MethodPost, "/api/v1/consent-elements", orgID, "", body)
	ts.Require().Equal(http.StatusOK, status, "mustCreateElement: unexpected status for '%s'", name)

	var batchResp struct {
		Results []struct {
			Status  string `json:"status"`
			Element *struct {
				ElementID string `json:"elementId"`
			} `json:"element"`
			Error *string `json:"error"`
		} `json:"results"`
	}
	ts.Require().NoError(json.Unmarshal(respBody, &batchResp))
	ts.Require().Len(batchResp.Results, 1)
	ts.Require().Equal("SUCCESS", batchResp.Results[0].Status,
		"mustCreateElement: FAILED — error: %v", batchResp.Results[0].Error)
	ts.Require().NotNil(batchResp.Results[0].Element)
	return batchResp.Results[0].Element.ElementID
}

// mustCreatePurpose creates a consent purpose at org level (no group-id header)
// and returns its purposeId. Org-level purposes are accessible to consents from any group.
func (ts *ConsentAPITestSuite) mustCreatePurpose(orgID, purposeName, elementName string) string {
	return ts.mustCreatePurposeWithGroup(orgID, "", purposeName, elementName)
}

// mustCreatePurposeWithGroup creates a consent purpose scoped to a specific group and returns
// its purposeId. Pass groupID="" to create an org-level purpose (groupId stored as orgId).
// Group-scoped purposes are only accessible to consents whose group-id matches.
func (ts *ConsentAPITestSuite) mustCreatePurposeWithGroup(orgID, groupID, purposeName, elementName string) string {
	body := map[string]any{
		"name":     purposeName,
		"elements": []map[string]any{{"name": elementName}},
	}
	status, respBody := ts.doRequest(http.MethodPost, "/api/v1/consent-purposes", orgID, groupID, body)
	ts.Require().Equal(http.StatusCreated, status,
		"mustCreatePurposeWithGroup: unexpected status for '%s' (group=%q): %s", purposeName, groupID, respBody)

	var resp struct {
		PurposeID string `json:"purposeId"`
	}
	ts.Require().NoError(json.Unmarshal(respBody, &resp))
	ts.Require().NotEmpty(resp.PurposeID)
	return resp.PurposeID
}

// =============================================================================
// Assertion helpers
// =============================================================================

// assertAPIError parses body as an ErrorResponse, asserts the error code, and
// returns the parsed struct for additional assertions.
func (ts *ConsentAPITestSuite) assertAPIError(body []byte, wantCode string) ErrorResponse {
	var errResp ErrorResponse
	ts.Require().NoError(json.Unmarshal(body, &errResp),
		"body is not a valid ErrorResponse: %s", string(body))
	ts.Require().Equal(wantCode, errResp.Code, "unexpected error code; body: %s", string(body))
	ts.Require().NotEmpty(errResp.Message, "error response must have a non-empty message")
	return errResp
}

// assertConsentResponse validates the fields that the API spec mandates are always
// present on a ConsentResponse. Call this from checkResult closures.
func (ts *ConsentAPITestSuite) assertConsentResponse(c *ConsentResponse, wantType, wantGroupID string) {
	ts.Require().NotNil(c)
	ts.Require().NotEmpty(c.ID, "id must not be empty")
	ts.Require().NotEmpty(c.Status, "status must not be empty")
	// 946684800000 = 2000-01-01 in Unix milliseconds — guards against Unix seconds.
	ts.Require().Greater(c.CreatedTime, int64(946684800000), "createdTime must be a Unix millisecond timestamp")
	ts.Require().Greater(c.UpdatedTime, int64(946684800000), "updatedTime must be a Unix millisecond timestamp")
	ts.Equal(wantType, c.Type, "type mismatch")
	ts.Equal(wantGroupID, c.GroupID, "groupId mismatch")
}
