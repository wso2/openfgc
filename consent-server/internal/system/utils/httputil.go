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
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wso2/openfgc/consent-server/internal/system/constants"
	"github.com/wso2/openfgc/consent-server/internal/system/error/apierror"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/consent-server/internal/system/log"
)

// DecodeJSONBody decodes the JSON request body into v.
func DecodeJSONBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
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

	// For server errors, never expose internal details (DB errors, stack traces, etc.) to the client.
	// The full description is already logged above with the traceId for server-side debugging.
	clientDescription := err.Description
	if err.Type == serviceerror.ServerErrorType {
		clientDescription = "An internal server error occurred. Please reference the traceId for support."
	}

	errorResponse := apierror.NewErrorResponse(
		err.Code,
		err.Message,
		clientDescription,
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
		strings.HasSuffix(err.Code, "4041") || strings.HasSuffix(err.Code, "4042") {
		return http.StatusConflict
	}

	// Not Found errors
	// Pattern 1: CSE-404x, CS-4040, CP-4040, CE-4040, AR-4040
	// Pattern 2: CE-1016 (ElementNotFound), CS-4040 (ConsentNotFound), etc.
	if strings.Contains(err.Code, "-404") || strings.HasSuffix(err.Code, "4040") ||
		strings.HasSuffix(err.Code, "1016") {
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
