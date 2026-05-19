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

// Package proxy contains outbound consent-server proxying logic.
package proxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wso2/openfgc/portal/backend/internal/config"
)

var hopByHopHeaders = toCanonicalHeaderSet(
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
)

var (
	// ErrUpstreamTimeout is returned when upstream request times out.
	ErrUpstreamTimeout = errors.New("upstream timeout")
	// ErrUpstreamUnavailable is returned when upstream cannot be reached.
	ErrUpstreamUnavailable = errors.New("upstream unavailable")
)

var proxyFallbackSequence uint64

type routeSpec struct {
	pathParts []string
	methods   map[string]struct{}
}

var allowedAPIRoutes = []routeSpec{
	{pathParts: []string{"consents"}, methods: toMethodSet("GET", "POST")},
	{pathParts: []string{"consents", "attributes"}, methods: toMethodSet("GET")},
	{pathParts: []string{"consents", "validate"}, methods: toMethodSet("POST")},
	{pathParts: []string{"consents", "*"}, methods: toMethodSet("GET", "PUT")},
	{pathParts: []string{"consents", "*", "revoke"}, methods: toMethodSet("PUT")},
	{pathParts: []string{"consents", "*", "authorizations"}, methods: toMethodSet("GET", "POST")},
	{pathParts: []string{"consents", "*", "authorizations", "*"}, methods: toMethodSet("GET", "PUT")},
	{pathParts: []string{"consent-elements"}, methods: toMethodSet("GET", "POST")},
	{pathParts: []string{"consent-elements", "validate"}, methods: toMethodSet("POST")},
	{pathParts: []string{"consent-elements", "*"}, methods: toMethodSet("GET", "PUT", "DELETE")},
	{pathParts: []string{"consent-purposes"}, methods: toMethodSet("GET", "POST")},
	{pathParts: []string{"consent-purposes", "*"}, methods: toMethodSet("GET", "PUT", "DELETE")},
}

// Service proxies requests to consent-server with route-specific transforms.
type Service struct {
	cfg       config.ProxyConfig
	baseURL   *url.URL
	http      *http.Client
	allowlist map[string]struct{}
}

// UpstreamResponse captures proxied response details for caller-managed handling.
type UpstreamResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// NewService builds a proxy service from app config.
func NewService(cfg config.ProxyConfig) (*Service, error) {
	parsed, err := config.ValidateOpenFGCAPIURL(cfg.OpenFGCAPIURL)
	if err != nil {
		return nil, err
	}
	allow := make(map[string]struct{}, len(cfg.AllowedPassthrough))
	for _, m := range cfg.AllowedPassthrough {
		allow[strings.ToUpper(strings.TrimSpace(m))] = struct{}{}
	}
	return &Service{
		cfg:     cfg,
		baseURL: parsed,
		http: &http.Client{
			Timeout: cfg.OpenFGCAPITimeout,
		},
		allowlist: allow,
	}, nil
}

// IsAllowedPassthroughMethod checks whether a passthrough /api method is allowed.
func (s *Service) IsAllowedPassthroughMethod(method string) bool {
	_, ok := s.allowlist[strings.ToUpper(method)]
	return ok
}

// CheckAPIAccess returns whether the API path is known and whether method is allowed.
func (s *Service) CheckAPIAccess(method, fullPath string) (knownPath bool, methodAllowed bool) {
	path, ok := strings.CutPrefix(fullPath, "/api/")
	if !ok {
		return false, false
	}
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return false, false
	}
	parts := strings.Split(trimmed, "/")
	method = strings.ToUpper(method)

	for _, spec := range allowedAPIRoutes {
		if !routeMatches(spec.pathParts, parts) {
			continue
		}
		_, allowed := spec.methods[method]
		return true, allowed
	}

	return false, false
}

// Forward sends a transformed request to upstream and writes the response.
func (s *Service) Forward(w http.ResponseWriter, r *http.Request, upstreamMethod, upstreamPath string, queryMutator func(url.Values), body []byte) error {
	resp, err := s.ForwardRaw(r, upstreamMethod, upstreamPath, queryMutator, body)
	if err != nil {
		return err
	}

	s.copyResponseHeaders(w.Header(), resp.Headers)
	w.WriteHeader(resp.StatusCode)
	if len(resp.Body) == 0 {
		return nil
	}
	_, err = w.Write(resp.Body)
	return err
}

// ForwardWithClientID sends a transformed request to upstream using the provided trusted client id.
func (s *Service) ForwardWithClientID(w http.ResponseWriter, r *http.Request, upstreamMethod, upstreamPath string, queryMutator func(url.Values), body []byte, trustedClientID string) error {
	resp, err := s.ForwardRawWithClientID(r, upstreamMethod, upstreamPath, queryMutator, body, trustedClientID)
	if err != nil {
		return err
	}

	s.copyResponseHeaders(w.Header(), resp.Headers)
	w.WriteHeader(resp.StatusCode)
	if len(resp.Body) == 0 {
		return nil
	}
	_, err = w.Write(resp.Body)
	return err
}

