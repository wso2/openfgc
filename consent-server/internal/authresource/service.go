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

package authresource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wso2/openfgc/internal/authresource/model"
	authvalidator "github.com/wso2/openfgc/internal/authresource/validator"
	consentModel "github.com/wso2/openfgc/internal/consent/model"
	"github.com/wso2/openfgc/internal/consent/validator"
	"github.com/wso2/openfgc/internal/system/config"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/internal/system/utils"
)

// AuthResourceServiceInterface defines the contract for auth resource business operations
type AuthResourceServiceInterface interface {
	CreateAuthResource(ctx context.Context, consentID, orgID string, request *model.CreateRequest) (*model.Response, *serviceerror.ServiceError)
	GetAuthResource(ctx context.Context, authID, consentID, orgID string) (*model.Response, *serviceerror.ServiceError)
	GetAuthResourcesByConsentID(ctx context.Context, consentID, orgID string) (*model.ListResponse, *serviceerror.ServiceError)
	UpdateAuthResource(ctx context.Context, authID, consentID, orgID string, request *model.UpdateRequest) (*model.Response, *serviceerror.ServiceError)
	UpdateAllStatusByConsentID(ctx context.Context, consentID, orgID string, status string) *serviceerror.ServiceError
}

// authResourceService implements the AuthResourceServiceInterface
type authResourceService struct {
	stores *stores.StoreRegistry
}

// newAuthResourceService creates a new auth resource service
func newAuthResourceService(registry *stores.StoreRegistry) AuthResourceServiceInterface {
	return &authResourceService{
		stores: registry,
	}
}

// CreateAuthResource creates a new authorization resource for a consent
func (s *authResourceService) CreateAuthResource(
	ctx context.Context,
	consentID, orgID string,
	request *model.CreateRequest,
) (*model.Response, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Creating authorization resource",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
		log.String("auth_type", request.AuthType),
		log.String("auth_status", request.AuthStatus))

	// Validate inputs
	if err := s.validateCreateRequest(consentID, orgID, request); err != nil {
		logger.Warn("Auth resource create request validation failed", log.String("error", err.Error()))
		return nil, err
	}

	// Generate auth ID
	authID := utils.GenerateUUID()

	// Marshal resources to JSON if present
	var resourcesJSON *string
	if request.Resources != nil {
		resourcesBytes, err := json.Marshal(request.Resources)
		if err != nil {
			logger.Error("Failed to marshal authorization resources", log.Error(err), log.String("auth_id", authID))
			return nil, serviceerror.CustomServiceError(ErrorValidationFailed, fmt.Sprintf("failed to marshal resources: %v", err))
		}
		resourcesStr := string(resourcesBytes)
		resourcesJSON = &resourcesStr
	}

	// Build auth resource model
	authResource := &model.AuthResource{
		AuthID:      authID,
		ConsentID:   consentID,
		AuthType:    request.AuthType,
		UserID:      request.UserID,
		AuthStatus:  request.AuthStatus,
		UpdatedTime: utils.GetCurrentTimeMillis(),
		Resources:   resourcesJSON,
		OrgID:       orgID,
	}

	// Fetch existing auth resources and consent outside transaction to avoid races
	store := s.stores.AuthResource
	allAuthResources, err := store.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve auth resources",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return nil, serviceerror.CustomServiceError(
			ErrorInternalServerError,
			fmt.Sprintf("failed to retrieve auth resources: %v", err),
		)
	}

	currentConsent, err := s.stores.Consent.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return nil, serviceerror.CustomServiceError(
			ErrorInternalServerError,
			fmt.Sprintf("failed to retrieve consent: %v", err),
		)
	}

	// Extract auth statuses including the newly created one
	authStatuses := make([]string, 0, len(allAuthResources)+1)
	authStatuses = append(authStatuses, authResource.AuthStatus)
	for _, ar := range allAuthResources {
		authStatuses = append(authStatuses, ar.AuthStatus)
	}

	// Derive consent status based on all authorization statuses
	derivedConsentStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)

	// Create auth resource and update consent status in a transaction
	err = s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.Create(tx, authResource)
		},
		func(tx dbmodel.TxInterface) error {
			// Check if status actually changed
			if currentConsent.CurrentStatus == derivedConsentStatus {
				// Status hasn't changed, skip update and audit
				return nil
			}

			// Status changed - update consent status
			updatedTime := utils.GetCurrentTimeMillis()
			if err := s.stores.Consent.UpdateStatus(tx, consentID, orgID, derivedConsentStatus, updatedTime); err != nil {
				return err
			}

			// Create status audit record
			auditID := utils.GenerateUUID()
			reason := fmt.Sprintf("Authorization %s created with status %s", authID, request.AuthStatus)
			audit := &consentModel.ConsentStatusAudit{
				StatusAuditID:  auditID,
				ConsentID:      consentID,
				CurrentStatus:  derivedConsentStatus,
				ActionTime:     updatedTime,
				Reason:         &reason,
				ActionBy:       nil,
				PreviousStatus: &currentConsent.CurrentStatus,
				OrgID:          orgID,
			}
			if err := s.stores.Consent.CreateStatusAudit(tx, audit); err != nil {
				return err
			}
			return nil
		},
	})
	if err != nil {
		logger.Error("Transaction failed for auth resource creation",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return nil, serviceerror.CustomServiceError(
			ErrorInternalServerError,
			fmt.Sprintf("failed to create auth resource: %v", err),
		)
	}

	logger.Info("Auth resource created successfully",
		log.String("auth_id", authResource.AuthID),
	)
	return s.buildResponse(authResource), nil
}

