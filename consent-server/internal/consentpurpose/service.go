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

package consentpurpose

import (
	"context"
	"fmt"
	"time"

	"github.com/wso2/consent-management-api/internal/consentpurpose/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/stores"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// Service defines the exported service interface
type ConsentPurposeService interface {
	CreatePurpose(ctx context.Context, req model.CreateRequest, orgID, clientID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	GetPurpose(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	ListPurposes(ctx context.Context, orgID, name string, clientIDs []string, purposeNames []string, offset, limit int) ([]model.ConsentPurpose, int, *serviceerror.ServiceError)
	UpdatePurpose(ctx context.Context, purposeID string, req model.UpdateRequest, orgID, clientID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError
}

// service implements the Service interface
type consentPurposeService struct {
	stores *stores.StoreRegistry
}

// NewService creates a new purpose purpose service
func NewConsentPurposeService(registry *stores.StoreRegistry) ConsentPurposeService {
	return &consentPurposeService{
		stores: registry,
	}
}

// CreatePurpose creates a new consent purpose
func (s *consentPurposeService) CreatePurpose(ctx context.Context, req model.CreateRequest, orgID, clientID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Creating consent purpose",
		log.String("name", req.Name),
		log.String("client_id", clientID),
		log.String("org_id", orgID))

	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		logger.Warn("Purpose purpose create request validation failed", log.String("error", err.Error()))
		return nil, err
	}

	// Check if purpose name already exists for this client
	exists, dbErr := s.stores.ConsentPurpose.CheckPurposeNameExists(ctx, req.Name, clientID, orgID, nil)
	if dbErr != nil {
		logger.Error("Failed to check purpose name existence", log.Error(dbErr), log.String("name", req.Name))
		return nil, &ErrorInternalServerError
	}
	if exists {
		logger.Warn("Purpose name already exists for this client", log.String("name", req.Name), log.String("client_id", clientID))
		return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists, fmt.Sprintf("purpose purpose with name '%s' already exists for this client", req.Name))
	}

	// Validate that all purpose names exist
	purposeNameToID, err := s.validatePurposeNamesExist(ctx, req.Elements, orgID)
	if err != nil {
		return nil, err
	}

	// Check for duplicate purpose names within the request
	if duplicateErr := s.checkDuplicatePurposeNames(req.Elements); duplicateErr != nil {
		return nil, duplicateErr
	}

	// Create purpose entity
	purposeID := utils.GenerateUUID()
	now := time.Now().Unix()
	desc := &req.Description
	if req.Description == "" {
		desc = nil
	}

	purpose := &model.ConsentPurpose{
		ID:          purposeID,
		Name:        req.Name,
		Description: desc,
		ClientID:    clientID,
		CreatedTime: now,
		UpdatedTime: now,
		OrgID:       orgID,
	}

	// Execute transaction for purpose creation and purpose linking
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.CreatePurpose(tx, purpose)
		},
	}

	// Add purpose linking operations
	for _, elem := range req.Elements {
		elementID := purposeNameToID[elem.ElementName]
		isMandatory := elem.IsMandatory
		elementName := elem.ElementName

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.LinkElementToPurpose(tx, purposeID, elementID, orgID, isMandatory)
		})

		purpose.Elements = append(purpose.Elements, model.PurposeElement{
			ElementID:   elementID,
			ElementName: elementName,
			IsMandatory: isMandatory,
		})
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to create consent purpose", log.Error(err))
		return nil, &ErrorInternalServerError
	}

	logger.Info("Purpose purpose created successfully", log.String("purpose_id", purposeID))
	return purpose, nil
}

// GetPurpose retrieves a purpose purpose by ID
func (s *consentPurposeService) GetPurpose(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Debug("Retrieving consent purpose", log.String("purpose_id", purposeID), log.String("org_id", orgID))

	purpose, err := s.stores.ConsentPurpose.GetPurposeByID(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent purpose", log.Error(err), log.String("purpose_id", purposeID))
		return nil, serviceerror.CustomServiceError(ErrorPurposeNotFound, "purpose purpose not found")
	}

	return purpose, nil
}

