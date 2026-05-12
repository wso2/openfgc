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

	"github.com/wso2/openfgc/portal/backend/internal/config"
	"github.com/wso2/openfgc/portal/backend/internal/logger"
	"github.com/wso2/openfgc/portal/backend/internal/router"
)

func newPhase2Server(t *testing.T, upstreamURL string) *httptest.Server {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Proxy.OpenFGCAPIURL = upstreamURL
	cfg.Proxy.PlaceholderModeEnabled = true
	cfg.Proxy.PlaceholderUserID = "user@example.com"
	cfg.Proxy.PlaceholderOrgID = "ORG-001"
	cfg.Proxy.PlaceholderClientID = "TPP-CLIENT-001"

	h, err := router.New(logger.New("error"), *cfg)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}
	return httptest.NewServer(h)
}

func newPhase2ServerWithTimeout(t *testing.T, upstreamURL string, timeout time.Duration) *httptest.Server {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Proxy.OpenFGCAPIURL = upstreamURL
	cfg.Proxy.OpenFGCAPITimeout = timeout
	cfg.Proxy.PlaceholderModeEnabled = true
	cfg.Proxy.PlaceholderUserID = "user@example.com"
	cfg.Proxy.PlaceholderOrgID = "ORG-001"
	cfg.Proxy.PlaceholderClientID = "TPP-CLIENT-001"

	h, err := router.New(logger.New("error"), *cfg)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}
	return httptest.NewServer(h)
}

func newPhase2ServerWithMaxBytes(t *testing.T, upstreamURL string, maxBytes int64) *httptest.Server {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Proxy.OpenFGCAPIURL = upstreamURL
	cfg.Proxy.MaxRequestBytes = maxBytes
	cfg.Proxy.PlaceholderModeEnabled = true
	cfg.Proxy.PlaceholderUserID = "user@example.com"
	cfg.Proxy.PlaceholderOrgID = "ORG-001"
	cfg.Proxy.PlaceholderClientID = "TPP-CLIENT-001"

	h, err := router.New(logger.New("error"), *cfg)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}
	return httptest.NewServer(h)
}

func newPhase2ServerPlaceholderDisabled(t *testing.T, upstreamURL string) *httptest.Server {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Proxy.OpenFGCAPIURL = upstreamURL
	cfg.Proxy.PlaceholderModeEnabled = false
	cfg.Proxy.PlaceholderUserID = ""
	cfg.Proxy.PlaceholderOrgID = "ORG-001"
	cfg.Proxy.PlaceholderClientID = "TPP-CLIENT-001"

	h, err := router.New(logger.New("error"), *cfg)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}
	return httptest.NewServer(h)
}

func TestAPIPassthroughRewriteAndHeaderSafety(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotOrg string
	var gotTPP string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotOrg = r.Header.Get("org-id")
		gotTPP = r.Header.Get("TPP-client-id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	bff := newPhase2Server(t, upstream.URL)
	defer bff.Close()

	req, err := http.NewRequest(http.MethodGet, bff.URL+"/api/consents?limit=10&offset=2", nil)
	if err != nil {
		t.Fatalf("request creation failed: %v", err)
	}
	req.Header.Set("org-id", "MALICIOUS")
	req.Header.Set("TPP-client-id", "MALICIOUS")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if gotPath != "/api/v1/consents" {
		t.Fatalf("expected rewritten path /api/v1/consents, got %s", gotPath)
	}
	if gotQuery != "limit=10&offset=2" {
		t.Fatalf("expected preserved query limit=10&offset=2, got %s", gotQuery)
	}
	if gotOrg != "ORG-001" {
		t.Fatalf("expected trusted org-id header, got %s", gotOrg)
	}
	if gotTPP != "TPP-CLIENT-001" {
		t.Fatalf("expected trusted TPP-client-id header, got %s", gotTPP)
	}
}

func TestMeConsentsForcesUserIDs(t *testing.T) {
	var gotPath string
	var gotQuery string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	bff := newPhase2Server(t, upstream.URL)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents?userIds=attacker&limit=5")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if gotPath != "/api/v1/consents" {
		t.Fatalf("expected path /api/v1/consents, got %s", gotPath)
	}
	queryValues, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("expected valid query string, got parse error: %v", err)
	}
	if queryValues.Get("limit") != "5" {
		t.Fatalf("expected limit=5, got %v", queryValues["limit"])
	}
	userIDs := queryValues["userIds"]
	if len(userIDs) != 1 || userIDs[0] != "user@example.com" {
		t.Fatalf("expected forced userIds=user@example.com, got %v", userIDs)
	}
}

