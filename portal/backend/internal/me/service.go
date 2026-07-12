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

package me

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/wso2/openfgc/portal/backend/internal/proxy"
	"github.com/wso2/openfgc/portal/backend/internal/system/config"
)

// Service contains /me endpoint logic layered on top of proxy transport.
type Service struct {
	proxy *proxy.Service
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

// NewService builds a me service from app config.
func NewService(cfg config.ProxyConfig) (*Service, error) {
	svc, err := proxy.NewService(cfg)
	if err != nil {
		return nil, err
	}
	return &Service{proxy: svc}, nil
}

// Forward forwards to upstream via the shared proxy service.
func (s *Service) Forward(w http.ResponseWriter, r *http.Request, upstreamMethod, upstreamPath string, queryMutator func(url.Values), body []byte) error {
	return s.proxy.Forward(w, r, upstreamMethod, upstreamPath, queryMutator, body)
}

// ForwardWithClientID forwards to upstream with a trusted client ID.
func (s *Service) ForwardWithClientID(w http.ResponseWriter, r *http.Request, upstreamMethod, upstreamPath string, queryMutator func(url.Values), body []byte, trustedClientID string) error {
	return s.proxy.ForwardWithClientID(w, r, upstreamMethod, upstreamPath, queryMutator, body, trustedClientID)
}

// ForwardRaw forwards to upstream and returns a caller-managed response.
func (s *Service) ForwardRaw(r *http.Request, upstreamMethod, upstreamPath string, queryMutator func(url.Values), body []byte) (*proxy.UpstreamResponse, error) {
	return s.proxy.ForwardRaw(r, upstreamMethod, upstreamPath, queryMutator, body)
}

// WriteUpstreamResponse writes an upstream response back to the caller.
func (s *Service) WriteUpstreamResponse(w http.ResponseWriter, resp *proxy.UpstreamResponse) {
	_ = s.proxy.WriteUpstreamResponse(w, resp)
}

func toConsentApprovalKey(purposeName, elementName string) string {
	return purposeName + "::" + elementName
}

// BuildAggregatedConsentResponse enriches a consent response with purpose and element metadata.
func (s *Service) BuildAggregatedConsentResponse(r *http.Request, baseBody []byte) ([]byte, error) {
	var consent consentRetrievalResponse
	if err := json.Unmarshal(baseBody, &consent); err != nil {
		return nil, proxy.ErrUpstreamUnavailable
	}

	purposeMetadataByName := make(map[string]purposeMetadata, len(consent.Purposes))
	for _, purpose := range consent.Purposes {
		if _, exists := purposeMetadataByName[purpose.Name]; exists {
			continue
		}
		metadata, err := s.fetchPurposeMetadata(r, consent.ClientID, purpose)
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
			metadata, err := s.fetchElementMetadata(r, element.Name)
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
			return nil, proxy.ErrUpstreamUnavailable
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
				return nil, proxy.ErrUpstreamUnavailable
			}
			element.IsMandatory = &mandatory

			elementMetadata, exists := elementMetadataByName[element.Name]
			if !exists {
				return nil, proxy.ErrUpstreamUnavailable
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
		return nil, proxy.ErrUpstreamUnavailable
	}

	return aggregated, nil
}

// IsConsentOwnedByUser reports whether the fetched consent contains an
// authorization for the effective user. It is used before returning or
// changing a consent through a self-scoped /me route.
func IsConsentOwnedByUser(baseBody []byte, userID string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}

	var consent consentRetrievalResponse
	if err := json.Unmarshal(baseBody, &consent); err != nil {
		return false
	}

	for _, authorization := range consent.Authorizations {
		if authorization.UserID != nil && strings.EqualFold(strings.TrimSpace(*authorization.UserID), userID) {
			return true
		}
	}

	return false
}