// GetAuthResource retrieves an authorization resource by ID
func (s *authResourceService) GetAuthResource(
	ctx context.Context,
	authID, consentID, orgID string,
) (*model.Response, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving auth resource",
		log.String("auth_id", authID),
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
	)

	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		logger.Warn("Validation failed for get auth resource", log.String("error", err.Error()))
		return nil, err
	}
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		logger.Warn("Validation failed for get auth resource", log.String("error", err.Error()))
		return nil, err
	}

	store := s.stores.AuthResource
	authResource, err := store.GetByID(ctx, authID, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			logger.Warn("Auth resource not found",
				log.String("auth_id", authID),
			)
			return nil, serviceerror.CustomServiceError(
				ErrorAuthResourceNotFound,
				fmt.Sprintf("auth resource not found: %s", authID),
			)
		}
		logger.Error("Failed to retrieve auth resource",
			log.Error(err),
			log.String("auth_id", authID),
		)
		return nil, serviceerror.CustomServiceError(
			ErrorInternalServerError,
			fmt.Sprintf("failed to retrieve auth resource: %v", err),
		)
	}

	// Validate that the auth resource belongs to the specified consent
	if authResource.ConsentID != consentID {
		logger.Warn("Auth resource does not belong to specified consent",
			log.String("auth_id", authID),
			log.String("expected_consent_id", consentID),
			log.String("actual_consent_id", authResource.ConsentID),
		)
		return nil, serviceerror.CustomServiceError(
			ErrorAuthResourceNotFound,
			fmt.Sprintf("auth resource %s does not belong to consent %s", authID, consentID),
		)
	}

	logger.Debug("Auth resource retrieved successfully",
		log.String("auth_id", authResource.AuthID),
		log.String("auth_status", authResource.AuthStatus),
	)
	return s.buildResponse(authResource), nil
}

// GetAuthResourcesByConsentID retrieves all authorization resources for a consent
func (s *authResourceService) GetAuthResourcesByConsentID(
	ctx context.Context,
	consentID, orgID string,
) (*model.ListResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving auth resources by consent ID",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
	)

	// Validate inputs
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		logger.Warn("Validation failed for get auth resources by consent", log.String("error", err.Error()))
		return nil, err
	}

	store := s.stores.AuthResource
	authResources, err := store.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to fetch auth resources by consent ID",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return nil, serviceerror.CustomServiceError(
			ErrorInternalServerError,
			fmt.Sprintf("failed to fetch auth resources: %v", err),
		)
	}

	// Initialize as empty slice to ensure JSON serialization returns [] instead of null
	responses := make([]model.Response, 0, len(authResources))
	for _, ar := range authResources {
		responses = append(responses, *s.buildResponse(&ar))
	}

	logger.Debug("Auth resources retrieved successfully",
		log.String("consent_id", consentID),
		log.Int("count", len(authResources)),
	)
	return &model.ListResponse{
		Data: responses,
	}, nil
}

