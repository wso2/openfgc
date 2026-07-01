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
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/wso2/openfgc/consent-server/internal/consentpurpose/model"
	elementmodel "github.com/wso2/openfgc/consent-server/internal/consentelement/model"
	dbmodel "github.com/wso2/openfgc/consent-server/internal/system/database/model"
	"github.com/wso2/openfgc/consent-server/internal/system/error/serviceerror"
	"github.com/wso2/openfgc/consent-server/internal/system/log"
	"github.com/wso2/openfgc/consent-server/internal/system/stores"
	"github.com/wso2/openfgc/consent-server/internal/system/utils"
)

// ConsentPurposeService manages versioned consent purposes within an organization.
// All inputs and return types are clean Go types — no json or db tags.
type ConsentPurposeService interface {
	// CreatePurpose creates a new purpose at version 1.
	// GroupID in input is set to orgID by the caller when the group-id header is absent.
	CreatePurpose(ctx context.Context, input model.CreatePurposeInput, orgID string) (*model.PurposeOutput, *serviceerror.ServiceError)

	// GetPurpose returns the latest version of a purpose.
	GetPurpose(ctx context.Context, purposeID, orgID string) (*model.PurposeOutput, *serviceerror.ServiceError)

	// ListPurposes returns a paginated list of purposes (latest version of each) matching the filters.
	ListPurposes(ctx context.Context, orgID string, filters model.PurposeListFilter) (*model.PurposeListOutput, *serviceerror.ServiceError)

	// GetPurposeVersions returns all versions of a purpose ordered ascending.
	GetPurposeVersions(ctx context.Context, purposeID, orgID string) (*model.PurposeVersionListOutput, *serviceerror.ServiceError)

	// CreatePurposeVersion appends a new version to an existing purpose.
	CreatePurposeVersion(ctx context.Context, purposeID string, input model.CreatePurposeVersionInput, orgID string) (*model.PurposeOutput, *serviceerror.ServiceError)

	// GetPurposeVersion returns a specific version of a purpose.
	GetPurposeVersion(ctx context.Context, purposeID string, version int, orgID string) (*model.PurposeOutput, *serviceerror.ServiceError)

	// DeletePurposeVersion deletes a specific version. Returns 409 if referenced by a consent.
	// Deleting the last version also deletes the purpose entity.
	DeletePurposeVersion(ctx context.Context, purposeID string, version int, orgID string) *serviceerror.ServiceError

	// DeletePurpose deletes a purpose and all its versions.
	// Returns 409 if any version is referenced by a consent.
	DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError
}

// consentPurposeService implements the ConsentPurposeService interface.
type consentPurposeService struct {
	stores *stores.StoreRegistry
}

// NewConsentPurposeService creates a new consent purpose service.
func NewConsentPurposeService(registry *stores.StoreRegistry) ConsentPurposeService {
	return &consentPurposeService{stores: registry}
}

// =============================================================================
// Create
// =============================================================================

