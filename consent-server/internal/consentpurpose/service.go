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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wso2/openfgc/internal/consentpurpose/model"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/internal/system/utils"
)

// ConsentPurposeService manages consent purposes within an organization.
type ConsentPurposeService interface {
	CreatePurpose(ctx context.Context, req model.CreateRequest, orgID, clientID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	GetPurpose(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	ListPurposes(ctx context.Context, orgID, name string, clientIDs []string, elementNames []string, offset, limit int) ([]model.ConsentPurpose, int, *serviceerror.ServiceError)
	UpdatePurpose(ctx context.Context, purposeID string, req model.UpdateRequest, orgID, clientID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError
}

// consentPurposeService implements the ConsentPurposeService interface
type consentPurposeService struct {
	stores *stores.StoreRegistry
}

// NewConsentPurposeService creates a new consent purpose service
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
		logger.Warn("Consent purpose create request validation failed", log.String("error", err.Error()))
		return nil, err
	}

	// Check if purpose name already exists for this client
	exists, dbErr := s.stores.ConsentPurpose.CheckPurposeNameExists(ctx, req.Name, clientID, orgID, nil)
	if dbErr != nil {
		logger.Error("Failed to check purpose name existence", log.Error(dbErr), log.String("name", req.Name))
		return nil, &ErrorCheckNameExistence
	}
	if exists {
		logger.Warn("Consent purpose name already exists for this client", log.String("name", req.Name), log.String("client_id", clientID))
		return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists, fmt.Sprintf("consent purpose with name '%s' already exists for this client", req.Name))
	}

	// Validate that all element names exist
	elementNameToID, err := s.validateElementNamesExist(ctx, req.Elements, orgID)
	if err != nil {
		return nil, err
	}

	// Check for duplicate element names within the request
	if duplicateErr := s.checkDuplicateElementNames(req.Elements); duplicateErr != nil {
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

	// Add element linking operations
	for _, elem := range req.Elements {
		elementID := elementNameToID[elem.ElementName]
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

	logger.Info("Consent purpose created successfully", log.String("purpose_id", purposeID))
	return purpose, nil
}

// GetPurpose retrieves a consent purpose by ID
func (s *consentPurposeService) GetPurpose(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Debug("Retrieving consent purpose", log.String("purpose_id", purposeID), log.String("org_id", orgID))

	purpose, err := s.stores.ConsentPurpose.GetPurposeByID(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent purpose", log.Error(err), log.String("purpose_id", purposeID))
		// Check if purpose was not found
		if errors.Is(err, ErrPurposeNotFound) {
			return nil, &ErrorPurposeNotFound
		}
		return nil, &ErrorRetrievePurpose
	}

	return purpose, nil
}

// ListPurposes retrieves a list of consent purposes with optional filters
func (s *consentPurposeService) ListPurposes(ctx context.Context, orgID, name string, clientIDs []string, elementNames []string, offset, limit int) ([]model.ConsentPurpose, int, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Debug("Listing consent purposes",
		log.String("org_id", orgID),
		log.String("name", name),
		log.String("client_ids", strings.Join(clientIDs, ",")),
		log.String("element_names", strings.Join(elementNames, ",")),
		log.Int("offset", offset),
		log.Int("limit", limit))

	purposes, total, err := s.stores.ConsentPurpose.ListPurposes(ctx, orgID, name, clientIDs, elementNames, offset, limit)
	if err != nil {
		logger.Error("Failed to list consent purposes", log.Error(err))
		return nil, 0, &ErrorListPurposes
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
		// Check if purpose was not found
		if errors.Is(err, ErrPurposeNotFound) {
			return nil, &ErrorPurposeNotFound
		}
		return nil, &ErrorRetrievePurpose
	}

	// Verify client ownership
	if existingPurpose.ClientID != clientID {
		logger.Warn("Client does not own this consent purpose",
			log.String("purpose_client_id", existingPurpose.ClientID),
			log.String("request_client_id", clientID))
		return nil, serviceerror.CustomServiceError(ErrorValidationFailed, "you do not have permission to update this consent purpose")
	}

	// Check if purpose is being used in any consents
	inUse, checkErr := s.stores.Consent.CheckPurposeUsedInConsents(ctx, purposeID, orgID)
	if checkErr != nil {
		logger.Error("Failed to check if purpose is in use", log.Error(checkErr))
		return nil, &ErrorCheckPurposeUsage
	}
	if inUse {
		logger.Warn("Cannot update purpose that is in use by consents", log.String("purpose_id", purposeID))
		return nil, serviceerror.CustomServiceError(ErrorPurposeInUse, "cannot update purpose that is currently used in consents")
	}

	// Check if new name conflicts with another purpose (excluding current name)
	exists, dbErr := s.stores.ConsentPurpose.CheckPurposeNameExists(ctx, req.Name, clientID, orgID, &purposeID)
	if dbErr != nil {
		logger.Error("Failed to check purpose name existence", log.Error(dbErr))
		return nil, &ErrorCheckNameExistence
	}
	if exists {
		logger.Warn("Purpose name already exists for this client", log.String("name", req.Name))
		return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists, fmt.Sprintf("purpose with name '%s' already exists for this client", req.Name))
	}

	// Validate element names exist
	elementNameToID, validationErr := s.validateElementNamesExist(ctx, req.Elements, orgID)
	if validationErr != nil {
		return nil, validationErr
	}

	// Check for duplicate element names
	if duplicateErr := s.checkDuplicateElementNames(req.Elements); duplicateErr != nil {
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
		elementID := elementNameToID[elem.ElementName]
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
		// Check if purpose was not found
		if errors.Is(err, ErrPurposeNotFound) {
			return &ErrorPurposeNotFound
		}
		return &ErrorRetrievePurpose
	}

	// Check if purpose is being used in any consents
	inUse, checkErr := s.stores.Consent.CheckPurposeUsedInConsents(ctx, purposeID, orgID)
	if checkErr != nil {
		logger.Error("Failed to check if purpose is in use", log.Error(checkErr))
		return &ErrorCheckPurposeUsage
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

	logger.Info("Purpose deleted successfully", log.String("purpose_id", purposeID))
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
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "at least one element is required")
	}
	for _, element := range req.Elements {
		if element.ElementName == "" {
			return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "element names cannot be empty")
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
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "at least one element is required")
	}
	for _, element := range req.Elements {
		if element.ElementName == "" {
			return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "element names cannot be empty")
		}
	}
	return nil
}

