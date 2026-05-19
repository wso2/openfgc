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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wso2/openfgc/portal/backend/internal/middleware"
)

func TestUserIDMiddleware_InsertsUserIDIntoContext(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user id in context")
		}
		if userID != "user@example.com" {
			t.Fatalf("expected user@example.com, got %s", userID)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.UserID(next, middleware.UserIDOptions{
		PlaceholderModeEnabled: true,
		PlaceholderUserID:      "user@example.com",
	})

	req := httptest.NewRequest(http.MethodGet, "/me/consents", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", res.Code)
	}
}

func TestUserIDMiddleware_Returns503WhenPlaceholderModeDisabled(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.UserID(next, middleware.UserIDOptions{
		PlaceholderModeEnabled: false,
		PlaceholderUserID:      "",
		Environment:            "development",
	})

	req := httptest.NewRequest(http.MethodGet, "/me/consents", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json payload: %v", err)
	}
	if payload["code"] != "IDENTITY_UNAVAILABLE" {
		t.Fatalf("expected IDENTITY_UNAVAILABLE, got %v", payload["code"])
	}
	message, ok := payload["message"].(string)
	if !ok {
		t.Fatalf("expected string message, got %T", payload["message"])
	}
	if message != "identity unavailable; consider enabling placeholder identity mode for development" {
		t.Fatalf("unexpected message: %s", message)
	}
}

func TestUserIDMiddleware_ProductionModeDoesNotIncludeDevelopmentHint(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.UserID(next, middleware.UserIDOptions{
		PlaceholderModeEnabled: false,
		PlaceholderUserID:      "",
		Environment:            "production",
	})

	req := httptest.NewRequest(http.MethodGet, "/me/consents", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json payload: %v", err)
	}
	if payload["code"] != "IDENTITY_UNAVAILABLE" {
		t.Fatalf("expected IDENTITY_UNAVAILABLE, got %v", payload["code"])
	}
	if payload["message"] != "identity unavailable" {
		t.Fatalf("expected base production message, got %v", payload["message"])
	}
}

func TestUserIDMiddleware_Returns503WhenUserIDMissing(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.UserID(next, middleware.UserIDOptions{
		PlaceholderModeEnabled: true,
		PlaceholderUserID:      "  ",
	})

	req := httptest.NewRequest(http.MethodGet, "/me/consents", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("expected json payload: %v", err)
	}
	if payload["code"] != "PLACEHOLDER_UNAVAILABLE" {
		t.Fatalf("expected PLACEHOLDER_UNAVAILABLE, got %v", payload["code"])
	}
}

func TestUserIDFromContext_ReturnsFalseWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/me/consents", nil)
	if _, ok := middleware.UserIDFromContext(req.Context()); ok {
		t.Fatal("expected false for missing user id")
	}
}
