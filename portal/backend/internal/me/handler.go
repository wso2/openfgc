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

// Package me contains user-facing aggregated /me endpoints for the BFF.
package me

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/wso2/openfgc/portal/backend/internal/proxy"
	"github.com/wso2/openfgc/portal/backend/internal/system/auth"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
	systemcontext "github.com/wso2/openfgc/portal/backend/internal/system/context"
)

var errInvalidConsentState = errors.New("invalid consent state")

// Handler serves /me route groups.
type Handler struct {
	svc *Service
	cfg config.ProxyConfig
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type currentUserResponse struct {
	UserID         string   `json:"userId"`
	OrganizationID string   `json:"organizationId"`
	Scopes         []string `json:"scopes"`
}

var errRequestBodyTooLarge = errors.New("request body too large")
var consentIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// NewHandler creates a me handler with initialized service.
func NewHandler(cfg config.ProxyConfig) (*Handler, error) {
	svc, err := NewService(cfg)
	if err != nil {
		return nil, err
	}
	return &Handler{svc: svc, cfg: cfg}, nil
}

// CurrentUser handles GET /me.
func (h *Handler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := systemcontext.PrincipalFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(currentUserResponse{
		UserID:         principal.UserID,
		OrganizationID: principal.OrgID,
		Scopes:         auth.GrantedPortalScopes(principal.Scopes),
	})
}

// Consents handles GET /me/consents.
func (h *Handler) Consents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Forward(w, r, http.MethodGet, "/api/v1/consents", func(q url.Values) {
		q.Set("userIds", userID)
		q.Set("details", "true")
	}, nil); err != nil {
		writeProxyError(w, err)
	}
}

// ConsentByID handles GET /me/consents/{consentId}.
func (h *Handler) ConsentByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}
	consentID := r.PathValue("consentId")
	if consentID == "" {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent id not found")
		return
	}
	if !isValidConsentID(consentID) {
		writeJSONError(w, http.StatusBadRequest, "INVALID_CONSENT_ID", "invalid consent id")
		return
	}

	baseResp, err := h.svc.ForwardRaw(r, http.MethodGet, "/api/v1/consents/"+url.PathEscape(consentID), func(q url.Values) {
		q.Set("details", "true")
		q.Set("includeStatusHistory", "true")
	}, nil)
	if err != nil {
		writeProxyError(w, err)
		return
	}
	if baseResp.StatusCode != http.StatusOK {
		h.svc.WriteUpstreamResponse(w, baseResp)
		return
	}
	if _, owned, ownershipErr := OwnedConsentGroupID(baseResp.Body, userID); ownershipErr != nil {
		writeProxyError(w, ownershipErr)
		return
	} else if !owned {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent not found")
		return
	}

	h.svc.WriteUpstreamResponse(w, baseResp)
}

// ConsentApprove handles POST /me/consents/{consentId}/approve.
func (h *Handler) ConsentApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}
	consentID := r.PathValue("consentId")
	if consentID == "" {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent id not found")
		return
	}
	if !isValidConsentID(consentID) {
		writeJSONError(w, http.StatusBadRequest, "INVALID_CONSENT_ID", "invalid consent id")
		return
	}
	body, err := h.readBoundedBody(r)
	if err != nil {
		writeBodyReadError(w, err)
		return
	}
	selections, err := parseApprovalSelections(body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload")
		return
	}
	baseResp, err := h.svc.ForwardRaw(r, http.MethodGet, "/api/v1/consents/"+url.PathEscape(consentID), func(q url.Values) {
		q.Set("details", "true")
		q.Del("includeStatusHistory")
	}, nil)
	if err != nil {
		writeProxyError(w, err)
		return
	}
	if baseResp.StatusCode != http.StatusOK {
		h.svc.WriteUpstreamResponse(w, baseResp)
		return
	}
	if _, owned, ownershipErr := OwnedConsentGroupID(baseResp.Body, userID); ownershipErr != nil {
		writeProxyError(w, ownershipErr)
		return
	} else if !owned {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent not found")
		return
	}

	payload, trustedGroupID, err := h.svc.BuildApprovalUpdatePayload(baseResp.Body, selections, userID)
	if err != nil {
		if errors.Is(err, proxy.ErrUpstreamTimeout) || errors.Is(err, proxy.ErrUpstreamUnavailable) || errors.Is(err, proxy.ErrUpstreamResponseTooLarge) {
			writeProxyError(w, err)
			return
		}
		writeJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload")
		return
	}
	if err := h.svc.ForwardWithGroupID(w, r, http.MethodPut, "/api/v1/consents/"+url.PathEscape(consentID), nil, payload, trustedGroupID); err != nil {
		writeProxyError(w, err)
	}
}

