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
	"time"

	"github.com/wso2/openfgc/consent-server/internal/consentelement/model"
	"github.com/wso2/openfgc/consent-server/internal/consentelement/validator"
	dbmodel "github.com/wso2/openfgc/consent-server/internal/system/database/model"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
	"github.com/wso2/openfgc/consent-server/internal/system/utils"
)

// ConsentElementService defines the exported service interface.
// All inputs and return types are clean Go types — no json tags.
type ConsentElementService interface {
	// CreateElementsInBatch creates elements with partial success — failures do not block other items.
	CreateElementsInBatch(ctx context.Context, inputs []model.CreateElementInput, orgID string) (*model.BatchCreateOutput, *serviceerror.ServiceError)

	// GetElement returns the latest version of an element.
	GetElement(ctx context.Context, elementID, orgID string) (*model.ElementVersion, *serviceerror.ServiceError)

	// GetElementVersion returns a specific version of an element.
	GetElementVersion(ctx context.Context, elementID string, version int, orgID string) (*model.ElementVersion, *serviceerror.ServiceError)

	// ListElementVersions returns all versions of one element ordered ascending.
	ListElementVersions(ctx context.Context, elementID, orgID string) (*model.ElementVersionListOutput, *serviceerror.ServiceError)

	// ListElements returns paginated latest versions matching the given filters.
	ListElements(ctx context.Context, orgID string, filters model.ElementListFilter) (*model.ElementListOutput, *serviceerror.ServiceError)

	// CreateElementVersion appends a new version to an existing element.
	CreateElementVersion(ctx context.Context, elementID string, input model.CreateElementVersionInput, orgID string) (*model.ElementVersion, *serviceerror.ServiceError)

	// DeleteElementVersion deletes a specific version. Returns 409 if referenced by a purpose.
	// Deleting the last version also deletes the element.
	DeleteElementVersion(ctx context.Context, elementID string, version int, orgID string) *serviceerror.ServiceError
}

// consentElementService implements the ConsentElementService interface.
type consentElementService struct {
	stores *stores.StoreRegistry
}

// NewConsentElementService constructs a ConsentElementService backed by the given stores.
func NewConsentElementService(registry *stores.StoreRegistry) ConsentElementService {
	return &consentElementService{stores: registry}
}

// CreateElementsInBatch creates multiple elements. Each item is processed independently;
// per-item failures are collected and returned as FAILED results, not as a top-level error.
func (s *consentElementService) CreateElementsInBatch(ctx context.Context, inputs []model.CreateElementInput, orgID string) (*model.BatchCreateOutput, *serviceerror.ServiceError) {
	if len(inputs) == 0 {
		return nil, &ErrorAtLeastOneElement
	}

	results := make([]model.CreateElementOutput, 0, len(inputs))
	for _, input := range inputs {
		elementVersion, svcErr := s.createSingleElement(ctx, input, orgID)
		if svcErr != nil {
			msg := svcErr.Description
			results = append(results, model.CreateElementOutput{Status: "FAILED", Error: &msg})
		} else {
			results = append(results, model.CreateElementOutput{Status: "SUCCESS", Element: elementVersion})
		}
	}

	return &model.BatchCreateOutput{Results: results}, nil
}

func (s *consentElementService) createSingleElement(ctx context.Context, input model.CreateElementInput, orgID string) (*model.ElementVersion, *serviceerror.ServiceError) {
	if svcErr := validateCreateInput(input); svcErr != nil {
		return nil, svcErr
	}

	if input.Namespace == "" {
		input.Namespace = model.DefaultNamespace
	}

	store := s.stores.ConsentElement
	existing, err := store.GetByNameAndNamespace(ctx, input.Name, input.Namespace, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorCreateElement, fmt.Sprintf("failed to check name existence: %v", err))
	}
	if existing != nil {
		return nil, serviceerror.CustomServiceError(ErrorElementNameExists,
			fmt.Sprintf("element with name '%s' and namespace '%s' already exists", input.Name, input.Namespace))
	}

	elementVersion := &model.ElementVersion{
		VersionID:   utils.GenerateUUID(),
		ID:          utils.GenerateUUID(),
		Name:        input.Name,
		Namespace:   input.Namespace,
		Type:        input.Type,
		VersionNum:  1,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Schema:      input.Schema,
		CreatedTime: time.Now().UnixMilli(),
		OrgID:       orgID,
		Properties:  input.Properties,
	}

	if err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error { return store.CreateVersion(tx, elementVersion) },
	}); err != nil {
		return nil, serviceerror.CustomServiceError(ErrorCreateElement, fmt.Sprintf("failed to create element: %v", err))
	}
	return elementVersion, nil
}

