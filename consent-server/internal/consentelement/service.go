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

import (
	"context"
	"fmt"
	"strings"

	"github.com/wso2/openfgc/internal/consentelement/model"
	"github.com/wso2/openfgc/internal/consentelement/validators"
	dbmodel "github.com/wso2/openfgc/internal/system/database/model"
	"github.com/wso2/openfgc/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/internal/system/log"
	"github.com/wso2/openfgc/internal/system/stores"
	"github.com/wso2/openfgc/internal/system/utils"
)

// ConsentElementService defines the exported service interface
type ConsentElementService interface {
	CreateElementsInBatch(ctx context.Context, requests []model.ConsentElementCreateRequest, orgID string) ([]model.ConsentElement, *serviceerror.ServiceError)
	GetElement(ctx context.Context, elementID, orgID string) (*model.ConsentElement, *serviceerror.ServiceError)
	ListElements(ctx context.Context, orgID string, limit, offset int, name string) ([]model.ConsentElement, int, *serviceerror.ServiceError)
	UpdateElement(ctx context.Context, elementID string, req model.ConsentElementUpdateRequest, orgID string) (*model.ConsentElement, *serviceerror.ServiceError)
	DeleteElement(ctx context.Context, elementID, orgID string) *serviceerror.ServiceError
	ValidateElementNames(ctx context.Context, orgID string, elementNames []string) ([]string, *serviceerror.ServiceError)
}

// consentElementService implements the ConsentElementService interface
type consentElementService struct {
	stores *stores.StoreRegistry
}

// newConsentElementService creates a new consent element service
func newConsentElementService(registry *stores.StoreRegistry) ConsentElementService {
	return &consentElementService{
		stores: registry,
	}
}

// CreateElementsInBatch creates multiple consent elements in a single transaction
// Either all elements are created or none (atomic operation)
func (service *consentElementService) CreateElementsInBatch(ctx context.Context, requests []model.ConsentElementCreateRequest, orgID string) ([]model.ConsentElement, *serviceerror.ServiceError) {
	// Validate inputs
	if len(requests) == 0 {
		return nil, &ErrorAtLeastOneElement
	}

	store := service.stores.ConsentElement

	// Pre-validate all requests and check for duplicate names within the batch
	namesSeen := make(map[string]bool)
	for i, req := range requests {
		// Validate request
		if valErr := service.validateCreateRequest(req); valErr != nil {
			// Return error with index information
			return nil, serviceerror.CustomServiceError(*valErr, fmt.Sprintf("invalid request at index %d: %s", i, valErr.Description))
		}

		// Check for duplicate names within the batch
		if namesSeen[req.Name] {
			return nil, serviceerror.CustomServiceError(ErrorDuplicateNameInBatch, fmt.Sprintf("duplicate element name '%s' in request batch at index %d", req.Name, i))
		}
		namesSeen[req.Name] = true

		// Check if element name already exists in database
		exists, dbErr := store.CheckNameExists(ctx, req.Name, orgID)
		if dbErr != nil {
			return nil, serviceerror.CustomServiceError(ErrorCheckNameExistence, fmt.Sprintf("failed to validate element name at index %d: %v", i, dbErr))
		}
		if exists {
			return nil, serviceerror.CustomServiceError(ErrorElementNameExists, fmt.Sprintf("element name '%s' already exists for this organization (at index %d)", req.Name, i))
		}
	}

	// Prepare transaction operations
	var queries []func(tx dbmodel.TxInterface) error
	createdElements := make([]model.ConsentElement, 0, len(requests))

	// Create all elements within the transaction
	for _, req := range requests {
		elementID := utils.GenerateUUID()
		desc := req.Description

		element := &model.ConsentElement{
			ID:          elementID,
			Name:        req.Name,
			Description: &desc,
			Type:        req.Type,
			OrgID:       orgID,
			Properties:  req.Properties,
		}

		// Add element creation to transaction
		elementCopy := *element // Create a copy for the closure
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return store.Create(tx, &elementCopy)
		})

		// Add properties if provided
		if len(req.Properties) > 0 {
			properties := make([]model.ConsentElementProperty, 0, len(req.Properties))
			for key, value := range req.Properties {
				prop := model.ConsentElementProperty{
					ElementID: elementID,
					Key:       key,
					Value:     value,
					OrgID:     orgID,
				}
				properties = append(properties, prop)
			}

			// Capture properties for this iteration
			propsCopy := properties
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return store.CreateProperties(tx, propsCopy)
			})
		}

		createdElements = append(createdElements, *element)
	}

	// Execute all operations in a single transaction
	if err := service.stores.ExecuteTransaction(queries); err != nil {
		// Check if error is due to duplicate name constraint violation
		errMsg := err.Error()
		if strings.Contains(errMsg, "Duplicate entry") || strings.Contains(errMsg, "unique constraint") || strings.Contains(errMsg, "already exists") {
			// Extract element name from error if possible, otherwise use generic message
			return nil, serviceerror.CustomServiceError(ErrorElementNameExists, "one or more element names already exist for this organization")
		}
		return nil, serviceerror.CustomServiceError(ErrorCreateElement, fmt.Sprintf("failed to create elements in batch: %v", err))
	}

	return createdElements, nil
}