// ForwardRaw sends a transformed request to upstream and returns response status, headers, and body.
func (s *Service) ForwardRaw(r *http.Request, upstreamMethod, upstreamPath string, queryMutator func(url.Values), body []byte) (*UpstreamResponse, error) {
	return s.ForwardRawWithClientID(r, upstreamMethod, upstreamPath, queryMutator, body, "")
}

// ForwardRawWithClientID sends a transformed request to upstream using the provided trusted client id.
func (s *Service) ForwardRawWithClientID(r *http.Request, upstreamMethod, upstreamPath string, queryMutator func(url.Values), body []byte, trustedClientID string) (*UpstreamResponse, error) {
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.OpenFGCAPITimeout)
	defer cancel()

	target := *s.baseURL
	target.Path = joinURLPaths(s.baseURL.Path, upstreamPath)
	query := r.URL.Query()
	if queryMutator != nil {
		queryMutator(query)
	}
	target.RawQuery = query.Encode()

	upstreamReq, err := http.NewRequestWithContext(ctx, upstreamMethod, target.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	s.copyHeaders(r.Header, upstreamReq.Header)
	s.setTrustedHeaders(r, upstreamReq, trustedClientID)

	resp, err := s.http.Do(upstreamReq)
	if err != nil {
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			return nil, ErrUpstreamTimeout
		}
		return nil, ErrUpstreamUnavailable
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upstream response body: %w", ErrUpstreamUnavailable)
	}

	return &UpstreamResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       respBody,
	}, nil
}

func joinURLPaths(basePath string, upstreamPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	upstreamPath = strings.TrimLeft(upstreamPath, "/")

	switch {
	case basePath == "" && upstreamPath == "":
		return ""
	case basePath == "":
		return "/" + upstreamPath
	case upstreamPath == "":
		return basePath
	default:
		return basePath + "/" + upstreamPath
	}
}

func toMethodSet(methods ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		set[strings.ToUpper(m)] = struct{}{}
	}
	return set
}

func toCanonicalHeaderSet(names ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[http.CanonicalHeaderKey(name)] = struct{}{}
	}
	return set
}

func routeMatches(patternParts, parts []string) bool {
	if len(patternParts) != len(parts) {
		return false
	}
	for i := range patternParts {
		if patternParts[i] == "*" {
			if parts[i] == "" {
				return false
			}
			continue
		}
		if patternParts[i] != parts[i] {
			return false
		}
	}
	return true
}

func (s *Service) copyHeaders(src, dst http.Header) {
	connectionHeaders := connectionHeaderNames(src)
	for k, vals := range src {
		if s.skipHeader(k, connectionHeaders) {
			continue
		}
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}

func (s *Service) copyResponseHeaders(dst, src http.Header) {
	connectionHeaders := connectionHeaderNames(src)
	for k, vals := range src {
		canonical := http.CanonicalHeaderKey(k)
		if _, drop := hopByHopHeaders[canonical]; drop {
			continue
		}
		if _, drop := connectionHeaders[canonical]; drop {
			continue
		}
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}

func (s *Service) skipHeader(name string, connectionHeaders map[string]struct{}) bool {
	canonical := http.CanonicalHeaderKey(name)
	if _, drop := hopByHopHeaders[canonical]; drop {
		return true
	}
	if _, drop := connectionHeaders[canonical]; drop {
		return true
	}
	if strings.EqualFold(canonical, "Org-Id") || strings.EqualFold(canonical, "TPP-Client-Id") {
		return true
	}
	if isForwardingHeader(canonical) {
		return true
	}
	if strings.EqualFold(canonical, "Content-Length") {
		return true
	}
	return false
}

func isForwardingHeader(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Forwarded",
		"X-Forwarded-For",
		"X-Forwarded-Host",
		"X-Forwarded-Proto",
		"X-Forwarded-Port",
		"X-Real-Ip",
		"X-Original-Forwarded-For":
		return true
	default:
		return false
	}
}

func connectionHeaderNames(headers http.Header) map[string]struct{} {
	names := make(map[string]struct{})
	for _, value := range headers.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			name := strings.TrimSpace(token)
			if name == "" {
				continue
			}
			names[http.CanonicalHeaderKey(name)] = struct{}{}
		}
	}
	return names
}

func (s *Service) setTrustedHeaders(incoming *http.Request, outgoing *http.Request, trustedClientID string) {
	if s.cfg.PlaceholderOrgID != "" {
		outgoing.Header.Set("org-id", s.cfg.PlaceholderOrgID)
	}
	if trustedClientID != "" {
		outgoing.Header.Set("TPP-client-id", trustedClientID)
	} else if s.cfg.PlaceholderClientID != "" {
		outgoing.Header.Set("TPP-client-id", s.cfg.PlaceholderClientID)
	}
	correlationID := incoming.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = newCorrelationID()
	}
	outgoing.Header.Set("X-Correlation-ID", correlationID)
}

func newCorrelationID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fallbackCorrelationID()
	}
	return hex.EncodeToString(buf)
}

func fallbackCorrelationID() string {
	sequence := atomic.AddUint64(&proxyFallbackSequence, 1)
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	pid := strconv.Itoa(os.Getpid())
	seq := strconv.FormatUint(sequence, 36)

	return "fb-" + timestamp + "-" + pid + "-" + seq
}