// UpdateAuthResource updates an existing authorization resource
func (s *authResourceService) UpdateAuthResource(
	ctx context.Context,
	authID, consentID, orgID string,
	request *model.UpdateRequest,
) (*model.Response, *serviceerror.ServiceError) {

	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating auth resource",
		log.String("auth_id", authID),
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
		log.String("new_auth_status", request.AuthStatus),
	)

	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		logger.Warn("Validation failed for update auth resource",
			log.String("error", err.Error()),
			log.String("auth_id", authID),
		)
		return nil, err
	}
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		logger.Warn("Validation failed for update auth resource",
			log.String("error", err.Error()),
			log.String("consent_id", consentID),
		)
		return nil, err
	}

	// Get existing auth resource
	store := s.stores.AuthResource
	existingAuthResource, err := store.GetByID(ctx, authID, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, serviceerror.CustomServiceError(
				ErrorAuthResourceNotFound,
				fmt.Sprintf("auth resource not found: %s", authID),
			)
		}
		return nil, serviceerror.CustomServiceError(
			ErrorRetrieveAuthResource,
			fmt.Sprintf("failed to retrieve auth resource: %v", err),
		)
	}

	// Validate that the auth resource belongs to the specified consent
	if existingAuthResource.ConsentID != consentID {
		logger.Warn("Auth resource does not belong to specified consent",
			log.String("auth_id", authID),
			log.String("expected_consent_id", consentID),
			log.String("actual_consent_id", existingAuthResource.ConsentID),
		)
		return nil, serviceerror.CustomServiceError(
			ErrorAuthResourceNotFound,
			fmt.Sprintf("auth resource %s does not belong to consent %s", authID, consentID),
		)
	}

	// Update fields if provided
	updatedAuthResource := *existingAuthResource
	updatedAuthResource.UpdatedTime = utils.GetCurrentTimeMillis()

	statusChanged := false
	if request.AuthStatus != "" {
		// Validate that auth status is not a system-reserved status
		cfg := config.Get()
		if cfg == nil {
			logger.Error("Configuration not initialized")
			return nil, serviceerror.CustomServiceError(
				ErrorInternalServerError,
				"configuration not initialized",
			)
		}
		if err := authvalidator.ValidateAuthStatus(request.AuthStatus, cfg.Consent.AuthStatusMappings); err != nil {
			logger.Warn("Invalid auth status provided",
				log.String("auth_id", authID),
				log.String("status", request.AuthStatus),
				log.Error(err),
			)
			return nil, serviceerror.CustomServiceError(
				ErrorValidationFailed,
				err.Error(),
			)
		}
		updatedAuthResource.AuthStatus = request.AuthStatus
		statusChanged = (existingAuthResource.AuthStatus != request.AuthStatus)
		if statusChanged {
			logger.Debug("Auth status changed",
				log.String("auth_id", authID),
				log.String("old_status", existingAuthResource.AuthStatus),
				log.String("new_status", request.AuthStatus),
			)
		}
	}

	if request.UserID != nil {
		updatedAuthResource.UserID = request.UserID
	}

	if request.Resources != nil {
		resourcesBytes, err := json.Marshal(request.Resources)
		if err != nil {
			return nil, serviceerror.CustomServiceError(
				ErrorValidationFailed,
				fmt.Sprintf("failed to marshal resources: %v", err),
			)
		}
		resourcesStr := string(resourcesBytes)
		updatedAuthResource.Resources = &resourcesStr
	}

	// Fetch all auth resources and consent outside transaction if status changed
	var allAuthResources []model.AuthResource
	var currentConsent *consentModel.Consent
	var derivedConsentStatus string

	if statusChanged {
		// Pre-fetch data outside transaction to avoid races
		var err error
		allAuthResources, err = store.GetByConsentID(ctx, existingAuthResource.ConsentID, orgID)
		if err != nil {
			logger.Error("Failed to retrieve auth resources",
				log.Error(err),
				log.String("consent_id", existingAuthResource.ConsentID),
			)
			return nil, serviceerror.CustomServiceError(
				ErrorInternalServerError,
				fmt.Sprintf("failed to retrieve auth resources: %v", err),
			)
		}

		currentConsent, err = s.stores.Consent.GetByID(ctx, existingAuthResource.ConsentID, orgID)
		if err != nil {
			logger.Error("Failed to retrieve consent",
				log.Error(err),
				log.String("consent_id", existingAuthResource.ConsentID),
			)
			return nil, serviceerror.CustomServiceError(
				ErrorInternalServerError,
				fmt.Sprintf("failed to retrieve consent: %v", err),
			)
		}

		// Extract auth statuses (including the updated one)
		authStatuses := make([]string, 0, len(allAuthResources))
		for _, ar := range allAuthResources {
			if ar.AuthID == authID {
				// Use the new status for this auth resource
				authStatuses = append(authStatuses, updatedAuthResource.AuthStatus)
			} else {
				authStatuses = append(authStatuses, ar.AuthStatus)
			}
		}

		// Derive consent status
		derivedConsentStatus = validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
		logger.Debug("Derived consent status from auth statuses",
			log.String("consent_id", existingAuthResource.ConsentID),
			log.String("derived_status", derivedConsentStatus),
			log.Int("auth_count", len(authStatuses)),
		)
	}

	// Update auth resource and potentially consent status in transaction
	transactionSteps := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.Update(tx, &updatedAuthResource)
		},
	}

	// If auth status changed, update consent status accordingly
	if statusChanged {
		transactionSteps = append(transactionSteps, func(tx dbmodel.TxInterface) error {
			// Only update if consent status actually changed
			if currentConsent.CurrentStatus != derivedConsentStatus {
				updatedTime := utils.GetCurrentTimeMillis()

				// Update consent status
				if err := s.stores.Consent.UpdateStatus(tx, existingAuthResource.ConsentID, orgID, derivedConsentStatus, updatedTime); err != nil {
					return err
				}

				// Create status audit record
				auditID := utils.GenerateUUID()
				reason := fmt.Sprintf("Authorization %s status updated from %s to %s", authID, existingAuthResource.AuthStatus, updatedAuthResource.AuthStatus)
				audit := &consentModel.ConsentStatusAudit{
					StatusAuditID:  auditID,
					ConsentID:      existingAuthResource.ConsentID,
					CurrentStatus:  derivedConsentStatus,
					ActionTime:     updatedTime,
					Reason:         &reason,
					ActionBy:       nil,
					PreviousStatus: &currentConsent.CurrentStatus,
					OrgID:          orgID,
				}
				if err := s.stores.Consent.CreateStatusAudit(tx, audit); err != nil {
					return err
				}
				return nil
			}
			return nil
		})
	}

	err = s.stores.ExecuteTransaction(transactionSteps)
	if err != nil {
		logger.Error("Transaction failed for auth resource update",
			log.Error(err),
			log.String("auth_id", authID),
		)
		return nil, serviceerror.CustomServiceError(
			ErrorUpdateAuthResource,
			fmt.Sprintf("failed to update auth resource: %v", err),
		)
	}

	logger.Info("Auth resource updated successfully",
		log.String("auth_id", updatedAuthResource.AuthID),
		log.Bool("status_changed", statusChanged),
	)
	return s.buildResponse(&updatedAuthResource), nil
}