// GetElement returns the latest version of an element.
func (s *consentElementService) GetElement(ctx context.Context, elementID, orgID string) (*model.ElementVersion, *serviceerror.ServiceError) {
	elementVersion, err := s.stores.ConsentElement.GetLatestVersion(ctx, elementID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to retrieve element: %v", err))
	}
	if elementVersion == nil {
		return nil, serviceerror.CustomServiceError(ErrorElementNotFound, fmt.Sprintf("element '%s' not found", elementID))
	}
	return elementVersion, nil
}

// GetElementVersion returns a specific version of an element.
func (s *consentElementService) GetElementVersion(ctx context.Context, elementID string, version int, orgID string) (*model.ElementVersion, *serviceerror.ServiceError) {
	elementVersion, err := s.stores.ConsentElement.GetVersion(ctx, elementID, version, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to retrieve element version: %v", err))
	}
	if elementVersion == nil {
		return nil, serviceerror.CustomServiceError(ErrorElementNotFound,
			fmt.Sprintf("element '%s' version %d not found", elementID, version))
	}
	return elementVersion, nil
}

// ListElementVersions returns all versions of one element ordered ascending.
func (s *consentElementService) ListElementVersions(ctx context.Context, elementID, orgID string) (*model.ElementVersionListOutput, *serviceerror.ServiceError) {
	store := s.stores.ConsentElement
	exists, err := store.ElementExists(ctx, elementID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to check element: %v", err))
	}
	if !exists {
		return nil, serviceerror.CustomServiceError(ErrorElementNotFound, fmt.Sprintf("element '%s' not found", elementID))
	}

	versions, err := store.ListVersions(ctx, elementID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to list element versions: %v", err))
	}

	result := &model.ElementVersionListOutput{ElementID: elementID, Versions: versions}
	if len(versions) > 0 {
		result.Name = versions[0].Name
		result.Namespace = versions[0].Namespace
		result.Type = versions[0].Type
	}
	return result, nil
}

// ListElements returns paginated latest versions matching the given filters.
func (s *consentElementService) ListElements(ctx context.Context, orgID string, filters model.ElementListFilter) (*model.ElementListOutput, *serviceerror.ServiceError) {
	if filters.Limit <= 0 {
		filters.Limit = 100
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	versions, total, err := s.stores.ConsentElement.List(ctx, orgID, filters)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to list elements: %v", err))
	}
	return &model.ElementListOutput{
		Data:   versions,
		Total:  total,
		Offset: filters.Offset,
		Count:  len(versions),
		Limit:  filters.Limit,
	}, nil
}

// CreateElementVersion appends a new immutable version to an existing element.
func (s *consentElementService) CreateElementVersion(ctx context.Context, elementID string, input model.CreateElementVersionInput, orgID string) (*model.ElementVersion, *serviceerror.ServiceError) {
	store := s.stores.ConsentElement

	latest, err := store.GetLatestVersion(ctx, elementID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to retrieve element: %v", err))
	}
	if latest == nil {
		return nil, serviceerror.CustomServiceError(ErrorElementNotFound, fmt.Sprintf("element '%s' not found", elementID))
	}

	if svcErr := validateVersionInput(latest.Type, input); svcErr != nil {
		return nil, svcErr
	}

	nextVersionNum := latest.VersionNum + 1
	elementVersion := &model.ElementVersion{
		VersionID:   utils.GenerateUUID(),
		ID:          elementID,
		Name:        latest.Name,
		Namespace:   latest.Namespace,
		Type:        latest.Type,
		VersionNum:  nextVersionNum,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Schema:      input.Schema,
		CreatedTime: time.Now().UnixMilli(),
		OrgID:       orgID,
		Properties:  input.Properties,
	}

	if err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error { return store.CreateVersion(tx, elementVersion) },
	}); err != nil {
		return nil, serviceerror.CustomServiceError(ErrorCreateElement, fmt.Sprintf("failed to create element version: %v", err))
	}
	return elementVersion, nil
}

