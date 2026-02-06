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

// Package consentpurpose provides consent purpose management functionality.
package consentpurpose

import "github.com/wso2/openfgc/internal/system/error/serviceerror"

// Client errors for consent purpose operations.
var (
	// ErrorInvalidRequestBody is the error returned when the request body is invalid or malformed.
	ErrorInvalidRequestBody = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4001",
		Message:     "Invalid request body",
		Description: "The request body is malformed or contains invalid data",
	}
	// ErrorValidationFailed is the error returned when request validation fails.
	ErrorValidationFailed = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4002",
		Message:     "Validation failed",
		Description: "Request validation failed",
	}
	// ErrorPurposeNotFound is the error returned when a purpose is not found.
	ErrorPurposeNotFound = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4040",
		Message:     "Purpose not found",
		Description: "The requested consent purpose could not be found",
	}
	// ErrorPurposeNameExists is the error returned when a purpose with the same name already exists.
	ErrorPurposeNameExists = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4041",
		Message:     "Purpose name already exists",
		Description: "A purpose with the same name already exists for this organization",
	}
	// ErrorPurposeInUse is the error returned when attempting to change a purpose that is in use.
	ErrorPurposeInUse = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4090",
		Message:     "Purpose in use",
		Description: "Cannot change purpose as it is being used in one or more consents",
	}
	// ErrorInvalidPurposeElements is the error returned when purpose elements are invalid.
	ErrorInvalidPurposeElements = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4003",
		Message:     "Invalid purpose elements",
		Description: "The provided purpose elements are invalid",
	}
	// ErrorOrgIDRequired is the error returned when organization ID is missing.
	ErrorOrgIDRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4004",
		Message:     "Organization ID required",
		Description: "Organization ID is required",
	}
	// ErrorPurposeIDRequired is the error returned when purpose ID is missing.
	ErrorPurposeIDRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4005",
		Message:     "Purpose ID required",
		Description: "Purpose ID is required",
	}
	// ErrorClientIDRequired is the error returned when client ID is missing.
	ErrorClientIDRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CP-4006",
		Message:     "Client ID required",
		Description: "Client ID is required",
	}
)

// Server errors for consent purpose operations.
var (
	// ErrorCreatePurpose is the error returned when creating a purpose fails.
	ErrorCreatePurpose = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CP-5002",
		Message:     "Failed to create purpose",
		Description: "An error occurred while creating the consent purpose",
	}
	// ErrorRetrievePurpose is the error returned when retrieving a purpose fails.
	ErrorRetrievePurpose = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CP-5003",
		Message:     "Failed to retrieve purpose",
		Description: "An error occurred while retrieving the consent purpose",
	}
	// ErrorListPurposes is the error returned when listing purposes fails.
	ErrorListPurposes = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CP-5004",
		Message:     "Failed to list purposes",
		Description: "An error occurred while listing consent purposes",
	}
	// ErrorUpdatePurpose is the error returned when updating a purpose fails.
	ErrorUpdatePurpose = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CP-5005",
		Message:     "Failed to update purpose",
		Description: "An error occurred while updating the consent purpose",
	}
	// ErrorDeletePurpose is the error returned when deleting a purpose fails.
	ErrorDeletePurpose = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CP-5006",
		Message:     "Failed to delete purpose",
		Description: "An error occurred while deleting the consent purpose",
	}
	// ErrorCheckPurposeUsage is the error returned when checking purpose usage fails.
	ErrorCheckPurposeUsage = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CP-5007",
		Message:     "Failed to check purpose usage",
		Description: "An error occurred while checking if the purpose is in use",
	}
	// ErrorCheckNameExistence is the error returned when checking name existence fails.
	ErrorCheckNameExistence = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CP-5008",
		Message:     "Failed to check name existence",
		Description: "An error occurred while checking if the purpose name exists",
	}
	// ErrorInternalServerError is the error returned when an internal operation fails.
	ErrorInternalServerError = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CP-5000",
		Message:     "Internal server error",
		Description: "An unexpected error occurred while processing the request",
	}
)
