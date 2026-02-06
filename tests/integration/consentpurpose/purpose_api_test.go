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
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wso2/openfgc/tests/integration/testutils"
)

var (
	testServerURL = testutils.GetTestServerURL()
)

const (
	testOrgID    = "test-org-purpose"
	testClientID = "test-client-purpose"
)

type PurposeAPITestSuite struct {
	suite.Suite
	createdPurposeIDs []string
	createdElementIDs []string
}

// SetupSuite runs once before all tests
func (ts *PurposeAPITestSuite) SetupSuite() {
	ts.createdPurposeIDs = make([]string, 0)
	ts.createdElementIDs = make([]string, 0)
	ts.T().Logf("=== ConsentPurpose Test Suite Starting ===")
	ts.setupTestElements()
}

// TearDownSuite runs once after all tests to cleanup
func (ts *PurposeAPITestSuite) TearDownSuite() {
	if len(ts.createdPurposeIDs) > 0 {
		ts.T().Logf("=== Cleaning up %d created purposes ===", len(ts.createdPurposeIDs))
		successCount := 0
		failCount := 0
		for _, id := range ts.createdPurposeIDs {
			if ts.deletePurposeWithCheck(id) {
				successCount++
			} else {
				failCount++
			}
		}
		ts.T().Logf("=== Purpose cleanup complete: %d deleted, %d failed ===", successCount, failCount)
	}

	if len(ts.createdElementIDs) > 0 {
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
		ts.T().Logf("=== Element cleanup complete: %d deleted, %d failed ===", successCount, failCount)
	}

	ts.T().Logf("=== ConsentPurpose Test Suite Complete ===")
}

func TestPurposeAPITestSuite(t *testing.T) {
	suite.Run(t, new(PurposeAPITestSuite))
}

// setupTestElements creates elements needed for tests
func (ts *PurposeAPITestSuite) setupTestElements() {
	elementNames := []string{
		"test_email",
		"test_phone",
		"test_address",
		"test_marketing",
		"test_analytics",
	}

	for _, name := range elementNames {
		payload := []map[string]interface{}{
			{
				"name":        name,
				"description": fmt.Sprintf("Test element for %s", name),
				"type":        "basic",
				"properties": map[string]string{
					"value": fmt.Sprintf("/user/%s", name),
				},
			},
		}

		reqBody, _ := json.Marshal(payload)
		httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-elements",
			bytes.NewBuffer(reqBody))
		httpReq.Header.Set("org-id", testOrgID)
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			ts.T().Logf("ERROR creating test element %s: %v", name, err)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusCreated {
			var createResp struct {
				Data []struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if json.Unmarshal(bodyBytes, &createResp) == nil && len(createResp.Data) > 0 {
				ts.createdElementIDs = append(ts.createdElementIDs, createResp.Data[0].ID)
				ts.T().Logf("✓ Created test element: %s (ID: %s)", name, createResp.Data[0].ID)
			} else {
				ts.T().Logf("ERROR: Failed to parse element response for %s", name)
			}
		} else {
			ts.T().Logf("ERROR: Failed to create element %s - Status: %d, Body: %s", name, resp.StatusCode, string(bodyBytes))
		}
	}

	if len(ts.createdElementIDs) == 0 {
		ts.T().Fatal("Failed to create any test elements - tests cannot continue")
	}
	ts.T().Logf("Successfully created %d/%d test elements", len(ts.createdElementIDs), len(elementNames))
}

// createPurpose creates a consent purpose and returns the response
func (ts *PurposeAPITestSuite) createPurpose(payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purposes",
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("org-id", testOrgID)
	httpReq.Header.Set("TPP-client-id", testClientID)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// getPurpose retrieves a consent purpose by ID
func (ts *PurposeAPITestSuite) getPurpose(purposeID string) (*http.Response, []byte) {
	httpReq, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID),
		nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// listPurposes lists consent purposes with optional filters
func (ts *PurposeAPITestSuite) listPurposes(name string, clientIDs []string, elementNames []string, limit, offset int) (*http.Response, []byte) {
	url := fmt.Sprintf("%s/api/v1/consent-purposes?limit=%d&offset=%d", testServerURL, limit, offset)

	if name != "" {
		url += fmt.Sprintf("&name=%s", name)
	}

	if len(clientIDs) > 0 {
		url += fmt.Sprintf("&clientIds=%s", joinStrings(clientIDs, ","))
	}

	if len(elementNames) > 0 {
		url += fmt.Sprintf("&elementNames=%s", joinStrings(elementNames, ","))
	}

	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// updatePurpose updates a consent purpose
func (ts *PurposeAPITestSuite) updatePurpose(purposeID string, payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	httpReq, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("org-id", testOrgID)
	httpReq.Header.Set("TPP-client-id", testClientID)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// deletePurpose deletes a consent purpose
func (ts *PurposeAPITestSuite) deletePurpose(purposeID string) (*http.Response, []byte) {
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID),
		nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// trackPurpose tracks a created purpose for cleanup
func (ts *PurposeAPITestSuite) trackPurpose(purposeID string) {
	ts.createdPurposeIDs = append(ts.createdPurposeIDs, purposeID)
}

// deletePurposeWithCheck attempts to delete a purpose and returns success status
func (ts *PurposeAPITestSuite) deletePurposeWithCheck(purposeID string) bool {
	resp, body := ts.deletePurpose(purposeID)
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		ts.T().Logf("Deleted purpose: %s", purposeID)
		return true
	}
	ts.T().Logf("Failed to delete purpose %s: %d - %s", purposeID, resp.StatusCode, body)
	return false
}

// deleteElementWithCheck attempts to delete an element and returns success status
func (ts *PurposeAPITestSuite) deleteElementWithCheck(elementID string) bool {
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-elements/%s", testServerURL, elementID),
		nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.T().Logf("Failed to delete element %s: %v", elementID, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		ts.T().Logf("Deleted element: %s", elementID)
		return true
	}
	ts.T().Logf("Failed to delete element %s: %d", elementID, resp.StatusCode)
	return false
}

// joinStrings joins string slice with delimiter
func joinStrings(strs []string, delim string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += delim
		}
		result += s
	}
	return result
}