func (s *Service) fetchPurposeMetadata(r *http.Request, clientID string, consentPurpose consentPurposeItem) (purposeMetadata, error) {
	exactByClient, err := s.fetchPurposeMetadataPage(r, consentPurpose.Name, clientID)
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

	exactByName, err := s.fetchPurposeMetadataPage(r, consentPurpose.Name, "")
	if err != nil {
		return purposeMetadata{}, err
	}
	if purpose, ok := selectPurposeCandidate(exactByName, consentPurpose.Name, clientID, elementNames); ok {
		return purpose, nil
	}

	return purposeMetadata{}, proxy.ErrUpstreamUnavailable
}

func (s *Service) fetchPurposeMetadataPage(r *http.Request, purposeName, clientID string) ([]purposeMetadata, error) {
	resp, err := s.proxy.ForwardRaw(r, http.MethodGet, "/api/v1/consent-purposes", func(q url.Values) {
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
		return nil, proxy.ErrUpstreamUnavailable
	}

	var payload purposeListResponse
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return nil, proxy.ErrUpstreamUnavailable
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

func (s *Service) fetchElementMetadata(r *http.Request, elementName string) (elementMetadata, error) {
	resp, err := s.proxy.ForwardRaw(r, http.MethodGet, "/api/v1/consent-elements", func(q url.Values) {
		q.Set("name", elementName)
		q.Set("limit", strconv.Itoa(50))
		q.Set("offset", "0")
	}, nil)
	if err != nil {
		return elementMetadata{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return elementMetadata{}, proxy.ErrUpstreamUnavailable
	}

	var payload elementListResponse
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return elementMetadata{}, proxy.ErrUpstreamUnavailable
	}
	for _, element := range payload.Data {
		if element.Name == elementName {
			return element, nil
		}
	}

	return elementMetadata{}, proxy.ErrUpstreamUnavailable
}

func buildRevokePayload(in []byte, userID string) ([]byte, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("missing actionBy")
	}

	payload := map[string]any{}
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

// BuildApprovalUpdatePayload builds the consent update payload for an approval action.
func (s *Service) BuildApprovalUpdatePayload(r *http.Request, baseBody []byte, selections []consentApprovalSelection, userID string) ([]byte, string, error) {
	var consent consentRetrievalResponse
	if err := json.Unmarshal(baseBody, &consent); err != nil {
		return nil, "", proxy.ErrUpstreamUnavailable
	}

	if err := s.enrichMandatoryFlags(r, &consent); err != nil {
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
				if _, isSelected := selectedOptionalElements[key]; isSelected {
					updatedElement.IsUserApproved = true
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
		return nil, "", proxy.ErrUpstreamUnavailable
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
		return nil, "", proxy.ErrUpstreamUnavailable
	}

	return serializedPayload, consent.ClientID, nil
}

func (s *Service) enrichMandatoryFlags(r *http.Request, consent *consentRetrievalResponse) error {
	purposeMetadataByName := make(map[string]purposeMetadata, len(consent.Purposes))
	for _, purpose := range consent.Purposes {
		if _, exists := purposeMetadataByName[purpose.Name]; exists {
			continue
		}
		metadata, err := s.fetchPurposeMetadata(r, consent.ClientID, purpose)
		if err != nil {
			return err
		}
		purposeMetadataByName[purpose.Name] = metadata
	}

	for purposeIndex := range consent.Purposes {
		purpose := &consent.Purposes[purposeIndex]
		purposeMetadata, exists := purposeMetadataByName[purpose.Name]
		if !exists {
			return proxy.ErrUpstreamUnavailable
		}

		mandatoryByElement := make(map[string]bool, len(purposeMetadata.Elements))
		for _, entry := range purposeMetadata.Elements {
			mandatoryByElement[entry.Name] = entry.IsMandatory
		}

		for elementIndex := range purpose.Elements {
			element := &purpose.Elements[elementIndex]
			mandatory, exists := mandatoryByElement[element.Name]
			if !exists {
				return proxy.ErrUpstreamUnavailable
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
	userID = strings.TrimSpace(userID)
	for index, authorization := range authorizations {
		if authorization.UserID != nil && strings.EqualFold(strings.TrimSpace(*authorization.UserID), userID) {
			return index, true
		}
	}

	return -1, false
}