// UpdateAllStatusByConsentID updates status for all auth resources of a consent
func (s *authResourceService) UpdateAllStatusByConsentID(
	ctx context.Context,
	consentID, orgID string,
	status string,
) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating all auth resource statuses for consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
		log.String("new_status", status),
	)

	// Validate inputs
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		logger.Warn("Validation failed for update auth statuses", log.String("error", err.Error()))
		return err
	}
	if status == "" {
		logger.Warn("Status is required")
		return serviceerror.CustomServiceError(
			ErrorInvalidRequestBody,
			"status is required",
		)
	}

	// Update all statuses
	store := s.stores.AuthResource
	updatedTime := utils.GetCurrentTimeMillis()
	err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.UpdateAllStatusByConsentID(tx, consentID, orgID, status, updatedTime)
		},
	})
	if err != nil {
		logger.Error("Transaction failed for auth statuses update",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return serviceerror.CustomServiceError(
			ErrorUpdateAuthResource,
			fmt.Sprintf("failed to update auth resource statuses: %v", err),
		)
	}

	logger.Info("Auth resource statuses updated successfully",
		log.String("consent_id", consentID),
		log.String("status", status),
	)
	return nil
}

// Helper methods for validation

func (s *authResourceService) validateCreateRequest(consentID, orgID string, request *model.CreateRequest) *serviceerror.ServiceError {
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return err
	}
	if request == nil {
		return serviceerror.CustomServiceError(
			ErrorInvalidRequestBody,
			"request body is required",
		)
	}
	if request.AuthType == "" {
		return serviceerror.CustomServiceError(
			ErrorInvalidRequestBody,
			"auth type is required",
		)
	}
	if request.AuthStatus == "" {
		return serviceerror.CustomServiceError(
			ErrorInvalidRequestBody,
			"auth status is required",
		)
	}
	// Validate that auth status is not a system-reserved status
	cfg := config.Get()
	if cfg == nil {
		return serviceerror.CustomServiceError(
			ErrorInternalServerError,
			"configuration not initialized",
		)
	}
	if err := authvalidator.ValidateAuthStatus(request.AuthStatus, cfg.Consent.AuthStatusMappings); err != nil {
		return serviceerror.CustomServiceError(
			ErrorValidationFailed,
			err.Error(),
		)
	}
	return nil
}

