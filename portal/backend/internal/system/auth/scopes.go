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

package auth

import (
	"net/http"
	"strings"
)

// ScopePrefix and the remaining constants define the canonical portal authorization scopes.
const (
	// ScopePrefix is the namespace shared by all portal-owned scopes.
	// Set it to an empty string when the identity provider uses unprefixed scopes.
	ScopePrefix = "portal:"

	ScopeConsentsReadSelf  = ScopePrefix + "consents:read:self"
	ScopeConsentsWriteSelf = ScopePrefix + "consents:write:self"
	ScopeConsentsReadAny   = ScopePrefix + "consents:read:any"
	ScopeConsentsWriteAny  = ScopePrefix + "consents:write:any"
	ScopeElementsRead      = ScopePrefix + "elements:read"
	ScopeElementsWrite     = ScopePrefix + "elements:write"
	ScopePurposesRead      = ScopePrefix + "purposes:read"
	ScopePurposesWrite     = ScopePrefix + "purposes:write"
)

// AllPortalScopes lists every canonical portal authorization scope.
var AllPortalScopes = []string{
	ScopeConsentsReadSelf, ScopeConsentsWriteSelf,
	ScopeConsentsReadAny, ScopeConsentsWriteAny,
	ScopeElementsRead, ScopeElementsWrite,
	ScopePurposesRead, ScopePurposesWrite,
}

// GrantedPortalScopes returns the canonical portal scopes present in granted,
// preserving the stable order of AllPortalScopes.
func GrantedPortalScopes(granted map[string]struct{}) []string {
	result := make([]string, 0, len(AllPortalScopes))
	for _, scope := range AllPortalScopes {
		if _, ok := granted[scope]; ok {
			result = append(result, scope)
		}
	}
	return result
}

type apiScopePolicy struct {
	method, pattern, scope string
}

// apiScopePolicies explicitly allowlists operations exposed through the
// catch-all API proxy. New upstream operations require a deliberate policy.
var apiScopePolicies = []apiScopePolicy{
	{http.MethodGet, "/api/consents", ScopeConsentsReadAny},
	{http.MethodPost, "/api/consents", ScopeConsentsWriteAny},
	{http.MethodGet, "/api/consents/attributes", ScopeConsentsReadAny},
	{http.MethodPost, "/api/consents/validate", ScopeConsentsReadAny},
	{http.MethodGet, "/api/consents/{consentId}", ScopeConsentsReadAny},
	{http.MethodPut, "/api/consents/{consentId}", ScopeConsentsWriteAny},
	{http.MethodGet, "/api/consents/{consentId}/history", ScopeConsentsReadAny},
	{http.MethodPost, "/api/consents/{consentId}/revoke", ScopeConsentsWriteAny},
	{http.MethodGet, "/api/consents/{consentId}/authorizations", ScopeConsentsReadAny},
	{http.MethodPost, "/api/consents/{consentId}/authorizations", ScopeConsentsWriteAny},
	{http.MethodGet, "/api/consents/{consentId}/authorizations/{authorizationId}", ScopeConsentsReadAny},
	{http.MethodPut, "/api/consents/{consentId}/authorizations/{authorizationId}", ScopeConsentsWriteAny},
	{http.MethodGet, "/api/consent-elements", ScopeElementsRead},
	{http.MethodPost, "/api/consent-elements", ScopeElementsWrite},
	{http.MethodGet, "/api/consent-elements/{elementId}", ScopeElementsRead},
	{http.MethodGet, "/api/consent-elements/{elementId}/versions", ScopeElementsRead},
	{http.MethodPost, "/api/consent-elements/{elementId}/versions", ScopeElementsWrite},
	{http.MethodGet, "/api/consent-elements/{elementId}/versions/{version}", ScopeElementsRead},
	{http.MethodDelete, "/api/consent-elements/{elementId}/versions/{version}", ScopeElementsWrite},
	{http.MethodGet, "/api/consent-purposes", ScopePurposesRead},
	{http.MethodPost, "/api/consent-purposes", ScopePurposesWrite},
	{http.MethodGet, "/api/consent-purposes/{purposeId}", ScopePurposesRead},
	{http.MethodGet, "/api/consent-purposes/{purposeId}/versions", ScopePurposesRead},
	{http.MethodPost, "/api/consent-purposes/{purposeId}/versions", ScopePurposesWrite},
	{http.MethodGet, "/api/consent-purposes/{purposeId}/versions/{version}", ScopePurposesRead},
	{http.MethodDelete, "/api/consent-purposes/{purposeId}/versions/{version}", ScopePurposesWrite},
}

// ScopeForAPIRequest returns the canonical scope for an explicitly allowlisted
// API operation.
func ScopeForAPIRequest(method, path string) (string, bool) {
	method = strings.ToUpper(method)
	for _, policy := range apiScopePolicies {
		if method == policy.method && matchAPIPath(policy.pattern, path) {
			return policy.scope, true
		}
	}
	return "", false
}

func isKnownAPIPath(path string) bool {
	for _, policy := range apiScopePolicies {
		if matchAPIPath(policy.pattern, path) {
			return true
		}
	}
	return false
}

func matchAPIPath(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for i, patternPart := range patternParts {
		placeholder := strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}")
		if (!placeholder && patternPart != pathParts[i]) || pathParts[i] == "" {
			return false
		}
	}
	return true
}