// CreatePurpose creates a new purpose at version 1.
func (s *consentPurposeService) CreatePurpose(ctx context.Context, input model.CreatePurposeInput, orgID string) (*model.PurposeOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	if svcErr := s.validatePurposeInput(input.Name, input.Description); svcErr != nil {
		return nil, svcErr
	}

	existing, err := s.stores.ConsentPurpose.GetByNameAndGroupID(ctx, input.Name, input.GroupID, orgID)
	if err != nil {
		logger.Error("Failed to check purpose name existence", log.Error(err))
		return nil, &ErrorCheckNameExistence
	}
	if existing != nil {
		if input.GroupID == orgID {
			return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists,
				fmt.Sprintf("a purpose named '%s' already exists in this org", input.Name))
		}
		return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists,
			fmt.Sprintf("a purpose named '%s' already exists in this group", input.Name))
	}

	if input.GroupID != orgID {
		// Group-scoped creation: also block if an org-level purpose with the same name exists.
		// Org-level purposes are visible to all groups, so a same-named group-scoped purpose
		// would be ambiguous when referenced by name in a consent.
		orgLevel, err := s.stores.ConsentPurpose.GetByNameAndGroupID(ctx, input.Name, orgID, orgID)
		if err != nil {
			logger.Error("Failed to check org-level purpose name existence", log.Error(err))
			return nil, &ErrorCheckNameExistence
		}
		if orgLevel != nil {
			return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists,
				fmt.Sprintf("a purpose named '%s' already exists as an org-level purpose", input.Name))
		}
	} else {
		// Org-level creation: block if ANY purpose with the same name exists anywhere in the org
		// (org-level or group-scoped). An org-level purpose is visible to all groups, so sharing
		// its name with a group-scoped purpose would cause ambiguous resolution.
		exists, err := s.stores.ConsentPurpose.ExistsByNameInOrg(ctx, input.Name, orgID)
		if err != nil {
			logger.Error("Failed to check purpose name existence across org", log.Error(err))
			return nil, &ErrorCheckNameExistence
		}
		if exists {
			return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists,
				fmt.Sprintf("a purpose named '%s' already exists in this org", input.Name))
		}
	}

	resolvedElements, svcErr := s.validateAndResolveElements(ctx, input.Elements, orgID)
	if svcErr != nil {
		return nil, svcErr
	}

	purposeID := utils.GenerateUUID()
	versionID := utils.GenerateUUID()
	now := time.Now().UnixMilli()

	pv := &model.PurposeVersion{
		VersionID:   versionID,
		ID:          purposeID,
		Name:        input.Name,
		GroupID:     input.GroupID,
		VersionNum:  1,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Properties:  input.Properties,
		CreatedTime: now,
		OrgID:       orgID,
		Elements:    resolvedElements,
	}

	if err := s.stores.ExecuteTransaction(s.buildCreateVersionTx(pv, resolvedElements)); err != nil {
		if isMySQLDuplicateKeyError(err) {
			if input.GroupID == orgID {
				return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists,
					fmt.Sprintf("a purpose named '%s' already exists in this org", input.Name))
			}
			return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists,
				fmt.Sprintf("a purpose named '%s' already exists in this group", input.Name))
		}
		logger.Error("Failed to create consent purpose", log.Error(err))
		return nil, &ErrorCreatePurpose
	}

	logger.Info("Consent purpose created", log.String("purpose_id", purposeID), log.String("org_id", orgID))
	return pvToOutput(pv), nil
}

// CreatePurposeVersion appends a new version to an existing purpose.
func (s *consentPurposeService) CreatePurposeVersion(ctx context.Context, purposeID string, input model.CreatePurposeVersionInput, orgID string) (*model.PurposeOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	// Validate input fields shared with create.
	if input.Description != nil && len(*input.Description) > 1024 {
		return nil, serviceerror.CustomServiceError(ErrorInvalidRequestBody, "description must not exceed 1024 characters")
	}

	// Confirm purpose exists and get current max version.
	latest, err := s.stores.ConsentPurpose.GetLatestVersion(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to get latest purpose version", log.Error(err), log.String("purpose_id", purposeID))
		return nil, &ErrorRetrievePurpose
	}
	if latest == nil {
		return nil, &ErrorPurposeNotFound
	}

	resolvedElements, svcErr := s.validateAndResolveElements(ctx, input.Elements, orgID)
	if svcErr != nil {
		return nil, svcErr
	}

	versionID := utils.GenerateUUID()
	now := time.Now().UnixMilli()

	pv := &model.PurposeVersion{
		VersionID:   versionID,
		ID:          purposeID,
		Name:        latest.Name,
		GroupID:     latest.GroupID,
		VersionNum:  latest.VersionNum + 1,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Properties:  input.Properties,
		CreatedTime: now,
		OrgID:       orgID,
		Elements:    resolvedElements,
	}

	if err := s.stores.ExecuteTransaction(s.buildCreateVersionTx(pv, resolvedElements)); err != nil {
		if isMySQLDuplicateKeyError(err) {
			return nil, serviceerror.CustomServiceError(ErrorPurposeNameExists,
				fmt.Sprintf("purpose '%s' was updated concurrently — please fetch the latest version and retry", latest.Name))
		}
		logger.Error("Failed to create purpose version", log.Error(err))
		return nil, &ErrorCreatePurpose
	}

	logger.Info("Purpose version created",
		log.String("purpose_id", purposeID),
		log.Int("version", pv.VersionNum),
		log.String("org_id", orgID))
	return pvToOutput(pv), nil
}

