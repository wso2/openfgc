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

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wso2/openfgc/portal/backend/internal/system/auth"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
)

const consentID = "550e8400-e29b-41d4-a716-446655440000"

func newPortalServer(t *testing.T, upstreamURL string, configure func(*config.Config)) *httptest.Server {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Proxy.OpenFGCAPIURL = upstreamURL
	cfg.Proxy.PlaceholderModeEnabled = true
	cfg.Proxy.PlaceholderUserID = "user@example.com"
	cfg.Proxy.PlaceholderOrgID = "ORG-001"
	cfg.Proxy.PlaceholderScopes = append([]string(nil), auth.AllPortalScopes...)
	if configure != nil {
		configure(cfg)
	}
	h, err := newIntegrationHandler(*cfg)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	return httptest.NewServer(h)
}

type failingReadCloser struct{}

func (failingReadCloser) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failingReadCloser) Close() error               { return nil }

func TestAPIPassthroughRewriteAndHeaderSafety(t *testing.T) {
	var gotPath, gotQuery, gotOrg, gotGroup, gotLegacyClient string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotOrg = r.Header.Get("org-id")
		gotGroup = r.Header.Get("group-id")
		gotLegacyClient = r.Header.Get("TPP-client-id")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	req, _ := http.NewRequest(http.MethodGet, bff.URL+"/api/consents?limit=10&offset=2", nil)
	req.Header.Set("org-id", "MALICIOUS")
	req.Header.Set("group-id", "MALICIOUS")
	req.Header.Set("TPP-client-id", "MALICIOUS")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK || gotPath != "/api/v1/consents" || gotQuery != "limit=10&offset=2" {
		t.Fatalf("unexpected passthrough result: status=%d path=%s query=%s", resp.StatusCode, gotPath, gotQuery)
	}
	if gotOrg != "ORG-001" || gotGroup != "" || gotLegacyClient != "" {
		t.Fatalf("unexpected trusted headers: org=%q group=%q legacy=%q", gotOrg, gotGroup, gotLegacyClient)
	}
}

func TestMeConsentsForcesUserIDs(t *testing.T) {
	var gotQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents?userIds=attacker&limit=5&details=false")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	values, _ := url.ParseQuery(gotQuery)
	if resp.StatusCode != http.StatusOK || values.Get("limit") != "5" || values.Get("details") != "true" || values.Get("userIds") != "user@example.com" {
		t.Fatalf("unexpected forced query: status=%d query=%v", resp.StatusCode, values)
	}
}

