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

package proxy

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/wso2/openfgc/portal/backend/internal/config"
	"github.com/wso2/openfgc/portal/backend/internal/middleware"
)

// Handler serves /api and /me route groups for Phase 2.
type Handler struct {
	svc *Service
	cfg config.ProxyConfig
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type consentRetrievalResponse struct {
	ID                         string                      `json:"id"`
	Purposes                   []consentPurposeItem        `json:"purposes"`
	CreatedTime                int64                       `json:"createdTime"`
	UpdatedTime                int64                       `json:"updatedTime"`
	ClientID                   string                      `json:"clientId"`
	Type                       string                      `json:"type"`
	Status                     string                      `json:"status"`
	Frequency                  *int                        `json:"frequency,omitempty"`
	ValidityTime               *int64                      `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                       `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                      `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]any              `json:"attributes,omitempty"`
	Authorizations             []consentAuthorizationEntry `json:"authorizations,omitempty"`
}

type consentPurposeItem struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Elements    []consentElementItem `json:"elements"`
}

type consentElementItem struct {
	Name           string         `json:"name"`
	IsUserApproved bool           `json:"isUserApproved"`
	Value          any            `json:"value,omitempty"`
	IsMandatory    *bool          `json:"isMandatory,omitempty"`
	Type           string         `json:"type,omitempty"`
	Description    string         `json:"description,omitempty"`
	Properties     map[string]any `json:"properties,omitempty"`
}

type consentAuthorizationEntry struct {
	ID          string  `json:"id"`
	UserID      *string `json:"userId,omitempty"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	UpdatedTime int64   `json:"updatedTime"`
	Resources   any     `json:"resources,omitempty"`
}

type consentApprovalSelection struct {
	PurposeName string `json:"purposeName"`
	ElementName string `json:"elementName"`
}

type consentAuthorizationPayload struct {
	UserID    *string `json:"userId,omitempty"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	Resources any     `json:"resources"`
}

type consentUpdatePayload struct {
	Type                       string                        `json:"type"`
	ValidityTime               *int64                        `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                         `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                        `json:"dataAccessValidityDuration,omitempty"`
	Frequency                  *int                          `json:"frequency,omitempty"`
	Purposes                   []consentPurposeItem          `json:"purposes"`
	Attributes                 map[string]any                `json:"attributes,omitempty"`
	Authorizations             []consentAuthorizationPayload `json:"authorizations,omitempty"`
}

func toConsentApprovalKey(purposeName, elementName string) string {
	return purposeName + "::" + elementName
}

type purposeListResponse struct {
	Data []purposeMetadata `json:"data"`
}

type purposeMetadata struct {
	ClientID    string                `json:"clientId"`
	Name        string                `json:"name"`
	Description *string               `json:"description"`
	Elements    []purposeElementEntry `json:"elements"`
}

type purposeElementEntry struct {
	Name        string `json:"name"`
	IsMandatory bool   `json:"isMandatory"`
}

type elementListResponse struct {
	Data []elementMetadata `json:"data"`
}

type elementMetadata struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description *string        `json:"description"`
	Properties  map[string]any `json:"properties"`
}

// NewHandler creates a proxy handler with initialized service.
func NewHandler(cfg config.ProxyConfig) (*Handler, error) {
	svc, err := NewService(cfg)
	if err != nil {
		return nil, err
	}
	return &Handler{svc: svc, cfg: cfg}, nil
}

// API proxies passthrough /api/* routes to /api/v1/* after allowlist checks.
func (h *Handler) API(w http.ResponseWriter, r *http.Request) {
	if !h.svc.IsAllowedPassthroughMethod(r.Method) {
		writeJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	knownPath, methodAllowed := h.svc.CheckAPIAccess(r.Method, r.URL.Path)
	if !knownPath {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	if !methodAllowed {
		writeJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	path, ok := strings.CutPrefix(r.URL.Path, "/api")
	if !ok {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	if path == "" {
		path = "/"
	}
	body, err := h.readBoundedBody(r)
	if err != nil {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request entity too large")
		return
	}
	if err := h.svc.Forward(w, r, r.Method, "/api/v1"+path, nil, body); err != nil {
		h.writeProxyError(w, err)
	}
}

// MeConsents handles GET /me/consents.
func (h *Handler) MeConsents(w http.ResponseWriter, r *http.Request) {
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
	}, nil); err != nil {
		h.writeProxyError(w, err)
	}
}

// MeConsentByID handles GET /me/consents/{consentId}.
func (h *Handler) MeConsentByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	if _, ok := h.resolveUserID(w, r); !ok {
		return
	}
	consentID := r.PathValue("consentId")
	if consentID == "" {
		writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "consent id not found")
		return
	}

	baseResp, err := h.svc.ForwardRaw(r, http.MethodGet, "/api/v1/consents/"+consentID, nil, nil)
	if err != nil {
		h.writeProxyError(w, err)
		return
	}
	if baseResp.StatusCode != http.StatusOK {
		h.writeUpstreamResponse(w, baseResp)
		return
	}

	aggregatedBody, err := h.buildAggregatedConsentResponse(r, baseResp.Body)
	if err != nil {
		h.writeProxyError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(aggregatedBody)
}

// MeConsentApprove handles POST /me/consents/{consentId}/approve.
func (h *Handler) MeConsentApprove(w http.ResponseWriter, r *http.Request) {
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
	body, err := h.readBoundedBody(r)
	if err != nil {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request entity too large")
		return
	}
	selections, err := parseApprovalSelections(body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload")
		return
	}
	baseResp, err := h.svc.ForwardRaw(r, http.MethodGet, "/api/v1/consents/"+consentID, nil, nil)
	if err != nil {
		h.writeProxyError(w, err)
		return
	}
	if baseResp.StatusCode != http.StatusOK {
		h.writeUpstreamResponse(w, baseResp)
		return
	}

	payload, trustedClientID, err := h.buildApprovalUpdatePayload(r, baseResp.Body, selections, userID)
	if err != nil {
		if errors.Is(err, ErrUpstreamTimeout) || errors.Is(err, ErrUpstreamUnavailable) {
			h.writeProxyError(w, err)
			return
		}
		writeJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload")
		return
	}
	if err := h.svc.ForwardWithClientID(w, r, http.MethodPut, "/api/v1/consents/"+consentID, nil, payload, trustedClientID); err != nil {
		h.writeProxyError(w, err)
	}
}

// MeConsentRevoke handles PUT /me/consents/{consentId}/revoke.
func (h *Handler) MeConsentRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
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
	body, err := h.readBoundedBody(r)
	if err != nil {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request entity too large")
		return
	}
	payload, err := h.buildRevokePayload(body, userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload")
		return
	}
	if err := h.svc.Forward(w, r, http.MethodPut, "/api/v1/consents/"+consentID+"/revoke", nil, payload); err != nil {
		h.writeProxyError(w, err)
	}
}

func (h *Handler) writeProxyError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrUpstreamTimeout) {
		writeJSONError(w, http.StatusServiceUnavailable, "UPSTREAM_TIMEOUT", "upstream timeout")
		return
	}
	writeJSONError(w, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "upstream unavailable")
}

func (h *Handler) writeUpstreamResponse(w http.ResponseWriter, resp *UpstreamResponse) {
	h.svc.copyResponseHeaders(w.Header(), resp.Headers)
	w.WriteHeader(resp.StatusCode)
	if len(resp.Body) > 0 {
		_, _ = w.Write(resp.Body)
	}
}

func (h *Handler) buildAggregatedConsentResponse(r *http.Request, baseBody []byte) ([]byte, error) {
	var consent consentRetrievalResponse
	if err := json.Unmarshal(baseBody, &consent); err != nil {
		return nil, ErrUpstreamUnavailable
	}

	purposeMetadataByName := make(map[string]purposeMetadata, len(consent.Purposes))
	for _, purpose := range consent.Purposes {
		if _, exists := purposeMetadataByName[purpose.Name]; exists {
			continue
		}
		metadata, err := h.fetchPurposeMetadata(r, consent.ClientID, purpose)
		if err != nil {
			return nil, err
		}
		purposeMetadataByName[purpose.Name] = metadata
	}

	elementMetadataByName := make(map[string]elementMetadata)
	for _, purpose := range consent.Purposes {
		for _, element := range purpose.Elements {
			if _, exists := elementMetadataByName[element.Name]; exists {
				continue
			}
			metadata, err := h.fetchElementMetadata(r, element.Name)
			if err != nil {
				return nil, err
			}
			elementMetadataByName[element.Name] = metadata
		}
	}

	for purposeIndex := range consent.Purposes {
		purpose := &consent.Purposes[purposeIndex]
		purposeMetadata, exists := purposeMetadataByName[purpose.Name]
		if !exists {
			return nil, ErrUpstreamUnavailable
		}
		if purposeMetadata.Description != nil {
			purpose.Description = *purposeMetadata.Description
		}

		mandatoryByElement := make(map[string]bool, len(purposeMetadata.Elements))
		for _, entry := range purposeMetadata.Elements {
			mandatoryByElement[entry.Name] = entry.IsMandatory
		}

		for elementIndex := range purpose.Elements {
			element := &purpose.Elements[elementIndex]
			mandatory, exists := mandatoryByElement[element.Name]
			if !exists {
				return nil, ErrUpstreamUnavailable
			}
			element.IsMandatory = &mandatory

			elementMetadata, exists := elementMetadataByName[element.Name]
			if !exists {
				return nil, ErrUpstreamUnavailable
			}
			element.Type = elementMetadata.Type
			if elementMetadata.Description != nil {
				element.Description = *elementMetadata.Description
			}
			element.Properties = elementMetadata.Properties
		}
	}

	aggregated, err := json.Marshal(consent)
	if err != nil {
		return nil, ErrUpstreamUnavailable
	}

	return aggregated, nil
}

func (h *Handler) fetchPurposeMetadata(r *http.Request, clientID string, consentPurpose consentPurposeItem) (purposeMetadata, error) {
	exactByClient, err := h.fetchPurposeMetadataPage(r, consentPurpose.Name, clientID)
	if err != nil {
		return purposeMetadata{}, err
	}
	elementNames := make(map[string]struct{}, len(consentPurpose.Elements))
	for _, element := range consentPurpose.Elements {
		elementNames[element.Name] = struct{}{}
	}

	if purpose, ok := selectPurposeCandidate(exactByClient, consentPurpose.Name, clientID, elementNames); ok {
		return purpose, nil
	}

	// Fallback without clientIds filter handles inconsistent historical data.
	exactByName, err := h.fetchPurposeMetadataPage(r, consentPurpose.Name, "")
	if err != nil {
		return purposeMetadata{}, err
	}
	if purpose, ok := selectPurposeCandidate(exactByName, consentPurpose.Name, clientID, elementNames); ok {
		return purpose, nil
	}

	return purposeMetadata{}, ErrUpstreamUnavailable
}

func (h *Handler) fetchPurposeMetadataPage(r *http.Request, purposeName, clientID string) ([]purposeMetadata, error) {
	resp, err := h.svc.ForwardRaw(r, http.MethodGet, "/api/v1/consent-purposes", func(q url.Values) {
		q.Set("name", purposeName)
		if clientID != "" {
			q.Set("clientIds", clientID)
		}
		q.Set("limit", "50")
		q.Set("offset", "0")
	}, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrUpstreamUnavailable
	}

	var payload purposeListResponse
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return nil, ErrUpstreamUnavailable
	}

	return payload.Data, nil
}

func selectPurposeCandidate(candidates []purposeMetadata, purposeName, clientID string, requiredElements map[string]struct{}) (purposeMetadata, bool) {
	matchingName := make([]purposeMetadata, 0, len(candidates))
	for _, purpose := range candidates {
		if purpose.Name != purposeName {
			continue
		}
		matchingName = append(matchingName, purpose)
	}

	if len(matchingName) == 0 {
		return purposeMetadata{}, false
	}

	best := make([]purposeMetadata, 0, len(matchingName))
	for _, purpose := range matchingName {
		if purposeContainsAllElements(purpose, requiredElements) {
			best = append(best, purpose)
		}
	}
	if len(best) == 0 {
		best = matchingName
	}

	if clientID != "" {
		for _, purpose := range best {
			if purpose.ClientID == clientID {
				return purpose, true
			}
		}
	}

	return best[0], true
}

func purposeContainsAllElements(purpose purposeMetadata, requiredElements map[string]struct{}) bool {
	if len(requiredElements) == 0 {
		return true
	}
	purposeElements := make(map[string]struct{}, len(purpose.Elements))
	for _, element := range purpose.Elements {
		purposeElements[element.Name] = struct{}{}
	}
	for name := range requiredElements {
		if _, ok := purposeElements[name]; !ok {
			return false
		}
	}
	return true
}

func (h *Handler) fetchElementMetadata(r *http.Request, elementName string) (elementMetadata, error) {
	resp, err := h.svc.ForwardRaw(r, http.MethodGet, "/api/v1/consent-elements", func(q url.Values) {
		q.Set("name", elementName)
		q.Set("limit", strconv.Itoa(50))
		q.Set("offset", "0")
	}, nil)
	if err != nil {
		return elementMetadata{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return elementMetadata{}, ErrUpstreamUnavailable
	}

	var payload elementListResponse
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return elementMetadata{}, ErrUpstreamUnavailable
	}
	for _, element := range payload.Data {
		if element.Name == elementName {
			return element, nil
		}
	}

	return elementMetadata{}, ErrUpstreamUnavailable
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Code: code, Message: message})
}

func (h *Handler) resolveUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusServiceUnavailable, "PLACEHOLDER_UNAVAILABLE", "placeholder identity unavailable")
		return "", false
	}
	return userID, true
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
		return nil, errors.New("body too large")
	}
	return body, nil
}

func (h *Handler) buildRevokePayload(in []byte, userID string) ([]byte, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("missing actionBy")
	}

	payload := map[string]any{
		"actionBy": userID,
	}
	if len(in) > 0 {
		if err := json.Unmarshal(in, &payload); err != nil {
			return nil, err
		}
		if payload == nil {
			payload = map[string]any{}
		}
	}
	payload["actionBy"] = userID
	return json.Marshal(payload)
}

func parseApprovalSelections(in []byte) ([]consentApprovalSelection, error) {
	if len(in) == 0 {
		return nil, nil
	}
	var selections []consentApprovalSelection
	if err := json.Unmarshal(in, &selections); err != nil {
		return nil, err
	}
	for _, selection := range selections {
		if strings.TrimSpace(selection.PurposeName) == "" || strings.TrimSpace(selection.ElementName) == "" {
			return nil, errors.New("invalid approval selection")
		}
	}
	return selections, nil
}

func (h *Handler) buildApprovalUpdatePayload(r *http.Request, baseBody []byte, selections []consentApprovalSelection, userID string) ([]byte, string, error) {
	var consent consentRetrievalResponse
	if err := json.Unmarshal(baseBody, &consent); err != nil {
		return nil, "", ErrUpstreamUnavailable
	}

	if err := h.enrichMandatoryFlags(r, &consent); err != nil {
		return nil, "", err
	}

	selectedOptionalElements := make(map[string]struct{}, len(selections))
	for _, selection := range selections {
		selectedOptionalElements[toConsentApprovalKey(selection.PurposeName, selection.ElementName)] = struct{}{}
	}

	matchedSelections := make(map[string]struct{}, len(selectedOptionalElements))
	updatedPurposes := make([]consentPurposeItem, len(consent.Purposes))
	for purposeIndex, purpose := range consent.Purposes {
		updatedPurpose := purpose
		updatedPurpose.Elements = make([]consentElementItem, len(purpose.Elements))
		for elementIndex, element := range purpose.Elements {
			updatedElement := element
			if element.IsMandatory != nil && *element.IsMandatory {
				updatedElement.IsUserApproved = true
			} else {
				key := toConsentApprovalKey(purpose.Name, element.Name)
				_, updatedElement.IsUserApproved = selectedOptionalElements[key]
				if updatedElement.IsUserApproved {
					matchedSelections[key] = struct{}{}
				}
			}
			updatedPurpose.Elements[elementIndex] = updatedElement
		}
		updatedPurposes[purposeIndex] = updatedPurpose
	}

	if len(matchedSelections) != len(selectedOptionalElements) {
		return nil, "", errors.New("invalid approval selection")
	}

	updatedAuthorizations := make([]consentAuthorizationPayload, 0, len(consent.Authorizations))
	for _, authorization := range consent.Authorizations {
		updatedAuthorizations = append(updatedAuthorizations, consentAuthorizationPayload{
			UserID:    authorization.UserID,
			Type:      authorization.Type,
			Status:    authorization.Status,
			Resources: normalizeAuthorizationResources(authorization.Resources),
		})
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, "", ErrUpstreamUnavailable
	}
	updatedAuthorization := consentAuthorizationPayload{
		UserID:    &userID,
		Type:      "authorisation",
		Status:    "APPROVED",
		Resources: map[string]any{},
	}

	if index, ok := findAuthorizationIndexToUpdate(consent.Authorizations, userID); ok {
		updatedAuthorizations[index] = updatedAuthorization
	} else {
		updatedAuthorizations = append(updatedAuthorizations, updatedAuthorization)
	}

	payload := consentUpdatePayload{
		Type:                       consent.Type,
		ValidityTime:               consent.ValidityTime,
		RecurringIndicator:         consent.RecurringIndicator,
		DataAccessValidityDuration: consent.DataAccessValidityDuration,
		Frequency:                  consent.Frequency,
		Purposes:                   updatedPurposes,
		Attributes:                 consent.Attributes,
		Authorizations:             updatedAuthorizations,
	}

	serializedPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, "", ErrUpstreamUnavailable
	}

	return serializedPayload, consent.ClientID, nil
}

func (h *Handler) enrichMandatoryFlags(r *http.Request, consent *consentRetrievalResponse) error {
	purposeMetadataByName := make(map[string]purposeMetadata, len(consent.Purposes))
	for _, purpose := range consent.Purposes {
		if _, exists := purposeMetadataByName[purpose.Name]; exists {
			continue
		}
		metadata, err := h.fetchPurposeMetadata(r, consent.ClientID, purpose)
		if err != nil {
			return err
		}
		purposeMetadataByName[purpose.Name] = metadata
	}

	for purposeIndex := range consent.Purposes {
		purpose := &consent.Purposes[purposeIndex]
		purposeMetadata, exists := purposeMetadataByName[purpose.Name]
		if !exists {
			return ErrUpstreamUnavailable
		}

		mandatoryByElement := make(map[string]bool, len(purposeMetadata.Elements))
		for _, entry := range purposeMetadata.Elements {
			mandatoryByElement[entry.Name] = entry.IsMandatory
		}

		for elementIndex := range purpose.Elements {
			element := &purpose.Elements[elementIndex]
			mandatory, exists := mandatoryByElement[element.Name]
			if !exists {
				return ErrUpstreamUnavailable
			}
			element.IsMandatory = &mandatory
		}
	}

	return nil
}

func normalizeAuthorizationResources(resources any) any {
	if resources == nil {
		return map[string]any{}
	}
	return resources
}

func findAuthorizationIndexToUpdate(authorizations []consentAuthorizationEntry, userID string) (int, bool) {
	if len(authorizations) == 0 {
		return -1, false
	}

	for index, authorization := range authorizations {
		if authorization.UserID != nil && strings.EqualFold(strings.TrimSpace(*authorization.UserID), userID) {
			return index, true
		}
	}

	latestIndex := 0
	latestUpdatedTime := authorizations[0].UpdatedTime
	for index := 1; index < len(authorizations); index++ {
		if authorizations[index].UpdatedTime > latestUpdatedTime {
			latestUpdatedTime = authorizations[index].UpdatedTime
			latestIndex = index
		}
	}

	return latestIndex, true
}