// buildCreateVersionTx returns the transaction steps for creating a purpose version and its element links.
func (s *consentPurposeService) buildCreateVersionTx(pv *model.PurposeVersion, elements []model.PurposeMappedElement) []func(tx dbmodel.TxInterface) error {
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.CreateVersion(tx, pv)
		},
	}
	for _, elem := range elements {
		e := elem // capture
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.LinkElementVersion(tx, pv.VersionID, e.ElementVersionID, pv.OrgID, e.Mandatory)
		})
	}
	return queries
}

// =============================================================================
// Read
// =============================================================================

// GetPurpose returns the latest version of a purpose.
func (s *consentPurposeService) GetPurpose(ctx context.Context, purposeID, orgID string) (*model.PurposeOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	pv, err := s.stores.ConsentPurpose.GetLatestVersion(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve purpose", log.Error(err), log.String("purpose_id", purposeID))
		return nil, &ErrorRetrievePurpose
	}
	if pv == nil {
		return nil, &ErrorPurposeNotFound
	}
	return pvToOutput(pv), nil
}

// GetPurposeVersion returns a specific version of a purpose.
func (s *consentPurposeService) GetPurposeVersion(ctx context.Context, purposeID string, version int, orgID string) (*model.PurposeOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	pv, err := s.stores.ConsentPurpose.GetVersion(ctx, purposeID, version, orgID)
	if err != nil {
		logger.Error("Failed to retrieve purpose version", log.Error(err), log.String("purpose_id", purposeID))
		return nil, &ErrorRetrievePurpose
	}
	if pv == nil {
		return nil, &ErrorPurposeNotFound
	}
	return pvToOutput(pv), nil
}

// GetPurposeVersions returns all versions of a purpose ordered ascending.
func (s *consentPurposeService) GetPurposeVersions(ctx context.Context, purposeID, orgID string) (*model.PurposeVersionListOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	// Verify purpose exists before listing.
	exists, err := s.stores.ConsentPurpose.PurposeExists(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to check purpose existence", log.Error(err), log.String("purpose_id", purposeID))
		return nil, &ErrorRetrievePurpose
	}
	if !exists {
		return nil, &ErrorPurposeNotFound
	}

	versions, err := s.stores.ConsentPurpose.ListVersions(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to list purpose versions", log.Error(err), log.String("purpose_id", purposeID))
		return nil, &ErrorListPurposes
	}

	outputs := make([]model.PurposeOutput, 0, len(versions))
	for i := range versions {
		outputs = append(outputs, *pvToOutput(&versions[i]))
	}

	// Hoist the purpose-level fields from the first version (all versions share the same ID, Name, GroupID).
	out := &model.PurposeVersionListOutput{
		PurposeID: purposeID,
		Versions:  outputs,
	}
	if len(outputs) > 0 {
		out.Name = outputs[0].Name
		out.GroupID = outputs[0].GroupID
	}
	return out, nil
}

// ListPurposes returns paginated purposes matching the filters.
func (s *consentPurposeService) ListPurposes(ctx context.Context, orgID string, filters model.PurposeListFilter) (*model.PurposeListOutput, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	versions, total, err := s.stores.ConsentPurpose.List(ctx, orgID, filters)
	if err != nil {
		logger.Error("Failed to list purposes", log.Error(err), log.String("org_id", orgID))
		return nil, &ErrorListPurposes
	}

	outputs := make([]model.PurposeOutput, 0, len(versions))
	for i := range versions {
		outputs = append(outputs, *pvToOutput(&versions[i]))
	}

	return &model.PurposeListOutput{
		Data:   outputs,
		Total:  total,
		Offset: filters.Offset,
		Count:  len(outputs),
		Limit:  filters.Limit,
	}, nil
}