func TestMeConsentByIDStripsHopByHopHeadersFromUpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Connection", "keep-alive, X-Upstream-Hop")
		w.Header().Set("Keep-Alive", "timeout=5")
		w.Header().Set("TE", "trailers")
		w.Header().Set("Upgrade", "websocket")
		w.Header().Set("X-Upstream-Hop", "1")
		w.Header().Set("X-Upstream-End", "response-ok")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND"}`))
	}))
	defer upstream.Close()

	bff := newPhase2Server(t, upstream.URL)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents/missing")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	for _, name := range []string{"Connection", "Keep-Alive", "TE", "Upgrade", "X-Upstream-Hop"} {
		if got := resp.Header.Get(name); got != "" {
			t.Fatalf("expected %s to be stripped, got %q", name, got)
		}
	}
	if got := resp.Header.Get("X-Upstream-End"); got != "response-ok" {
		t.Fatalf("expected end-to-end header to be forwarded, got %q", got)
	}
}

func TestApproveAndRevokeMappings(t *testing.T) {
	t.Run("approve fetches consent and builds put payload", func(t *testing.T) {
		var gotMethod string
		var gotPath string
		var gotBody map[string]any
		var gotTPPClientID string

		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/consents/consent-123":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id":"consent-123",
					"clientId":"TPP-CLIENT-001",
					"type":"accounts",
					"status":"CREATED",
					"frequency":0,
					"validityTime":0,
					"recurringIndicator":false,
					"dataAccessValidityDuration":86400,
					"attributes":{"department":"sales","region":"APAC"},
					"authorizations":[
						{"id":"auth-1","userId":"user1@example.com","type":"authorisation","status":"APPROVED","updatedTime":1702800000,"resources":{"accountIds":["acc-123","acc-456"]}}
					],
					"purposes":[
						{"name":"profile_access","elements":[
							{"name":"first_name","isUserApproved":false,"value":{}},
							{"name":"last_name","isUserApproved":false,"value":{}}
						]}
					]
				}`))
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/consent-purposes":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"data":[
						{"name":"profile_access","clientId":"TPP-CLIENT-001","description":null,"elements":[
							{"name":"first_name","isMandatory":true},
							{"name":"last_name","isMandatory":false}
						]}
					]
				}`))
			case r.Method == http.MethodPut && r.URL.Path == "/api/v1/consents/consent-123":
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotTPPClientID = r.Header.Get("TPP-client-id")
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &gotBody)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer upstream.Close()

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		cfg.Proxy.OpenFGCAPIURL = upstream.URL
		cfg.Proxy.PlaceholderModeEnabled = true
		cfg.Proxy.PlaceholderUserID = "user@example.com"
		cfg.Proxy.PlaceholderOrgID = "ORG-001"
		cfg.Proxy.PlaceholderClientID = "PLACEHOLDER-CLIENT-999"

		h, err := router.New(logger.New("error"), *cfg)
		if err != nil {
			t.Fatalf("failed to create router: %v", err)
		}
		bff := httptest.NewServer(h)
		defer bff.Close()

		payload := []byte(`[{"purposeName":"profile_access","elementName":"last_name"}]`)
		resp, err := http.Post(bff.URL+"/me/consents/consent-123/approve", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if gotMethod != http.MethodPut {
			t.Fatalf("expected PUT, got %s", gotMethod)
		}
		if gotPath != "/api/v1/consents/consent-123" {
			t.Fatalf("unexpected path: %s", gotPath)
		}
		if gotTPPClientID != "TPP-CLIENT-001" {
			t.Fatalf("expected TPP-client-id from consent clientId, got %s", gotTPPClientID)
		}
		if gotBody["type"] != "accounts" {
			t.Fatalf("expected type accounts, got %v", gotBody["type"])
		}
		if gotBody["frequency"] != float64(0) {
			t.Fatalf("expected frequency 0, got %v", gotBody["frequency"])
		}
		if gotBody["validityTime"] != float64(0) {
			t.Fatalf("expected validityTime 0, got %v", gotBody["validityTime"])
		}
		if gotBody["recurringIndicator"] != false {
			t.Fatalf("expected recurringIndicator false, got %v", gotBody["recurringIndicator"])
		}

		purposes, ok := gotBody["purposes"].([]any)
		if !ok || len(purposes) != 1 {
			t.Fatalf("expected one purpose, got %v", gotBody["purposes"])
		}
		purpose, ok := purposes[0].(map[string]any)
		if !ok {
			t.Fatalf("expected purpose object, got %T", purposes[0])
		}
		elements, ok := purpose["elements"].([]any)
		if !ok || len(elements) != 2 {
			t.Fatalf("expected two elements, got %v", purpose["elements"])
		}
		mandatoryElement, ok := elements[0].(map[string]any)
		if !ok {
			t.Fatalf("expected first element object, got %T", elements[0])
		}
		if mandatoryElement["isUserApproved"] != true {
			t.Fatalf("expected mandatory element approved, got %v", mandatoryElement["isUserApproved"])
		}
		optionalElement, ok := elements[1].(map[string]any)
		if !ok {
			t.Fatalf("expected optional element object, got %T", elements[1])
		}
		if optionalElement["isUserApproved"] != true {
			t.Fatalf("expected selected optional element approved, got %v", optionalElement["isUserApproved"])
		}

		authorizations, ok := gotBody["authorizations"].([]any)
		if !ok || len(authorizations) != 1 {
			t.Fatalf("expected one authorization, got %v", gotBody["authorizations"])
		}
		updatedAuthorization, ok := authorizations[0].(map[string]any)
		if !ok {
			t.Fatalf("expected authorization object, got %T", authorizations[0])
		}
		if updatedAuthorization["userId"] != "user@example.com" {
			t.Fatalf("expected current user id, got %v", updatedAuthorization["userId"])
		}
		if updatedAuthorization["status"] != "APPROVED" {
			t.Fatalf("expected approved authorization status, got %v", updatedAuthorization["status"])
		}
		if updatedAuthorization["type"] != "authorisation" {
			t.Fatalf("expected authorization type authorisation, got %v", updatedAuthorization["type"])
		}
		if resources, exists := updatedAuthorization["resources"]; !exists || resources == nil {
			t.Fatalf("expected resources to be present as an empty object, got %v", updatedAuthorization["resources"])
		}
	})

	t.Run("revoke maps to revoke endpoint", func(t *testing.T) {
		var gotMethod string
		var gotPath string
		var gotBody map[string]any

		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			w.WriteHeader(http.StatusOK)
		}))
		defer upstream.Close()

		bff := newPhase2Server(t, upstream.URL)
		defer bff.Close()

		req, err := http.NewRequest(http.MethodPut, bff.URL+"/me/consents/consent-123/revoke", bytes.NewReader([]byte(`{"revocationReason":"test"}`)))
		if err != nil {
			t.Fatalf("request creation failed: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if gotMethod != http.MethodPut {
			t.Fatalf("expected PUT, got %s", gotMethod)
		}
		if gotPath != "/api/v1/consents/consent-123/revoke" {
			t.Fatalf("unexpected path: %s", gotPath)
		}
		if gotBody["actionBy"] != "user@example.com" {
			t.Fatalf("expected placeholder actionBy, got %v", gotBody["actionBy"])
		}
		if gotBody["revocationReason"] != "test" {
			t.Fatalf("expected revocationReason passthrough, got %v", gotBody["revocationReason"])
		}
	})

	t.Run("revoke accepts null payload", func(t *testing.T) {
		var gotMethod string
		var gotPath string
		var gotBody map[string]any

		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			w.WriteHeader(http.StatusOK)
		}))
		defer upstream.Close()

		bff := newPhase2Server(t, upstream.URL)
		defer bff.Close()

		req, err := http.NewRequest(http.MethodPut, bff.URL+"/me/consents/consent-123/revoke", bytes.NewReader([]byte(`null`)))
		if err != nil {
			t.Fatalf("request creation failed: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if gotMethod != http.MethodPut {
			t.Fatalf("expected PUT, got %s", gotMethod)
		}
		if gotPath != "/api/v1/consents/consent-123/revoke" {
			t.Fatalf("unexpected path: %s", gotPath)
		}
		if gotBody["actionBy"] != "user@example.com" {
			t.Fatalf("expected placeholder actionBy, got %v", gotBody["actionBy"])
		}
		if len(gotBody) != 1 {
			t.Fatalf("expected only actionBy in payload, got %v", gotBody)
		}
	})
}