func TestConsentDetailsUseSingleDetailedConsentRequest(t *testing.T) {
	requested := make(map[string]int)
	var gotDetails string
	var gotStatusHistory string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/consents/" + consentID:
			gotDetails = r.URL.Query().Get("details")
			gotStatusHistory = r.URL.Query().Get("includeStatusHistory")
			_, _ = w.Write([]byte(`{"id":"` + consentID + `","groupId":"GROUP-001","type":"accounts","status":"ACTIVE","createdTime":1702800000000,"updatedTime":1702800001000,"attributes":{},"authorizations":[{"userId":"user@example.com","type":"authorisation","status":"APPROVED"}],"purposes":[{"purposeId":"purpose-profile","name":"profile_access","version":"v2","displayName":"Profile access","description":"Profile purpose","elements":[{"elementId":"element-email","name":"email","namespace":"profile","version":"v3","displayName":"Email address","description":"User email address","mandatory":true,"approved":true}]}],"statusHistory":[{"statusAuditId":"audit-1","currentStatus":"CREATED","actionTime":1702800000000,"actionBy":"client","reason":"Consent created"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents/" + consentID + "?details=false&includeStatusHistory=true")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var payload struct {
		GroupID       string `json:"groupId"`
		StatusHistory []struct {
			StatusAuditID string `json:"statusAuditId"`
			CurrentStatus string `json:"currentStatus"`
		} `json:"statusHistory"`
		Purposes []struct {
			PurposeID   string  `json:"purposeId"`
			Version     string  `json:"version"`
			DisplayName *string `json:"displayName"`
			Description *string `json:"description"`
			Elements    []struct {
				ElementID   string  `json:"elementId"`
				Namespace   string  `json:"namespace"`
				Version     string  `json:"version"`
				DisplayName *string `json:"displayName"`
				Description *string `json:"description"`
				Approved    bool    `json:"approved"`
				Mandatory   bool    `json:"mandatory"`
			} `json:"elements"`
		} `json:"purposes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	element := payload.Purposes[0].Elements[0]
	if payload.GroupID != "GROUP-001" || payload.Purposes[0].PurposeID != "purpose-profile" || payload.Purposes[0].Version != "v2" {
		t.Fatalf("unexpected purpose binding: %+v", payload.Purposes[0])
	}
	if element.ElementID != "element-email" || element.Namespace != "profile" || element.Version != "v3" || !element.Approved || !element.Mandatory {
		t.Fatalf("unexpected element binding: %+v", element)
	}
	if payload.Purposes[0].DisplayName == nil || *payload.Purposes[0].DisplayName != "Profile access" ||
		payload.Purposes[0].Description == nil || *payload.Purposes[0].Description != "Profile purpose" ||
		element.DisplayName == nil || *element.DisplayName != "Email address" ||
		element.Description == nil || *element.Description != "User email address" {
		t.Fatalf("missing details from consent response: purpose=%+v element=%+v", payload.Purposes[0], element)
	}
	if len(payload.StatusHistory) != 1 || payload.StatusHistory[0].CurrentStatus != "CREATED" {
		t.Fatalf("missing status history from consent response: %+v", payload.StatusHistory)
	}
	if gotDetails != "true" || gotStatusHistory != "true" || len(requested) != 1 ||
		requested["/api/v1/consents/"+consentID] != 1 {
		t.Fatalf("expected one detailed consent request with history, details=%q statusHistory=%q requests=%v",
			gotDetails, gotStatusHistory, requested)
	}
}

func TestApprovalBuildsVersionedUpdateAndTrustedGroupHeader(t *testing.T) {
	var updateBody map[string]any
	var updateGroup string
	var consentDetails string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/consents/"+consentID:
			consentDetails = r.URL.Query().Get("details")
			_, _ = w.Write([]byte(`{"id":"` + consentID + `","groupId":"GROUP-BOUND","type":"accounts","status":"CREATED","frequency":0,"expirationTime":0,"recurringIndicator":false,"dataAccessValidityDuration":86400,"attributes":{"region":"APAC"},"authorizations":[{"id":"auth-1","userId":"existing@example.com","type":"authorisation","status":"APPROVED","updatedTime":1702800000000,"resources":{}},{"id":"auth-2","userId":"user@example.com","type":"authorisation","status":"CREATED","updatedTime":1702800000000,"resources":{}}],"purposes":[{"purposeId":"purpose-profile","name":"profile_access","version":"v2","elements":[{"elementId":"element-first","name":"first_name","namespace":"profile","version":"v1","mandatory":true,"approved":false},{"elementId":"element-last","name":"last_name","namespace":"profile","version":"v2","mandatory":false,"approved":false},{"elementId":"element-email","name":"email","namespace":"profile","version":"v3","mandatory":false,"approved":true}]}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/consents/"+consentID:
			updateGroup = r.Header.Get("group-id")
			_ = json.NewDecoder(r.Body).Decode(&updateBody)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	selection := `[{"purposeId":"purpose-profile","purposeVersion":"v2","elementId":"element-last","elementVersion":"v2"}]`
	resp, err := http.Post(bff.URL+"/me/consents/"+consentID+"/approve", "application/json", strings.NewReader(selection))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK || updateGroup != "GROUP-BOUND" || consentDetails != "true" {
		t.Fatalf("unexpected approval result: status=%d group=%q details=%q", resp.StatusCode, updateGroup, consentDetails)
	}
	if updateBody["expirationTime"] != float64(0) {
		t.Fatalf("expected expirationTime to be preserved, got %v", updateBody)
	}
	purpose := updateBody["purposes"].([]any)[0].(map[string]any)
	if purpose["version"] != "v2" {
		t.Fatalf("expected bound purpose version, got %v", purpose)
	}
	elements := purpose["elements"].([]any)
	for index, expectedVersion := range []string{"v1", "v2", "v3"} {
		element := elements[index].(map[string]any)
		if element["version"] != expectedVersion || element["namespace"] != "profile" || element["approved"] != true {
			t.Fatalf("unexpected updated element %d: %v", index, element)
		}
	}
	if len(updateBody["authorizations"].([]any)) != 2 {
		t.Fatalf("expected existing and current-user authorizations, got %v", updateBody["authorizations"])
	}
}

func TestApprovalRejectsUnknownOrMandatorySelection(t *testing.T) {
	for _, selection := range []string{
		`[{"purposeId":"purpose-profile","purposeVersion":"v2","elementId":"unknown","elementVersion":"v1"}]`,
		`[{"purposeId":"purpose-profile","purposeVersion":"v2","elementId":"element-first","elementVersion":"v1"}]`,
	} {
		t.Run(selection, func(t *testing.T) {
			upstream := approvalReadOnlyUpstream(t)
			defer upstream.Close()
			bff := newPortalServer(t, upstream.URL, nil)
			defer bff.Close()
			resp, err := http.Post(bff.URL+"/me/consents/"+consentID+"/approve", "application/json", strings.NewReader(selection))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func approvalReadOnlyUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/consents/" + consentID:
			if r.URL.Query().Get("details") != "true" {
				t.Errorf("expected details=true, got %v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`{"id":"` + consentID + `","groupId":"GROUP-001","type":"accounts","status":"CREATED","attributes":{},"authorizations":[{"userId":"user@example.com","type":"authorisation","status":"CREATED"}],"purposes":[{"purposeId":"purpose-profile","name":"profile_access","version":"v2","elements":[{"elementId":"element-first","name":"first_name","namespace":"profile","version":"v1","mandatory":true,"approved":false}]}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestRevokeUsesPostAndInjectsActionBy(t *testing.T) {
	var method, path string
	var body map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/consents/"+consentID {
			_, _ = w.Write([]byte(`{"id":"` + consentID + `","groupId":"GROUP-001","authorizations":[{"userId":"user@example.com"}]}`))
			return
		}
		method, path = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	req, _ := http.NewRequest(http.MethodPost, bff.URL+"/me/consents/"+consentID+"/revoke", strings.NewReader(`{"revocationReason":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK || method != http.MethodPost || path != "/api/v1/consents/"+consentID+"/revoke" {
		t.Fatalf("unexpected revoke mapping: status=%d method=%s path=%s", resp.StatusCode, method, path)
	}
	if body["actionBy"] != "user@example.com" || body["revocationReason"] != "test" {
		t.Fatalf("unexpected revoke payload: %v", body)
	}
}

func TestRejectBuildsAuthorizationOnlyUpdateAndTrustedGroupHeader(t *testing.T) {
	var updateBody map[string]any
	var updateGroup string
	var consentDetails string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/consents/"+consentID:
			consentDetails = r.URL.Query().Get("details")
			_, _ = w.Write([]byte(`{"id":"` + consentID + `","groupId":"GROUP-BOUND","type":"accounts","status":"CREATED","attributes":{"region":"APAC"},"authorizations":[{"id":"auth-1","userId":"existing@example.com","type":"delegated","status":"APPROVED","updatedTime":1702800000000,"resources":{"accountIds":["acc-1"]}},{"id":"auth-2","userId":"user@example.com","type":"authorisation","status":"CREATED","updatedTime":1702800000000,"resources":{"accountIds":["acc-2"]}}],"purposes":[{"purposeId":"purpose-profile","name":"profile_access","version":"v2","elements":[{"elementId":"element-first","name":"first_name","namespace":"profile","version":"v1","mandatory":true,"approved":false}]}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/consents/"+consentID:
			updateGroup = r.Header.Get("group-id")
			_ = json.NewDecoder(r.Body).Decode(&updateBody)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	resp, err := http.Post(bff.URL+"/me/consents/"+consentID+"/reject", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK || updateGroup != "GROUP-BOUND" || consentDetails != "true" {
		t.Fatalf("unexpected rejection result: status=%d group=%q details=%q", resp.StatusCode, updateGroup, consentDetails)
	}
	if len(updateBody) != 1 || updateBody["purposes"] != nil || updateBody["attributes"] != nil {
		t.Fatalf("expected authorization-only update, got %v", updateBody)
	}
	authorizations := updateBody["authorizations"].([]any)
	if len(authorizations) != 2 {
		t.Fatalf("expected all authorizations to be preserved, got %v", authorizations)
	}
	existing := authorizations[0].(map[string]any)
	if existing["userId"] != "existing@example.com" || existing["type"] != "delegated" ||
		existing["status"] != "APPROVED" {
		t.Fatalf("unexpected existing authorization: %v", existing)
	}
	currentUser := authorizations[1].(map[string]any)
	if currentUser["userId"] != "user@example.com" || currentUser["type"] != "authorisation" ||
		currentUser["status"] != "REJECTED" {
		t.Fatalf("unexpected current-user authorization: %v", currentUser)
	}
	resources := currentUser["resources"].(map[string]any)
	if resources["accountIds"].([]any)[0] != "acc-2" {
		t.Fatalf("expected current-user resources to be preserved, got %v", resources)
	}
}

func TestRejectRequiresCreatedConsent(t *testing.T) {
	updateCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/consents/"+consentID {
			_, _ = w.Write([]byte(`{"id":"` + consentID + `","groupId":"GROUP-001","status":"ACTIVE","authorizations":[{"userId":"user@example.com","type":"authorisation","status":"APPROVED","resources":{}}]}`))
			return
		}
		if r.Method == http.MethodPut {
			updateCalled = true
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	resp, err := http.Post(bff.URL+"/me/consents/"+consentID+"/reject", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusConflict || updateCalled {
		t.Fatalf("expected local 409 without update, got status=%d updateCalled=%v", resp.StatusCode, updateCalled)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["code"] != "INVALID_CONSENT_STATE" {
		t.Fatalf("unexpected error response: %v", body)
	}
}

func TestRejectConcealsOwnershipFailure(t *testing.T) {
	updateCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/consents/"+consentID {
			_, _ = w.Write([]byte(`{"id":"` + consentID + `","groupId":"GROUP-001","status":"CREATED","authorizations":[{"userId":"another-user@example.com","type":"authorisation","status":"CREATED","resources":{}}]}`))
			return
		}
		if r.Method == http.MethodPut {
			updateCalled = true
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	resp, err := http.Post(bff.URL+"/me/consents/"+consentID+"/reject", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusNotFound || updateCalled {
		t.Fatalf("expected concealed 404 without update, got status=%d updateCalled=%v", resp.StatusCode, updateCalled)
	}
}

func TestAPIDenyByDefault(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	tests := []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/unknown/resource", http.StatusNotFound},
		{http.MethodDelete, "/api/consents", http.StatusMethodNotAllowed},
		{http.MethodPut, "/api/consents/" + consentID + "/revoke", http.StatusMethodNotAllowed},
		{http.MethodPut, "/api/consent-elements/element-1", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/api/consent-purposes/purpose-1", http.StatusMethodNotAllowed},
	}
	for _, tc := range tests {
		req, _ := http.NewRequest(tc.method, bff.URL+tc.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Fatalf("%s %s: expected %d, got %d", tc.method, tc.path, tc.status, resp.StatusCode)
		}
	}
}

func TestProxyErrorAndBodyLimits(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer upstream.Close()
		bff := newPortalServer(t, upstream.URL, func(cfg *config.Config) { cfg.Proxy.OpenFGCAPITimeout = 20 * time.Millisecond })
		defer bff.Close()
		resp, _ := http.Get(bff.URL + "/api/consents")
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != http.StatusGatewayTimeout {
			t.Fatalf("expected 504, got %d", resp.StatusCode)
		}
	})

	t.Run("request too large", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
		defer upstream.Close()
		bff := newPortalServer(t, upstream.URL, func(cfg *config.Config) { cfg.Proxy.MaxRequestBytes = 8 })
		defer bff.Close()
		resp, _ := http.Post(bff.URL+"/api/consents", "application/json", bytes.NewReader(bytes.Repeat([]byte("a"), 32)))
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413, got %d", resp.StatusCode)
		}
	})

	t.Run("body read failure", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
		defer upstream.Close()
		cfg, _ := config.Load()
		cfg.Proxy.OpenFGCAPIURL = upstream.URL
		cfg.Proxy.PlaceholderModeEnabled = true
		cfg.Proxy.PlaceholderUserID = "user@example.com"
		cfg.Proxy.PlaceholderOrgID = "ORG-001"
		cfg.Proxy.PlaceholderScopes = append([]string(nil), auth.AllPortalScopes...)
		h, _ := newIntegrationHandler(*cfg)
		req := httptest.NewRequest(http.MethodPost, "/api/consents", nil)
		req.Body = failingReadCloser{}
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", res.Code)
		}
	})
}

func TestMeEndpointsReturn503WhenPlaceholderModeDisabled(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, func(cfg *config.Config) {
		cfg.Proxy.PlaceholderModeEnabled = false
		cfg.Proxy.PlaceholderUserID = ""
		cfg.Proxy.PlaceholderOrgID = ""
		cfg.Proxy.PlaceholderScopes = nil
	})
	defer bff.Close()

	tests := []struct{ method, path, body string }{
		{http.MethodGet, "/me/consents", ""},
		{http.MethodGet, "/me/consents/" + consentID, ""},
		{http.MethodPost, "/me/consents/" + consentID + "/approve", "[]"},
		{http.MethodPost, "/me/consents/" + consentID + "/reject", "{}"},
		{http.MethodPost, "/me/consents/" + consentID + "/revoke", "{}"},
	}
	for _, tc := range tests {
		req, _ := http.NewRequest(tc.method, bff.URL+tc.path, strings.NewReader(tc.body))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized || called {
			t.Fatalf("%s %s: expected local 401, got status=%d called=%v", tc.method, tc.path, resp.StatusCode, called)
		}
	}
}
func TestConsentDetailsConcealsOwnershipFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/consents/"+consentID {
			_, _ = w.Write([]byte(`{"id":"` + consentID + `","groupId":"GROUP-001","authorizations":[{"userId":"another-user@example.com"}],"purposes":[]}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()
	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents/" + consentID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected concealed 404, got %d", resp.StatusCode)
	}
}

func TestMeEndpointsRejectInvalidConsentID(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	bff := newPortalServer(t, upstream.URL, nil)
	defer bff.Close()

	testCases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "get by id", method: http.MethodGet, path: "/me/consents/not-a-uuid"},
		{name: "approve", method: http.MethodPost, path: "/me/consents/not-a-uuid/approve", body: "[]"},
		{name: "reject", method: http.MethodPost, path: "/me/consents/not-a-uuid/reject", body: "{}"},
		{name: "revoke", method: http.MethodPost, path: "/me/consents/not-a-uuid/revoke", body: "{}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			upstreamCalled = false

			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}

			req, err := http.NewRequest(tc.method, bff.URL+tc.path, body)
			if err != nil {
				t.Fatalf("request creation failed: %v", err)
			}
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
			if upstreamCalled {
				t.Fatal("expected request to be rejected before upstream call")
			}

			var payload map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatalf("expected json error payload: %v", err)
			}
			if payload["code"] != "INVALID_CONSENT_ID" {
				t.Fatalf("expected INVALID_CONSENT_ID, got %v", payload["code"])
			}
		})
	}
}