// validateElementNamesExist validates that all element names exist and returns a map of name -> ID
func (s *consentPurposeService) validateElementNamesExist(ctx context.Context, elements []model.ElementInput, orgID string) (map[string]string, *serviceerror.ServiceError) {
	elementNames := make([]string, len(elements))
	for i, elementInput := range elements {
		elementNames[i] = elementInput.ElementName
	}

	elementNameToID, err := s.stores.ConsentElement.GetIDsByNames(ctx, elementNames, orgID)
	if err != nil {
		return nil, &ErrorInternalServerError
	}

	// Check that all elements were found
	for _, element := range elements {
		if _, found := elementNameToID[element.ElementName]; !found {
			return nil, serviceerror.CustomServiceError(ErrorInvalidRequestBody, fmt.Sprintf("element '%s' does not exist", element.ElementName))
		}
	}

	return elementNameToID, nil
}

// checkDuplicateElementNames checks for duplicate element names within the request
func (s *consentPurposeService) checkDuplicateElementNames(elements []model.ElementInput) *serviceerror.ServiceError {
	seen := make(map[string]bool)
	for _, element := range elements {
		if seen[element.ElementName] {
			return serviceerror.CustomServiceError(ErrorInvalidRequestBody, fmt.Sprintf("duplicate element '%s' found in request", element.ElementName))
		}
		seen[element.ElementName] = true
	}
	return nil
}