func TestAPIDenyByDefault(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	bff := newPhase2Server(t, upstream.URL)
	defer bff.Close()

	t.Run("unknown path returns 404", func(t *testing.T) {
		resp, err := http.Get(bff.URL + "/api/unknown/resource")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("unknown path with disallowed method returns 404", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodTrace, bff.URL+"/api/unknown/resource", nil)
		if err != nil {
			t.Fatalf("request creation failed: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("known path wrong method returns 405", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, bff.URL+"/api/consents", nil)
		if err != nil {
			t.Fatalf("request creation failed: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})
}

func TestProxyTimeoutMapsTo503(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	bff := newPhase2ServerWithTimeout(t, upstream.URL, 40*time.Millisecond)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/api/consents")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json error payload: %v", err)
	}
	if payload["code"] != "UPSTREAM_TIMEOUT" {
		t.Fatalf("expected code UPSTREAM_TIMEOUT, got %v", payload["code"])
	}
}

func TestRequestBodySizeLimitReturns413(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	bff := newPhase2ServerWithMaxBytes(t, upstream.URL, 8)
	defer bff.Close()

	t.Run("api passthrough returns 413 with json error", func(t *testing.T) {
		big := bytes.Repeat([]byte("a"), 32)
		req, err := http.NewRequest(http.MethodPost, bff.URL+"/api/consents", bytes.NewReader(big))
		if err != nil {
			t.Fatalf("request creation failed: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected content type application/json, got %s", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("expected json error payload: %v", err)
		}
		if payload["code"] != "REQUEST_TOO_LARGE" {
			t.Fatalf("expected code REQUEST_TOO_LARGE, got %v", payload["code"])
		}
	})

	t.Run("me approve returns 413 with json error", func(t *testing.T) {
		big := bytes.Repeat([]byte("b"), 64)
		resp, err := http.Post(bff.URL+"/me/consents/c-1/approve", "application/json", bytes.NewReader(big))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413, got %d", resp.StatusCode)
		}
		var payload map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("expected json error payload: %v", err)
		}
		if payload["code"] != "REQUEST_TOO_LARGE" {
			t.Fatalf("expected code REQUEST_TOO_LARGE, got %v", payload["code"])
		}
	})
}

func TestUpstreamUnavailableMapsTo502(t *testing.T) {
	bff := newPhase2Server(t, "http://127.0.0.1:1")
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/api/consents")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json error payload: %v", err)
	}
	if payload["code"] != "UPSTREAM_UNAVAILABLE" {
		t.Fatalf("expected code UPSTREAM_UNAVAILABLE, got %v", payload["code"])
	}
}

func TestMeEndpointsReturn503WhenPlaceholderModeDisabled(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	bff := newPhase2ServerPlaceholderDisabled(t, upstream.URL)
	defer bff.Close()

	testCases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "me consents", method: http.MethodGet, path: "/me/consents"},
		{name: "me consent by id", method: http.MethodGet, path: "/me/consents/consent-123"},
		{name: "me approve", method: http.MethodPost, path: "/me/consents/consent-123/approve", body: "[]"},
		{name: "me revoke", method: http.MethodPut, path: "/me/consents/consent-123/revoke", body: "{}"},
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

			if resp.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("expected 503, got %d", resp.StatusCode)
			}

			if upstreamCalled {
				t.Fatal("expected request to be blocked before upstream call")
			}

			var payload map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatalf("expected json error payload: %v", err)
			}
			if payload["code"] != "IDENTITY_UNAVAILABLE" {
				t.Fatalf("expected IDENTITY_UNAVAILABLE, got %v", payload["code"])
			}
		})
	}
}

