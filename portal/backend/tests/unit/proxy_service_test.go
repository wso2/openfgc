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

package unit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wso2/openfgc/portal/backend/internal/config"
	"github.com/wso2/openfgc/portal/backend/internal/proxy"
)

func TestNewServiceRejectsInvalidUpstreamURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		errText string
	}{
		{name: "empty url", url: "", errText: "must not be empty"},
		{name: "relative url", url: "/consent-server", errText: "must use http or https scheme"},
		{name: "missing host", url: "http:///api", errText: "must include a host"},
		{name: "unsupported scheme", url: "ftp://localhost:9090", errText: "must use http or https scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := proxy.NewService(config.ProxyConfig{
				OpenFGCAPIURL:      tt.url,
				OpenFGCAPITimeout:  2 * time.Second,
				MaxRequestBytes:    1024,
				MaxResponseBytes:   1024,
				AllowedPassthrough: []string{"GET"},
			})
			if err == nil {
				t.Fatal("expected constructor error for invalid URL")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("expected error to contain %q, got %v", tt.errText, err)
			}
		})
	}
}

func TestCheckAPIAccess(t *testing.T) {
	svc, err := proxy.NewService(config.ProxyConfig{
		OpenFGCAPIURL:       "http://localhost:9090",
		OpenFGCAPITimeout:   2 * time.Second,
		MaxRequestBytes:     1024,
		MaxResponseBytes:    1024,
		AllowedPassthrough:  []string{"GET", "POST", "PUT", "DELETE"},
		PlaceholderOrgID:    "ORG-001",
		PlaceholderClientID: "TPP-CLIENT-001",
	})
	if err != nil {
		t.Fatalf("failed to construct service: %v", err)
	}

	tests := []struct {
		name          string
		method        string
		path          string
		expectKnown   bool
		expectAllowed bool
	}{
		{name: "known path and method", method: "GET", path: "/api/consents", expectKnown: true, expectAllowed: true},
		{name: "known path wrong method", method: "DELETE", path: "/api/consents", expectKnown: true, expectAllowed: false},
		{name: "known wildcard path", method: "PUT", path: "/api/consents/abc-123/revoke", expectKnown: true, expectAllowed: true},
		{name: "known wildcard wrong method", method: "POST", path: "/api/consents/abc-123/revoke", expectKnown: true, expectAllowed: false},
		{name: "unknown path", method: "GET", path: "/api/does-not-exist", expectKnown: false, expectAllowed: false},
		{name: "non api prefix", method: "GET", path: "/health", expectKnown: false, expectAllowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			known, allowed := svc.CheckAPIAccess(tt.method, tt.path)
			if known != tt.expectKnown || allowed != tt.expectAllowed {
				t.Fatalf("expected (known=%v, allowed=%v), got (known=%v, allowed=%v)", tt.expectKnown, tt.expectAllowed, known, allowed)
			}
		})
	}
}

func TestIsAllowedPassthroughMethod(t *testing.T) {
	svc, err := proxy.NewService(config.ProxyConfig{
		OpenFGCAPIURL:      "http://localhost:9090",
		OpenFGCAPITimeout:  2 * time.Second,
		MaxRequestBytes:    1024,
		MaxResponseBytes:   1024,
		AllowedPassthrough: []string{"GET", "POST"},
	})
	if err != nil {
		t.Fatalf("failed to construct service: %v", err)
	}

	if !svc.IsAllowedPassthroughMethod("get") {
		t.Fatal("expected lowercase get to be allowed")
	}
	if svc.IsAllowedPassthroughMethod("DELETE") {
		t.Fatal("expected DELETE to be disallowed")
	}
}

func TestForwardRawMapsBodyReadFailureToUpstreamUnavailable(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "10")
		_, _ = w.Write([]byte("abc"))
	}))
	defer upstream.Close()

	svc, err := proxy.NewService(config.ProxyConfig{
		OpenFGCAPIURL:      upstream.URL,
		OpenFGCAPITimeout:  2 * time.Second,
		MaxRequestBytes:    1024,
		MaxResponseBytes:   1024,
		AllowedPassthrough: []string{"GET"},
	})
	if err != nil {
		t.Fatalf("failed to construct service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://bff.local/api/consents", nil)
	_, err = svc.ForwardRaw(req, http.MethodGet, "/api/v1/consents", nil, nil)
	if !errors.Is(err, proxy.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got: %v", err)
	}
}

func TestForwardRawRejectsOversizedUpstreamResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("too-large"))
	}))
	defer upstream.Close()

	svc, err := proxy.NewService(config.ProxyConfig{
		OpenFGCAPIURL:      upstream.URL,
		OpenFGCAPITimeout:  2 * time.Second,
		MaxRequestBytes:    1024,
		MaxResponseBytes:   4,
		AllowedPassthrough: []string{"GET"},
	})
	if err != nil {
		t.Fatalf("failed to construct service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://bff.local/api/consents", nil)
	_, err = svc.ForwardRaw(req, http.MethodGet, "/api/v1/consents", nil, nil)
	if !errors.Is(err, proxy.ErrUpstreamResponseTooLarge) {
		t.Fatalf("expected ErrUpstreamResponseTooLarge, got: %v", err)
	}
}