// GetElement retrieves a consent element by ID
func (service *consentElementService) GetElement(ctx context.Context, elementID, orgID string) (*model.ConsentElement, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving consent element",
		log.String("element_id", elementID),
		log.String("org_id", orgID),
	)

	elementStore := service.stores.ConsentElement
	element, err := elementStore.GetByID(ctx, elementID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve element",
			log.Error(err),
			log.String("element_id", elementID),
		)
		return nil, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to retrieve element: %v", err))
	}
	if element == nil {
		logger.Warn("Element not found", log.String("element_id", elementID))
		return nil, serviceerror.CustomServiceError(ErrorElementNotFound, fmt.Sprintf("element with ID '%s' not found", elementID))
	}

	// Load properties
	properties, err := elementStore.GetPropertiesByElementID(ctx, elementID, orgID)
	if err != nil {
		logger.Error("Failed to load element properties",
			log.Error(err),
			log.String("element_id", elementID),
		)
		return nil, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to load properties: %v", err))
	}

	// Convert properties to map
	if element.Properties == nil {
		element.Properties = make(map[string]string)
	}
	for _, prop := range properties {
		element.Properties[prop.Key] = prop.Value
	}

	logger.Debug("Element retrieved successfully",
		log.String("element_id", elementID),
		log.String("name", element.Name),
		log.Int("properties_count", len(properties)),
	)
	return element, nil
}

// ListElements retrieves paginated list of consent elements with optional name filter
func (service *consentElementService) ListElements(ctx context.Context, orgID string, limit, offset int, name string) ([]model.ConsentElement, int, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Listing consent elements",
		log.String("org_id", orgID),
		log.Int("limit", limit),
		log.Int("offset", offset),
		log.String("name_filter", name),
	)

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	elementStore := service.stores.ConsentElement
	elements, total, err := elementStore.List(ctx, orgID, limit, offset, name)
	if err != nil {
		logger.Error("Failed to list elements",
			log.Error(err),
			log.String("org_id", orgID),
		)
		return nil, 0, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to list elements: %v", err))
	}

	// Load properties for each element
	for i := range elements {
		properties, propErr := elementStore.GetPropertiesByElementID(ctx, elements[i].ID, orgID)
		if propErr != nil {
			logger.Error("Failed to load properties for element",
				log.Error(propErr),
				log.String("element_id", elements[i].ID),
			)
			return nil, 0, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to load properties: %v", propErr))
		}

		if elements[i].Properties == nil {
			elements[i].Properties = make(map[string]string)
		}
		for _, prop := range properties {
			elements[i].Properties[prop.Key] = prop.Value
		}
	}

	logger.Debug("Elements listed successfully",
		log.Int("count", len(elements)),
		log.Int("total", total),
	)
	return elements, total, nil
}