// ListPurposes retrieves a list of consent purposes with optional filters
func (s *consentPurposeService) ListPurposes(ctx context.Context, orgID, name string, clientIDs []string, purposeNames []string, offset, limit int) ([]model.ConsentPurpose, int, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Debug("Listing consent purposes",
		log.String("org_id", orgID),
		log.String("name", name),
		log.Int("offset", offset),
		log.Int("limit", limit))

	purposes, total, err := s.stores.ConsentPurpose.ListPurposes(ctx, orgID, name, clientIDs, purposeNames, offset, limit)
	if err != nil {
		logger.Error("Failed to list consent purposes", log.Error(err))
		return nil, 0, &ErrorInternalServerError
	}

	return purposes, total, nil
}

// UpdatePurpose updates an existing consent purpose
func (s *consentPurposeService) UpdatePurpose(ctx context.Context, purposeID string, req model.UpdateRequest, orgID, clientID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Updating consent purpose",
		log.String("purpose_id", purposeID),
		log.String("name", req.Name),
		log.String("org_id", orgID))

	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		logger.Warn("Purpose update request validation failed", log.String("error", err.Error()))
		return nil, err
	}

	// Check if purpose exists
	existingPurpose, err := s.stores.ConsentPurpose.GetPurposeByID(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent purpose", log.Error(err), log.String("purpose_id", purposeID))
		return nil, serviceerror.CustomServiceError(ErrorPurposeNotFound, "purpose not found")
	}

	// Verify client ownership
	if existingPurpose.ClientID != clientID {
		logger.Warn("Client does not own this consent purpose",
			log.String("purpose_client_id", existingPurpose.ClientID),
			log.String("request_client_id", clientID))
		return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists, "you do not have permission to update this consent purpose")
	}

	// Check if purpose is being used in any consents
	inUse, checkErr := s.stores.Consent.CheckPurposeUsedInConsents(ctx, purposeID, orgID)
	if checkErr != nil {
		logger.Error("Failed to check if purpose is in use", log.Error(checkErr))
		return nil, &ErrorInternalServerError
	}
	if inUse {
		logger.Warn("Cannot update purpose that is in use by consents", log.String("purpose_id", purposeID))
		return nil, serviceerror.CustomServiceError(ErrorPurposeInUse, "cannot update purpose that is currently used in consents")
	}

	// Check if new name conflicts with another purpose (excluding current name)
	exists, dbErr := s.stores.ConsentPurpose.CheckPurposeNameExists(ctx, req.Name, clientID, orgID, &purposeID)
	if dbErr != nil {
		logger.Error("Failed to check purpose name existence", log.Error(dbErr))
		return nil, &ErrorInternalServerError
	}
	if exists {
		logger.Warn("Purpose name already exists for this client", log.String("name", req.Name))
		return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists, fmt.Sprintf("purpose with name '%s' already exists for this client", req.Name))
	}

	// Validate purpose names exist
	purposeNameToID, validationErr := s.validatePurposeNamesExist(ctx, req.Elements, orgID)
	if validationErr != nil {
		return nil, validationErr
	}

	// Check for duplicate purpose names
	if duplicateErr := s.checkDuplicatePurposeNames(req.Elements); duplicateErr != nil {
		return nil, duplicateErr
	}

	// Update purpose
	now := time.Now().Unix()
	desc := &req.Description
	if req.Description == "" {
		desc = nil
	}

	purpose := &model.ConsentPurpose{
		ID:          purposeID,
		Name:        req.Name,
		Description: desc,
		ClientID:    clientID,
		CreatedTime: existingPurpose.CreatedTime,
		UpdatedTime: now,
		OrgID:       orgID,
	}

	// Execute transaction for purpose update
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.UpdatePurpose(tx, purpose)
		},
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.DeletePurposeElements(tx, purposeID, orgID)
		},
	}

	// Add new purpose mappings
	for _, elem := range req.Elements {
		elementID := purposeNameToID[elem.ElementName]
		isMandatory := elem.IsMandatory
		elementName := elem.ElementName

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.LinkElementToPurpose(tx, purposeID, elementID, orgID, isMandatory)
		})

		purpose.Elements = append(purpose.Elements, model.PurposeElement{
			ElementID:   elementID,
			ElementName: elementName,
			IsMandatory: isMandatory,
		})
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to update consent purpose", log.Error(err))
		return nil, &ErrorInternalServerError
	}

	logger.Info("Purpose updated successfully", log.String("purpose_id", purposeID))
	return purpose, nil
}

