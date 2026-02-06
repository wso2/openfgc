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

// Package consentelement provides consent element management functionality.
package consentelement

import "github.com/wso2/openfgc/internal/system/error/serviceerror"

// Client errors for consent element operations.
var (
	// ErrorInvalidRequestBody is the error returned when the request body is invalid or malformed.
	ErrorInvalidRequestBody = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1001",
		Message:     "Invalid request body",
		Description: "The request body is malformed or contains invalid data",
	}
	// ErrorAtLeastOneElement is the error returned when no elements are provided in a batch operation.
	ErrorAtLeastOneElement = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1002",
		Message:     "At least one element required",
		Description: "At least one element must be provided",
	}
	// ErrorOrgIDRequired is the error returned when organization ID is missing.
	ErrorOrgIDRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1003",
		Message:     "Organization ID required",
		Description: "Organization ID is required",
	}
	// ErrorElementNameRequired is the error returned when element name is missing.
	ErrorElementNameRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1004",
		Message:     "Element name required",
		Description: "Element name is required",
	}
	// ErrorElementTypeRequired is the error returned when element type is missing.
	ErrorElementTypeRequired = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1005",
		Message:     "Element type required",
		Description: "Element type is required",
	}
	// ErrorElementNameTooLong is the error returned when element name exceeds maximum length.
	ErrorElementNameTooLong = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1006",
		Message:     "Element name too long",
		Description: "Element name must not exceed 255 characters",
	}
	// ErrorElementDescriptionTooLong is the error returned when element description exceeds maximum length.
	ErrorElementDescriptionTooLong = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1008",
		Message:     "Element description too long",
		Description: "Element description must not exceed 1024 characters",
	}
	// ErrorInvalidElementType is the error returned when an invalid element type is provided.
	ErrorInvalidElementType = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1010",
		Message:     "Invalid element type",
		Description: "The provided element type is invalid",
	}
	// ErrorElementNameExists is the error returned when an element with the same name already exists.
	ErrorElementNameExists = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1011",
		Message:     "Element name already exists",
		Description: "An element with the same name already exists for this organization",
	}
	// ErrorDuplicateNameInBatch is the error returned when duplicate element names are found in a batch request.
	ErrorDuplicateNameInBatch = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1012",
		Message:     "Duplicate element name in batch",
		Description: "Duplicate element names found in the request batch",
	}
	// ErrorElementInUse is the error returned when attempting to modify an element that is in use.
	ErrorElementInUse = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1013",
		Message:     "Element in use",
		Description: "Cannot modify element as it is being used in one or more consent purposes",
	}
	// ErrorAtLeastOneElementName is the error returned when no element names are provided for validation.
	ErrorAtLeastOneElementName = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1014",
		Message:     "At least one element name required",
		Description: "At least one element name must be provided",
	}
	// ErrorNoValidElements is the error returned when no valid elements are found.
	ErrorNoValidElements = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1015",
		Message:     "No valid elements found",
		Description: "No valid elements found",
	}
	// ErrorElementNotFound is the error returned when an element is not found.
	ErrorElementNotFound = serviceerror.ServiceError{
		Type:        serviceerror.ClientErrorType,
		Code:        "CE-1016",
		Message:     "Element not found",
		Description: "The requested element could not be found",
	}
)

// Server errors for consent element operations.
var (
	// ErrorCheckNameExistence is the error returned when checking name existence fails.
	ErrorCheckNameExistence = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CE-5001",
		Message:     "Failed to check name existence",
		Description: "An error occurred while checking if the element name exists",
	}
	// ErrorReadElement is the error returned when reading an element fails.
	ErrorReadElement = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CE-5002",
		Message:     "Failed to read element",
		Description: "An error occurred while reading the element",
	}
	// ErrorCreateElement is the error returned when creating an element fails.
	ErrorCreateElement = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CE-5003",
		Message:     "Failed to create element",
		Description: "An error occurred while creating the element",
	}
	// ErrorUpdateElement is the error returned when updating an element fails.
	ErrorUpdateElement = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CE-5008",
		Message:     "Failed to update element",
		Description: "An error occurred while updating the element",
	}
	// ErrorDeleteElement is the error returned when deleting an element fails.
	ErrorDeleteElement = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CE-5009",
		Message:     "Failed to delete element",
		Description: "An error occurred while deleting the element",
	}
	// ErrorValidateElement is the error returned when validating element fails.
	ErrorValidateElement = serviceerror.ServiceError{
		Type:        serviceerror.ServerErrorType,
		Code:        "CE-5010",
		Message:     "Failed to validate element",
		Description: "An error occurred while validating element",
	}
)