func TestMeConsentByIDAggregatesPurposeAndElementDetails(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/consents/consent-123":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id":"consent-123",
				"clientId":"TPP-CLIENT-001",
				"type":"accounts",
				"status":"ACTIVE",
				"createdTime":1702800000,
				"updatedTime":1702800001,
				"purposes":[
					{
						"name":"marketing_communication_preferences",
						"elements":[
							{"name":"user_email","isUserApproved":true}
						]
					}
				]
			}`))
		case "/api/v1/consent-purposes":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data":[
					{
						"name":"marketing_communication_preferences",
						"description":"Marketing communication consent",
						"elements":[
							{"name":"user_email","isMandatory":true}
						]
					}
				]
			}`))
		case "/api/v1/consent-elements":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data":[
					{
						"name":"user_email",
						"type":"basic",
						"description":"User email address",
						"properties":{"channel":"email"}
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	bff := newPhase2Server(t, upstream.URL)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents/consent-123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json response: %v", err)
	}

	purposes, ok := payload["purposes"].([]any)
	if !ok || len(purposes) != 1 {
		t.Fatalf("expected one purpose, got %v", payload["purposes"])
	}
	purpose, ok := purposes[0].(map[string]any)
	if !ok {
		t.Fatalf("expected purpose object, got %T", purposes[0])
	}
	if purpose["description"] != "Marketing communication consent" {
		t.Fatalf("expected purpose description to be enriched, got %v", purpose["description"])
	}

	elements, ok := purpose["elements"].([]any)
	if !ok || len(elements) != 1 {
		t.Fatalf("expected one element, got %v", purpose["elements"])
	}
	element, ok := elements[0].(map[string]any)
	if !ok {
		t.Fatalf("expected element object, got %T", elements[0])
	}
	if mandatory, ok := element["isMandatory"].(bool); !ok || !mandatory {
		t.Fatalf("expected isMandatory=true, got %v", element["isMandatory"])
	}
	if element["type"] != "basic" {
		t.Fatalf("expected enriched element type, got %v", element["type"])
	}
	if element["description"] != "User email address" {
		t.Fatalf("expected enriched element description, got %v", element["description"])
	}
}

