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
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wso2/openfgc/internal/system/constants"
	"github.com/wso2/openfgc/internal/system/error/apierror"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/log"
)

func DecodeJSONBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// WriteErrorResponse writes a JSON error response with the given status code and error details.
func WriteErrorResponse(w http.ResponseWriter, statusCode int, errorResp apierror.ErrorResponse) {
	logger := log.GetLogger()

	// Encode to buffer first to ensure encoding succeeds before sending headers
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(errorResp); err != nil {
		logger.Error("Failed to encode error response", log.Error(err))
		http.Error(w, ErrorEncodingError.Error(), http.StatusInternalServerError)
		return
	}

	// Encoding succeeded, now safe to send headers and write response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(statusCode)
	_, _ = w.Write(buf.Bytes())
}

// SendError writes a ServiceError as an HTTP response with appropriate status code and trace ID.
// This function extracts the trace ID from the request context, logs the error, and includes it in the error response.
func SendError(w http.ResponseWriter, r *http.Request, err *serviceerror.ServiceError) {
	// Determine HTTP status code based on error type and code
	statusCode := mapErrorToStatusCode(err)

	// Extract trace ID from request context
	traceID := extractTraceID(r)

	// Log the error with context
	logger := log.GetLogger().WithContext(r.Context())
	if err.Type == serviceerror.ServerErrorType {
		logger.Error("Server error occurred",
			log.String("code", err.Code),
			log.String("message", err.Message),
			log.String("description", err.Description),
			log.Int("http_status", statusCode),
		)
	} else {
		logger.Warn("Client error occurred",
			log.String("code", err.Code),
			log.String("message", err.Message),
			log.String("description", err.Description),
			log.Int("http_status", statusCode),
		)
	}

	// Create error response with new format
	errorResponse := apierror.NewErrorResponse(
		err.Code,
		err.Message,
		err.Description,
		traceID,
	)

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// mapErrorToStatusCode maps service error codes to HTTP status codes
func mapErrorToStatusCode(err *serviceerror.ServiceError) int {
	if err.Type == serviceerror.ServerErrorType {
		return http.StatusInternalServerError
	}

	// Client error type - map based on error code patterns
	// Conflict errors (must check before Not Found to avoid conflicts with -404x patterns)
	// Pattern 1: CSE-409x, CS-4090, CP-4090, CE-4090, AR-4090
	// Pattern 2: CE-1011 (ElementNameExists), CE-1012 (DuplicateNameInBatch)
	// Pattern 3: CP-4041 (PurposeNameExists - has 404 in it but should be conflict)
	// Pattern 4: CS-4042 (ConsentStatusConflict - has 404 in it but should be conflict)
	if strings.Contains(err.Code, "-409") || strings.HasSuffix(err.Code, "4090") ||
		strings.HasSuffix(err.Code, "1011") || strings.HasSuffix(err.Code, "1012") ||
		strings.HasSuffix(err.Code, "4041") || strings.HasSuffix(err.Code, "4042") { // CP-4041, CS-4042
		return http.StatusConflict
	}

	// Not Found errors
	// Pattern 1: CSE-404x, CS-4040, CP-4040, CE-4040, AR-4040
	// Pattern 2: CE-1016 (ElementNotFound), CS-4040 (ConsentNotFound), etc.
	if strings.Contains(err.Code, "-404") || strings.HasSuffix(err.Code, "4040") ||
		strings.HasSuffix(err.Code, "1016") { // CE-1016 ElementNotFound
		return http.StatusNotFound
	}

	// All other client errors default to BadRequest
	return http.StatusBadRequest
}

// extractTraceID extracts the trace ID (correlation ID) from the request context
func extractTraceID(r *http.Request) string {
	if r == nil || r.Context() == nil {
		return ""
	}

	traceID := r.Context().Value(log.ContextKeyTraceID)
	if traceID != nil {
		if tid, ok := traceID.(string); ok {
			return tid
		}
	}
	return ""
}

// Server errors
var (
	// InternalServerError is the error returned for unexpected server errors.
	ErrorEncodingError = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "SSE-5000",
		Message:     "Encoding error",
		Description: "An error occurred while encoding the response",
	}
)
