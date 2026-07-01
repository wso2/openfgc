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

package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wso2/openfgc/consent-server/internal/system/error/apierror"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/consent-server/internal/system/log"
)

// makeClientErr builds a client-type ServiceError with the given code.
func makeClientErr(code string) *serviceerror.ServiceError {
	e := serviceerror.CustomServiceError(serviceerror.ServiceError{
		Type:    serviceerror.ClientErrorType,
		Code:    code,
		Message: "test message",
	}, "test description")
	return e
}

// makeServerErr builds a server-type ServiceError.
func makeServerErr(code string) *serviceerror.ServiceError {
	e := serviceerror.CustomServiceError(serviceerror.ServiceError{
		Type:    serviceerror.ServerErrorType,
		Code:    code,
		Message: "internal error",
	}, "sensitive DB details")
	return e
}

// =============================================================================
// mapErrorToStatusCode
// =============================================================================

func TestMapErrorToStatusCode(t *testing.T) {
	cases := []struct {
		name       string
		err        *serviceerror.ServiceError
		wantStatus int
	}{
		// Server errors always → 500
		{name: "server error type → 500", err: makeServerErr("AR-5000"), wantStatus: http.StatusInternalServerError},

		// Conflict patterns
		{name: "code contains -409 → 409", err: makeClientErr("CS-4090"), wantStatus: http.StatusConflict},
		{name: "suffix 4090 → 409", err: makeClientErr("CE-4090"), wantStatus: http.StatusConflict},
		{name: "suffix 1011 → 409 (ElementNameExists)", err: makeClientErr("CE-1011"), wantStatus: http.StatusConflict},
		{name: "suffix 1012 → 409 (DuplicateNameInBatch)", err: makeClientErr("CE-1012"), wantStatus: http.StatusConflict},
		{name: "suffix 4041 → 409 (PurposeNameExists)", err: makeClientErr("CP-4041"), wantStatus: http.StatusConflict},
		{name: "suffix 4042 → 409 (ConsentStatusConflict)", err: makeClientErr("CS-4042"), wantStatus: http.StatusConflict},

		// Not found patterns
		{name: "code contains -404 → 404", err: makeClientErr("AR-4040"), wantStatus: http.StatusNotFound},
		{name: "suffix 4040 → 404", err: makeClientErr("CS-4040"), wantStatus: http.StatusNotFound},
		{name: "suffix 1016 → 404 (ElementNotFound)", err: makeClientErr("CE-1016"), wantStatus: http.StatusNotFound},

		// Conflict is checked before not-found: CP-4041 has "404" but maps to 409
		{name: "CP-4041 → 409, not 404 (conflict takes priority)", err: makeClientErr("CP-4041"), wantStatus: http.StatusConflict},

		// Everything else → 400
		{name: "generic validation error → 400", err: makeClientErr("AR-4002"), wantStatus: http.StatusBadRequest},
		{name: "org ID required → 400", err: makeClientErr("AR-4007"), wantStatus: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapErrorToStatusCode(tc.err)
			if got != tc.wantStatus {
				t.Errorf("mapErrorToStatusCode(%q) = %d, want %d", tc.err.Code, got, tc.wantStatus)
			}
		})
	}
}

// =============================================================================
// SendError
// =============================================================================

func TestSendError_ClientError_DescriptionExposed(t *testing.T) {
	err := makeClientErr("AR-4002")
	err.Description = "field X is required"

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	SendError(w, r, err)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}

	var resp apierror.ErrorResponse
	if e := json.Unmarshal(w.Body.Bytes(), &resp); e != nil {
		t.Fatalf("response is not valid JSON: %v", e)
	}
	if resp.Description != "field X is required" {
		t.Errorf("client description should be passed through; got %q", resp.Description)
	}
	if resp.Code != "AR-4002" {
		t.Errorf("code = %q, want AR-4002", resp.Code)
	}
}

func TestSendError_ServerError_DescriptionMasked(t *testing.T) {
	err := makeServerErr("AR-5000")
	// Description contains sensitive internal detail.
	err.Description = "sensitive DB details"

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	SendError(w, r, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}

	var resp apierror.ErrorResponse
	if e := json.Unmarshal(w.Body.Bytes(), &resp); e != nil {
		t.Fatalf("response is not valid JSON: %v", e)
	}
	if strings.Contains(resp.Description, "sensitive DB details") {
		t.Errorf("server error description must not be exposed to client; got %q", resp.Description)
	}
	if resp.Description == "" {
		t.Error("client description must not be empty for server errors")
	}
}

func TestSendError_SetsContentTypeHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	SendError(w, r, makeClientErr("AR-4002"))

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestSendError_TraceIDFromContext(t *testing.T) {
	const traceID = "test-trace-id-123"

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(r.Context(), log.ContextKeyTraceID, traceID)
	r = r.WithContext(ctx)

	SendError(w, r, makeClientErr("AR-4002"))

	var resp apierror.ErrorResponse
	if e := json.Unmarshal(w.Body.Bytes(), &resp); e != nil {
		t.Fatalf("response is not valid JSON: %v", e)
	}
	if resp.TraceID != traceID {
		t.Errorf("traceId = %q, want %q", resp.TraceID, traceID)
	}
}

// =============================================================================
// DecodeJSONBody
// =============================================================================

func TestDecodeJSONBody(t *testing.T) {
	t.Run("valid JSON decoded into struct", func(t *testing.T) {
		var out struct {
			Name string `json:"name"`
		}
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice"}`))
		if err := DecodeJSONBody(r, &out); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Name != "alice" {
			t.Errorf("name = %q, want alice", out.Name)
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		var out struct{ Name string }
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
		if err := DecodeJSONBody(r, &out); err == nil {
			t.Error("expected error for malformed JSON, got nil")
		}
	})
}