func (s *authResourceService) validateAuthIDAndOrgID(authID, orgID string) *serviceerror.ServiceError {
	if authID == "" {
		return serviceerror.CustomServiceError(
			ErrorAuthResourceIDRequired,
			"auth ID is required",
		)
	}
	if len(authID) > 255 {
		return serviceerror.CustomServiceError(
			ErrorValidationFailed,
			"auth ID too long: maximum 255 characters",
		)
	}
	return s.validateOrgID(orgID)
}

func (s *authResourceService) validateConsentIDAndOrgID(consentID, orgID string) *serviceerror.ServiceError {
	if consentID == "" {
		return serviceerror.CustomServiceError(
			ErrorConsentIDRequired,
			"consent ID is required",
		)
	}
	if len(consentID) > 255 {
		return serviceerror.CustomServiceError(
			ErrorValidationFailed,
			"consent ID too long: maximum 255 characters",
		)
	}
	return s.validateOrgID(orgID)
}

func (s *authResourceService) validateOrgID(orgID string) *serviceerror.ServiceError {
	if orgID == "" {
		return serviceerror.CustomServiceError(
			ErrorOrgIDRequired,
			"organization ID is required",
		)
	}
	if len(orgID) > 255 {
		return serviceerror.CustomServiceError(
			ErrorValidationFailed,
			"organization ID too long: maximum 255 characters",
		)
	}
	return nil
}

func (s *authResourceService) buildResponse(authResource *model.AuthResource) *model.Response {
	var resources interface{}
	if authResource.Resources != nil && *authResource.Resources != "" {
		// Try to unmarshal resources
		if err := json.Unmarshal([]byte(*authResource.Resources), &resources); err != nil {
			// Log the error with contextual information
			log.GetLogger().Error("Failed to unmarshal auth resource resources field",
				log.String("auth_id", authResource.AuthID),
				log.String("raw_resources", *authResource.Resources),
				log.Error(err),
			)
			// Continue with nil resources rather than failing the entire response
		}
	}

	return &model.Response{
		AuthID:      authResource.AuthID,
		AuthType:    authResource.AuthType,
		UserID:      authResource.UserID,
		AuthStatus:  authResource.AuthStatus,
		UpdatedTime: authResource.UpdatedTime,
		Resources:   resources,
	}
}
