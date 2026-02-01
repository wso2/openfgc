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

package serviceerror

// ServiceErrorType defines the type of error (client or server error)
type ServiceErrorType string

const (
	// ClientErrorType indicates an error caused by client input (4xx errors)
	ClientErrorType ServiceErrorType = "client_error"
	// ServerErrorType indicates an error caused by server issues (5xx errors)
	ServerErrorType ServiceErrorType = "server_error"
)

// ServiceError represents an error that occurred in the service layer.
// Services should create and return ServiceError instances to provide
// structured error information to the handlers.
type ServiceError struct {
	Code        string           `json:"code"`        // Error code (e.g., "CSE-4040")
	Type        ServiceErrorType `json:"type"`        // Error type (client_error or server_error)
	Message     string           `json:"message"`     // Human-readable error message
	Description string           `json:"description"` // Detailed error description
}

// NewServiceError creates a new ServiceError with the specified details.
// This is the primary way to create service errors.
//
// Example usage:
//
//	return serviceerror.NewServiceError(
//	    "CSE-4040",
//	    serviceerror.ClientErrorType,
//	    "Consent Not Found",
//	    fmt.Sprintf("Consent with ID '%s' not found", consentID),
//	)
func NewServiceError(code string, errorType ServiceErrorType, message, description string) *ServiceError {
	return &ServiceError{
		Code:        code,
		Type:        errorType,
		Message:     message,
		Description: description,
	}
}

// CustomServiceError creates a custom service error based on a base error with a custom description.
// This is useful when you want to use a predefined error but customize the description.
//
// Example usage:
//
//	return serviceerror.CustomServiceError(ErrorInvalidRequestBody, err.Error())
func CustomServiceError(baseError ServiceError, description string) *ServiceError {
	return &ServiceError{
		Code:        baseError.Code,
		Type:        baseError.Type,
		Message:     baseError.Message,
		Description: description,
	}
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	return e.Message + ": " + e.Description
}
