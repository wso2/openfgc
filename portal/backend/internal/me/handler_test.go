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
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/wso2/openfgc/portal/backend/internal/system/auth"
	systemcontext "github.com/wso2/openfgc/portal/backend/internal/system/context"
)

func TestCurrentUserReturnsCanonicalPortalScopes(t *testing.T) {
	principal := systemcontext.Principal{
		UserID: "user-1",
		OrgID:  "org-1",
		Scopes: map[string]struct{}{
			"openid":                   {},
			auth.ScopePurposesWrite:    {},
			auth.ScopeConsentsReadSelf: {},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = req.WithContext(systemcontext.WithPrincipal(req.Context(), principal))
	recorder := httptest.NewRecorder()

	(&Handler{}).CurrentUser(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	var response currentUserResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.UserID != "user-1" || response.OrganizationID != "org-1" {
		t.Fatalf("unexpected identity: %+v", response)
	}
	wantScopes := []string{auth.ScopeConsentsReadSelf, auth.ScopePurposesWrite}
	if !reflect.DeepEqual(response.Scopes, wantScopes) {
		t.Fatalf("scopes = %v, want %v", response.Scopes, wantScopes)
	}
}

func TestCurrentUserRequiresPrincipal(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&Handler{}).CurrentUser(recorder, httptest.NewRequest(http.MethodGet, "/me", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