// UpdateElement updates an existing consent element
func (service *consentElementService) UpdateElement(ctx context.Context, elementID string, req model.ConsentElementUpdateRequest, orgID string) (*model.ConsentElement, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating consent element",
		log.String("element_id", elementID),
		log.String("name", req.Name),
		log.String("org_id", orgID),
	)

	// Validate request
	if err := service.validateUpdateRequest(req); err != nil {
		logger.Warn("Update element request validation failed", log.String("error", err.Error()))
		return nil, err
	}

	// Check if element exists
	elementStore := service.stores.ConsentElement
	existing, err := elementStore.GetByID(ctx, elementID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve existing element",
			log.Error(err),
			log.String("element_id", elementID),
		)
		return nil, serviceerror.CustomServiceError(ErrorUpdateElement, fmt.Sprintf("failed to retrieve element: %v", err))
	}
	if existing == nil {
		logger.Warn("Element not found for update", log.String("element_id", elementID))
		return nil, serviceerror.CustomServiceError(ErrorElementNotFound, fmt.Sprintf("element with ID '%s' not found", elementID))
	}

	// Check if the new name conflicts with another element (only if name is changing)
	if req.Name != existing.Name {
		exists, dbErr := elementStore.CheckNameExists(ctx, req.Name, orgID)
		if dbErr != nil {
			logger.Error("Failed to check element name existence during update",
				log.Error(dbErr),
				log.String("name", req.Name),
			)
			return nil, serviceerror.CustomServiceError(ErrorUpdateElement, fmt.Sprintf("failed to check name existence: %v", dbErr))
		}
		if exists {
			logger.Warn("Element name already exists for another element",
				log.String("name", req.Name),
				log.String("element_id", elementID),
			)
			return nil, serviceerror.CustomServiceError(ErrorElementNameExists, fmt.Sprintf("element with name '%s' already exists", req.Name))
		}
	}

	// Check if element is used in any consent purposes
	isUsed, err := service.stores.ConsentPurpose.IsElementUsedInPurposes(ctx, elementID, orgID)
	if err != nil {
		logger.Error("Failed to check if element is used in groups",
			log.Error(err),
			log.String("element_id", elementID),
		)
		return nil, serviceerror.CustomServiceError(ErrorUpdateElement, fmt.Sprintf("failed to check element usage: %v", err))
	}
	if isUsed {
		logger.Warn("Cannot update element that is used in consent purposes",
			log.String("element_id", elementID),
			log.String("element_name", existing.Name),
		)
		return nil, serviceerror.CustomServiceError(ErrorElementInUse, fmt.Sprintf("cannot update element '%s' as it is being used in one or more consent purposes", existing.Name))
	}

	// Update element fields
	element := &model.ConsentElement{
		ID:          elementID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		OrgID:       orgID,
	}

	// Prepare properties if provided
	var properties []model.ConsentElementProperty
	if req.Properties != nil {
		element.Properties = req.Properties
		if len(req.Properties) > 0 {
			properties = make([]model.ConsentElementProperty, 0, len(req.Properties))
			for key, value := range req.Properties {
				prop := model.ConsentElementProperty{
					ElementID: elementID,
					Key:       key,
					Value:     value,
					OrgID:     orgID,
				}
				properties = append(properties, prop)
			}
		}
	}

	// Execute all updates in a transaction
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return elementStore.Update(tx, element)
		},
	}
	if req.Properties != nil {
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return elementStore.DeletePropertiesByElementID(tx, elementID, orgID)
		})
		if len(properties) > 0 {
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return elementStore.CreateProperties(tx, properties)
			})
		}
	}

	logger.Debug("Executing transaction for element update",
		log.Int("properties_count", len(properties)),
	)
	err = service.stores.ExecuteTransaction(queries)
	if err != nil {
		logger.Error("Transaction failed for element update",
			log.Error(err),
			log.String("element_id", elementID),
		)
		return nil, serviceerror.CustomServiceError(ErrorUpdateElement, fmt.Sprintf("failed to update element: %v", err))
	}

	logger.Info("Element updated successfully",
		log.String("element_id", elementID),
		log.String("name", element.Name),
	)
	return element, nil
}

// DeleteElement deletes a consent element
func (service *consentElementService) DeleteElement(ctx context.Context, elementID, orgID string) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Deleting consent element",
		log.String("element_id", elementID),
		log.String("org_id", orgID),
	)

	// Check if element exists
	elementStore := service.stores.ConsentElement
	existing, err := elementStore.GetByID(ctx, elementID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve element for deletion",
			log.Error(err),
			log.String("element_id", elementID),
		)
		return serviceerror.CustomServiceError(ErrorDeleteElement, fmt.Sprintf("failed to retrieve element: %v", err))
	}
	if existing == nil {
		logger.Warn("Element not found for deletion", log.String("element_id", elementID))
		return serviceerror.CustomServiceError(ErrorElementNotFound, fmt.Sprintf("element with ID '%s' not found", elementID))
	}

	// Check if element is used in any consent purposes
	isUsed, err := service.stores.ConsentPurpose.IsElementUsedInPurposes(ctx, elementID, orgID)
	if err != nil {
		logger.Error("Failed to check if element is used in groups",
			log.Error(err),
			log.String("element_id", elementID),
		)
		return serviceerror.CustomServiceError(ErrorDeleteElement, fmt.Sprintf("failed to check element usage: %v", err))
	}
	if isUsed {
		logger.Warn("Cannot delete element that is used in consent purposes",
			log.String("element_id", elementID),
			log.String("element_name", existing.Name),
		)
		return serviceerror.CustomServiceError(ErrorDeleteElement, fmt.Sprintf("cannot delete element '%s' as it is being used in one or more consent purposes", existing.Name))
	}

	// Delete properties and element in a transaction
	logger.Debug("Executing transaction for element deletion")
	err = service.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return elementStore.DeletePropertiesByElementID(tx, elementID, orgID)
		},
		func(tx dbmodel.TxInterface) error {
			return elementStore.Delete(tx, elementID, orgID)
		},
	})
	if err != nil {
		logger.Error("Transaction failed for element deletion",
			log.Error(err),
			log.String("element_id", elementID),
		)
		return serviceerror.CustomServiceError(ErrorDeleteElement, fmt.Sprintf("failed to delete element: %v", err))
	}

	logger.Info("Element deleted successfully",
		log.String("element_id", elementID),
		log.String("name", existing.Name),
	)
	return nil
}