func TestForwardRawPreservesConfiguredBasePath(t *testing.T) {
	var gotPath string
	var gotQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	svc, err := proxy.NewService(config.ProxyConfig{
		OpenFGCAPIURL:      upstream.URL + "/openfgc/",
		OpenFGCAPITimeout:  2 * time.Second,
		MaxRequestBytes:    1024,
		MaxResponseBytes:   1024,
		AllowedPassthrough: []string{"GET"},
	})
	if err != nil {
		t.Fatalf("failed to construct service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://bff.local/api/consents?limit=10", nil)
	_, err = svc.ForwardRaw(req, http.MethodGet, "/api/v1/consents", func(q url.Values) {
		q.Set("offset", "0")
	}, nil)
	if err != nil {
		t.Fatalf("unexpected forward error: %v", err)
	}

	if gotPath != "/openfgc/api/v1/consents" {
		t.Fatalf("expected joined upstream path, got %s", gotPath)
	}
	if gotQuery != "limit=10&offset=0" {
		t.Fatalf("expected forwarded and mutated query, got %s", gotQuery)
	}
}

func TestForwardStripsHeadersNamedByConnection(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Hop-Debug"); got != "" {
			t.Fatalf("expected X-Hop-Debug to be stripped, got %q", got)
		}
		if got := r.Header.Get("Connection"); got != "" {
			t.Fatalf("expected Connection to be stripped, got %q", got)
		}
		if got := r.Header.Get("TE"); got != "" {
			t.Fatalf("expected TE to be stripped, got %q", got)
		}
		if got := r.Header.Get("X-End-To-End"); got != "request-ok" {
			t.Fatalf("expected X-End-To-End to be forwarded, got %q", got)
		}

		w.Header().Set("Connection", "keep-alive, X-Upstream-Hop")
		w.Header().Set("Keep-Alive", "timeout=5")
		w.Header().Set("TE", "trailers")
		w.Header().Set("X-Upstream-Hop", "1")
		w.Header().Set("X-Upstream-End", "response-ok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	svc, err := proxy.NewService(config.ProxyConfig{
		OpenFGCAPIURL:      upstream.URL,
		OpenFGCAPITimeout:  2 * time.Second,
		MaxRequestBytes:    1024,
		MaxResponseBytes:   1024,
		AllowedPassthrough: []string{"GET"},
	})
	if err != nil {
		t.Fatalf("failed to construct service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://bff.local/api/consents", nil)
	req.Header.Set("Connection", "keep-alive, X-Hop-Debug")
	req.Header.Set("Keep-Alive", "timeout=5")
	req.Header.Set("TE", "trailers")
	req.Header.Set("X-Hop-Debug", "1")
	req.Header.Set("X-End-To-End", "request-ok")

	rr := httptest.NewRecorder()
	err = svc.Forward(rr, req, http.MethodGet, "/api/v1/consents", nil, nil)
	if err != nil {
		t.Fatalf("unexpected forward error: %v", err)
	}

	if got := rr.Header().Get("X-Upstream-Hop"); got != "" {
		t.Fatalf("expected X-Upstream-Hop to be stripped, got %q", got)
	}
	if got := rr.Header().Get("Connection"); got != "" {
		t.Fatalf("expected Connection to be stripped from response, got %q", got)
	}
	if got := rr.Header().Get("TE"); got != "" {
		t.Fatalf("expected TE to be stripped from response, got %q", got)
	}
	if got := rr.Header().Get("X-Upstream-End"); got != "response-ok" {
		t.Fatalf("expected X-Upstream-End to be forwarded, got %q", got)
	}
}

func TestForwardStripsClientForwardingHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, name := range []string{
			"Forwarded",
			"X-Forwarded-For",
			"X-Forwarded-Host",
			"X-Forwarded-Proto",
			"X-Forwarded-Port",
			"X-Real-IP",
			"X-Original-Forwarded-For",
		} {
			if got := r.Header.Get(name); got != "" {
				t.Fatalf("expected %s to be stripped, got %q", name, got)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	svc, err := proxy.NewService(config.ProxyConfig{
		OpenFGCAPIURL:      upstream.URL,
		OpenFGCAPITimeout:  2 * time.Second,
		MaxRequestBytes:    1024,
		MaxResponseBytes:   1024,
		AllowedPassthrough: []string{"GET"},
	})
	if err != nil {
		t.Fatalf("failed to construct service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://bff.local/api/consents", nil)
	req.Header.Set("Forwarded", "for=203.0.113.7;proto=https")
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	req.Header.Set("X-Forwarded-Host", "evil.example")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Port", "443")
	req.Header.Set("X-Real-IP", "203.0.113.7")
	req.Header.Set("X-Original-Forwarded-For", "203.0.113.7")

	rr := httptest.NewRecorder()
	if err := svc.Forward(rr, req, http.MethodGet, "/api/v1/consents", nil, nil); err != nil {
		t.Fatalf("unexpected forward error: %v", err)
	}
}

func TestForwardGeneratesCorrelationIDWhenMissing(t *testing.T) {
	var gotCorrelationID string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCorrelationID = r.Header.Get("X-Correlation-ID")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	svc, err := proxy.NewService(config.ProxyConfig{
		OpenFGCAPIURL:      upstream.URL,
		OpenFGCAPITimeout:  2 * time.Second,
		MaxRequestBytes:    1024,
		MaxResponseBytes:   1024,
		AllowedPassthrough: []string{"GET"},
	})
	if err != nil {
		t.Fatalf("failed to construct service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://bff.local/api/consents", nil)
	rr := httptest.NewRecorder()
	if err := svc.Forward(rr, req, http.MethodGet, "/api/v1/consents", nil, nil); err != nil {
		t.Fatalf("unexpected forward error: %v", err)
	}

	if gotCorrelationID == "" {
		t.Fatal("expected generated correlation id")
	}
}
