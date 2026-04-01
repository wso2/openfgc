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

// Delegation errors — used when consent is given on behalf of another person.
// Covers parent→child, carer→disabled adult, power-of-attorney, etc.
var (
	// ErrorInvalidDelegation is returned when a delegated consent request is
	// missing required attributes such as principal_id or valid_until.
	ErrorInvalidDelegation = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4050",
		Message:     "Invalid delegation",
		Description: "Delegated consent is missing required attributes",
	}

	// ErrorNotAuthorizedForPrincipal is returned when the caller tries to read
	// another person's consents without being a registered delegate .
	ErrorNotAuthorizedForPrincipal = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4051",
		Message:     "Not authorized for principal",
		Description: "Caller is not a registered delegate for the requested data principal",
	}

	// ErrorRevocationNotPermitted is returned when the caller does not have
	// permission to revoke a delegated consent .
	ErrorRevocationNotPermitted = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4052",
		Message:     "Revocation not permitted",
		Description: "Caller does not have permission to revoke this consent",
	}

	// ErrorDelegationExpired is returned when a delegate tries to act after
	// guardian.valid_until has passed — only the principal may act now .
	ErrorDelegationExpired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4053",
		Message:     "Delegation expired",
		Description: "The delegation period has ended; only the data principal may act on this consent",
	}

	// ErrorUnauthorized is returned when the caller is not permitted to perform
	// the requested operation (e.g., accessing another principal's consents without
	// being a registered delegate). Used by handler_test.go .
	ErrorUnauthorized = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CS-4054",
		Message:     "Unauthorized",
		Description: "Caller is not authorized to perform this operation",
	}
)
