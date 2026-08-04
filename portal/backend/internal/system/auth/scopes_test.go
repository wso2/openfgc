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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestValidateConfiguredScopesSupportsEmptyPrefix(t *testing.T) {
	if err := validateConfiguredScopesWithPrefix(
		[]string{"openid", "consents:read:self"},
		"",
		[]string{"consents:read:self"},
	); err != nil {
		t.Fatalf("empty scope prefix should be supported: %v", err)
	}
}

func TestValidateConfiguredScopesRejectsUnknownPrefixedScope(t *testing.T) {
	err := validateConfiguredScopesWithPrefix(
		[]string{"openid", "portal:unknown"},
		"portal:",
		[]string{"portal:consents:read:self"},
	)
	if err == nil {
		t.Fatal("expected an unknown prefixed scope to be rejected")
	}
}

func TestCanonicalScopesAreUniqueAndPrefixed(t *testing.T) {
	seen := make(map[string]struct{}, len(AllPortalScopes))
	for _, scope := range AllPortalScopes {
		if strings.TrimSpace(scope) == "" {
			t.Fatal("canonical scope must not be empty")
		}
		if ScopePrefix != "" && !strings.HasPrefix(scope, ScopePrefix) {
			t.Errorf("scope %q does not use prefix %q", scope, ScopePrefix)
		}
		if _, duplicate := seen[scope]; duplicate {
			t.Errorf("duplicate canonical scope %q", scope)
		}
		seen[scope] = struct{}{}
	}
	if len(seen) != 8 {
		t.Fatalf("expected eight canonical scopes, got %d", len(seen))
	}
}

func TestGrantedPortalScopesFiltersAndPreservesCanonicalOrder(t *testing.T) {
	granted := map[string]struct{}{
		"openid":              {},
		ScopePurposesWrite:    {},
		ScopeConsentsReadSelf: {},
		"portal:future:scope": {},
	}
	want := []string{ScopeConsentsReadSelf, ScopePurposesWrite}
	got := GrantedPortalScopes(granted)
	if len(got) != len(want) {
		t.Fatalf("got %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v; want %v", got, want)
		}
	}
}

func TestEveryAPIRouteHasCanonicalScopePolicy(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/api/consents", ScopeConsentsReadAny},
		{"POST", "/api/consents", ScopeConsentsWriteAny},
		{"GET", "/api/consents/attributes", ScopeConsentsReadAny},
		{"POST", "/api/consents/validate", ScopeConsentsReadAny},
		{"GET", "/api/consents/c1", ScopeConsentsReadAny},
		{"PUT", "/api/consents/c1", ScopeConsentsWriteAny},
		{"GET", "/api/consents/c1/history", ScopeConsentsReadAny},
		{"POST", "/api/consents/c1/revoke", ScopeConsentsWriteAny},
		{"GET", "/api/consents/c1/authorizations", ScopeConsentsReadAny},
		{"POST", "/api/consents/c1/authorizations", ScopeConsentsWriteAny},
		{"GET", "/api/consents/c1/authorizations/a1", ScopeConsentsReadAny},
		{"PUT", "/api/consents/c1/authorizations/a1", ScopeConsentsWriteAny},
		{"GET", "/api/consent-elements", ScopeElementsRead},
		{"POST", "/api/consent-elements", ScopeElementsWrite},
		{"GET", "/api/consent-elements/e1", ScopeElementsRead},
		{"GET", "/api/consent-elements/e1/versions", ScopeElementsRead},
		{"POST", "/api/consent-elements/e1/versions", ScopeElementsWrite},
		{"GET", "/api/consent-elements/e1/versions/v1", ScopeElementsRead},
		{"DELETE", "/api/consent-elements/e1/versions/v1", ScopeElementsWrite},
		{"GET", "/api/consent-purposes", ScopePurposesRead},
		{"POST", "/api/consent-purposes", ScopePurposesWrite},
		{"GET", "/api/consent-purposes/p1", ScopePurposesRead},
		{"GET", "/api/consent-purposes/p1/versions", ScopePurposesRead},
		{"POST", "/api/consent-purposes/p1/versions", ScopePurposesWrite},
		{"GET", "/api/consent-purposes/p1/versions/v1", ScopePurposesRead},
		{"DELETE", "/api/consent-purposes/p1/versions/v1", ScopePurposesWrite},
	}
	known := make(map[string]struct{}, len(AllPortalScopes))
	for _, scope := range AllPortalScopes {
		known[scope] = struct{}{}
	}
	for _, test := range tests {
		got, ok := ScopeForAPIRequest(test.method, test.path)
		if !ok || got != test.want {
			t.Errorf("%s %s: got %q, %v; want %q", test.method, test.path, got, ok, test.want)
		}
		if _, ok := known[got]; !ok {
			t.Errorf("%s %s references unregistered scope %q", test.method, test.path, got)
		}
	}
	for _, test := range []struct{ method, path string }{
		{"PATCH", "/api/consents/c1"},
		{"GET", "/api/unknown"},
		{"GET", "/not-api/consents"},
		{"POST", "/api/consents/c1/export"},
		{"DELETE", "/api/consent-elements/e1"},
	} {
		if scope, ok := ScopeForAPIRequest(test.method, test.path); ok {
			t.Errorf("unexpected policy for %s %s: %q", test.method, test.path, scope)
		}
	}
}