// DeletePurpose deletes a consent purpose
func (s *consentPurposeService) DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Deleting consent purpose", log.String("purpose_id", purposeID), log.String("org_id", orgID))

	// Check if purpose exists
	_, err := s.stores.ConsentPurpose.GetPurposeByID(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent purpose", log.Error(err))
		return serviceerror.CustomServiceError(ErrorPurposeNotFound, "purpose purpose not found")
	}

	// Check if purpose is being used in any consents
	inUse, checkErr := s.stores.Consent.CheckPurposeUsedInConsents(ctx, purposeID, orgID)
	if checkErr != nil {
		logger.Error("Failed to check if purpose is in use", log.Error(checkErr))
		return &ErrorInternalServerError
	}
	if inUse {
		logger.Warn("Cannot delete purpose that is in use by consents", log.String("purpose_id", purposeID))
		return serviceerror.CustomServiceError(ErrorPurposeInUse, "cannot delete purpose that is currently used in consents")
	}

	// Execute transaction for deletion
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.DeletePurpose(tx, purposeID, orgID)
		},
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to delete consent purpose", log.Error(err))
		return &ErrorInternalServerError
	}

	logger.Info("Purpose purpose deleted successfully", log.String("purpose_id", purposeID))
	return nil
}

// validateCreateRequest validates the create request
func (s *consentPurposeService) validateCreateRequest(req model.CreateRequest) *serviceerror.ServiceError {
	if req.Name == "" {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "name is required")
	}
	if len(req.Name) > 255 {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "name must not exceed 255 characters")
	}
	if len(req.Description) > 1024 {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "description must not exceed 1024 characters")
	}
	if len(req.Elements) == 0 {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "at least one purpose is required")
	}
	for _, purpose := range req.Elements {
		if purpose.ElementName == "" {
			return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "purpose name is required")
		}
	}
	return nil
}

// validateUpdateRequest validates the update request
func (s *consentPurposeService) validateUpdateRequest(req model.UpdateRequest) *serviceerror.ServiceError {
	if req.Name == "" {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "name is required")
	}
	if len(req.Name) > 255 {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "name must not exceed 255 characters")
	}
	if len(req.Description) > 1024 {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "description must not exceed 1024 characters")
	}
	if len(req.Elements) == 0 {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "at least one purpose is required")
	}
	for _, purpose := range req.Elements {
		if purpose.ElementName == "" {
			return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "purpose name is required")
		}
	}
	return nil
}

// validatePurposeNamesExist validates that all element names exist and returns a map of name -> ID
func (s *consentPurposeService) validatePurposeNamesExist(ctx context.Context, purposes []model.ElementInput, orgID string) (map[string]string, *serviceerror.ServiceError) {
	elementNames := make([]string, len(purposes))
	for i, p := range purposes {
		elementNames[i] = p.ElementName
	}

	elementNameToID, err := s.stores.ConsentElement.GetIDsByNames(ctx, elementNames, orgID)
	if err != nil {
		return nil, &ErrorInternalServerError
	}

	// Check that all elements were found
	for _, purpose := range purposes {
		if _, found := elementNameToID[purpose.ElementName]; !found {
			return nil, serviceerror.CustomServiceError(ErrorInvalidRequestBody, fmt.Sprintf("element '%s' does not exist", purpose.ElementName))
		}
	}

	return elementNameToID, nil
}

// checkDuplicatePurposeNames checks for duplicate purpose names within the request
func (s *consentPurposeService) checkDuplicatePurposeNames(purposes []model.ElementInput) *serviceerror.ServiceError {
	seen := make(map[string]bool)
	for _, purpose := range purposes {
		if seen[purpose.ElementName] {
			return serviceerror.CustomServiceError(ErrorInvalidRequestBody, fmt.Sprintf("duplicate purpose '%s' found in request", purpose.ElementName))
		}
		seen[purpose.ElementName] = true
	}
	return nil
}