func TestMeConsentByIDFailsClosedWhenPurposeMetadataMissing(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/consents/consent-123":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id":"consent-123",
				"clientId":"TPP-CLIENT-001",
				"type":"accounts",
				"status":"ACTIVE",
				"createdTime":1702800000,
				"updatedTime":1702800001,
				"purposes":[{"name":"marketing_communication_preferences","elements":[{"name":"user_email","isUserApproved":true}]}]
			}`))
		case "/api/v1/consent-purposes":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	bff := newPhase2Server(t, upstream.URL)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents/consent-123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json error payload: %v", err)
	}
	if payload["code"] != "UPSTREAM_UNAVAILABLE" {
		t.Fatalf("expected code UPSTREAM_UNAVAILABLE, got %v", payload["code"])
	}
}

func TestMeConsentByIDHandlesNullableAndMixedProperties(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/consents/consent-123":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id":"consent-123",
				"clientId":"TPP-CLIENT-001",
				"type":"accounts",
				"status":"ACTIVE",
				"frequency":null,
				"validityTime":null,
				"recurringIndicator":null,
				"dataAccessValidityDuration":null,
				"attributes":{"consentMode":1},
				"authorizations":[{"id":"auth-1","userId":null,"type":"authorisation","status":"APPROVED","updatedTime":1702800002}],
				"createdTime":1702800000,
				"updatedTime":1702800001,
				"purposes":[{"name":"marketing_communication_preferences","elements":[{"name":"user_email","isUserApproved":true}]}]
			}`))
		case "/api/v1/consent-purposes":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data":[{"name":"marketing_communication_preferences","description":null,"elements":[{"name":"user_email","isMandatory":true}]}]
			}`))
		case "/api/v1/consent-elements":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data":[{"name":"user_email","type":"json-payload","description":null,"properties":{"validationSchema":{"type":"object"}}}]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	bff := newPhase2Server(t, upstream.URL)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents/consent-123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json response: %v", err)
	}

	purposes, ok := payload["purposes"].([]any)
	if !ok || len(purposes) != 1 {
		t.Fatalf("expected one purpose, got %v", payload["purposes"])
	}
	purpose := purposes[0].(map[string]any)
	elements := purpose["elements"].([]any)
	element := elements[0].(map[string]any)

	if mandatory, ok := element["isMandatory"].(bool); !ok || !mandatory {
		t.Fatalf("expected isMandatory=true, got %v", element["isMandatory"])
	}
	if element["type"] != "json-payload" {
		t.Fatalf("expected enriched element type, got %v", element["type"])
	}
	properties, ok := element["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %T", element["properties"])
	}
	if _, ok := properties["validationSchema"].(map[string]any); !ok {
		t.Fatalf("expected validationSchema object, got %T", properties["validationSchema"])
	}
}

func TestMeConsentByIDPurposeLookupFallsBackWithoutClientFilter(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/consents/consent-123":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id":"consent-123",
				"clientId":"TPP-CLIENT-003",
				"type":"accounts",
				"status":"ACTIVE",
				"createdTime":1702800000,
				"updatedTime":1702800001,
				"purposes":[{"name":"data_sharing_purpose","elements":[{"name":"user_email","isUserApproved":true}]}]
			}`))
		case "/api/v1/consent-purposes":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if strings.Contains(r.URL.RawQuery, "clientIds=TPP-CLIENT-003") {
				_, _ = w.Write([]byte(`{"data":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"data":[{"clientId":"TPP-CLIENT-002","name":"data_sharing_purpose","description":"Third-party data sharing purpose","elements":[{"name":"user_email","isMandatory":false}]}]
			}`))
		case "/api/v1/consent-elements":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"name":"user_email","type":"basic","description":"User email","properties":{}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	bff := newPhase2Server(t, upstream.URL)
	defer bff.Close()

	resp, err := http.Get(bff.URL + "/me/consents/consent-123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json response: %v", err)
	}
	purposes, ok := payload["purposes"].([]any)
	if !ok || len(purposes) != 1 {
		t.Fatalf("expected one purpose, got %v", payload["purposes"])
	}
	purpose := purposes[0].(map[string]any)
	if purpose["description"] != "Third-party data sharing purpose" {
		t.Fatalf("expected fallback purpose description, got %v", purpose["description"])
	}
}