func TestKnownAPIPathIgnoresMethod(t *testing.T) {
	if !isKnownAPIPath("/api/consents/c1/revoke") {
		t.Fatal("documented API path should be known")
	}
	if isKnownAPIPath("/api/consents/c1/export") {
		t.Fatal("undocumented API path should not be known")
	}
}

func TestOpenAPIScopesMatchCanonicalRoutePolicies(t *testing.T) {
	path := filepath.Join("..", "..", "..", "openapi", "portal-backend.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI document: %v", err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatalf("parse OpenAPI document: %v", err)
	}

	known := make(map[string]struct{}, len(AllPortalScopes))
	for _, scope := range AllPortalScopes {
		known[scope] = struct{}{}
	}
	components := mustMap(t, document["components"], "components")
	schemes := mustMap(t, components["securitySchemes"], "securitySchemes")
	oauth := mustMap(t, schemes["PortalOAuthDocumentation"], "PortalOAuthDocumentation")
	flows := mustMap(t, oauth["flows"], "flows")
	authorizationCode := mustMap(t, flows["authorizationCode"], "authorizationCode")
	documentedScopes := mustMap(t, authorizationCode["scopes"], "scopes")
	if len(documentedScopes) != len(known) {
		t.Fatalf("OpenAPI documents %d scopes; registry has %d", len(documentedScopes), len(known))
	}
	for scope := range documentedScopes {
		if _, ok := known[scope]; !ok {
			t.Errorf("OpenAPI documents unknown scope %q", scope)
		}
	}

	paths := mustMap(t, document["paths"], "paths")
	placeholder := regexp.MustCompile(`\{[^}]+\}`)
	for route, rawPathItem := range paths {
		pathItem := mustMap(t, rawPathItem, route)
		for method, rawOperation := range pathItem {
			upperMethod := strings.ToUpper(method)
			if !isHTTPMethod(upperMethod) {
				continue
			}
			operation := mustMap(t, rawOperation, upperMethod+" "+route)
			scopes := operationOAuthScopes(t, operation)
			for _, scope := range scopes {
				if _, ok := known[scope]; !ok {
					t.Errorf("%s %s uses unknown scope %q", upperMethod, route, scope)
				}
			}
			samplePath := placeholder.ReplaceAllString(route, "sample")
			var want string
			switch {
			case strings.HasPrefix(route, "/api/"):
				policy, ok := ScopeForAPIRequest(upperMethod, samplePath)
				if !ok {
					t.Errorf("OpenAPI operation lacks code policy: %s %s", upperMethod, route)
					continue
				}
				want = policy
			case route == "/me":
				if len(scopes) != 0 {
					t.Errorf("%s %s documents %v; want no business scope", upperMethod, route, scopes)
				}
				continue
			case strings.HasPrefix(route, "/me/"):
				if upperMethod == "GET" {
					want = ScopeConsentsReadSelf
				} else {
					want = ScopeConsentsWriteSelf
				}
			default:
				continue
			}
			if len(scopes) != 1 || scopes[0] != want {
				t.Errorf("%s %s documents %v; code requires %q", upperMethod, route, scopes, want)
			}
		}
	}
}

func operationOAuthScopes(t *testing.T, operation map[string]any) []string {
	t.Helper()
	rawSecurity, ok := operation["security"].([]any)
	if !ok {
		return nil
	}
	for _, rawRequirement := range rawSecurity {
		requirement := mustMap(t, rawRequirement, "security requirement")
		rawScopes, ok := requirement["PortalOAuthDocumentation"].([]any)
		if !ok {
			continue
		}
		result := make([]string, 0, len(rawScopes))
		for _, rawScope := range rawScopes {
			scope, ok := rawScope.(string)
			if !ok {
				t.Fatalf("non-string OAuth scope: %#v", rawScope)
			}
			result = append(result, scope)
		}
		return result
	}
	return nil
}

func mustMap(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object: %#v", name, value)
	}
	return result
}

func isHTTPMethod(value string) bool {
	switch value {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE":
		return true
	default:
		return false
	}
}
