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

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wso2/openfgc/consent-server/internal/system/constants"
	sysContext "github.com/wso2/openfgc/consent-server/internal/system/context"
)

// captureHandler records the correlation ID found in the request context.
type captureHandler struct {
	capturedID string
}

func (h *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.capturedID = sysContext.GetTraceID(r.Context())
	w.WriteHeader(http.StatusOK)
}

func applyMiddleware(r *http.Request) (w *httptest.ResponseRecorder, captured *captureHandler) {
	captured = &captureHandler{}
	w = httptest.NewRecorder()
	CorrelationIDMiddleware(captured).ServeHTTP(w, r)
	return
}

func TestCorrelationIDMiddleware_UsesXCorrelationID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(constants.CorrelationIDHeaderName, "corr-111")
	r.Header.Set(constants.RequestIdHeaderName, "req-222")

	_, captured := applyMiddleware(r)

	// X-Correlation-ID has highest priority.
	if captured.capturedID != "corr-111" {
		t.Errorf("context trace ID = %q, want corr-111", captured.capturedID)
	}
}

func TestCorrelationIDMiddleware_FallsBackToXRequestID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(constants.RequestIdHeaderName, "req-222")
	r.Header.Set(constants.TraceIDHeaderName, "trace-333")

	_, captured := applyMiddleware(r)

	// X-Request-ID wins over X-Trace-ID when X-Correlation-ID is absent.
	if captured.capturedID != "req-222" {
		t.Errorf("context trace ID = %q, want req-222", captured.capturedID)
	}
}

func TestCorrelationIDMiddleware_FallsBackToXTraceID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(constants.TraceIDHeaderName, "trace-333")

	_, captured := applyMiddleware(r)

	if captured.capturedID != "trace-333" {
		t.Errorf("context trace ID = %q, want trace-333", captured.capturedID)
	}
}

func TestCorrelationIDMiddleware_GeneratesUUIDWhenNoHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	_, captured := applyMiddleware(r)

	// A UUID was generated — it must be non-empty and not the same as a second request.
	if captured.capturedID == "" {
		t.Error("context trace ID must not be empty when no header is provided")
	}

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	_, captured2 := applyMiddleware(r2)

	if captured.capturedID == captured2.capturedID {
		t.Errorf("two requests without headers must get different trace IDs; both got %q", captured.capturedID)
	}
}

func TestCorrelationIDMiddleware_SetsResponseHeader(t *testing.T) {
	const id = "client-provided-id"
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(constants.CorrelationIDHeaderName, id)

	w, _ := applyMiddleware(r)

	got := w.Header().Get(constants.CorrelationIDHeaderName)
	if got != id {
		t.Errorf("response header %s = %q, want %q", constants.CorrelationIDHeaderName, got, id)
	}
}

func TestCorrelationIDMiddleware_GeneratedIDEchoedInResponseHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w, captured := applyMiddleware(r)

	got := w.Header().Get(constants.CorrelationIDHeaderName)
	if got == "" {
		t.Error("response must include X-Correlation-ID even when generated")
	}
	if got != captured.capturedID {
		t.Errorf("response header ID %q != context ID %q", got, captured.capturedID)
	}
}