// =============================================================================
// Delete
// =============================================================================

// DeletePurposeVersion deletes a specific version.
// Returns 409 if the version is referenced by a consent.
// Deleting the last version also deletes the purpose entity.
func (s *consentPurposeService) DeletePurposeVersion(ctx context.Context, purposeID string, version int, orgID string) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)

	// Fetch the target version to get its VERSION_ID.
	pv, err := s.stores.ConsentPurpose.GetVersion(ctx, purposeID, version, orgID)
	if err != nil {
		logger.Error("Failed to retrieve purpose version", log.Error(err))
		return &ErrorRetrievePurpose
	}
	if pv == nil {
		return &ErrorPurposeNotFound
	}

	// Reject if this version is bound to any consent.
	inUse, err := s.stores.ConsentPurpose.IsVersionUsedInConsents(ctx, pv.VersionID, orgID)
	if err != nil {
		logger.Error("Failed to check version usage", log.Error(err))
		return &ErrorCheckPurposeUsage
	}
	if inUse {
		return serviceerror.CustomServiceError(ErrorPurposeVersionInUse,
			fmt.Sprintf("purpose version v%d is referenced by one or more consents and cannot be deleted", version))
	}

	// Determine whether this is the last remaining version.
	allVersions, err := s.stores.ConsentPurpose.ListVersions(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to list purpose versions", log.Error(err))
		return &ErrorRetrievePurpose
	}

	isLastVersion := len(allVersions) == 1

	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.DeleteVersion(tx, pv.VersionID, orgID)
		},
	}
	if isLastVersion {
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.DeletePurpose(tx, purposeID, orgID)
		})
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to delete purpose version", log.Error(err))
		return &ErrorDeletePurpose
	}

	logger.Info("Purpose version deleted",
		log.String("purpose_id", purposeID),
		log.Int("version", version),
		log.Bool("purpose_also_deleted", isLastVersion),
	)
	return nil
}

// DeletePurpose deletes a purpose and all its versions.
// Returns 409 if any version is referenced by a consent.
func (s *consentPurposeService) DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)

	exists, err := s.stores.ConsentPurpose.PurposeExists(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to check purpose existence", log.Error(err))
		return &ErrorRetrievePurpose
	}
	if !exists {
		return &ErrorPurposeNotFound
	}

	// Check every version before deleting — any version in use blocks the entire delete.
	versions, err := s.stores.ConsentPurpose.ListVersions(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to list purpose versions", log.Error(err))
		return &ErrorRetrievePurpose
	}

	for _, v := range versions {
		inUse, err := s.stores.ConsentPurpose.IsVersionUsedInConsents(ctx, v.VersionID, orgID)
		if err != nil {
			logger.Error("Failed to check version usage", log.Error(err))
			return &ErrorCheckPurposeUsage
		}
		if inUse {
			return serviceerror.CustomServiceError(ErrorPurposeVersionInUse,
				fmt.Sprintf("purpose version v%d is referenced by one or more consents and cannot be deleted", v.VersionNum))
		}
	}

	if err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurpose.DeletePurpose(tx, purposeID, orgID)
		},
	}); err != nil {
		logger.Error("Failed to delete purpose", log.Error(err))
		return &ErrorDeletePurpose
	}

	logger.Info("Purpose deleted", log.String("purpose_id", purposeID))
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

// validatePurposeInput validates the fields shared by create and version-create requests.
func (s *consentPurposeService) validatePurposeInput(name string, description *string) *serviceerror.ServiceError {
	if name == "" {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "name is required")
	}
	if len(name) > 255 {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "name must not exceed 255 characters")
	}
	if description != nil && len(*description) > 1024 {
		return serviceerror.CustomServiceError(ErrorInvalidRequestBody, "description must not exceed 1024 characters")
	}
	return nil
}

