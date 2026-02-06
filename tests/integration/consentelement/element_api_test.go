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
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wso2/openfgc/tests/integration/testutils"
)

var (
	testServerURL = testutils.GetTestServerURL()
	baseURL       = testutils.GetTestServerURL()
)

const (
	testOrgID    = "test-org-element"
	testClientID = "test-client-element"
)

type ElementAPITestSuite struct {
	suite.Suite
	createdElementIDs []string // Track created elements for cleanup
}

// SetupSuite runs once before all tests
func (ts *ElementAPITestSuite) SetupSuite() {
	ts.createdElementIDs = make([]string, 0)
	ts.T().Logf("=== ConsentElement Test Suite Starting ===")
}

// TearDownSuite runs once after all tests to cleanup
func (ts *ElementAPITestSuite) TearDownSuite() {
	if len(ts.createdElementIDs) == 0 {
		ts.T().Logf("=== No elements to clean up ===")
		return
	}

	ts.T().Logf("=== Cleaning up %d created elements ===", len(ts.createdElementIDs))
	successCount := 0
	failCount := 0

	for _, id := range ts.createdElementIDs {
		if ts.deleteElementWithCheck(id) {
			successCount++
		} else {
			failCount++
		}
	}

	ts.T().Logf("=== Cleanup complete: %d deleted, %d failed ===", successCount, failCount)
	ts.T().Logf("=== ConsentElement Test Suite Complete ===")
}

// TearDownTest runs after each test to ensure cleanup
func (ts *ElementAPITestSuite) TearDownTest() {
	// Clean up elements created in this test
	if len(ts.createdElementIDs) == 0 {
		return
	}

	for _, id := range ts.createdElementIDs {
		ts.deleteElementWithCheck(id)
	}

	// Clear the list for next test
	ts.createdElementIDs = make([]string, 0)
}

func TestElementAPITestSuite(t *testing.T) {
	suite.Run(t, new(ElementAPITestSuite))
}

// Helper functions

// createElement creates element(s) and returns the response and body for flexible assertions
func (ts *ElementAPITestSuite) createElement(payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	// Handle both []ConsentElementCreateRequest and string (for malformed JSON tests)
	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-elements",
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// getElement retrieves an element by ID and returns response and body
func (ts *ElementAPITestSuite) getElement(elementID string) (*http.Response, []byte) {
	url := fmt.Sprintf("%s/api/v1/consent-elements/%s", testServerURL, elementID)
	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// updateElement updates an element by ID and returns response and body
func (ts *ElementAPITestSuite) updateElement(elementID string, payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	url := fmt.Sprintf("%s/api/v1/consent-elements/%s", testServerURL, elementID)
	httpReq, _ := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// trackElement registers an element ID for cleanup in TearDownSuite
func (ts *ElementAPITestSuite) trackElement(elementID string) {
	ts.createdElementIDs = append(ts.createdElementIDs, elementID)
}

// deleteElementWithCheck deletes an element and returns success status
// Returns true only for successful deletion (204/200), false for 404 or other errors
func (ts *ElementAPITestSuite) deleteElementWithCheck(elementID string) bool {
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-elements/%s", testServerURL, elementID),
		nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.T().Logf("Warning: failed to delete element %s: %v", elementID, err)
		return false
	}
	defer resp.Body.Close()

	// Return true only for successful deletion (204 or 200)
	// Return false for 404 (not found) or any other status
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return true
	}

	return false
}

// deleteElementResponse deletes an element and returns the response for assertion
func (ts *ElementAPITestSuite) deleteElementResponse(elementID string) (*http.Response, []byte) {
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-elements/%s", testServerURL, elementID),
		nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, body
}
