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

// Package authresource provides authorization resource management functionality.
package authresource

import "github.com/wso2/openfgc/internal/system/error/serviceerror"

// Client errors for authorization resource operations.
var (
	// ErrorInvalidRequestBody is the error returned when the request body is invalid or malformed.
	ErrorInvalidRequestBody = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4001",
		Message:     "Invalid request body",
		Description: "The request body is malformed or contains invalid data",
	}
	// ErrorValidationFailed is the error returned when request validation fails.
	ErrorValidationFailed = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4002",
		Message:     "Validation failed",
		Description: "Request validation failed",
	}
	// ErrorAuthResourceNotFound is the error returned when an authorization resource is not found.
	ErrorAuthResourceNotFound = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4040",
		Message:     "Authorization resource not found",
		Description: "The requested authorization resource could not be found",
	}
	// ErrorInvalidAuthStatus is the error returned when an invalid authorization status is provided.
	ErrorInvalidAuthStatus = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4003",
		Message:     "Invalid authorization status",
		Description: "The provided authorization status is invalid",
	}
	// ErrorInvalidAuthType is the error returned when an invalid authorization type is provided.
	ErrorInvalidAuthType = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4004",
		Message:     "Invalid authorization type",
		Description: "The provided authorization type is invalid",
	}
	// ErrorAuthResourceIDRequired is the error returned when authorization resource ID is missing.
	ErrorAuthResourceIDRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4005",
		Message:     "Authorization resource ID required",
		Description: "Authorization resource ID is required",
	}
	// ErrorConsentIDRequired is the error returned when consent ID is missing.
	ErrorConsentIDRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4006",
		Message:     "Consent ID required",
		Description: "Consent ID is required",
	}
	// ErrorOrgIDRequired is the error returned when organization ID is missing.
	ErrorOrgIDRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4007",
		Message:     "Organization ID required",
		Description: "Organization ID is required",
	}
	// ErrorAuthStatusConflict is the error returned when there's an authorization status conflict.
	ErrorAuthStatusConflict = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "AR-4041",
		Message:     "Authorization status conflict",
		Description: "The authorization status does not allow this operation",
	}
)

// Server errors for authorization resource operations.
var (
	// ErrorCreateAuthResource is the error returned when creating an authorization resource fails.
	ErrorCreateAuthResource = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "AR-5002",
		Message:     "Failed to create authorization resource",
		Description: "An error occurred while creating the authorization resource",
	}
	// ErrorRetrieveAuthResource is the error returned when retrieving an authorization resource fails.
	ErrorRetrieveAuthResource = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "AR-5003",
		Message:     "Failed to retrieve authorization resource",
		Description: "An error occurred while retrieving the authorization resource",
	}
	// ErrorUpdateAuthResource is the error returned when updating an authorization resource fails.
	ErrorUpdateAuthResource = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "AR-5004",
		Message:     "Failed to update authorization resource",
		Description: "An error occurred while updating the authorization resource",
	}
	// ErrorDeleteAuthResource is the error returned when deleting an authorization resource fails.
	ErrorDeleteAuthResource = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "AR-5005",
		Message:     "Failed to delete authorization resource",
		Description: "An error occurred while deleting the authorization resource",
	}
	// ErrorListAuthResources is the error returned when listing authorization resources fails.
	ErrorListAuthResources = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "AR-5006",
		Message:     "Failed to list authorization resources",
		Description: "An error occurred while listing authorization resources",
	}
	// ErrorUpdateAuthStatus is the error returned when updating authorization status fails.
	ErrorUpdateAuthStatus = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "AR-5007",
		Message:     "Failed to update authorization status",
		Description: "An error occurred while updating the authorization status",
	}
	// ErrorInternalOperation is the error returned when an internal operation fails.
	ErrorInternalServerError = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "AR-5000",
		Message:     "Internal server error",
		Description: "An unexpected internal error occurred",
	}
)