// DeleteElementVersion deletes a specific version. Returns 409 if referenced by a purpose.
// Deleting the last version also removes the element entity.
func (s *consentElementService) DeleteElementVersion(ctx context.Context, elementID string, version int, orgID string) *serviceerror.ServiceError {
	store := s.stores.ConsentElement

	elementVersion, err := store.GetVersion(ctx, elementID, version, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to retrieve element version: %v", err))
	}
	if elementVersion == nil {
		return serviceerror.CustomServiceError(ErrorElementNotFound,
			fmt.Sprintf("element '%s' version %d not found", elementID, version))
	}

	referenced, err := store.IsVersionReferencedByPurpose(ctx, elementVersion.VersionID, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to check version references: %v", err))
	}
	if referenced {
		return serviceerror.CustomServiceError(ErrorVersionReferencedByPurpose,
			fmt.Sprintf("element '%s' version %d is referenced by one or more purposes and cannot be deleted", elementID, version))
	}

	allVersions, err := store.ListVersions(ctx, elementID, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(ErrorReadElement, fmt.Sprintf("failed to check versions: %v", err))
	}
	isLastVersion := len(allVersions) == 1

	txOps := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error { return store.DeleteVersion(tx, elementVersion.VersionID, orgID) },
	}
	if isLastVersion {
		txOps = append(txOps, func(tx dbmodel.TxInterface) error { return store.DeleteElement(tx, elementID, orgID) })
	}

	if err := s.stores.ExecuteTransaction(txOps); err != nil {
		return serviceerror.CustomServiceError(ErrorDeleteElement, fmt.Sprintf("failed to delete element version: %v", err))
	}
	return nil
}

// validateVersionInput validates the mutable fields of a new element version.
// The element type is inherited from the existing element and cannot be changed.
func validateVersionInput(elementType string, input model.CreateElementVersionInput) *serviceerror.ServiceError {
	if input.Description != nil && len(*input.Description) > 1024 {
		return &ErrorElementDescriptionTooLong
	}
	if elementTypeDef, err := validator.GetTypeRegistry().Get(elementType); err == nil {
		if verr := elementTypeDef.ValidateSchema(input.Schema); verr != nil {
			return serviceerror.CustomServiceError(ErrorValidateElement, verr.Message)
		}
		if errs := elementTypeDef.ValidateProperties(input.Properties); len(errs) > 0 {
			return serviceerror.CustomServiceError(ErrorValidateElement, errs[0].Message)
		}
	}
	return nil
}

// validateCreateInput validates a single element create input.
// schemaStr is already parsed from the raw request by the handler.
func validateCreateInput(input model.CreateElementInput) *serviceerror.ServiceError {
	if input.Name == "" {
		return &ErrorElementNameRequired
	}
	if len(input.Name) > 255 {
		return &ErrorElementNameTooLong
	}
	if input.Description != nil && len(*input.Description) > 1024 {
		return &ErrorElementDescriptionTooLong
	}
	if input.Type == "" {
		return &ErrorElementTypeRequired
	}
	switch input.Type {
	case model.ElementTypeBasic, model.ElementTypeJSON, model.ElementTypeXML:
	default:
		return serviceerror.CustomServiceError(ErrorInvalidElementType, fmt.Sprintf("invalid element type: %s", input.Type))
	}
	if elementTypeDef, err := validator.GetTypeRegistry().Get(input.Type); err == nil {
		if verr := elementTypeDef.ValidateSchema(input.Schema); verr != nil {
			return serviceerror.CustomServiceError(ErrorValidateElement, verr.Message)
		}
		if errs := elementTypeDef.ValidateProperties(input.Properties); len(errs) > 0 {
			return serviceerror.CustomServiceError(ErrorValidateElement, errs[0].Message)
		}
	}
	return nil
}