// ValidateElementNames validates a list of element names and returns only the valid ones
func (service *consentElementService) ValidateElementNames(ctx context.Context, orgID string, elementNames []string) ([]string, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Validating element names",
		log.String("org_id", orgID),
		log.Int("name_count", len(elementNames)),
	)

	// Validate input
	if len(elementNames) == 0 {
		logger.Warn("No element names provided for validation")
		return nil, &ErrorAtLeastOneElementName
	}

	elementStore := service.stores.ConsentElement

	// Get elements that exist
	elementIDMap, err := elementStore.GetIDsByNames(ctx, elementNames, orgID)
	if err != nil {
		logger.Error("Failed to validate element names",
			log.Error(err),
			log.String("org_id", orgID),
		)
		return nil, serviceerror.CustomServiceError(ErrorValidateElement, fmt.Sprintf("failed to validate element names: %v", err))
	}

	// Extract valid names from the map
	validNames := make([]string, 0, len(elementIDMap))
	for name := range elementIDMap {
		validNames = append(validNames, name)
	}

	// Return error if no valid elements found
	if len(validNames) == 0 {
		logger.Warn("No valid elements found")
		return nil, &ErrorNoValidElements
	}

	logger.Debug("Element names validated",
		log.Int("valid_count", len(validNames)),
		log.Int("requested_count", len(elementNames)),
	)
	return validNames, nil
}

// validateCreateRequest validates create request
func (service *consentElementService) validateCreateRequest(req model.ConsentElementCreateRequest) *serviceerror.ServiceError {
	if req.Name == "" {
		return &ErrorElementNameRequired
	}
	if len(req.Name) > 255 {
		return &ErrorElementNameTooLong
	}
	if len(req.Description) > 1024 {
		return &ErrorElementDescriptionTooLong
	}
	if req.Type == "" {
		return &ErrorElementTypeRequired
	}

	// Validate element type using validators
	handler, err := validators.GetHandler(req.Type)
	if err != nil {
		return serviceerror.CustomServiceError(ErrorInvalidElementType, fmt.Sprintf("invalid element type: %s", req.Type))
	}

	// Validate properties using type handler
	if validationErrors := handler.ValidateProperties(req.Properties); len(validationErrors) > 0 {
		return serviceerror.CustomServiceError(ErrorValidateElement, fmt.Sprintf("property validation failed: %v", validationErrors[0].Message))
	}

	return nil
}

// validateUpdateRequest validates update request
func (service *consentElementService) validateUpdateRequest(req model.ConsentElementUpdateRequest) *serviceerror.ServiceError {
	if req.Name == "" {
		return &ErrorElementNameRequired
	}
	if len(req.Name) > 255 {
		return &ErrorElementNameTooLong
	}
	if req.Description != nil && len(*req.Description) > 1024 {
		return &ErrorElementDescriptionTooLong
	}
	if req.Type == "" {
		return &ErrorElementTypeRequired
	}

	// Validate element type using validators
	handler, err := validators.GetHandler(req.Type)
	if err != nil {
		return serviceerror.CustomServiceError(ErrorInvalidElementType, fmt.Sprintf("invalid element type: %s", req.Type))
	}

	// Validate properties using type handler
	if validationErrors := handler.ValidateProperties(req.Properties); len(validationErrors) > 0 {
		return serviceerror.CustomServiceError(ErrorValidateElement, fmt.Sprintf("property validation failed: %v", validationErrors[0].Message))
	}

	return nil
}