// validateAndResolveElements validates element refs and resolves each to a concrete PurposeMappedElement
// with a populated ElementVersionID. Returns an error if any element does not exist.
func (s *consentPurposeService) validateAndResolveElements(ctx context.Context, refs []model.ElementRef, orgID string) ([]model.PurposeMappedElement, *serviceerror.ServiceError) {
	if len(refs) == 0 {
		return nil, serviceerror.CustomServiceError(ErrorInvalidPurposeElements, "at least one element is required")
	}

	seen := make(map[string]bool, len(refs))
	resolved := make([]model.PurposeMappedElement, 0, len(refs))

	for _, ref := range refs {
		ns := ref.Namespace
		if ns == "" {
			ns = elementmodel.DefaultNamespace
		}

		// Dedup key includes version when specified to allow the same element at different versions.
		dupKey := ref.Name + "|" + ns
		if ref.Version != nil {
			dupKey += fmt.Sprintf("|v%d", *ref.Version)
		}
		if seen[dupKey] {
			return nil, serviceerror.CustomServiceError(ErrorInvalidPurposeElements,
				fmt.Sprintf("duplicate element reference '%s' in namespace '%s'", ref.Name, ns))
		}
		seen[dupKey] = true

		// GetByNameAndNamespace returns the latest version and gives us the element ID.
		ev, err := s.stores.ConsentElement.GetByNameAndNamespace(ctx, ref.Name, ns, orgID)
		if err != nil {
			return nil, &ErrorInternalServerError
		}
		if ev == nil {
			return nil, serviceerror.CustomServiceError(ErrorInvalidPurposeElements,
				fmt.Sprintf("element '%s' in namespace '%s' does not exist", ref.Name, ns))
		}

		elementVersionID := ev.VersionID
		versionNum := ev.VersionNum

		// If a specific version was requested, fetch that exact version.
		if ref.Version != nil {
			specific, err := s.stores.ConsentElement.GetVersion(ctx, ev.ID, *ref.Version, orgID)
			if err != nil {
				return nil, &ErrorInternalServerError
			}
			if specific == nil {
				return nil, serviceerror.CustomServiceError(ErrorInvalidPurposeElements,
					fmt.Sprintf("element '%s' version v%d does not exist", ref.Name, *ref.Version))
			}
			elementVersionID = specific.VersionID
			versionNum = specific.VersionNum
		}

		resolved = append(resolved, model.PurposeMappedElement{
			ElementVersionID: elementVersionID,
			ElementID:        ev.ID,
			Name:             ref.Name,
			Namespace:        ns,
			VersionNum:       versionNum,
			Mandatory:        ref.Mandatory,
		})
	}

	return resolved, nil
}

// isMySQLDuplicateKeyError reports whether err is a MySQL unique-constraint violation (error 1062).
func isMySQLDuplicateKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// pvToOutput maps a DB-layer PurposeVersion to the service-layer PurposeOutput.
// This keeps DB types (db tags) inside the store layer and exposes clean types to the handler.
func pvToOutput(pv *model.PurposeVersion) *model.PurposeOutput {
	if pv == nil {
		return nil
	}
	elems := make([]model.PurposeElementOutput, 0, len(pv.Elements))
	for _, e := range pv.Elements {
		elems = append(elems, model.PurposeElementOutput{
			ElementVersionID: e.ElementVersionID,
			ElementID:        e.ElementID,
			Name:             e.Name,
			Namespace:        e.Namespace,
			VersionNum:       e.VersionNum,
			Mandatory:        e.Mandatory,
		})
	}
	return &model.PurposeOutput{
		VersionID:   pv.VersionID,
		ID:          pv.ID,
		Name:        pv.Name,
		GroupID:     pv.GroupID,
		VersionNum:  pv.VersionNum,
		DisplayName: pv.DisplayName,
		Description: pv.Description,
		CreatedTime: pv.CreatedTime,
		OrgID:       pv.OrgID,
		Properties:  pv.Properties,
		Elements:    elems,
	}
}