// ConsentReject handles POST /me/consents/{consentId}/reject.
func (h *Handler) ConsentReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}
	consentID := r.PathValue("consentId")
	if consentID == "" {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent id not found")
		return
	}
	if !isValidConsentID(consentID) {
		writeJSONError(w, http.StatusBadRequest, "INVALID_CONSENT_ID", "invalid consent id")
		return
	}
	body, err := h.readBoundedBody(r)
	if err != nil {
		writeBodyReadError(w, err)
		return
	}
	if err := validateEmptyObjectPayload(body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload")
		return
	}
	baseResp, err := h.svc.ForwardRaw(r, http.MethodGet, "/api/v1/consents/"+url.PathEscape(consentID), func(q url.Values) {
		q.Set("details", "true")
		q.Del("includeStatusHistory")
	}, nil)
	if err != nil {
		writeProxyError(w, err)
		return
	}
	if baseResp.StatusCode != http.StatusOK {
		h.svc.WriteUpstreamResponse(w, baseResp)
		return
	}
	if _, owned, ownershipErr := OwnedConsentGroupID(baseResp.Body, userID); ownershipErr != nil {
		writeProxyError(w, ownershipErr)
		return
	} else if !owned {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent not found")
		return
	}

	payload, trustedGroupID, err := h.svc.BuildRejectionUpdatePayload(baseResp.Body, userID)
	if err != nil {
		if errors.Is(err, errInvalidConsentState) {
			writeJSONError(w, http.StatusConflict, "INVALID_CONSENT_STATE", "only created consents can be rejected")
			return
		}
		writeProxyError(w, err)
		return
	}
	if err := h.svc.ForwardWithGroupID(w, r, http.MethodPut, "/api/v1/consents/"+url.PathEscape(consentID), nil, payload, trustedGroupID); err != nil {
		writeProxyError(w, err)
	}
}

// ConsentRevoke handles POST /me/consents/{consentId}/revoke.
func (h *Handler) ConsentRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}
	consentID := r.PathValue("consentId")
	if consentID == "" {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent id not found")
		return
	}
	if !isValidConsentID(consentID) {
		writeJSONError(w, http.StatusBadRequest, "INVALID_CONSENT_ID", "invalid consent id")
		return
	}
	body, err := h.readBoundedBody(r)
	if err != nil {
		writeBodyReadError(w, err)
		return
	}
	payload, err := buildRevokePayload(body, userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload")
		return
	}
	baseResp, err := h.svc.ForwardRaw(r, http.MethodGet, "/api/v1/consents/"+url.PathEscape(consentID), nil, nil)
	if err != nil {
		writeProxyError(w, err)
		return
	}
	if baseResp.StatusCode != http.StatusOK {
		h.svc.WriteUpstreamResponse(w, baseResp)
		return
	}
	trustedGroupID, owned, err := OwnedConsentGroupID(baseResp.Body, userID)
	if err != nil {
		writeProxyError(w, err)
		return
	}
	if !owned {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent not found")
		return
	}
	if err := h.svc.ForwardWithGroupID(w, r, http.MethodPost, "/api/v1/consents/"+url.PathEscape(consentID)+"/revoke", nil, payload, trustedGroupID); err != nil {
		writeProxyError(w, err)
	}
}

func isValidConsentID(id string) bool {
	return consentIDPattern.MatchString(strings.TrimSpace(id))
}

func writeProxyError(w http.ResponseWriter, err error) {
	if errors.Is(err, proxy.ErrUpstreamTimeout) {
		writeJSONError(w, http.StatusGatewayTimeout, "UPSTREAM_TIMEOUT", "upstream timeout")
		return
	}
	if errors.Is(err, proxy.ErrUpstreamResponseTooLarge) {
		writeJSONError(w, http.StatusBadGateway, "UPSTREAM_RESPONSE_TOO_LARGE", "upstream response too large")
		return
	}
	writeJSONError(w, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "upstream unavailable")
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Code: code, Message: message})
}

func writeBodyReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, errRequestBodyTooLarge) {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request entity too large")
		return
	}
	writeJSONError(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body")
}

func validateEmptyObjectPayload(body []byte) error {
	if strings.TrimSpace(string(body)) == "" {
		return nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil || len(payload) > 0 {
		return errors.New("expected an empty JSON object")
	}
	return nil
}

func (h *Handler) resolveUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	principal, ok := systemcontext.PrincipalFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusServiceUnavailable, "PLACEHOLDER_UNAVAILABLE", "placeholder identity unavailable")
		return "", false
	}
	return principal.UserID, true
}

func (h *Handler) readBoundedBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer func() {
		_ = r.Body.Close()
	}()
	limited := io.LimitReader(r.Body, h.cfg.MaxRequestBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > h.cfg.MaxRequestBytes {
		return nil, errRequestBodyTooLarge
	}
	return body, nil
}
