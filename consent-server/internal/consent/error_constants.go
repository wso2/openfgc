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

// Package consent provides consent management functionality.
package consent

import "github.com/wso2/openfgc/internal/system/error/serviceerror"

// Client errors for consent operations.
var (
	// ErrorInvalidRequestBody is the error returned when the request body is invalid or malformed.
	ErrorInvalidRequestBody = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4001",
		Message:     "Invalid request body",
		Description: "The request body is malformed or contains invalid data",
	}
	// ErrorValidationFailed is the error returned when request validation fails.
	ErrorValidationFailed = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4002",
		Message:     "Validation failed",
		Description: "Request validation failed",
	}
	// ErrorConsentNotFound is the error returned when a consent is not found.
	ErrorConsentNotFound = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4040",
		Message:     "Consent not found",
		Description: "The requested consent could not be found",
	}
	// ErrorConsentAlreadyRevoked is the error returned when attempting to revoke an already revoked consent.
	ErrorConsentAlreadyRevoked = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4041",
		Message:     "Consent already revoked",
		Description: "The consent has already been revoked",
	}
)

// Server errors for consent operations.
var (
	// ErrorInternalServerError is the error returned when an internal operation fails.
	ErrorInternalServerError = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CS-5000",
		Message:     "Internal server error",
		Description: "An unexpected internal error occurred",
	}
)
